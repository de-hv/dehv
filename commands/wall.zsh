# dehv wall: my wallpapers, kept in a GitHub repo and synced with a folder.
# Loaded by ../dehv.plugin.zsh, which says how a command registers itself.

__dehv_commands+=(wall)
__dehv_about[wall]='add wallpapers to a GitHub repo and sync it with a folder'

# ─── wall ──────────────────────────────────────────────────────────────────────
# Each repo is a GitHub repo of images and the folder it is cloned into, kept
# per machine in ~/.config/dehv/walls, out of this repo, with the one in use.
# Shaped like a git config, as the profiles of `dehv as`:
#
#   [wall]
#   	use = personal
#   [repo "personal"]
#   	url = git@github.com:de-hv/wallpapers.git
#   	dir = ~/Pictures/wallpapers
#   	cloned = true
#
# cloned is written once its folder is a clone: a folder never cloned is cloned
# right away, but one that was and is now gone or empty only after asking. Nor
# does -s remove images, here or on GitHub, without asking.
__dehv.wall.help() {
  __dehv.usage wall "dehv wall <url> [<name>] [-t <repo>] | -s [-t <repo> | -a] | -o [-t <repo>] | -u [<repo>] | -n <repo> | -r <repo> | -l | -i | -x" \
    "Add images to a GitHub repo of wallpapers, and keep a folder in sync." \
    "<url> [<name>]|add it to the repo in use: download, commit, push" \
    "-t, --to <repo>|use that repo just this once, with <url>, -s or -o" \
    "-s, --sync|commit its folder's changes, pull and push; clones it first" \
    "-s -a, --all|the same for every repo, one after the other" \
    "-u, --use [<repo>]|choose the repo the rest use, from a list without one" \
    "-n, --new <repo>|add or edit a repo: asks its GitHub url and its folder" \
    "-r, --remove <repo>|forget a repo; its folder stays" \
    "-l, --list|list my repos, marking the one in use" \
    "-o, --open|open the folder of the repo in use" \
    "-i, --init|create the repos file, or keep or blank it" \
    "-x, --clear|forget every repo, after asking"
}

__dehv.wall.file() {
  print -r -- "${XDG_CONFIG_HOME:-$HOME/.config}/dehv/walls"
}

# Creates the file, empty and readable only by me, unless it is there.
__dehv.wall.touch() {
  local file=$(__dehv.wall.file)
  [ -f "$file" ] && return
  mkdir -p "${file:h}" && : > "$file" && chmod 600 "$file"
}

# $1 a repo, $2 url or dir.
__dehv.wall.get() {
  git config -f "$(__dehv.wall.file)" --get "repo.$1.$2" 2>/dev/null
}

# The folder of repo $1, with ~ expanded.
__dehv.wall.dir() {
  local dir=$(__dehv.wall.get "$1" dir)
  print -r -- "${dir/#\~/$HOME}"
}

# owner/name out of a GitHub url, ssh or https.
__dehv.wall.short() {
  local url=${1%.git}
  print -r -- "${${url%/*}##*[:/]}/${url##*[:/]}"
}

# Sets $reply to my repos, in the file's order.
__dehv.wall.names() {
  reply=()
  local file=$(__dehv.wall.file) key
  [ -f "$file" ] || return 0
  for key in ${(f)"$(git config -f "$file" --name-only --get-regexp '^repo\..*\.url$' 2>/dev/null)"}; do
    key=${key#repo.}
    reply+=("${key%.url}")
  done
}

# Sets $REPLY to the repo in use: the one chosen with -u, or the only one.
# With $1 = ask, when there are several and none in use, the one picked from
# the list, which stays in use as with -u.
__dehv.wall.current() {
  local file=$(__dehv.wall.file)
  REPLY=$(git config -f "$file" --get wall.use 2>/dev/null)
  [ -n "$REPLY" ] && [ -n "$(__dehv.wall.get "$REPLY" url)" ] && return
  __dehv.wall.names
  case ${#reply} in
    0) REPLY=; __dehv.fail "no repos yet" "'dehv wall -n <repo>' adds one" ;;
    1) REPLY=${reply[1]} ;;
    *)
      REPLY=
      if [ "$1" != ask ] || [ ! -t 0 ] || [ ! -t 1 ]; then
        __dehv.fail "no repo in use" "'dehv wall -u' chooses one"
        return
      fi
      local b c d g y x r; __dehv.style
      print -r -- "${d}no repo in use, pick one:${r}"
      __dehv.wall.choose || return 1
      git config -f "$file" wall.use "$REPLY" || return
      __dehv.ok "${b}${REPLY}${r} ${d}· in use from now on${r}"
      ;;
  esac
}

# Sets $REPLY to the number of images in folder $1.
__dehv.wall.count() {
  setopt localoptions extendedglob
  local -a files=("$1"/**/*.(#i)(jpg|jpeg|png|webp|gif|heic|heif|avif|bmp|tif|tiff|svg)(N.))
  REPLY=${#files}
}

# Runs the command after $1 with a spinner and $1 on stderr while it lasts,
# when stderr is a terminal. Sets $REPLY to its output, and returns its status:
# 130 when stopped with ctrl-c, which stops the command too.
__dehv.wall.run() {
  setopt localoptions localtraps nomonitor
  local msg="$1"; shift
  local b c d g y x r; __dehv.style 2
  local out=$(mktemp) i=0 spin='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏' s stop=false
  # Never a login prompt: it would wait unseen behind the spinner.
  GIT_TERMINAL_PROMPT=0 GIT_SSH_COMMAND=${GIT_SSH_COMMAND:-ssh -o BatchMode=yes} "$@" < /dev/null > "$out" 2>&1 &
  local pid=$!
  trap 'stop=true; kill $pid 2>/dev/null' INT
  while kill -0 $pid 2>/dev/null; do
    [ -t 2 ] && print -n -- "\r${c}${spin[i % 10 + 1]}${r} ${d}${msg}…${r}" >&2
    (( i++ ))
    sleep 0.08
  done
  wait $pid; s=$?
  [ -t 2 ] && print -n -- $'\r\e[K' >&2
  REPLY=$(<"$out")
  rm -f "$out"
  [ "$stop" = true ] && return 130
  return $s
}

# Fails with $2, and why git failed in words when its output, $3, says a usual
# reason; otherwise with that output. Nothing more when $1, the status, is 130.
__dehv.wall.gitfail() {
  (( $1 == 130 )) && return 130
  case "$3" in
    *"terminal prompts disabled"*|*"could not read Username"*)
      __dehv.fail "$2" "GitHub asks for a login over https: 'gh auth setup-git', or an ssh url" ;;
    *"Permission denied (publickey)"*|*"Host key verification failed"*)
      __dehv.fail "$2" "GitHub refused the ssh key: 'ssh -T git@github.com' says why" ;;
    *"Repository not found"*|*"does not appear to be a git repository"*)
      __dehv.fail "$2" "there is no such repo, or it is not mine: check its url" ;;
    *)
      print -r -- "$3" >&2
      __dehv.fail "$2" ;;
  esac
}

# Sets $REPLY to what the `git diff --name-status` in $1 did, as "2 added,
# 1 removed", or with $2 = verb as "add 2, remove 1", the commit's way.
__dehv.wall.changes() {
  local -A n=()
  local line
  for line in ${(f)1}; do
    case "$line" in
      A*) (( n[A]++ )) ;;
      D*) (( n[D]++ )) ;;
      R*) (( n[R]++ )) ;;
      *)  (( n[M]++ )) ;;
    esac
  done
  local -a parts
  local k
  for k in A D R M; do
    (( n[$k] )) || continue
    if [ "$2" = verb ]; then
      case $k in A) parts+=("add ${n[$k]}") ;; D) parts+=("remove ${n[$k]}") ;; R) parts+=("rename ${n[$k]}") ;; M) parts+=("change ${n[$k]}") ;; esac
    else
      case $k in A) parts+=("${n[$k]} added") ;; D) parts+=("${n[$k]} removed") ;; R) parts+=("${n[$k]} renamed") ;; M) parts+=("${n[$k]} changed") ;; esac
    fi
  done
  REPLY=${(j:, :)parts}
}

# Says $1 with a yellow mark, the files after $2 (the first 10) and asks $2
# [y/N]. Returns 1 for a no, and 2 without a terminal to ask in, after failing.
__dehv.wall.confirm() {
  local what="$1" ask="$2"; shift 2
  local b c d g y x r; __dehv.style
  if [ ! -t 0 ] || [ ! -t 1 ]; then
    __dehv.fail "$what" "run it in a terminal to confirm"
    return 2
  fi
  print -r -- "${y}!${r} $what"
  local f
  for f in $argv[1,10]; do print -r -- "  ${x}-${r} $f"; done
  (( $# > 10 )) && print -r -- "  ${d}… and $(( $# - 10 )) more${r}"
  if ! read -q "?$ask [y/N] "; then
    echo
    return 1
  fi
  echo
}

# Makes sure repo $1 is cloned into its folder, cloning it if the folder is not
# there or empty, and that git ignores the Finder's .DS_Store in it. Sets
# $REPLY to cloned when it did. A folder once cloned, marked in the repos file,
# is cloned again only after asking.
__dehv.wall.ready() {
  local url=$(__dehv.wall.get "$1" url) dir=$(__dehv.wall.dir "$1")
  local home=${dir/#$HOME/\~}
  local -a inside=("$dir"/*(DN))
  REPLY=
  if [ ! -e "$dir" ] || [ ${#${inside:#*/.DS_Store}} -eq 0 ]; then
    # dehv wall never removes it nor empties it: something else did.
    if [ "$(__dehv.wall.get "$1" cloned)" = true ]; then
      local gone="is gone"
      [ -e "$dir" ] && gone="is empty"
      __dehv.wall.confirm "$home was cloned and now $gone" "Clone $(__dehv.wall.short "$url") into it again?"
      local s=$?
      (( s == 1 )) && __dehv.noop "left $home as it is"
      (( s == 0 )) || return 1
    fi
    # Into a folder next to it, moved in place once complete, so a clone
    # that fails or is stopped leaves nothing half done.
    local tmp="${dir:h}/.${dir:t}.cloning"
    mkdir -p "${dir:h}" && rm -rf "$tmp" || return
    __dehv.wall.run "cloning $(__dehv.wall.short "$url")" git clone -q "$url" "$tmp" || {
      local s=$?
      rm -rf "$tmp"
      __dehv.wall.gitfail $s "could not clone $(__dehv.wall.short "$url")" "$REPLY"
      return
    }
    rm -f "$dir/.DS_Store"; rmdir "$dir" 2>/dev/null
    mv "$tmp" "$dir" || return
    REPLY=cloned
  elif [ "$(git -C "$dir" rev-parse --show-toplevel 2>/dev/null)" != "${dir:A}" ]; then
    if [ -e "$dir/.git" ]; then
      __dehv.fail "$home has a .git that is not a working repo" "a clone stopped halfway? remove that .git and try again"
    else
      __dehv.fail "$home is not a clone of the repo" "move it, or give the repo another folder with 'dehv wall -n $1'"
    fi
    return
  elif [ "$(git -C "$dir" remote get-url origin 2>/dev/null)" != "$url" ]; then
    # Its url edited with -n, like https to ssh: the clone follows.
    git -C "$dir" remote set-url origin "$url" || return
  fi
  local exclude=$(git -C "$dir" rev-parse --git-path info/exclude)
  [[ $exclude == /* ]] || exclude="$dir/$exclude"
  grep -qx '.DS_Store' "$exclude" 2>/dev/null || { mkdir -p "${exclude:h}" && print -r -- '.DS_Store' >> "$exclude"; }
  [ "$(__dehv.wall.get "$1" cloned)" = true ] || git config -f "$(__dehv.wall.file)" "repo.$1.cloned" true
}

# Joins repo $1's commits with GitHub's, theirs first, and pushes. Sets $reply
# to what went up and what came down, each as __dehv.wall.changes says it.
__dehv.wall.publish() {
  local dir=$(__dehv.wall.dir "$1")
  local -a git=(git -C "$dir")
  reply=("" "")

  __dehv.wall.run "asking GitHub" $git fetch -q origin || {
    __dehv.wall.gitfail $? "could not reach GitHub for $1" "$REPLY"
    return
  }

  local branch=$($git symbolic-ref --short HEAD) out
  local remote=origin/$branch base
  local head=true
  $git rev-parse -q --verify HEAD >/dev/null || head=false

  if $git rev-parse -q --verify "$remote" >/dev/null; then
    if [ "$head" = false ]; then
      # Nothing of mine yet: start from GitHub's.
      __dehv.wall.changes "$($git diff --name-status $($git hash-object -t tree /dev/null) "$remote")"
      reply[2]=$REPLY
      $git checkout -q -B "$branch" "$remote" || return
      return
    fi
    if ! $git merge-base --is-ancestor "$remote" HEAD; then
      # Images removed on GitHub: removed here too, only once said so.
      local -a gone=(${(f)"$($git diff --name-only -M --diff-filter=D HEAD..."$remote")"})
      if (( ${#gone} )); then
        __dehv.wall.confirm "${#gone} removed on GitHub" "Remove them in ${dir/#$HOME/~} too?" $gone
        local s=$?
        (( s == 1 )) && __dehv.noop "kept them: nothing came down nor went up, and 'dehv wall -s' asks again"
        (( s == 0 )) || return 1
      fi
      __dehv.wall.changes "$($git diff --name-status -M HEAD..."$remote")"
      reply[2]=$REPLY
      # A folder renamed on GitHub takes along the images added to it here.
      if ! out=$($git -c merge.directoryRenames=true rebase -q "$remote" 2>&1); then
        local -a both=(${(f)"$($git diff --name-only --diff-filter=U)"})
        $git rebase --abort 2>/dev/null
        if [ ${#both} -eq 0 ]; then
          print -r -- "$out" >&2
          __dehv.fail "could not join mine with GitHub's"
        else
          __dehv.fail "${(j:, :)both} changed both here and on GitHub" "fix it in ${dir/#$HOME/~} with git, then 'dehv wall -s'"
        fi
        return
      fi
    fi
    base=$remote
  else
    # Empty on GitHub: everything goes up.
    [ "$head" = true ] || return 0
    base=$($git hash-object -t tree /dev/null)
  fi

  [ -n "$($git diff --name-only "$base" HEAD)" ] || return 0
  __dehv.wall.changes "$($git diff --name-status -M "$base" HEAD)"
  reply[1]=$REPLY
  __dehv.wall.run "pushing to GitHub" $git push -q -u origin HEAD || {
    local s=$?
    (( s == 130 )) || __dehv.noop "committed here: 'dehv wall -s' pushes it later" >&2
    __dehv.wall.gitfail $s "could not push $1" "$REPLY"
    return
  }
}

# Downloads $2 into repo $1 as $3, or the name in the url, then commits and
# pushes it.
__dehv.wall.add() {
  setopt localoptions extendedglob
  local repo="$1" url="$2" name="$3"
  local b c d g y x r; __dehv.style
  local dir=$(__dehv.wall.dir "$repo")

  # The last part of its path, decoded: %20 is a space.
  if [ -z "$name" ]; then
    name=${${url%%[?#]*}:t}
    name=$(printf '%b' "${name//(#m)%[0-9A-Fa-f](#c2)/\\x${MATCH:1}}")
    [[ -z "$name" || "$name" == .* ]] && name=wallpaper
  fi

  local tmp=$(mktemp) code
  __dehv.wall.run "downloading ${name}" curl -sSL --max-time 120 -A 'Mozilla/5.0' -o "$tmp" -w '%{http_code}' "$url" || {
    local s=$?
    rm -f "$tmp"
    (( s == 130 )) && return 130
    __dehv.fail "could not download it" "${REPLY##*curl: \([0-9]#\) }"
    return
  }
  code=$REPLY
  if [[ $code != 2?? ]]; then
    rm -f "$tmp"
    __dehv.fail "could not download it" "the server answered $code"
    return
  fi

  local mime=$(file -b --mime-type "$tmp") ext
  case "$mime" in
    image/jpeg)        ext=jpg ;;
    image/png)         ext=png ;;
    image/webp)        ext=webp ;;
    image/gif)         ext=gif ;;
    image/heic)        ext=heic ;;
    image/heif)        ext=heif ;;
    image/avif)        ext=avif ;;
    image/*bmp)        ext=bmp ;;
    image/tiff)        ext=tif ;;
    image/svg+xml)     ext=svg ;;
    *)
      rm -f "$tmp"
      __dehv.fail "it is not an image" "the server sent $mime"
      return
      ;;
  esac
  [[ ${name:e:l} == (jpg|jpeg|png|webp|gif|heic|heif|avif|bmp|tif|tiff|svg) ]] || name+=".$ext"

  __dehv.wall.ready "$repo" || { rm -f "$tmp"; return; }

  if [ -e "$dir/$name" ]; then
    if cmp -s "$tmp" "$dir/$name"; then
      rm -f "$tmp"
      __dehv.noop "$name is already in $repo, nothing to do"
    else
      rm -f "$tmp"
      __dehv.fail "there is already a $name in $repo" "name it: 'dehv wall <url> <name>'"
    fi
    return
  fi

  mv "$tmp" "$dir/$name" && chmod 644 "$dir/$name" || return
  local size=$(stat -f %z "$dir/$name") dims
  dims=$(sips -g pixelWidth -g pixelHeight "$dir/$name" 2>/dev/null | awk '/pixelWidth/ { w = $2 } /pixelHeight/ { h = $2 } END { if (w && h) print w "×" h }')
  if (( size >= 1048576 )); then
    size=$(printf '%.1f MB' $(( size / 1048576.0 )))
  else
    size="$(( (size + 1023) / 1024 )) KB"
  fi
  __dehv.ok "$name ${d}·${dims:+ $dims ·} $size${r}"

  local -a git=(git -C "$dir")
  $git add -- "$name" && $git commit -q -m "add $name" -- "$name" || return
  __dehv.wall.publish "$repo" || return
  print -r -- "${g}✓${r} ${y}$($git rev-parse --short HEAD)${r} add $name ${d}→ $repo${r}"
}

# Commits what changed in repo $1's folder, joins it with GitHub's and pushes;
# the first time, only clones it.
__dehv.wall.sync() {
  local repo="$1"
  local b c d g y x r; __dehv.style
  local dir=$(__dehv.wall.dir "$repo")
  local home=${dir/#$HOME/\~}

  __dehv.wall.ready "$repo" || return
  if [ "$REPLY" = cloned ]; then
    __dehv.wall.count "$dir"
    __dehv.ok "cloned ${b}${repo}${r} ${d}· $REPLY images · $home${r}"
    return
  fi

  local -a git=(git -C "$dir")
  $git add -A || return
  if ! $git diff --cached --quiet; then
    # Images removed here: removed on GitHub too, only once said so.
    local -a gone=(${(f)"$($git diff --cached --name-only -M --diff-filter=D)"})
    if (( ${#gone} )); then
      __dehv.wall.confirm "${#gone} removed in $home" "Remove them on GitHub too?" $gone
      local s=$?
      if (( s )); then
        $git reset -q
        (( s == 1 )) && __dehv.noop "synced nothing: they are still on GitHub, and 'git -C $home restore .' brings them back"
        return 1
      fi
    fi
    __dehv.wall.changes "$($git diff --cached --name-status -M)" verb
    $git commit -q -m "sync: $REPLY" || return
  fi

  __dehv.wall.publish "$repo" || return
  local up=${reply[1]} down=${reply[2]}
  if [ -z "$up$down" ]; then
    __dehv.noop "$repo already in sync, nothing to do"
    return
  fi
  __dehv.ok "${b}${repo}${r} ${d}·${up:+ ↑ $up ·}${down:+ ↓ $down ·} $home${r}"
}

# Sets $REPLY to a repo: the only one, or the one picked from the list.
__dehv.wall.choose() {
  __dehv.wall.names
  local -a names=($reply)
  if [ ${#names} -eq 0 ]; then
    __dehv.fail "no repos yet" "'dehv wall -n <repo>' adds one"
    return
  fi
  if [ ${#names} -eq 1 ]; then
    REPLY=${names[1]}
    return
  fi
  if [ ! -t 0 ] || [ ! -t 1 ]; then
    __dehv.misuse wall "name the repo: there is ${(j:, :)names}"; return
  fi

  local -a items
  local p w=0
  for p in $names; do (( ${#p} > w )) && w=${#p}; done
  for p in $names; do items+=("${(r:w:)p}  $(__dehv.wall.short "$(__dehv.wall.get "$p" url)")"); done
  __dehv.pick "${items[@]}" || return 1
  REPLY=${names[REPLY]}
}

__dehv.wall() {
  [ $# -gt 0 ] || { __dehv.wall.help; return; }

  local sync=false all=false open=false use=false list=false init=false clear=false
  local to= pick= new= remove= src= name=

  while [ $# -gt 0 ]; do
    case "$1" in
      -s|--sync)   sync=true ;;
      -a|--all)    all=true ;;
      -o|--open)   open=true ;;
      -l|--list)   list=true ;;
      -i|--init)   init=true ;;
      -x|--clear)  clear=true ;;
      -u|--use)    use=true
                   if [ $# -ge 2 ] && [[ "$2" != -* ]]; then pick="$2"; shift; fi ;;
      -t|--to)     [ $# -ge 2 ] || { __dehv.misuse wall "-t needs the repo, like personal"; return; }
                   to="$2"; shift ;;
      -n|--new)    [ $# -ge 2 ] || { __dehv.misuse wall "-n needs the repo, like personal"; return; }
                   new="$2"; shift ;;
      -r|--remove) [ $# -ge 2 ] || { __dehv.misuse wall "-r needs the repo"; return; }
                   remove="$2"; shift ;;
      -*)          __dehv.misuse wall "unknown option '$1'"; return ;;
      *)           if [ -z "$src" ]; then
                     src="$1"
                   elif [ -z "$name" ]; then
                     name="$1"
                   else
                     __dehv.misuse wall "unexpected argument '$1'"; return
                   fi ;;
    esac
    shift
  done

  # -u, -n, -r, -l, -i and -x each go alone; <url>, -s and -o one at a time.
  local -a solo acts
  [ "$use" = true ]    && solo+=(-u)
  [ -n "$new" ]        && solo+=(-n)
  [ -n "$remove" ]     && solo+=(-r)
  [ "$list" = true ]   && solo+=(-l)
  [ "$init" = true ]   && solo+=(-i)
  [ "$clear" = true ]  && solo+=(-x)
  [ -n "$src" ]        && acts+=("<url>")
  [ "$sync" = true ]   && acts+=(-s)
  [ "$open" = true ]   && acts+=(-o)
  if [ ${#solo} -gt 1 ] || { [ ${#solo} -eq 1 ] && { [ -n "${acts}${to}" ] || [ "$all" = true ]; }; }; then
    __dehv.misuse wall "${solo[1]} goes alone"; return
  fi
  if [ ${#acts} -gt 1 ]; then
    __dehv.misuse wall "${acts[1]} and ${acts[2]} go apart"; return
  fi
  if [ "$all" = true ] && [ "$sync" = false ]; then
    __dehv.misuse wall "-a goes with -s"; return
  fi
  if [ "$all" = true ] && [ -n "$to" ]; then
    __dehv.misuse wall "-a and -t go apart"; return
  fi
  if [ -n "$to" ] && [ ${#acts} -eq 0 ]; then
    __dehv.misuse wall "-t goes with <url>, -s or -o"; return
  fi
  if [ -n "$src" ] && [[ "$src" != http(s|)://?* ]]; then
    __dehv.misuse wall "'$src' is not an http(s) url"; return
  fi
  if [ -n "$name" ] && [[ "$name" == */* || "$name" == .* ]]; then
    __dehv.misuse wall "invalid name '$name': a file name, without / nor a leading ."; return
  fi

  # Lower-case letters, digits, - and _.
  local p
  for p in "$to" "$pick" "$new" "$remove"; do
    if [ -n "$p" ] && [[ ! "$p" =~ '^[a-z0-9_-]+$' ]]; then
      __dehv.misuse wall "invalid repo '$p': lower-case letters, digits, - and _"; return
    fi
  done

  local b c d g y x r; __dehv.style
  local file=$(__dehv.wall.file)
  local home=${file/#$HOME/\~}

  if [ "$init" = true ]; then
    if [ ! -f "$file" ]; then
      __dehv.wall.touch || return
      __dehv.ok "created ${b}${home}${r}"
      return
    fi
    if [ ! -t 0 ] || [ ! -t 1 ]; then
      __dehv.misuse wall "-i needs a terminal to choose what to do with $home"; return
    fi
    __dehv.wall.names
    case ${#reply} in
      0) print -r -- "found ${b}${home}${r}, with no repos" ;;
      1) print -r -- "found ${b}${home}${r} with 1 repo: ${c}${reply[1]}${r}" ;;
      *) print -r -- "found ${b}${home}${r} with ${#reply} repos: ${c}${(j:, :)reply}${r}" ;;
    esac
    __dehv.pick "Use it" "Start blank" || return 1
    if [ "$REPLY" -eq 1 ]; then
      __dehv.noop "kept $home"
      return
    fi
    mv -f "$file" "$file.bak" && __dehv.wall.touch || return
    __dehv.ok "blank ${b}${home}${r}"
    print -r -- "${d}  the old one is ${home}.bak${r}"
    return
  fi

  if [ "$clear" = true ]; then
    __dehv.wall.names
    if [ ${#reply} -eq 0 ]; then
      __dehv.noop "no repos, nothing to do"
      return
    fi
    if [ ! -t 0 ] || [ ! -t 1 ]; then
      __dehv.misuse wall "-x needs a terminal to confirm"; return
    fi
    local count=${#reply} ask="Forget all ${#reply} repos? Their folders stay. [y/N] "
    (( count == 1 )) && ask="Forget the only repo? Its folder stays. [y/N] "
    print -r -- "${c}${(j:, :)reply}${r}"
    if ! read -q "?$ask"; then
      echo
      __dehv.noop "kept them"
      return
    fi
    echo
    : > "$file" || return
    (( count == 1 )) && __dehv.ok "forgot 1 repo" || __dehv.ok "forgot $count repos"
    return
  fi

  if [ -n "$new" ]; then
    if [ ! -t 0 ] || [ ! -t 1 ]; then
      __dehv.misuse wall "-n needs a terminal to ask the url and the folder"; return
    fi
    local url dir verb=updated
    url=$(__dehv.wall.get "$new" url)
    dir=$(__dehv.wall.get "$new" dir)
    [ -n "$url" ] || verb=added
    vared -p "  url    " url
    # owner/name alone is a GitHub repo, over the protocol gh uses, or ssh
    # when there is a key for it.
    if [[ "$url" =~ '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$' ]]; then
      local proto=$(gh config get git_protocol 2>/dev/null)
      if [ -z "$proto" ]; then
        local -a keys=(~/.ssh/id_*.pub(N))
        (( ${#keys} )) && proto=ssh || proto=https
      fi
      [ "$proto" = ssh ] && url="git@github.com:${url%.git}.git" || url="https://github.com/${url%.git}.git"
    fi
    [[ "$url" == (git@*:*|https://*|ssh://*|file://*) ]] || { __dehv.fail "'$url' is not a git url, nothing saved"; return; }
    [ -n "$dir" ] || dir="~/Pictures/${${url%.git}##*[:/]}"
    vared -p "  folder " dir
    [ -n "$dir" ] || { __dehv.fail "no folder, nothing saved"; return; }
    dir=${dir/#$HOME/\~}

    __dehv.wall.touch || return
    # Another folder has not been cloned yet.
    [ "$dir" = "$(__dehv.wall.get "$new" dir)" ] || git config -f "$file" --unset "repo.$new.cloned"
    git config -f "$file" "repo.$new.url" "$url" && git config -f "$file" "repo.$new.dir" "$dir" || return
    # The first one is the one in use.
    [ -n "$(git config -f "$file" --get wall.use)" ] || git config -f "$file" wall.use "$new"
    __dehv.ok "$verb ${c}${new}${r} ${d}· $(__dehv.wall.short "$url") → $dir${r}"
    [ -e "$(__dehv.wall.dir "$new")" ] || print -r -- "${d}  'dehv wall -s' clones it${r}"
    return
  fi

  if [ -n "$remove" ]; then
    if [ -z "$(__dehv.wall.get "$remove" url)" ]; then
      __dehv.fail "no repo ${c}${remove}${r}"
      return
    fi
    local dir=$(__dehv.wall.get "$remove" dir)
    git config -f "$file" --remove-section "repo.$remove" || return
    [ "$(git config -f "$file" --get wall.use)" = "$remove" ] && git config -f "$file" --unset wall.use
    __dehv.ok "forgot ${c}${remove}${r} ${d}· $dir stays${r}"
    return
  fi

  if [ "$list" = true ]; then
    __dehv.wall.names
    local -a names=($reply) lines
    if [ ${#names} -eq 0 ]; then
      __dehv.fail "no repos yet" "'dehv wall -n <repo>' or 'dehv wall -i'"
      return
    fi
    local cur=
    __dehv.wall.current 2>/dev/null && cur=$REPLY

    if [ -t 1 ]; then
      local w=0 sw=0 dw=0 mark short dir count
      for p in $names; do
        (( ${#p} > w )) && w=${#p}
        short=$(__dehv.wall.short "$(__dehv.wall.get "$p" url)"); (( ${#short} > sw )) && sw=${#short}
        dir=$(__dehv.wall.get "$p" dir); (( ${#dir} > dw )) && dw=${#dir}
      done
      for p in $names; do
        [ "$p" = "$cur" ] && mark="${g}●${r}" || mark=" "
        short=$(__dehv.wall.short "$(__dehv.wall.get "$p" url)")
        dir=$(__dehv.wall.get "$p" dir)
        if [ -d "$(__dehv.wall.dir "$p")" ]; then
          __dehv.wall.count "$(__dehv.wall.dir "$p")"
          (( REPLY == 1 )) && count="1 image" || count="$REPLY images"
        elif [ "$(__dehv.wall.get "$p" cloned)" = true ]; then
          count="${x}gone${r}${d}"
        else
          count="not cloned"
        fi
        lines+=("$mark ${b}${(r:w:)p}${r}  ${(r:sw:)short}  ${d}${(r:dw:)dir}  ${count}${r}")
      done
      __dehv.box repos "${#names}" "${lines[@]}"
    else
      for p in $names; do
        print -r -- "$p"$'\t'"$(__dehv.wall.get "$p" url)"$'\t'"$(__dehv.wall.get "$p" dir)"
      done
    fi
    return
  fi

  if [ "$use" = true ]; then
    if [ -n "$pick" ]; then
      [ -n "$(__dehv.wall.get "$pick" url)" ] || { __dehv.fail "no repo ${c}${pick}${r}" "'dehv wall -n $pick' adds it"; return; }
    else
      __dehv.wall.choose || return
      pick=$REPLY
    fi
    local from=$(git config -f "$file" --get wall.use)
    if [ "$from" = "$pick" ]; then
      __dehv.noop "already using $pick, nothing to do"
      return
    fi
    git config -f "$file" wall.use "$pick" || return
    if [ -n "$from" ]; then
      __dehv.ok "${d}${from} →${r} ${b}${pick}${r} ${d}· $(__dehv.wall.short "$(__dehv.wall.get "$pick" url)")${r}"
    else
      __dehv.ok "${b}${pick}${r} ${d}· $(__dehv.wall.short "$(__dehv.wall.get "$pick" url)")${r}"
    fi
    return
  fi

  local repo

  # -s -a: every repo, on to the next when one fails.
  if [ "$all" = true ]; then
    __dehv.wall.names
    local -a names=($reply)
    [ ${#names} -gt 0 ] || { __dehv.fail "no repos yet" "'dehv wall -n <repo>' adds one"; return; }
    local failed=0
    for repo in $names; do
      __dehv.wall.sync "$repo" || { (( $? == 130 )) && return 130; (( failed++ )); }
    done
    (( failed == 0 ))
    return
  fi

  # <url>, -s and -o: on the repo named with -t, or the one in use.
  if [ -n "$to" ]; then
    [ -n "$(__dehv.wall.get "$to" url)" ] || { __dehv.fail "no repo ${c}${to}${r}" "'dehv wall -n $to' adds it"; return; }
    repo=$to
  else
    __dehv.wall.current ask || return
    repo=$REPLY
  fi

  if [ "$open" = true ]; then
    local dir=$(__dehv.wall.dir "$repo")
    if [ ! -d "$dir" ]; then
      if [ "$(__dehv.wall.get "$repo" cloned)" = true ]; then
        __dehv.fail "${dir/#$HOME/~} is gone" "'dehv wall -s' clones it again, after asking"
      else
        __dehv.fail "${repo} is not cloned yet" "'dehv wall -s' clones it"
      fi
      return
    fi
    open "$dir"
    return
  fi

  if [ "$sync" = true ]; then
    __dehv.wall.sync "$repo"
    return
  fi

  __dehv.wall.add "$repo" "$src" "$name"
}

# ─── completion ───────────────────────────────────────────────────────────────
_dehv_walls() {
  local -a reply
  __dehv.wall.names
  compadd -a reply
}

_dehv.wall() {
  _arguments '(-a --all -u --use -n --new -r --remove -l --list -i --init -x --clear)'{-t,--to}'[use that repo just this once]:repo:_dehv_walls' \
             '(-o --open -u --use -n --new -r --remove -l --list -i --init -x --clear)'{-s,--sync}'[commit what changed in its folder, pull and push]' \
             '(-t --to -u --use -n --new -r --remove -l --list -i --init -x --clear -o --open)'{-a,--all}'[with -s, every repo]' \
             '(-s --sync -a --all -u --use -n --new -r --remove -l --list -i --init -x --clear)'{-o,--open}'[open the folder of the repo in use]' \
             '(- *)'{-u,--use}'[choose the repo the rest use]::repo:_dehv_walls' \
             '(- *)'{-n,--new}'[add or edit a repo]:repo:_dehv_walls' \
             '(- *)'{-r,--remove}'[forget a repo, keeping its folder]:repo:_dehv_walls' \
             '(- *)'{-l,--list}'[list my repos]' \
             '(- *)'{-i,--init}'[create the repos file, or keep or blank it]' \
             '(- *)'{-x,--clear}'[forget every repo, after asking]' \
             '1:url:_urls' \
             '2:name:'
}
