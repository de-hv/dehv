# dehv ip: my local or public IP address.
# Loaded by ../bin/dehv, which says how a command registers itself.

__dehv_commands+=(ip)
__dehv_about[ip]='print my local IP address'

# ─── ip ──────────────────────────────────────────────────────────────────────
# The address of whichever interface the default route goes through, so it
# follows Wi-Fi, cable or a dock. The public one is asked to api.ipify.org.
__dehv.ip.help() {
  __dehv.usage ip "dehv ip [-p] [-c]" \
    "Print my local IP address, the one of the interface in use." \
    "-p, --public|print my public address instead (asks api.ipify.org)" \
    "-c, --copy|also copy it to the clipboard"
}

__dehv.ip() {
  local copy=false public=false arg

  for arg in "$@"; do
    case "$arg" in
      -c|--copy)   copy=true ;;
      -p|--public) public=true ;;
      -*)          __dehv.misuse ip "unknown option '$arg'"; return ;;
      *)           __dehv.misuse ip "unexpected argument '$arg'"; return ;;
    esac
  done

  local b c d g y x r; __dehv.style
  local ip label iface
  iface=$(route -n get default 2>/dev/null | awk '/interface:/ { print $2 }')

  if [ "$public" = true ]; then
    # curl in the background and a spinner until its line arrives; the echo
    # makes sure one does, empty when curl fails.
    local fd i=0 spin='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
    exec {fd}< <(curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null; echo)
    until read -t 0.08 -u $fd ip; do
      [ -t 2 ] && print -n -- "\r${c}${spin[i % 10 + 1]}${r} ${d}asking api.ipify.org…${r}" >&2
      (( i++ ))
    done
    [ -t 2 ] && print -n -- $'\r\e[K' >&2
    exec {fd}<&-

    if [ -z "$ip" ]; then
      if [ -z "$iface" ]; then
        __dehv.fail "no public address" "offline"
      else
        __dehv.fail "no public address" "api.ipify.org did not answer"
      fi
      return
    fi
    label="public"
  else
    ip=$(ipconfig getifaddr "${iface:-en0}" 2>/dev/null)
    if [ -z "$ip" ]; then
      local wifi
      wifi=$(networksetup -listallhardwareports 2>/dev/null | awk '/Hardware Port: Wi-Fi/ { getline; print $2 }')
      if [ -n "$wifi" ] && networksetup -getairportpower "$wifi" 2>/dev/null | grep -q 'Off$'; then
        __dehv.fail "no local address" "Wi-Fi is off"
      else
        __dehv.fail "no local address" "not connected to any network"
      fi
      return
    fi
    label=$(networksetup -listallhardwareports 2>/dev/null \
      | awk -v dev="$iface" '/Hardware Port:/ { p = substr($0, 16) } $1 == "Device:" && $2 == dev { print p }')
    label="${iface}${label:+ · $label}"
  fi

  [ "$copy" = true ] && printf '%s' "$ip" | pbcopy

  # Only the address when piped, so `$(dehv ip)` stays usable.
  if [ -t 1 ]; then
    local copied=
    [ "$copy" = true ] && copied="   ${g}✓${r} ${d}copied${r}"
    print -r -- "${b}${ip}${r}   ${d}${label}${r}${copied}"
  else
    echo "$ip"
  fi
}

# ─── completion ───────────────────────────────────────────────────────────────
_dehv.ip() {
  _arguments '(-p --public)'{-p,--public}'[print my public address instead]' \
             '(-c --copy)'{-c,--copy}'[also copy it to the clipboard]'
}
