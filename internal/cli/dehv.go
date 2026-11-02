package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/de-hv/dehv/internal/config"
	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "doctor",
		Short:   "check the install of dehv and what it needs",
		Long:    "Check the install of dehv: the command in the PATH, its completion, what it needs and its files. It only looks; exit code 1 when something is wrong.",
		GroupID: "dehv",
		Args:    cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			checks := core.Doctor()
			for _, l := range DoctorLines(checks, ui.IsTTY(ui.Stdout)) {
				ui.Println(l)
			}
			if !core.Healthy(checks) {
				return errSilent
			}
			return nil
		},
	}
}

// DoctorLines is what `dehv doctor` prints, shared with the hub: in a
// terminal, by group with a mark per check and the tally; piped, a word per
// check.
func DoctorLines(checks []core.Check, tty bool) []string {
	w := 0
	for _, c := range checks {
		w = max(w, len(c.What))
	}
	var lines []string
	group := ""
	for _, c := range checks {
		if tty && c.Group != group {
			if group != "" {
				lines = append(lines, "")
			}
			lines = append(lines, ui.Dim.Render(c.Group))
			group = c.Group
		}
		if !tty {
			lines = append(lines, ui.Pad(c.Status, 4)+" "+ui.Pad(c.What, w)+"  "+c.Detail)
			continue
		}
		mark := map[string]string{
			"ok":   ui.OKMark.Render("✓"),
			"warn": ui.Warn.Render("!"),
			"fail": ui.Err.Render("✗"),
		}[c.Status]
		lines = append(lines, mark+" "+ui.Pad(c.What, w)+"  "+ui.Dim.Render(c.Detail))
	}
	if tty {
		lines = append(lines, "", ui.Dim.Render(core.Tally(checks)))
	}
	return lines
}

func whereCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "where",
		Short:   "show where dehv is installed",
		Long:    "Where dehv is installed: the command, its folder, where it comes from and the version. Piped, only the folder's path.",
		GroupID: "dehv",
		Example: `cd "$(dehv where)"`,
		Args:    cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			in := core.ThisInstall()
			// Only the folder when piped, so cd "$(dehv where)" works.
			if !ui.IsTTY(ui.Stdout) {
				ui.Println(in.Dir)
				return nil
			}
			for _, l := range WhereLines(in) {
				ui.Println(l)
			}
			return nil
		},
	}
}

// WhereLines is what `dehv where` shows in a terminal.
func WhereLines(in core.Install) []string {
	from := map[string]string{
		core.FromHomebrew: "Homebrew " + ui.Dim.Render("· brew upgrade dehv"),
		core.FromSource:   "a build from source " + ui.Dim.Render("· go install ./cmd/dehv"),
		core.FromFolder:   "a folder",
	}[in.From]
	version := in.Version
	if in.Rev != "" {
		version += " " + ui.Dim.Render("· "+in.Rev)
	}
	return []string{
		ui.Name.Render("command") + "  " + config.Tilde(in.Bin),
		ui.Name.Render("folder ") + "  " + ui.Generic.Render(config.Tilde(in.Dir)),
		ui.Name.Render("from   ") + "  " + from,
		ui.Name.Render("version") + "  " + version,
	}
}

func tapCmd() *cobra.Command {
	var kinds []string
	for _, k := range core.TapKinds {
		kinds = append(kinds, k.Kind)
	}
	return &cobra.Command{
		Use:       "tap [" + strings.Join(kinds, "|") + "]",
		Short:     "the steps to release to my homebrew-tap",
		Long:      "The steps to release a new version to my homebrew-tap: in the repo, in the tap and on this Mac. It only prints them; piped, as a script whose notes are comments.",
		GroupID:   "dehv",
		ValidArgs: kinds,
		Args:      cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := ""
			if len(args) == 1 {
				kind = args[0]
				ok := false
				for _, k := range kinds {
					ok = ok || k == kind
				}
				if !ok {
					return misuse(cmd, "'"+kind+"' is not "+strings.Join(kinds, ", "))
				}
			} else {
				if err := needsTTY(cmd, "no terminal to ask on: say "+strings.Join(kinds, ", ")); err != nil {
					return err
				}
				var opts []string
				for _, k := range core.TapKinds {
					opts = append(opts, ui.Pad(k.Kind, 11)+ui.Dim.Render(k.About))
				}
				i, err := ui.Pick("What are you releasing?", opts)
				if err != nil {
					return err
				}
				kind = core.TapKinds[i].Kind
			}
			for _, l := range TapLines(kind, ui.IsTTY(ui.Stdout)) {
				ui.Println(l)
			}
			return nil
		},
	}
}

// TapLines is the steps for kind: in a terminal notes dimmed, generic names in
// yellow and commands never cut, so they copy whole; piped, a script.
func TapLines(kind string, tty bool) []string {
	about, rows := core.TapSteps(kind)
	generic := func(s string) string {
		if !tty {
			return s
		}
		return GenericNames(s)
	}
	var lines []string
	if tty {
		lines = append(lines, ui.Name.Render("dehv tap "+kind)+" "+ui.Dim.Render("· "+about), "")
	} else {
		lines = append(lines, "# dehv tap "+kind+": "+about, "")
	}
	for _, r := range rows {
		switch r.Kind {
		case "h":
			if tty {
				l := ui.Bold.Render(r.Text)
				if r.Note != "" {
					l += " " + ui.Dim.Render("·") + " " + generic(r.Note)
				}
				lines = append(lines, l)
			} else if r.Note != "" {
				lines = append(lines, "# "+r.Text+": "+r.Note)
			} else {
				lines = append(lines, "# "+r.Text)
			}
		case "s":
			n, text, _ := strings.Cut(r.Text, " ")
			if tty {
				lines = append(lines, "  "+ui.Name.Render(n)+" "+ui.Dim.Render(text))
			} else {
				lines = append(lines, "# "+r.Text)
			}
		case "c":
			if tty {
				lines = append(lines, "    "+generic(r.Text))
			} else {
				lines = append(lines, r.Text)
			}
		case "e":
			if tty {
				lines = append(lines, "    "+generic(r.Text))
			} else {
				lines = append(lines, "#   "+r.Text)
			}
		default:
			lines = append(lines, "")
		}
	}
	return lines
}

// GenericNames styles the generic names of s, like <name>, in yellow, for
// whoever copies the line to know what to replace.
func GenericNames(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, "<")
		j := strings.Index(s, ">")
		if i < 0 || j < i {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i] + ui.Generic.Render(s[i:j+1]))
		s = s[j+1:]
	}
}
