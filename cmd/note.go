package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jz-wilson/vkit/cmd/style"
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
		resolvedPath, err := creator.Create(vault, relPath, noteTitle, tags, today)
		if err != nil {
			return err
		}
		title := noteTitle
		if title == "" {
			title = note.DeriveTitle(relPath)
		}
		fmt.Println(style.Section("📝", "Note created"))
		fmt.Println(style.Step(true, style.Dim(resolvedPath)))
		fmt.Println(style.Row("title", title))
		if len(tags) > 0 {
			fmt.Println(style.Row("tags", "["+strings.Join(tags, ", ")+"]"))
		}
		return nil
	},
}

func init() {
	noteCmd.Flags().StringVar(&noteTitle, "title", "", "note title (default: derived from filename)")
	noteCmd.Flags().StringVar(&noteTags, "tags", "", "comma-separated tags")
}
