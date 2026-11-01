# dehv tap: the steps to release a new version of an app or a command line tool
# to my homebrew-tap.
# Loaded by ../bin/dehv, which says how a command registers itself.

__dehv_commands+=(tap)
__dehv_about[tap]='the steps to release to my homebrew-tap'

# ─── tap ───────────────────────────────────────────────────────────────────────
# Only prints the commands, never runs them, with generic names: an app (.app)
# is a cask, released as a zip like inlay; a command line tool is a formula,
# released as the tag's source like dehv. The GitHub owner of the tap and of the
# apps is tap.owner in the config, de-hv without it.
__dehv.tap.help() {
  __dehv.usage tap "dehv tap [cask|formula]" \
    "The steps to release a new version to my homebrew-tap: in the repo, in the tap and on this Mac. It only prints them; piped, as a script whose notes are comments." \
    "|asks which of the two, from a list" \
    "cask|an app (.app), like inlay: build it, zip it, create the release" \
    "formula|a command line tool, like dehv: tag it, and brew installs the tag's source"
}

# Sets $REPLY to cask or formula, picked from a list wiped after, heading too.
__dehv.tap.ask() {
  local b c d g y x r; __dehv.style
  print -r -- "${d}What are you releasing?${r}" > /dev/tty
  __dehv.pick "cask     an app (.app), like inlay" "formula  a command line tool, like dehv"
  local picked=$? choice=$REPLY
  print -n -- $'\e[1A\e[J' > /dev/tty
  (( picked == 0 )) || return 1
  (( choice == 1 )) && REPLY=cask || REPLY=formula
}

__dehv.tap() {
  local kind
  (( $# > 1 )) && { __dehv.misuse tap "unexpected argument '$2'"; return; }
  case "$1" in
    cask|formula) kind=$1 ;;
    -*) __dehv.misuse tap "unknown option '$1'"; return ;;
    ?*) __dehv.misuse tap "'$1' is not cask or formula"; return ;;
    *)
      if [ ! -t 0 ] || [ ! -t 1 ]; then
        __dehv.misuse tap "no terminal to ask on: say cask or formula"
        return
      fi
      __dehv.tap.ask || return 1
      kind=$REPLY
      ;;
  esac

  local b c d g y x r; __dehv.style
  local owner=$(__dehv.config tap.owner de-hv)

  # The generic names, in yellow.
  local N=$y'<name>'$r A=$y'<App>'$r V=$y'<version>'$r S=$y'<sha256>'$r
  local B=$y'<build command>'$r

  # Each row as kind|text: h a heading (optionally |what it is), s a step, c a
  # command, e a line of the .rb, and "" a blank line.
  local -a rows
  local about src="https://github.com/$owner/$N/archive/refs/tags/v$V.tar.gz"

  if [ "$kind" = cask ]; then
    about='a new version of an app, like inlay'
    rows=(
      "h|In the repo|$N"
      "s|1 raise the version: edit VERSION, and CHANGELOG if it has one"
      "c|git commit -am \"$N $V\" && git push"
      "s|2 build the app: the version goes inside it, so after step 1"
      "c|$B"
      "s|3 zip it"
      "c|ditto -c -k --keepParent build/$A.app build/$A-$V.zip"
      "s|4 create the release: it also creates the tag"
      "c|gh release create v$V build/$A-$V.zip --generate-notes"
      "s|5 its sha256: of this zip, the one uploaded"
      "c|shasum -a 256 build/$A-$V.zip"
      ""
      "h|In homebrew-tap"
      "s|6 in Casks/$N.rb, these two lines: the url follows the version"
      "e|version \"$V\""
      "e|sha256 \"$S\""
      "s|7 push it"
      "c|git commit -am \"$N $V\" && git push"
      ""
      "h|On this Mac"
      "s|8 check it"
      "c|brew update && brew upgrade --cask $owner/tap/$N"
    )
  else
    about='a new version of a command line tool, like dehv'
    rows=(
      "h|In the repo|$N"
      "s|1 raise the version: edit VERSION, and CHANGELOG if it has one"
      "c|git commit -am \"$N $V\" && git push"
      "s|2 tag it: this is what brew downloads"
      "c|git tag v$V && git push origin v$V"
      "s|3 its sha256: of the tarball GitHub makes of the tag"
      "c|curl -sL $src | shasum -a 256"
      ""
      "h|In homebrew-tap"
      "s|4 in Formula/$N.rb, these two lines"
      "e|url \"$src\""
      "e|sha256 \"$S\""
      "s|5 push it"
      "c|git commit -am \"$N $V\" && git push"
      ""
      "h|On this Mac"
      "s|6 check it"
      "c|brew update && brew upgrade $owner/tap/$N"
      "c|brew test $owner/tap/$N"
    )
  fi

  # In a terminal, notes dimmed and the commands as they are, never cut so
  # they copy whole; piped, a script: the notes become comments.
  local row text head note tty=false
  [ -t 1 ] && tty=true
  if [ "$tty" = true ]; then
    print -r -- "${c}dehv tap $kind${r} ${d}· $about${r}"
    print
  else
    print -r -- "# dehv tap $kind: $about"
    print
  fi

  for row in "${rows[@]}"; do
    text=${row#*|}
    case "${row%%|*}" in
      h) head=${text%%|*} note=
         [[ "$text" == *\|* ]] && note=${text#*|}
         if [ "$tty" = true ]; then
           print -r -- "${b}${head}${r}${note:+ ${d}· ${r}${note}}"
         else
           print -r -- "# $head${note:+: $note}"
         fi ;;
      s) # Dimmed text stays dimmed past a value's colour.
         [ -n "$r" ] && text=${text//$r/$r$d}
         if [ "$tty" = true ]; then
           print -r -- "  ${c}${text%% *}${r} ${d}${text#* }${r}"
         else
           print -r -- "# $text"
         fi ;;
      c) if [ "$tty" = true ]; then print -r -- "    $text"; else print -r -- "$text"; fi ;;
      e) if [ "$tty" = true ]; then print -r -- "    $text"; else print -r -- "#   $text"; fi ;;
      *) print ;;
    esac
  done
}

# ─── completion ────────────────────────────────────────────────────────────────
_dehv.tap() {
  _arguments '1:what:((cask\:"an app (.app), like inlay" formula\:"a command line tool, like dehv"))'
}
