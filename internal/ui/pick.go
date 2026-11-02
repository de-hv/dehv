package ui

import (
	"errors"
	"os"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// ErrCancelled is a picker or prompt left with esc or ctrl+c.
var ErrCancelled = errors.New("cancelled")

// Theme is huh's base theme in One Dark's colours. It ignores the terminal's
// background: dehv's palette is a dark one.
var Theme = huh.ThemeFunc(func(bool) *huh.Styles {
	t := huh.ThemeBase(true)
	t.Focused.Title = t.Focused.Title.Foreground(Fg).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(Muted)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(Red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(Red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(Blue).SetString("▌ ")
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(Blue).SetString("▌ ")
	t.Focused.Option = t.Focused.Option.Foreground(Fg)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(Fg)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(Fg)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(Green).SetString("[x] ")
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(Muted).SetString("[ ] ")
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(Bg).Background(Blue)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(Fg).Background(SelBg)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(Blue)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(Muted)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(Muted)
	t.Help.ShortKey = t.Help.ShortKey.Foreground(Blue)
	t.Help.ShortDesc = t.Help.ShortDesc.Foreground(Muted)
	t.Help.ShortSeparator = t.Help.ShortSeparator.Foreground(Muted)
	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description
	return t
})

// run shows a one-field form inline, drawn on stderr so stdout stays the
// command's answer, and erased when done.
func run(field huh.Field) error {
	err := huh.NewForm(huh.NewGroup(field)).
		WithTheme(Theme).
		WithOutput(os.Stderr).
		WithShowHelp(true).
		Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return ErrCancelled
	}
	return err
}

// Pick asks for one of options; the index chosen.
func Pick(title string, options []string) (int, error) {
	opts := make([]huh.Option[int], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, i)
	}
	choice := 0
	err := run(huh.NewSelect[int]().Title(title).Options(opts...).Value(&choice))
	return choice, err
}

// PickMany asks to tick some of options, those in ticked to begin with; the
// indexes left ticked.
func PickMany(title string, options []string, ticked []int) ([]int, error) {
	on := map[int]bool{}
	for _, i := range ticked {
		on[i] = true
	}
	opts := make([]huh.Option[int], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, i).Selected(on[i])
	}
	var chosen []int
	err := run(huh.NewMultiSelect[int]().Title(title).Options(opts...).Value(&chosen))
	return chosen, err
}

// Confirm asks a yes/no question, no by default.
func Confirm(question string) (bool, error) {
	yes := false
	err := run(huh.NewConfirm().Title(question).Affirmative("Yes").Negative("No").Value(&yes))
	return yes, err
}

// Input asks for a line, value prefilled.
func Input(title, value string, validate func(string) error) (string, error) {
	in := huh.NewInput().Title(title).Value(&value)
	if validate != nil {
		in = in.Validate(validate)
	}
	err := run(in)
	return value, err
}
