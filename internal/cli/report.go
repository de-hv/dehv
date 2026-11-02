package cli

import (
	"fmt"

	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

// reporter is core.Reporter for a one-shot command: a spinner on stderr while
// it works, the usual marks for what it did, and a yes/no prompt.
type reporter struct {
	spin *ui.Spinner
}

func (r *reporter) Step(msg string) {
	if r.spin == nil {
		r.spin = ui.Spin(msg)
		return
	}
	r.spin.Set(msg)
}

// done stops the spinner; call it before printing anything else.
func (r *reporter) done() {
	r.spin.Stop()
	r.spin = nil
}

func (r *reporter) OK(msg, detail string) {
	r.done()
	ui.OK(msg, detail)
}

func (r *reporter) Noop(msg string) {
	r.done()
	ui.Noop(msg)
}

func (r *reporter) Confirm(what, ask string, items []string) (bool, error) {
	r.done()
	if !ui.Interactive() {
		return false, core.ErrNoTerminal
	}
	ui.Println(ui.Warn.Render("!") + " " + what)
	for i, it := range items {
		if i == 10 {
			ui.Println(ui.Dim.Render(fmt.Sprintf("  … and %d more", len(items)-10)))
			break
		}
		ui.Println("  " + ui.Err.Render("-") + " " + it)
	}
	return ui.Confirm(ask)
}

var _ core.Reporter = (*reporter)(nil)
