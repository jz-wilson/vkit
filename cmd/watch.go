package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"vkit/internal/watcher"
)

var (
	watchPoll     bool
	watchInterval int
)

var watchCmd = &cobra.Command{
	Use:   "watch [path]",
	Short: "Watch the vault and rebuild MOC.md on every change.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vault, err := resolveExisting(args)
		if err != nil {
			return err
		}
		return watcher.Watch(vault, watchPoll, time.Duration(watchInterval)*time.Second)
	},
}

func init() {
	watchCmd.Flags().BoolVar(&watchPoll, "poll", false, "force the mtime-polling backend (no fsnotify)")
	watchCmd.Flags().IntVar(&watchInterval, "interval", 5, "poll interval in seconds (polling backend)")
}
