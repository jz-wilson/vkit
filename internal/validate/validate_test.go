package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestShouldSkip(t *testing.T) {
	cases := map[string]bool{
		"projects/a.md":           false,
		"decisions/2026-06.md":    false,
		"MOC.md":                  true,
		"CLAUDE.md":               true,
		"_format.md":              true,
		"README.md":               true,
		"AGENTS.md":               true,
		".claude/commands/x.md":   true,
		".githooks/pre-commit":    true,
		"scripts/build.md":        true,
		"services/s.md":           true,
		"projects/archive/old.md": true,
		"notes/file.txt":          true,
	}
	for rel, want := range cases {
		if got := ShouldSkip(rel); got != want {
			t.Errorf("ShouldSkip(%q)=%v want %v", rel, got, want)
		}
	}
}

func TestFiles(t *testing.T) {
	v := t.TempDir()

	write(t, v, "good.md", "---\nupdated: 2026-06-14\ntags: [x]\n---\n\n# Good Note\n\nbody [[link]]\n")
	write(t, v, "nofront.md", "# No Frontmatter\n\nbody\n")
	write(t, v, "noupdated.md", "---\ntags: [x]\n---\n\n# No Updated\n")
	write(t, v, "multih1.md", "---\nupdated: 2026-06-14\n---\n\n# One\n\n# Two\n")
	write(t, v, "noh1.md", "---\nupdated: 2026-06-14\n---\n\nno heading\n")
	write(t, v, "abspath.md", "---\nupdated: 2026-06-14\n---\n\n# Abs\n\nsee /mnt/c/secret\n")
	write(t, v, "fenced.md", "---\nupdated: 2026-06-14\n---\n\n# Fenced\n\n```\n/mnt/c/ok\n```\n")

	check := func(rel string, wantProblem bool) {
		t.Helper()
		probs, err := Files(v, []string{rel})
		if err != nil {
			t.Fatal(err)
		}
		if (len(probs) > 0) != wantProblem {
			t.Errorf("%s: problems=%v wantProblem=%v", rel, probs, wantProblem)
		}
	}

	check("good.md", false)
	check("nofront.md", true)
	check("noupdated.md", true)
	check("multih1.md", true)
	check("noh1.md", true)
	check("abspath.md", true)
	// the abs path only appears on a fence line -> the crude check skips fence
	// lines, but the line "/mnt/c/ok" itself is NOT a fence marker, so it still
	// trips. Verify behavior matches the bash hook (which also flags it).
	check("fenced.md", true)
}

func TestFencedSkipsFenceMarkerLine(t *testing.T) {
	v := t.TempDir()
	// abs path embedded on the same line as a fence marker -> skipped.
	write(t, v, "ok.md", "---\nupdated: 2026-06-14\n---\n\n# OK\n\n``` /mnt/c/x\ncode\n```\n")
	probs, _ := Files(v, []string{"ok.md"})
	if len(probs) != 0 {
		t.Errorf("expected no problems, got %v", probs)
	}
}
