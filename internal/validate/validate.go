// Package validate ports .githooks/pre-commit: it checks markdown notes for
// frontmatter on line 1, a present `updated:` field, exactly one H1, and no
// absolute filesystem paths in the body. It uses the same skip list as the hook.
package validate

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"vkit/internal/vaultpath"
)

// Problem is one validation failure.
type Problem struct {
	File string
	Msg  string
}

var absPathRe = regexp.MustCompile(`(/Users/|/mnt/|file://|[A-Z]:\\)`)

// ShouldSkip reports whether a path (relative, slash-separated) is exempt from
// validation. It defers to the shared note classifier and adds validate's own
// README.md exemption (docs, not a note).
func ShouldSkip(rel string) bool {
	rel = filepath.ToSlash(rel)
	if !vaultpath.IsNote(rel) {
		return true
	}
	return filepath.Base(rel) == "README.md"
}

// Files validates the given relative paths against the vault root and returns
// all problems found.
func Files(vaultRoot string, rels []string) ([]Problem, error) {
	var probs []Problem
	for _, rel := range rels {
		if ShouldSkip(rel) {
			continue
		}
		path := filepath.Join(vaultRoot, rel)
		fi, err := os.Stat(path)
		if err != nil || fi.IsDir() {
			continue
		}
		probs = append(probs, checkFile(rel, path)...)
	}
	return probs, nil
}

func checkFile(rel, path string) []Problem {
	var probs []Problem
	f, err := os.Open(path)
	if err != nil {
		return probs
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}

	if len(lines) == 0 || lines[0] != "---" {
		probs = append(probs, Problem{rel, "frontmatter must start at line 1"})
	}

	hasUpdated := false
	h1 := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "updated:") {
			hasUpdated = true
		}
		if strings.HasPrefix(line, "# ") {
			h1++
		}
	}
	if !hasUpdated {
		probs = append(probs, Problem{rel, "missing 'updated:' frontmatter"})
	}
	if h1 != 1 {
		probs = append(probs, Problem{rel, "expected exactly one H1, found " + strconv.Itoa(h1)})
	}

	// crude absolute-path check (ignores lines that are fence markers)
	for _, line := range lines {
		if absPathRe.MatchString(line) && !strings.Contains(line, "```") {
			probs = append(probs, Problem{rel, "absolute path in body — use [[wikilinks]]"})
			break
		}
	}
	return probs
}

// StagedFiles returns the staged (added/copied/modified) paths from git, as
// slash-separated paths relative to the repo root.
func StagedFiles(vaultRoot string) ([]string, error) {
	cmd := exec.Command("git", "-C", vaultRoot, "diff", "--cached", "--name-only", "-z", "--diff-filter=ACM")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var rels []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			rels = append(rels, filepath.ToSlash(p))
		}
	}
	return rels, nil
}
