package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/de-hv/dehv/internal/ui"
)

// run runs dehv with args, stdout and stderr not terminals, in a home of its
// own: what it printed on each, and the exit code.
func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	t.Setenv("NO_COLOR", "1")
	var out, errb bytes.Buffer
	ui.Stdout, ui.Stderr, ui.Stdin = &out, &errb, strings.NewReader("")
	t.Cleanup(func() { ui.Stdout, ui.Stderr, ui.Stdin = nil, nil, nil })
	err := Execute(context.Background(), args)
	return out.String(), errb.String(), ExitCode(err)
}

func TestList(t *testing.T) {
	out, _, code := run(t)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"Usage: dehv <command>", " Git ", "as", "branches", "skill", "--help for its options"} {
		if !strings.Contains(out, want) {
			t.Errorf("the list has no %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b[") {
		t.Error("piped, the list has no colours")
	}
}

func TestMisuse(t *testing.T) {
	for _, c := range []struct {
		args []string
		err  string
	}{
		{[]string{"port", "foo"}, "dehv port: invalid port 'foo'"},
		{[]string{"port", "-a", "3000"}, "dehv port: -a goes alone"},
		{[]string{"as", "-l", "-c"}, "dehv as: -l goes alone"},
		{[]string{"as", "3"}, "dehv as: <n> goes with -a"},
		{[]string{"as", "Acme"}, "invalid profile 'Acme'"},
		{[]string{"skill", "a/b"}, "invalid name 'a/b'"},
		{[]string{"skill", "-a", "x"}, "-a goes alone"},
		{[]string{"tap", "brew"}, "'brew' is not goreleaser, cask, formula"},
		{[]string{"tap"}, "no terminal to ask on"},
		{[]string{"nope"}, "unknown command"},
		{[]string{"ip", "--what"}, "unknown flag"},
	} {
		_, errOut, code := run(t, c.args...)
		if code != 2 {
			t.Errorf("%v: exit %d, want 2", c.args, code)
		}
		if !strings.Contains(errOut, c.err) {
			t.Errorf("%v: stderr %q has no %q", c.args, errOut, c.err)
		}
	}
}

func TestFailures(t *testing.T) {
	for _, c := range []struct {
		args []string
		err  string
	}{
		{[]string{"as", "-l"}, "✗ no profiles yet · 'dehv as -n <profile>' or 'dehv as -i'"},
		{[]string{"as", "-r", "acme"}, "✗ no profile acme"},
		{[]string{"skill", "notes"}, "does not exist"},
	} {
		_, errOut, code := run(t, c.args...)
		if code != 1 {
			t.Errorf("%v: exit %d, want 1", c.args, code)
		}
		if !strings.Contains(errOut, c.err) {
			t.Errorf("%v: stderr %q has no %q", c.args, errOut, c.err)
		}
	}
}

func TestPipedOutput(t *testing.T) {
	out, _, code := run(t, "tap", "formula")
	if code != 0 || !strings.HasPrefix(out, "# dehv tap formula:") || !strings.Contains(out, "\ngit tag v<version>") {
		t.Errorf("piped, tap is a script:\n%s", out)
	}
	out, _, _ = run(t, "where")
	if strings.Count(out, "\n") != 1 || !strings.HasPrefix(out, "/") {
		t.Errorf("piped, where is only the folder: %q", out)
	}
	out, _, code = run(t, "as", "-i")
	if code != 0 || !strings.Contains(out, "✓ created") {
		t.Errorf("as -i creates the file: %q", out)
	}
}

func TestHelpWithoutArgs(t *testing.T) {
	// A command that needs an argument shows its help alone, and exits 0.
	for _, cmd := range []string{"port", "as", "skill"} {
		if _, _, code := run(t, cmd); code != 0 {
			t.Errorf("dehv %s: exit %d", cmd, code)
		}
	}
}
