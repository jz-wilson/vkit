package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"vkit/internal/moc"
	"vkit/internal/vaultpath"
)

var syncMsg string

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Rebuild the index and commit docs only (never `git add -A`).",
	RunE: func(cmd *cobra.Command, args []string) error {
		vault, err := vaultRoot()
		if err != nil {
			return err
		}
		n, err := moc.Build(vault, vaultpath.Today())
		if err != nil {
			return err
		}
		fmt.Printf("Rebuilt MOC.md (%d notes)\n", n)

		// Show status first.
		_ = git(vault, "status", "--short")

		// Stage ONLY documentation assets — never `git add -A`. Pathspecs cover
		// every markdown file plus the note dirs; non-doc files are left alone.
		addArgs := []string{"add", "--", "*.md", "MOC.md",
			"decisions", "infrastructure", "projects", "reference"}
		_ = git(vault, addArgs...)

		msg := syncMsg
		if msg == "" {
			msg = "vault: sync"
		}
		if err := gitCommit(vault, msg, false); err != nil {
			fmt.Fprintln(os.Stderr, "nothing committed (no staged changes?)")
			return nil
		}
		fmt.Printf("Committed: %s\n", msg)
		return nil
	},
}

func init() {
	syncCmd.Flags().StringVarP(&syncMsg, "message", "m", "", "commit message (default: \"vault: sync\")")
}
