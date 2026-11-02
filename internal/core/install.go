package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/de-hv/dehv/internal/config"
	"github.com/de-hv/dehv/internal/sh"
)

// Version is dehv's, set at build time; "dev" for a `go build` or `go run`.
var Version = "dev"

// DisplayVersion is "v1.0.0", or "dev" for a build from source.
func DisplayVersion() string {
	if Version == "dev" {
		return Version
	}
	return "v" + Version
}

// Where dehv comes from.
const (
	FromHomebrew = "homebrew"
	FromSource   = "source" // built from a clone
	FromFolder   = "folder"
)

// Install is where the dehv running is.
type Install struct {
	Bin     string // the command, past any link
	Dir     string // the folder it is in
	From    string
	Version string
	Rev     string // the commit it was built from, "a1c9e2f" or "a1c9e2f-dirty"
}

// ThisInstall reads it.
func ThisInstall() Install {
	bin, _ := os.Executable()
	if real, err := filepath.EvalSymlinks(bin); err == nil {
		bin = real
	}
	in := Install{Bin: bin, Dir: filepath.Dir(bin), Version: Version, From: FromFolder}
	switch {
	case strings.Contains(bin, "/Caskroom/dehv/") || strings.Contains(bin, "/Cellar/dehv/"):
		in.From = FromHomebrew
	case Version == "dev":
		in.From = FromSource
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		var rev string
		dirty := false
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
			case "vcs.modified":
				dirty = s.Value == "true"
			}
		}
		if len(rev) > 7 {
			rev = rev[:7]
		}
		if rev != "" && dirty {
			rev += "-dirty"
		}
		in.Rev = rev
	}
	return in
}

// BrewPrefix is Homebrew's prefix, /opt/homebrew, when dehv comes from it.
func (in Install) BrewPrefix() string {
	for _, mark := range []string{"/Caskroom/", "/Cellar/"} {
		if i := strings.Index(in.Bin, mark); i >= 0 {
			return in.Bin[:i]
		}
	}
	return ""
}

// Check is one line of `dehv doctor`.
type Check struct {
	Group  string // install, needs, files
	Status string // ok, warn, fail
	What   string
	Detail string
}

// Needs is what dehv runs: name, where it comes from, whether it is required,
// what for when it is not.
var Needs = [][4]string{
	{"git", "Command Line Tools", "required", ""},
	{"lsof", "macOS", "required", ""},
	{"pbcopy", "macOS", "required", ""},
	{"osascript", "macOS", "required", ""},
	{"sips", "macOS", "required", ""},
	{"networksetup", "macOS", "required", ""},
	{"gh", "Homebrew", "optional", "dehv wall uses it for ssh or https"},
}

// hasGit counts git once the Command Line Tools are there: macOS ships a
// /usr/bin/git that only offers to install them.
func has(name string) bool {
	p, err := exec.LookPath(name)
	if err != nil {
		return false
	}
	if name == "git" && p == "/usr/bin/git" {
		return exec.Command("xcode-select", "-p").Run() == nil
	}
	return true
}

// Doctor checks the install of dehv, what it needs and its files. It only
// looks.
func Doctor() []Check {
	in := ThisInstall()
	var cs []Check
	add := func(g, s, w, d string) { cs = append(cs, Check{g, s, w, d}) }

	if p, err := exec.LookPath("dehv"); err != nil {
		add("install", "warn", "command", "dehv is not in the PATH, only "+config.Tilde(in.Bin))
	} else if real, _ := filepath.EvalSymlinks(p); real == in.Bin {
		add("install", "ok", "command", config.Tilde(p))
	} else {
		add("install", "warn", "command", "the dehv in the PATH is "+config.Tilde(p)+", not this one")
	}
	if pre := in.BrewPrefix(); pre != "" {
		comp := filepath.Join(pre, "share/zsh/site-functions/_dehv")
		if config.Exists(comp) {
			add("install", "ok", "completion", config.Tilde(comp))
		} else {
			add("install", "fail", "completion", config.Tilde(comp)+" is missing: brew reinstall dehv")
		}
	} else {
		add("install", "warn", "completion", "not from Homebrew: 'dehv completion zsh' prints it")
	}

	for _, n := range Needs {
		switch {
		case has(n[0]):
			add("needs", "ok", n[0], n[1])
		case n[2] == "optional":
			add("needs", "warn", n[0], "not installed, optional: "+n[3])
		default:
			add("needs", "fail", n[0], "not installed, comes with "+n[1])
		}
	}

	// The files the commands write, when they are there; profiles and walls
	// stay readable only by me.
	for _, name := range []string{"config", "profiles", "walls"} {
		f := config.File(name)
		st, err := os.Stat(f)
		switch {
		case err != nil:
			add("files", "ok", name, "none yet")
		case name != "config" && st.Mode().Perm() != 0o600:
			add("files", "warn", name, "readable by others: chmod 600 "+config.Tilde(f))
		default:
			add("files", "ok", name, config.Tilde(f))
		}
	}
	return cs
}

// Healthy says no check failed.
func Healthy(cs []Check) bool {
	for _, c := range cs {
		if c.Status == "fail" {
			return false
		}
	}
	return true
}

// Tally is "n/total ok".
func Tally(cs []Check) string {
	ok := 0
	for _, c := range cs {
		if c.Status == "ok" {
			ok++
		}
	}
	return fmt.Sprintf("%d/%d ok", ok, len(cs))
}

// Repo is the context the hub's header shows: the folder, its branch and the
// profile it commits as.
type Repo struct {
	Dir     string
	Branch  string
	Profile string
}

// ThisRepo reads it for the current folder.
func ThisRepo() Repo {
	wd, _ := os.Getwd()
	r := Repo{Dir: config.Tilde(wd)}
	if !InRepo("") {
		return r
	}
	r.Branch, _ = sh.Out("git", "branch", "--show-current")
	if p, ok := CurrentProfile(""); ok {
		r.Profile = p.Name
	}
	return r
}
