package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jz-wilson/vkit/cmd/ui"
	"github.com/jz-wilson/vkit/internal/moc"
	"github.com/jz-wilson/vkit/internal/vaultpath"
)

var mocCmd = &cobra.Command{
	Use:   "moc [path]",
	Short: "Regenerate MOC.md (the Map of Content index).",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vault, err := resolveExisting(args)
		if err != nil {
			return err
		}
		n, err := moc.Build(vault, vaultpath.Today())
		if err != nil {
			return err
		}
		fmt.Println(ui.Line("🗂️", ui.OK(fmt.Sprintf("Wrote MOC.md (%d notes)", n))))
		return nil
	},
}
