package core

import "github.com/de-hv/dehv/internal/config"

// Tap kinds: what is being released to my homebrew-tap.
var TapKinds = []struct{ Kind, About string }{
	{"goreleaser", "a Go tool released by GoReleaser, like dehv"},
	{"cask", "an app (.app), like inlay"},
	{"formula", "a command line tool built from its tag's source"},
}

// TapRow is a line of the steps: h a heading (Note, what it is about), s a
// step, c a command, e a line of a .rb, and "" a blank line. Generic names
// are written <name>, for the caller to style.
type TapRow struct {
	Kind string
	Text string
	Note string
}

// TapOwner is the GitHub owner of the tap and of what it releases: tap.owner
// in the config, de-hv without it.
func TapOwner() string { return config.Setting("tap.owner", "de-hv") }

// TapSteps are the steps to release a new version of kind; it only says
// them, never runs them.
func TapSteps(kind string) (about string, rows []TapRow) {
	o := TapOwner()
	h := func(t, n string) TapRow { return TapRow{Kind: "h", Text: t, Note: n} }
	s := func(t string) TapRow { return TapRow{Kind: "s", Text: t} }
	c := func(t string) TapRow { return TapRow{Kind: "c", Text: t} }
	e := func(t string) TapRow { return TapRow{Kind: "e", Text: t} }
	blank := TapRow{}
	src := "https://github.com/" + o + "/<name>/archive/refs/tags/v<version>.tar.gz"

	switch kind {
	case "goreleaser":
		return "a new version of a Go tool, like dehv", []TapRow{
			h("In the repo", "<name>"),
			s("1 raise the version: CHANGELOG, and VERSION if it has one"),
			c(`git commit -am "<name> <version>" && git push`),
			s("2 tag it: the release workflow builds it and updates the tap"),
			c("git tag v<version> && git push origin v<version>"),
			s("3 watch it"),
			c("gh run watch"),
			blank,
			h("On this Mac", ""),
			s("4 check it"),
			c("brew update && brew upgrade " + o + "/tap/<name>"),
			c("<name> --version"),
		}
	case "cask":
		return "a new version of an app, like inlay", []TapRow{
			h("In the repo", "<name>"),
			s("1 raise the version: edit VERSION, and CHANGELOG if it has one"),
			c(`git commit -am "<name> <version>" && git push`),
			s("2 build the app: the version goes inside it, so after step 1"),
			c("<build command>"),
			s("3 zip it"),
			c("ditto -c -k --keepParent build/<App>.app build/<App>-<version>.zip"),
			s("4 create the release: it also creates the tag"),
			c("gh release create v<version> build/<App>-<version>.zip --generate-notes"),
			s("5 its sha256: of this zip, the one uploaded"),
			c("shasum -a 256 build/<App>-<version>.zip"),
			blank,
			h("In homebrew-tap", ""),
			s("6 in Casks/<name>.rb, these two lines: the url follows the version"),
			e(`version "<version>"`),
			e(`sha256 "<sha256>"`),
			s("7 push it"),
			c(`git commit -am "<name> <version>" && git push`),
			blank,
			h("On this Mac", ""),
			s("8 check it"),
			c("brew update && brew upgrade --cask " + o + "/tap/<name>"),
		}
	}
	return "a new version of a command line tool built from source", []TapRow{
		h("In the repo", "<name>"),
		s("1 raise the version: edit VERSION, and CHANGELOG if it has one"),
		c(`git commit -am "<name> <version>" && git push`),
		s("2 tag it: this is what brew downloads"),
		c("git tag v<version> && git push origin v<version>"),
		s("3 its sha256: of the tarball GitHub makes of the tag"),
		c("curl -sL " + src + " | shasum -a 256"),
		blank,
		h("In homebrew-tap", ""),
		s("4 in Formula/<name>.rb, these two lines"),
		e(`url "` + src + `"`),
		e(`sha256 "<sha256>"`),
		s("5 push it"),
		c(`git commit -am "<name> <version>" && git push`),
		blank,
		h("On this Mac", ""),
		s("6 check it"),
		c("brew update && brew upgrade " + o + "/tap/<name>"),
		c("brew test " + o + "/tap/<name>"),
	}
}
