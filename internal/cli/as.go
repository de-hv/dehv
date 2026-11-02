package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/de-hv/dehv/internal/config"
	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

func asCmd() *cobra.Command {
	var set, amend, check, list, initF, clear bool
	var newP, removeP string
	c := &cobra.Command{
		Use:   "as [<profile>] [-s] [-a [<n>]] | -n <profile> | -r <profile> | -l | -c | -i | -x",
		Short: "commit as one of my git profiles",
		Long: "Commit as one of my git profiles in this repo. The profiles are this Mac's, in ~/.config/dehv/profiles, " +
			"in git's config format. Names are lower-case letters, digits, - and _, never only digits.",
		GroupID: "git",
		Example: "dehv as acme               # this repo commits as acme\n" +
			"dehv as acme -s -a         # and fix the last commit\n" +
			"dehv as -a 2               # the last 2 commits as the repo's profile\n" +
			"dehv as -n acme            # create or edit acme",
		ValidArgsFunction: completeProfiles,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			var profile string
			n := 0
			for _, a := range args {
				if v, err := strconv.Atoi(a); err == nil && v > 0 {
					if n != 0 {
						return misuse(cmd, "unexpected argument '"+a+"'")
					}
					n = v
				} else {
					if profile != "" {
						return misuse(cmd, "unexpected argument '"+a+"'")
					}
					profile = a
				}
			}

			// -n, -r, -l, -c, -i and -x each go alone.
			var solo []string
			for _, f := range []struct {
				on   bool
				name string
			}{{newP != "", "-n"}, {removeP != "", "-r"}, {list, "-l"}, {check, "-c"}, {initF, "-i"}, {clear, "-x"}} {
				if f.on {
					solo = append(solo, f.name)
				}
			}
			if len(solo) > 1 || len(solo) == 1 && (set || amend || profile != "" || n != 0) {
				return misuse(cmd, solo[0]+" goes alone")
			}
			if n != 0 && !amend {
				return misuse(cmd, "<n> goes with -a")
			}
			for _, p := range []string{profile, newP, removeP} {
				if p != "" && !core.ValidProfile(p) {
					return misuse(cmd, "invalid profile '"+p+"': lower-case letters, digits, - and _")
				}
			}

			file := core.ProfilesFile()
			home := config.Tilde(file)
			switch {
			case initF:
				return initFile(cmd, file, "profile", "profiles")
			case clear:
				ps := core.Profiles()
				if len(ps) == 0 {
					ui.Noop("no profiles, nothing to do")
					return nil
				}
				if err := needsTTY(cmd, "-x needs a terminal to confirm"); err != nil {
					return err
				}
				var names []string
				for _, p := range ps {
					names = append(names, p.Name)
				}
				ui.Println(ui.Name.Render(strings.Join(names, ", ")))
				ask := fmt.Sprintf("Remove all %d profiles?", len(ps))
				if len(ps) == 1 {
					ask = "Remove the only profile?"
				}
				if yes, err := ui.Confirm(ask); err != nil || !yes {
					ui.Noop("kept them")
					return err
				}
				if err := config.Truncate(file); err != nil {
					return err
				}
				ui.OK("removed "+ui.Plural(len(ps), "profile", "profiles"), "")
				return nil
			case newP != "":
				if err := needsTTY(cmd, "-n needs a terminal to ask the name and the email"); err != nil {
					return err
				}
				p, exists := core.GetProfile(newP)
				verb := "updated"
				if !exists {
					verb = "created"
					user, email := core.GlobalIdent()
					if p.User == "" {
						p.User = user
					}
					p.Email = email
				}
				var err error
				if p.User, err = ui.Input("Name for "+newP, p.User, nonEmpty("a name")); err != nil {
					return err
				}
				if p.Email, err = ui.Input("Email for "+newP, p.Email, nonEmpty("an email")); err != nil {
					return err
				}
				if err := core.SaveProfile(p); err != nil {
					return err
				}
				ui.OK(verb+" "+ui.Name.Render(newP), p.Ident())
				return nil
			case removeP != "":
				if err := core.RemoveProfile(removeP); err != nil {
					return err
				}
				ui.OK("removed "+ui.Name.Render(removeP), "")
				return nil
			case list:
				ps := core.Profiles()
				if len(ps) == 0 {
					return core.Fail("no profiles yet", "'dehv as -n <profile>' or 'dehv as -i'")
				}
				cur, _ := core.CurrentProfile("")
				if !ui.IsTTY(ui.Stdout) {
					for _, p := range ps {
						ui.Println(p.Name + "\t" + p.User + "\t" + p.Email)
					}
					return nil
				}
				w := 0
				for _, p := range ps {
					w = max(w, len(p.Name))
				}
				for _, p := range ps {
					mark := " "
					if p.Name == cur.Name {
						mark = ui.OKMark.Render("●")
					}
					ui.Println(mark + " " + ui.Bold.Render(ui.Pad(p.Name, w)) + "  " + p.User + " " + ui.Dim.Render("<"+p.Email+">"))
				}
				ui.Println(ui.Dim.Render(fmt.Sprintf("%s · in %s", ui.Plural(len(ps), "profile", "profiles"), home)))
				return nil
			}

			if !core.InRepo("") {
				return core.Fail("not in a git repo", "")
			}
			rn, re := core.RepoIdent("")
			repo := rn + " <" + re + ">"

			if check {
				if rn == "" && re == "" {
					repo = "none set"
				}
				ui.Println(ui.Dim.Render("repo") + "   " + repo)
				if p, ok := core.CurrentProfile(""); ok {
					ui.Println(ui.Dim.Render("as") + "     " + ui.Name.Render(p.Name))
					return nil
				}
				ui.Println(ui.Dim.Render("as") + "     " + ui.Warn.Render("none of my profiles"))
				return errSilent
			}

			// A profile alone sets it; with -a alone it only rewrites the authors.
			if profile != "" && !amend {
				set = true
			}
			// Without a name -a keeps the repo's profile; otherwise one is picked.
			if profile == "" {
				if cur, ok := core.CurrentProfile(""); ok && !set {
					profile = cur.Name
				} else {
					p, err := chooseProfile(cmd)
					if err != nil {
						return err
					}
					profile = p
				}
			}
			p, ok := core.GetProfile(profile)
			if !ok {
				return core.Fail("no profile "+profile, "'dehv as -n "+profile+"' creates it")
			}

			if set {
				if repo == p.Ident() {
					ui.Noop("already as " + profile + ", nothing to do")
				} else {
					from := repo
					if cur, ok := core.CurrentProfile(""); ok {
						from = cur.Name
					}
					if err := core.UseProfile("", p); err != nil {
						return err
					}
					if rn == "" && re == "" {
						ui.OK(ui.Bold.Render(profile), p.Ident())
					} else {
						ui.OK(ui.Dim.Render(from+" →")+" "+ui.Bold.Render(profile), p.Ident())
					}
				}
			}

			if amend {
				if n == 0 {
					n = 1
				}
				done, err := core.AmendAuthors("", p, n)
				if err != nil {
					return err
				}
				if done == nil {
					if n == 1 {
						ui.Noop("the last commit is already as " + profile + ", nothing to do")
					} else {
						ui.Noop(fmt.Sprintf("the last %d commits are already as %s, nothing to do", n, profile))
					}
					return nil
				}
				for _, a := range done {
					ui.OK(ui.Generic.Render(a.Hash)+" "+a.Subject, "")
				}
				ui.Println(ui.Dim.Render(ui.Plural(n, "commit", "commits") + " now as " + profile))
			}
			return nil
		},
	}
	f := c.Flags()
	f.BoolVarP(&set, "set", "s", false, "set this repo's user.name/user.email to the profile's, picked from a list without one")
	f.BoolVarP(&amend, "amend", "a", false, "make it the author of the last <n> commits (1 without it): rewritten, so only before pushing")
	f.StringVarP(&newP, "new", "n", "", "create or edit a profile: asks the name and the email")
	f.StringVarP(&removeP, "remove", "r", "", "remove a profile")
	f.BoolVarP(&list, "list", "l", false, "list my profiles, marking the one this repo commits as")
	f.BoolVarP(&check, "check", "c", false, "show which profile this repo commits as")
	f.BoolVarP(&initF, "init", "i", false, "create the profiles file, or keep or blank the one there")
	f.BoolVarP(&clear, "clear", "x", false, "remove every profile, after asking")
	_ = c.RegisterFlagCompletionFunc("new", completeProfiles)
	_ = c.RegisterFlagCompletionFunc("remove", completeProfiles)
	return c
}

// chooseProfile is the only profile, or the one picked from the list.
func chooseProfile(cmd *cobra.Command) (string, error) {
	ps := core.Profiles()
	switch len(ps) {
	case 0:
		return "", core.Fail("no profiles yet", "'dehv as -n <profile>' or 'dehv as -i'")
	case 1:
		return ps[0].Name, nil
	}
	if !ui.Interactive() {
		var names []string
		for _, p := range ps {
			names = append(names, p.Name)
		}
		return "", misuse(cmd, "name the profile: there is "+strings.Join(names, ", "))
	}
	w := 0
	for _, p := range ps {
		w = max(w, len(p.Name))
	}
	var opts []string
	for _, p := range ps {
		opts = append(opts, ui.Pad(p.Name, w)+"  "+ui.Dim.Render(p.Email))
	}
	i, err := ui.Pick("Commit as", opts)
	if err != nil {
		return "", err
	}
	return ps[i].Name, nil
}

// initFile is -i of as and wall: create the file, or keep or blank the one
// there, keeping the old one as .bak.
func initFile(cmd *cobra.Command, file, one, many string) error {
	home := config.Tilde(file)
	if !config.Exists(file) {
		if err := config.Touch(file); err != nil {
			return err
		}
		ui.OK("created "+ui.Bold.Render(home), "")
		return nil
	}
	if err := needsTTY(cmd, "-i needs a terminal to choose what to do with "+home); err != nil {
		return err
	}
	var names []string
	if one == "profile" {
		for _, p := range core.Profiles() {
			names = append(names, p.Name)
		}
	} else {
		for _, w := range core.Walls() {
			names = append(names, w.Name)
		}
	}
	switch len(names) {
	case 0:
		ui.Println("found " + ui.Bold.Render(home) + ", with no " + many)
	default:
		ui.Println("found " + ui.Bold.Render(home) + " with " + ui.Plural(len(names), one, many) + ": " + ui.Name.Render(strings.Join(names, ", ")))
	}
	i, err := ui.Pick("What now?", []string{"Use it", "Start blank"})
	if err != nil {
		return err
	}
	if i == 0 {
		ui.Noop("kept " + home)
		return nil
	}
	if err := config.Blank(file); err != nil {
		return err
	}
	ui.OK("blank "+ui.Bold.Render(home), "")
	ui.Note("the old one is " + home + ".bak")
	return nil
}

func nonEmpty(what string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("needs " + what)
		}
		return nil
	}
}

func completeProfiles(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	var names []string
	for _, p := range core.Profiles() {
		names = append(names, p.Name+"\t"+p.Email)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
