package rename

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"vkit/internal/vaultpath"
)

// LinkRewriter rewrites Obsidian wiki-links in note content when a note is
// renamed. It is pure (string in, string out) and has no filesystem or git
// dependency, making it independently unit-testable.
type LinkRewriter struct{}

// Rewrite replaces all wiki-link occurrences of oldStem with newStem inside
// content. oldStem and newStem may be bare filenames ("old-note") or
// vault-relative paths without the .md extension ("folder/old-note"). Both
// the full path form and the basename-only form are rewritten so that
// [[folder/old-note]], [[old-note]], [[old-note|alias]], and
// [[old-note#section]] are all updated correctly.
func (LinkRewriter) Rewrite(content, oldStem, newStem string) string {
	type rule struct {
		re  *regexp.Regexp
		rep string
	}

	oldBase := filepath.Base(oldStem)
	newBase := filepath.Base(newStem)

	rules := []rule{
		{linkRe(oldStem), "[[" + newStem + "${1}"},
	}
	if oldBase != oldStem {
		rules = append(rules, rule{linkRe(oldBase), "[[" + newBase + "${1}"})
	}

	out := content
	for _, r := range rules {
		out = r.re.ReplaceAllString(out, r.rep)
	}
	return out
}

func Rename(vault, oldRel, newRel string) ([]string, error) {
	oldRel = filepath.ToSlash(oldRel)
	newRel = filepath.ToSlash(newRel)

	oldFull := filepath.Join(vault, oldRel)
	if _, err := os.Stat(oldFull); err != nil {
		return nil, fmt.Errorf("source note not found: %s", oldRel)
	}
	if _, err := os.Stat(filepath.Join(vault, newRel)); err == nil {
		return nil, fmt.Errorf("destination already exists: %s", newRel)
	}

	oldNoExt := strings.TrimSuffix(oldRel, ".md")
	newNoExt := strings.TrimSuffix(newRel, ".md")

	if err := os.MkdirAll(filepath.Dir(filepath.Join(vault, newRel)), 0o755); err != nil {
		return nil, err
	}
	if err := gitMv(vault, oldRel, newRel); err != nil {
		return nil, err
	}

	rw := LinkRewriter{}
	touched := map[string]bool{}
	touched[newRel] = true

	err := filepath.WalkDir(vault, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != vault && vaultpath.IsExcludedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		orig := string(b)
		out := rw.Rewrite(orig, oldNoExt, newNoExt)
		if out != orig {
			if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
				return err
			}
			rel, _ := filepath.Rel(vault, path)
			touched[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(touched))
	for f := range touched {
		out = append(out, f)
	}
	sort.Strings(out)
	return out, nil
}

func linkRe(target string) *regexp.Regexp {
	return regexp.MustCompile(`\[\[` + regexp.QuoteMeta(target) + `([\]|#])`)
}

func gitMv(vault, oldRel, newRel string) error {
	cmd := exec.Command("git", "-C", vault, "mv", oldRel, newRel)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
