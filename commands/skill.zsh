# dehv skill: my skills and the agents that see them.
# Loaded by ../dehv.plugin.zsh, which says how a command registers itself.

__dehv_commands+=(skill)
__dehv_about[skill]='manage my skills and the agents that see them'
__dehv_rows+=(
  "skill|<name>|create a skill"
  "|-l, --link <name>|link it into the agents I pick, creating it if missing"
  "|-u, --unlink <name>|take it out of the agents I pick, keeping it"
  "|-d, --delete <name>|unlink it everywhere and move it to the Trash"
  "|-r, --restore <name>|bring it back from the Trash, and pick where to link it"
  "|-t, --to <agents>|name the agents instead of picking: claude,cursor"
  "|-a, --all|list them, and the agents each is linked into"
)

# The agents `dehv skill` can link a skill into, as name|the folder that tells
# the agent is installed|where its skills go. Only the installed ones are ever
# offered; the first one is the default.
__dehv_agents=(
  "claude|$HOME/.claude|$HOME/.claude/skills"
  "cursor|$HOME/Library/Application Support/Cursor/User|$HOME/Library/Application Support/Cursor/User/skills"
  "copilot|$HOME/.copilot|$HOME/.copilot/skills"
  "xcode|$HOME/Library/Developer/Xcode/CodingAssistant|$HOME/Library/Developer/Xcode/CodingAssistant"
  "codex|$HOME/.agents|$HOME/.agents/skills"
)

# ─── skill ─────────────────────────────────────────────────────────────────────
# Skills are written in one folder, skill.dir in the config
# (~/Documents/dev/skills without it), and linked into the agents that should
# see them (the table at the top), picked from a list or named with -t.
# A deleted one goes to the Trash, and through the Finder both ways: macOS
# keeps the shell out of ~/.Trash, and only what the Finder trashed gets
# "Put Back". The first time, macOS asks to let the terminal control Finder.
__dehv.skill.help() {
  __dehv.usage skill "dehv skill [-l] [-t <agents>] <name> | -u <name> | -d <name> | -r <name> | -a" \
    "Manage my skills, kept in ${$(__dehv.skill.dir)/#$HOME/~}, and the agents that see them." \
    "<name>|create it, with a SKILL.md that has its frontmatter" \
    "-l, --link <name>|link it into the agents ticked in a list, creating it if missing" \
    "-u, --unlink <name>|take it out of the agents ticked, keeping the skill" \
    "-d, --delete <name>|unlink it from every agent and move it to the Trash" \
    "-r, --restore <name>|bring it back from the Trash, then tick where to link it" \
    "-t, --to <agents>|no list: the agents by name, like claude,cursor (with -l, -u, -r)" \
    "-a, --all|list my skills, and the agents each one is linked into"
}

# Where my skills are.
__dehv.skill.dir() {
  __dehv.config skill.dir "$HOME/Documents/dev/skills"
}

# $1 the AppleScript line, the rest its argv. Quiet: the caller reports.
__dehv.finder() {
  local line="$1"; shift
  osascript -e 'on run argv' -e "tell application \"Finder\" to $line" -e 'end run' "$@" >/dev/null 2>&1
}

# Reads the table of agents into the caller's $known (every name), $agents
# (the installed ones) and $adir (name -> where its skills go).
__dehv.skill.agents() {
  local row
  local -a f
  known=() agents=() adir=()
  for row in $__dehv_agents; do
    f=("${(@s:|:)row}")
    known+=("${f[1]}")
    adir[${f[1]}]="${f[3]}"
    [ -d "${f[2]}" ] && agents+=("${f[1]}")
  done
}

# The installed agents the caller's $name is linked into, in $reply.
__dehv.skill.where() {
  local a
  reply=()
  for a in $agents; do
    [ -L "${adir[$a]}/$name" ] && reply+=("$a")
  done
}

# Which agents to act on, into the caller's $chosen: the ones named with -t,
# or the ones ticked in a list of the candidates, or without a terminal the
# ones that would start ticked. $1 the list's heading, $2 the action as in
# "cannot <link into> x", $3 the names ticked to begin with, the rest the
# candidates. Returns 1 when the list is cancelled.
__dehv.skill.choose() {
  local head="$1" action="$2"
  local -a marked=(${=3}); shift 3
  local -a cand=("$@")
  local a i
  chosen=()

  if [ ${#to_names} -gt 0 ]; then
    for a in $to_names; do
      if (( ! ${agents[(Ie)$a]} )); then
        __dehv.fail "cannot $action $a" "it is not installed"
        rc=1
        continue
      fi
      chosen+=("$a")
    done
    return 0
  fi

  if [ ! -t 0 ] || [ ! -t 1 ]; then
    chosen=($marked)
    return 0
  fi

  local w=0
  local -a items on
  for a in $cand; do (( ${#a} > w )) && w=${#a}; done
  for i in {1..${#cand}}; do
    a=${cand[i]}
    items+=("${(r:w:)a}  ${adir[$a]/#$HOME/~}${${(M)a:#${known[1]}}:+  default}")
    (( ${marked[(Ie)$a]} )) && on+=($i)
  done
  __dehv.pickmany "$head" "$on" "${items[@]}" || return 1
  for i in $reply; do chosen+=("${cand[i]}"); done
  return 0
}

# Link the caller's $name into agent $1, or take it out. Each says what it did.
__dehv.skill.link() {
  local dir="${adir[$1]}"
  if [ -e "$dir/$name" ] && [ ! -L "$dir/$name" ]; then
    __dehv.fail "cannot link into $1" "it has a $name of its own"
    return
  fi
  [ -L "$dir/$name" ] && rm "${dir:?}/${name:?}"
  mkdir -p "$dir" && ln -s "$skills/$name" "$dir/$name" \
    && __dehv.ok "linked   ${d}${dir/#$HOME/~}/${r}${b}${name}${r}"
}

__dehv.skill.unlink() {
  local dir="${adir[$1]}"
  # Only my own link: a real folder of that name in the agent stays.
  [ -L "$dir/$name" ] || return 0
  rm "${dir:?}/${name:?}" && __dehv.ok "unlinked ${d}${dir/#$HOME/~}/${r}${b}${name}${r}"
}

# One skill per line: its name, a dot under each installed agent, green when
# that agent links to it, and the description in its frontmatter. Flags the
# ones without description, which no agent can trigger, and my links left
# pointing at nothing.
__dehv.skill.all() {
  local skills=$(__dehv.skill.dir)
  local b c d g y x r; __dehv.style
  local -a known agents names broken
  local -A adir brokenin
  __dehv.skill.agents
  names=("$skills"/*(N/:t))

  local a link
  for a in $agents; do
    for link in "${adir[$a]}"/*(N@); do
      [ -e "$link" ] && continue
      [[ "$(readlink "$link")" == "$skills"/* ]] || continue
      (( ${broken[(Ie)${link:t}]} )) || broken+=("${link:t}")
      brokenin[${link:t}]+=" $a "
    done
  done

  if [ ${#names} -eq 0 ] && [ ${#broken} -eq 0 ]; then
    echo "No skills in ${skills/#$HOME/~}."
    return
  fi

  local name width=4 used=0 pad
  for name in $names $broken; do (( ${#name} > width )) && width=${#name}; done
  local head="${(r:width:):-NAME}  "
  for a in $agents; do
    head+="$a  "
    (( used += ${#a} + 2 ))
  done
  local max=$(( ${COLUMNS:-80} - width - used - 3 ))
  print -r -- "${d}${head}DESCRIPTION${r}"

  local line desc
  for name in $names; do
    line="${b}${(r:width:)name}${r}  "
    for a in $agents; do
      pad=$(( ${#a} + 1 ))
      [ -L "${adir[$a]}/$name" ] && line+="${g}●${r}${(l:pad:):-}" || line+="${d}○${r}${(l:pad:):-}"
    done
    desc=$(awk '/^---/ { n++; next } n == 1 && /^description:/ { sub(/^description:[ \t]*/, ""); print; exit }' \
      "$skills/$name/SKILL.md" 2>/dev/null)
    if [ -z "$desc" ]; then
      desc="${y}! no description${r}"
    else
      (( max > 1 && ${#desc} > max )) && desc="${desc[1,max-1]}…"
      desc="${d}${desc}${r}"
    fi
    print -r -- "${line}${desc}"
  done
  for name in $broken; do
    line="${b}${(r:width:)name}${r}  "
    for a in $agents; do
      pad=$(( ${#a} + 1 ))
      [[ "${brokenin[$name]}" == *" $a "* ]] && line+="${x}●${r}${(l:pad:):-}" || line+="${d}○${r}${(l:pad:):-}"
    done
    print -r -- "${line}${x}! broken link${r}"
  done
  print -r -- "${d}● linked  ○ not linked${r}"
}

__dehv.skill() {
  [ $# -gt 0 ] || { __dehv.skill.help; return; }

  local link=false unlink=false delete=false restore=false all=false name= to=

  while [ $# -gt 0 ]; do
    case "$1" in
      -l|--link)    link=true ;;
      -u|--unlink)  unlink=true ;;
      -d|--delete)  delete=true ;;
      -r|--restore) restore=true ;;
      -a|--all)     all=true ;;
      -t|--to)      [ $# -ge 2 ] || { __dehv.misuse skill "-t needs the agents, like claude,cursor"; return; }
                    shift; to="$1" ;;
      -*)           __dehv.misuse skill "unknown option '$1'"; return ;;
      *)            [ -z "$name" ] || { __dehv.misuse skill "too many arguments"; return; }
                    name="$1" ;;
    esac
    shift
  done

  if [ "$all" = true ]; then
    if [ -n "$name$to" ] || [ "$link" = true ] || [ "$unlink" = true ] || [ "$delete" = true ] || [ "$restore" = true ]; then
      __dehv.misuse skill "-a goes alone"; return
    fi
    __dehv.skill.all; return
  fi

  [ -n "$name" ] || { __dehv.misuse skill "missing <name>"; return; }
  # The name becomes a path that gets moved around: keep it a single folder.
  case "$name" in
    */*|.*) __dehv.misuse skill "invalid name '$name'"; return ;;
  esac
  if [ "$delete" = true ] && { [ -n "$to" ] || [ "$link" = true ] || [ "$unlink" = true ] || [ "$restore" = true ]; }; then
    __dehv.misuse skill "-d goes alone"; return
  fi
  if [ "$unlink" = true ] && { [ "$link" = true ] || [ "$restore" = true ]; }; then
    __dehv.misuse skill "-u goes alone, or with -t"; return
  fi

  local -a known agents to_names chosen now
  local -A adir
  __dehv.skill.agents
  to_names=(${(s:,:)to})
  local a
  for a in $to_names; do
    (( ${known[(Ie)$a]} )) || { __dehv.misuse skill "unknown agent '$a': there is ${(j:, :)known}"; return; }
  done
  # Naming the agents is asking to link into them.
  [ -n "$to" ] && [ "$unlink" = false ] && [ "$restore" = false ] && link=true

  local skills=$(__dehv.skill.dir)
  if [ ! -d "$skills" ]; then
    __dehv.fail "${skills/#$HOME/~} does not exist"
    return
  fi

  local b c d g y x r; __dehv.style
  local folder="${d}${skills/#$HOME/~}/${r}${b}${name}${r}"
  local rc=0
  __dehv.skill.where; now=($reply)
  # The default agent, when it is installed.
  local -a first=(${(M)agents:#${known[1]}})

  if [ "$unlink" = true ]; then
    if [ ${#now} -eq 0 ]; then
      __dehv.fail "skill '$name' is not linked into any agent"
      return
    fi
    if [ -z "$to" ] && [ ${#now} -eq 1 ]; then
      chosen=($now)
    elif [ -z "$to" ] && { [ ! -t 0 ] || [ ! -t 1 ]; }; then
      __dehv.fail "skill '$name' is linked into ${(j:, :)now}" "name the ones to leave with -t"
      return
    else
      __dehv.skill.choose "Unlink $name from:" "unlink from" "$now" $now || return 1
    fi
    if [ ${#chosen} -eq 0 ] && [ $rc -eq 0 ]; then
      __dehv.noop "no agent ticked, nothing to do"
      return
    fi
    for a in $chosen; do
      if (( ! ${now[(Ie)$a]} )); then
        __dehv.fail "skill '$name' is not linked into $a"
        rc=1
        continue
      fi
      __dehv.skill.unlink "$a" || rc=1
    done
    return $rc
  fi

  if [ "$delete" = true ]; then
    if [ ! -d "$skills/$name" ]; then
      __dehv.fail "no skill '$name' in ${skills/#$HOME/~}"
      return
    fi
    if ! __dehv.finder 'delete (POSIX file (item 1 of argv) as alias)' "$skills/$name"; then
      __dehv.fail "Finder could not move '$name' to the Trash"
      return
    fi
    __dehv.ok "trashed  $folder"
    for a in $now; do __dehv.skill.unlink "$a"; done
    print -r -- "${d}  dehv skill -r $name brings it back${r}"
    return
  fi

  if [ "$restore" = true ]; then
    if [ -e "$skills/$name" ]; then
      __dehv.fail "skill '$name' already exists"
      return
    fi
    if ! __dehv.finder 'move item (item 1 of argv) of trash to (POSIX file (item 2 of argv) as alias)' "$name" "$skills"; then
      __dehv.fail "no '$name' in the Trash, or Finder could not move it"
      return
    fi
    __dehv.ok "restored $folder"
    # It is back either way: leaving the list just links it nowhere.
    if [ ${#agents} -gt 0 ] || [ -n "$to" ]; then
      __dehv.skill.choose "Link $name into (none to skip):" "link into" "$first" $agents || chosen=()
      for a in $chosen; do __dehv.skill.link "$a" || rc=1; done
    fi
    return $rc
  fi

  # Create, link, or both. The agents are asked first, so leaving the list
  # creates nothing.
  local listed=false
  if [ "$link" = true ]; then
    if [ ${#agents} -eq 0 ]; then
      __dehv.fail "no agent to link into" "none of ${(j:, :)known} is installed"
      return
    fi
    [ -z "$to" ] && [ -t 0 ] && [ -t 1 ] && listed=true
    local -a marked=($now)
    [ ${#marked} -gt 0 ] || marked=($first)
    __dehv.skill.choose "Link $name into:" "link into" "$marked" $agents || return 1
    if [ "$listed" = true ] && [ ${#chosen} -eq 0 ]; then
      __dehv.fail "no agent ticked" "-u takes it out of them"
      return
    fi
  fi

  local did=false
  if [ ! -e "$skills/$name" ]; then
    mkdir "$skills/$name" \
      && printf '%s\n' '---' "name: $name" 'description:' '---' > "$skills/$name/SKILL.md" || return
    __dehv.ok "created  $folder"
    did=true
  elif [ "$link" = false ]; then
    __dehv.fail "skill '$name' already exists"
    return
  fi

  if [ "$link" = true ]; then
    for a in $chosen; do
      (( ${now[(Ie)$a]} )) && continue
      __dehv.skill.link "$a" && did=true || rc=1
    done
    # The list shows where it is: what got unticked there comes out.
    if [ "$listed" = true ]; then
      for a in $now; do
        (( ${chosen[(Ie)$a]} )) && continue
        __dehv.skill.unlink "$a" && did=true || rc=1
      done
    fi
    [ "$did" = true ] || [ $rc -ne 0 ] || __dehv.noop "already linked there, nothing to do"
  fi
  return $rc
}

# ─── completion ───────────────────────────────────────────────────────────────
_dehv_skills() {
  local -a names
  names=("$(__dehv.skill.dir)"/*(N/:t))
  compadd -a names
}

_dehv_agents() {
  _sequence compadd - ${__dehv_agents%%\|*}
}

_dehv.skill() {
  _arguments '(-l --link -u --unlink -d --delete -r --restore -a --all)'{-l,--link}'[link it into the agents I pick]' \
             '(-l --link -u --unlink -d --delete -r --restore -a --all)'{-u,--unlink}'[take it out of the agents I pick, keeping it]' \
             '(-l --link -u --unlink -d --delete -r --restore -a --all)'{-d,--delete}'[unlink it everywhere and move it to the Trash]' \
             '(-l --link -u --unlink -d --delete -r --restore -a --all)'{-r,--restore}'[bring it back from the Trash and pick where to link it]' \
             '(-l --link -u --unlink -d --delete -r --restore -a --all)'{-a,--all}'[list my skills]' \
             '(-t --to -d --delete -a --all)'{-t,--to}'[the agents by name, instead of the list]:agents:_dehv_agents' \
             '1:name:_dehv_skills'
}
