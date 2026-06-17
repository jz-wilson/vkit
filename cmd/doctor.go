package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jz-wilson/vkit/cmd/style"
	"github.com/jz-wilson/vkit/internal/osdetect"
	"github.com/jz-wilson/vkit/internal/vaultpath"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor [vault]",
	Short: "Print detected OS, package manager, systemd, tty, and Obsidian state.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := ""
		if len(args) > 0 {
			arg = args[0]
		}
		vault, _ := vaultpath.Resolve(arg, vaultFlag)
		info := osdetect.Detect(vault)

		fmt.Println(style.Section("🩺", "System"))
		fmt.Println(style.Row("OS", info.OS))
		fmt.Println(style.Row("Pkg mgr", info.PkgMgr))
		fmt.Println(style.Row("systemd", style.Check(info.SystemdUser)))
		fmt.Println(style.Row("TTY", style.Check(info.HasTTY)))

		fmt.Println()
		fmt.Println(style.Section("🔮", "Obsidian"))
		fmt.Println(style.Row("Binary", style.Check(info.ObsidianBinary)))
		fmt.Println(style.Row("CLI mode", obsidianCLIStatus(info)))
		fmt.Println(style.Row("Vault", style.Dim(vault)))
		return nil
	},
}

func obsidianCLIStatus(info osdetect.Info) string {
	switch {
	case info.ObsidianCLI:
		return style.StyleSuccess.Render("enabled")
	case info.ObsidianBinary:
		return style.StyleDim.Render("disabled (VAULT_OBSIDIAN_CLI=0)")
	default:
		return style.StyleDim.Render("binary not found")
	}
}
