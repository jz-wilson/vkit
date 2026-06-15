package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"vkit/internal/moc"
	"vkit/internal/osdetect"
	"vkit/internal/scaffold"
	"vkit/internal/vaultpath"
)

var (
	updForce  bool
	updKeep   bool
	updDryRun bool
)

var updateCmd = &cobra.Command{
	Use:   "update [path]",
	Short: "Eval-first update of an existing vault from the embedded kit.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vault, err := resolveExisting(args)
		if err != nil {
			return err
		}

		mode := scaffold.ModePrompt
		switch {
		case updForce:
			mode = scaffold.ModeForce
		case updKeep:
			mode = scaffold.ModeKeep
		}

		// have_tty(): a real /dev/tty must be openable for the prompt to run.
		hasTTY := osdetect.Detect("").HasTTY
		in, closeIn := promptInput(hasTTY)
		defer closeIn()

		res, err := scaffold.Update(vault, mode, updDryRun, in, os.Stdout, hasTTY)
		if err != nil {
			return err
		}
		if res.AlreadyMatches || res.DryRun {
			return nil
		}

		// Update never auto-commits — rebuild the index only.
		if _, err := moc.Build(vault, vaultpath.Today()); err != nil {
			return err
		}

		fmt.Printf("\nDone (update — %s).\n", res.Action)
		fmt.Printf("  Tooling:   %d refreshed\n", res.Tool)
		fmt.Printf("  Templates: %d added, %d overwritten, %d kept\n", res.New, res.Over, res.Keep)
		fmt.Println("  Rebuilt MOC.md. Backups (if any) written as <file>.bak. Review & commit yourself (no auto-commit).")
		return nil
	},
}

// promptInput returns the reader for the interactive menu. When a real tty is
// available it reads from /dev/tty (matching the bash kit); otherwise stdin.
func promptInput(hasTTY bool) (in *os.File, closer func()) {
	if hasTTY {
		if f, err := os.Open("/dev/tty"); err == nil {
			return f, func() { _ = f.Close() }
		}
	}
	return os.Stdin, func() {}
}

func init() {
	updateCmd.Flags().BoolVarP(&updForce, "force", "f", false, "apply all (overwrite changed templates, with .bak)")
	updateCmd.Flags().BoolVar(&updKeep, "keep", false, "safe: refresh tooling + add new templates, keep changed ones")
	updateCmd.Flags().BoolVar(&updDryRun, "dry-run", false, "show the plan and exit without writing")
}
