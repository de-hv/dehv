// Package config reads and writes this Mac's files in ~/.config/dehv. They are
// shaped like git configs and handled with `git config -f`, as the zsh dehv
// did, so the files it wrote keep working and git does the quoting:
//
//	[skill]
//		dir = ~/Documents/dev/skills
//	[tap]
//		owner = de-hv
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/de-hv/dehv/internal/sh"
)

// Home is the user's home folder.
func Home() string {
	h, _ := os.UserHomeDir()
	return h
}

// Dir is ~/.config/dehv, or the same under $XDG_CONFIG_HOME.
func Dir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(Home(), ".config")
	}
	return filepath.Join(base, "dehv")
}

// File is the path of one of them: config, profiles or walls.
func File(name string) string { return filepath.Join(Dir(), name) }

// Tilde writes the home folder at the start of path as ~.
func Tilde(path string) string {
	h := Home()
	if h != "" && (path == h || strings.HasPrefix(path, h+"/")) {
		return "~" + path[len(h):]
	}
	return path
}

// Expand turns a leading ~ into the home folder.
func Expand(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		return Home() + path[1:]
	}
	return path
}

// Exists says whether file is there.
func Exists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

// Get is the value of key in file, or "". Without the file git is not asked
// at all.
func Get(file, key string) string {
	if !Exists(file) {
		return ""
	}
	v, _ := sh.Out("git", "config", "-f", file, "--get", key)
	return v
}

// Set sets key to value in file, creating it.
func Set(file, key, value string) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	_, err := sh.Out("git", "config", "-f", file, key, value)
	return err
}

// Unset removes key from file, if it is there.
func Unset(file, key string) {
	if Exists(file) {
		_, _ = sh.Out("git", "config", "-f", file, "--unset", key)
	}
}

// RemoveSection removes a whole [section "name"].
func RemoveSection(file, section string) error {
	_, err := sh.Out("git", "config", "-f", file, "--remove-section", section)
	return err
}

// Sections lists the names of the [kind "name"] sections that have key, in the
// file's order.
func Sections(file, kind, key string) []string {
	if !Exists(file) {
		return nil
	}
	out, _ := sh.Out("git", "config", "-f", file, "--name-only", "--get-regexp", `^`+kind+`\..*\.`+key+`$`)
	var names []string
	for _, k := range sh.Lines(out) {
		k = strings.TrimPrefix(k, kind+".")
		names = append(names, strings.TrimSuffix(k, "."+key))
	}
	return names
}

// Touch creates file, empty and readable only by me, unless it is there.
func Touch(file string) error {
	if Exists(file) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, nil, 0o600)
}

// Blank moves file to file.bak and starts an empty one.
func Blank(file string) error {
	if err := os.Rename(file, file+".bak"); err != nil {
		return err
	}
	return Touch(file)
}

// Truncate empties file, keeping its mode.
func Truncate(file string) error {
	return os.Truncate(file, 0)
}

// Setting is a key of ~/.config/dehv/config, or def without it, with a leading
// ~ expanded.
func Setting(key, def string) string {
	v := Get(File("config"), key)
	if v == "" {
		v = def
	}
	return Expand(v)
}
