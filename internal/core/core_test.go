package core

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestParseLsof(t *testing.T) {
	out := `COMMAND     PID   USER   FD   TYPE             DEVICE SIZE/OFF NODE NAME
node       4821  david   23u  IPv4 0x1234567890abcdef      0t0  TCP *:3000 (LISTEN)
node       4821  david   24u  IPv6 0x1234567890abcdee      0t0  TCP [::1]:3000 (LISTEN)
postgres    612  david    7u  IPv4 0x1234567890abcded      0t0  TCP 127.0.0.1:5432 (LISTEN)
Google\x20Chrome 900 david 7u IPv4 0x1 0t0 TCP 127.0.0.1:9222 (LISTEN)`
	got := ParseLsof(out)
	want := []Listener{
		{Port: 3000, Program: "node", PID: 4821, User: "david"},
		{Port: 5432, Program: "postgres", PID: 612, User: "david"},
		{Port: 9222, Program: "Google Chrome", PID: 900, User: "david"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	if ParseLsof("") != nil {
		t.Error("nothing listening is no rows")
	}
}

func TestParseBattery(t *testing.T) {
	for _, c := range []struct {
		in    string
		pct   int
		state string
	}{
		{"Now drawing from 'Battery Power'\n -InternalBattery-0 (id=123)\t87%; discharging; 5:12 remaining present: true", 87, ""},
		{"Now drawing from 'AC Power'\n -InternalBattery-0 (id=123)\t100%; charged; 0:00 remaining present: true", 100, "charged"},
		{"Now drawing from 'AC Power'", -1, ""},
	} {
		pct, state := ParseBattery(c.in)
		if pct != c.pct || state != c.state {
			t.Errorf("%q: got %d %q, want %d %q", c.in, pct, state, c.pct, c.state)
		}
	}
}

func TestUptime(t *testing.T) {
	for d, want := range map[time.Duration]string{
		12 * time.Minute:             "12m",
		2*time.Hour + 5*time.Minute:  "2h 5m",
		25 * time.Hour:               "1 day",
		3*24*time.Hour + time.Minute: "3 days",
	} {
		if got := Uptime(d); got != want {
			t.Errorf("%v: got %q, want %q", d, got, want)
		}
	}
}

func TestShortURL(t *testing.T) {
	for in, want := range map[string]string{
		"git@github.com:de-hv/wallpapers.git":     "de-hv/wallpapers",
		"https://github.com/de-hv/wallpapers.git": "de-hv/wallpapers",
		"https://github.com/de-hv/wallpapers":     "de-hv/wallpapers",
		"ssh://git@github.com/de-hv/walls.git":    "de-hv/walls",
	} {
		if got := ShortURL(in); got != want {
			t.Errorf("%s: got %q, want %q", in, got, want)
		}
	}
}

func TestChanges(t *testing.T) {
	st := "A\ta.jpg\nA\tb.jpg\nD\tc.jpg\nR100\td.jpg\te.jpg\nM\tf.jpg"
	if got := changes(st, false); got != "2 added, 1 removed, 1 renamed, 1 changed" {
		t.Errorf("got %q", got)
	}
	if got := changes(st, true); got != "add 2, remove 1, rename 1, change 1" {
		t.Errorf("got %q", got)
	}
}

func TestValidNames(t *testing.T) {
	for name, ok := range map[string]bool{"acme": true, "my-co_2": true, "123": false, "Acme": false, "a b": false, "": false} {
		if ValidProfile(name) != ok {
			t.Errorf("ValidProfile(%q) should be %v", name, ok)
		}
	}
	for name, ok := range map[string]bool{"notes": true, "a/b": false, ".hidden": false, "": false} {
		if ValidSkill(name) != ok {
			t.Errorf("ValidSkill(%q) should be %v", name, ok)
		}
	}
}

func TestImageName(t *testing.T) {
	for in, want := range map[string]string{
		"https://x.com/img/aurora.jpg?w=3840":  "aurora.jpg",
		"https://x.com/img/my%20wall.png":      "my wall.png",
		"https://x.com/":                       "wallpaper",
		"https://images.unsplash.com/photo-12": "photo-12",
	} {
		if got := ImageName(in); got != want {
			t.Errorf("%s: got %q, want %q", in, got, want)
		}
	}
}

func TestDescription(t *testing.T) {
	dir := t.TempDir()
	write := func(name, s string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	for name, c := range map[string]struct{ in, want string }{
		"plain":  {"---\nname: a\ndescription: Take notes\n---\n# body\n", "Take notes"},
		"quoted": {"---\ndescription: \"Take notes\"\n---\n", "Take notes"},
		"folded": {"---\nname: a\ndescription: >-\n  Take meeting\n  notes\nother: x\n---\n", "Take meeting notes"},
		"empty":  {"---\nname: a\ndescription:\n---\n", ""},
		"body":   {"---\nname: a\n---\ndescription: not this\n", ""},
	} {
		if got := Description(write(name, c.in)); got != c.want {
			t.Errorf("%s: got %q, want %q", name, got, c.want)
		}
	}
}

// home points HOME and XDG_CONFIG_HOME at a temp folder for the test.
func home(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	return h
}

func TestProfilesFile(t *testing.T) {
	home(t)
	if len(Profiles()) != 0 {
		t.Fatal("no file is no profiles")
	}
	if err := SaveProfile(Profile{Name: "acme", User: "David H", Email: "d@acme.com"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveProfile(Profile{Name: "personal", User: "David H", Email: "d@me.com"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveProfile(Profile{Name: "bad", User: "x", Email: "nope"}); err == nil {
		t.Error("an email needs an @")
	}
	ps := Profiles()
	if len(ps) != 2 || ps[0].Name != "acme" || ps[1].Email != "d@me.com" {
		t.Fatalf("got %+v", ps)
	}
	st, _ := os.Stat(ProfilesFile())
	if st.Mode().Perm() != 0o600 {
		t.Errorf("profiles is %v, not 600", st.Mode().Perm())
	}
	if err := RemoveProfile("acme"); err != nil {
		t.Fatal(err)
	}
	if _, ok := GetProfile("acme"); ok {
		t.Error("acme is still there")
	}
}

func TestWallsFile(t *testing.T) {
	home(t)
	if _, _, err := WallInUse(); err == nil {
		t.Error("no repos is an error")
	}
	if err := SaveWall(Wall{Name: "personal", URL: "git@github.com:de-hv/w.git", Dir: "~/Pictures/w"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveWall(Wall{Name: "work", URL: "https://github.com/acme/w.git", Dir: "~/Pictures/acme"}); err != nil {
		t.Fatal(err)
	}
	w, ok, err := WallInUse()
	if err != nil || !ok || w.Name != "personal" {
		t.Fatalf("the first one is in use: got %+v %v %v", w, ok, err)
	}
	if _, err := RemoveWall("personal"); err != nil {
		t.Fatal(err)
	}
	// With the one in use forgotten and one left, it is that one.
	if w, ok, _ := WallInUse(); !ok || w.Name != "work" {
		t.Fatalf("got %+v", w)
	}
	if got := WallState(Wall{Dir: "~/nowhere"}); got != "not cloned" {
		t.Errorf("got %q", got)
	}
}

func TestSkills(t *testing.T) {
	h := home(t)
	if err := os.MkdirAll(filepath.Join(h, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(SkillsDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CreateSkill("notes"); err != nil {
		t.Fatal(err)
	}
	if err := CreateSkill("notes"); err == nil {
		t.Error("creating it twice fails")
	}
	claude, _ := AgentNamed("claude")
	if err := LinkSkill("notes", claude); err != nil {
		t.Fatal(err)
	}
	ss, err := Skills()
	if err != nil || len(ss) != 1 || !ss[0].Linked["claude"] || ss[0].Description != "" {
		t.Fatalf("got %+v %v", ss, err)
	}
	// Its folder gone, the link left behind is a broken one.
	if err := os.RemoveAll(filepath.Join(SkillsDir(), "notes")); err != nil {
		t.Fatal(err)
	}
	ss, _ = Skills()
	if len(ss) != 1 || !ss[0].Broken {
		t.Fatalf("got %+v", ss)
	}
	if did, err := UnlinkSkill("notes", claude); !did || err != nil {
		t.Fatalf("unlink: %v %v", did, err)
	}
}
