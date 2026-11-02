// Package ui is how one-shot commands talk: One Dark's colours as semantic
// tokens, the marks for what a command did, tables, pickers and a spinner.
// Everything goes through lipgloss' writers, so colours are downsampled to
// what the terminal has and dropped when it is not one, or with NO_COLOR.
package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// One Dark's palette, the one dehv always had.
var (
	Fg     color.Color = lipgloss.Color("#abb2bf")
	Muted  color.Color = lipgloss.Color("#7f848e")
	Red    color.Color = lipgloss.Color("#e06c75")
	Green  color.Color = lipgloss.Color("#98c379")
	Yellow color.Color = lipgloss.Color("#e5c07b")
	Blue   color.Color = lipgloss.Color("#61afef")
	Purple color.Color = lipgloss.Color("#c678dd")
	Cyan   color.Color = lipgloss.Color("#56b6c2")
	SelBg  color.Color = lipgloss.Color("#3e4451")
	Bg     color.Color = lipgloss.Color("#282c34")
)

// What each colour means: commands style through these, never the hex above.
var (
	Bold    = lipgloss.NewStyle().Bold(true)
	Dim     = lipgloss.NewStyle().Foreground(Muted)
	Accent  = lipgloss.NewStyle().Foreground(Blue)
	Name    = lipgloss.NewStyle().Foreground(Cyan)
	Branch  = lipgloss.NewStyle().Foreground(Purple)
	Generic = lipgloss.NewStyle().Foreground(Yellow)
	OKMark  = lipgloss.NewStyle().Foreground(Green)
	Warn    = lipgloss.NewStyle().Foreground(Yellow)
	Err     = lipgloss.NewStyle().Foreground(Red)
)
