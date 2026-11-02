package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/ui"
)

func branchesCmd() *cobra.Command {
	var list bool
	c := &cobra.Command{
		Use:   "branches [-l]",
		Short: "delete the git branches already merged or gone",
		Long: "Delete the local git branches already merged or whose remote is gone. It fetches with --prune first, " +
			"then lists every branch but the default one and the one checked out: merged and gone ones come ticked.",
		GroupID: "git",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !core.InRepo("") {
				return core.Fail("not in a git repo", "")
			}
			spin := ui.Spin("fetching")
			core.Fetch(cmd.Context(), "")
			spin.Stop()
			if err := cmd.Context().Err(); err != nil {
				return err
			}
			bs, err := core.ListBranches("")
			if err != nil {
				return err
			}
			base := bs.Base
			if base == "" {
				base = "none"
			}
			if len(bs.List) == 0 {
				only := bs.Base
				if only == "" {
					only = bs.Current
				}
				ui.Noop("only " + only + " here, nothing to delete")
				return nil
			}
			w := 0
			for _, b := range bs.List {
				w = max(w, len(b.Name))
			}

			if list || !ui.Interactive() {
				tty := ui.IsTTY(ui.Stdout)
				can := 0
				for _, b := range bs.List {
					why := ui.Pad(b.Why, 8) + " · " + b.Age
					switch {
					case !tty:
						ui.Println(ui.Pad(b.Name, w) + "  " + why)
					case b.CanGo():
						can++
						ui.Println(ui.Generic.Render(ui.Pad(b.Name, w)) + "  " + why)
					default:
						ui.Println(ui.Pad(b.Name, w) + "  " + ui.Dim.Render(why))
					}
				}
				if tty {
					ui.Println(ui.Dim.Render(fmt.Sprintf("%d can go · base %s", can, base)))
				}
				return nil
			}

			var opts []string
			var on []int
			for i, b := range bs.List {
				opts = append(opts, ui.Pad(b.Name, w)+"  "+ui.Dim.Render(ui.Pad(b.Why, 8)+" · "+b.Age))
				if b.CanGo() {
					on = append(on, i)
				}
			}
			picked, err := ui.PickMany(fmt.Sprintf("Delete which branches? (base: %s)", base), opts, on)
			if err != nil {
				return err
			}
			if len(picked) == 0 {
				ui.Noop("none ticked, nothing deleted")
				return nil
			}
			failed := false
			for _, i := range picked {
				b := bs.List[i]
				if err := core.DeleteBranch("", b.Name); err != nil {
					ui.Fail("could not delete "+b.Name, "")
					failed = true
					continue
				}
				ui.OK("deleted "+ui.Bold.Render(b.Name), b.Why)
			}
			if failed {
				return errSilent
			}
			return nil
		},
	}
	c.Flags().BoolVarP(&list, "list", "l", false, "only list them, with why each can go")
	return c
}
