// Package cli is dehv's commands: `dehv <command> [options] [args]`, each a
// one-shot that a script can call, and `dehv` alone, which opens the hub in
// a terminal and prints the list of commands anywhere else.
package cli

import (
	"context"
	"image/color"
	"io"
	"os"
	"strings"

	"charm.land/fang/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

// RunHub opens the full-screen hub; main sets it, so this package never
// needs a terminal in its tests.
var RunHub func(ctx context.Context) error

// The groups `dehv` and `dehv --help` list the commands in.
var groups = []*cobra.Group{
	{ID: "git", Title: "Git"},
	{ID: "system", Title: "System"},
	{ID: "agents", Title: "Agents"},
	{ID: "media", Title: "Media"},
	{ID: "dehv", Title: "dehv"},
}

// Root is the dehv command with every other one under it.
func Root() *cobra.Command {
	var plain bool
	root := &cobra.Command{
		Use:   "dehv",
		Short: "my own desktop toolkit",
		Long: "My own desktop toolkit: every personal command behind one name.\n\n" +
			"Alone in a terminal it opens the hub; piped, or with --plain, it lists the commands.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !plain && os.Getenv("NO_TUI") == "" && ui.Interactive() && RunHub != nil {
				return RunHub(cmd.Context())
			}
			printList(cmd)
			return nil
		},
	}
	root.Flags().BoolVar(&plain, "plain", false, "list the commands instead of opening the hub")
	root.AddGroup(groups...)
	root.AddCommand(
		asCmd(), branchesCmd(),
		portCmd(), ipCmd(), hiCmd(),
		skillCmd(),
		wallCmd(),
		doctorCmd(), whereCmd(), tapCmd(),
	)
	return root
}

// Execute runs dehv with args, the way main does.
func Execute(ctx context.Context, args []string) error {
	root := Root()
	root.SetArgs(args)
	return fang.Execute(ctx, root,
		fang.WithVersion(core.Version),
		fang.WithColorSchemeFunc(scheme),
		fang.WithErrorHandler(func(w io.Writer, _ fang.Styles, err error) { handleError(w, err) }),
		fang.WithNotifySignal(os.Interrupt),
	)
}

// scheme is fang's help in One Dark.
func scheme(lipgloss.LightDarkFunc) fang.ColorScheme {
	return fang.ColorScheme{
		Base:           ui.Fg,
		Title:          ui.Blue,
		Description:    ui.Fg,
		Codeblock:      ui.Bg,
		Program:        ui.Cyan,
		DimmedArgument: ui.Muted,
		Comment:        ui.Muted,
		Flag:           ui.Yellow,
		FlagDefault:    ui.Muted,
		Command:        ui.Cyan,
		QuotedString:   ui.Green,
		Argument:       ui.Fg,
		Help:           ui.Muted,
		Dash:           ui.Muted,
		ErrorHeader:    [2]color.Color{ui.Bg, ui.Red},
		ErrorDetails:   ui.Red,
	}
}

// printList is `dehv` outside a terminal: the commands by group, what each
// is for, and whether `dehv doctor` passes.
func printList(root *cobra.Command) {
	tty := ui.IsTTY(ui.Stdout)
	head := ui.Accent.Bold(true).Render("dehv") + "  " + ui.Dim.Render("my own desktop toolkit")
	right := ui.Dim.Render(core.DisplayVersion())
	if core.Healthy(core.Doctor()) {
		right += "  " + ui.OKMark.Render("✓") + ui.Dim.Render(" doctor")
	} else {
		right += "  " + ui.Err.Render("✗") + ui.Dim.Render(" dehv doctor")
	}
	if tty {
		gap := max(2, min(ui.Width(), 72)-lipgloss.Width(head)-lipgloss.Width(right))
		ui.Println(head + strings.Repeat(" ", gap) + right)
	} else {
		ui.Println("dehv - my own desktop toolkit\n\nUsage: dehv <command> [options] [args]")
	}
	ui.Println()

	nameW, groupW := 0, 0
	for _, c := range root.Commands() {
		if c.GroupID != "" {
			nameW = max(nameW, len(c.Name()))
		}
	}
	for _, g := range groups {
		groupW = max(groupW, len(g.Title))
	}
	for _, g := range groups {
		first := true
		for _, c := range root.Commands() {
			if c.GroupID != g.ID || c.Hidden {
				continue
			}
			label := ""
			if first {
				label = g.Title
				first = false
			}
			line := " " + ui.Pad(ui.Dim.Render(label), groupW) + "  " + ui.Pad(ui.Name.Render(c.Name()), nameW) + "  " + c.Short
			if tty {
				line = ui.Cut(line, ui.Width())
			}
			ui.Println(line)
		}
	}
	ui.Println()
	ui.Println(ui.Dim.Render("dehv <command> --help for its options"))
}

// needsTTY fails a command that has to ask but has no terminal to ask in.
func needsTTY(cmd *cobra.Command, what string) error {
	if ui.Interactive() {
		return nil
	}
	return misuse(cmd, what)
}
