package cmd

import "github.com/spf13/cobra"

// Version is the build version. It defaults to "dev" and is overridden at
// release time: goreleaser injects it into package main via
// -ldflags "-X main.version=...", and main passes it to Execute.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the vkit version.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		cmd.Println("vkit version", Version)
	},
}

// setVersion updates both the package var and cobra's built-in --version flag
// source so the two stay in sync.
func setVersion(v string) {
	Version = v
	rootCmd.Version = v
}
