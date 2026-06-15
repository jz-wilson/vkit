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

	touched, err := Rename(v, "old.md", "renamed.md", false)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(v, "renamed.md")); err != nil {
		t.Error("renamed.md not present")
	}
	if _, err := os.Stat(filepath.Join(v, "old.md")); err == nil {
		t.Error("old.md still present")
	}

	linker, _ := os.ReadFile(filepath.Join(v, "projects", "linker.md"))
	want := "see [[renamed]] and [[renamed|alias]] and [[renamed#section]].\n"
	if string(linker) != want {
		t.Errorf("linker = %q want %q", linker, want)
	}

	nolink, _ := os.ReadFile(filepath.Join(v, "projects", "nolink.md"))
	if !strings.Contains(string(nolink), "[[older]]") {
		t.Errorf("boundary failure, [[older]] changed: %s", nolink)
	}

	joined := strings.Join(touched, ",")
	if !strings.Contains(joined, "renamed.md") || !strings.Contains(joined, "projects/linker.md") {
		t.Errorf("touched = %v", touched)
	}

	out, _ := exec.Command("git", "-C", v, "ls-files", "renamed.md").Output()
	if strings.TrimSpace(string(out)) != "renamed.md" {
		t.Errorf("git mv not applied, ls-files: %q", out)
	}
}

func TestRenameDryRun(t *testing.T) {
	v := t.TempDir()
	write(t, v, "old.md", "---\nupdated: 2026-06-14\n---\n\n# Old\n")
	write(t, v, "projects/linker.md", "see [[old]] here.\n")
	gitInit(t, v)

	touched, err := Rename(v, "old.md", "renamed.md", true)
	if err != nil {
		t.Fatal(err)
	}

	// source still exists, destination does not
	if _, err := os.Stat(filepath.Join(v, "old.md")); err != nil {
		t.Error("old.md should still exist in dry-run mode")
	}
	if _, err := os.Stat(filepath.Join(v, "renamed.md")); err == nil {
		t.Error("renamed.md must not exist in dry-run mode")
	}

	// linker content unchanged
	linker, _ := os.ReadFile(filepath.Join(v, "projects", "linker.md"))
	if string(linker) != "see [[old]] here.\n" {
		t.Errorf("linker was modified in dry-run: %q", linker)
	}

	// touch list still reports what would have changed
	joined := strings.Join(touched, ",")
	if !strings.Contains(joined, "renamed.md") || !strings.Contains(joined, "projects/linker.md") {
		t.Errorf("dry-run touched = %v; expected renamed.md and projects/linker.md", touched)
	}
}

func TestLinkRewriterRewrite(t *testing.T) {
	rw := LinkRewriter{}

	tests := []struct {
		name    string
		content string
		oldStem string
		newStem string
		want    string
	}{
		{
			name:    "bare link rewritten",
			content: "see [[old-note]] here",
			oldStem: "old-note",
			newStem: "new-note",
			want:    "see [[new-note]] here",
		},
		{
			name:    "link with alias rewritten",
			content: "see [[old-note|My Alias]] here",
			oldStem: "old-note",
			newStem: "new-note",
			want:    "see [[new-note|My Alias]] here",
		},
		{
			name:    "link with section rewritten",
			content: "see [[old-note#Introduction]] here",
			oldStem: "old-note",
			newStem: "new-note",
			want:    "see [[new-note#Introduction]] here",
		},
		{
			name:    "unrelated link unchanged",
			content: "see [[unrelated]] here",
			oldStem: "old-note",
			newStem: "new-note",
			want:    "see [[unrelated]] here",
		},
		{
			name:    "prefix boundary — [[older]] not rewritten when stem is old",
			content: "[[older]] stays",
			oldStem: "old",
			newStem: "new",
			want:    "[[older]] stays",
		},
		{
			name:    "multiple occurrences all rewritten",
			content: "[[old-note]] and again [[old-note|alias]] and [[old-note#sec]]",
			oldStem: "old-note",
			newStem: "new-note",
			want:    "[[new-note]] and again [[new-note|alias]] and [[new-note#sec]]",
		},
		{
			name:    "path stem — basename form rewritten",
			content: "[[old-note]] and [[folder/old-note]]",
			oldStem: "folder/old-note",
			newStem: "folder/new-note",
			want:    "[[new-note]] and [[folder/new-note]]",
		},
		{
			name:    "content with no links unchanged",
			content: "no wiki links at all",
			oldStem: "old-note",
			newStem: "new-note",
			want:    "no wiki links at all",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := rw.Rewrite(tc.content, tc.oldStem, tc.newStem)
			if got != tc.want {
				t.Errorf("Rewrite(%q, %q, %q)\n  got  %q\n  want %q",
					tc.content, tc.oldStem, tc.newStem, got, tc.want)
			}
		})
	}
}
