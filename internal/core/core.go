// Package core is what dehv does, without any of how it looks: the one-shot
// commands and the hub both call it, and its tests need no terminal.
package core

import "fmt"

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

// Plural is "1 thing" or "n things".
func Plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
