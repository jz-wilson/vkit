package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jz-wilson/vkit/cmd/ui"
	"github.com/jz-wilson/vkit/internal/validate"
	"github.com/jz-wilson/vkit/internal/vaultpath"
)

var valStaged bool

var validateCmd = &cobra.Command{
	Use:   "validate [files...]",
	Short: "Validate note frontmatter (line-1 ---, updated:, one H1, no abs paths).",
	RunE: func(cmd *cobra.Command, args []string) error {
		// validate's positional args are FILES, not a vault path, so resolve the
		// vault from --vault / env / walk-up only.
		vault, err := vaultRoot()
		if err != nil {
			return err
		}

		var rels []string
		switch {
		case valStaged:
			rels, err = validate.StagedFiles(vault)
			if err != nil {
				return err
			}
		case len(args) > 0:
			for _, a := range args {
				abs, err := filepath.Abs(a)
				if err != nil {
					return err
				}
				rel, err := filepath.Rel(vault, abs)
				if err != nil {
					return err
				}
				rels = append(rels, filepath.ToSlash(rel))
			}
		default:
			rels, err = allNotes(vault)
			if err != nil {
				return err
			}
		}

		probs, err := validate.Files(vault, rels)
		if err != nil {
			return err
		}
		if len(probs) > 0 {
			fmt.Fprintln(os.Stderr, ui.Line("🔍", ui.Fail(fmt.Sprintf("Validation failed (%d notes checked)", len(rels)))))
			for _, p := range probs {
				fmt.Fprintf(os.Stderr, "    %s: %s\n", ui.Dim(p.File), p.Msg)
			}
			fmt.Fprintln(os.Stderr, ui.Dim("  Fix the above, then re-commit."))
			os.Exit(1)
		}
		fmt.Println(ui.Line("🔍", ui.OK(fmt.Sprintf("%d notes valid", len(rels)))))
		return nil
	},
}

func allNotes(vault string) ([]string, error) {
	var rels []string
	err := vaultpath.WalkNotes(vault, func(rel string) error {
		if !validate.ShouldSkip(rel) {
			rels = append(rels, rel)
		}
		return nil
	})
	return rels, err
}

func init() {
	validateCmd.Flags().BoolVar(&valStaged, "staged", false, "validate git-staged files (pre-commit hook mode)")
}
