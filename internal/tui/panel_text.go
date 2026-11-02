package tui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/de-hv/dehv/internal/cli"
	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

// scroller is a page of lines moved with j/k, for the panels that are text.
type scroller struct {
	lines []string
	off   int
	h     int
}

func (s *scroller) key(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "up", "k":
		s.off--
	case "down", "j":
		s.off++
	case "pgup", "ctrl+u":
		s.off -= 10
	case "pgdown", "ctrl+d":
		s.off += 10
	case "home", "g":
		s.off = 0
	case "end", "G":
		s.off = len(s.lines)
	default:
		return false
	}
	s.clamp()
	return true
}

func (s *scroller) clamp() {
	s.off = max(0, min(s.off, len(s.lines)-s.h))
}

func (s *scroller) render(w, h int) string {
	s.h = h
	s.clamp()
	end := min(len(s.lines), s.off+h)
	out := make([]string, 0, h)
	for _, l := range s.lines[s.off:end] {
		out = append(out, ui.Cut(l, w))
	}
	if s.off+h < len(s.lines) && len(out) > 0 {
		out[len(out)-1] = ui.Dim.Render("↓ more · j/k scroll")
	}
	return strings.Join(out, "\n")
}

type machineLoaded struct {
	m    core.Machine
	addr core.Address
	err  error
}

type publicLoaded struct {
	ip  string
	err error
}

// machine is the `dehv hi` card and this Mac's addresses.
type machine struct {
	sc     scroller
	m      core.Machine
	addr   core.Address
	err    error
	public string
	loaded bool
}

func newMachine() *machine { return &machine{} }

func (m *machine) name() string  { return "Machine" }
func (m *machine) badge() string { return "" }
func (m *machine) heading() string {
	if !m.loaded {
		return "loading…"
	}
	return m.m.Host
}

func (m *machine) load() tea.Cmd {
	return func() tea.Msg {
		addr, err := core.LocalIP()
		return machineLoaded{core.ThisMac(), addr, err}
	}
}

func (m *machine) update(msg tea.Msg, _ bool) tea.Cmd {
	switch msg := msg.(type) {
	case machineLoaded:
		m.loaded, m.m, m.addr, m.err = true, msg.m, msg.addr, msg.err
	case publicLoaded:
		if msg.err != nil {
			return failed(msg.err)
		}
		m.public = msg.ip
		return busy("")
	default:
		return nil
	}
	m.sc.lines = m.lines()
	return nil
}

func (m *machine) lines() []string {
	ls := cli.HiLines(m.m, time.Now(), true)
	ls = append(ls, "", ui.Dim.Render("addresses"))
	if m.err != nil {
		ls = append(ls, ui.Warn.Render("!")+" "+m.err.Error())
	} else {
		label := m.addr.Iface
		if m.addr.Port != "" {
			label += " · " + m.addr.Port
		}
		ls = append(ls, ui.Pad(ui.Dim.Render("local"), 8)+ui.Bold.Render(m.addr.IP)+"  "+ui.Dim.Render(label))
	}
	if m.public != "" {
		ls = append(ls, ui.Pad(ui.Dim.Render("public"), 8)+ui.Bold.Render(m.public)+"  "+ui.Dim.Render("api.ipify.org"))
	} else {
		ls = append(ls, ui.Pad(ui.Dim.Render("public"), 8)+ui.Dim.Render("p asks api.ipify.org"))
	}
	return ls
}

func (m *machine) key(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if m.sc.key(msg) {
		return true, nil
	}
	switch msg.String() {
	case "p":
		return true, tea.Batch(busy("asking api.ipify.org"), func() tea.Msg {
			ip, err := core.PublicIP(hubCtx)
			return publicLoaded{ip, err}
		})
	case "c", "C":
		ip := m.addr.IP
		if msg.String() == "C" {
			ip = m.public
		}
		if ip == "" {
			return true, say("warn", "no address to copy")
		}
		if err := core.Copy(ip); err != nil {
			return true, failed(err)
		}
		return true, say("ok", "copied "+ip)
	}
	return false, nil
}

func (m *machine) capturing() bool { return false }
func (m *machine) hints() []hint {
	return []hint{{"c", "copy IP"}, {"p", "public IP"}, {"C", "copy public"}, {"r", "refresh"}}
}

func (m *machine) view(w, h int, focused bool) string {
	if !m.loaded {
		return ui.Dim.Render("asking this Mac…")
	}
	return m.sc.render(w, h)
}

type doctorLoaded struct {
	checks []core.Check
	in     core.Install
}

// doctor is `dehv doctor`, `dehv where` and the release steps of `dehv tap`.
type doctor struct {
	sc     scroller
	checks []core.Check
	in     core.Install
	kind   int
	loaded bool
}

func newDoctor() *doctor { return &doctor{} }

func (d *doctor) name() string { return "Doctor" }

func (d *doctor) badge() string {
	if !d.loaded {
		return ""
	}
	if core.Healthy(d.checks) {
		return ui.OKMark.Render("✓")
	}
	return ui.Err.Render("✗")
}

func (d *doctor) heading() string {
	if !d.loaded {
		return "checking…"
	}
	return core.Tally(d.checks)
}

func (d *doctor) load() tea.Cmd {
	return func() tea.Msg { return doctorLoaded{core.Doctor(), core.ThisInstall()} }
}

func (d *doctor) update(msg tea.Msg, _ bool) tea.Cmd {
	if msg, ok := msg.(doctorLoaded); ok {
		d.loaded, d.checks, d.in = true, msg.checks, msg.in
		d.refresh()
	}
	return nil
}

func (d *doctor) refresh() {
	ls := cli.DoctorLines(d.checks, true)
	ls = append(ls, "", ui.Dim.Render("install"))
	ls = append(ls, cli.WhereLines(d.in)...)
	kind := core.TapKinds[d.kind].Kind
	ls = append(ls, "", ui.Dim.Render("release")+"  "+ui.Dim.Render("t: ")+kind+ui.Dim.Render(" (goreleaser, cask, formula)"))
	ls = append(ls, cli.TapLines(kind, true)[2:]...)
	d.sc.lines = ls
}

func (d *doctor) key(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if d.sc.key(msg) {
		return true, nil
	}
	if msg.String() == "t" {
		d.kind = (d.kind + 1) % len(core.TapKinds)
		d.refresh()
		return true, nil
	}
	return false, nil
}

func (d *doctor) capturing() bool { return false }
func (d *doctor) hints() []hint {
	return []hint{{"t", "release steps"}, {"j/k", "scroll"}, {"r", "check again"}}
}

func (d *doctor) view(w, h int, focused bool) string {
	if !d.loaded {
		return ui.Dim.Render("checking…")
	}
	return d.sc.render(w, h)
}
