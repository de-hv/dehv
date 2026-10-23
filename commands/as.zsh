# dehv as: my git profiles, and committing as one of them.
# Loaded by ../dehv.plugin.zsh, which says how a command registers itself.

__dehv_commands+=(as)
__dehv_about[as]='commit as one of my git profiles'
__dehv_rows+=(
  "as|<profile>|commit as that git profile in this repo"
  "|-s, --set|the same, picking the profile from a list"
  "|-a, --amend [<n>]|make it the author of the last <n> commits"
  "|-n, --new <profile>|create or edit a profile: asks name and email"
  "|-r, --remove <profile>|remove a profile"
  "|-l, --list|list my profiles"
  "|-c, --check|show which profile this repo commits as"
  "|-i, --init|create the profiles file, or keep or blank it"
  "|-x, --clear|remove every profile, after asking"
)

# ─── as ────────────────────────────────────────────────────────────────────────
# My git identities, one profile each (personal, a company, a client), kept per
# machine in ~/.config/dehv/profiles, out of this repo. The file is shaped like
# a git config and read and written with `git config -f`, so git does the
# quoting:
#
#   [profile "acme"]
#   	name = David Higuita
#   	email = david@acme.com
__dehv.as.help() {
  __dehv.usage as "dehv as [<profile>] [-s] [-a [<n>]] | -n <profile> | -r <profile> | -l | -c | -i | -x" \
    "Commit as one of my git profiles in this repo." \
    "<profile>|set this repo's user.name/user.email to that profile's" \
    "-s, --set|the same, picking the profile from a list" \
    "-a, --amend [<n>]|make it the author of the last <n>: rewritten, so only before pushing" \
    "-n, --new <profile>|create or edit it: asks the name and the email" \
    "-r, --remove <profile>|remove it" \
    "-l, --list|list my profiles, marking the one this repo commits as" \
    "-c, --check|show which profile this repo commits as" \
    "-i, --init|create the profiles file, or keep or blank the one there" \
    "-x, --clear|remove every profile, after asking"
}

__dehv.as.file() {
  print -r -- "${XDG_CONFIG_HOME:-$HOME/.config}/dehv/profiles"
}

# Creates the file, empty and readable only by me, unless it is there.
__dehv.as.touch() {
  local file=$(__dehv.as.file)
  [ -f "$file" ] && return
  mkdir -p "${file:h}" && : > "$file" && chmod 600 "$file"
}

# $1 a profile, $2 name or email.
__dehv.as.get() {
  git config -f "$(__dehv.as.file)" --get "profile.$1.$2" 2>/dev/null
}

# Sets $reply to my profiles, in the file's order.
__dehv.as.names() {
  reply=()
  local file=$(__dehv.as.file) key
  [ -f "$file" ] || return 0
  for key in ${(f)"$(git config -f "$file" --name-only --get-regexp '^profile\..*\.email$' 2>/dev/null)"}; do
    key=${key#profile.}
    reply+=("${key%.email}")
  done
}

# Sets $REPLY to the profile this repo commits as, or fails when it is none.
# Overwrites $reply.
__dehv.as.match() {
  REPLY=
  local name=$(git config user.name) email=$(git config user.email) p
  __dehv.as.names
  for p in $reply; do
    if [ "$(__dehv.as.get "$p" name)" = "$name" ] && [ "$(__dehv.as.get "$p" email)" = "$email" ]; then
      REPLY=$p
      return 0
    fi
  done
  return 1
}

# Sets $REPLY to a profile: the only one, or the one picked from the list.
__dehv.as.choose() {
  __dehv.as.names
  local -a names=($reply)
  if [ ${#names} -eq 0 ]; then
    __dehv.fail "no profiles yet" "'dehv as -n <profile>' or 'dehv as -i'"
    return
  fi
  if [ ${#names} -eq 1 ]; then
    REPLY=${names[1]}
    return
  fi
  if [ ! -t 0 ] || [ ! -t 1 ]; then
    __dehv.misuse as "name the profile: there is ${(j:, :)names}"; return
  fi

  local -a items
  local p w=0
  for p in $names; do (( ${#p} > w )) && w=${#p}; done
  for p in $names; do items+=("${(r:w:)p}  $(__dehv.as.get "$p" email)"); done
  __dehv.pick "${items[@]}" || return 1
  REPLY=${names[REPLY]}
}

__dehv.as() {
  [ $# -gt 0 ] || { __dehv.as.help; return; }

  local set=false amend=false check=false list=false init=false clear=false
  local new= remove= profile= n=

  while [ $# -gt 0 ]; do
    case "$1" in
      -s|--set)    set=true ;;
      -a|--amend)  amend=true ;;
      -c|--check)  check=true ;;
      -l|--list)   list=true ;;
      -i|--init)   init=true ;;
      -x|--clear)  clear=true ;;
      -n|--new)    [ $# -ge 2 ] || { __dehv.misuse as "-n needs the profile, like personal"; return; }
                   new="$2"; shift ;;
      -r|--remove) [ $# -ge 2 ] || { __dehv.misuse as "-r needs the profile"; return; }
                   remove="$2"; shift ;;
      -*)          __dehv.misuse as "unknown option '$1'"; return ;;
      <1->)        [ -z "$n" ] || { __dehv.misuse as "unexpected argument '$1'"; return; }
                   n="$1" ;;
      *)           [ -z "$profile" ] || { __dehv.misuse as "unexpected argument '$1'"; return; }
                   profile="$1" ;;
    esac
    shift
  done

  # -n, -r, -l, -c, -i and -x each go alone.
  local -a solo
  [ -n "$new" ]        && solo+=(-n)
  [ -n "$remove" ]     && solo+=(-r)
  [ "$list" = true ]   && solo+=(-l)
  [ "$check" = true ]  && solo+=(-c)
  [ "$init" = true ]   && solo+=(-i)
  [ "$clear" = true ]  && solo+=(-x)
  if [ ${#solo} -gt 1 ] || { [ ${#solo} -eq 1 ] && { [ "$set" = true ] || [ "$amend" = true ] || [ -n "$profile$n" ]; }; }; then
    __dehv.misuse as "${solo[1]} goes alone"; return
  fi
  if [ -n "$n" ] && [ "$amend" = false ]; then
    __dehv.misuse as "<n> goes with -a"; return
  fi

  # Lower-case letters, digits, - and _, and never only digits: those are <n>.
  local p
  for p in "$profile" "$new" "$remove"; do
    if [ -n "$p" ] && { [[ ! "$p" =~ '^[a-z0-9_-]+$' ]] || [[ "$p" == <-> ]]; }; then
      __dehv.misuse as "invalid profile '$p': lower-case letters, digits, - and _"; return
    fi
  done

  local b c d g y x r; __dehv.style
  local file=$(__dehv.as.file)
  local home=${file/#$HOME/\~}

  if [ "$init" = true ]; then
    if [ ! -f "$file" ]; then
      __dehv.as.touch || return
      __dehv.ok "created ${b}${home}${r}"
      return
    fi
    if [ ! -t 0 ] || [ ! -t 1 ]; then
      __dehv.misuse as "-i needs a terminal to choose what to do with $home"; return
    fi
    __dehv.as.names
    case ${#reply} in
      0) print -r -- "found ${b}${home}${r}, with no profiles" ;;
      1) print -r -- "found ${b}${home}${r} with 1 profile: ${c}${reply[1]}${r}" ;;
      *) print -r -- "found ${b}${home}${r} with ${#reply} profiles: ${c}${(j:, :)reply}${r}" ;;
    esac
    __dehv.pick "Use it" "Start blank" || return 1
    if [ "$REPLY" -eq 1 ]; then
      __dehv.noop "kept $home"
      return
    fi
    mv -f "$file" "$file.bak" && __dehv.as.touch || return
    __dehv.ok "blank ${b}${home}${r}"
    print -r -- "${d}  the old one is ${home}.bak${r}"
    return
  fi

  if [ "$clear" = true ]; then
    __dehv.as.names
    if [ ${#reply} -eq 0 ]; then
      __dehv.noop "no profiles, nothing to do"
      return
    fi
    if [ ! -t 0 ] || [ ! -t 1 ]; then
      __dehv.misuse as "-x needs a terminal to confirm"; return
    fi
    local count=${#reply} ask="Remove all ${#reply} profiles? [y/N] "
    (( count == 1 )) && ask="Remove the only profile? [y/N] "
    print -r -- "${c}${(j:, :)reply}${r}"
    if ! read -q "?$ask"; then
      echo
      __dehv.noop "kept them"
      return
    fi
    echo
    : > "$file" || return
    (( count == 1 )) && __dehv.ok "removed 1 profile" || __dehv.ok "removed $count profiles"
    return
  fi

  if [ -n "$new" ]; then
    if [ ! -t 0 ] || [ ! -t 1 ]; then
      __dehv.misuse as "-n needs a terminal to ask the name and the email"; return
    fi
    # Its current values to edit, or for a new one git's global identity.
    local name email verb=updated
    name=$(__dehv.as.get "$new" name)
    email=$(__dehv.as.get "$new" email)
    if [ -z "$email" ]; then
      verb=created
      [ -n "$name" ] || name=$(git config --global user.name)
      email=$(git config --global user.email)
    fi
    vared -p "  name  " name
    [ -n "$name" ] || { __dehv.fail "no name, nothing saved"; return; }
    vared -p "  email " email
    [[ "$email" == ?*@?* ]] || { __dehv.fail "'$email' is not an email, nothing saved"; return; }

    __dehv.as.touch || return
    git config -f "$file" "profile.$new.name" "$name" && git config -f "$file" "profile.$new.email" "$email" || return
    __dehv.ok "$verb ${c}${new}${r} ${d}· $name <$email>${r}"
    return
  fi

  if [ -n "$remove" ]; then
    if [ -z "$(__dehv.as.get "$remove" email)" ]; then
      __dehv.fail "no profile ${c}${remove}${r}"
      return
    fi
    git config -f "$file" --remove-section "profile.$remove" && __dehv.ok "removed ${c}${remove}${r}"
    return
  fi

  if [ "$list" = true ]; then
    __dehv.as.names
    local -a names=($reply) lines
    if [ ${#names} -eq 0 ]; then
      __dehv.fail "no profiles yet" "'dehv as -n <profile>' or 'dehv as -i'"
      return
    fi
    local cur=
    git rev-parse --git-dir >/dev/null 2>&1 && __dehv.as.match && cur=$REPLY

    if [ -t 1 ]; then
      local w=0 mark
      for p in $names; do (( ${#p} > w )) && w=${#p}; done
      for p in $names; do
        [ "$p" = "$cur" ] && mark="${g}●${r}" || mark=" "
        lines+=("$mark ${b}${(r:w:)p}${r}  $(__dehv.as.get "$p" name) ${d}<$(__dehv.as.get "$p" email)>${r}")
      done
      __dehv.box profiles "${#names}" "${lines[@]}"
    else
      for p in $names; do
        print -r -- "$p"$'\t'"$(__dehv.as.get "$p" name)"$'\t'"$(__dehv.as.get "$p" email)"
      done
    fi
    return
  fi

  git rev-parse --git-dir >/dev/null 2>&1 || { __dehv.fail "not in a git repo"; return; }

  local repo="$(git config user.name) <$(git config user.email)>"

  if [ "$check" = true ]; then
    [ "$repo" != " <>" ] || repo="none set"
    echo "repo   $repo"
    if __dehv.as.match; then
      print -r -- "as     ${c}${REPLY}${r}"
    else
      print -r -- "as     ${y}none of my profiles${r}"
      return 1
    fi
    return
  fi

  # A profile alone sets it; with -a alone it only rewrites the authors.
  [ -n "$profile" ] && [ "$amend" = false ] && set=true

  # Without a name -a keeps the repo's profile; otherwise one is picked.
  if [ -z "$profile" ]; then
    if [ "$set" = false ] && __dehv.as.match; then
      profile=$REPLY
    else
      __dehv.as.choose || return
      profile=$REPLY
    fi
  fi

  local name email
  name=$(__dehv.as.get "$profile" name)
  email=$(__dehv.as.get "$profile" email)
  if [ -z "$email" ]; then
    __dehv.fail "no profile ${c}${profile}${r}" "'dehv as -n $profile' creates it"
    return
  fi
  local mine="$name <$email>"

  if [ "$set" = true ]; then
    if [ "$repo" = "$mine" ]; then
      __dehv.noop "already as $profile, nothing to do"
    else
      local from=
      __dehv.as.match && from=$REPLY
      git config --local user.name "$name" && git config --local user.email "$email" || return
      if [ "$repo" = " <>" ]; then
        __dehv.ok "${b}${profile}${r} ${d}· ${mine}${r}"
      else
        __dehv.ok "${d}${from:-$repo} →${r} ${b}${profile}${r} ${d}· ${mine}${r}"
      fi
    fi
  fi

  if [ "$amend" = true ]; then
    n=${n:-1}
    local total
    total=$(git rev-list --count HEAD 2>/dev/null) || { __dehv.fail "no commits here yet"; return; }
    if (( n > total )); then
      __dehv.fail "only $total commits here"
      return
    fi

    local -a authors others
    authors=("${(@f)$(git log -n $n --format='%an <%ae>')}")
    others=(${authors:#$mine})
    if [ ${#others} -eq 0 ]; then
      (( n == 1 )) && __dehv.noop "the last commit is already as $profile, nothing to do" \
                   || __dehv.noop "the last $n commits are already as $profile, nothing to do"
      return
    fi

    if (( n == 1 )); then
      git commit --amend -q --author="$mine" || return
    else
      # git's own account of the rebase only when it fails.
      local -a base
      (( n == total )) && base=(--root) || base=("HEAD~$n")
      local out
      if ! out=$(git rebase "${base[@]}" --exec "git commit --amend --no-edit -q --author=${(qq)mine}" 2>&1); then
        print -r -- "$out" >&2
        __dehv.fail "git could not rewrite them" "'git rebase --abort' undoes it"
        return
      fi
    fi

    git --no-pager log -n $n --format="${g}✓${r} ${y}%h${r} %s"
    (( n == 1 )) && print -r -- "${d}1 commit now as $profile${r}" \
                 || print -r -- "${d}$n commits now as $profile${r}"
  fi
}

# ─── completion ───────────────────────────────────────────────────────────────
_dehv_profiles() {
  local -a reply
  __dehv.as.names
  compadd -a reply
}

_dehv.as() {
  _arguments '(-n --new -r --remove -l --list -c --check -i --init -x --clear)'{-s,--set}'[set this repo'\''s user.name/user.email]' \
             '(-n --new -r --remove -l --list -c --check -i --init -x --clear)'{-a,--amend}'[make the profile the author of the last commits]' \
             '(- *)'{-n,--new}'[create or edit a profile]:profile:_dehv_profiles' \
             '(- *)'{-r,--remove}'[remove a profile]:profile:_dehv_profiles' \
             '(- *)'{-l,--list}'[list my profiles]' \
             '(- *)'{-c,--check}'[show which profile this repo commits as]' \
             '(- *)'{-i,--init}'[create the profiles file, or keep or blank it]' \
             '(- *)'{-x,--clear}'[remove every profile, after asking]' \
             '*:profile:_dehv_profiles'
}
