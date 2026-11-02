package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/de-hv/dehv/internal/config"
	"github.com/de-hv/dehv/internal/sh"
)

// Wall is a GitHub repo of wallpapers and the folder it is cloned into, kept
// per machine in ~/.config/dehv/walls with the one in use:
//
//	[wall]
//		use = personal
//	[repo "personal"]
//		url = git@github.com:de-hv/wallpapers.git
//		dir = ~/Pictures/wallpapers
//		cloned = true
//
// cloned is written once its folder is a clone: a folder never cloned is
// cloned right away, but one that was and is now gone or empty only after
// asking. Nor does a sync remove images, here or on GitHub, without asking.
type Wall struct {
	Name   string
	URL    string
	Dir    string // as written, with ~
	Cloned bool
}

// Path is its folder, ~ expanded.
func (w Wall) Path() string { return config.Expand(w.Dir) }

// Short is owner/name out of its url.
func (w Wall) Short() string { return ShortURL(w.URL) }

// WallsFile is where they are.
func WallsFile() string { return config.File("walls") }

var wallName = regexp.MustCompile(`^[a-z0-9_-]+$`)

// ValidWall says whether name can be a repo's.
func ValidWall(name string) bool { return wallName.MatchString(name) }

// ShortURL is owner/name out of a GitHub url, ssh or https.
func ShortURL(u string) string {
	u = strings.TrimSuffix(u, ".git")
	i := strings.LastIndexAny(u, ":/")
	if i < 0 {
		return u
	}
	name, rest := u[i+1:], u[:i]
	j := strings.LastIndexAny(rest, ":/")
	return rest[j+1:] + "/" + name
}

// Walls are my repos, in the file's order.
func Walls() []Wall {
	var ws []Wall
	for _, n := range config.Sections(WallsFile(), "repo", "url") {
		w, _ := GetWall(n)
		ws = append(ws, w)
	}
	return ws
}

// GetWall is the repo named, and whether there is one.
func GetWall(name string) (Wall, bool) {
	f := WallsFile()
	w := Wall{
		Name:   name,
		URL:    config.Get(f, "repo."+name+".url"),
		Dir:    config.Get(f, "repo."+name+".dir"),
		Cloned: config.Get(f, "repo."+name+".cloned") == "true",
	}
	return w, w.URL != ""
}

// WallInUse is the repo chosen with -u, or the only one. ok is false when
// there are several and none in use; err when there is none at all.
func WallInUse() (w Wall, ok bool, err error) {
	if n := config.Get(WallsFile(), "wall.use"); n != "" {
		if w, ok := GetWall(n); ok {
			return w, true, nil
		}
	}
	ws := Walls()
	switch len(ws) {
	case 0:
		return Wall{}, false, Fail("no repos yet", "'dehv wall -n <repo>' adds one")
	case 1:
		return ws[0], true, nil
	}
	return Wall{}, false, nil
}

// UseWall makes it the one in use, and says which it was.
func UseWall(name string) (string, error) {
	from := config.Get(WallsFile(), "wall.use")
	return from, config.Set(WallsFile(), "wall.use", name)
}

// GitURL turns owner/name into a GitHub url, over the protocol gh uses, or ssh
// when there is a key for it; any other url stays as it is.
func GitURL(u string) (string, error) {
	if regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`).MatchString(u) {
		proto, _ := sh.Out("gh", "config", "get", "git_protocol")
		if proto == "" {
			keys, _ := filepath.Glob(filepath.Join(config.Home(), ".ssh/id_*.pub"))
			proto = "https"
			if len(keys) > 0 {
				proto = "ssh"
			}
		}
		u = strings.TrimSuffix(u, ".git")
		if proto == "ssh" {
			return "git@github.com:" + u + ".git", nil
		}
		return "https://github.com/" + u + ".git", nil
	}
	for _, p := range []string{"https://", "ssh://", "file://"} {
		if strings.HasPrefix(u, p) {
			return u, nil
		}
	}
	if strings.HasPrefix(u, "git@") && strings.Contains(u, ":") {
		return u, nil
	}
	return "", Fail(fmt.Sprintf("'%s' is not a git url, nothing saved", u), "")
}

// DefaultWallDir is ~/Pictures/<name of the repo>.
func DefaultWallDir(u string) string {
	u = strings.TrimSuffix(u, ".git")
	return "~/Pictures/" + u[strings.LastIndexAny(u, ":/")+1:]
}

// SaveWall adds or edits a repo; the first one becomes the one in use. Another
// folder has not been cloned yet.
func SaveWall(w Wall) error {
	if w.Dir == "" {
		return Fail("no folder, nothing saved", "")
	}
	f := WallsFile()
	if err := config.Touch(f); err != nil {
		return err
	}
	w.Dir = config.Tilde(config.Expand(w.Dir))
	if old, ok := GetWall(w.Name); ok && old.Dir != w.Dir {
		config.Unset(f, "repo."+w.Name+".cloned")
	}
	if err := config.Set(f, "repo."+w.Name+".url", w.URL); err != nil {
		return err
	}
	if err := config.Set(f, "repo."+w.Name+".dir", w.Dir); err != nil {
		return err
	}
	if config.Get(f, "wall.use") == "" {
		return config.Set(f, "wall.use", w.Name)
	}
	return nil
}

// RemoveWall forgets it; its folder stays.
func RemoveWall(name string) (Wall, error) {
	w, ok := GetWall(name)
	if !ok {
		return w, Fail("no repo "+name, "")
	}
	f := WallsFile()
	if err := config.RemoveSection(f, "repo."+name); err != nil {
		return w, err
	}
	if config.Get(f, "wall.use") == name {
		config.Unset(f, "wall.use")
	}
	return w, nil
}

var imageExt = map[string]bool{
	"jpg": true, "jpeg": true, "png": true, "webp": true, "gif": true, "heic": true,
	"heif": true, "avif": true, "bmp": true, "tif": true, "tiff": true, "svg": true,
}

// CountImages is the number of images in dir and below.
func CountImages(dir string) int {
	n := 0
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.Type().IsRegular() && imageExt[strings.ToLower(strings.TrimPrefix(filepath.Ext(p), "."))] {
			n++
		}
		return nil
	})
	return n
}

// WallState is how a repo's folder is: its number of images, "gone" when it
// was cloned and is no more, or "not cloned".
func WallState(w Wall) string {
	if st, err := os.Stat(w.Path()); err == nil && st.IsDir() {
		return Plural(CountImages(w.Path()), "image", "images")
	}
	if w.Cloned {
		return "gone"
	}
	return "not cloned"
}

// Plural is "1 thing" or "n things".
func Plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// gitFail turns git's output into a reason in words, when it is a usual one.
func gitFail(err error, msg string) error {
	if errors.Is(err, context.Canceled) {
		return err
	}
	out := sh.Output(err)
	switch {
	case strings.Contains(out, "terminal prompts disabled"), strings.Contains(out, "could not read Username"):
		return Fail(msg, "GitHub asks for a login over https: 'gh auth setup-git', or an ssh url")
	case strings.Contains(out, "Permission denied (publickey)"), strings.Contains(out, "Host key verification failed"):
		return Fail(msg, "GitHub refused the ssh key: 'ssh -T git@github.com' says why")
	case strings.Contains(out, "Repository not found"), strings.Contains(out, "does not appear to be a git repository"):
		return Fail(msg, "there is no such repo, or it is not mine: check its url")
	}
	return Fail(msg, strings.TrimSpace(out))
}

// changes says what `git diff --name-status` did, as "2 added, 1 removed", or
// with verb as "add 2, remove 1", the commit's way.
func changes(status string, verb bool) string {
	n := map[byte]int{}
	for _, l := range sh.Lines(status) {
		switch l[0] {
		case 'A', 'D', 'R':
			n[l[0]]++
		default:
			n['M']++
		}
	}
	var parts []string
	for _, k := range []byte{'A', 'D', 'R', 'M'} {
		if n[k] == 0 {
			continue
		}
		if verb {
			parts = append(parts, map[byte]string{'A': "add", 'D': "remove", 'R': "rename", 'M': "change"}[k]+fmt.Sprintf(" %d", n[k]))
		} else {
			parts = append(parts, fmt.Sprintf("%d ", n[k])+map[byte]string{'A': "added", 'D': "removed", 'R': "renamed", 'M': "changed"}[k])
		}
	}
	return strings.Join(parts, ", ")
}

// confirm asks r, and turns a no into ErrDeclined and no terminal into a
// failure that says so.
func confirm(r Reporter, what, ask string, items []string) error {
	yes, err := r.Confirm(what, ask, items)
	if errors.Is(err, ErrNoTerminal) {
		return Fail(what, "run it in a terminal to confirm")
	}
	if err != nil {
		return err
	}
	if !yes {
		return ErrDeclined
	}
	return nil
}

// ErrDeclined is a confirmation answered no; whoever asked already said what
// it kept.
var ErrDeclined = errors.New("declined")

// readyWall makes sure the repo is cloned into its folder, cloning it if the
// folder is not there or empty, and that git ignores the Finder's .DS_Store in
// it. True when it cloned.
func readyWall(ctx context.Context, w Wall, r Reporter) (bool, error) {
	dir := w.Path()
	home := config.Tilde(dir)
	entries, statErr := os.ReadDir(dir)
	empty := true
	for _, e := range entries {
		if e.Name() != ".DS_Store" {
			empty = false
		}
	}
	if os.IsNotExist(statErr) || empty {
		// dehv wall never removes it nor empties it: something else did.
		if w.Cloned {
			gone := "is gone"
			if statErr == nil {
				gone = "is empty"
			}
			if err := confirm(r, home+" was cloned and now "+gone, "Clone "+w.Short()+" into it again?", nil); err != nil {
				if errors.Is(err, ErrDeclined) {
					r.Noop("left " + home + " as it is")
				}
				return false, err
			}
		}
		// Into a folder next to it, moved in place once complete, so a clone
		// that fails or is stopped leaves nothing half done.
		tmp := filepath.Join(filepath.Dir(dir), "."+filepath.Base(dir)+".cloning")
		if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
			return false, err
		}
		_ = os.RemoveAll(tmp)
		r.Step("cloning " + w.Short())
		if _, err := sh.Git(ctx, "", "clone", "-q", w.URL, tmp); err != nil {
			_ = os.RemoveAll(tmp)
			return false, gitFail(err, "could not clone "+w.Short())
		}
		_ = os.Remove(filepath.Join(dir, ".DS_Store"))
		_ = os.Remove(dir)
		if err := os.Rename(tmp, dir); err != nil {
			return false, err
		}
		if err := excludeDSStore(ctx, dir); err != nil {
			return true, err
		}
		return true, config.Set(WallsFile(), "repo."+w.Name+".cloned", "true")
	}

	top, _ := sh.Git(ctx, dir, "rev-parse", "--show-toplevel")
	real, _ := filepath.EvalSymlinks(dir)
	if top != real {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return false, Fail(home+" has a .git that is not a working repo", "a clone stopped halfway? remove that .git and try again")
		}
		return false, Fail(home+" is not a clone of the repo", "move it, or give the repo another folder with 'dehv wall -n "+w.Name+"'")
	}
	if origin, _ := sh.Git(ctx, dir, "remote", "get-url", "origin"); origin != w.URL {
		// Its url edited with -n, like https to ssh: the clone follows.
		if _, err := sh.Git(ctx, dir, "remote", "set-url", "origin", w.URL); err != nil {
			return false, err
		}
	}
	if err := excludeDSStore(ctx, dir); err != nil {
		return false, err
	}
	if !w.Cloned {
		return false, config.Set(WallsFile(), "repo."+w.Name+".cloned", "true")
	}
	return false, nil
}

func excludeDSStore(ctx context.Context, dir string) error {
	ex, err := sh.Git(ctx, dir, "rev-parse", "--git-path", "info/exclude")
	if err != nil {
		return err
	}
	if !filepath.IsAbs(ex) {
		ex = filepath.Join(dir, ex)
	}
	b, _ := os.ReadFile(ex)
	for _, l := range strings.Split(string(b), "\n") {
		if l == ".DS_Store" {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(ex), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(ex, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if len(b) > 0 && !strings.HasSuffix(string(b), "\n") {
		_, _ = f.WriteString("\n")
	}
	_, err = f.WriteString(".DS_Store\n")
	return err
}

// publish joins the repo's commits with GitHub's, theirs first, and pushes.
// What went up and what came down, each as changes says it.
func publish(ctx context.Context, w Wall, r Reporter) (up, down string, err error) {
	dir := w.Path()
	git := func(args ...string) (string, error) { return sh.Git(ctx, dir, args...) }

	r.Step("asking GitHub")
	if _, err := git("fetch", "-q", "origin"); err != nil {
		return "", "", gitFail(err, "could not reach GitHub for "+w.Name)
	}
	branch, _ := git("symbolic-ref", "--short", "HEAD")
	remote := "origin/" + branch
	_, headErr := git("rev-parse", "-q", "--verify", "HEAD")
	head := headErr == nil
	emptyTree, _ := git("hash-object", "-t", "tree", "/dev/null")

	var base string
	if _, err := git("rev-parse", "-q", "--verify", remote); err == nil {
		if !head {
			// Nothing of mine yet: start from GitHub's.
			st, _ := git("diff", "--name-status", emptyTree, remote)
			if _, err := git("checkout", "-q", "-B", branch, remote); err != nil {
				return "", "", err
			}
			return "", changes(st, false), nil
		}
		if _, err := git("merge-base", "--is-ancestor", remote, "HEAD"); err != nil {
			// Images removed on GitHub: removed here too, only once said so.
			gone, _ := git("diff", "--name-only", "-M", "--diff-filter=D", "HEAD..."+remote)
			if g := sh.Lines(gone); len(g) > 0 {
				if err := confirm(r, Plural(len(g), "removed", "removed")+" on GitHub", "Remove them in "+config.Tilde(dir)+" too?", g); err != nil {
					if errors.Is(err, ErrDeclined) {
						r.Noop("kept them: nothing came down nor went up, and 'dehv wall -s' asks again")
					}
					return "", "", err
				}
			}
			st, _ := git("diff", "--name-status", "-M", "HEAD..."+remote)
			down = changes(st, false)
			// A folder renamed on GitHub takes along the images added to it here.
			if _, err := git("-c", "merge.directoryRenames=true", "rebase", "-q", remote); err != nil {
				both, _ := git("diff", "--name-only", "--diff-filter=U")
				_, _ = git("rebase", "--abort")
				if both == "" {
					return "", "", Fail("could not join mine with GitHub's", strings.TrimSpace(sh.Output(err)))
				}
				return "", "", Fail(strings.Join(sh.Lines(both), ", ")+" changed both here and on GitHub",
					"fix it in "+config.Tilde(dir)+" with git, then 'dehv wall -s'")
			}
		}
		base = remote
	} else {
		// Empty on GitHub: everything goes up.
		if !head {
			return "", "", nil
		}
		base = emptyTree
	}

	if d, _ := git("diff", "--name-only", base, "HEAD"); d == "" {
		return "", down, nil
	}
	st, _ := git("diff", "--name-status", "-M", base, "HEAD")
	up = changes(st, false)
	r.Step("pushing to GitHub")
	if _, err := git("push", "-q", "-u", "origin", "HEAD"); err != nil {
		if !errors.Is(err, context.Canceled) {
			r.Noop("committed here: 'dehv wall -s' pushes it later")
		}
		return "", "", gitFail(err, "could not push "+w.Name)
	}
	return up, down, nil
}

// SyncWall commits what changed in the folder, joins it with GitHub's and
// pushes; the first time, only clones it.
func SyncWall(ctx context.Context, w Wall, r Reporter) error {
	dir := w.Path()
	home := config.Tilde(dir)
	cloned, err := readyWall(ctx, w, r)
	if err != nil {
		return err
	}
	if cloned {
		r.OK("cloned "+w.Name, Plural(CountImages(dir), "image", "images")+" · "+home)
		return nil
	}
	git := func(args ...string) (string, error) { return sh.Git(ctx, dir, args...) }
	if _, err := git("add", "-A"); err != nil {
		return err
	}
	if _, err := git("diff", "--cached", "--quiet"); err != nil {
		// Images removed here: removed on GitHub too, only once said so.
		gone, _ := git("diff", "--cached", "--name-only", "-M", "--diff-filter=D")
		if g := sh.Lines(gone); len(g) > 0 {
			if err := confirm(r, Plural(len(g), "removed", "removed")+" in "+home, "Remove them on GitHub too?", g); err != nil {
				_, _ = git("reset", "-q")
				if errors.Is(err, ErrDeclined) {
					r.Noop("synced nothing: they are still on GitHub, and 'git -C " + home + " restore .' brings them back")
				}
				return err
			}
		}
		st, _ := git("diff", "--cached", "--name-status", "-M")
		if _, err := git("commit", "-q", "-m", "sync: "+changes(st, true)); err != nil {
			return err
		}
	}
	up, down, err := publish(ctx, w, r)
	if err != nil {
		return err
	}
	if up == "" && down == "" {
		r.Noop(w.Name + " already in sync, nothing to do")
		return nil
	}
	var d []string
	if up != "" {
		d = append(d, "↑ "+up)
	}
	if down != "" {
		d = append(d, "↓ "+down)
	}
	r.OK(w.Name, strings.Join(append(d, home), " · "))
	return nil
}

var mimeExt = map[string]string{
	"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp", "image/gif": "gif",
	"image/heic": "heic", "image/heif": "heif", "image/avif": "avif", "image/bmp": "bmp",
	"image/x-ms-bmp": "bmp", "image/tiff": "tif", "image/svg+xml": "svg",
}

// ImageName is the last part of u's path, decoded, or "wallpaper".
func ImageName(u string) string {
	p, err := url.Parse(u)
	if err != nil {
		return "wallpaper"
	}
	name := path.Base(p.Path)
	if name == "" || name == "/" || name == "." || strings.HasPrefix(name, ".") {
		return "wallpaper"
	}
	return name
}

// AddWallpaper downloads u into the repo as name, or the name in the url with
// the extension of its type added when missing, then commits and pushes it.
// Anything that is not an image is refused.
func AddWallpaper(ctx context.Context, w Wall, u, name string, r Reporter) error {
	if name == "" {
		name = ImageName(u)
	}
	r.Step("downloading " + name)
	tmp, err := os.CreateTemp("", "dehv-wall-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	dctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(dctx, http.MethodGet, u, nil)
	if err != nil {
		return Fail("could not download it", err.Error())
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return Fail("could not download it", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return Fail("could not download it", fmt.Sprintf("the server answered %d", resp.StatusCode))
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return Fail("could not download it", err.Error())
	}
	tmp.Close()

	mime, _ := sh.Out("file", "-b", "--mime-type", tmp.Name())
	ext, ok := mimeExt[mime]
	if !ok {
		return Fail("it is not an image", "the server sent "+mime)
	}
	if !imageExt[strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))] {
		name += "." + ext
	}

	if _, err := readyWall(ctx, w, r); err != nil {
		return err
	}
	dir := w.Path()
	dst := filepath.Join(dir, name)
	if old, err := os.ReadFile(dst); err == nil {
		got, _ := os.ReadFile(tmp.Name())
		if string(old) == string(got) {
			r.Noop(name + " is already in " + w.Name + ", nothing to do")
			return nil
		}
		return Fail("there is already a "+name+" in "+w.Name, "name it: 'dehv wall <url> <name>'")
	}
	if err := moveFile(tmp.Name(), dst); err != nil {
		return err
	}
	_ = os.Chmod(dst, 0o644)

	st, _ := os.Stat(dst)
	detail := humanSize(st.Size())
	if dims := imageDims(dst); dims != "" {
		detail = dims + " · " + detail
	}
	r.OK(name, detail)

	if _, err := sh.Git(ctx, dir, "add", "--", name); err != nil {
		return err
	}
	if _, err := sh.Git(ctx, dir, "commit", "-q", "-m", "add "+name, "--", name); err != nil {
		return err
	}
	if _, _, err := publish(ctx, w, r); err != nil {
		return err
	}
	h, _ := sh.Git(ctx, dir, "rev-parse", "--short", "HEAD")
	r.OK(h+" add "+name, "→ "+w.Name)
	return nil
}

// moveFile renames, or copies when src is on another volume (the temp folder).
func moveFile(src, dst string) error {
	if os.Rename(src, dst) == nil {
		return nil
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

func imageDims(file string) string {
	out, _ := sh.Out("sips", "-g", "pixelWidth", "-g", "pixelHeight", file)
	var w, h string
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) == 2 && f[0] == "pixelWidth:" {
			w = f[1]
		} else if len(f) == 2 && f[0] == "pixelHeight:" {
			h = f[1]
		}
	}
	if w == "" || h == "" {
		return ""
	}
	return w + "×" + h
}

func humanSize(n int64) string {
	if n >= 1<<20 {
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	}
	return fmt.Sprintf("%d KB", (n+1023)/1024)
}
