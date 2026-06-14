package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vkit/internal/moc"
	"vkit/internal/vaultpath"
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
		fmt.Printf("Wrote MOC.md (%d notes)\n", n)
		return nil
	},
}
