// Package rename ports the /note rename Tier B logic: a link-safe move that
// scans inbound [[wikilinks]], does a git mv (preserving history), and rewrites
// every inbound link from the old target to the new one.
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

// Rename moves oldRel -> newRel (both relative to vault) and rewrites inbound
// wikilinks. It returns the sorted list of files it touched (the rewritten link
// files plus the moved note).
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
	oldBase := filepath.Base(oldNoExt)
	newBase := filepath.Base(newNoExt)

	// link rewrite rules: full relpath form, and bare basename form.
	type rule struct {
		re  *regexp.Regexp
		rep string
	}
	rules := []rule{
		{linkRe(oldNoExt), "[[" + newNoExt + "${1}"},
	}
	if oldBase != oldNoExt {
		rules = append(rules, rule{linkRe(oldBase), "[[" + newBase + "${1}"})
	}

	touched := map[string]bool{}

	// Scan every note for inbound links and rewrite in place.
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
		out := orig
		for _, r := range rules {
			out = r.re.ReplaceAllString(out, r.rep)
		}
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

	// Ensure destination dir exists, then git mv (preserves history).
	if err := os.MkdirAll(filepath.Dir(filepath.Join(vault, newRel)), 0o755); err != nil {
		return nil, err
	}
	if err := gitMv(vault, oldRel, newRel); err != nil {
		return nil, err
	}
	touched[newRel] = true

	out := make([]string, 0, len(touched))
	for f := range touched {
		out = append(out, f)
	}
	sort.Strings(out)
	return out, nil
}

// linkRe matches `[[target` followed by a link terminator (]] , | alias, or #
// heading), capturing that terminator so the replacement can preserve it. Go's
// regexp has no lookahead, so the terminator is a capture group.
func linkRe(target string) *regexp.Regexp {
	return regexp.MustCompile(`\[\[` + regexp.QuoteMeta(target) + `([\]|#])`)
}

func gitMv(vault, oldRel, newRel string) error {
	cmd := exec.Command("git", "-C", vault, "mv", oldRel, newRel)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
