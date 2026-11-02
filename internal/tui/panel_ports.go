package tui

import (
	"fmt"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

type portsLoaded struct {
	ls  []core.Listener
	err error
}

type portsTick struct{}

// ports is what listens on TCP, read again every 2 seconds while it shows.
type ports struct {
	t       table
	ls      []core.Listener
	err     error
	loaded  bool
	ticking bool
}

func newPorts() *ports {
	return &ports{t: newTable(column{title: "PORT"}, column{title: "PROGRAM"}, column{title: "PID"}, column{title: "USER", drop: 1})}
}

func (p *ports) name() string { return "Ports" }

func (p *ports) badge() string {
	if !p.loaded || len(p.ls) == 0 {
		return ""
	}
	return ui.Dim.Render(strconv.Itoa(len(p.ls)))
}

func (p *ports) heading() string {
	switch {
	case p.err != nil:
		return "could not read them"
	case !p.loaded:
		return "loading…"
	}
	return fmt.Sprintf("%d listening", len(p.ls))
}

func (p *ports) load() tea.Cmd {
	return func() tea.Msg {
		ls, err := core.Listeners()
		return portsLoaded{ls, err}
	}
}

func (p *ports) update(msg tea.Msg, active bool) tea.Cmd {
	switch msg := msg.(type) {
	case portsLoaded:
		p.loaded, p.err, p.ls = true, msg.err, msg.ls
		var rows []row
		for _, l := range msg.ls {
			port, pid := strconv.Itoa(l.Port), strconv.Itoa(l.PID)
			rows = append(rows, row{
				key:    port + " " + pid,
				cells:  []string{ui.Name.Render(port), ui.Bold.Render(l.Program), ui.Dim.Render(pid), ui.Dim.Render(l.User)},
				search: port + " " + l.Program + " " + pid,
			})
		}
		p.t.setRows(rows)
		if !p.ticking {
			p.ticking = true
			return tick()
		}
	case portsTick:
		if active {
			return tea.Batch(p.load(), tick())
		}
		return tick()
	}
	return nil
}

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return portsTick{} })
}

func (p *ports) key(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if handled, cmd := p.t.key(msg); handled {
		return true, cmd
	}
	if msg.String() != "x" {
		return false, nil
	}
	r, ok := p.t.selected()
	if !ok {
		return true, nil
	}
	var l core.Listener
	for _, x := range p.ls {
		if strconv.Itoa(x.Port)+" "+strconv.Itoa(x.PID) == r.key {
			l = x
		}
	}
	kill := func() tea.Msg {
		if err := core.Kill(l.PID); err != nil {
			return statusMsg{"err", fmt.Sprintf("could not kill %s (%d) · %v", l.Program, l.PID, err)}
		}
		return statusMsg{"ok", fmt.Sprintf("killed %s (%d) on port %d", l.Program, l.PID, l.Port)}
	}
	reload := tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return p.load()() })
	return true, open(&confirm{
		title: fmt.Sprintf("Kill %s (%d) on port %d?", l.Program, l.PID, l.Port),
		lines: []string{"it gets SIGKILL: no chance to clean up"},
		yes:   "kill",
		onYes: tea.Batch(kill, reload),
	})
}

func (p *ports) capturing() bool { return p.t.filtering }

func (p *ports) hints() []hint {
	return []hint{{"x", "kill"}, {"/", "filter"}, {"r", "refresh"}}
}

func (p *ports) view(w, h int, focused bool) string {
	switch {
	case p.err != nil:
		return ui.Err.Render("✗") + " " + p.err.Error() + "\n" + ui.Dim.Render("r try again")
	case !p.loaded:
		return ui.Dim.Render("reading lsof…")
	case len(p.ls) == 0:
		return ui.Dim.Render("Nothing is listening on TCP.") + "\n" + ui.Dim.Render("Start a server and it shows up here within 2 seconds.")
	}
	return p.t.render(w, h, focused)
}
