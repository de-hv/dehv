package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

type branchesLoaded struct {
	bs     core.Branches
	inRepo bool
	err    error
}

type fetchedMsg struct{}

// branches is the current repo's branches but the default one and the one
// checked out, merged and gone ones ticked, to delete the ticked ones.
type branches struct {
	t       table
	bs      core.Branches
	inRepo  bool
	err     error
	loaded  bool
	fetched bool
	ticked  map[string]bool
}

func newBranches() *branches {
	return &branches{
		t:      newTable(column{title: "   "}, column{title: "BRANCH"}, column{title: "WHY"}, column{title: "LAST COMMIT", drop: 1}),
		ticked: map[string]bool{},
	}
}

func (b *branches) name() string { return "Branches" }

func (b *branches) badge() string {
	if n := b.count(); n > 0 {
		return ui.Warn.Render(fmt.Sprint(n))
	}
	return ""
}

func (b *branches) count() int {
	n := 0
	for _, x := range b.bs.List {
		if x.CanGo() {
			n++
		}
	}
	return n
}

func (b *branches) heading() string {
	switch {
	case !b.loaded:
		return "loading…"
	case !b.inRepo:
		return "not in a git repo"
	}
	base := b.bs.Base
	if base == "" {
		base = "none"
	}
	return fmt.Sprintf("%d can go · base %s", b.count(), base)
}

func (b *branches) load() tea.Cmd {
	return func() tea.Msg {
		if !core.InRepo("") {
			return branchesLoaded{}
		}
		bs, err := core.ListBranches("")
		return branchesLoaded{bs: bs, inRepo: true, err: err}
	}
}

func (b *branches) fetch() tea.Cmd {
	return tea.Batch(busy("fetching"), func() tea.Msg {
		core.Fetch(hubCtx, "")
		return fetchedMsg{}
	})
}

func (b *branches) update(msg tea.Msg, _ bool) tea.Cmd {
	switch msg := msg.(type) {
	case branchesLoaded:
		b.loaded, b.inRepo, b.err, b.bs = true, msg.inRepo, msg.err, msg.bs
		seen := map[string]bool{}
		for _, x := range msg.bs.List {
			seen[x.Name] = true
			if _, ok := b.ticked[x.Name]; !ok {
				b.ticked[x.Name] = x.CanGo()
			}
		}
		for n := range b.ticked {
			if !seen[n] {
				delete(b.ticked, n)
			}
		}
		b.rows()
		// Fetched once, with --prune, so "gone" is current.
		if b.inRepo && !b.fetched {
			b.fetched = true
			return b.fetch()
		}
	case fetchedMsg:
		return tea.Batch(busy(""), b.load())
	}
	return nil
}

func (b *branches) rows() {
	var rows []row
	for _, x := range b.bs.List {
		box := ui.Dim.Render("[ ]")
		if b.ticked[x.Name] {
			box = ui.OKMark.Render("[x]")
		}
		name := x.Name
		why := ui.Dim.Render(x.Why)
		if x.CanGo() {
			name = ui.Generic.Render(x.Name)
			why = x.Why
		}
		rows = append(rows, row{key: x.Name, cells: []string{box, name, why, ui.Dim.Render(x.Age)}, search: x.Name + " " + x.Why})
	}
	b.t.setRows(rows)
}

func (b *branches) key(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if handled, cmd := b.t.key(msg); handled {
		return true, cmd
	}
	switch msg.String() {
	case "space", " ":
		if r, ok := b.t.selected(); ok {
			b.ticked[r.key] = !b.ticked[r.key]
			b.rows()
		}
		return true, nil
	case "a":
		all := true
		for _, x := range b.bs.List {
			all = all && b.ticked[x.Name]
		}
		for _, x := range b.bs.List {
			b.ticked[x.Name] = !all
		}
		b.rows()
		return true, nil
	case "f":
		if b.inRepo {
			return true, b.fetch()
		}
		return true, nil
	case "d":
		var names []string
		for _, x := range b.bs.List {
			if b.ticked[x.Name] {
				names = append(names, x.Name)
			}
		}
		if len(names) == 0 {
			return true, say("info", "none ticked, nothing to delete")
		}
		del := func() tea.Msg {
			var failed []string
			for _, n := range names {
				if err := core.DeleteBranch("", n); err != nil {
					failed = append(failed, n)
				}
			}
			if len(failed) > 0 {
				return statusMsg{"err", "could not delete " + strings.Join(failed, ", ")}
			}
			return statusMsg{"ok", "deleted " + core.Plural(len(names), "branch", "branches") + " · " + strings.Join(names, ", ")}
		}
		return true, open(&confirm{
			title: "Delete " + core.Plural(len(names), "branch", "branches") + "?",
			lines: names,
			yes:   "delete",
			onYes: tea.Sequence(del, b.load()),
		})
	}
	return false, nil
}

func (b *branches) capturing() bool { return b.t.filtering }

func (b *branches) hints() []hint {
	return []hint{{"space", "tick"}, {"d", "delete ticked"}, {"a", "tick all"}, {"f", "fetch"}, {"/", "filter"}}
}

func (b *branches) view(w, h int, focused bool) string {
	switch {
	case !b.loaded:
		return ui.Dim.Render("reading branches…")
	case !b.inRepo:
		return ui.Dim.Render("Not in a git repo.") + "\n" + ui.Dim.Render("Open dehv from a repo to clean up its branches.")
	case b.err != nil:
		return ui.Err.Render("✗") + " " + b.err.Error()
	case len(b.bs.List) == 0:
		only := b.bs.Base
		if only == "" {
			only = b.bs.Current
		}
		return ui.Dim.Render("Only " + only + " here, nothing to delete.")
	}
	return b.t.render(w, h, focused)
}
