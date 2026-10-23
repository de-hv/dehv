# dehv branches: the git branches that can go, and deleting the ones I tick.
# Loaded by ../dehv.plugin.zsh, which says how a command registers itself.

__dehv_commands+=(branches)
__dehv_about[branches]='delete the git branches already merged or gone'
__dehv_rows+=(
  "branches||tick which merged or gone branches to delete"
  "|-l, --list|only list them, with why each can go"
)

# ─── branches ──────────────────────────────────────────────────────────────────
# Every local branch but the one checked out and the default one, each with why
# it can go: merged into the default branch, or gone (its upstream was deleted,
# as a squash-merged PR leaves it). Those come ticked; the rest are listed
# unticked, as unmerged. It fetches with --prune first, so "gone" is current.
__dehv.branches.help() {
  __dehv.usage branches "dehv branches [-l]" \
    "Delete the local git branches already merged or whose remote is gone." \
    "|fetch, then tick the branches to delete from a list" \
    "-l, --list|only list them, with why each can go"
}

__dehv.branches() {
  local list=false arg
  for arg in "$@"; do
    case "$arg" in
      -l|--list) list=true ;;
      -*)        __dehv.misuse branches "unknown option '$arg'"; return ;;
      *)         __dehv.misuse branches "unexpected argument '$arg'"; return ;;
    esac
  done

  local b c d g y x r; __dehv.style
  git rev-parse --git-dir >/dev/null 2>&1 || { __dehv.fail "not in a git repo"; return; }

  # A spinner while it fetches, as dehv ip does; offline it goes on with what
  # it knew.
  if [ -n "$(git remote)" ]; then
    local fd i=0 spin='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏' done
    exec {fd}< <(GIT_TERMINAL_PROMPT=0 git fetch --prune --quiet >/dev/null 2>&1; echo)
    until read -t 0.08 -u $fd done; do
      [ -t 2 ] && print -n -- "\r${c}${spin[i % 10 + 1]}${r} ${d}fetching…${r}" >&2
      (( i++ ))
    done
    [ -t 2 ] && print -n -- $'\r\e[K' >&2
    exec {fd}<&-
  fi

  local base current
  base=$(git symbolic-ref -q --short refs/remotes/origin/HEAD 2>/dev/null)
  base=${base#origin/}
  if [ -z "$base" ]; then
    for base in main master ""; do
      git show-ref -q --verify "refs/heads/$base" 2>/dev/null && break
    done
  fi
  current=$(git branch --show-current)

  local -a merged names why on
  [ -n "$base" ] && merged=(${(f)"$(git branch --format='%(refname:short)' --merged "$base")"})

  local row name track age
  for row in ${(f)"$(git for-each-ref --format='%(refname:short)|%(upstream:track)|%(committerdate:relative)' refs/heads)"}; do
    name=${row%%|*} row=${row#*|} track=${row%%|*} age=${row#*|}
    [ "$name" = "$base" ] || [ "$name" = "$current" ] && continue
    names+=("$name")
    if (( ${merged[(Ie)$name]} )); then
      why+=("merged   · $age"); on+=(${#names})
    elif [ "$track" = "[gone]" ]; then
      why+=("gone     · $age"); on+=(${#names})
    else
      why+=("unmerged · $age")
    fi
  done

  if [ ${#names} -eq 0 ]; then
    __dehv.noop "only ${base:-$current} here, nothing to delete"
    return
  fi

  local w=0
  for name in $names; do (( ${#name} > w )) && w=${#name}; done

  if [ "$list" = true ] || [ ! -t 0 ] || [ ! -t 1 ]; then
    local -a lines
    for i in {1..${#names}}; do
      if [ -t 1 ]; then
        (( ${on[(Ie)$i]} )) && lines+=("${y}${(r:w:)names[i]}${r}  ${why[i]}") \
                            || lines+=("${(r:w:)names[i]}  ${d}${why[i]}${r}")
      else
        lines+=("${(r:w:)names[i]}  ${why[i]}")
      fi
    done
    if [ -t 1 ]; then
      __dehv.box branches "${#on} can go · base ${base:-none}" "${lines[@]}"
    else
      print -rl -- $lines
    fi
    return
  fi

  local -a items
  for i in {1..${#names}}; do items+=("${(r:w:)names[i]}  ${why[i]}"); done
  __dehv.pickmany "Delete which branches? (base: ${base:-none})" "$on" "${items[@]}" || return 1
  if [ ${#reply} -eq 0 ]; then
    __dehv.noop "none ticked, nothing deleted"
    return
  fi

  # -D, not -d: a gone branch was usually squash-merged, which -d cannot see.
  local rc=0
  for i in $reply; do
    if git branch -D -q "${names[i]}" 2>/dev/null; then
      __dehv.ok "deleted ${b}${names[i]}${r} ${d}· ${why[i]%% *}${r}"
    else
      __dehv.fail "could not delete ${names[i]}"
      rc=1
    fi
  done
  return $rc
}

# ─── completion ────────────────────────────────────────────────────────────────
_dehv.branches() {
  _arguments '(-l --list)'{-l,--list}'[only list them, with why each can go]'
}
