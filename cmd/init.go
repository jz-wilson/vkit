package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"vkit/internal/moc"
	"vkit/internal/osdetect"
	"vkit/internal/scaffold"
	"vkit/internal/vaultpath"
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Scaffold a new vault (install mode).",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := ""
		if len(args) > 0 {
			arg = args[0]
		}
		vault, err := vaultpath.Resolve(arg, vaultFlag)
		if err != nil {
			return err
		}
		if vaultpath.IsVault(vault) {
			return fmt.Errorf("%s is already a vault (_format.md present) — use `vkit update`", vault)
		}

		if err := scaffold.Install(vault); err != nil {
			return err
		}

		// git: init if absent, always (re)point hooksPath at our hook dir.
		if _, err := os.Stat(filepath.Join(vault, ".git")); err != nil {
			if err := git(vault, "init", "-q"); err != nil {
				return err
			}
		}
		if err := git(vault, "config", "core.hooksPath", ".githooks"); err != nil {
			return err
		}

		n, err := moc.Build(vault, vaultpath.Today())
		if err != nil {
			return err
		}

		// Initial commit (fresh install only — safe to add everything).
		// --no-verify: the pre-commit hook calls `vkit`, which may not be on PATH
		// yet at bootstrap, and the generated scaffold is known-valid anyway.
		_ = git(vault, "add", "-A")
		_ = gitCommit(vault, "vault: initial scaffold", true)

		info := osdetect.Detect(vault)
		fmt.Printf("Done.\n\n")
		fmt.Printf("  OS:       %s\n", info.OS)
		fmt.Printf("  Vault:    %s\n", vault)
		fmt.Printf("  Index:    MOC.md (%d notes)\n", n)
		fmt.Printf("  Obsidian: %s\n", obsidianStatus(info, vault))
		fmt.Printf("\nKeep the index fresh with `vkit watch --vault %s` (or install a\nservice from %s).\n", vault, filepath.Join(vault, "services"))
		return nil
	},
}

func obsidianStatus(info osdetect.Info, vault string) string {
	if info.ObsidianCLI {
		return "native mode ENABLED (.obsidian-cli-enabled present)"
	}
	return "portable core (default). Enable native mode: touch " + filepath.Join(vault, ".obsidian-cli-enabled")
}

func git(vault string, args ...string) error {
	full := append([]string{"-C", vault}, args...)
	cmd := exec.Command("git", full...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// gitCommit commits with identity fallbacks so it works on headless boxes that
// have no global git user configured. noVerify skips the pre-commit hook.
func gitCommit(vault, msg string, noVerify bool) error {
	args := []string{"-C", vault,
		"-c", "user.name=vkit", "-c", "user.email=vkit@local",
		"commit", "-q", "-m", msg}
	if noVerify {
		args = append(args, "--no-verify")
	}
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
