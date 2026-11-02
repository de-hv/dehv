package ui

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

// Where commands write; tests swap them.
var (
	Stdout io.Writer = os.Stdout
	Stderr io.Writer = os.Stderr
	Stdin  io.Reader = os.Stdin
)

// IsTTY says whether w is a terminal.
func IsTTY(w any) bool {
	f, ok := w.(term.File)
	return ok && term.IsTerminal(f.Fd())
}

// Interactive says whether a picker or a prompt can be shown: both stdin and
// stdout are terminals.
func Interactive() bool {
	return IsTTY(Stdin) && IsTTY(Stdout)
}

// Width is the terminal's width: what it says, $COLUMNS, or 80.
func Width() int {
	if f, ok := Stdout.(term.File); ok {
		if w, _, err := term.GetSize(f.Fd()); err == nil && w > 0 {
			return w
		}
	}
	if n, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && n > 0 {
		return n
	}
	return 80
}

// Println prints to stdout, downsampling colours to what it can show.
func Println(a ...any) { lipgloss.Fprintln(Stdout, a...) }

// Eprintln prints to stderr, the same way.
func Eprintln(a ...any) { lipgloss.Fprintln(Stderr, a...) }

// OK says what a command did: a green check, then msg, then a dimmed detail.
func OK(msg, detail string) {
	Println(OKMark.Render("✓") + " " + msg + suffix(detail))
}

// Fail says what it could not, on stderr: a red cross, msg, the cause dimmed.
func Fail(msg, detail string) {
	Eprintln(Err.Render("✗") + " " + msg + suffix(detail))
}

// Noop says there was nothing to do, dimmed.
func Noop(msg string) {
	Println(Dim.Render("· " + msg))
}

// Note is a dimmed line under what was done, indented.
func Note(msg string) {
	Println(Dim.Render("  " + msg))
}

func suffix(detail string) string {
	if detail == "" {
		return ""
	}
	return " " + Dim.Render("· "+detail)
}

// Cut cuts s to w cells with an …, keeping its colours.
func Cut(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return ansi.Truncate(s, w, "…")
}

// Pad pads s with spaces to w cells; measured without its colours.
func Pad(s string, w int) string {
	if n := w - lipgloss.Width(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// Table is rows of cells under a dimmed header, each column as wide as its
// widest cell, two spaces apart, the last one cut to fit the terminal. Piped,
// the same rows plain, as `column -t` gave them.
func Table(header []string, rows [][]string) {
	tty := IsTTY(Stdout)
	cols := len(header)
	widths := make([]int, cols)
	for i, h := range header {
		widths[i] = lipgloss.Width(h)
	}
	for _, r := range rows {
		for i := 0; i < cols && i < len(r); i++ {
			widths[i] = max(widths[i], lipgloss.Width(r[i]))
		}
	}
	line := func(cells []string) string {
		var b strings.Builder
		for i := 0; i < cols; i++ {
			c := ""
			if i < len(cells) {
				c = cells[i]
			}
			if i == cols-1 {
				b.WriteString(c)
			} else {
				b.WriteString(Pad(c, widths[i]) + "  ")
			}
		}
		s := strings.TrimRight(b.String(), " ")
		if tty {
			s = Cut(s, Width())
		}
		return s
	}
	if tty {
		Println(Dim.Render(line(header)))
	} else {
		Println(line(header))
	}
	for _, r := range rows {
		Println(line(r))
	}
}

// Card frames lines in a rounded border, title in its top edge; the only frame
// one-shot output still draws, for `dehv hi`.
func Card(lines []string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Muted).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
}

// Plural is "1 thing" or "n things".
func Plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
