package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/de-hv/dehv/internal/ui"
)

// modal is a dialog over the panes: it takes every key until it closes.
type modal interface {
	key(msg tea.KeyPressMsg) (closed bool, cmd tea.Cmd)
	update(msg tea.Msg) tea.Cmd
	view() string
}

// openMsg asks the app to show a dialog.
type openMsg struct{ m modal }

func open(m modal) tea.Cmd { return func() tea.Msg { return openMsg{m} } }

func keyHint(k, what string) string { return ui.Accent.Render(k) + " " + ui.Dim.Render(what) }

// confirm asks before something that removes or rewrites. Only y goes ahead:
// enter is no, so a stray one never deletes anything.
type confirm struct {
	title string
	lines []string
	yes   string // the verb: kill, delete…
	onYes tea.Cmd
	reply chan bool // instead of onYes, for work waiting in a goroutine
}

func (c *confirm) key(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if c.reply != nil {
			c.reply <- true
		}
		return true, c.onYes
	case "n", "N", "esc", "enter", "q":
		if c.reply != nil {
			c.reply <- false
		}
		return true, nil
	}
	return false, nil
}

func (c *confirm) update(tea.Msg) tea.Cmd { return nil }

func (c *confirm) view() string {
	body := []string{ui.Bold.Render(c.title)}
	for i, l := range c.lines {
		if i == 10 {
			body = append(body, ui.Dim.Render(fmt.Sprintf("… and %d more", len(c.lines)-10)))
			break
		}
		body = append(body, ui.Dim.Render(l))
	}
	yes := c.yes
	if yes == "" {
		yes = "yes"
	}
	body = append(body, "", ui.Err.Render("y")+" "+ui.Dim.Render(yes)+"    "+keyHint("n", "cancel"))
	return modalStyle.Render(strings.Join(body, "\n"))
}

// field is one line of a prompt.
type field struct {
	label string
	value string
	hint  string
}

// prompt asks for a few lines of text, prefilled.
type prompt struct {
	title    string
	labels   []string
	hints    []string
	inputs   []textinput.Model
	focus    int
	err      string
	onSubmit func(values []string) (tea.Cmd, error)
}

func newPrompt(title string, fields []field, onSubmit func([]string) (tea.Cmd, error)) *prompt {
	p := &prompt{title: title, onSubmit: onSubmit}
	for _, f := range fields {
		in := newInput("› ", f.hint)
		in.SetValue(f.value)
		in.SetWidth(44)
		p.labels = append(p.labels, f.label)
		p.hints = append(p.hints, f.hint)
		p.inputs = append(p.inputs, in)
	}
	p.inputs[0].Focus()
	return p
}

func (p *prompt) move(d int) tea.Cmd {
	p.inputs[p.focus].Blur()
	p.focus = (p.focus + d + len(p.inputs)) % len(p.inputs)
	return p.inputs[p.focus].Focus()
}

func (p *prompt) key(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return true, nil
	case "tab", "down":
		return false, p.move(1)
	case "shift+tab", "up":
		return false, p.move(-1)
	case "enter":
		if p.focus < len(p.inputs)-1 {
			return false, p.move(1)
		}
		var vals []string
		for _, in := range p.inputs {
			vals = append(vals, strings.TrimSpace(in.Value()))
		}
		cmd, err := p.onSubmit(vals)
		if err != nil {
			p.err = err.Error()
			return false, nil
		}
		return true, cmd
	}
	var cmd tea.Cmd
	p.inputs[p.focus], cmd = p.inputs[p.focus].Update(msg)
	p.err = ""
	return false, cmd
}

func (p *prompt) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.inputs[p.focus], cmd = p.inputs[p.focus].Update(msg)
	return cmd
}

func (p *prompt) view() string {
	body := []string{ui.Bold.Render(p.title), ""}
	for i, in := range p.inputs {
		label := ui.Dim.Render(p.labels[i])
		if i == p.focus {
			label = ui.Accent.Render(p.labels[i])
		}
		body = append(body, label, in.View())
	}
	if p.err != "" {
		body = append(body, "", ui.Err.Render("✗ "+p.err))
	}
	submit := "save"
	body = append(body, "", keyHint("enter", "next / "+submit)+"    "+keyHint("tab", "field")+"    "+keyHint("esc", "cancel"))
	return modalStyle.Render(strings.Join(body, "\n"))
}

// checklist ticks some of a few options, those ticked to begin with shown so.
type checklist struct {
	title   string
	options []string
	on      []bool
	cur     int
	onDone  func(on []bool) tea.Cmd
}

func (c *checklist) key(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return true, nil
	case "up", "k":
		c.cur = max(0, c.cur-1)
	case "down", "j":
		c.cur = min(len(c.options)-1, c.cur+1)
	case "space", " ", "x":
		c.on[c.cur] = !c.on[c.cur]
	case "enter":
		return true, c.onDone(c.on)
	}
	return false, nil
}

func (c *checklist) update(tea.Msg) tea.Cmd { return nil }

func (c *checklist) view() string {
	body := []string{ui.Bold.Render(c.title), ""}
	for i, o := range c.options {
		box := ui.Dim.Render("[ ]")
		if c.on[i] {
			box = ui.OKMark.Render("[x]")
		}
		line := box + " " + o
		if i == c.cur {
			line = barStyle.Render("▌") + line
		} else {
			line = " " + line
		}
		body = append(body, line)
	}
	body = append(body, "", keyHint("space", "tick")+"    "+keyHint("enter", "apply")+"    "+keyHint("esc", "cancel"))
	return modalStyle.Render(strings.Join(body, "\n"))
}

// overlay is a read-only page: the help or the message log.
type overlay struct {
	title string
	lines []string
}

func (o *overlay) key(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?", "m", "enter":
		return true, nil
	}
	return false, nil
}

func (o *overlay) update(tea.Msg) tea.Cmd { return nil }

func (o *overlay) view() string {
	body := append([]string{ui.Bold.Render(o.title), ""}, o.lines...)
	body = append(body, "", keyHint("esc", "close"))
	return modalStyle.Render(strings.Join(body, "\n"))
}
