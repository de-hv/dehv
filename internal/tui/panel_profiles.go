package tui

import (
	"errors"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

type profilesLoaded struct {
	ps     []core.Profile
	cur    string
	inRepo bool
}

// profiles is my git identities, the one this repo commits as marked.
type profiles struct {
	t      table
	ps     []core.Profile
	cur    string
	inRepo bool
	loaded bool
}

func newProfiles() *profiles {
	return &profiles{t: newTable(column{title: " "}, column{title: "PROFILE"}, column{title: "NAME", drop: 1}, column{title: "EMAIL"})}
}

func (p *profiles) name() string  { return "Profiles" }
func (p *profiles) badge() string { return "" }

func (p *profiles) heading() string {
	switch {
	case !p.loaded:
		return "loading…"
	case !p.inRepo:
		return ui.Plural(len(p.ps), "profile", "profiles") + " · not in a git repo"
	case p.cur == "":
		return ui.Plural(len(p.ps), "profile", "profiles") + " · this repo is none of them"
	}
	return ui.Plural(len(p.ps), "profile", "profiles") + " · this repo commits as " + p.cur
}

func (p *profiles) load() tea.Cmd {
	return func() tea.Msg {
		cur, _ := core.CurrentProfile("")
		return profilesLoaded{ps: core.Profiles(), cur: cur.Name, inRepo: core.InRepo("")}
	}
}

func (p *profiles) update(msg tea.Msg, _ bool) tea.Cmd {
	if msg, ok := msg.(profilesLoaded); ok {
		p.loaded, p.ps, p.cur, p.inRepo = true, msg.ps, msg.cur, msg.inRepo
		var rows []row
		for _, x := range msg.ps {
			mark := " "
			if x.Name == msg.cur {
				mark = ui.OKMark.Render("●")
			}
			rows = append(rows, row{key: x.Name, cells: []string{mark, ui.Bold.Render(x.Name), x.User, ui.Dim.Render(x.Email)}, search: x.Name + " " + x.User + " " + x.Email})
		}
		p.t.setRows(rows)
	}
	return nil
}

func (p *profiles) selected() (core.Profile, bool) {
	r, ok := p.t.selected()
	if !ok {
		return core.Profile{}, false
	}
	return core.GetProfile(r.key)
}

// form is the dialog that creates or edits a profile.
func (p *profiles) form(title string, x core.Profile, withName bool) tea.Cmd {
	fields := []field{}
	if withName {
		fields = append(fields, field{label: "Profile", hint: "acme"})
	}
	fields = append(fields, field{label: "Name", value: x.User, hint: "David Higuita"}, field{label: "Email", value: x.Email, hint: "david@acme.com"})
	return open(newPrompt(title, fields, func(v []string) (tea.Cmd, error) {
		if withName {
			if !core.ValidProfile(v[0]) {
				return nil, errors.New("a profile is lower-case letters, digits, - and _")
			}
			x.Name, v = v[0], v[1:]
		}
		x.User, x.Email = v[0], v[1]
		if err := core.SaveProfile(x); err != nil {
			return nil, err
		}
		return tea.Batch(say("ok", "saved "+x.Name+" · "+x.Ident()), p.load()), nil
	}))
}

func (p *profiles) key(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if handled, cmd := p.t.key(msg); handled {
		return true, cmd
	}
	k := msg.String()
	if k == "n" {
		u, e := core.GlobalIdent()
		return true, p.form("New profile", core.Profile{User: u, Email: e}, true)
	}
	x, ok := p.selected()
	if !ok {
		return k == "enter" || k == "e" || k == "d" || k == "a", nil
	}
	switch k {
	case "enter":
		if !p.inRepo {
			return true, say("warn", "not in a git repo: open dehv from one to commit as "+x.Name)
		}
		if x.Name == p.cur {
			return true, say("info", "already as "+x.Name+", nothing to do")
		}
		if err := core.UseProfile("", x); err != nil {
			return true, failed(err)
		}
		return true, tea.Batch(say("ok", "this repo commits as "+x.Name+" · "+x.Ident()), p.load(), loadRepo)
	case "e":
		return true, p.form("Edit "+x.Name, x, false)
	case "d":
		return true, open(&confirm{
			title: "Remove the profile " + x.Name + "?",
			lines: []string{x.Ident(), "repos that commit as it keep their identity"},
			yes:   "remove",
			onYes: tea.Sequence(func() tea.Msg {
				if err := core.RemoveProfile(x.Name); err != nil {
					return statusMsg{"err", err.Error()}
				}
				return statusMsg{"ok", "removed " + x.Name}
			}, p.load()),
		})
	case "a":
		if !p.inRepo {
			return true, say("warn", "not in a git repo")
		}
		return true, open(newPrompt("Make "+x.Name+" the author of the last commits", []field{
			{label: "How many: they are rewritten, so only before pushing", value: "1"},
		}, func(v []string) (tea.Cmd, error) {
			n, err := strconv.Atoi(v[0])
			if err != nil || n < 1 {
				return nil, errors.New("a number of commits, 1 or more")
			}
			return func() tea.Msg {
				done, err := core.AmendAuthors("", x, n)
				switch {
				case err != nil:
					var ce *core.Error
					if errors.As(err, &ce) {
						return statusMsg{"err", ce.Msg}
					}
					return statusMsg{"err", err.Error()}
				case done == nil:
					return statusMsg{"info", "already as " + x.Name + ", nothing to do"}
				}
				return statusMsg{"ok", ui.Plural(len(done), "commit", "commits") + " now as " + x.Name}
			}, nil
		}))
	}
	return false, nil
}

func (p *profiles) capturing() bool { return p.t.filtering }

func (p *profiles) hints() []hint {
	return []hint{{"enter", "commit as"}, {"n", "new"}, {"e", "edit"}, {"a", "amend"}, {"d", "remove"}}
}

func (p *profiles) view(w, h int, focused bool) string {
	switch {
	case !p.loaded:
		return ui.Dim.Render("reading profiles…")
	case len(p.ps) == 0:
		return "No profiles yet.\n" + ui.Dim.Render("n creates one: a name and an email to commit as, per repo.")
	}
	return p.t.render(w, h, focused)
}
