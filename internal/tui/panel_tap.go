package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/de-hv/dehv/internal/cli"
	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

// tapLine is a line of the steps; copy is the text c copies, "" for the
// notes and headings, which the selection skips.
type tapLine struct {
	text string
	copy string
}

// tap is `dehv tap`: the steps to release to my homebrew-tap, a kind at a
// time, with the commands selectable to copy them.
type tap struct {
	kind  int
	about string
	lines []tapLine
	sel   int // index in lines of the selected command
	off   int
}

func newTap() *tap {
	t := &tap{}
	t.build()
	return t
}

// build reads the steps of the kind shown and selects its first command.
func (t *tap) build() {
	about, rows := core.TapSteps(core.TapKinds[t.kind].Kind)
	t.about, t.lines, t.off = about, nil, 0
	for _, r := range rows {
		switch r.Kind {
		case "h":
			l := ui.Bold.Render(r.Text)
			if r.Note != "" {
				l += " " + ui.Dim.Render("·") + " " + cli.GenericNames(r.Note)
			}
			t.lines = append(t.lines, tapLine{text: l})
		case "s":
			n, text, _ := strings.Cut(r.Text, " ")
			t.lines = append(t.lines, tapLine{text: " " + ui.Name.Render(n) + " " + ui.Dim.Render(text)})
		case "c", "e":
			t.lines = append(t.lines, tapLine{text: "   " + cli.GenericNames(r.Text), copy: r.Text})
		default:
			t.lines = append(t.lines, tapLine{})
		}
	}
	t.sel = t.next(-1, 1)
}

// next is the command after (or before, d = -1) line i, or i at the end.
func (t *tap) next(i, d int) int {
	for j := i + d; j >= 0 && j < len(t.lines); j += d {
		if t.lines[j].copy != "" {
			return j
		}
	}
	return max(i, 0)
}

func (t *tap) name() string    { return "Tap" }
func (t *tap) badge() string   { return "" }
func (t *tap) heading() string { return core.TapKinds[t.kind].Kind + " · " + t.about }
func (t *tap) load() tea.Cmd   { return nil }

func (t *tap) update(tea.Msg, bool) tea.Cmd { return nil }

func (t *tap) key(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		t.sel = t.next(t.sel, -1)
	case "down", "j":
		t.sel = t.next(t.sel, 1)
	case "home", "g":
		t.sel = t.next(-1, 1)
	case "end", "G":
		t.sel = t.next(len(t.lines), -1)
	case "t":
		t.kind = (t.kind + 1) % len(core.TapKinds)
		t.build()
	case "T":
		t.kind = (t.kind + len(core.TapKinds) - 1) % len(core.TapKinds)
		t.build()
	case "c":
		line := t.lines[t.sel].copy
		if err := core.Copy(line); err != nil {
			return true, failed(err)
		}
		return true, say("ok", "copied "+line)
	default:
		return false, nil
	}
	return true, nil
}

func (t *tap) capturing() bool { return false }

func (t *tap) hints() []hint {
	return []hint{{"↑↓", "command"}, {"c", "copy"}, {"t", "kind"}}
}

func (t *tap) view(w, h int, focused bool) string {
	// The kinds as tabs, the one shown in brackets.
	var kinds []string
	for i, k := range core.TapKinds {
		if i == t.kind {
			kinds = append(kinds, ui.Accent.Bold(true).Render("["+k.Kind+"]"))
		} else {
			kinds = append(kinds, ui.Dim.Render(" "+k.Kind+" "))
		}
	}
	out := []string{" " + strings.Join(kinds, " ") + "  " + ui.Dim.Render("t next"), ""}

	rows := max(1, h-len(out))
	if t.sel < t.off {
		t.off = t.sel
	}
	if t.sel >= t.off+rows {
		t.off = t.sel - rows + 1
	}
	for i := t.off; i < len(t.lines) && i < t.off+rows; i++ {
		l := t.lines[i].text
		switch {
		case i == t.sel && focused:
			out = append(out, barStyle.Render("▌")+selStyle.Render(ui.Pad(ansi.Strip(ui.Cut(l, w-1)), w-1)))
		case i == t.sel:
			out = append(out, ui.Dim.Render("▌")+ui.Cut(l, w-1))
		default:
			out = append(out, " "+ui.Cut(l, w-1))
		}
	}
	return strings.Join(out, "\n")
}
