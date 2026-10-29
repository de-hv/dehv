# dehv hi: a small card about this Mac, like fastfetch's, but in a few lines.
# Loaded by ../bin/dehv, which says how a command registers itself.

__dehv_commands+=(hi)
__dehv_about[hi]='a small card about this Mac'

# ─── hi ────────────────────────────────────────────────────────────────────────
# Everything it asks is local and quick (scutil, sw_vers, sysctl, route,
# ipconfig, pmset), so an rc file can run it in every new terminal.
__dehv.hi.help() {
  __dehv.usage hi "dehv hi" \
    "A small card about this Mac: the date, its name and macOS, uptime, IP and battery." \
    "|show it"
}

__dehv.hi() {
  case "$1" in
    "") ;;
    -*) __dehv.misuse hi "unknown option '$1'"; return ;;
    *)  __dehv.misuse hi "unexpected argument '$1'"; return ;;
  esac

  local b c d g y x r; __dehv.style

  local host os up iface ip batt
  host=$(scutil --get ComputerName 2>/dev/null || hostname -s)
  os="macOS $(sw_vers -productVersion 2>/dev/null)"

  # kern.boottime is "{ sec = 1729…, usec = … } …".
  zmodload -F zsh/datetime p:EPOCHSECONDS
  local boot=$(sysctl -n kern.boottime 2>/dev/null | awk -F'[ ,]+' '{ print $4 }')
  if [[ "$boot" == <-> ]]; then
    local mins=$(( (EPOCHSECONDS - boot) / 60 ))
    if (( mins >= 1440 )); then
      up="$(( mins / 1440 )) day$( (( mins / 1440 == 1 )) || print s)"
    elif (( mins >= 60 )); then
      up="$(( mins / 60 ))h $(( mins % 60 ))m"
    else
      up="${mins}m"
    fi
  fi

  iface=$(route -n get default 2>/dev/null | awk '/interface:/ { print $2 }')
  [ -n "$iface" ] && ip=$(ipconfig getifaddr "$iface" 2>/dev/null)

  # "-InternalBattery-0 (id=…)	87%; discharging; 5:12 remaining …"; nothing on
  # a Mac without one.
  local pm=$(pmset -g batt 2>/dev/null | sed -n 2p)
  if [[ "$pm" == *InternalBattery* ]]; then
    local pct=${${pm##*$'\t'}%%\%*} state=${${pm#*; }%%;*} hue=$g
    (( pct <= 20 )) && hue=$y
    (( pct <= 10 )) && hue=$x
    [ "$state" = discharging ] && state= || state=" ${d}· $state${r}"
    batt="${hue}${pct}%${r}${state}"
  fi

  local -a lines
  lines+=("${x}>${r}${b}>${r}${g}>${r} ${b}dehv${r}  ${d}$(date '+%A %-d %b · %H:%M')${r}")
  lines+=("")
  lines+=("${d}host${r}  $host ${d}· $os${r}")
  [ -n "$up" ]   && lines+=("${d}up${r}    $up")
  if [ -n "$ip" ]; then
    lines+=("${d}net${r}   $ip ${d}· $iface${r}")
  else
    lines+=("${d}net${r}   ${y}offline${r}")
  fi
  [ -n "$batt" ] && lines+=("${d}batt${r}  $batt")
  [ -t 1 ] && [ -z "$NO_COLOR" ] && \
    lines+=("" $'\e[31m●\e[33m●\e[32m●\e[36m●\e[34m●\e[35m●\e[37m●\e[0m')

  local line
  for line in "${lines[@]}"; do print -r -- "${line:+  $line}"; done
  echo
}

# ─── completion ────────────────────────────────────────────────────────────────
_dehv.hi() {
  _message 'no arguments'
}
