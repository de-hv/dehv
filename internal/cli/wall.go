package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/de-hv/dehv/internal/config"
	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

func wallCmd() *cobra.Command {
	var sync, all, open, use, list, initF, clear bool
	var to, newW, removeW string
	c := &cobra.Command{
		Use:   "wall <url> [<name>] [-t <repo>] | -s [-t <repo> | -a] | -o [-t <repo>] | -u [<repo>] | -n <repo> | -r <repo> | -l | -i | -x",
		Short: "add wallpapers to a GitHub repo and sync it with a folder",
		Long: "Add images to a GitHub repo of wallpapers, and keep a folder in sync with it. " +
			"The repos are this Mac's, in ~/.config/dehv/walls, in git's config format.",
		GroupID: "media",
		Example: "dehv wall https://images.example.com/aurora.jpg   # download, commit and push it\n" +
			"dehv wall -s                                      # push what I dropped in the folder\n" +
			"dehv wall -n personal                             # add a repo: owner/name is enough",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			var src, name, pick string
			if use {
				if len(args) > 1 {
					return misuse(cmd, "unexpected argument '"+args[1]+"'")
				}
				if len(args) == 1 {
					pick = args[0]
				}
			} else {
				if len(args) > 2 {
					return misuse(cmd, "unexpected argument '"+args[2]+"'")
				}
				if len(args) > 0 {
					src = args[0]
				}
				if len(args) > 1 {
					name = args[1]
				}
			}

			// -u, -n, -r, -l, -i and -x each go alone; <url>, -s and -o one at a time.
			var solo, acts []string
			for _, f := range []struct {
				on   bool
				name string
			}{{use, "-u"}, {newW != "", "-n"}, {removeW != "", "-r"}, {list, "-l"}, {initF, "-i"}, {clear, "-x"}} {
				if f.on {
					solo = append(solo, f.name)
				}
			}
			if src != "" {
				acts = append(acts, "<url>")
			}
			if sync {
				acts = append(acts, "-s")
			}
			if open {
				acts = append(acts, "-o")
			}
			switch {
			case len(solo) > 1 || len(solo) == 1 && (len(acts) > 0 || to != "" || all):
				return misuse(cmd, solo[0]+" goes alone")
			case len(acts) > 1:
				return misuse(cmd, acts[0]+" and "+acts[1]+" go apart")
			case all && !sync:
				return misuse(cmd, "-a goes with -s")
			case all && to != "":
				return misuse(cmd, "-a and -t go apart")
			case to != "" && len(acts) == 0:
				return misuse(cmd, "-t goes with <url>, -s or -o")
			case src != "" && !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://"):
				return misuse(cmd, "'"+src+"' is not an http(s) url")
			case name != "" && (strings.Contains(name, "/") || strings.HasPrefix(name, ".")):
				return misuse(cmd, "invalid name '"+name+"': a file name, without / nor a leading .")
			}
			for _, p := range []string{to, pick, newW, removeW} {
				if p != "" && !core.ValidWall(p) {
					return misuse(cmd, "invalid repo '"+p+"': lower-case letters, digits, - and _")
				}
			}

			file := core.WallsFile()
			switch {
			case initF:
				return initFile(cmd, file, "repo", "repos")
			case clear:
				ws := core.Walls()
				if len(ws) == 0 {
					ui.Noop("no repos, nothing to do")
					return nil
				}
				if err := needsTTY(cmd, "-x needs a terminal to confirm"); err != nil {
					return err
				}
				var names []string
				for _, w := range ws {
					names = append(names, w.Name)
				}
				ui.Println(ui.Name.Render(strings.Join(names, ", ")))
				ask := fmt.Sprintf("Forget all %d repos? Their folders stay.", len(ws))
				if len(ws) == 1 {
					ask = "Forget the only repo? Its folder stays."
				}
				if yes, err := ui.Confirm(ask); err != nil || !yes {
					ui.Noop("kept them")
					return err
				}
				if err := config.Truncate(file); err != nil {
					return err
				}
				ui.OK("forgot "+ui.Plural(len(ws), "repo", "repos"), "")
				return nil
			case newW != "":
				return newWall(cmd, newW)
			case removeW != "":
				w, err := core.RemoveWall(removeW)
				if err != nil {
					return err
				}
				ui.OK("forgot "+ui.Name.Render(removeW), w.Dir+" stays")
				return nil
			case list:
				return listWalls()
			case use:
				if pick != "" {
					if _, ok := core.GetWall(pick); !ok {
						return core.Fail("no repo "+pick, "'dehv wall -n "+pick+"' adds it")
					}
				} else {
					p, err := chooseWall(cmd, "Use which repo")
					if err != nil {
						return err
					}
					pick = p
				}
				from := config.Get(file, "wall.use")
				if from == pick {
					ui.Noop("already using " + pick + ", nothing to do")
					return nil
				}
				if _, err := core.UseWall(pick); err != nil {
					return err
				}
				w, _ := core.GetWall(pick)
				if from != "" {
					ui.OK(ui.Dim.Render(from+" →")+" "+ui.Bold.Render(pick), w.Short())
				} else {
					ui.OK(ui.Bold.Render(pick), w.Short())
				}
				return nil
			}

			ctx := cmd.Context()
			r := &reporter{}
			defer r.done()

			// -s -a: every repo, on to the next when one fails.
			if all {
				ws := core.Walls()
				if len(ws) == 0 {
					return core.Fail("no repos yet", "'dehv wall -n <repo>' adds one")
				}
				failed := false
				for _, w := range ws {
					if err := core.SyncWall(ctx, w, r); err != nil {
						if errors.Is(err, context.Canceled) {
							return err
						}
						r.done()
						handleError(nil, err)
						failed = true
					}
				}
				if failed {
					return errSilent
				}
				return nil
			}

			// <url>, -s and -o: on the repo named with -t, or the one in use.
			var w core.Wall
			if to != "" {
				var ok bool
				if w, ok = core.GetWall(to); !ok {
					return core.Fail("no repo "+to, "'dehv wall -n "+to+"' adds it")
				}
			} else {
				var ok bool
				var err error
				if w, ok, err = core.WallInUse(); err != nil {
					return err
				}
				if !ok {
					if !ui.Interactive() {
						return core.Fail("no repo in use", "'dehv wall -u' chooses one")
					}
					ui.Println(ui.Dim.Render("no repo in use, pick one:"))
					p, err := chooseWall(cmd, "Use which repo")
					if err != nil {
						return err
					}
					if _, err := core.UseWall(p); err != nil {
						return err
					}
					ui.OK(ui.Bold.Render(p), "in use from now on")
					w, _ = core.GetWall(p)
				}
			}

			switch {
			case open:
				if st, err := os.Stat(w.Path()); err != nil || !st.IsDir() {
					if w.Cloned {
						return core.Fail(config.Tilde(w.Path())+" is gone", "'dehv wall -s' clones it again, after asking")
					}
					return core.Fail(w.Name+" is not cloned yet", "'dehv wall -s' clones it")
				}
				return exec.Command("open", w.Path()).Run()
			case sync:
				return core.SyncWall(ctx, w, r)
			}
			return core.AddWallpaper(ctx, w, src, name, r)
		},
	}
	f := c.Flags()
	f.BoolVarP(&sync, "sync", "s", false, "commit the folder's changes, pull and push; clones it first")
	f.BoolVarP(&all, "all", "a", false, "with -s, every repo, one after the other")
	f.BoolVarP(&open, "open", "o", false, "open the folder of the repo in use")
	f.BoolVarP(&use, "use", "u", false, "choose the repo the rest use; from a list without one")
	f.StringVarP(&to, "to", "t", "", "that repo just this once, with <url>, -s or -o")
	f.StringVarP(&newW, "new", "n", "", "add or edit a repo: asks its GitHub url and its folder")
	f.StringVarP(&removeW, "remove", "r", "", "forget a repo; its folder stays")
	f.BoolVarP(&list, "list", "l", false, "list my repos, marking the one in use")
	f.BoolVarP(&initF, "init", "i", false, "create the repos file, or keep or blank it")
	f.BoolVarP(&clear, "clear", "x", false, "forget every repo, after asking")
	for _, fl := range []string{"to", "new", "remove"} {
		_ = c.RegisterFlagCompletionFunc(fl, completeWalls)
	}
	return c
}

func newWall(cmd *cobra.Command, name string) error {
	if err := needsTTY(cmd, "-n needs a terminal to ask the url and the folder"); err != nil {
		return err
	}
	w, exists := core.GetWall(name)
	verb := "updated"
	if !exists {
		verb = "added"
	}
	u, err := ui.Input("GitHub url of "+name+" (owner/name is enough)", w.URL, nonEmpty("a url"))
	if err != nil {
		return err
	}
	if w.URL, err = core.GitURL(u); err != nil {
		return err
	}
	dir := w.Dir
	if dir == "" {
		dir = core.DefaultWallDir(w.URL)
	}
	if w.Dir, err = ui.Input("Its folder", dir, nonEmpty("a folder")); err != nil {
		return err
	}
	if err := core.SaveWall(w); err != nil {
		return err
	}
	w, _ = core.GetWall(name)
	ui.OK(verb+" "+ui.Name.Render(name), w.Short()+" → "+w.Dir)
	if !config.Exists(w.Path()) {
		ui.Note("'dehv wall -s' clones it")
	}
	return nil
}

func listWalls() error {
	ws := core.Walls()
	if len(ws) == 0 {
		return core.Fail("no repos yet", "'dehv wall -n <repo>' or 'dehv wall -i'")
	}
	if !ui.IsTTY(ui.Stdout) {
		for _, w := range ws {
			ui.Println(w.Name + "\t" + w.URL + "\t" + w.Dir)
		}
		return nil
	}
	cur, _, _ := core.WallInUse()
	var rows [][]string
	for _, w := range ws {
		mark := " "
		if w.Name == cur.Name {
			mark = ui.OKMark.Render("●")
		}
		state := core.WallState(w)
		if state == "gone" {
			state = ui.Err.Render(state)
		} else {
			state = ui.Dim.Render(state)
		}
		rows = append(rows, []string{mark + " " + ui.Bold.Render(w.Name), w.Short(), ui.Dim.Render(w.Dir), state})
	}
	ui.Table([]string{"  REPO", "GITHUB", "FOLDER", "IMAGES"}, rows)
	return nil
}

// chooseWall is the only repo, or the one picked from the list.
func chooseWall(cmd *cobra.Command, title string) (string, error) {
	ws := core.Walls()
	switch len(ws) {
	case 0:
		return "", core.Fail("no repos yet", "'dehv wall -n <repo>' adds one")
	case 1:
		return ws[0].Name, nil
	}
	if !ui.Interactive() {
		var names []string
		for _, w := range ws {
			names = append(names, w.Name)
		}
		return "", misuse(cmd, "name the repo: there is "+strings.Join(names, ", "))
	}
	w := 0
	for _, x := range ws {
		w = max(w, len(x.Name))
	}
	var opts []string
	for _, x := range ws {
		opts = append(opts, ui.Pad(x.Name, w)+"  "+ui.Dim.Render(x.Short()))
	}
	i, err := ui.Pick(title, opts)
	if err != nil {
		return "", err
	}
	return ws[i].Name, nil
}

func completeWalls(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	var names []string
	for _, w := range core.Walls() {
		names = append(names, w.Name+"\t"+w.Short())
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
