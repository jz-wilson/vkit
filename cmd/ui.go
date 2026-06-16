package cmd

import (
	"github.com/spf13/cobra"

	"github.com/jz-wilson/vkit/internal/tui"
)

var uiCmd = &cobra.Command{
	Use:   "ui [path]",
	Short: "Launch the interactive multi-pane vault TUI (nav, workspace, validation log).",
	Long: `ui opens an immersive terminal UI for the vault: browse notes on the left,
read the selected note in a scrolling workspace, and watch validation problems
update live as files change on disk.

Keys: tab cycles panels, j/k or arrows move, v validates the highlighted note,
m rebuilds MOC.md, q or ctrl+c quits.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vault, err := resolveExisting(args)
		if err != nil {
			return err
		}
		return tui.Run(vault)
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
