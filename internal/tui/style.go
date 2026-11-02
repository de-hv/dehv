package tui

import (
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	"github.com/de-hv/dehv/internal/ui"
)

// The hub's own styles, on top of ui's tokens.
var (
	barStyle   = lipgloss.NewStyle().Foreground(ui.Blue)
	selStyle   = lipgloss.NewStyle().Background(ui.SelBg).Foreground(lipgloss.Color("#e6e9ef"))
	navActive  = lipgloss.NewStyle().Background(ui.Blue).Foreground(ui.Bg).Bold(true)
	navIdle    = lipgloss.NewStyle().Background(ui.SelBg).Foreground(ui.Fg).Bold(true)
	ruleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#3e4451"))
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.Muted).
			Padding(1, 2)
)

// newInput is a text field in the hub's colours.
func newInput(prompt, placeholder string) textinput.Model {
	in := textinput.New()
	in.Prompt = prompt
	in.Placeholder = placeholder
	s := in.Styles()
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(ui.Blue)
	s.Blurred.Prompt = lipgloss.NewStyle().Foreground(ui.Muted)
	s.Focused.Placeholder = lipgloss.NewStyle().Foreground(ui.Muted)
	s.Blurred.Placeholder = lipgloss.NewStyle().Foreground(ui.Muted)
	in.SetStyles(s)
	return in
}
