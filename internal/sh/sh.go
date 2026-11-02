// Package sh runs the programs dehv leans on (git, lsof, osascript…) and
// gives back what they printed.
package sh

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
)

// Cmd is one program to run.
type Cmd struct {
	Ctx  context.Context
	Dir  string
	Env  []string // added to the environment
	Name string
	Args []string
}

// Error is a program that failed, with what it said on stderr and stdout.
type Error struct {
	Cmd    string
	Output string
	Err    error
}

func (e *Error) Error() string {
	if out := strings.TrimSpace(e.Output); out != "" {
		return e.Cmd + ": " + out
	}
	return e.Cmd + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

// Run runs it, stdin empty, and gives back stdout, trimmed at the end. On a
// failure the error carries stderr and stdout both.
func (c Cmd) Run() (string, error) {
	ctx := c.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Dir = c.Dir
	if len(c.Env) > 0 {
		cmd.Env = append(os.Environ(), c.Env...)
	}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return out.String(), &Error{Cmd: c.Name, Output: errb.String() + out.String(), Err: err}
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

// Out runs name with args in the current folder.
func Out(name string, args ...string) (string, error) {
	return Cmd{Name: name, Args: args}.Run()
}

// Git runs git in dir ("" for the current folder), never prompting for a
// login: it would wait unseen behind a spinner or the hub.
func Git(ctx context.Context, dir string, args ...string) (string, error) {
	return Cmd{
		Ctx:  ctx,
		Dir:  dir,
		Env:  []string{"GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND=" + sshCommand()},
		Name: "git",
		Args: args,
	}.Run()
}

func sshCommand() string {
	if s := os.Getenv("GIT_SSH_COMMAND"); s != "" {
		return s
	}
	return "ssh -o BatchMode=yes"
}

// Output is what a failed program said, or "".
func Output(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Output
	}
	return ""
}

// Lines splits s in lines, none when it is empty.
func Lines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// Has says whether name is in the PATH.
func Has(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
