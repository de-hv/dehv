# dehv doctor: whether dehv is installed whole on this Mac.
# Loaded by ../dehv.plugin.zsh, which says how a command registers itself.

__dehv_commands+=(doctor)
__dehv_about[doctor]='check the install of dehv and what it needs'
__dehv_rows+=(
  "doctor||check the link, the rc file, what dehv needs and its files"
)

# ─── doctor ────────────────────────────────────────────────────────────────────
# The same checks install.sh makes before installing, plus the files the
# commands write in ~/.config/dehv. It only looks: install.sh repairs.
__dehv.doctor.help() {
  __dehv.usage doctor "dehv doctor" \
    "Check the install of dehv: its link, the rc file that loads it, what it needs and its files." \
    "|check everything, in a card; exit code 1 when something is wrong"
}

__dehv.doctor() {
  [ $# -eq 0 ] || { __dehv.misuse doctor "unexpected argument '$1'"; return; }

  local b c d g y x r; __dehv.style
  local link=$(__dehv.link) home=${__dehv_dir/#$HOME/~} rc row
  local -a f checks

  # Each check as ok|fail|warn, what, detail, and a heading line as head|name.
  checks+=("head|install")
  if [ -L "$link" ] && [ "${link:A}" = "$__dehv_dir" ]; then
    checks+=("ok|link|${link/#$HOME/~} → $home")
  elif [ -L "$link" ] && [ ! -e "$link" ]; then
    checks+=("fail|link|${link/#$HOME/~} is broken")
  elif [ -L "$link" ]; then
    checks+=("warn|link|points to ${${link:A}/#$HOME/~}, not the clone loaded")
  elif [ -e "$link" ]; then
    checks+=("fail|link|${link/#$HOME/~} is a real folder, not a link")
  else
    checks+=("fail|link|${link/#$HOME/~} is missing")
  fi
  if rc=$(__dehv.rc); then
    checks+=("ok|rc file|${rc/#$HOME/~} loads it")
  else
    checks+=("fail|rc file|no rc file has the dehv block")
  fi
  if [ -n "${_comps[dehv]}" ]; then
    checks+=("ok|completion|dehv <tab>")
  else
    checks+=("warn|completion|not registered: compinit has not run")
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
    local foot="$ok/$total ok"
    (( failed > 0 )) && foot="$home/install.sh repairs it · $foot"
    __dehv.box doctor "$foot" "${lines[@]}"
  else
    print -rl -- ${lines:#}
  fi

  (( failed == 0 ))
}

# ─── completion ────────────────────────────────────────────────────────────────
_dehv.doctor() {
  _message 'no arguments'
}
