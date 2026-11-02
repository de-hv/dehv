package tui

import (
	"context"
	"errors"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

type wallsLoaded struct {
	ws     []core.Wall
	states []string
	use    string
}

type wallDone struct{ err error }

// walls is my wallpaper repos, the one in use marked, to sync them and add
// images to them.
type walls struct {
	ctx    context.Context
	t      table
	ws     []core.Wall
	use    string
	loaded bool
	busy   bool
}

func newWalls(ctx context.Context) *walls {
	return &walls{ctx: ctx, t: newTable(column{title: " "}, column{title: "REPO"}, column{title: "GITHUB"}, column{title: "FOLDER", drop: 2}, column{title: "IMAGES", drop: 1})}
}

func (w *walls) name() string { return "Walls" }

func (w *walls) badge() string {
	if w.busy {
		return ui.Accent.Render("↻")
	}
	return ""
}

func (w *walls) heading() string {
	switch {
	case !w.loaded:
		return "loading…"
	case w.use == "":
		return ui.Plural(len(w.ws), "repo", "repos")
	}
	return ui.Plural(len(w.ws), "repo", "repos") + " · " + w.use + " in use"
}

func (w *walls) load() tea.Cmd {
	return func() tea.Msg {
		ws := core.Walls()
		states := make([]string, len(ws))
		for i, x := range ws {
			states[i] = core.WallState(x)
		}
		cur, _, _ := core.WallInUse()
		return wallsLoaded{ws, states, cur.Name}
	}
}

func (w *walls) update(msg tea.Msg, _ bool) tea.Cmd {
	switch msg := msg.(type) {
	case wallsLoaded:
		w.loaded, w.ws, w.use = true, msg.ws, msg.use
		var rows []row
		for i, x := range msg.ws {
			mark := " "
			if x.Name == msg.use {
				mark = ui.OKMark.Render("●")
			}
			state := ui.Dim.Render(msg.states[i])
			if msg.states[i] == "gone" {
				state = ui.Err.Render("gone")
			}
			rows = append(rows, row{key: x.Name, cells: []string{mark, ui.Bold.Render(x.Name), x.Short(), ui.Dim.Render(x.Dir), state}, search: x.Name + " " + x.URL})
		}
		w.t.setRows(rows)
	case wallDone:
		w.busy = false
		var cmds []tea.Cmd
		if msg.err != nil {
			cmds = append(cmds, failed(msg.err))
		} else {
			cmds = append(cmds, busy(""))
		}
		return tea.Batch(append(cmds, w.load())...)
	}
	return nil
}

// run does slow work on a repo in the background, reporting as it goes.
func (w *walls) run(do func(ctx context.Context, r core.Reporter) error) tea.Cmd {
	if w.busy {
		return say("info", "a sync is already running")
	}
	w.busy = true
	return func() tea.Msg { return wallDone{do(w.ctx, reporter{})} }
}

func (w *walls) selected() (core.Wall, bool) {
	r, ok := w.t.selected()
	if !ok {
		return core.Wall{}, false
	}
	return core.GetWall(r.key)
}

func (w *walls) key(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if handled, cmd := w.t.key(msg); handled {
		return true, cmd
	}
	k := msg.String()
	switch k {
	case "n":
		return true, w.form("New wallpaper repo", core.Wall{}, true)
	case "S":
		ws := w.ws
		return true, w.run(func(ctx context.Context, r core.Reporter) error {
			var failedRepos []string
			for _, x := range ws {
				if err := core.SyncWall(ctx, x, r); err != nil {
					if errors.Is(err, context.Canceled) {
						return err
					}
					failedRepos = append(failedRepos, x.Name)
				}
			}
			if len(failedRepos) > 0 {
				return core.Fail("could not sync "+strings.Join(failedRepos, ", "), "s on each one says why")
			}
			return nil
		})
	}
	x, ok := w.selected()
	if !ok {
		return k == "s" || k == "a" || k == "o" || k == "u" || k == "e" || k == "d", nil
	}
	switch k {
	case "s":
		return true, w.run(func(ctx context.Context, r core.Reporter) error { return core.SyncWall(ctx, x, r) })
	case "a":
		return true, open(newPrompt("Add a wallpaper to "+x.Name, []field{
			{label: "Image url", hint: "https://…/aurora.jpg"},
			{label: "Save as (empty: the url's name)", hint: "aurora.jpg"},
		}, func(v []string) (tea.Cmd, error) {
			if !strings.HasPrefix(v[0], "http://") && !strings.HasPrefix(v[0], "https://") {
				return nil, errors.New("an http(s) url")
			}
			if strings.Contains(v[1], "/") || strings.HasPrefix(v[1], ".") {
				return nil, errors.New("a file name, without / nor a leading .")
			}
			return w.run(func(ctx context.Context, r core.Reporter) error { return core.AddWallpaper(ctx, x, v[0], v[1], r) }), nil
		}))
	case "o":
		if err := exec.Command("open", x.Path()).Run(); err != nil {
			return true, say("err", x.Name+" is not cloned yet · s clones it")
		}
		return true, nil
	case "u":
		if x.Name == w.use {
			return true, say("info", "already using "+x.Name)
		}
		if _, err := core.UseWall(x.Name); err != nil {
			return true, failed(err)
		}
		return true, tea.Batch(say("ok", x.Name+" in use · "+x.Short()), w.load())
	case "e":
		return true, w.form("Edit "+x.Name, x, false)
	case "d":
		return true, open(&confirm{
			title: "Forget " + x.Name + "?",
			lines: []string{x.Short(), x.Dir + " stays as it is"},
			yes:   "forget",
			onYes: tea.Sequence(func() tea.Msg {
				if _, err := core.RemoveWall(x.Name); err != nil {
					return statusMsg{"err", err.Error()}
				}
				return statusMsg{"ok", "forgot " + x.Name + " · " + x.Dir + " stays"}
			}, w.load()),
		})
	}
	return false, nil
}

// form is the dialog that adds or edits a repo.
func (w *walls) form(title string, x core.Wall, withName bool) tea.Cmd {
	var fields []field
	if withName {
		fields = append(fields, field{label: "Repo", hint: "personal"})
	}
	fields = append(fields,
		field{label: "GitHub url (owner/name is enough)", value: x.URL, hint: "de-hv/wallpapers"},
		field{label: "Folder (empty: ~/Pictures/<name>)", value: x.Dir, hint: "~/Pictures/wallpapers"},
	)
	return open(newPrompt(title, fields, func(v []string) (tea.Cmd, error) {
		if withName {
			if !core.ValidWall(v[0]) {
				return nil, errors.New("a repo is lower-case letters, digits, - and _")
			}
			x.Name, v = v[0], v[1:]
		}
		u, err := core.GitURL(v[0])
		if err != nil {
			return nil, err
		}
		x.URL, x.Dir = u, v[1]
		if x.Dir == "" {
			x.Dir = core.DefaultWallDir(u)
		}
		if err := core.SaveWall(x); err != nil {
			return nil, err
		}
		return tea.Batch(say("ok", "saved "+x.Name+" · "+x.Short()+" → "+x.Dir+" · s clones it"), w.load()), nil
	}))
}

func (w *walls) capturing() bool { return w.t.filtering }

func (w *walls) hints() []hint {
	return []hint{{"s", "sync"}, {"a", "add image"}, {"o", "open"}, {"u", "use"}, {"S", "sync all"}, {"n", "new"}, {"e", "edit"}, {"d", "forget"}}
}

func (w *walls) view(width, h int, focused bool) string {
	switch {
	case !w.loaded:
		return ui.Dim.Render("reading repos…")
	case len(w.ws) == 0:
		return "No wallpaper repos yet.\n" + ui.Dim.Render("n adds one: a GitHub repo of images and the folder it syncs with.")
	}
	return w.t.render(width, h, focused)
}
