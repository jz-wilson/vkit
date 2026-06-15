package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jz-wilson/vkit/cmd/ui"
	"github.com/jz-wilson/vkit/internal/moc"
	"github.com/jz-wilson/vkit/internal/osdetect"
	"github.com/jz-wilson/vkit/internal/scaffold"
	"github.com/jz-wilson/vkit/internal/vaultpath"
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
		fmt.Println(ui.Step(true, "Scaffolded vault structure"))

		// git: init if absent, always (re)point hooksPath at our hook dir.
		if _, err := os.Stat(filepath.Join(vault, ".git")); err != nil {
			if err := git(vault, "init", "-q"); err != nil {
				return err
			}
			fmt.Println(ui.Step(true, "git init"))
		}
		if err := git(vault, "config", "core.hooksPath", ".githooks"); err != nil {
			return err
		}
		fmt.Println(ui.Step(true, "Set core.hooksPath → .githooks"))

		n, err := moc.Build(vault, vaultpath.Today())
		if err != nil {
			return err
		}
		fmt.Println(ui.Step(true, fmt.Sprintf("Built MOC.md (%d notes)", n)))

		// Initial commit (fresh install only — safe to add everything).
		// --no-verify: the pre-commit hook calls `vkit`, which may not be on PATH
		// yet at bootstrap, and the generated scaffold is known-valid anyway.
		_ = git(vault, "add", "-A")
		_ = gitCommit(vault, "vault: initial scaffold", true)
		fmt.Println(ui.Step(true, "Initial commit"))

		fmt.Println()
		info := osdetect.Detect(vault)
		fmt.Println(ui.Line("🚀", ui.StyleLabel.Render("Vault initialized")))
		fmt.Println()
		fmt.Println(ui.Row("OS", info.OS))
		fmt.Println(ui.Row("Vault", ui.Dim(vault)))
		fmt.Println(ui.Row("Index", fmt.Sprintf("MOC.md (%d notes)", n)))
		fmt.Println(ui.Row("Obsidian", obsidianStatus(info)))
		fmt.Printf("\n%s\n", ui.Dim(fmt.Sprintf("Keep the index fresh with `vkit watch --vault %s`\n(or install a service from %s)", vault, filepath.Join(vault, "services"))))
		return nil
	},
}

func obsidianStatus(info osdetect.Info) string {
	if info.ObsidianCLI {
		return ui.StyleSuccess.Render("native mode enabled")
	}
	if info.ObsidianBinary {
		return ui.StyleDim.Render("native mode disabled (unset VAULT_OBSIDIAN_CLI=0 to enable)")
	}
	return ui.StyleDim.Render("portable mode (obsidian binary not found)")
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
