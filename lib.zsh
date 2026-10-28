# Helpers every dehv command shares: colours, help cards, output marks, the
# arrow-key menus, and where the install and the config are. Loaded first by
# dehv.plugin.zsh, and by install.sh.

# ─── style ─────────────────────────────────────────────────────────────────────
# Bold, and One Dark's grey, red, green, yellow and cyan: in 24-bit colour when
# the terminal says it has it ($COLORTERM), else the nearest of the 256 every
# terminal has. Sets $b $c $d $g $y $x and $r in the caller, which declares
# them local.
__dehv.palette() {
  b=$'\e[1m' r=$'\e[0m'
  if [[ "$COLORTERM" == (truecolor|24bit) ]]; then
    d=$'\e[38;2;127;132;142m' x=$'\e[38;2;224;108;117m' g=$'\e[38;2;152;195;121m'
    y=$'\e[38;2;229;192;123m' c=$'\e[38;2;86;182;194m'
  else
    d=$'\e[38;5;245m' x=$'\e[38;5;168m' g=$'\e[38;5;114m' y=$'\e[38;5;180m' c=$'\e[38;5;73m'
  fi
}

# The palette, or plain text when the output ($1, a file descriptor; stdout
# without it) is not a terminal or NO_COLOR is set.
__dehv.style() {
  b= c= d= g= y= x= r=
  [ -t ${1:-1} ] && [ -z "$NO_COLOR" ] && __dehv.palette
  return 0
}

# ─── install ───────────────────────────────────────────────────────────────────
# The link install.sh makes to the clone, so the rc file names one path
# wherever the clone is.
__dehv.link() {
  print -r -- "${XDG_DATA_HOME:-$HOME/.local/share}/dehv"
}

# The block install.sh writes in the rc file, between these two lines.
__dehv_mark_begin='# >>> dehv >>>'
__dehv_mark_end='# <<< dehv <<<'

# What dehv needs, as command|where it comes from|required or optional|what
# for. Checked by install.sh and `dehv doctor`; zsh itself is checked apart.
__dehv_needs=(
  "git|Command Line Tools|required|"
  "curl|macOS|required|"
  "lsof|macOS|required|"
  "pbcopy|macOS|required|"
  "osascript|macOS|required|"
  "sips|macOS|required|"
  "networksetup|macOS|required|"
  "gh|Homebrew|optional|dehv wall uses it for ssh or https"
)

# Whether $1 is installed. macOS ships a /usr/bin/git that only offers to
# install the Command Line Tools, so git counts once they are there.
__dehv.has() {
  (( $+commands[$1] )) || return 1
  [ "$1" = git ] && [ "${commands[git]}" = /usr/bin/git ] && { xcode-select -p >/dev/null 2>&1 || return 1; }
  return 0
}

# The version in the clone at $1, the one loaded without it.
__dehv.version() {
  local file="${1:-$__dehv_dir}/VERSION"
  [ -f "$file" ] && print -r -- "$(<"$file")"
}

# The rc file that has the dehv block: the one install.sh noted in the config,
# or else the first of the usual ones that has it. Fails when none does.
__dehv.rc() {
  local file z=${ZDOTDIR:-$HOME}
  for file in "$(__dehv.config install.rc)" "$z/.zshrc" "$z/.zprofile" "$z/.zshenv"; do
    [ -n "$file" ] && [ -f "$file" ] && grep -qxF "$__dehv_mark_begin" "$file" 2>/dev/null || continue
    print -r -- "$file"
    return 0
  done
  return 1
}

# ─── config ────────────────────────────────────────────────────────────────────
# This Mac's settings, in ~/.config/dehv/config, shaped like a git config as
# the profiles of `dehv as`:
#
#   [hi]
#   	daily = true
#   [skill]
#   	dir = ~/Documents/dev/skills
#   [install]
#   	rc = ~/.zshrc
#
# install.sh writes hi.daily and install.rc; skill.dir is mine to set.
__dehv.config.file() {
  print -r -- "${XDG_CONFIG_HOME:-$HOME/.config}/dehv/config"
}

# Prints the value of $1, or $2 when it is not set, with a leading ~ expanded.
# Without the file git is not asked at all: on a new Mac it would offer to
# install itself.
__dehv.config() {
  local file=$(__dehv.config.file) value
  [ -f "$file" ] && value=$(git config -f "$file" --get "$1" 2>/dev/null) || value=$2
  print -r -- "${value/#\~/$HOME}"
}

# Sets $1 to $2, or unsets it without $2.
__dehv.config.set() {
  local file=$(__dehv.config.file)
  if [ $# -ge 2 ]; then
    mkdir -p "${file:h}" && git config -f "$file" "$1" "$2"
  else
    [ -f "$file" ] && git config -f "$file" --unset "$1" 2>/dev/null
    return 0
  fi
}

# ─── help ──────────────────────────────────────────────────────────────────────
# A command's help. $1 the command, $2 its usage line (the one a misuse
# repeats), $3 what it is for, then a row per way to call it as arguments|what
# it does: in a terminal, a card; piped, plain text that starts with the usage.
__dehv.usage() {
  local name="$1" usage="$2" about="$3"; shift 3
  local b c d g y x r; __dehv.style
  local row w=0
  local -a f lines

  for row in "$@"; do
    f=("${(@s:|:)row}")
    (( ${#f[1]} > w )) && w=${#f[1]}
  done

  if [ -t 1 ]; then
    # No wider than the terminal: the card takes 4 columns, and what each row
    # does wraps beside its arguments, or under them when that leaves too few.
    local fit=$(( $(__dehv.cols) - 4 )) i
    local dw=$(( fit - w - 2 )) indent=$(( w + 2 ))
    (( dw < 24 )) && dw=$(( fit - 2 )) indent=2

    __dehv.wrap "$about" $fit
    for i in "${reply[@]}"; do lines+=("${d}${i}${r}"); done
    lines+=("")
    for row in "$@"; do
      f=("${(@s:|:)row}")
      __dehv.wrap "${f[2]}" $dw
      if (( indent == 2 )); then
        lines+=("${y}${f[1]}${r}")
        for i in "${reply[@]}"; do lines+=("  $i"); done
      else
        lines+=("${y}${(r:w:)f[1]}${r}  ${reply[1]}")
        for i in "${(@)reply[2,-1]}"; do lines+=("${(l:indent:)}$i"); done
      fi
    done
    __dehv.box "dehv $name" "" "${lines[@]}"
  else
    for row in "$@"; do
      f=("${(@s:|:)row}")
      lines+=("${(r:w:)f[1]}  ${f[2]}")
    done
    print -r -- "Usage: $usage

$about
"
    print -rl -- "  "${^lines}
  fi
}

__dehv.unknown() {
  echo "dehv: unknown command '$1'" >&2
  echo "Run 'dehv -h' for usage." >&2
  return 1
}

# $1 command, $2 message: the error, then the command's usage line.
__dehv.misuse() {
  echo "dehv $1: $2" >&2
  "__dehv.$1.help" | sed -n 1p >&2
  return 1
}

# ─── output ────────────────────────────────────────────────────────────────────
# What a command did, on stdout: a green check and $1.
__dehv.ok() {
  local b c d g y x r; __dehv.style
  print -r -- "${g}✓${r} $1"
}

# What it could not, on stderr: a red cross, $1 and, dimmed, the cause in $2.
__dehv.fail() {
  local b c d g y x r; __dehv.style 2
  print -r -- "${x}✗${r} $1${2:+ ${d}· $2${r}}" >&2
  return 1
}

# That there was nothing to do.
__dehv.noop() {
  local b c d g y x r; __dehv.style
  print -r -- "${d}· $1${r}"
}

# The terminal's width: $COLUMNS, or what tput says where it is not set.
__dehv.cols() {
  print -r -- ${COLUMNS:-$(tput cols 2>/dev/null || print 80)}
}

# $1 broken into lines of at most $2 columns, at spaces, into $reply.
__dehv.wrap() {
  local word line=
  reply=()
  for word in ${=1}; do
    if [ -z "$line" ]; then
      line=$word
    elif (( ${#line} + 1 + ${#word} <= $2 )); then
      line+=" $word"
    else
      reply+=("$line") line=$word
    fi
  done
  [ -n "$line" ] && reply+=("$line")
}

# $1 cut to $2 columns of text with an …, keeping its colours, which take none.
__dehv.cut() {
  setopt localoptions extendedglob
  local s="$1" out= seq n=0
  local plain=${s//$'\e'\[[0-9;]#m/}
  (( ${#plain} <= $2 )) && { print -r -- "$s"; return }
  while [ -n "$s" ] && (( n < $2 - 1 )); do
    if [ "${s[1]}" = $'\e' ]; then
      seq=${(M)s##$'\e'\[[0-9;]#m}
      [ -n "$seq" ] || seq=${s[1]}
      out+=$seq s=${s[${#seq}+1,-1]}
    else
      out+=${s[1]} s=${s[2,-1]}
      (( n++ ))
    fi
  done
  [[ "$out" == *$'\e'* ]] && out+=$'\e[0m'
  print -r -- "$out…"
}

# A body line of __dehv.box that is only this draws a rule across the frame.
__dehv_rule=$'\x1f'

# A rounded frame around the lines given, with $1 in its top border (a plain
# one without it) and $2 in the bottom one. The lines may carry colours: they
# are measured without. A line with a tab has its part after the tab pushed to
# the right edge, and a line that is only $__dehv_rule is a rule. In a terminal
# it is never wider than it: a line too long loses its right part first, then
# is cut to fit.
__dehv.box() {
  setopt localoptions extendedglob
  local title="$1" foot="$2"; shift 2
  local b c d g y x r; __dehv.style
  local -a body=("$@")

  local line plain left right i w=$(( ${#title} + 2 ))
  (( ${#foot} + 2 > w )) && w=$(( ${#foot} + 2 ))
  for line in "${body[@]}"; do
    plain=${${line//$'\e'\[[0-9;]#m/}/$'\t'/  }
    (( ${#plain} > w )) && w=${#plain}
  done

  if [ -t 1 ]; then
    local fit=$(( $(__dehv.cols) - 4 ))
    if (( w > fit )); then
      w=$fit
      for (( i = 1; i <= ${#body}; i++ )); do
        line=${body[i]}
        plain=${${line//$'\e'\[[0-9;]#m/}/$'\t'/  }
        (( ${#plain} > fit )) && [[ "$line" == *$'\t'* ]] && line=${line%%$'\t'*}
        body[i]=$(__dehv.cut "$line" $fit)
      done
      (( ${#foot} + 2 > w )) && foot=
    fi
  fi

  local n=$(( w - 1 - ${#title} )) full=$(( w + 2 ))
  if [ -n "$title" ]; then
    print -r -- "${d}╭─${r} ${c}${title}${r} ${d}${(l:n::─:):-}╮${r}"
  else
    print -r -- "${d}╭${(l:full::─:):-}╮${r}"
  fi
  for line in "${body[@]}"; do
    if [ "$line" = "$__dehv_rule" ]; then
      print -r -- "${d}├${(l:full::─:):-}┤${r}"
      continue
    fi
    if [[ "$line" == *$'\t'* ]]; then
      left=${line%%$'\t'*} right=${line#*$'\t'}
      plain=${left//$'\e'\[[0-9;]#m/}${right//$'\e'\[[0-9;]#m/}
      n=$(( w - ${#plain} ))
      line="${left}${(l:n:):-}${right}"
    fi
    plain=${line//$'\e'\[[0-9;]#m/}
    n=$(( w - ${#plain} ))
    print -r -- "${d}│${r} ${line}${(l:n:):-} ${d}│${r}"
  done
  if [ -n "$foot" ]; then
    n=$(( w - 1 - ${#foot} ))
    print -r -- "${d}╰${(l:n::─:):-} ${foot} ─╯${r}"
  else
    n=$(( w + 2 ))
    print -r -- "${d}╰${(l:n::─:):-}╯${r}"
  fi
}

# ─── pick ──────────────────────────────────────────────────────────────────────
# An arrow-key menu over the lines given: sets $REPLY to the chosen one's
# number, or returns 1 on esc/q. Drawn on the terminal itself and wiped after,
# so nothing of it ends up in a pipe.
__dehv.pick() {
  local -a items=("$@")
  local n=${#items} cur=1 i key rest
  local b c d g y x r; __dehv.palette
  local bold=$b cyan=$c reset=$r
  [ -z "$NO_COLOR" ] || bold= cyan= reset=

  print -n -- $'\e[?25l' > /dev/tty
  while true; do
    for i in {1..$n}; do
      if (( i == cur )); then
        print -r -- "${cyan}›${reset} ${bold}${items[i]}${reset}"
      else
        print -r -- "  ${items[i]}"
      fi
    done > /dev/tty
    print -r -- "  ↑↓ move · enter choose · esc cancel" > /dev/tty

    read -sk1 key < /dev/tty
    case "$key" in
      $'\e')
        rest=
        read -sk2 -t 0.05 rest < /dev/tty
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

    print -n -- $'\e['$(( n + 1 ))$'A\e[J' > /dev/tty
    [ "$key" = enter ] || (( cur == 0 )) && break
  done
  print -n -- $'\e[?25h' > /dev/tty

  REPLY=$cur
  (( cur > 0 ))
}

# The same menu with a box on each line: space ticks, enter confirms. $1 is the
# heading, $2 the numbers of the lines ticked to begin with, the rest the
# lines. Sets $reply to the numbers left ticked, or returns 1 on esc/q.
__dehv.pickmany() {
  local head="$1"
  local -a on=(${=2}); shift 2
  local -a items=("$@")
  local n=${#items} cur=1 i key rest box quit=false
  local -A tick
  for i in $on; do tick[$i]=1; done
  local b c d g y x r; __dehv.palette
  local bold=$b dim=$d green=$g cyan=$c reset=$r
  [ -z "$NO_COLOR" ] || bold= dim= green= cyan= reset=

  print -n -- $'\e[?25l' > /dev/tty
  while true; do
    {
      print -r -- "${dim}${head}${reset}"
      for i in {1..$n}; do
        [ -n "${tick[$i]}" ] && box="${green}[x]${reset}" || box="${dim}[ ]${reset}"
        if (( i == cur )); then
          print -r -- "${cyan}›${reset} $box ${bold}${items[i]}${reset}"
        else
          print -r -- "  $box ${items[i]}"
        fi
      done
      print -r -- "${dim}  ↑↓ move · space tick · enter confirm · esc cancel${reset}"
    } > /dev/tty

    read -sk1 key < /dev/tty
    case "$key" in
      $'\e')
        rest=
        read -sk2 -t 0.05 rest < /dev/tty
        case "$rest" in
          '[A') (( cur > 1 )) && (( cur-- )) ;;
          '[B') (( cur < n )) && (( cur++ )) ;;
          '')   quit=true ;;
        esac
        ;;
      k) (( cur > 1 )) && (( cur-- )) ;;
      j) (( cur < n )) && (( cur++ )) ;;
      ' ') [ -n "${tick[$cur]}" ] && unset "tick[$cur]" || tick[$cur]=1 ;;
      q) quit=true ;;
      $'\n'|$'\r') key=enter ;;
    esac

    print -n -- $'\e['$(( n + 2 ))$'A\e[J' > /dev/tty
    [ "$key" = enter ] || [ "$quit" = true ] && break
  done
  print -n -- $'\e[?25h' > /dev/tty

  reply=()
  for i in {1..$n}; do
    [ -n "${tick[$i]}" ] && reply+=($i)
  done
  [ "$quit" = false ]
}
