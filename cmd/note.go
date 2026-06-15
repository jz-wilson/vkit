package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jz-wilson/vkit/internal/moc"
	"github.com/jz-wilson/vkit/internal/note"
	"github.com/jz-wilson/vkit/internal/vaultpath"
)

var (
	noteTitle string
	noteTags  string
)

var noteCmd = &cobra.Command{
	Use:   "note <path>",
	Short: "Scaffold a new note from the schema (refuses to overwrite).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vault, err := vaultRoot()
		if err != nil {
			return err
		}
		relPath := note.EnsureMD(args[0])
		var tags []string
		for _, t := range strings.Split(noteTags, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
		today := vaultpath.Today()

		creator := note.New(vault)
		if err := creator.Create(vault, relPath, noteTitle, tags, today); err != nil {
			return err
		}
		if _, err := moc.Build(vault, today); err != nil {
			return err
		}
		fmt.Printf("Created %s\n", relPath)
		return nil
	},
}

func init() {
	noteCmd.Flags().StringVar(&noteTitle, "title", "", "note title (default: derived from filename)")
	noteCmd.Flags().StringVar(&noteTags, "tags", "", "comma-separated tags")
}
