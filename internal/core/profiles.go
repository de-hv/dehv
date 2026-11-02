package core

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/de-hv/dehv/internal/config"
	"github.com/de-hv/dehv/internal/sh"
)

// Profile is one of my git identities (personal, a company, a client), kept
// per machine in ~/.config/dehv/profiles, out of any repo:
//
//	[profile "acme"]
//		name = David Higuita
//		email = david@acme.com
type Profile struct {
	Name  string // acme
	User  string // David Higuita
	Email string
}

// Ident is "User <email>", as git writes an author.
func (p Profile) Ident() string { return p.User + " <" + p.Email + ">" }

// ProfilesFile is where they are.
func ProfilesFile() string { return config.File("profiles") }

var profileName = regexp.MustCompile(`^[a-z0-9_-]+$`)

// ValidProfile says whether name can be one: lower-case letters, digits, - and
// _, and never only digits (those are a count of commits).
func ValidProfile(name string) bool {
	if !profileName.MatchString(name) {
		return false
	}
	_, err := strconv.Atoi(name)
	return err != nil
}

// Profiles are mine, in the file's order.
func Profiles() []Profile {
	file := ProfilesFile()
	var ps []Profile
	for _, n := range config.Sections(file, "profile", "email") {
		p, _ := GetProfile(n)
		ps = append(ps, p)
	}
	return ps
}

// GetProfile is the one named, and whether there is one.
func GetProfile(name string) (Profile, bool) {
	file := ProfilesFile()
	p := Profile{
		Name:  name,
		User:  config.Get(file, "profile."+name+".name"),
		Email: config.Get(file, "profile."+name+".email"),
	}
	return p, p.Email != ""
}

// SaveProfile creates or edits it.
func SaveProfile(p Profile) error {
	if p.User == "" {
		return Fail("no name, nothing saved", "")
	}
	if !strings.Contains(p.Email, "@") || strings.HasPrefix(p.Email, "@") || strings.HasSuffix(p.Email, "@") {
		return Fail(fmt.Sprintf("'%s' is not an email, nothing saved", p.Email), "")
	}
	file := ProfilesFile()
	if err := config.Touch(file); err != nil {
		return err
	}
	if err := config.Set(file, "profile."+p.Name+".name", p.User); err != nil {
		return err
	}
	return config.Set(file, "profile."+p.Name+".email", p.Email)
}

// RemoveProfile removes it.
func RemoveProfile(name string) error {
	if _, ok := GetProfile(name); !ok {
		return Fail("no profile "+name, "")
	}
	return config.RemoveSection(ProfilesFile(), "profile."+name)
}

// GlobalIdent is git's global user.name and user.email, to start a new
// profile from.
func GlobalIdent() (string, string) {
	n, _ := sh.Out("git", "config", "--global", "user.name")
	e, _ := sh.Out("git", "config", "--global", "user.email")
	return n, e
}

// RepoIdent is the user.name and user.email the repo in dir commits with.
func RepoIdent(dir string) (string, string) {
	ctx := context.Background()
	n, _ := sh.Git(ctx, dir, "config", "user.name")
	e, _ := sh.Git(ctx, dir, "config", "user.email")
	return n, e
}

// CurrentProfile is the profile the repo in dir commits as, if any.
func CurrentProfile(dir string) (Profile, bool) {
	if !InRepo(dir) {
		return Profile{}, false
	}
	n, e := RepoIdent(dir)
	for _, p := range Profiles() {
		if p.User == n && p.Email == e {
			return p, true
		}
	}
	return Profile{}, false
}

// UseProfile sets the repo's user.name and user.email to p's.
func UseProfile(dir string, p Profile) error {
	ctx := context.Background()
	if _, err := sh.Git(ctx, dir, "config", "--local", "user.name", p.User); err != nil {
		return err
	}
	_, err := sh.Git(ctx, dir, "config", "--local", "user.email", p.Email)
	return err
}

// Amended is a commit rewritten by AmendAuthors.
type Amended struct {
	Hash    string
	Subject string
}

// AmendAuthors makes p the author of the last n commits: an amend for one, a
// rebase for more, so only before pushing them. Nothing, and no error, when
// they already are.
func AmendAuthors(dir string, p Profile, n int) ([]Amended, error) {
	ctx := context.Background()
	total, err := sh.Git(ctx, dir, "rev-list", "--count", "HEAD")
	if err != nil {
		return nil, Fail("no commits here yet", "")
	}
	if t, _ := strconv.Atoi(total); n > t {
		return nil, Fail(fmt.Sprintf("only %s commits here", total), "")
	} else if n == t {
		total = "root"
	}
	authors, _ := sh.Git(ctx, dir, "log", "-n", strconv.Itoa(n), "--format=%an <%ae>")
	other := false
	for _, a := range sh.Lines(authors) {
		if a != p.Ident() {
			other = true
		}
	}
	if !other {
		return nil, nil
	}
	if n == 1 {
		if _, err := sh.Git(ctx, dir, "commit", "--amend", "-q", "--no-edit", "--author="+p.Ident()); err != nil {
			return nil, err
		}
	} else {
		base := "HEAD~" + strconv.Itoa(n)
		if total == "root" {
			base = "--root"
		}
		exec := "git commit --amend --no-edit -q --author=" + shellQuote(p.Ident())
		if _, err := sh.Git(ctx, dir, "rebase", base, "--exec", exec); err != nil {
			return nil, &Error{Msg: "git could not rewrite them", Hint: "'git rebase --abort' undoes it\n" + strings.TrimSpace(sh.Output(err))}
		}
	}
	out, _ := sh.Git(ctx, dir, "log", "-n", strconv.Itoa(n), "--format=%h %s")
	var done []Amended
	for _, l := range sh.Lines(out) {
		h, s, _ := strings.Cut(l, " ")
		done = append(done, Amended{Hash: h, Subject: s})
	}
	return done, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
