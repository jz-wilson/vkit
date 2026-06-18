package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jz-wilson/vkit/cmd/style"
	"github.com/jz-wilson/vkit/internal/osdetect"
	"github.com/jz-wilson/vkit/internal/scaffold"
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
		fmt.Println(style.Section("🔧", "Update"))

		hasTTY := osdetect.Detect("").HasTTY
		in, closeIn := promptInput(hasTTY)
		defer closeIn()

		decider := &scaffold.InteractiveDecider{R: in, W: os.Stdout, HasTTY: hasTTY, Vault: vault}
		res, err := scaffold.Update(vault, mode, updDryRun, decider, os.Stdout)
		if err != nil {
			return err
		}
		if res.AlreadyMatches || res.DryRun {
			return nil
		}

		fmt.Println(style.Summary(
			res.Action,
			fmt.Sprintf("%d tooling, %d added, %d overwritten, %d kept", res.Tool, res.New, res.Over, res.Keep),
		))
		fmt.Println(style.Dim("  Backups (if any) written as <file>.bak — review & commit yourself."))
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
