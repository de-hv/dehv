package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

func portCmd() *cobra.Command {
	var all, kill bool
	c := &cobra.Command{
		Use:     "port <port>... | -a | -k [<port>...]",
		Short:   "inspect or free ports",
		Long:    "Inspect or free ports: what listens (TCP) on them, as PORT, PROGRAM and PID.",
		GroupID: "system",
		Example: "dehv port 3000 5173      # what listens on those\n" +
			"dehv port -a             # everything that listens\n" +
			"dehv port -k 3000        # free the port a dev server left behind\n" +
			"dehv port -k             # pick what to kill",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !all && !kill {
				return cmd.Help()
			}
			if all && (kill || len(args) > 0) {
				return misuse(cmd, "-a goes alone")
			}
			var ports []int
			for _, a := range args {
				p, err := strconv.Atoi(a)
				if err != nil || p <= 0 || p > 65535 {
					return misuse(cmd, fmt.Sprintf("invalid port '%s'", a))
				}
				ports = append(ports, p)
			}
			rows, err := core.Listeners(ports...)
			if err != nil {
				return err
			}

			// No ports: -a shows them all, -k offers them all.
			if len(ports) == 0 {
				if len(rows) == 0 {
					return core.Fail("nothing is listening", "")
				}
				if kill {
					if err := needsTTY(cmd, "-k without a port needs a terminal to pick from"); err != nil {
						return err
					}
					var opts []string
					for _, l := range rows {
						opts = append(opts, fmt.Sprintf("%-6d %-14s %d", l.Port, l.Program, l.PID))
					}
					i, err := ui.Pick("Kill what listens on", opts)
					if err != nil {
						return err
					}
					rows = rows[i : i+1]
				}
			}

			failed := false
			for _, p := range ports {
				found := false
				for _, l := range rows {
					found = found || l.Port == p
				}
				if !found {
					ui.Fail("nothing on port "+ui.Name.Render(strconv.Itoa(p)), "dehv port -a to see what listens")
					failed = true
				}
			}
			if len(rows) == 0 {
				return errSilent
			}

			if kill {
				for _, l := range rows {
					if err := core.Kill(l.PID); err != nil {
						ui.Fail(fmt.Sprintf("could not kill %s (%d)", l.Program, l.PID), err.Error())
						failed = true
						continue
					}
					ui.OK(fmt.Sprintf("killed %s %s on port %s", ui.Bold.Render(l.Program),
						ui.Dim.Render(fmt.Sprintf("(%d)", l.PID)), ui.Name.Render(strconv.Itoa(l.Port))), "")
				}
			} else {
				tty := ui.IsTTY(ui.Stdout)
				var cells [][]string
				for _, l := range rows {
					port, prog, pid := strconv.Itoa(l.Port), l.Program, strconv.Itoa(l.PID)
					if tty {
						port, prog, pid = ui.Name.Render(port), ui.Bold.Render(prog), ui.Dim.Render(pid)
					}
					cells = append(cells, []string{port, prog, pid})
				}
				ui.Table([]string{"PORT", "PROGRAM", "PID"}, cells)
				if tty {
					ui.Println(ui.Dim.Render(fmt.Sprintf("%d listening", len(rows))))
				}
			}
			if failed {
				return errSilent
			}
			return nil
		},
	}
	c.Flags().BoolVarP(&all, "all", "a", false, "show everything that listens, on any port")
	c.Flags().BoolVarP(&kill, "kill", "k", false, "kill what listens on those ports; without any, pick it from a list")
	return c
}
