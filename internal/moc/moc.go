// Package moc ports build-moc.sh: it regenerates MOC.md as one line per note,
// using each note's first H1 as its title, byte-sorted, with the same
// exclusions the bash kit applied.
package moc

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"vkit/internal/vaultpath"
)

// Build regenerates MOC.md in the vault and returns the note count.
func Build(vault, today string) (int, error) {
	content, n, err := Generate(vault, today)
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(filepath.Join(vault, "MOC.md"), content, 0o644); err != nil {
		return 0, err
	}
	return n, nil
}

// Generate produces the MOC.md bytes and the note count without writing.
func Generate(vault, today string) ([]byte, int, error) {
	rels, err := collectNotes(vault)
	if err != nil {
		return nil, 0, err
	}
	// LC_ALL=C sort == byte order; Go string comparison is byte order.
	sort.Strings(rels)

	var b strings.Builder
	b.WriteString("# Map of Content\n\n")
	fmt.Fprintf(&b, "_Auto-generated %s by vkit moc. Do not edit by hand._\n\n", today)
	for _, rel := range rels {
		title := firstH1(filepath.Join(vault, rel))
		if title == "" {
			title = "untitled"
		}
		noExt := strings.TrimSuffix(rel, ".md")
		fmt.Fprintf(&b, "- [[%s]] — %s\n", noExt, title)
	}
	return []byte(b.String()), len(rels), nil
}

// collectNotes returns slash-separated relative paths of every note via the
// shared vault walker.
func collectNotes(vault string) ([]string, error) {
	var rels []string
	err := vaultpath.WalkNotes(vault, func(rel string) error {
		rels = append(rels, rel)
		return nil
	})
	return rels, err
}

// firstH1 returns the text after the first "# " line in the body (after
// frontmatter), skipping lines inside code fences.
func firstH1(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Skip frontmatter if present; otherwise check the first line for H1.
	if sc.Scan() {
		first := sc.Text()
		if first == "---" {
			for sc.Scan() {
				if sc.Text() == "---" {
					break
				}
			}
		} else if strings.HasPrefix(first, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(first, "# "))
		}
	}

	inFence := false
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if !inFence && strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}
