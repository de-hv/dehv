package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/de-hv/dehv/internal/sh"
)

// Address is a local IP and the interface it is on.
type Address struct {
	IP    string
	Iface string // en0
	Port  string // Wi-Fi, the hardware port's name
}

// DefaultIface is the interface the default route goes through, so it follows
// Wi-Fi, cable or a dock; "" offline.
func DefaultIface() string {
	out, _ := sh.Out("route", "-n", "get", "default")
	for _, line := range strings.Split(out, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), ":"); ok && k == "interface" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// LocalIP is the address of the interface in use.
func LocalIP() (Address, error) {
	iface := DefaultIface()
	try := iface
	if try == "" {
		try = "en0"
	}
	ip, _ := sh.Out("ipconfig", "getifaddr", try)
	if ip == "" {
		ports, _ := sh.Out("networksetup", "-listallhardwareports")
		if wifi := deviceOf(ports, "Wi-Fi"); wifi != "" {
			if power, _ := sh.Out("networksetup", "-getairportpower", wifi); strings.HasSuffix(power, "Off") {
				return Address{}, Fail("no local address", "Wi-Fi is off")
			}
		}
		return Address{}, Fail("no local address", "not connected to any network")
	}
	ports, _ := sh.Out("networksetup", "-listallhardwareports")
	return Address{IP: ip, Iface: try, Port: portOf(ports, try)}, nil
}

// "Hardware Port: Wi-Fi\nDevice: en0\n…" pairs, read both ways.
func deviceOf(list, port string) string {
	cur := ""
	for _, line := range strings.Split(list, "\n") {
		if p, ok := strings.CutPrefix(line, "Hardware Port: "); ok {
			cur = p
		} else if d, ok := strings.CutPrefix(line, "Device: "); ok && cur == port {
			return d
		}
	}
	return ""
}

func portOf(list, dev string) string {
	cur := ""
	for _, line := range strings.Split(list, "\n") {
		if p, ok := strings.CutPrefix(line, "Hardware Port: "); ok {
			cur = p
		} else if d, ok := strings.CutPrefix(line, "Device: "); ok && d == dev {
			return cur
		}
	}
	return ""
}

// PublicIP asks api.ipify.org, for 5 seconds at most.
func PublicIP(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if DefaultIface() == "" {
			return "", Fail("no public address", "offline")
		}
		return "", Fail("no public address", "api.ipify.org did not answer")
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	ip := strings.TrimSpace(string(b))
	if err != nil || resp.StatusCode != http.StatusOK || ip == "" {
		return "", Fail("no public address", "api.ipify.org did not answer")
	}
	return ip, nil
}

// Copy puts s in the clipboard.
func Copy(s string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run()
}

// Machine is what `dehv hi` shows about this Mac. Every value is local and
// quick to ask, so an rc file can show it in every new terminal.
type Machine struct {
	Host    string
	OS      string
	Uptime  string // "" when unknown
	IP      string // "" offline
	Iface   string
	Battery int    // -1 without one
	Power   string // "charging", "charged", "AC attached"…; "" discharging
}

// ThisMac asks it.
func ThisMac() Machine {
	m := Machine{Battery: -1}
	m.Host, _ = sh.Out("scutil", "--get", "ComputerName")
	if m.Host == "" {
		m.Host, _ = os.Hostname()
	}
	v, _ := sh.Out("sw_vers", "-productVersion")
	m.OS = "macOS " + v
	// kern.boottime is "{ sec = 1729…, usec = … } …".
	if b, _ := sh.Out("sysctl", "-n", "kern.boottime"); b != "" {
		f := strings.FieldsFunc(b, func(r rune) bool { return r == ' ' || r == ',' })
		if len(f) > 3 {
			if sec, err := strconv.ParseInt(f[3], 10, 64); err == nil {
				m.Uptime = Uptime(time.Since(time.Unix(sec, 0)))
			}
		}
	}
	if m.Iface = DefaultIface(); m.Iface != "" {
		m.IP, _ = sh.Out("ipconfig", "getifaddr", m.Iface)
	}
	pm, _ := sh.Out("pmset", "-g", "batt")
	m.Battery, m.Power = ParseBattery(pm)
	return m
}

// Uptime is "3 days", "2h 5m" or "12m".
func Uptime(d time.Duration) string {
	mins := int(d.Minutes())
	switch {
	case mins >= 1440:
		days := mins / 1440
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	case mins >= 60:
		return fmt.Sprintf("%dh %dm", mins/60, mins%60)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}

// ParseBattery reads `pmset -g batt`'s second line, "-InternalBattery-0
// (id=…)	87%; discharging; 5:12 remaining …"; -1 on a Mac without one.
func ParseBattery(pm string) (int, string) {
	lines := strings.Split(pm, "\n")
	if len(lines) < 2 || !strings.Contains(lines[1], "InternalBattery") {
		return -1, ""
	}
	l := lines[1]
	if i := strings.LastIndex(l, "\t"); i >= 0 {
		l = l[i+1:]
	}
	parts := strings.Split(l, ";")
	pct, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(parts[0]), "%"))
	if err != nil {
		return -1, ""
	}
	state := ""
	if len(parts) > 1 {
		state = strings.TrimSpace(parts[1])
	}
	if state == "discharging" {
		state = ""
	}
	return pct, state
}
