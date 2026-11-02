package core

import (
	"context"
	"strings"

	"github.com/de-hv/dehv/internal/sh"
)

// Why a branch can go, or not.
const (
	Merged   = "merged"
	Gone     = "gone"
	Unmerged = "unmerged"
)

// Branch is a local branch, with why it can go and how old its last commit is.
type Branch struct {
	Name string
	Why  string // Merged, Gone or Unmerged
	Age  string // "3 weeks ago"
}

// CanGo says it comes ticked: merged into the default branch, or gone (its
// upstream deleted, as a squash-merged PR leaves it).
func (b Branch) CanGo() bool { return b.Why != Unmerged }

// Branches is a repo's local branches but the default one and the one checked
// out.
type Branches struct {
	Base    string // "" when there is no default branch
	Current string
	List    []Branch
}

// InRepo says whether dir is in a git repo.
func InRepo(dir string) bool {
	_, err := sh.Git(context.Background(), dir, "rev-parse", "--git-dir")
	return err == nil
}

// Fetch fetches with --prune, so "gone" is current; nothing without a remote,
// and offline it goes on with what it knew.
func Fetch(ctx context.Context, dir string) {
	if r, _ := sh.Git(ctx, dir, "remote"); r == "" {
		return
	}
	_, _ = sh.Git(ctx, dir, "fetch", "--prune", "--quiet")
}

// ListBranches reads them.
func ListBranches(dir string) (Branches, error) {
	ctx := context.Background()
	if !InRepo(dir) {
		return Branches{}, Fail("not in a git repo", "")
	}
	var bs Branches
	base, _ := sh.Git(ctx, dir, "symbolic-ref", "-q", "--short", "refs/remotes/origin/HEAD")
	bs.Base = strings.TrimPrefix(base, "origin/")
	if bs.Base == "" {
		for _, b := range []string{"main", "master"} {
			if _, err := sh.Git(ctx, dir, "show-ref", "-q", "--verify", "refs/heads/"+b); err == nil {
				bs.Base = b
				break
			}
		}
	}
	bs.Current, _ = sh.Git(ctx, dir, "branch", "--show-current")

	merged := map[string]bool{}
	if bs.Base != "" {
		out, _ := sh.Git(ctx, dir, "branch", "--format=%(refname:short)", "--merged", bs.Base)
		for _, n := range sh.Lines(out) {
			merged[n] = true
		}
	}
	out, err := sh.Git(ctx, dir, "for-each-ref", "--format=%(refname:short)|%(upstream:track)|%(committerdate:relative)", "refs/heads")
	if err != nil {
		return bs, err
	}
	for _, row := range sh.Lines(out) {
		f := strings.SplitN(row, "|", 3)
		if len(f) < 3 || f[0] == bs.Base || f[0] == bs.Current {
			continue
		}
		b := Branch{Name: f[0], Why: Unmerged, Age: f[2]}
		if merged[f[0]] {
			b.Why = Merged
		} else if f[1] == "[gone]" {
			b.Why = Gone
		}
		bs.List = append(bs.List, b)
	}
	return bs, nil
}

// DeleteBranch deletes it with -D, not -d: a gone branch was usually
// squash-merged, which -d cannot see.
func DeleteBranch(dir, name string) error {
	_, err := sh.Git(context.Background(), dir, "branch", "-D", "-q", name)
	return err
}
