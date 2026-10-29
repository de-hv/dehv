# dehv where: where dehv is installed.
# Loaded by ../bin/dehv, which says how a command registers itself.

__dehv_commands+=(where)
__dehv_about[where]='show where dehv is installed'

# ─── where ─────────────────────────────────────────────────────────────────────
__dehv.where.help() {
  __dehv.usage where "dehv where" \
    "Where dehv is installed: the command run, the folder with its files, where it comes from and the version." \
    "|show it, in a card; piped, only the folder's path, for cd \"\$(dehv where)\""
}

__dehv.where() {
  [ $# -eq 0 ] || { __dehv.misuse where "unexpected argument '$1'"; return; }

  # Only the folder when piped, so `cd "$(dehv where)"` works.
  if [ ! -t 1 ]; then
    print -r -- "$__dehv_dir"
    return
  fi

  local b c d g y x r; __dehv.style
  local version=$(__dehv.version) rev=$(__dehv.rev) from
  local -a lines

  case $(__dehv.from) in
    homebrew) from="Homebrew ${d}· brew upgrade dehv${r}" ;;
    clone)    from="a clone ${d}· git pull${r}" ;;
    *)        from="a folder" ;;
  esac

  lines+=("${c}command${r}  ${__dehv_bin/#$HOME/~}")
  lines+=("${c}files${r}    ${y}${__dehv_dir/#$HOME/~}${r}")
  lines+=("${c}from${r}     $from")
  lines+=("${c}version${r}  ${version:-unknown}${rev:+ ${d}· $rev${r}}")

  __dehv.box "dehv where" "" "${lines[@]}"
}

# ─── completion ────────────────────────────────────────────────────────────────
_dehv.where() {
  _message 'no arguments'
}
