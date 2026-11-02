package core

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/de-hv/dehv/internal/config"
)

// Agent is a coding agent a skill can be linked into.
type Agent struct {
	Name      string
	Marker    string // the folder that tells it is installed
	Dir       string // where its skills go
	Installed bool
}

// Agents is every agent dehv knows, installed or not; the first one is the
// default.
func Agents() []Agent {
	h := config.Home()
	cursor := filepath.Join(h, "Library/Application Support/Cursor/User")
	xcode := filepath.Join(h, "Library/Developer/Xcode/CodingAssistant")
	as := []Agent{
		{Name: "claude", Marker: filepath.Join(h, ".claude"), Dir: filepath.Join(h, ".claude/skills")},
		{Name: "cursor", Marker: cursor, Dir: filepath.Join(cursor, "skills")},
		{Name: "copilot", Marker: filepath.Join(h, ".copilot"), Dir: filepath.Join(h, ".copilot/skills")},
		{Name: "xcode", Marker: xcode, Dir: xcode},
		{Name: "codex", Marker: filepath.Join(h, ".agents"), Dir: filepath.Join(h, ".agents/skills")},
	}
	for i := range as {
		st, err := os.Stat(as[i].Marker)
		as[i].Installed = err == nil && st.IsDir()
	}
	return as
}

// InstalledAgents is the ones installed, in the table's order.
func InstalledAgents() []Agent {
	var in []Agent
	for _, a := range Agents() {
		if a.Installed {
			in = append(in, a)
		}
	}
	return in
}

// AgentNamed is the agent of that name.
func AgentNamed(name string) (Agent, bool) {
	for _, a := range Agents() {
		if a.Name == name {
			return a, true
		}
	}
	return Agent{}, false
}

// SkillsDir is where my skills are: skill.dir in the config, or
// ~/Documents/dev/skills.
func SkillsDir() string {
	return config.Setting("skill.dir", filepath.Join(config.Home(), "Documents/dev/skills"))
}

// ValidSkill says whether name can be a skill: a single folder.
func ValidSkill(name string) bool {
	return name != "" && !strings.Contains(name, "/") && !strings.HasPrefix(name, ".")
}

// Skill is a folder of SKILL.md in SkillsDir, or a link of mine left pointing
// at one that is gone.
type Skill struct {
	Name        string
	Description string          // from its frontmatter; "" means no agent can trigger it
	Linked      map[string]bool // agent name → linked
	Broken      bool            // only my links are left
}

// Skills lists them, by name, the broken ones last.
func Skills() ([]Skill, error) {
	dir := SkillsDir()
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	agents := InstalledAgents()
	var ss []Skill
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		s := Skill{Name: e.Name(), Linked: map[string]bool{}}
		s.Description = Description(filepath.Join(dir, e.Name(), "SKILL.md"))
		for _, a := range agents {
			s.Linked[a.Name] = isLink(filepath.Join(a.Dir, e.Name()))
		}
		ss = append(ss, s)
	}
	broken := map[string]*Skill{}
	var order []string
	for _, a := range agents {
		links, _ := os.ReadDir(a.Dir)
		for _, l := range links {
			p := filepath.Join(a.Dir, l.Name())
			if !isLink(p) {
				continue
			}
			if _, err := os.Stat(p); err == nil {
				continue
			}
			if t, _ := os.Readlink(p); !strings.HasPrefix(t, dir+"/") {
				continue
			}
			b, ok := broken[l.Name()]
			if !ok {
				b = &Skill{Name: l.Name(), Linked: map[string]bool{}, Broken: true}
				broken[l.Name()] = b
				order = append(order, l.Name())
			}
			b.Linked[a.Name] = true
		}
	}
	sort.Strings(order)
	for _, n := range order {
		ss = append(ss, *broken[n])
	}
	return ss, nil
}

// Description is the description in a SKILL.md's frontmatter.
func Description(file string) string {
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	fences := 0
	var block []string
	inBlock := false
	for sc.Scan() {
		line := sc.Text()
		if inBlock {
			// A YAML block (description: >- …) goes on while indented.
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				block = append(block, strings.TrimSpace(line))
				continue
			}
			return strings.Join(block, " ")
		}
		if strings.HasPrefix(line, "---") {
			if fences++; fences > 1 {
				return ""
			}
			continue
		}
		if fences == 1 {
			if v, ok := strings.CutPrefix(line, "description:"); ok {
				v = strings.TrimSpace(v)
				switch v {
				case ">", ">-", ">+", "|", "|-", "|+":
					inBlock = true
					continue
				}
				return strings.Trim(v, `"'`)
			}
		}
	}
	return strings.Join(block, " ")
}

func isLink(p string) bool {
	st, err := os.Lstat(p)
	return err == nil && st.Mode()&os.ModeSymlink != 0
}

// SkillExists says whether there is a skill of that name.
func SkillExists(name string) bool {
	_, err := os.Lstat(filepath.Join(SkillsDir(), name))
	return err == nil
}

// LinkedInto is the installed agents a skill is linked into.
func LinkedInto(name string) []string {
	var in []string
	for _, a := range InstalledAgents() {
		if isLink(filepath.Join(a.Dir, name)) {
			in = append(in, a.Name)
		}
	}
	return in
}

// CreateSkill makes its folder with a SKILL.md that has its frontmatter.
func CreateSkill(name string) error {
	dir := filepath.Join(SkillsDir(), name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		if os.IsExist(err) {
			return Fail("skill '"+name+"' already exists", "")
		}
		return err
	}
	front := "---\nname: " + name + "\ndescription:\n---\n"
	return os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(front), 0o644)
}

// LinkSkill links it into agent a.
func LinkSkill(name string, a Agent) error {
	p := filepath.Join(a.Dir, name)
	if _, err := os.Lstat(p); err == nil && !isLink(p) {
		return Fail("cannot link into "+a.Name, "it has a "+name+" of its own")
	}
	_ = os.Remove(p)
	if err := os.MkdirAll(a.Dir, 0o755); err != nil {
		return err
	}
	return os.Symlink(filepath.Join(SkillsDir(), name), p)
}

// UnlinkSkill takes it out of agent a; only my link, a real folder of that
// name stays. False when there was none.
func UnlinkSkill(name string, a Agent) (bool, error) {
	p := filepath.Join(a.Dir, name)
	if !isLink(p) {
		return false, nil
	}
	return true, os.Remove(p)
}

// TrashSkill moves it to the Trash, through the Finder: macOS keeps the shell
// out of ~/.Trash, and only what the Finder trashed gets "Put Back". The first
// time, macOS asks to let the terminal control Finder.
func TrashSkill(name string) error {
	dir := filepath.Join(SkillsDir(), name)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return Fail("no skill '"+name+"' in "+config.Tilde(SkillsDir()), "")
	}
	if finder("delete (POSIX file (item 1 of argv) as alias)", dir) != nil {
		return Fail("Finder could not move '"+name+"' to the Trash", "")
	}
	return nil
}

// RestoreSkill brings it back from the Trash.
func RestoreSkill(name string) error {
	if SkillExists(name) {
		return Fail("skill '"+name+"' already exists", "")
	}
	if finder("move item (item 1 of argv) of trash to (POSIX file (item 2 of argv) as alias)", name, SkillsDir()) != nil {
		return Fail("no '"+name+"' in the Trash, or Finder could not move it", "")
	}
	return nil
}

func finder(line string, args ...string) error {
	a := []string{"-e", "on run argv", "-e", `tell application "Finder" to ` + line, "-e", "end run"}
	return exec.Command("osascript", append(a, args...)...).Run()
}
