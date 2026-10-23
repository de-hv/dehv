# dehv where: where dehv is installed.
# Loaded by ../dehv.plugin.zsh, which says how a command registers itself.

__dehv_commands+=(where)
__dehv_about[where]='show where dehv is installed'
__dehv_rows+=(
  "where||the link, the clone it points to and the file that loads it"
)

# ─── where ─────────────────────────────────────────────────────────────────────
__dehv.where.help() {
  __dehv.usage where "dehv where" \
    "Where dehv is installed: the link, the clone it points to, the rc file that loads it and the version." \
    "|show it, in a card; piped, only the clone's path, for cd \"\$(dehv where)\""
}

__dehv.where() {
  [ $# -eq 0 ] || { __dehv.misuse where "unexpected argument '$1'"; return; }

  # Only the clone when piped, so `cd "$(dehv where)"` works.
  if [ ! -t 1 ]; then
    print -r -- "$__dehv_dir"
    return
  fi

  local b c d g y x r; __dehv.style
  local link=$(__dehv.link) rc version rev branch
  local -a lines

  if [ -L "$link" ] && [ "${link:A}" = "$__dehv_dir" ]; then
    lines+=("${c}link${r}       ${link/#$HOME/~}")
  elif [ -L "$link" ]; then
    lines+=("${c}link${r}       ${link/#$HOME/~} ${y}points to ${${link:A}/#$HOME/~}, not this clone${r}")
  else
    lines+=("${c}link${r}       ${y}none${r} ${d}· loaded straight from the clone${r}")
  fi
  lines+=("${c}points to${r}  ${y}${__dehv_dir/#$HOME/~}${r}")
  if rc=$(__dehv.rc); then
    lines+=("${c}loaded by${r}  ${rc/#$HOME/~}")
  else
    lines+=("${c}loaded by${r}  ${y}no rc file has the dehv block${r}")
  fi

  version=$(__dehv.version)
  rev=$(git -C "$__dehv_dir" log -1 --format=%h 2>/dev/null)
  branch=$(git -C "$__dehv_dir" branch --show-current 2>/dev/null)
  lines+=("${c}version${r}    ${version:-unknown}${rev:+ ${d}· ${branch:+$branch · }$rev${r}}")

  __dehv.box "dehv where" "" "${lines[@]}"
}

# ─── completion ────────────────────────────────────────────────────────────────
_dehv.where() {
  _message 'no arguments'
}
