#!/usr/bin/env zsh
[ -z "$ZSH_VERSION" ] && exec zsh "$0" "$@"
# ┌──────────────────────────────────────────────────────────────────────────────┐
#                                 DEHV INSTALLER
# └──────────────────────────────────────────────────────────────────────────────┘
# Installs, updates and uninstalls dehv from this clone, wherever it is:
#
#   ./install.sh                     a menu with what can be done from here
#   ./install.sh <action> [options]  the same without the menu
#
# Installing is two things, so moving the clone only takes running this again
# from its new folder:
#
#   1. a link, ~/.local/share/dehv, to the folder this script is in,
#   2. a block in the rc file (~/.zshrc) that sources dehv.plugin.zsh through
#      that link.
#
# What dehv needs is checked first, and nothing is installed without asking;
# --yes says yes to what is required and skips what is optional. The block is
# found again by its two marker lines, so update and uninstall touch nothing
# else in the file, and a link there (~/.zshrc kept by a dotfiles repo) is
# written through, never replaced.

here=${0:A:h}
source "$here/lib.zsh"
[[ "${LC_ALL:-${LC_CTYPE:-$LANG}}" == *UTF-8* ]] || export LC_ALL=en_US.UTF-8

link=$(__dehv.link)
version=$(__dehv.version "$here")
tty=false
[ -t 0 ] && [ -t 1 ] && tty=true

local b c d g y x r; __dehv.style
local w= sel=
# One Dark's foreground and selection, as __dehv.palette picks its colours.
if [ -n "$r" ]; then
  if [[ "$COLORTERM" == (truecolor|24bit) ]]; then
    w=$'\e[38;2;171;178;191m' sel=$'\e[48;2;62;68;81m'
  else
    w=$'\e[38;5;249m' sel=$'\e[48;5;238m'
  fi
fi

# ─── usage ─────────────────────────────────────────────────────────────────────
usage() {
  print -r -- "Usage: ./install.sh [install | update | uninstall | relink | status] [options]

Without an action, a menu with the ones that apply here.

Actions:
  install      link ~/.local/share/dehv to this folder and load it from the rc file
  update       git pull in the clone the link points to
  uninstall    remove the link and the block; ~/.config/dehv and the clone stay
  relink       point the link to this folder
  status       show what is installed, without changing anything

Options:
  -y, --yes            no questions: yes to what is required, skip what is optional
  --rc <file>          load dehv from this file instead of ~/.zshrc
  --greeting           show the dehv hi card in the first terminal of the day
  --no-greeting        do not show it
  -h, --help           this help"
}

misuse() {
  print -r -- "install.sh: $1" >&2
  print -r -- "Run './install.sh -h' for usage." >&2
  exit 1
}

action= yes=false rc_opt= greeting=
while [ $# -gt 0 ]; do
  case "$1" in
    install|update|uninstall|relink|status)
      [ -z "$action" ] || misuse "one action at a time"
      action=$1 ;;
    -y|--yes)      yes=true ;;
    --rc)          [ $# -ge 2 ] || misuse "--rc needs a file"
                   shift; rc_opt=${1/#\~/$HOME} ;;
    --greeting)    greeting=true ;;
    --no-greeting) greeting=false ;;
    -h|--help)     usage; exit 0 ;;
    *)             misuse "unknown argument '$1'" ;;
  esac
  shift
done

# ─── output ────────────────────────────────────────────────────────────────────
# A step done, as the menu's rows: a mark, a label and, dimmed, the detail.
step() {
  local mark
  case "$1" in
    ok)   mark="${g}✓${r}" ;;
    fail) mark="${x}✗${r}" ;;
    warn) mark="${y}!${r}" ;;
    *)    mark="${d}·${r}" ;;
  esac
  print -r -- "   $mark ${(r:10:)2} ${d}$3${r}"
  [ "$1" != fail ]
}

# The paths as I read them, with ~.
home() {
  print -r -- "${1/#$HOME/~}"
}

# Runs the rest with a spinner and $1 as what it does, its output into $REPLY.
# $2 the seconds to give it, 0 for no limit; past them it fails with 124.
spin() {
  setopt localoptions localtraps nomonitor
  local msg="$1" limit="$2"; shift 2
  local out=$(mktemp) i=0 frames='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏' s stop=false
  GIT_TERMINAL_PROMPT=0 "$@" < /dev/null > "$out" 2>&1 &
  local pid=$!
  trap 'stop=true; kill $pid 2>/dev/null' INT
  while kill -0 $pid 2>/dev/null; do
    [ "$tty" = true ] && print -n -- "\r   ${c}${frames[i % 10 + 1]}${r} ${d}${msg}…${r}\e[K"
    (( i++ ))
    if (( limit > 0 && i > limit * 12 )); then
      kill $pid 2>/dev/null
      stop=timeout
    fi
    sleep 0.08
  done
  wait $pid; s=$?
  [ "$tty" = true ] && print -n -- $'\r\e[K'
  REPLY=$(<"$out")
  rm -f "$out"
  [ "$stop" = true ] && return 130
  [ "$stop" = timeout ] && return 124
  return $s
}

# ─── menu ──────────────────────────────────────────────────────────────────────
# The rows given as label|what it does, picked with the arrows; the row on top
# is the one chosen to begin with, and $1 the one to start on otherwise. Sets
# $REPLY to the label chosen, or returns 1 on esc/q. Wiped after, leaving only
# the choice.
menu() {
  local cur=$1; shift
  local -a rows=("$@") f
  local n=${#rows} i key rest lw=0 dw=0 row

  for row in $rows; do
    f=("${(@s:|:)row}")
    (( ${#f[1]} > lw )) && lw=${#f[1]}
    (( ${#f[2]} > dw )) && dw=${#f[2]}
  done
  (( lw += 2 ))

  print -n -- $'\e[?25l'
  while true; do
    for i in {1..$n}; do
      f=("${(@s:|:)rows[i]}")
      if (( i == cur )); then
        print -r -- " ${sel}${c}${b}❯ ${(r:lw:)f[1]}${r}${sel}${(r:dw:)f[2]}  ${r}"
      else
        print -r -- "   ${(r:lw:)f[1]}${d}${f[2]}${r}"
      fi
    done
    print -r -- ""
    print -r -- " ${d}↑↓ move · enter choose · q quit${r}"

    # No more input (a closed stdin) is quitting.
    read -sk1 key || key=q
    case "$key" in
      $'\e')
        rest=
        read -sk2 -t 0.05 rest
        case "$rest" in
          '[A') (( cur > 1 )) && (( cur-- )) ;;
          '[B') (( cur < n )) && (( cur++ )) ;;
          '')   cur=0 ;;
        esac
        ;;
      k) (( cur > 1 )) && (( cur-- )) ;;
      j) (( cur < n )) && (( cur++ )) ;;
      q) cur=0 ;;
      $'\n'|$'\r') key=enter ;;
    esac

    print -n -- $'\e['$(( n + 2 ))$'A\e[J'
    [ "$key" = enter ] || (( cur == 0 )) && break
  done
  print -n -- $'\e[?25h'

  (( cur > 0 )) || return 1
  REPLY=${rows[cur]%%|*}
  print -r -- " ${c}❯${r} ${b}${REPLY}${r}"
}

# A question above a menu, kept above the choice once it is made.
ask() {
  print -r -- ""
  print -r -- "   ${c}?${r} $1"
  shift
  menu 1 "$@"
}

[ "$tty" = true ] && trap 'print -n -- $'"'"'\e[?25h'"'"'; print; exit 130' INT

# ─── state ─────────────────────────────────────────────────────────────────────
# What is installed: $target, the folder the link points to (empty without
# one), $rc_now, the rc file with the block (empty without one), and $state:
#   none       neither
#   here       both, the link to this folder
#   elsewhere  both, the link to another clone
#   partial    only one of them, or the link to a folder that is gone
read_state() {
  target= rc_now=
  # The link's own target, as is: :A cannot resolve one that is gone.
  if [ -L "$link" ]; then
    target=$(readlink "$link")
    [[ "$target" == /* ]] || target=${link:h}/$target
    [ -e "$target" ] && target=${target:A}
  fi
  rc_now=$(__dehv.rc)

  if [ -z "$target" ] && [ -z "$rc_now" ]; then
    state=none
  elif [ -z "$target" ] || [ -z "$rc_now" ] || [ ! -d "$target" ]; then
    state=partial
  elif [ "$target" = "$here" ]; then
    state=here
  else
    state=elsewhere
  fi
}

# The rc file install would write when none has the block yet: $rc_opt, or
# ~/.zshrc. Sets $rc_plan and, in $rc_note, what would happen to it.
plan_rc() {
  rc_plan=${rc_opt:-${ZDOTDIR:-$HOME}/.zshrc}
  if [ -L "$rc_plan" ] && [ -z "$rc_opt" ]; then
    rc_note="a link to $(home "${rc_plan:A}"): asked when installing"
  elif [ -e "$rc_plan" ]; then
    rc_note="the block goes at its end"
  else
    rc_note="does not exist yet, to create"
  fi
}

# What dehv needs and is missing, in $missing (required) and $optional.
read_needs() {
  missing=() optional=()
  local row
  local -a f
  for row in $__dehv_needs; do
    f=("${(@s:|:)row}")
    __dehv.has "$f[1]" && continue
    [ "$f[3]" = optional ] && optional+=("$f[1]") || missing+=("$f[1]")
  done
}

# Whether the clone at $1 has commits to pull: sets $remote to a few words.
read_remote() {
  local clone=$1 s
  remote= behind=0
  git -C "$clone" rev-parse --abbrev-ref '@{u}' >/dev/null 2>&1 || { remote="none · a local clone"; return; }
  spin "asking the remote" 6 git -C "$clone" fetch -q; s=$?
  behind=$(git -C "$clone" rev-list --count 'HEAD..@{u}' 2>/dev/null || echo 0)
  if (( s != 0 )); then
    remote="${y}could not reach it${r}"
  elif (( behind == 0 )); then
    remote="up to date"
  else
    remote="${c}${behind} new commit$( (( behind == 1 )) || print s)${r}"
  fi
}

# ─── header ────────────────────────────────────────────────────────────────────
field() {
  print -r -- "   ${d}${(r:8:)1}${r} $2"
}

header() {
  print -r -- ""
  print -r -- " ${x}${b}❯${r}${w}${b}❯${r}${g}${b}❯${r} ${b}dehv${r}  ${d}installer${version:+ · v$version}${r}"
  print -r -- ""

  case $state in
    none)      field status "${y}○${r} not installed" ;;
    here)      field status "${g}●${r} installed" ;;
    elsewhere) field status "${y}●${r} installed from another folder" ;;
    partial)   field status "${y}!${r} half installed" ;;
  esac

  local to=$(home "$link")
  case $state in
    none)
      field link "$to ${d}→${r} ${y}$(home "$here")${r}  ${d}this folder, to create${r}" ;;
    here)
      field link "$to ${d}→${r} ${y}$(home "$here")${r}  ${g}this folder${r}" ;;
    elsewhere)
      field link "$to ${d}→${r} ${x}$(home "$target")${r}"
      field this "${y}$(home "$here")${r}" ;;
    partial)
      if [ -z "$target" ]; then
        field link "$to ${y}missing${r}"
      elif [ ! -d "$target" ]; then
        field link "$to ${d}→${r} ${x}$(home "$target")${r}  ${y}gone${r}"
      else
        field link "$to ${d}→${r} ${y}$(home "$target")${r}"
      fi
      [ "$target" != "$here" ] && field this "${y}$(home "$here")${r}" ;;
  esac

  if [ -n "$rc_now" ]; then
    field loads "$(home "$rc_now")"
  elif [ "$state" = partial ]; then
    field loads "${y}no rc file has the dehv block${r}"
  else
    plan_rc
    field loads "$(home "$rc_plan")  ${d}${rc_note}${r}"
  fi

  read_needs
  autoload -Uz is-at-least
  if ! is-at-least 5.8; then
    field needs "${x}✗${r} zsh $ZSH_VERSION  ${d}needs 5.8 or newer${r}"
  elif [ ${#missing} -eq 0 ] && [ ${#optional} -eq 0 ]; then
    field needs "${g}✓${r} all there"
  else
    local tool rowtxt
    local -a f
    rowtxt="${g}✓${r} zsh $ZSH_VERSION"
    field needs "$rowtxt"
    for tool in $__dehv_needs; do
      f=("${(@s:|:)tool}")
      if __dehv.has "$f[1]"; then
        [ "$f[2]" = macOS ] && continue
        field "" "${g}✓${r} ${(r:13:)f[1]}  ${d}${f[2]}${r}"
      elif [ "$f[3]" = optional ]; then
        field "" "${y}○${r} ${(r:13:)f[1]}  ${d}optional · ${f[4]}${r}"
      else
        field "" "${x}✗${r} ${(r:13:)f[1]}  ${d}missing · comes with ${f[2]}${r}"
      fi
    done
    local -a builtin=(${${(M)__dehv_needs:#*\|macOS\|*}%%\|*})
    builtin=(${builtin:|missing})
    [ ${#builtin} -gt 0 ] && field "" "${g}✓${r} ${builtin}  ${d}macOS${r}"
  fi

  if [ "$state" = here ] || [ "$state" = elsewhere ]; then
    read_remote "$target"
    field remote "$remote"
  fi
  print -r -- ""
}

# ─── needs ─────────────────────────────────────────────────────────────────────
# Asks to install what is missing, before anything else is touched. Fails when
# something required is still missing.
install_needs() {
  autoload -Uz is-at-least
  is-at-least 5.8 || { step fail zsh "$ZSH_VERSION: dehv needs 5.8 or newer"; return 1; }
  read_needs

  local tool skipped=
  for tool in $missing; do
    if [ "$tool" != git ]; then
      step fail "$tool" "missing, and it comes with macOS: this Mac lacks it"
      return 1
    fi
    if [ "$yes" = false ]; then
      [ "$tty" = true ] || { step fail git "missing: run xcode-select --install"; return 1; }
      ask "git comes with Apple's Command Line Tools, which are not installed." \
        "Install them|xcode-select --install opens the macOS dialog" \
        "Cancel|install nothing" || return 1
      [ "$REPLY" = Cancel ] && return 1
    fi
    xcode-select --install >/dev/null 2>&1
    spin "waiting for the Command Line Tools: finish the macOS dialog" 0 \
      zsh -c 'until xcode-select -p >/dev/null 2>&1; do sleep 2; done' || return 1
    step ok git "the Command Line Tools are installed"
  done

  for tool in $optional; do
    local why=${${(M)__dehv_needs:#$tool\|*}##*\|}
    if [ "$yes" = true ] || [ "$tty" = false ]; then
      skipped+=" $tool"
      continue
    fi
    if (( ! $+commands[brew] )); then
      step skip "$tool" "optional, not installed · https://cli.github.com"
      continue
    fi
    ask "$tool is optional: $why." \
      "Install $tool|brew install $tool" \
      "Skip|install it later" \
      "Cancel|install nothing" || return 1
    case "$REPLY" in
      Cancel) return 1 ;;
      Skip)   skipped+=" $tool" ;;
      *)
        if spin "brew install $tool" 0 brew install "$tool"; then
          step ok "$tool" "installed with Homebrew"
        else
          print -r -- "$REPLY" >&2
          step warn "$tool" "brew could not install it; dehv works without it"
        fi ;;
    esac
  done

  if [ -n "$skipped" ]; then
    step ok needs "all the required ones; skipped:$skipped"
  else
    step ok needs "all there"
  fi
}

# ─── rc file ───────────────────────────────────────────────────────────────────
# Sets $rc to the file to write the block in, or to nothing when I add the
# line myself.
choose_rc() {
  if [ -n "$rc_opt" ]; then
    rc=$rc_opt
    return
  fi
  if [ -n "$rc_now" ]; then
    rc=$rc_now
    return
  fi

  local main=${ZDOTDIR:-$HOME}/.zshrc local=$HOME/.zshrc.local
  if [ ! -L "$main" ]; then
    rc=$main
    return
  fi

  # A ~/.zshrc that is a link belongs to something else, a dotfiles repo most
  # likely: writing in it would change that repo.
  if [ "$yes" = true ] || [ "$tty" = false ]; then
    if [ -f "$local" ]; then
      rc=$local
      return
    fi
    step fail "rc file" "$(home "$main") is a link: name the file with --rc <file>"
    return 1
  fi

  local -a rows
  [ -f "$local" ] && rows+=("$(home "$local")|exists")
  rows+=("$(home "$main")|write it anyway, in $(home "${main:A}")" "Another file…|type its path" "Print the line|I add it myself")
  ask "$(home "$main") is a link to $(home "${main:A}"): writing there changes that file. Where should dehv load from?" \
    "${rows[@]}" || return 1

  case "$REPLY" in
    "Another file…")
      local file
      vared -p "   ${c}file${r} " file || return 1
      [ -n "$file" ] || return 1
      rc=${file/#\~/$HOME}
      [[ "$rc" == /* ]] || rc=$PWD/$rc ;;
    "Print the line")
      rc= ;;
    *)
      rc=${REPLY/#\~/$HOME} ;;
  esac
}

# The block, as the lines it has. The path is written with $HOME, so the file
# reads the same on any Mac.
block() {
  local src="${link/#$HOME/\$HOME}/dehv.plugin.zsh"
  print -r -- "$__dehv_mark_begin"
  print -r -- "# Loads dehv through its link; the installer in its clone writes and removes this."
  print -r -- "[ -f \"$src\" ] && source \"$src\""
  print -r -- "$__dehv_mark_end"
}

# Writes the block in $1: in place of the one there, or else at its end.
# Through the file, with cat, so a link there stays a link.
write_block() {
  local file=$1 new=$(mktemp) out=$(mktemp)
  block > "$new"
  mkdir -p "${file:h}" && touch "$file" || return 1
  awk -v b="$__dehv_mark_begin" -v e="$__dehv_mark_end" -v f="$new" '
    BEGIN { while ((getline l < f) > 0) blk = blk l "\n" }
    $0 == b { skip = 1; if (!done) { printf "%s", blk; done = 1 }; next }
    skip && $0 == e { skip = 0; next }
    !skip { print }
    END { if (!done) { if (NR > 0) print ""; printf "%s", blk } }
  ' "$file" > "$out" && cat "$out" > "$file"
  local s=$?
  rm -f "$new" "$out"
  return $s
}

# Takes the block out of $1, and the blank line write_block left before it.
remove_block() {
  local file=$1 out=$(mktemp)
  awk -v b="$__dehv_mark_begin" -v e="$__dehv_mark_end" '
    $0 == b { skip = 1; held = 0; next }
    skip { if ($0 == e) skip = 0; next }
    held { print ""; held = 0 }
    $0 == "" { held = 1; next }
    { print }
    END { if (held) print "" }
  ' "$file" > "$out" && cat "$out" > "$file"
  local s=$?
  rm -f "$out"
  return $s
}

# ─── steps ─────────────────────────────────────────────────────────────────────
make_link() {
  if [ -e "$link" ] && [ ! -L "$link" ]; then
    step fail link "$(home "$link") is a real folder: move it away first"
    return 1
  fi
  mkdir -p "${link:h}" && ln -sfn "$here" "$link" || { step fail link "could not create $(home "$link")"; return 1; }
  step ok link "$(home "$link") → $(home "$here")"
}

choose_greeting() {
  if [ -z "$greeting" ]; then
    greeting=$(__dehv.config hi.daily false)
    if [ "$yes" = false ] && [ "$tty" = true ]; then
      local -a rows=("Yes|show it" "No|never")
      ask "Show the dehv hi card in the first terminal of the day?" "${rows[@]}" || return 1
      [ "$REPLY" = Yes ] && greeting=true || greeting=false
    fi
  fi
  __dehv.config.set hi.daily "$greeting"
  if [ "$greeting" = true ]; then
    step ok greeting "on: dehv hi in the first terminal of the day"
  else
    step ok greeting "off: dehv hi shows it whenever you ask"
  fi
}

# A new interactive zsh, which reads the rc files as a new terminal does, has
# to come up with the dehv of this clone.
verify() {
  local out
  spin "opening a new zsh" 20 zsh -ic 'print -r -- "dir=$__dehv_dir"; print -r -- "n=${#__dehv_commands}"'
  out=$REPLY
  local dir=${${(M)${(f)out}:#dir=*}#dir=} n=${${(M)${(f)out}:#n=*}#n=}
  if [ -z "$dir" ]; then
    step fail checked "a new zsh does not load dehv: is $(home "$rc") read by it?"
  elif [ "$dir" != "$here" ]; then
    step fail checked "a new zsh loads another dehv after this one: $(home "$dir")"
  else
    step ok checked "a new zsh loads dehv: $n commands"
  fi
}

# ─── actions ───────────────────────────────────────────────────────────────────
do_install() {
  install_needs || return 1
  choose_rc || return 1

  make_link || return 1

  if [ -z "$rc" ]; then
    step skip "rc file" "add these lines to the one you choose:"
    print -r -- ""
    block | sed 's/^/     /'
    print -r -- ""
  else
    write_block "$rc" || { step fail "rc file" "could not write $(home "$rc")"; return 1; }
    __dehv.config.set install.rc "$(home "$rc")"
    step ok "rc file" "$(home "$rc") loads it"
    # Moved with --rc: the old file would load it a second time.
    if [ -n "$rc_now" ] && [ "${rc_now:A}" != "${rc:A}" ]; then
      remove_block "$rc_now" && step ok "rc file" "the block is out of $(home "$rc_now")"
    fi
  fi

  choose_greeting || return 1
  [ -n "$rc" ] && { verify || return 1; }

  print -r -- ""
  print -r -- "   ${g}Installed${r} ${d}·${r} run ${c}exec zsh${r} to load it, then ${c}dehv where${r}"
}

do_update() {
  local clone=${target:-$here}
  if ! git -C "$clone" rev-parse --abbrev-ref '@{u}' >/dev/null 2>&1; then
    step skip update "$(home "$clone") has no remote to pull from"
    return 0
  fi

  local old=$(git -C "$clone" rev-parse HEAD) oldv=$(__dehv.version "$clone")
  if ! spin "pulling in $(home "$clone")" 60 git -C "$clone" pull --ff-only -q; then
    print -r -- "$REPLY" | sed 's/^/     /' >&2
    step fail update "git could not pull: the clone has changes of its own?"
    return 1
  fi
  local new=$(git -C "$clone" rev-parse HEAD) newv=$(__dehv.version "$clone")

  if [ "$old" = "$new" ]; then
    step ok update "already up to date"
    return 0
  fi

  local count=$(git -C "$clone" rev-list --count "$old..$new")
  step ok pulled "$count commit$( (( count == 1 )) || print s)${newv:+ · v$oldv → v$newv}"
  git -C "$clone" log --format="%h  %s" "$old..$new" | while read -r line; do
    print -r -- "                ${y}${line%%  *}${r}  ${line#*  }"
  done
  print -r -- ""
  print -r -- "   ${g}Updated${r} ${d}·${r} run ${c}exec zsh${r} to load it"
}

do_relink() {
  make_link || return 1
  print -r -- ""
  print -r -- "   ${g}Relinked${r} ${d}·${r} run ${c}exec zsh${r} to load this clone"
}

do_uninstall() {
  if [ "$state" = none ]; then
    step skip uninstall "dehv is not installed"
    return 0
  fi

  if [ "$yes" = false ]; then
    [ "$tty" = true ] || { step fail uninstall "no terminal to confirm: add --yes"; return 1; }
    print -r -- ""
    print -r -- "   ${x}Uninstall dehv?${r} It removes the link${rc_now:+ and the block in $(home "$rc_now")}."
    read -q "?   ${d}Your data in ~/.config/dehv and the clone stay.${r}  [y/N] " || { print; return 1; }
    print -r -- ""
    print -r -- ""
  fi

  # Every rc file with the block, should there be more than one.
  local file
  while file=$(__dehv.rc); do
    remove_block "$file" || { step fail "rc file" "could not write $(home "$file")"; return 1; }
    step ok "rc file" "the block is out of $(home "$file")"
  done
  __dehv.config.set install.rc

  if [ -L "$link" ]; then
    rm "$link" && step ok link "$(home "$link") removed"
  fi

  local conf=${XDG_CONFIG_HOME:-$HOME/.config}/dehv cache=${XDG_CACHE_HOME:-$HOME/.cache}/dehv
  if [ "$yes" = false ] && [ "$tty" = true ] && { [ -d "$conf" ] || [ -d "$cache" ]; }; then
    if read -q "?   Also delete your data, $(home "$conf") and $(home "$cache")? [y/N] "; then
      print
      rm -rf "$conf" "$cache" && step ok data "deleted"
    else
      print
      step ok data "kept in $(home "$conf")"
    fi
  fi

  print -r -- ""
  print -r -- "   ${g}Uninstalled${r} ${d}·${r} open terminals keep dehv until ${c}exec zsh${r}"
}

# ─── main ──────────────────────────────────────────────────────────────────────
read_state

if [ -z "$action" ] && [ "$tty" = false ]; then
  action=status
fi

if [ -z "$action" ]; then
  header
  local -a rows
  case $state in
    none)
      rows=("Install|create the link and load it from the rc file") ;;
    here)
      if (( behind > 0 )); then
        rows=("Update|pull the $behind new commit$( (( behind == 1 )) || print s)")
      elif [[ "$remote" == none* ]]; then
        rows=("Update|no remote to pull from yet")
      else
        rows=("Update|git pull; it looks up to date")
      fi
      rows+=("Reinstall|write the link and the block again") ;;
    elsewhere)
      rows=("Relink|point the link to $(home "$here")" "Update|pull in $(home "$target") instead") ;;
    partial)
      rows=("Repair|write the link and the block again") ;;
  esac
  [ "$state" = none ] || rows+=("Uninstall|remove the link and the block")
  rows+=("Quit|")

  menu 1 "${rows[@]}" || exit 0
  print -r -- ""
  case "$REPLY" in
    Install|Reinstall|Repair) action=install ;;
    Update)    action=update ;;
    Relink)    action=relink ;;
    Uninstall) action=uninstall ;;
    *)         exit 0 ;;
  esac
elif [ "$action" = status ]; then
  header
  [ "$state" = here ] || [ "$state" = elsewhere ]
  exit
else
  print -r -- ""
  print -r -- " ${x}${b}❯${r}${w}${b}❯${r}${g}${b}❯${r} ${b}dehv${r}  ${d}${action}${version:+ · v$version}${r}"
  print -r -- ""
fi

case $action in
  install)   do_install ;;
  update)    do_update ;;
  relink)    do_relink ;;
  uninstall) do_uninstall ;;
esac
s=$?
print -r -- ""
exit $s
