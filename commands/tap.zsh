# dehv tap: the steps to release an app or a command line tool to my
# homebrew-tap.
# Loaded by ../dehv.plugin.zsh, which says how a command registers itself.

__dehv_commands+=(tap)
__dehv_about[tap]='the steps to release to my homebrew-tap'

# ─── tap ───────────────────────────────────────────────────────────────────────
# Only prints the commands, never runs them: an app (.app) goes in Casks/ as a
# zip, a command line tool in Formula/ as a tarball of its binary or, with -s,
# as the tag's source that brew builds. The GitHub owner of the tap and of the
# apps is tap.owner in the config, de-hv without it.
__dehv.tap.help() {
  __dehv.usage tap "dehv tap [-c [<App> <version>] | -f [<name> <version>] [-s]]" \
    "The steps to release an app (a cask) or a command line tool (a formula) to my homebrew-tap: zip or pack it, its sha256, upload it and what to change in its .rb. It only prints them; piped, as a script whose notes are comments." \
    "|the steps for both, with <placeholders>" \
    "-c, --cask [<App> <version>]|only the cask's, filled in; without them, from build/*.app and ./VERSION" \
    "-f, --formula [<name> <version>]|only the formula's, filled in; without them, from the repo's folder and ./VERSION" \
    "-s, --source|the formula's, built by brew from the tag's source instead of a binary"
}

__dehv.tap() {
  local kind= source=false arg
  local -a args

  for arg in "$@"; do
    case "$arg" in
      -c|--cask)    [ "$kind" = formula ] && { __dehv.misuse tap "-c and -f go apart"; return; }; kind=cask ;;
      -f|--formula) [ "$kind" = cask ] && { __dehv.misuse tap "-c and -f go apart"; return; }; kind=formula ;;
      -s|--source)  source=true ;;
      -*)           __dehv.misuse tap "unknown option '$arg'"; return ;;
      *)            args+=("$arg") ;;
    esac
  done
  if [ "$source" = true ]; then
    [ "$kind" = cask ] && { __dehv.misuse tap "-s is for a formula"; return; }
    kind=formula
  fi
  (( ${#args} > 0 )) && [ -z "$kind" ] && { __dehv.misuse tap "'${args[1]}' goes with -c or -f"; return; }
  (( ${#args} > 2 )) && { __dehv.misuse tap "unexpected argument '${args[3]}'"; return; }

  # What is not given comes from the repo in this folder, when it has it.
  local name=${args[1]} version=${args[2]#v} top
  local -a apps
  if [ -n "$kind" ]; then
    [ -z "$version" ] && [ -f VERSION ] && version=${$(<VERSION)//[[:space:]]/}
    if [ -z "$name" ] && [ "$kind" = cask ]; then
      apps=(build/*.app(N/))
      (( ${#apps} == 1 )) && name=${apps[1]:t:r}
    elif [ -z "$name" ]; then
      top=$(git rev-parse --show-toplevel 2>/dev/null) && name=${top:t}
    fi
  fi

  local b c d g y x r; __dehv.style
  local owner=$(__dehv.config tap.owner de-hv) missing=false

  # A value filled in, in green, or its <placeholder>, in yellow.
  local App app N V S=${y}'<the sha256 of 2>'${r}
  if [ -n "$name" ]; then
    App=$g$name$r N=$g$name$r app=$g${(L)name// /-}$r
  else
    App=$y'<App>'$r N=$y'<name>'$r app=$y'<app>'$r missing=true
  fi
  if [ -n "$version" ]; then
    V=$g$version$r
  else
    V=$y'<version>'$r missing=true
  fi

  # Each row as kind|text: h a heading (its first word, then what it is), s a
  # step, c a command, e a line of the .rb, n a note, and "" a blank line.
  local -a rows
  local k

  if [ "$kind" != formula ]; then
    k=0
    rows+=(
      "h|Cask an app (.app), in Casks/$app.rb"
      "s|$((++k)) zip the app"
      "c|ditto -c -k --keepParent build/$App.app build/$App-$V.zip"
      "s|$((++k)) its sha256"
      "c|shasum -a 256 build/$App-$V.zip"
      "s|$((++k)) upload it, where the cask's url points"
      "c|gh release create v$V build/$App-$V.zip"
      "s|$((++k)) in homebrew-tap, Casks/$app.rb"
      "e|version \"$V\""
      "e|sha256 \"$S\""
      "s|$((++k)) push it"
      "c|git commit -am \"$app $V\" && git push"
      "s|$((++k)) check it"
      "c|brew update && brew audit --cask $owner/tap/$app"
      "c|brew upgrade --cask $owner/tap/$app"
      ""
      "n|a new one, from its url:"
      "c|brew create --cask --tap $owner/tap $y<url>$r"
    )
  fi

  if [ "$kind" != cask ]; then
    [ ${#rows} -gt 0 ] && rows+=("")
    k=0
    rows+=("h|Formula a command line tool, in Formula/$N.rb")
    if [ "$source" = true ]; then
      rows+=(
        "s|$((++k)) tag it"
        "c|git tag v$V && git push origin v$V"
        "s|$((++k)) the sha256 of the tag's source"
        "c|curl -sL https://github.com/$owner/$N/archive/refs/tags/v$V.tar.gz | shasum -a 256"
        "s|$((++k)) in homebrew-tap, Formula/$N.rb"
        "e|url \"https://github.com/$owner/$N/archive/refs/tags/v$V.tar.gz\""
        "e|sha256 \"$S\""
      )
    else
      rows+=(
        "s|$((++k)) pack the binary"
        "c|tar -czf build/$N-$V.tar.gz -C build $N"
        "s|$((++k)) its sha256"
        "c|shasum -a 256 build/$N-$V.tar.gz"
        "s|$((++k)) upload it, where the formula's url points"
        "c|gh release create v$V build/$N-$V.tar.gz"
        "s|$((++k)) in homebrew-tap, Formula/$N.rb"
        "e|url \"https://github.com/$owner/$N/releases/download/v$V/$N-$V.tar.gz\""
        "e|sha256 \"$S\""
      )
    fi
    rows+=(
      "s|$((++k)) push it"
      "c|git commit -am \"$N $V\" && git push"
      "s|$((++k)) check it"
      "c|brew update && brew audit --strict --formula $owner/tap/$N"
      "c|brew install $owner/tap/$N && brew test $owner/tap/$N"
    )
    [ "$source" = false ] && rows+=(
      ""
      "n|built by brew from the tag's source instead: dehv tap -s"
    )
    rows+=(
      ""
      "n|a new one, from its url:"
      "c|brew create --tap $owner/tap $y<url>$r"
    )
  fi

  # In a terminal, notes dimmed and the commands as they are, never cut so
  # they copy whole; piped, a script: the notes become comments.
  local row text head tty=false
  [ -t 1 ] && tty=true
  if [ "$tty" = true ]; then
    print -r -- "${c}dehv tap${r} ${d}· the steps to release to $owner/homebrew-tap${r}"
    [ "$missing" = true ] && print -r -- "${d}dehv tap -c|-f <name> <version> fills in the${r} ${y}<placeholders>${r}"
    print
  else
    print -r -- "# dehv tap: the steps to release to $owner/homebrew-tap"
    print
  fi

  for row in "${rows[@]}"; do
    text=${row#*|}
    # Dimmed text stays dimmed past a value's colour.
    [ -n "$r" ] && text=${text//$r/$r$d}
    case "${row%%|*}" in
      h) head=${text%% *} text=${text#* }
         if [ "$tty" = true ]; then
           print -r -- "${b}${head}${r} ${d}· ${text}${r}"
         else
           print -r -- "# $head: $text"
         fi ;;
      s) if [ "$tty" = true ]; then
           print -r -- "  ${c}${text%% *}${r} ${d}${text#* }${r}"
         else
           print -r -- "# $text"
         fi ;;
      c) text=${row#*|}
         if [ "$tty" = true ]; then print -r -- "    $text"; else print -r -- "$text"; fi ;;
      e) text=${row#*|}
         if [ "$tty" = true ]; then print -r -- "    $text"; else print -r -- "#   $text"; fi ;;
      n) if [ "$tty" = true ]; then print -r -- "  ${d}${text}${r}"; else print -r -- "# $text"; fi ;;
      *) print ;;
    esac
  done
}

# ─── completion ────────────────────────────────────────────────────────────────
_dehv.tap() {
  _arguments '(-f --formula)'{-c,--cask}'[only the cask'\''s steps]' \
             '(-c --cask)'{-f,--formula}'[only the formula'\''s steps]' \
             '(-c --cask)'{-s,--source}'[the formula built from the tag'\''s source]' \
             '1:name: ' '2:version: '
}
