# dehv

My own command line tool for macOS: every personal function lives behind this one name, as `dehv <command> [options] [args]`, so none can clash with a system command. It is plain zsh: a plugin file, a library of helpers and one file per command, loaded into the shell by one line in the rc file.

## Contents

- [Install](#install)
- [What it needs](#what-it-needs)
- [Update, move and uninstall](#update-move-and-uninstall)
- [Where things are](#where-things-are)
- [Commands](#commands)
- [Config](#config)
- [In the repo](#in-the-repo)
- [Adding a command](#adding-a-command)

## Install

Clone it wherever you like, and run the installer from the clone:

```sh
git clone <this repo> ~/code/dehv
~/code/dehv/install.sh
```

Nothing depends on that folder. The installer does two things:

1. **A link**, `~/.local/share/dehv`, to the folder the installer is in: the clone, wherever it is.
2. **A block in the rc file**, `~/.zshrc` (created if there is none), that loads dehv through that link:

   ```zsh
   # >>> dehv >>>
   # Loads dehv through its link; the installer in its clone writes and removes this.
   [ -f "$HOME/.local/share/dehv/dehv.plugin.zsh" ] && source "$HOME/.local/share/dehv/dehv.plugin.zsh"
   # <<< dehv <<<
   ```

   zsh only runs what its startup files tell it to, so a function like `dehv` always needs a line there. It names the link and never the clone, so moving the clone only changes the link. The two marker lines are how update and uninstall find the block again without touching anything else in the file.

Then it asks whether to show the `dehv hi` card in the first terminal of the day, and checks that a new zsh loads dehv. `exec zsh` loads it in the terminal you are in.

Run without arguments, `install.sh` shows what is installed and a menu with what can be done from that state:

```
 ❯❯❯ dehv  installer · v0.1.0

   status   ● installed
   link     ~/.local/share/dehv → ~/code/dehv  this folder
   loads    ~/.zshrc
   needs    ✓ all there
   remote   3 new commits

 ❯ Update     pull the 3 new commits
   Reinstall  write the link and the block again
   Uninstall  remove the link and the block
   Quit

 ↑↓ move · enter choose · q quit
```

| State | Menu |
|---|---|
| not installed | Install |
| installed from this folder | Update, Reinstall, Uninstall |
| installed from another clone | Relink (to this one), Update (that one), Uninstall |
| half installed (link or block missing, or the clone gone) | Repair, Uninstall |

The same, without the menu:

| Command | Does |
|---|---|
| `./install.sh install` | link `~/.local/share/dehv` to this folder and write the block |
| `./install.sh update` | `git pull --ff-only` in the clone the link points to, listing the commits it brought |
| `./install.sh uninstall` | remove the link and the block, after a `[y/N]`; then asks whether to delete `~/.config/dehv` and `~/.cache/dehv` too (no by default). The clone stays |
| `./install.sh relink` | point the link to this folder |
| `./install.sh status` | what is installed; exit code 1 when it is not |
| `-y`, `--yes` | no questions: yes to what is required, skip what is optional |
| `--rc <file>` | write the block in that file instead of `~/.zshrc` |
| `--greeting`, `--no-greeting` | show the `dehv hi` card in the first terminal of the day, or not |

When the output is not a terminal there is no menu: `./install.sh` alone is `status`.

**When `~/.zshrc` is a link**, as a dotfiles repo leaves it, writing there would change that repo. The installer says where it points and asks where to load dehv from instead: `~/.zshrc.local` when there is one, `~/.zshrc` anyway, another file, or only print the lines to add by hand. With `--yes` it takes `~/.zshrc.local`, or stops and asks for `--rc`. The file is always written through, so a link there stays a link.

## What it needs

Checked before anything is touched; nothing is installed without asking.

| Needs | From | When it is missing |
|---|---|---|
| zsh 5.8 or newer | macOS | stops |
| `git` | Apple's Command Line Tools | offers `xcode-select --install` and waits for the macOS dialog to finish |
| `curl`, `lsof`, `pbcopy`, `osascript`, `sips`, `networksetup` | macOS | stops: they come with every Mac |
| `gh` (optional) | Homebrew | offers `brew install gh`, or to skip it; without Homebrew it shows https://cli.github.com and goes on. Only `dehv wall` uses it, to choose ssh or https for `owner/name` urls |

`/usr/bin/git` exists on every Mac, but until the Command Line Tools are installed it only offers to install them, so git counts once `xcode-select -p` answers.

## Update, move and uninstall

- **Update**: `install.sh` in the menu, or `./install.sh update`.
- **Move the clone**: move it, then run `install.sh` from its new folder and pick Repair (or `./install.sh relink`). The rc file does not change.
- **Two clones**: the link decides which one loads. `install.sh` from the other one says so and offers Relink.
- **Uninstall**: `install.sh` and Uninstall, or `./install.sh uninstall`. Open terminals keep `dehv` until `exec zsh`.

## Where things are

| Path | What |
|---|---|
| the clone | the code, wherever it was cloned |
| `~/.local/share/dehv` | the link to the clone (`$XDG_DATA_HOME/dehv` when set) |
| `~/.zshrc`, or the file chosen | the block that loads dehv |
| `~/.config/dehv/config` | settings: the greeting, the skills folder, the rc file in use (see [Config](#config)) |
| `~/.config/dehv/profiles` | my git profiles for `dehv as` (`chmod 600`) |
| `~/.config/dehv/walls` | my wallpaper repos for `dehv wall` (`chmod 600`) |
| `~/.cache/dehv/hi` | the day `dehv hi -d` last showed its card |

`dehv where` shows the first three:

```
$ dehv where
╭─ dehv where ─────────────────────────────╮
│ link       ~/.local/share/dehv           │
│ points to  ~/code/dehv                   │
│ loaded by  ~/.zshrc                      │
│ version    0.1.0 · main · a1c9e2f        │
╰──────────────────────────────────────────╯

$ cd "$(dehv where)"      # piped, only the clone's path
```

`dehv doctor` checks all of it: the link, the block, the completion, what dehv needs and the permissions of its files, with exit code 1 when something is wrong. It only looks; `install.sh` repairs.

## Commands

![dehv help](docs/screenshots/help.png)

It is shaped like a regular CLI: `dehv` alone prints a short usage, a command that needs an argument prints its own when run without any (`dehv help <command>` does for all of them; commands take no `-h`), options are flags (short and long), and a misuse prints the error and the command's usage line to stderr with exit code 1.

```
$ dehv
╭─ dehv ────────────────────────────────────────────────────────────────────────────────────────╮
│ my own desktop toolkit                                                                        │
│                                                                                               │
│ as        <profile>               commit as that git profile in this repo                     │
│           -s, --set               the same, picking the profile from a list                   │
│           -a, --amend [<n>]       make it the author of the last <n> commits                  │
│           -n, --new <profile>     create or edit a profile: asks name and email               │
│           -r, --remove <profile>  remove a profile                                            │
│           -l, --list              list my profiles                                            │
│           -c, --check             show which profile this repo commits as                     │
│           -i, --init              create the profiles file, or keep or blank it               │
│           -x, --clear             remove every profile, after asking                          │
│                                                                                               │
│ branches                          tick which merged or gone branches to delete                │
│           -l, --list              only list them, with why each can go                        │
│                                                                                               │
│ doctor                            check the link, the rc file, what dehv needs and its files  │
│                                                                                               │
│ hi                                a small card: the date, this Mac, its network and battery   │
│           -d, --daily             the same, only once a day (how the greeting shows it)       │
│                                                                                               │
│ ip                                print my local IP address                                   │
│           -p, --public            print my public one instead                                 │
│           -c, --copy              also copy it to the clipboard                               │
│                                                                                               │
│ port      <port>...               show what listens on those ports                            │
│           -a, --all               show everything that listens                                │
│           -k, --kill [<port>...]  kill what listens there, or pick from a list                │
│                                                                                               │
│ skill     <name>                  create a skill                                              │
│           -l, --link <name>       link it into the agents I pick, creating it if missing      │
│           -u, --unlink <name>     take it out of the agents I pick, keeping it                │
│           -d, --delete <name>     unlink it everywhere and move it to the Trash               │
│           -r, --restore <name>    bring it back from the Trash, and pick where to link it     │
│           -t, --to <agents>       name the agents instead of picking: claude,cursor           │
│           -a, --all               list them, and the agents each is linked into               │
│                                                                                               │
│ wall      <url> [<name>]          add the image to the repo in use: download, commit, push    │
│           -t, --to <repo>         use that repo just this once, with <url>, -s or -o          │
│           -s, --sync              commit what changed in its folder, pull and push            │
│           -s -a, --all            the same for every repo                                     │
│           -u, --use [<repo>]      choose the repo the rest use, from a list without one       │
│           -n, --new <repo>        add or edit a repo: asks its GitHub url and folder          │
│           -r, --remove <repo>     forget a repo, keeping its folder                           │
│           -l, --list              list my repos                                               │
│           -o, --open              open the folder of the repo in use                          │
│           -i, --init              create the repos file, or keep or blank it                  │
│           -x, --clear             forget every repo, after asking                             │
│                                                                                               │
│ where                             the link, the clone it points to and the file that loads it │
╰───────────────────────────────────────────────────────────────────────── dehv help <command> ─╯
```

```
$ dehv port
╭─ dehv port ──────────────────────────────────────────────────────────────────╮
│ Inspect or free ports: what listens (TCP) on them, as PORT, PROGRAM and PID. │
│                                                                              │
│ <port>...             show what listens on those ports                       │
│ -a, --all             show everything that listens, on any port              │
│ -k, --kill <port>...  kill what listens on those ports                       │
│ -k, --kill            pick what to kill from everything that listens         │
╰──────────────────────────────────────────────────────────────────────────────╯
```

The card lists every command with each way to call it: commands in cyan and arguments in yellow, in the terminal's own ANSI colours, so they follow its palette. When the output is not a terminal (`dehv -h | less`, a file) the same rows come as plain text without the frame, and with `NO_COLOR=1` without colours. Each command's help is a card of its own, and piped it starts with the usage line a misuse repeats.

What a command did comes after a green `✓`, what it could not after a red `✗` (on stderr, with exit code 1) and, dimmed after a `·`, that there was nothing to do:

```
$ dehv port -k 3000 5173
✓ killed node (4821) on port 3000
✗ nothing on port 5173

$ dehv skill notes -t claude
✓ created  ~/Documents/dev/skills/notes
✓ linked   ~/.claude/skills/notes

$ dehv as -n acme
  name  David Higuita
  email david@acme.com
✓ created acme · David Higuita <david@acme.com>

$ dehv as acme
✓ personal → acme · David Higuita <david@acme.com>

$ dehv as -a 2
✓ a1b2c3d fix login redirect
✓ 9f8e7d6 add rate limit
2 commits now as acme
```

In a terminal `dehv port` frames its list and `dehv ip` adds the interface, a spinner while it asks for the public address and whether it copied it; piped, they print the bare table and the bare address, so `$(dehv ip)` stays usable.

```
$ dehv port -a
╭─ ports ───────────────────╮
│ PORT   PROGRAM       PID  │
│ 3000   node          4821 │
│ 5432   postgres      612  │
╰───────────── 2 listening ─╯

$ dehv ip -c
192.168.0.23   en0 · Wi-Fi   ✓ copied

$ dehv skill -a
NAME   claude  copilot  DESCRIPTION
notes  ●       ●        Take meeting notes in my format
plan   ○       ○        ! no description
old    ●       ○        ! broken link
● linked  ○ not linked
```

### dehv, port, ip, branches, hi, where, doctor

| Command | Does |
|---|---|
| `dehv`, `dehv -h`, `dehv help` | the usage and the list of commands |
| `dehv help <command>` | that command's usage and options |
| `dehv port`, `dehv as`, `dehv skill`, `dehv wall` | the same: they need an argument, so alone they print their usage |
| `dehv port <port>...` | what listens (TCP) on those ports, in columns `PORT`, `PROGRAM` and `PID` |
| `dehv port -a` | everything that listens, on any port (`--all`) |
| `dehv port -k <port>...` | kill whatever listens on those ports (`--kill`) |
| `dehv port -k` | the same, picked with the arrows from the list of everything that listens |
| `dehv ip` | my local IP address: the one of the interface the default route uses (Wi-Fi, cable, dock) |
| `dehv ip -p` | my public IP address instead, asked to `api.ipify.org` (`--public`) |
| `dehv ip -c` | also copy it to the clipboard (`--copy`) |
| `dehv branches` | fetch with `--prune`, then tick which local branches to delete: those merged into the default branch, or gone (their remote deleted, as a squash-merged PR leaves them), come ticked; unmerged ones are listed unticked. Never the default one nor the one checked out |
| `dehv branches -l` | only list them, with why each can go (`--list`) |
| `dehv hi` | a small card: the date, this Mac's name and macOS, uptime, IP and battery |
| `dehv hi -d` | the same, only if not shown yet today (`--daily`); the greeting runs it, so it shows in the first terminal of the day |
| `dehv where` | where dehv is installed: the link, the clone it points to, the rc file that loads it and the version; piped, only the clone's path |
| `dehv doctor` | check the link, the block, the completion, what dehv needs and its files; exit code 1 when something is wrong |

### dehv as

| Command | Does |
|---|---|
| `dehv as <profile>` | set this repo's `user.name`/`user.email` to that profile's |
| `dehv as -s` | the same, picked with the arrows from my profiles; directly when there is only one (`--set`) |
| `dehv as -n <profile>` | create or edit a profile: asks the name and the email, prefilled with its current ones or, for a new one, git's global identity (`--new`) |
| `dehv as -r <profile>` | remove a profile (`--remove`) |
| `dehv as -l` | list my profiles, with a green dot on the one this repo commits as (`--list`) |
| `dehv as -a` | amend the last commit with the repo's profile as author, or the one named (`dehv as acme -a`) or picked (`--amend`) |
| `dehv as -a <n>` | the same for the last `<n>` commits, with a rebase: only before pushing them |
| `dehv as -c` | show the identity this repo commits with and which profile it is, if any (`--check`) |
| `dehv as -i` | create the profiles file; if there is one, say what it holds and choose to use it or start blank, keeping the old one as `profiles.bak` (`--init`) |
| `dehv as -x` | remove every profile, after a `[y/N]` (`--clear`) |

The profiles are this Mac's, in `~/.config/dehv/profiles` (`chmod 600`), written in git's config format, so `git config -f` reads and edits them too:

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
| `dehv skill -l <name>` | link it into the agents ticked in a list, creating it first if missing; the list starts with the agents it is already in, so unticking one takes it out (`--link`) |
| `dehv skill -u <name>` | take it out of the agents ticked, keeping the skill; no list when it is in just one (`--unlink`) |
| `dehv skill -d <name>` | unlink it from every agent and move it to the Trash, through the Finder, so "Put Back" works too (`--delete`) |
| `dehv skill -r <name>` | bring it back from the Trash, then tick where to link it, or nowhere (`--restore`) |
| `dehv skill -t <agents> ...` | with `-l`, `-u` or `-r`: the agents by name instead of the list, like `claude,copilot`; alone it means `-l` (`--to`) |
| `dehv skill -a` | list my skills with their description and a column per installed agent: a green dot where it is linked, a warning on those without description (no agent can trigger them) and on broken links (`--all`) |

The skills live in `~/Documents/dev/skills`, or the folder in `skill.dir` of the [config](#config). The agents are the table at the top of `commands/skill.zsh`: a name, the folder that tells the agent is installed and where its skills go. Only the installed ones are listed, and the first one, `claude`, is the default: the one ticked in the list, and the one used when there is no terminal to show it in.

| Agent | Installed if | Skills go in |
|---|---|---|
| `claude` | `~/.claude` | `~/.claude/skills` |
| `cursor` | `~/Library/Application Support/Cursor/User` | `~/Library/Application Support/Cursor/User/skills` |
| `copilot` | `~/.copilot` | `~/.copilot/skills` |
| `xcode` | `~/Library/Developer/Xcode/CodingAssistant` | the same folder |
| `codex` | `~/.agents` | `~/.agents/skills` |

```
$ dehv skill notes -l
Link notes into:
› [x] claude   ~/.claude/skills  default
  [ ] copilot  ~/.copilot/skills
  ↑↓ move · space tick · enter confirm · esc cancel
✓ created  ~/Documents/dev/skills/notes
✓ linked   ~/.claude/skills/notes

$ dehv skill notes -t copilot,cursor
✗ cannot link into cursor · it is not installed
✓ linked   ~/.copilot/skills/notes
```

### dehv wall

`dehv wall` keeps my wallpapers in a GitHub repo, cloned into a folder, so adding one from the web or dropping one in the folder ends up in both. It only needs `git` and `curl`.

| Command | Does |
|---|---|
| `dehv wall <url>` | download the image into the folder of the repo in use, then commit and push it. The name is the last part of the url, with the extension of its type added when missing (`?w=3840`-style urls); anything that is not an image is refused |
| `dehv wall <url> <name>` | the same, saved as `<name>` |
| `dehv wall -s` | commit what changed in the folder (images added, removed or renamed by hand), rebase on GitHub's and push; the first time, when the folder is not there or is empty, only clone it (`--sync`). An image changed both here and on GitHub stops it, with the local commit kept |
| `dehv wall -s -a` | the same for every repo, one after the other; one that fails does not stop the rest (`--all`) |
| `dehv wall -o` | open the folder (`--open`) |
| `dehv wall -t <repo> ...` | with `<url>`, `-s` or `-o`: that repo just this once (`--to`) |
| `dehv wall -u [<repo>]` | the repo the rest use; without one, picked with the arrows (`--use`). With several repos and none in use (the one in use was removed), `<url>`, `-s` and `-o` show that list first, and the one picked stays in use |
| `dehv wall -n <repo>` | add or edit a repo: asks its url, where `owner/name` is enough for GitHub, and its folder, `~/Pictures/<name>` by default. The first one becomes the one in use (`--new`) |
| `dehv wall -r <repo>` | forget a repo; its folder stays (`--remove`) |
| `dehv wall -l` | list my repos with their folder and images, a green dot on the one in use (`--list`) |
| `dehv wall -i`, `dehv wall -x` | create the repos file, or forget every repo, as `dehv as -i` and `-x` do (`--init`, `--clear`) |

The repos are this Mac's, in `~/.config/dehv/walls`, in git's config format as the profiles:

```ini
[wall]
	use = personal
[repo "personal"]
	url = git@github.com:de-hv/wallpapers.git
	dir = ~/Pictures/wallpapers
```

The commits are `add <name>` for `dehv wall <url>` and `sync: add 2, remove 1` for `-s`, as the repo's git identity (`dehv as` sets it). The Finder's `.DS_Store` never goes in: it is in the clone's `.git/info/exclude`.

```
$ dehv wall https://images.example.com/aurora.jpg
✓ aurora.jpg · 3840×2160 · 2.4 MB
✓ a3f9c21 add aurora.jpg → personal

$ dehv wall -s
✓ personal · ↑ 2 added, 1 removed · ↓ 3 added · ~/Pictures/wallpapers
```

### Examples

```sh
dehv port -k 3000                                 # free the port a dev server left behind
dehv branches                                     # clean up after merged PRs
dehv as acme -s -a                                # commit as acme here and fix the last commit
dehv skill notes -l                               # start a skill and tick the agents that see it
dehv wall -s                                      # push the wallpapers I dropped in the folder
```

Tab completion works at every level: `dehv <tab>` lists the commands and `dehv <command> -<tab>` its options with what they do. `dehv skill <tab>` offers the names of my skills and `-t <tab>` the agents; `dehv wall -t <tab>`, `-u`, `-n` and `-r` my wallpaper repos. It registers itself whether the block is read before or after `compinit`.

## Config

`~/.config/dehv/config`, in git's config format like the profiles. Every key is optional:

```ini
[hi]
	daily = true                   # the dehv hi card in the first terminal of the day
[skill]
	dir = ~/Documents/dev/skills   # where dehv skill keeps the skills
[install]
	rc = ~/.zshrc                  # the file with the block; install.sh writes it
```

`install.sh` writes `hi.daily` and `install.rc`; `git config -f ~/.config/dehv/config skill.dir ~/somewhere` sets the rest.

## In the repo

```
dehv/
├── dehv.plugin.zsh      the entry point: loads the rest, dispatches, help, completion, greeting
├── lib.zsh              helpers every command shares (cards, menus, output, config, install paths)
├── commands/*.zsh       one file per command (as, branches, doctor, hi, ip, port, skill, wall, where)
├── completions/_dehv    the completion of dehv, found by compinit through $fpath
├── install.sh           install, update, uninstall, relink and status, from wherever the clone is
├── VERSION
└── docs/screenshots/
```

## Adding a command

A new command is one new file, `commands/<command>.zsh`: `dehv.plugin.zsh` loads every file there, in alphabetical order (also their order in `dehv help`), so nothing else changes. The file:

1. registers the command: `__dehv_commands+=(<command>)`, what it is for `dehv <tab>` in `__dehv_about[<command>]`, and its rows for `dehv help` in `__dehv_rows`, one per way to call it, as `command|arguments|what it does` (the command only on the first),
2. defines `__dehv.<command>` and its help in `__dehv.<command>.help`, a call to `__dehv.usage` with the usage line errors repeat, what the command is for and a row per way to call it,
3. defines its completion, `_dehv.<command>`.

`port.zsh` is the shortest one to copy from. The helpers it can use (`__dehv.ok`, `__dehv.fail`, `__dehv.box`, `__dehv.pick`…) are in `lib.zsh`.
