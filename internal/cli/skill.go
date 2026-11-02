package cli

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/de-hv/dehv/internal/config"
	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

func skillCmd() *cobra.Command {
	var link, unlink, del, restore, all bool
	var to string
	c := &cobra.Command{
		Use:   "skill [-l] [-t <agents>] <name> | -u <name> | -d <name> | -r <name> | -a",
		Short: "manage my skills and the agents that see them",
		Long: "Manage my skills, kept in skill.dir of the config (~/Documents/dev/skills without it), " +
			"and the agents that see them: claude, cursor, copilot, xcode and codex, when installed.",
		GroupID: "agents",
		Example: "dehv skill notes -l           # create it and tick the agents that see it\n" +
			"dehv skill notes -t copilot   # link it into copilot\n" +
			"dehv skill -a                 # every skill and where it is linked",
		ValidArgsFunction: completeSkills,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) > 1 {
				return misuse(cmd, "too many arguments")
			}
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			if all {
				if name != "" || to != "" || link || unlink || del || restore {
					return misuse(cmd, "-a goes alone")
				}
				return listSkills()
			}
			if name == "" {
				return misuse(cmd, "missing <name>")
			}
			if !core.ValidSkill(name) {
				return misuse(cmd, "invalid name '"+name+"'")
			}
			if del && (to != "" || link || unlink || restore) {
				return misuse(cmd, "-d goes alone")
			}
			if unlink && (link || restore) {
				return misuse(cmd, "-u goes alone, or with -t")
			}

			var known []string
			for _, a := range core.Agents() {
				known = append(known, a.Name)
			}
			var toNames []string
			if to != "" {
				for _, a := range strings.Split(to, ",") {
					if !slices.Contains(known, a) {
						return misuse(cmd, "unknown agent '"+a+"': there is "+strings.Join(known, ", "))
					}
					toNames = append(toNames, a)
				}
				// Naming the agents is asking to link into them.
				if !unlink && !restore {
					link = true
				}
			}

			skills := core.SkillsDir()
			if _, err := filepath.Abs(skills); err != nil || !config.Exists(skills) {
				return core.Fail(config.Tilde(skills)+" does not exist", "")
			}
			agents := core.InstalledAgents()
			now := core.LinkedInto(name)
			var first []string
			if len(agents) > 0 && agents[0].Name == known[0] {
				first = []string{known[0]}
			}
			folder := ui.Dim.Render(config.Tilde(skills)+"/") + ui.Bold.Render(name)
			failed := false
			fail := func(msg, hint string) { ui.Fail(msg, hint); failed = true }
			done := func() error {
				if failed {
					return errSilent
				}
				return nil
			}

			// choose is the agents to act on: those named with -t, or the ones
			// ticked in a list of the candidates, or without a terminal the
			// ones that would start ticked.
			choose := func(head, action string, marked, cand []string) ([]core.Agent, error) {
				var chosen []core.Agent
				if len(toNames) > 0 {
					for _, n := range toNames {
						a, _ := core.AgentNamed(n)
						if !a.Installed {
							fail("cannot "+action+" "+n, "it is not installed")
							continue
						}
						chosen = append(chosen, a)
					}
					return chosen, nil
				}
				names := marked
				if ui.Interactive() {
					w := 0
					for _, n := range cand {
						w = max(w, len(n))
					}
					var opts []string
					var on []int
					for i, n := range cand {
						a, _ := core.AgentNamed(n)
						o := ui.Pad(n, w) + "  " + ui.Dim.Render(config.Tilde(a.Dir))
						if n == known[0] {
							o += ui.Dim.Render("  default")
						}
						opts = append(opts, o)
						if slices.Contains(marked, n) {
							on = append(on, i)
						}
					}
					picked, err := ui.PickMany(head, opts, on)
					if err != nil {
						return nil, err
					}
					names = nil
					for _, i := range picked {
						names = append(names, cand[i])
					}
				}
				for _, n := range names {
					a, _ := core.AgentNamed(n)
					chosen = append(chosen, a)
				}
				return chosen, nil
			}
			linkInto := func(a core.Agent) bool {
				if err := core.LinkSkill(name, a); err != nil {
					var ce *core.Error
					if errors.As(err, &ce) {
						fail(ce.Msg, ce.Hint)
					} else {
						fail("cannot link into "+a.Name, err.Error())
					}
					return false
				}
				ui.OK("linked   "+ui.Dim.Render(config.Tilde(a.Dir)+"/")+ui.Bold.Render(name), "")
				return true
			}
			unlinkFrom := func(a core.Agent) bool {
				did, err := core.UnlinkSkill(name, a)
				if err != nil {
					fail("cannot unlink from "+a.Name, err.Error())
					return false
				}
				if did {
					ui.OK("unlinked "+ui.Dim.Render(config.Tilde(a.Dir)+"/")+ui.Bold.Render(name), "")
				}
				return did
			}
			names := func(as []core.Agent) []string {
				var ns []string
				for _, a := range as {
					ns = append(ns, a.Name)
				}
				return ns
			}

			switch {
			case unlink:
				if len(now) == 0 {
					return core.Fail("skill '"+name+"' is not linked into any agent", "")
				}
				var chosen []core.Agent
				switch {
				case to == "" && len(now) == 1:
					a, _ := core.AgentNamed(now[0])
					chosen = []core.Agent{a}
				case to == "" && !ui.Interactive():
					return core.Fail("skill '"+name+"' is linked into "+strings.Join(now, ", "), "name the ones to leave with -t")
				default:
					var err error
					if chosen, err = choose("Unlink "+name+" from", "unlink from", now, now); err != nil {
						return err
					}
				}
				if len(chosen) == 0 && !failed {
					ui.Noop("no agent ticked, nothing to do")
					return nil
				}
				for _, a := range chosen {
					if !slices.Contains(now, a.Name) {
						fail("skill '"+name+"' is not linked into "+a.Name, "")
						continue
					}
					unlinkFrom(a)
				}
				return done()

			case del:
				if err := core.TrashSkill(name); err != nil {
					return err
				}
				ui.OK("trashed  "+folder, "")
				for _, n := range now {
					a, _ := core.AgentNamed(n)
					unlinkFrom(a)
				}
				ui.Note("dehv skill -r " + name + " brings it back")
				return nil

			case restore:
				if err := core.RestoreSkill(name); err != nil {
					return err
				}
				ui.OK("restored "+folder, "")
				// It is back either way: leaving the list just links it nowhere.
				if len(agents) > 0 || to != "" {
					chosen, err := choose("Link "+name+" into (none to skip)", "link into", first, names(agents))
					if err != nil {
						chosen = nil
					}
					for _, a := range chosen {
						linkInto(a)
					}
				}
				return done()
			}

			// Create, link, or both. The agents are asked first, so leaving the
			// list creates nothing.
			listed := false
			var chosen []core.Agent
			if link {
				if len(agents) == 0 {
					return core.Fail("no agent to link into", "none of "+strings.Join(known, ", ")+" is installed")
				}
				listed = to == "" && ui.Interactive()
				marked := now
				if len(marked) == 0 {
					marked = first
				}
				var err error
				if chosen, err = choose("Link "+name+" into", "link into", marked, names(agents)); err != nil {
					return err
				}
				if listed && len(chosen) == 0 {
					return core.Fail("no agent ticked", "-u takes it out of them")
				}
			}

			did := false
			if !core.SkillExists(name) {
				if err := core.CreateSkill(name); err != nil {
					return err
				}
				ui.OK("created  "+folder, "")
				did = true
			} else if !link {
				return core.Fail("skill '"+name+"' already exists", "")
			}
			if link {
				for _, a := range chosen {
					if slices.Contains(now, a.Name) {
						continue
					}
					did = linkInto(a) || did
				}
				// The list shows where it is: what got unticked there comes out.
				if listed {
					for _, n := range now {
						if !slices.Contains(names(chosen), n) {
							a, _ := core.AgentNamed(n)
							did = unlinkFrom(a) || did
						}
					}
				}
				if !did && !failed {
					ui.Noop("already linked there, nothing to do")
				}
			}
			return done()
		},
	}
	f := c.Flags()
	f.BoolVarP(&link, "link", "l", false, "link it into the agents ticked in a list, creating it if missing")
	f.BoolVarP(&unlink, "unlink", "u", false, "take it out of the agents ticked, keeping the skill")
	f.BoolVarP(&del, "delete", "d", false, "unlink it from every agent and move it to the Trash")
	f.BoolVarP(&restore, "restore", "r", false, "bring it back from the Trash, then tick where to link it")
	f.StringVarP(&to, "to", "t", "", "no list: the agents by name, like claude,cursor (with -l, -u or -r)")
	f.BoolVarP(&all, "all", "a", false, "list my skills, and the agents each one is linked into")
	_ = c.RegisterFlagCompletionFunc("to", completeAgents)
	return c
}

// listSkills is `dehv skill -a`: a column per installed agent, a green dot
// where the skill is linked, a warning on those without description (no
// agent can trigger them) and on my links left pointing at nothing.
func listSkills() error {
	ss, err := core.Skills()
	if err != nil {
		return err
	}
	if len(ss) == 0 {
		ui.Println("No skills in " + config.Tilde(core.SkillsDir()) + ".")
		return nil
	}
	agents := core.InstalledAgents()
	tty := ui.IsTTY(ui.Stdout)
	header := []string{"NAME"}
	for _, a := range agents {
		header = append(header, a.Name)
	}
	header = append(header, "DESCRIPTION")
	var rows [][]string
	for _, s := range ss {
		row := []string{ui.Bold.Render(s.Name)}
		for _, a := range agents {
			switch {
			case !tty && s.Linked[a.Name]:
				row = append(row, "linked")
			case !tty:
				row = append(row, "-")
			case s.Linked[a.Name] && s.Broken:
				row = append(row, ui.Err.Render("●"))
			case s.Linked[a.Name]:
				row = append(row, ui.OKMark.Render("●"))
			default:
				row = append(row, ui.Dim.Render("○"))
			}
		}
		switch {
		case s.Broken:
			row = append(row, ui.Err.Render("! broken link"))
		case s.Description == "":
			row = append(row, ui.Warn.Render("! no description"))
		default:
			row = append(row, ui.Dim.Render(s.Description))
		}
		rows = append(rows, row)
	}
	ui.Table(header, rows)
	if tty {
		ui.Println(ui.Dim.Render("● linked  ○ not linked"))
	}
	return nil
}

func completeSkills(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ss, _ := core.Skills()
	var names []string
	for _, s := range ss {
		if !s.Broken {
			names = append(names, s.Name+"\t"+s.Description)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func completeAgents(_ *cobra.Command, _ []string, typed string) ([]string, cobra.ShellCompDirective) {
	// Comma-separated: complete the last one after what is already typed.
	prefix := ""
	if i := strings.LastIndex(typed, ","); i >= 0 {
		prefix = typed[:i+1]
	}
	var names []string
	for _, a := range core.Agents() {
		names = append(names, prefix+a.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}
