package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jz-wilson/vkit/cmd/ui"
	"github.com/jz-wilson/vkit/internal/moc"
	"github.com/jz-wilson/vkit/internal/rename"
	"github.com/jz-wilson/vkit/internal/vaultpath"
)

var renameDryRun bool

var renameCmd = &cobra.Command{
	Use:   "rename <old> <new>",
	Short: "Link-safe rename/move: scan inbound [[links]], git mv, rewrite them.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		vault, err := vaultRoot()
		if err != nil {
			return err
		}
		res, err := rename.Rename(vault, args[0], args[1], renameDryRun)
		if err != nil {
			return err
		}
		header := ui.Section("✏️", "Rename") + "  " + ui.Dim(args[0]+" → "+args[1])
		if renameDryRun {
			fmt.Println(header)
			fmt.Println(ui.Step(true, "(dry) git mv "+args[0]+" → "+args[1]))
			fmt.Println(ui.Step(true, fmt.Sprintf("(dry) Scanned %d files", res.Scanned)))
			fmt.Println(ui.Step(true, fmt.Sprintf("(dry) Rewrote %d links", res.Rewritten)))
			for _, f := range res.Touched {
				if f != args[1] {
					fmt.Println("    " + ui.Dim(f))
				}
			}
			fmt.Println(ui.Summary("no changes written"))
			return nil
		}
		n, err := moc.Build(vault, vaultpath.Today())
		if err != nil {
			return err
		}
		fmt.Println(header)
		fmt.Println(ui.Step(true, "git mv "+args[0]+" → "+args[1]))
		fmt.Println(ui.Step(true, fmt.Sprintf("Scanned %d files", res.Scanned)))
		fmt.Println(ui.Step(true, fmt.Sprintf("Rewrote %d links", res.Rewritten)))
		for _, f := range res.Touched {
			if f != args[1] {
				fmt.Println("    " + ui.Dim(f))
			}
		}
		fmt.Println(ui.Step(true, fmt.Sprintf("MOC rebuilt (%d notes)", n)))
		fmt.Println(ui.Summary("Renamed "+args[0]+" → "+args[1], fmt.Sprintf("%d files updated", res.Rewritten)))
		return nil
	},
}

func init() {
	renameCmd.Flags().BoolVar(&renameDryRun, "dry-run", false, "show what would change without writing anything")
}
