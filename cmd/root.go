// Package cmd is the cobra layer for vkit. It stays thin — every command
// resolves the vault root and delegates to an internal package.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jz-wilson/vkit/internal/vaultpath"
)

var vaultFlag string

var rootCmd = &cobra.Command{
	Use:   "vkit",
	Short: "Cross-platform vault starter kit — one binary, no shell scripts.",
	Long: `vkit scaffolds and maintains a plain-folder knowledge vault that Claude Code
reads live off disk: a Map of Content index, frontmatter validation, a
file watcher, and link-safe note operations — all in one static binary.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&vaultFlag, "vault", "", "vault root (default: $VKIT_VAULT, walk-up to _format.md, or $HOME/vault)")
	rootCmd.Version = Version // enables the built-in --version flag
	rootCmd.AddCommand(initCmd, updateCmd, validateCmd, noteCmd, renameCmd, syncCmd, doctorCmd, versionCmd)
}

// Execute runs the root command. v is the build version injected by main
// (goreleaser ldflags); an empty value keeps the "dev" default.
func Execute(v string) {
	if v != "" {
		setVersion(v)
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "vkit:", err)
		os.Exit(1)
	}
}

// resolveVault resolves the vault from arg + the --vault flag and requires it to
// exist.
func resolveVault(arg string) (string, error) {
	v, err := vaultpath.Resolve(arg, vaultFlag)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(v); err != nil {
		return "", fmt.Errorf("vault not found: %s", v)
	}
	return v, nil
}

// resolveExisting resolves the vault for commands whose first positional arg IS
// the vault path (init/update).
func resolveExisting(args []string) (string, error) {
	arg := ""
	if len(args) > 0 {
		arg = args[0]
	}
	return resolveVault(arg)
}

// vaultRoot resolves the vault for commands whose positional args are NOT the
// vault path (note, rename, sync, validate).
func vaultRoot() (string, error) {
	return resolveVault("")
}
