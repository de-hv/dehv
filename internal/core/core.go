// Package core is what dehv does, without any of how it looks: the one-shot
// commands and the hub both call it, and its tests need no terminal.
package core

import "errors"

// Error is something a command could not do: what, and why or what to try.
type Error struct {
	Msg  string
	Hint string
}

func (e *Error) Error() string {
	if e.Hint == "" {
		return e.Msg
	}
	return e.Msg + " · " + e.Hint
}

// Fail makes an Error.
func Fail(msg, hint string) error { return &Error{Msg: msg, Hint: hint} }

// ErrNoTerminal is a Reporter that cannot ask.
var ErrNoTerminal = errors.New("no terminal to ask in")

// Reporter is how long work says how it goes, and asks before it removes
// anything: a spinner and prompts for a command, the status line and a dialog
// in the hub.
type Reporter interface {
	// Step says what it is doing now.
	Step(msg string)
	// OK says what it did, with a detail.
	OK(msg, detail string)
	// Noop says there was nothing to do.
	Noop(msg string)
	// Confirm asks before something that removes, listing what; false for a
	// no, ErrNoTerminal when it cannot ask.
	Confirm(what, ask string, items []string) (bool, error)
}

// Quiet is a Reporter that says nothing and never agrees.
type Quiet struct{}

func (Quiet) Step(string)       {}
func (Quiet) OK(string, string) {}
func (Quiet) Noop(string)       {}
func (Quiet) Confirm(string, string, []string) (bool, error) {
	return false, ErrNoTerminal
}
