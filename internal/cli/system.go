package cli

import (
	"image/color"
	"strconv"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

func ipCmd() *cobra.Command {
	var public, copyIt bool
	c := &cobra.Command{
		Use:     "ip [-p] [-c]",
		Short:   "print my local or public IP address",
		Long:    "Print my local IP address: the one of the interface the default route uses (Wi-Fi, cable, dock).",
		GroupID: "system",
		Example: "dehv ip -c               # copy it too\n" +
			"ssh me@$(dehv ip)        # piped, only the address",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var ip, label string
			if public {
				spin := ui.Spin("asking api.ipify.org")
				addr, err := core.PublicIP(cmd.Context())
				spin.Stop()
				if err != nil {
					return err
				}
				ip, label = addr, "public"
			} else {
				a, err := core.LocalIP()
				if err != nil {
					return err
				}
				ip, label = a.IP, a.Iface
				if a.Port != "" {
					label += " · " + a.Port
				}
			}
			if copyIt {
				if err := core.Copy(ip); err != nil {
					return core.Fail("could not copy it", err.Error())
				}
			}
			// Only the address when piped, so $(dehv ip) stays usable.
			if !ui.IsTTY(ui.Stdout) {
				ui.Println(ip)
				return nil
			}
			line := ui.Bold.Render(ip) + "   " + ui.Dim.Render(label)
			if copyIt {
				line += "   " + ui.OKMark.Render("✓") + " " + ui.Dim.Render("copied")
			}
			ui.Println(line)
			return nil
		},
	}
	c.Flags().BoolVarP(&public, "public", "p", false, "print my public address instead (asks api.ipify.org)")
	c.Flags().BoolVarP(&copyIt, "copy", "c", false, "also copy it to the clipboard")
	return c
}

func hiCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "hi",
		Short:   "a small card about this Mac",
		Long:    "A small card about this Mac: the date, its name and macOS, uptime, IP and battery. Quick enough for every new terminal: add `dehv hi` to ~/.zshrc.",
		GroupID: "system",
		Args:    cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			for _, l := range HiLines(core.ThisMac(), time.Now(), ui.IsTTY(ui.Stdout)) {
				if l == "" {
					ui.Println()
				} else {
					ui.Println("  " + l)
				}
			}
			ui.Println()
			return nil
		},
	}
}

// HiLines is the card of `dehv hi`, shared with the hub's Machine panel.
func HiLines(m core.Machine, now time.Time, swatch bool) []string {
	lines := []string{
		ui.Err.Render(">") + ui.Bold.Render(">") + ui.OKMark.Render(">") + " " + ui.Bold.Render("dehv") + "  " +
			ui.Dim.Render(now.Format("Monday 2 Jan · 15:04")),
		"",
		ui.Dim.Render("host") + "  " + m.Host + " " + ui.Dim.Render("· "+m.OS),
	}
	if m.Uptime != "" {
		lines = append(lines, ui.Dim.Render("up")+"    "+m.Uptime)
	}
	if m.IP != "" {
		lines = append(lines, ui.Dim.Render("net")+"   "+m.IP+" "+ui.Dim.Render("· "+m.Iface))
	} else {
		lines = append(lines, ui.Dim.Render("net")+"   "+ui.Warn.Render("offline"))
	}
	if m.Battery >= 0 {
		hue := ui.OKMark
		if m.Battery <= 20 {
			hue = ui.Warn
		}
		if m.Battery <= 10 {
			hue = ui.Err
		}
		b := hue.Render(strconv.Itoa(m.Battery) + "%")
		if m.Power != "" {
			b += " " + ui.Dim.Render("· "+m.Power)
		}
		lines = append(lines, ui.Dim.Render("batt")+"  "+b)
	}
	if swatch {
		dots := ""
		for _, c := range []color.Color{ui.Red, ui.Yellow, ui.Green, ui.Cyan, ui.Blue, ui.Purple, ui.Fg} {
			dots += lipgloss.NewStyle().Foreground(c).Render("●")
		}
		lines = append(lines, "", dots)
	}
	return lines
}
