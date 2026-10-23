# ┌──────────────────────────────────────────────────────────────────────────────┐
#                                    DEHV
# └──────────────────────────────────────────────────────────────────────────────┘
# My own commands, all behind one name so none can clash with the system's:
#
#   dehv <command> [options] [args]
#
# Shaped like a regular CLI: `dehv` alone prints a short usage, a command that
# needs an argument prints its own when run without any (`dehv help <command>`
# does for all of them; commands take no -h), options are flags (short and
# long) rather than action words, and a misuse prints the command's usage line
# to stderr.
#
# This file loads lib.zsh (the helpers every command shares) and then every
# commands/*.zsh, in alphabetical order, which is also their order in
# `dehv help`. Adding a command is adding one file there that:
#
#   1. registers it:  __dehv_commands+=(<command>)
#                     __dehv_about[<command>]='what it is, for dehv <tab>'
#                     __dehv_rows+=("<command>|<arguments>|what it does"
#                                   "|<arguments>|what it does" ...)
#   2. defines __dehv.<command> and its help, __dehv.<command>.help, a call to
#      __dehv.usage,
#   3. and its completion, _dehv.<command>.
#
# Sourced from the shell's rc file, through the link install.sh makes in
# ~/.local/share/dehv, so the clone can live anywhere.

# Where this file really is: the clone, past the link.
__dehv_dir=${${(%):-%x}:A:h}

# Emptied first, so sourcing again does not repeat any row.
typeset -ga __dehv_commands=() __dehv_rows=()
typeset -gA __dehv_about=()

() {
  local file
  source "$__dehv_dir/lib.zsh"
  for file in "$__dehv_dir"/commands/*.zsh(N); do
    source "$file"
  done
}

dehv() {
  # The frames, marks and spinner are UTF-8: without such a locale zsh would
  # measure them in bytes.
  [[ "${LC_ALL:-${LC_CTYPE:-$LANG}}" == *UTF-8* ]] || local LC_ALL=en_US.UTF-8

  local cmd="$1"
  [ $# -gt 0 ] && shift

  case "$cmd" in
    ""|-h|--help) __dehv.help ;;
    help)
      if [ -z "$1" ]; then
        __dehv.help
      elif (( ${__dehv_commands[(Ie)$1]} )); then
        "__dehv.$1.help"
      else
        __dehv.unknown "$1"
      fi
      ;;
    *)
      if (( ${__dehv_commands[(Ie)$cmd]} )); then
        "__dehv.$cmd" "$@"
      else
        __dehv.unknown "$cmd"
      fi
      ;;
  esac
}

# ─── help ──────────────────────────────────────────────────────────────────────
# Every command with each way to call it, the rows the commands registered as
# command|arguments|what it does: in a terminal, a card; piped, the same rows
# as plain text.
__dehv.help() {
  local b c d g y x r; __dehv.style
  local -a lines f

  # The command column as wide as the longest name, plus two.
  local row cmd w=0
  for cmd in $__dehv_commands; do (( ${#cmd} > w )) && w=${#cmd}; done
  (( w += 2 ))

  for row in $__dehv_rows; do
    f=("${(@s:|:)row}")
    [ -n "${f[1]}" ] && [ ${#lines} -gt 0 ] && lines+=("")
    lines+=("${c}${(r:w:)f[1]}${r}${y}${(r:24:)f[2]}${r}${f[3]}")
  done

  if [ -t 1 ]; then
    __dehv.box dehv "dehv help <command>" "${d}my own desktop toolkit${r}" "" "${lines[@]}"
  else
    print -r -- "dehv - my own desktop toolkit

Usage: dehv <command> [options] [args]
"
    print -rl -- "${lines[@]}"
    print -r -- "
Run 'dehv help <command>' for details."
  fi
}

# ─── completion ────────────────────────────────────────────────────────────────
# _dehv lives in completions/, which compinit finds through $fpath. Registered
# here too, now if compinit already ran, or else at the first prompt, after it:
# a compinit that trusts its cached dump never looks at a new folder in $fpath.
fpath=("$__dehv_dir/completions" ${fpath:#$__dehv_dir/completions})
autoload -Uz _dehv

if (( $+functions[compdef] )); then
  compdef _dehv dehv
else
  __dehv.compdef() {
    (( $+functions[compdef] )) && compdef _dehv dehv
    add-zsh-hook -d precmd __dehv.compdef
  }
  autoload -Uz add-zsh-hook
  add-zsh-hook precmd __dehv.compdef
fi

# ─── greeting ──────────────────────────────────────────────────────────────────
# The card of `dehv hi` in the first terminal of the day, when the config says
# so (install.sh asks).
if [[ -o interactive ]] && [ "$(__dehv.config hi.daily)" = true ]; then
  dehv hi --daily
fi
