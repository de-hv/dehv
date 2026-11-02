package core

import (
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/de-hv/dehv/internal/sh"
)

// Listener is a program listening on a TCP port.
type Listener struct {
	Port    int
	Program string
	PID     int
	User    string
}

// Listeners is what listens on TCP: on the ports given, or on any without
// them, sorted by port. One per port and program, though it listens on both
// IPv4 and IPv6.
func Listeners(ports ...int) ([]Listener, error) {
	var outs []string
	if len(ports) == 0 {
		out, err := sh.Out("lsof", "+c", "0", "-nP", "-iTCP", "-sTCP:LISTEN")
		if err != nil && out == "" && sh.Output(err) != "" {
			return nil, err
		}
		outs = append(outs, out)
	} else {
		for _, p := range ports {
			// lsof answers 1 when nothing is there: not an error here.
			out, _ := sh.Out("lsof", "+c", "0", "-nP", "-iTCP:"+strconv.Itoa(p), "-sTCP:LISTEN")
			outs = append(outs, out)
		}
	}
	return ParseLsof(strings.Join(outs, "\n")), nil
}

// ParseLsof reads `lsof -nP -iTCP -sTCP:LISTEN`: COMMAND PID USER FD TYPE
// DEVICE SIZE/OFF NODE NAME, where NAME is host:port.
func ParseLsof(out string) []Listener {
	seen := map[string]bool{}
	var ls []Listener
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 9 || f[0] == "COMMAND" {
			continue
		}
		name := f[8]
		port, err := strconv.Atoi(name[strings.LastIndex(name, ":")+1:])
		if err != nil {
			continue
		}
		pid, _ := strconv.Atoi(f[1])
		prog := strings.ReplaceAll(f[0], `\x20`, " ")
		key := strconv.Itoa(port) + " " + prog + " " + f[1]
		if seen[key] {
			continue
		}
		seen[key] = true
		ls = append(ls, Listener{Port: port, Program: prog, PID: pid, User: f[2]})
	}
	sort.SliceStable(ls, func(i, j int) bool {
		if ls[i].Port != ls[j].Port {
			return ls[i].Port < ls[j].Port
		}
		return ls[i].PID < ls[j].PID
	})
	return ls
}

// Kill kills pid, at once.
func Kill(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}
