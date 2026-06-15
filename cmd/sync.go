package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jz-wilson/vkit/cmd/ui"
	"github.com/jz-wilson/vkit/internal/moc"
	"github.com/jz-wilson/vkit/internal/vaultpath"
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
		fmt.Println(ui.Line("🔄", ui.OK(fmt.Sprintf("Rebuilt MOC.md (%d notes)", n))))

		// Show status first.
		_ = git(vault, "status", "--short")

		// Stage ONLY documentation assets — never `git add -A`. Always stage the
		// *.md glob and MOC.md; only include named dirs that exist so a missing dir
		// can't abort the whole git add (git treats a missing literal pathspec as fatal).
		// This set mirrors scaffold.contentDirs minus "archive" (archived notes are
		// not staged by sync — they are committed explicitly when archived).
		addArgs := []string{"add", "--", "*.md", "MOC.md"}
		for _, d := range []string{"decisions", "infrastructure", "projects", "reference"} {
			if fi, err := os.Stat(filepath.Join(vault, d)); err == nil && fi.IsDir() {
				addArgs = append(addArgs, d)
			}
		}
		_ = git(vault, addArgs...)

		msg := syncMsg
		if msg == "" {
			msg = "vault: sync"
		}
		if err := gitCommit(vault, msg, false); err != nil {
			fmt.Fprintln(os.Stderr, ui.Fail("nothing committed (no staged changes?)"))
			return nil
		}
		fmt.Println(ui.OK(fmt.Sprintf("Committed: %s", ui.Dim(msg))))
		return nil
	},
}

func init() {
	syncCmd.Flags().StringVarP(&syncMsg, "message", "m", "", "commit message (default: \"vault: sync\")")
}
