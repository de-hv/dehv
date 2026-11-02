package cli

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

// UsageError is a command called the wrong way: the error, then its usage
// line, exit code 2.
type UsageError struct {
	Cmd *cobra.Command
	Msg string
}

func (e *UsageError) Error() string { return e.Msg }

func misuse(cmd *cobra.Command, format string) error {
	return &UsageError{Cmd: cmd, Msg: format}
}

// errSilent is a failure already reported, line by line.
var errSilent = errors.New("")

// handleError prints err the dehv way. Fang calls it for whatever a command
// returned.
func handleError(_ io.Writer, err error) {
	var ue *UsageError
	var ce *core.Error
	switch {
	case errors.Is(err, errSilent), errors.Is(err, ui.ErrCancelled),
		errors.Is(err, core.ErrDeclined), errors.Is(err, context.Canceled):
	case errors.As(err, &ue):
		ui.Eprintln(ue.Cmd.CommandPath() + ": " + ue.Msg)
		ui.Eprintln(ui.Dim.Render("Usage: " + ue.Cmd.UseLine()))
	case errors.As(err, &ce):
		hint, more, _ := strings.Cut(ce.Hint, "\n")
		ui.Fail(ce.Msg, hint)
		if more != "" {
			ui.Eprintln(ui.Dim.Render(more))
		}
	case isCobraUsage(err):
		ui.Eprintln("dehv: " + err.Error())
		ui.Eprintln(ui.Dim.Render("Run 'dehv --help' for usage."))
	default:
		ui.Fail(err.Error(), "")
	}
}

// ExitCode is what dehv exits with for err: 2 for a misuse, 130 when stopped
// with ctrl+c, 1 for the rest.
func ExitCode(err error) int {
	var ue *UsageError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &ue), isCobraUsage(err):
		return 2
	case errors.Is(err, context.Canceled):
		return 130
	}
	return 1
}

func isCobraUsage(err error) bool {
	s := err.Error()
	for _, p := range []string{"flag needs an argument", "unknown flag", "unknown shorthand flag", "unknown command", "invalid argument", "accepts ", "requires "} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
