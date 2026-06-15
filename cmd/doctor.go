package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jz-wilson/vkit/cmd/ui"
	"github.com/jz-wilson/vkit/internal/osdetect"
	"github.com/jz-wilson/vkit/internal/vaultpath"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Print detected OS, package manager, systemd, tty, and Obsidian state.",
	RunE: func(cmd *cobra.Command, args []string) error {
		vault, _ := vaultpath.Resolve("", vaultFlag)
		info := osdetect.Detect(vault)

		fmt.Println(ui.Section("🩺", "System"))
		fmt.Println(ui.Row("OS", info.OS))
		fmt.Println(ui.Row("Pkg mgr", info.PkgMgr))
		fmt.Println(ui.Row("systemd", ui.Check(info.SystemdUser)))
		fmt.Println(ui.Row("TTY", ui.Check(info.HasTTY)))

		fmt.Println()
		fmt.Println(ui.Section("🔮", "Obsidian"))
		fmt.Println(ui.Row("Binary", ui.Check(info.ObsidianBinary)))
		fmt.Println(ui.Row("CLI mode", obsidianCLIStatus(info)))
		fmt.Println(ui.Row("Vault", ui.Dim(vault)))
		return nil
	},
}

func obsidianCLIStatus(info osdetect.Info) string {
	switch {
	case info.ObsidianCLI:
		return ui.StyleSuccess.Render("enabled")
	case info.ObsidianBinary:
		return ui.StyleDim.Render("disabled (VAULT_OBSIDIAN_CLI=0)")
	default:
		return ui.StyleDim.Render("binary not found")
	}
}
