package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"vkit/internal/osdetect"
	"vkit/internal/vaultpath"
)

func obsidianCLIStatus(info osdetect.Info) string {
	switch {
	case info.ObsidianCLI:
		return "enabled"
	case info.ObsidianBinary:
		return "installed (disabled via VAULT_OBSIDIAN_CLI=0)"
	default:
		return "false"
	}
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Print detected OS, package manager, systemd, tty, and Obsidian state.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Best-effort vault resolution (doctor works even without a vault).
		vault, _ := vaultpath.Resolve("", vaultFlag)
		info := osdetect.Detect(vault)
		fmt.Printf("os:           %s\n", info.OS)
		fmt.Printf("pkgmgr:       %s\n", info.PkgMgr)
		fmt.Printf("systemd-user: %v\n", info.SystemdUser)
		fmt.Printf("tty:          %v\n", info.HasTTY)
		fmt.Printf("obsidian-cli: %s\n", obsidianCLIStatus(info))
		fmt.Printf("vault:        %s\n", vault)
		return nil
	},
}
