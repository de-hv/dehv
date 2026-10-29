# dehv port: what listens on a port, and freeing it.
# Loaded by ../bin/dehv, which says how a command registers itself.

__dehv_commands+=(port)
__dehv_about[port]='inspect or free ports'

# ─── port ──────────────────────────────────────────────────────────────────────
__dehv.port.help() {
  __dehv.usage port "dehv port <port>... | -a | -k [<port>...]" \
    "Inspect or free ports: what listens (TCP) on them, as PORT, PROGRAM and PID." \
    "<port>...|show what listens on those ports" \
    "-a, --all|show everything that listens, on any port" \
    "-k, --kill <port>...|kill what listens on those ports" \
    "-k, --kill|pick what to kill from everything that listens"
}

# "port program pid" of what listens on TCP, one per line: on the ports given,
# or on any without them.
__dehv.port.rows() {
  local port
  {
    if [ $# -eq 0 ]; then
      lsof +c 0 -nP -iTCP -sTCP:LISTEN
    else
      for port in "$@"; do lsof +c 0 -nP -iTCP:"$port" -sTCP:LISTEN; done
    fi
  } 2>/dev/null | awk '$1 != "COMMAND" { n = split($9, a, ":"); print a[n], $1, $2 }' | sort -u | sort -n
}

__dehv.port() {
  [ $# -gt 0 ] || { __dehv.port.help; return; }

  local kill=false all=false arg
  local -a ports
  local b c d g y x r; __dehv.style

  for arg in "$@"; do
    case "$arg" in
      -k|--kill) kill=true ;;
      -a|--all)  all=true ;;
      -*)        __dehv.misuse port "unknown option '$arg'"; return ;;
      *)         [[ "$arg" == <-> ]] || { __dehv.misuse port "invalid port '$arg'"; return; }
                 ports+=("$arg") ;;
    esac
  done

  if [ "$all" = true ] && { [ "$kill" = true ] || [ ${#ports} -gt 0 ]; }; then
    __dehv.misuse port "-a goes alone"; return
  fi

  local -a rows
  rows=("${(@f)$(__dehv.port.rows "${ports[@]}")}")
  rows=(${rows:#})

  # No ports: -a shows them all, -k offers them all.
  if [ ${#ports} -eq 0 ]; then
    if [ ${#rows} -eq 0 ]; then
      __dehv.fail "nothing is listening"
      return
    fi
    if [ "$kill" = true ]; then
      if [ ! -t 0 ] || [ ! -t 1 ]; then
        __dehv.misuse port "-k without a port needs a terminal to pick from"; return
      fi
      __dehv.pick "${(@f)$(print -l -- $rows | column -t)}" || return 1
      rows=("${rows[REPLY]}")
    fi
  fi

  local port rc=0
  local -a found
  for port in $ports; do
    found=(${(M)rows:#$port *})
    if [ ${#found} -eq 0 ]; then
      __dehv.fail "nothing on port ${c}${port}${r}"
      rc=1
    fi
  done
  [ ${#rows} -gt 0 ] || return 1

  local row
  local -a f
  if [ "$kill" = true ]; then
    for row in $rows; do
      f=(${=row})
      kill -9 "${f[3]}" && __dehv.ok "killed ${b}${f[2]}${r} ${d}(${f[3]})${r} on port ${c}${f[1]}${r}" || rc=1
    done
  elif [ -t 1 ]; then
    # In a frame, each column as wide as its longest value.
    local w1=4 w2=7
    local -a lines
    for row in $rows; do
      f=(${=row})
      (( ${#f[1]} > w1 )) && w1=${#f[1]}
      (( ${#f[2]} > w2 )) && w2=${#f[2]}
    done
    lines=("${d}${(r:w1:):-PORT}  ${(r:w2:):-PROGRAM}  PID${r}")
    for row in $rows; do
      f=(${=row})
      lines+=("${c}${(r:w1:)f[1]}${r}  ${b}${(r:w2:)f[2]}${r}  ${d}${f[3]}${r}")
    done
    __dehv.box ports "${#rows} listening" "${lines[@]}"
  else
    print -l -- "PORT PROGRAM PID" $rows | column -t
  fi
  return $rc
}

# ─── completion ───────────────────────────────────────────────────────────────
_dehv.port() {
  _arguments '(-k --kill -a --all)'{-k,--kill}'[kill what listens there, or pick from a list]' \
             '(-a --all -k --kill)'{-a,--all}'[show everything that listens]' \
             '*:port:'
}
