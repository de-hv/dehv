package tui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/de-hv/dehv/internal/config"
	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

type skillsLoaded struct {
	ss     []core.Skill
	agents []core.Agent
	err    error
}

// skills is my skills, a column per installed agent with a dot where each is
// linked.
type skills struct {
	t      table
	ss     []core.Skill
	agents []core.Agent
	err    error
	loaded bool
}

func newSkills() *skills { return &skills{t: newTable()} }

func (s *skills) name() string { return "Skills" }

func (s *skills) badge() string {
	for _, x := range s.ss {
		if x.Broken || x.Description == "" {
			return ui.Warn.Render("!")
		}
	}
	if len(s.ss) > 0 {
		return ui.Dim.Render(strconv.Itoa(len(s.ss)))
	}
	return ""
}

func (s *skills) heading() string {
	if !s.loaded {
		return "loading…"
	}
	return ui.Plural(len(s.ss), "skill", "skills") + " in " + config.Tilde(core.SkillsDir())
}

func (s *skills) load() tea.Cmd {
	return func() tea.Msg {
		ss, err := core.Skills()
		return skillsLoaded{ss, core.InstalledAgents(), err}
	}
}

func (s *skills) update(msg tea.Msg, _ bool) tea.Cmd {
	switch msg := msg.(type) {
	case skillsLoaded:
		s.loaded, s.ss, s.agents, s.err = true, msg.ss, msg.agents, msg.err
		cols := []column{{title: "SKILL"}}
		for _, a := range msg.agents {
			cols = append(cols, column{title: a.Name, drop: 2})
		}
		s.t.cols = append(cols, column{title: "DESCRIPTION"})
		var rows []row
		for _, x := range msg.ss {
			cells := []string{ui.Bold.Render(x.Name)}
			for _, a := range msg.agents {
				switch {
				case x.Linked[a.Name] && x.Broken:
					cells = append(cells, ui.Err.Render("●"))
				case x.Linked[a.Name]:
					cells = append(cells, ui.OKMark.Render("●"))
				default:
					cells = append(cells, ui.Dim.Render("○"))
				}
			}
			switch {
			case x.Broken:
				cells = append(cells, ui.Err.Render("! broken link"))
			case x.Description == "":
				cells = append(cells, ui.Warn.Render("! no description"))
			default:
				cells = append(cells, ui.Dim.Render(x.Description))
			}
			rows = append(rows, row{key: x.Name, cells: cells, search: x.Name + " " + x.Description})
		}
		s.t.setRows(rows)
	case editedMsg:
		return s.load()
	}
	return nil
}

type editedMsg struct{ err error }

func (s *skills) selected() (core.Skill, bool) {
	r, ok := s.t.selected()
	if !ok {
		return core.Skill{}, false
	}
	for _, x := range s.ss {
		if x.Name == r.key {
			return x, true
		}
	}
	return core.Skill{}, false
}

func (s *skills) key(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if handled, cmd := s.t.key(msg); handled {
		return true, cmd
	}
	k := msg.String()
	switch k {
	case "n":
		return true, open(newPrompt("New skill", []field{{label: "Name", hint: "notes"}}, func(v []string) (tea.Cmd, error) {
			if !core.ValidSkill(v[0]) {
				return nil, errors.New("a single folder name, not starting with .")
			}
			if err := core.CreateSkill(v[0]); err != nil {
				return nil, err
			}
			return tea.Batch(say("ok", "created "+v[0]+" · enter links it, e edits its SKILL.md"), s.load()), nil
		}))
	case "R":
		return true, open(newPrompt("Restore a skill from the Trash", []field{{label: "Name"}}, func(v []string) (tea.Cmd, error) {
			if err := core.RestoreSkill(v[0]); err != nil {
				return nil, err
			}
			return tea.Batch(say("ok", "restored "+v[0]+" · enter links it"), s.load()), nil
		}))
	}
	x, ok := s.selected()
	if !ok {
		return k == "enter" || k == "e" || k == "d", nil
	}
	switch k {
	case "enter", "l":
		if len(s.agents) == 0 {
			return true, say("warn", "no agent installed to link into")
		}
		var opts []string
		var on []bool
		for _, a := range s.agents {
			opts = append(opts, ui.Pad(a.Name, 9)+ui.Dim.Render(config.Tilde(a.Dir)))
			on = append(on, x.Linked[a.Name])
		}
		agents := s.agents
		return true, open(&checklist{title: "Link " + x.Name + " into", options: opts, on: on, onDone: func(on []bool) tea.Cmd {
			return tea.Sequence(func() tea.Msg {
				var did, failed []string
				for i, a := range agents {
					switch {
					case on[i] && !x.Linked[a.Name]:
						if err := core.LinkSkill(x.Name, a); err != nil {
							failed = append(failed, a.Name)
						} else {
							did = append(did, "+"+a.Name)
						}
					case !on[i] && x.Linked[a.Name]:
						if _, err := core.UnlinkSkill(x.Name, a); err != nil {
							failed = append(failed, a.Name)
						} else {
							did = append(did, "-"+a.Name)
						}
					}
				}
				switch {
				case len(failed) > 0:
					return statusMsg{"err", "could not change " + strings.Join(failed, ", ")}
				case len(did) == 0:
					return statusMsg{"info", "already linked there, nothing to do"}
				}
				return statusMsg{"ok", x.Name + " · " + strings.Join(did, " ")}
			}, s.load())
		}})
	case "e":
		if x.Broken {
			return true, say("warn", x.Name+" is only a broken link")
		}
		file := filepath.Join(core.SkillsDir(), x.Name, "SKILL.md")
		editor := os.Getenv("EDITOR")
		var c *exec.Cmd
		if editor != "" {
			parts := strings.Fields(editor)
			c = exec.Command(parts[0], append(parts[1:], file)...)
		} else {
			c = exec.Command("open", "-t", file)
		}
		return true, tea.ExecProcess(c, func(err error) tea.Msg { return editedMsg{err} })
	case "d":
		if x.Broken {
			return true, open(&confirm{title: "Remove the broken links of " + x.Name + "?", yes: "remove", onYes: tea.Sequence(func() tea.Msg {
				for _, a := range s.agents {
					_, _ = core.UnlinkSkill(x.Name, a)
				}
				return statusMsg{"ok", "removed the links of " + x.Name}
			}, s.load())})
		}
		return true, open(&confirm{
			title: "Move " + x.Name + " to the Trash?",
			lines: []string{"it is unlinked from every agent", "R brings it back, or the Finder's Put Back"},
			yes:   "trash",
			onYes: tea.Sequence(func() tea.Msg {
				if err := core.TrashSkill(x.Name); err != nil {
					return statusMsg{"err", err.Error()}
				}
				for _, a := range s.agents {
					_, _ = core.UnlinkSkill(x.Name, a)
				}
				return statusMsg{"ok", "trashed " + x.Name}
			}, s.load()),
		})
	}
	return false, nil
}

func (s *skills) capturing() bool { return s.t.filtering }

func (s *skills) hints() []hint {
	return []hint{{"enter", "link"}, {"e", "edit"}, {"n", "new"}, {"d", "trash"}, {"R", "restore"}, {"/", "filter"}}
}

func (s *skills) view(w, h int, focused bool) string {
	switch {
	case !s.loaded:
		return ui.Dim.Render("reading skills…")
	case s.err != nil:
		return ui.Err.Render("✗") + " " + s.err.Error()
	case len(s.ss) == 0:
		return "No skills in " + config.Tilde(core.SkillsDir()) + ".\n" + ui.Dim.Render("n creates one; skill.dir in ~/.config/dehv/config moves the folder.")
	}
	return s.t.render(w, h, focused)
}
