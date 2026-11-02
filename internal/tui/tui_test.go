package tui

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/de-hv/dehv/internal/core"
)

var update = flag.Bool("update", false, "rewrite the golden frames")

// fixture is the hub with known data in every panel, at w×h.
func fixture(t *testing.T, w, h int) model {
	t.Helper()
	core.Version = "1.0.0"
	m := newModel(context.Background())
	msgs := []tea.Msg{
		tea.WindowSizeMsg{Width: w, Height: h},
		repoMsg{core.Repo{Dir: "~/dev/dehv", Branch: "main", Profile: "personal"}},
		portsLoaded{ls: []core.Listener{
			{Port: 3000, Program: "node", PID: 4821, User: "david"},
			{Port: 5173, Program: "vite", PID: 4907, User: "david"},
			{Port: 5432, Program: "postgres", PID: 612, User: "david"},
		}},
		branchesLoaded{inRepo: true, bs: core.Branches{Base: "main", Current: "main", List: []core.Branch{
			{Name: "fix/login-redirect", Why: core.Merged, Age: "3 days ago"},
			{Name: "feat/rate-limit", Why: core.Gone, Age: "2 weeks ago"},
			{Name: "spike/hub", Why: core.Unmerged, Age: "5 hours ago"},
		}}},
		profilesLoaded{inRepo: true, cur: "personal", ps: []core.Profile{
			{Name: "personal", User: "David Higuita", Email: "david@example.com"},
			{Name: "acme", User: "David Higuita", Email: "david@acme.com"},
		}},
		doctorLoaded{checks: []core.Check{{Group: "install", Status: "ok", What: "command", Detail: "/opt/homebrew/bin/dehv"}}},
	}
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(model)
	}
	return m
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	file := filepath.Join("testdata", name+".golden")
	if *update {
		_ = os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("%v (go test ./internal/tui -update writes it)", err)
	}
	if string(want) != got {
		t.Errorf("%s differs from its golden frame:\n%s", name, got)
	}
}

// frame is the view without colours, every line trimmed at the right.
func frame(m model) string {
	lines := strings.Split(ansi.Strip(m.render()), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

func TestFrames(t *testing.T) {
	for _, size := range []struct {
		name string
		w, h int
	}{
		{"wide", 120, 20}, {"standard", 80, 24}, {"narrow", 60, 20}, {"too-small", 49, 11},
	} {
		t.Run(size.name, func(t *testing.T) {
			golden(t, "ports-"+size.name, frame(fixture(t, size.w, size.h)))
		})
	}
}

func press(m model, keys ...string) model {
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "tab":
			msg = tea.KeyPressMsg{Code: tea.KeyTab}
		case "esc":
			msg = tea.KeyPressMsg{Code: tea.KeyEscape}
		case "space":
			msg = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
		case "left":
			msg = tea.KeyPressMsg{Code: tea.KeyLeft}
		case "right":
			msg = tea.KeyPressMsg{Code: tea.KeyRight}
		case "up":
			msg = tea.KeyPressMsg{Code: tea.KeyUp}
		case "down":
			msg = tea.KeyPressMsg{Code: tea.KeyDown}
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "shift+tab":
			msg = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
		default:
			r := []rune(k)[0]
			msg = tea.KeyPressMsg{Code: r, Text: k}
		}
		next, cmd := m.Update(msg)
		m = next.(model)
		// A dialog opens through a command: run it, but nothing else, which
		// would touch this Mac.
		if cmd != nil {
			if om, ok := cmd().(openMsg); ok {
				next, _ = m.Update(om)
				m = next.(model)
			}
		}
	}
	return m
}

func TestJumpAndConfirm(t *testing.T) {
	m := fixture(t, 100, 24)
	m = press(m, "2")
	if m.active != 1 {
		t.Fatalf("2 shows Branches, got panel %d", m.active)
	}
	golden(t, "branches-standard", frame(m))

	// space unticks the selected one; d asks before deleting the rest.
	m = press(m, "space", "d")
	if m.modal == nil {
		t.Fatal("d opens a confirmation")
	}
	golden(t, "branches-confirm", frame(m))
	m = press(m, "n")
	if m.modal != nil {
		t.Fatal("n closes it")
	}
}

func TestQuitLadder(t *testing.T) {
	m := fixture(t, 100, 24)
	m = press(m, "/", "n", "o")
	if !m.panels[0].capturing() {
		t.Fatal("/ starts the filter")
	}
	// q is filter text while filtering, not quit.
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	m = next.(model)
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Fatal("q quit while typing a filter")
		}
	}
	m = press(m, "esc")
	if m.panels[0].capturing() {
		t.Fatal("esc leaves the filter")
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("q quits")
	}
	if _, quit := cmd().(tea.QuitMsg); !quit {
		t.Fatal("q quits")
	}
}

func TestHelpOverlay(t *testing.T) {
	m := press(fixture(t, 100, 30), "?")
	if m.modal == nil {
		t.Fatal("? opens the help")
	}
	golden(t, "help", frame(m))
	m = press(m, "esc")
	if m.modal != nil {
		t.Fatal("esc closes the help")
	}
}

func TestToolsFocus(t *testing.T) {
	// In the sidebar: ← focuses the tools, ↑↓ choose one, → gives it back.
	m := press(fixture(t, 120, 20), "left")
	if !m.onNav {
		t.Fatal("← focuses the tools")
	}
	m = press(m, "down", "down")
	if m.active != 2 || !m.onNav {
		t.Fatalf("↓↓ moves to Profiles and keeps the focus: panel %d, onNav %v", m.active, m.onNav)
	}
	golden(t, "tools-focused-wide", frame(m))
	m = press(m, "x") // a panel key does nothing while the tools have the focus
	if m.modal != nil {
		t.Fatal("x reached the panel")
	}
	m = press(m, "up", "up", "up")
	if m.active != 0 {
		t.Fatalf("↑ stops at the first tool, got %d", m.active)
	}
	m = press(m, "right")
	if m.onNav {
		t.Fatal("→ gives the focus back to the panel")
	}
	// tab and shift+tab still change the tool, the focus where it was.
	m = press(m, "tab", "tab", "shift+tab")
	if m.active != 1 || m.onNav {
		t.Fatalf("tab, tab, shift+tab: panel %d, onNav %v", m.active, m.onNav)
	}

	// In the tabs: ←→ choose, ↓ or enter gives the focus back.
	m = press(fixture(t, 80, 24), "left", "right", "right")
	if m.active != 2 || !m.onNav {
		t.Fatalf("→→ in the tabs: panel %d, onNav %v", m.active, m.onNav)
	}
	golden(t, "tools-focused-standard", frame(m))
	m = press(m, "enter")
	if m.onNav {
		t.Fatal("enter gives the focus back")
	}
}

func TestTap(t *testing.T) {
	m := press(fixture(t, 120, 24), "7")
	tp, ok := m.panels[m.active].(*tap)
	if !ok {
		t.Fatalf("7 is Tap, got %s", m.panels[m.active].name())
	}
	first := tp.sel
	if tp.lines[first].copy == "" {
		t.Fatal("the selection starts on a command")
	}
	m = press(m, "down")
	if tp.sel <= first || tp.lines[tp.sel].copy == "" {
		t.Fatal("↓ goes to the next command, past the notes")
	}
	golden(t, "tap-wide", frame(m))
	m = press(m, "t")
	if core.TapKinds[tp.kind].Kind != "cask" || tp.lines[tp.sel].copy == "" {
		t.Fatalf("t shows the next kind from its first command: %s", core.TapKinds[tp.kind].Kind)
	}
	if m.panels[7].name() != "Doctor" {
		t.Fatal("Doctor is the last one")
	}
}
