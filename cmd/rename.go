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
		touched, err := rename.Rename(vault, args[0], args[1], renameDryRun)
		if err != nil {
			return err
		}
		if renameDryRun {
			fmt.Println(ui.Line("🔍", ui.Dim(fmt.Sprintf("Dry run: %s → %s", args[0], args[1]))))
			fmt.Println(ui.Dim(fmt.Sprintf("  Would touch %d file(s):", len(touched))))
			for _, f := range touched {
				fmt.Printf("    %s\n", ui.Dim(f))
			}
			fmt.Println(ui.Dim("  (no changes written)"))
			return nil
		}
		if _, err := moc.Build(vault, vaultpath.Today()); err != nil {
			return err
		}
		fmt.Println(ui.Line("✏️", ui.OK(fmt.Sprintf("Renamed %s → %s", ui.Dim(args[0]), ui.Dim(args[1])))))
		fmt.Println(ui.Dim(fmt.Sprintf("  Touched %d file(s):", len(touched))))
		for _, f := range touched {
			fmt.Printf("    %s\n", ui.Dim(f))
		}
		return nil
	},
}

func init() {
	renameCmd.Flags().BoolVar(&renameDryRun, "dry-run", false, "show what would change without writing anything")
}
