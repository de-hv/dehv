# dehv doctor: whether dehv is installed whole on this Mac.
# Loaded by ../bin/dehv, which says how a command registers itself.

__dehv_commands+=(doctor)
__dehv_about[doctor]='check the install of dehv and what it needs'

# ─── doctor ────────────────────────────────────────────────────────────────────
# What dehv needs, whether the dehv in the PATH is this one and has its
# completion, plus the files the commands write in ~/.config/dehv. It only
# looks.
__dehv.doctor.help() {
  __dehv.usage doctor "dehv doctor" \
    "Check the install of dehv: the command in the PATH, its completion, what it needs and its files." \
    "|check everything, in a card; exit code 1 when something is wrong"
}

__dehv.doctor() {
  [ $# -eq 0 ] || { __dehv.misuse doctor "unexpected argument '$1'"; return; }

  local b c d g y x r; __dehv.style
  local home=${__dehv_dir/#$HOME/~} row comp
  local -a f checks

  # Each check as ok|fail|warn, what, detail, and a heading line as head|name.
  checks+=("head|install")
  if (( ! $+commands[dehv] )); then
    checks+=("warn|command|dehv is not in the PATH, only ${__dehv_bin/#$HOME/~}")
  elif [ "${commands[dehv]:A:h:h}" = "$__dehv_dir" ]; then
    checks+=("ok|command|${commands[dehv]/#$HOME/~}")
  else
    checks+=("warn|command|the dehv in the PATH is ${commands[dehv]/#$HOME/~}, not this one")
  fi
  if [ "$(__dehv.from)" = homebrew ]; then
    comp=${__dehv_dir%%/Cellar/*}/share/zsh/site-functions/_dehv
    if [ -e "$comp" ]; then
      checks+=("ok|completion|${comp/#$HOME/~}")
    else
      checks+=("fail|completion|${comp/#$HOME/~} is missing: brew link dehv")
    fi
  else
    checks+=("warn|completion|not from Homebrew: $home/completions goes in \$fpath by hand")
  fi

  checks+=("head|needs")
  autoload -Uz is-at-least
  if is-at-least 5.8; then
    checks+=("ok|zsh|$ZSH_VERSION")
  else
    checks+=("fail|zsh|$ZSH_VERSION, needs 5.8 or newer")
  fi
  for row in $__dehv_needs; do
    f=("${(@s:|:)row}")
    if __dehv.has "$f[1]"; then
      checks+=("ok|$f[1]|$f[2]")
    elif [ "$f[3]" = optional ]; then
      checks+=("warn|$f[1]|not installed, optional: $f[4]")
    else
      checks+=("fail|$f[1]|not installed, comes with $f[2]")
    fi
  done

  # The files the commands write, when they are there; profiles and walls
  # stay readable only by me.
  checks+=("head|files")
  local name file conf=${XDG_CONFIG_HOME:-$HOME/.config}/dehv
  for name in config profiles walls; do
    file=$conf/$name
    if [ ! -f "$file" ]; then
      checks+=("ok|$name|none yet")
    elif [ "$name" != config ] && [ "$(stat -f %Lp "$file")" != 600 ]; then
      checks+=("warn|$name|readable by others: chmod 600 ${file/#$HOME/~}")
    else
      checks+=("ok|$name|${file/#$HOME/~}")
    fi
  done

  local ok=0 total=0 failed=0
  for row in $checks; do
    case ${row%%|*} in
      ok)   (( ok++, total++ )) ;;
      warn) (( total++ )) ;;
      fail) (( failed++, total++ )) ;;
    esac
  done

  local w=0
  for row in $checks; do
    f=("${(@s:|:)row}")
    [ "$f[1]" != head ] && (( ${#f[2]} > w )) && w=${#f[2]}
  done

  local -a lines
  local mark
  for row in $checks; do
    f=("${(@s:|:)row}")
    case $f[1] in
      head) [ ${#lines} -gt 0 ] && lines+=(""); lines+=("${d}${f[2]}${r}"); continue ;;
      ok)   mark="${g}✓${r}" ;;
      warn) mark="${y}!${r}" ;;
      fail) mark="${x}✗${r}" ;;
    esac
    if [ -t 1 ]; then
      lines+=("$mark ${(r:w:)f[2]}  ${d}${f[3]}${r}")
    else
      lines+=("${(r:4:)f[1]} ${(r:w:)f[2]}  ${f[3]}")
    fi
  done

  if [ -t 1 ]; then
    __dehv.box doctor "$ok/$total ok" "${lines[@]}"
  else
    print -rl -- ${lines:#}
  fi

  (( failed == 0 ))
}

# ─── completion ────────────────────────────────────────────────────────────────
_dehv.doctor() {
  _message 'no arguments'
}
