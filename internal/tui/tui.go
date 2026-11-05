// Package tui is the hub `dehv` opens in a terminal: the tools on the left
// (or a tab row when narrow), the selected one's panel on the right, what
// just happened in the status line and the keys that matter in the footer.
// Every panel calls the same core the commands do.
package tui

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

// The smallest terminal the hub draws in; below it, it says so.
const minW, minH = 50, 12

// panel is one tool of the hub.
type panel interface {
	name() string
	badge() string   // a count or a mark beside its name, or ""
	heading() string // what it shows, beside its name above it
	load() tea.Cmd   // reads its data; the result comes back as a message
	update(msg tea.Msg, active bool) tea.Cmd
	key(msg tea.KeyPressMsg) (bool, tea.Cmd)
	capturing() bool // a filter is being typed: every key is its
	hints() []hint
	view(w, h int, focused bool) string // focused: keys go to it, not to the tools
}

// hint is a key and what it does, for the footer and the help.
type hint struct{ key, what string }

// Status line messages.
type statusMsg struct{ kind, text string } // kind: ok, err, warn, info
type busyMsg struct{ text string }         // "" when done
type fadeMsg struct{ id int }
type repoMsg struct{ repo core.Repo }

func say(kind, text string) tea.Cmd {
	return func() tea.Msg { return statusMsg{kind, text} }
}

// failed puts err in the status line, as `✗ what · why`.
func failed(err error) tea.Cmd {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	var ce *core.Error
	if errors.As(err, &ce) {
		hint, _, _ := strings.Cut(ce.Hint, "\n")
		if hint != "" {
			return say("err", ce.Msg+" · "+hint)
		}
		return say("err", ce.Msg)
	}
	return say("err", err.Error())
}

func busy(text string) tea.Cmd { return func() tea.Msg { return busyMsg{text} } }

// hubCtx is the running hub's context, cancelled when it is stopped, for
// work in goroutines.
var hubCtx = context.Background()

type logEntry struct {
	at         time.Time
	kind, text string
}

type model struct {
	panels []panel
	active int
	w, h   int
	repo   core.Repo

	status   statusMsg
	statusID int
	log      []logEntry
	working  string
	spin     spinner.Model
	modal    modal
	onNav    bool // the tools have the focus: the arrows move between them
}

func newModel() model {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	sp.Style = lipgloss.NewStyle().Foreground(ui.Blue)
	return model{
		panels: []panel{
			newPorts(), newBranches(), newProfiles(), newSkills(),
			newMachine(), newTap(), newDoctor(),
		},
		spin: sp,
	}
}

// Run opens the hub until it is quit.
func Run(ctx context.Context) error {
	p := tea.NewProgram(newModel(), tea.WithContext(ctx))
	hubCtx = ctx
	_, err := p.Run()
	if errors.Is(err, tea.ErrProgramKilled) || errors.Is(err, tea.ErrInterrupted) {
		return nil
	}
	return err
}

func loadRepo() tea.Msg { return repoMsg{core.ThisRepo()} }

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{loadRepo, m.spin.Tick}
	for _, p := range m.panels {
		cmds = append(cmds, p.load())
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case tea.KeyPressMsg:
		return m.onKey(msg)
	case tea.ResumeMsg:
		return m, tea.Batch(loadRepo, m.panels[m.active].load())
	case repoMsg:
		m.repo = msg.repo
		return m, nil
	case openMsg:
		m.modal = msg.m
		return m, nil
	case busyMsg:
		m.working = msg.text
		return m, nil
	case statusMsg:
		m.working = ""
		m.status = msg
		m.statusID++
		m.log = append(m.log, logEntry{time.Now(), msg.kind, msg.text})
		if len(m.log) > 200 {
			m.log = m.log[len(m.log)-200:]
		}
		// Errors and warnings stay until the next message; the rest fade.
		if msg.kind == "ok" || msg.kind == "info" {
			id := m.statusID
			return m, tea.Tick(4*time.Second, func(time.Time) tea.Msg { return fadeMsg{id} })
		}
		return m, nil
	case fadeMsg:
		if msg.id == m.statusID {
			m.status = statusMsg{}
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}
	var cmds []tea.Cmd
	if m.modal != nil {
		cmds = append(cmds, m.modal.update(msg))
	}
	for i, p := range m.panels {
		cmds = append(cmds, p.update(msg, i == m.active))
	}
	return m, tea.Batch(cmds...)
}

func (m model) onKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if k == "ctrl+c" {
		return m, tea.Quit
	}
	if m.modal != nil {
		closed, cmd := m.modal.key(msg)
		if closed {
			m.modal = nil
		}
		return m, cmd
	}
	if k == "ctrl+z" {
		return m, tea.Suspend
	}
	p := m.panels[m.active]
	if p.capturing() {
		_, cmd := p.key(msg)
		return m, cmd
	}
	switch k {
	case "q":
		return m, tea.Quit
	case "?":
		m.modal = m.help()
		return m, nil
	case "m":
		m.modal = m.messages()
		return m, nil
	case "tab", "]":
		return m.focus((m.active + 1) % len(m.panels))
	case "shift+tab", "[":
		return m.focus((m.active + len(m.panels) - 1) % len(m.panels))
	}
	if n, err := strconv.Atoi(k); err == nil && n >= 1 && n <= min(9, len(m.panels)) {
		return m.focus(n - 1)
	}
	if m.onNav {
		return m.navKey(k)
	}
	if k == "left" || k == "h" {
		m.onNav = true
		return m, nil
	}
	if handled, cmd := p.key(msg); handled {
		return m, cmd
	}
	if k == "r" {
		return m, tea.Batch(loadRepo, p.load())
	}
	return m, nil
}

// navKey moves between the tools while they have the focus, the panel
// following along: up and down in the sidebar, left and right in the tabs.
// The other way, or enter, gives the focus back to the panel.
func (m model) navKey(k string) (tea.Model, tea.Cmd) {
	prev, next, back := []string{"up", "k"}, []string{"down", "j"}, []string{"right", "l", "enter"}
	if !m.sidebarMode() {
		prev, next, back = []string{"left", "h"}, []string{"right", "l"}, []string{"down", "j", "enter"}
	}
	switch {
	case slices.Contains(prev, k):
		return m.focus(max(0, m.active-1))
	case slices.Contains(next, k):
		return m.focus(min(len(m.panels)-1, m.active+1))
	case slices.Contains(back, k):
		m.onNav = false
	case k == "home" || k == "g":
		return m.focus(0)
	case k == "end" || k == "G":
		return m.focus(len(m.panels) - 1)
	case k == "r":
		return m, tea.Batch(loadRepo, m.panels[m.active].load())
	}
	return m, nil
}

// sidebarMode says the tools are a sidebar, not a row of tabs.
func (m model) sidebarMode() bool { return m.w >= 120 }

// focus shows panel i, with its data read again.
func (m model) focus(i int) (tea.Model, tea.Cmd) {
	if i == m.active {
		return m, nil
	}
	m.active = i
	return m, m.panels[i].load()
}

func (m model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "dehv"
	return v
}

func (m model) render() string {
	if m.w == 0 {
		return ""
	}
	if m.w < minW || m.h < minH {
		return ui.Bold.Render("terminal too small") + "\n" +
			ui.Dim.Render("need "+strconv.Itoa(minW)+"×"+strconv.Itoa(minH)+" · "+strconv.Itoa(m.w)+"×"+strconv.Itoa(m.h)+" now · q quit")
	}
	rule := ruleStyle.Render(strings.Repeat("─", m.w))
	var rows []string
	rows = append(rows, m.header(), rule)

	bodyH := m.h - 5
	p := m.panels[m.active]
	var body string
	if m.sidebarMode() {
		side := m.sidebar(bodyH)
		mainW := m.w - lipgloss.Width(side) - 2
		body = lipgloss.JoinHorizontal(lipgloss.Top, side, ruleStyle.Render(strings.Repeat("│\n", bodyH-1)+"│"), indent(m.pane(p, mainW-1, bodyH), " "))
	} else {
		rows = append(rows, m.nav())
		bodyH--
		body = m.pane(p, m.w-1, bodyH)
		body = indent(body, " ")
	}
	rows = append(rows, fit(body, bodyH), rule, m.statusLine(), m.footer())
	frame := strings.Join(rows, "\n")

	if m.modal != nil {
		box := m.modal.view()
		bw, bh := lipgloss.Width(box), lipgloss.Height(box)
		x, y := max(0, (m.w-bw)/2), max(0, (m.h-bh)/2)
		frame = lipgloss.NewCompositor(
			lipgloss.NewLayer(frame).ID("base"),
			lipgloss.NewLayer(box).ID("modal").X(x).Y(y).Z(1),
		).Render()
	}
	return frame
}

func (m model) header() string {
	left := " " + ui.Accent.Bold(true).Render("dehv") + "  " + ui.Dim.Render(m.repo.Dir)
	if m.repo.Branch != "" {
		left += ui.Dim.Render(" · ") + ui.Branch.Render(m.repo.Branch)
	}
	if m.repo.Profile != "" {
		left += ui.Dim.Render(" · as ") + ui.Name.Render(m.repo.Profile)
	}
	right := ui.Dim.Render(core.DisplayVersion()) + " "
	gap := m.w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return ui.Cut(left, m.w)
	}
	return left + strings.Repeat(" ", gap) + right
}

// num is the key that jumps to tool i: 1 to 9, none past them.
func num(i int) string {
	if i < 9 {
		return strconv.Itoa(i + 1)
	}
	return " "
}

// lit draws the active tool's label: with the focus, a bar and the accent;
// without it, only a quiet highlight, so it shows where the focus is.
func (m model) lit(label string) string {
	if m.onNav {
		return barStyle.Render("▌") + navActive.Render(label)
	}
	return " " + navIdle.Render(label)
}

// sidebar is the tools, numbered, with their badges, the active one lit.
func (m model) sidebar(h int) string {
	const w = 18
	var lines []string
	for i, p := range m.panels {
		label := num(i) + " " + p.name()
		badge := p.badge()
		gap := w - 1 - lipgloss.Width(label) - lipgloss.Width(badge) - 1
		if i == m.active {
			// The lit row keeps the badge's glyph, not its colour.
			lines = append(lines, m.lit(label+strings.Repeat(" ", max(1, gap))+ansi.Strip(badge)+" "))
			continue
		}
		lines = append(lines, " "+ui.Dim.Render(num(i))+" "+p.name()+strings.Repeat(" ", max(1, gap))+badge+" ")
	}
	return fit(strings.Join(lines, "\n"), h)
}

// nav is the sidebar folded into a row of tabs, or into ‹ name n/7 › when
// even that does not fit.
func (m model) nav() string {
	var tabs []string
	for i, p := range m.panels {
		// Each tab starts with a column for the focus bar.
		switch {
		case i == m.active:
			tabs = append(tabs, m.lit(strings.TrimSpace(num(i)+" "+p.name())))
		case i < 9:
			tabs = append(tabs, " "+ui.Dim.Render(num(i))+" "+p.name())
		default:
			tabs = append(tabs, " "+p.name())
		}
	}
	row := strings.Join(tabs, " ")
	if lipgloss.Width(row) <= m.w {
		return row
	}
	p := m.panels[m.active]
	return ui.Dim.Render("‹") + m.lit(p.name()+" ") + ui.Dim.Render(" "+strconv.Itoa(m.active+1)+"/"+strconv.Itoa(len(m.panels))+" ›")
}

// pane is the panel's heading and its view.
func (m model) pane(p panel, w, h int) string {
	head := ui.Bold.Render(p.name())
	if s := p.heading(); s != "" {
		head += ui.Dim.Render(" · " + s)
	}
	return ui.Cut(head, w) + "\n" + p.view(w, h-1, !m.onNav)
}

func (m model) statusLine() string {
	if m.working != "" {
		return " " + m.spin.View() + " " + ui.Dim.Render(m.working+"…")
	}
	s := m.status
	switch s.kind {
	case "ok":
		return ui.Cut(" "+ui.OKMark.Render("✓")+" "+s.text, m.w)
	case "err":
		return ui.Cut(" "+ui.Err.Render("✗")+" "+s.text+ui.Dim.Render("  m messages"), m.w)
	case "warn":
		return ui.Cut(" "+ui.Warn.Render("!")+" "+s.text, m.w)
	case "info":
		return ui.Cut(" "+ui.Dim.Render("· "+s.text), m.w)
	}
	return ""
}

// footer is the keys that matter here: the panel's, most used first, then
// the global ones; from the right they give way when narrow, but ? stays.
func (m model) footer() string {
	var hs []hint
	switch {
	case m.onNav && m.sidebarMode():
		hs = []hint{{"↑↓", "choose"}, {"→", "open"}}
	case m.onNav:
		hs = []hint{{"←→", "choose"}, {"↓", "open"}}
	default:
		hs = append(hs, m.panels[m.active].hints()...)
		hs = append(hs, hint{"←", "tools"})
	}
	hs = append(hs, hint{m.jump(), "jump"}, hint{"q", "quit"})
	sep := ui.Dim.Render(" ⋅ ")
	help := keyHint("?", "help")
	for len(hs) > 0 {
		var parts []string
		for _, h := range hs {
			parts = append(parts, keyHint(h.key, h.what))
		}
		line := " " + strings.Join(append(parts, help), sep)
		if lipgloss.Width(line) <= m.w {
			return line
		}
		hs = hs[:len(hs)-1]
	}
	return " " + help
}

// jump is the number keys there are: 1-7, up to 1-9.
func (m model) jump() string { return "1-" + strconv.Itoa(min(9, len(m.panels))) }

func (m model) help() modal {
	choose, open := "↑↓ / j k", "→ / l / enter"
	if !m.sidebarMode() {
		choose, open = "←→ / h l", "↓ / j / enter"
	}
	global := []hint{
		{"← / h", "focus the tools"}, {choose, "with the tools focused: choose one"}, {open, "with the tools focused: back to it"},
		{m.jump(), "jump to a tool"}, {"tab / ]", "next tool"}, {"shift+tab / [", "previous tool"},
		{"↑↓ / j k", "move"}, {"/", "filter"}, {"esc", "close, clear the filter"},
		{"r", "read it again"}, {"m", "messages"}, {"ctrl+z", "suspend"}, {"q", "quit"},
	}
	var lines []string
	lines = append(lines, ui.Dim.Render(m.panels[m.active].name()))
	for _, h := range m.panels[m.active].hints() {
		lines = append(lines, "  "+ui.Pad(ui.Accent.Render(h.key), 14)+h.what)
	}
	lines = append(lines, "", ui.Dim.Render("everywhere"))
	for _, h := range global {
		lines = append(lines, "  "+ui.Pad(ui.Accent.Render(h.key), 14)+h.what)
	}
	return &overlay{title: "Keys", lines: lines}
}

func (m model) messages() modal {
	if len(m.log) == 0 {
		return &overlay{title: "Messages", lines: []string{ui.Dim.Render("nothing yet")}}
	}
	var lines []string
	from := max(0, len(m.log)-15)
	for _, e := range m.log[from:] {
		mark := map[string]string{
			"ok": ui.OKMark.Render("✓"), "err": ui.Err.Render("✗"),
			"warn": ui.Warn.Render("!"), "info": ui.Dim.Render("·"),
		}[e.kind]
		lines = append(lines, ui.Dim.Render(e.at.Format("15:04:05"))+" "+mark+" "+ui.Cut(e.text, 70))
	}
	return &overlay{title: "Messages", lines: lines}
}

// fit pads or cuts s to exactly h lines.
func fit(s string, h int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func indent(s, pre string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = pre + l
	}
	return strings.Join(lines, "\n")
}
