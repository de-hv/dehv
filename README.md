# dehv

My own desktop toolkit for macOS: every personal command lives behind this one name, as `dehv <command> [options] [args]`, so none can clash with a system command. `dehv` alone, in a terminal, opens a hub with all of them. It is one Go binary, installed with Homebrew.

## Contents

- [Install](#install)
- [What it needs](#what-it-needs)
- [The hub](#the-hub)
- [Commands](#commands)
- [Config](#config)
- [In the repo](#in-the-repo)
- [Working on it](#working-on-it)
- [Releasing](#releasing)
- [Adding a command](#adding-a-command)

## Install

From my Homebrew tap:

```sh
brew install de-hv/tap/dehv
```

It is a cask with a prebuilt binary for Apple silicon and Intel: brew links `dehv` into its `bin` and the zsh completion into its `site-functions`, so nothing goes in `~/.zshrc`.

| Command | Does |
|---|---|
| `brew upgrade dehv` | update it |
| `brew uninstall dehv` | uninstall it; `~/.config/dehv` stays |

From 0.2.x (the zsh formula): `brew uninstall dehv && brew update && brew install de-hv/tap/dehv`. The files in `~/.config/dehv` keep working as they are.

To see the `dehv hi` card in every new terminal, add `dehv hi` to `~/.zshrc` by hand.

## What it needs

`dehv doctor` checks it, with exit code 1 when something required is missing.

| Needs | From |
|---|---|
| `git` | Apple's Command Line Tools (`xcode-select --install`) |
| `lsof`, `pbcopy`, `osascript`, `sips`, `networksetup` | macOS: they come with every Mac |
| `gh` (optional) | Homebrew (`brew install gh`). Only `dehv wall` uses it, to choose ssh or https for `owner/name` urls |

## The hub

`dehv` in a terminal opens a full-screen hub: the tools on the left, the one selected on the right, what just happened in the status line and the keys that matter in the footer.

```
 dehv  ~/dev/dehv · main · as personal                                          v1.0.0
──────────────────────────────────────────────────────────────────────────────────────
 1 Ports        3 │ Ports · 3 listening
 2 Branches     2 │   PORT  PROGRAM   PID   USER
 3 Profiles       │ ▌ 3000  node      4821  david
 4 Skills       4 │   5173  vite      4907  david
 5 Walls          │   5432  postgres  612   david
 6 Machine        │
 7 Tap            │
 8 Doctor       ✓ │
──────────────────────────────────────────────────────────────────────────────────────
 ✓ killed node (4821) on port 3000
 x kill ⋅ / filter ⋅ r refresh ⋅ ← tools ⋅ 1-8 jump ⋅ q quit ⋅ ? help
```

| Tool | Shows | Keys |
|---|---|---|
| 1 Ports | what listens on TCP, read again every 2 seconds | `x` kill, after asking |
| 2 Branches | this repo's branches, merged and gone ones ticked; it fetches with `--prune` once | `space` tick, `a` tick all, `d` delete the ticked ones, after asking, `f` fetch |
| 3 Profiles | my git profiles, `●` on the one this repo commits as | `enter` commit as it, `n` new, `e` edit, `a` amend the last commits, `d` remove |
| 4 Skills | my skills, a column per installed agent, `!` on those without description or broken | `enter` tick the agents to link it into, `e` edit its `SKILL.md` in `$EDITOR`, `n` new, `d` trash, `R` restore |
| 5 Walls | my wallpaper repos, `●` on the one in use | `s` sync, `S` sync all, `a` add an image by url, `o` open the folder, `u` use, `n` new, `e` edit, `d` forget |
| 6 Machine | the `dehv hi` card and this Mac's addresses | `c` copy the local IP, `p` ask the public one, `C` copy it |
| 7 Tap | the steps of `dehv tap` to release to my homebrew-tap, a kind at a time | `↑↓` choose a command, `c` copy it, `t` the next kind (`T` the previous) |
| 8 Doctor | `dehv doctor` and `dehv where` | `j k` scroll |

The focus is either on the tools or on the panel. `←` or `h` moves it to the tools: there `↑↓` or `j k` choose one, its panel following along, and `→`, `l` or `enter` give the focus back to it. In the row of tabs of a narrower terminal it is `←→` to choose and `↓` or `enter` to go back. The bar `▌` marks where the focus is, and panel keys only work with the focus on the panel.

Everywhere: `1`–`9` jump to the first nine tools, `tab` and `shift+tab` (or `]` and `[`) go to the next and previous, `↑↓` or `j k` move, `/` filters, `r` reads it all again, `m` shows the messages, `?` the keys, `ctrl+z` suspends, `esc` closes what is open and `q` quits. What removes or rewrites anything asks first, and only `y` goes ahead.

From 120 columns the tools are a sidebar; narrower, a row of tabs; under 80, only the one selected. Columns of less use fold away first. Under 50×12 it says the terminal is too small. It uses One Dark's colours, and every state has its mark too (`✓ ✗ ! ●`), so `NO_COLOR` loses nothing.

Piped, or with `dehv --plain` or `NO_TUI=1`, `dehv` prints the list of commands instead.

## Commands

Each command is a one-shot a script can call: results on stdout, what went wrong on stderr with exit code 1, a misuse with its usage line and exit code 2. A command that needs an argument shows its help when run without any, and `dehv <command> --help` always does.

```
$ dehv port -k 3000 5173
✓ killed node (4821) on port 3000
✗ nothing on port 5173 · dehv port -a to see what listens

$ dehv port -a
PORT  PROGRAM   PID
3000  node      4821
5432  postgres  612
2 listening

$ dehv as acme
✓ personal → acme · David Higuita <david@acme.com>
```

In a terminal the output has colours and pickers; piped, it is plain text, so `$(dehv ip)` and `cd "$(dehv where)"` stay usable. Pickers and prompts draw on stderr and need a terminal: without one, a command that would ask fails and says what to pass instead.

### port, ip, hi, branches, where, doctor, tap

| Command | Does |
|---|---|
| `dehv port <port>...` | what listens (TCP) on those ports, as `PORT`, `PROGRAM` and `PID` |
| `dehv port -a` | everything that listens, on any port (`--all`) |
| `dehv port -k <port>...` | kill whatever listens on those ports (`--kill`) |
| `dehv port -k` | the same, picked from the list of everything that listens |
| `dehv ip` | my local IP address: the one of the interface the default route uses (Wi-Fi, cable, dock) |
| `dehv ip -p` | my public IP address instead, asked to `api.ipify.org` (`--public`) |
| `dehv ip -c` | also copy it to the clipboard (`--copy`) |
| `dehv hi` | a small card: the date, this Mac's name and macOS, uptime, IP and battery |
| `dehv branches` | fetch with `--prune`, then tick which local branches to delete: merged into the default branch or gone (their remote deleted, as a squash-merged PR leaves them) come ticked. Never the default one nor the one checked out |
| `dehv branches -l` | only list them, with why each can go (`--list`) |
| `dehv where` | where dehv is installed: the command, its folder, where it comes from and the version; piped, only the folder |
| `dehv doctor` | check the command in the PATH, its completion, what dehv needs and its files; exit code 1 when something is wrong |
| `dehv tap [goreleaser\|cask\|formula]` | the steps to release a new version to my homebrew-tap, with generic names; asks which without one. Piped, a script whose notes are comments |
| `dehv completion zsh` | the zsh completion, for a dehv not installed with Homebrew |

### dehv as

| Command | Does |
|---|---|
| `dehv as <profile>` | set this repo's `user.name`/`user.email` to that profile's |
| `dehv as -s` | the same, picked from my profiles; directly when there is only one (`--set`) |
| `dehv as -n <profile>` | create or edit a profile: asks the name and the email, prefilled with its current ones or git's global identity (`--new`) |
| `dehv as -r <profile>` | remove a profile (`--remove`) |
| `dehv as -l` | list my profiles, with a green dot on the one this repo commits as (`--list`) |
| `dehv as -a [<n>]` | make the repo's profile, or the one named (`dehv as acme -a`), the author of the last `<n>` commits, 1 without it; rewritten, so only before pushing (`--amend`) |
| `dehv as -c` | show the identity this repo commits with and which profile it is (`--check`) |
| `dehv as -i` | create the profiles file; if there is one, keep it or start blank, keeping the old one as `profiles.bak` (`--init`) |
| `dehv as -x` | remove every profile, after asking (`--clear`) |

The profiles are this Mac's, in `~/.config/dehv/profiles` (`chmod 600`), in git's config format, so `git config -f` reads and edits them too:

```ini
[profile "personal"]
	name = David Higuita
	email = david@example.com
[profile "acme"]
	name = David Higuita
	email = david@acme.com
```

Names are lower-case letters, digits, `-` and `_`, never only digits (those are `<n>`).

### dehv skill

| Command | Does |
|---|---|
| `dehv skill <name>` | create `<skills>/<name>` with a `SKILL.md` that has its frontmatter (`name`, `description`) |
| `dehv skill -l <name>` | link it into the agents ticked in a list, creating it first if missing; unticking one takes it out (`--link`) |
| `dehv skill -u <name>` | take it out of the agents ticked, keeping the skill (`--unlink`) |
| `dehv skill -d <name>` | unlink it from every agent and move it to the Trash, through the Finder, so "Put Back" works too (`--delete`) |
| `dehv skill -r <name>` | bring it back from the Trash, then tick where to link it (`--restore`) |
| `dehv skill -t <agents> ...` | with `-l`, `-u` or `-r`: the agents by name instead of the list, like `claude,copilot`; alone it means `-l` (`--to`) |
| `dehv skill -a` | list my skills with their description and a column per installed agent (`--all`) |

The skills live in `~/Documents/dev/skills`, or the folder in `skill.dir` of the [config](#config). The agents, in `internal/core/skills.go`, and the first one, `claude`, is the default:

| Agent | Installed if | Skills go in |
|---|---|---|
| `claude` | `~/.claude` | `~/.claude/skills` |
| `cursor` | `~/Library/Application Support/Cursor/User` | `…/Cursor/User/skills` |
| `copilot` | `~/.copilot` | `~/.copilot/skills` |
| `xcode` | `~/Library/Developer/Xcode/CodingAssistant` | the same folder |
| `codex` | `~/.agents` | `~/.agents/skills` |

### dehv wall

`dehv wall` keeps my wallpapers in a GitHub repo, cloned into a folder, so adding one from the web or dropping one in the folder ends up in both.

| Command | Does |
|---|---|
| `dehv wall <url> [<name>]` | download the image into the repo in use, then commit and push it. Anything that is not an image is refused |
| `dehv wall -s` | commit what changed in the folder, rebase on GitHub's and push; the first time, only clone it (`--sync`). Removals, here or on GitHub, are asked first |
| `dehv wall -s -a` | the same for every repo; one that fails does not stop the rest (`--all`) |
| `dehv wall -o` | open the folder (`--open`) |
| `dehv wall -t <repo> ...` | with `<url>`, `-s` or `-o`: that repo just this once (`--to`) |
| `dehv wall -u [<repo>]` | the repo the rest use; picked from a list without one (`--use`) |
| `dehv wall -n <repo>` | add or edit a repo: asks its url (`owner/name` is enough) and its folder, `~/Pictures/<name>` by default (`--new`) |
| `dehv wall -r <repo>` | forget a repo; its folder stays (`--remove`) |
| `dehv wall -l` | list my repos with their folder and images (`--list`) |
| `dehv wall -i`, `dehv wall -x` | create the repos file, or forget every repo, as `dehv as -i` and `-x` do |

The repos are this Mac's, in `~/.config/dehv/walls`, in git's config format:

```ini
[wall]
	use = personal
[repo "personal"]
	url = git@github.com:de-hv/wallpapers.git
	dir = ~/Pictures/wallpapers
```

## Config

`~/.config/dehv/config`, in git's config format. Every key is optional:

```ini
[skill]
	dir = ~/Documents/dev/skills   # where dehv skill keeps the skills
[tap]
	owner = de-hv                  # the GitHub owner of my homebrew-tap, for dehv tap
```

`git config -f ~/.config/dehv/config skill.dir ~/somewhere` sets one.

## In the repo

```
dehv/
├── cmd/dehv/            main: the version, and the hub plugged into the commands
├── internal/core/       what dehv does, without how it looks: ports, branches, profiles, skills, walls, doctor…
├── internal/cli/        the commands, one file each, on Cobra and Fang
├── internal/tui/        the hub, on Bubble Tea: the app, the table, the dialogs and a file per panel
├── internal/ui/         one-shot output: One Dark's tokens, marks, tables, pickers, the spinner
├── internal/config/     ~/.config/dehv, through git config -f
├── internal/sh/         running git, lsof and the rest
├── .goreleaser.yaml     the release: binaries, archives, the cask
└── .github/workflows/   the release on every v* tag
```

## Working on it

Go 1.26 or newer (`brew install go`).

| Command | Does |
|---|---|
| `go run ./cmd/dehv` | run the hub from the clone |
| `go run ./cmd/dehv port -a` | run a command |
| `go test ./...` | the tests: core on a temp home, the commands piped, the hub's frames at 120, 80, 60 and 49 columns |
| `go test ./internal/tui -update` | write the hub's golden frames again, after changing how it looks |
| `go install ./cmd/dehv` | put it in `~/go/bin`; `dehv where` says it is a build from source |

## Releasing

A tag is a release: the workflow builds dehv for both Macs, creates the GitHub release with the CHANGELOG's notes for that version and commits the updated cask to `de-hv/homebrew-tap`. It needs the `HOMEBREW_TAP_TOKEN` secret, a token that can push to the tap.

```sh
git commit -am "dehv 1.0.1" && git push   # with the CHANGELOG's ## 1.0.1
git tag v1.0.1 && git push origin v1.0.1
```

`goreleaser release --snapshot --clean` tries it locally, into `dist/`. `dehv tap goreleaser` prints these steps.

## Adding a command

1. Its logic in `internal/core/<command>.go`, with tests, returning `core.Fail(what, hint)` for what it could not do.
2. The command in `internal/cli/<command>.go`: a `*cobra.Command` with a `GroupID`, added in `Root()`. Misuses return `misuse(cmd, "…")`.
3. Its panel, if it has one, in `internal/tui/panel_<command>.go`, added to `newModel`: a `table` for lists, a `scroller` for text.
