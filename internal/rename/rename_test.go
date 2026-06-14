package rename

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func gitInit(t *testing.T, vault string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q"},
		{"add", "-A"},
	} {
		cmd := exec.Command("git", append([]string{"-C", vault}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func TestRenameRewritesLinks(t *testing.T) {
	v := t.TempDir()
	write(t, v, "old.md", "---\nupdated: 2026-06-14\n---\n\n# Old\n")
	write(t, v, "projects/linker.md", "see [[old]] and [[old|alias]] and [[old#section]].\n")
	write(t, v, "projects/nolink.md", "nothing relevant here, [[older]] stays.\n")
	gitInit(t, v)

	touched, err := Rename(v, "old.md", "renamed.md")
	if err != nil {
		t.Fatal(err)
	}

	// moved on disk
	if _, err := os.Stat(filepath.Join(v, "renamed.md")); err != nil {
		t.Error("renamed.md not present")
	}
	if _, err := os.Stat(filepath.Join(v, "old.md")); err == nil {
		t.Error("old.md still present")
	}

	// links rewritten, alias/heading preserved
	linker, _ := os.ReadFile(filepath.Join(v, "projects", "linker.md"))
	want := "see [[renamed]] and [[renamed|alias]] and [[renamed#section]].\n"
	if string(linker) != want {
		t.Errorf("linker = %q want %q", linker, want)
	}

	// [[older]] must NOT be rewritten (boundary respected)
	nolink, _ := os.ReadFile(filepath.Join(v, "projects", "nolink.md"))
	if !strings.Contains(string(nolink), "[[older]]") {
		t.Errorf("boundary failure, [[older]] changed: %s", nolink)
	}

	// touched contains the new path and the linker
	joined := strings.Join(touched, ",")
	if !strings.Contains(joined, "renamed.md") || !strings.Contains(joined, "projects/linker.md") {
		t.Errorf("touched = %v", touched)
	}

	// git knows about the move (renamed.md is staged/tracked)
	out, _ := exec.Command("git", "-C", v, "ls-files", "renamed.md").Output()
	if strings.TrimSpace(string(out)) != "renamed.md" {
		t.Errorf("git mv not applied, ls-files: %q", out)
	}
}
