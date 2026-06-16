package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jz-wilson/vkit/cmd/style"
	"github.com/jz-wilson/vkit/internal/config"
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

		fmt.Println(style.Section("🔄", "Sync"))

		n, err := moc.Build(vault, vaultpath.Today())
		if err != nil {
			return err
		}
		fmt.Println(style.Step(true, fmt.Sprintf("Rebuilt MOC.md (%d notes)", n)))

		// Stage ONLY documentation assets — never `git add -A`.
		addArgs := []string{"add", "--", "*.md", "MOC.md"}
		for _, d := range config.ContentDirs(vault) {
			if fi, err := os.Stat(filepath.Join(vault, d)); err == nil && fi.IsDir() {
				addArgs = append(addArgs, d)
			}
		}
		if err := git(vault, addArgs...); err != nil {
			return fmt.Errorf("git add: %w", err)
		}

		if stagedStr, err := gitOutput(vault, "diff", "--cached", "--name-only"); err == nil && stagedStr != "" {
			lines := strings.Split(stagedStr, "\n")
			fmt.Println(style.Step(true, fmt.Sprintf("Staged %d files", len(lines))))
			for _, f := range lines {
				if f != "" {
					fmt.Println("    " + style.Dim(f))
				}
			}
		}

		msg := syncMsg
		if msg == "" {
			msg = "vault: sync"
		}
		if err := gitCommit(vault, msg, false); err != nil {
			fmt.Fprintln(os.Stderr, style.Step(false, "nothing committed (no staged changes?)"))
			return nil
		}
		fmt.Println(style.Step(true, "Committed: "+style.Dim(msg)))
		return nil
	},
}

func gitOutput(vault string, args ...string) (string, error) {
	full := append([]string{"-C", vault}, args...)
	out, err := exec.Command("git", full...).Output()
	return strings.TrimSpace(string(out)), err
}

func init() {
	syncCmd.Flags().StringVarP(&syncMsg, "message", "m", "", "commit message (default: \"vault: sync\")")
}
