package moc

import (
	"os"
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

func TestGenerate(t *testing.T) {
	v := t.TempDir()
	// notes (sorted by byte order of relpath: infrastructure < projects < zeta)
	write(t, v, "projects/beta.md", "---\nupdated: 2026-01-01\n---\n\n# Beta Project\n\nbody")
	write(t, v, "infrastructure/nodes.md", "# Nodes\n")
	write(t, v, "zeta.md", "no h1 here\nsecond line")
	// exclusions: meta files, dotfiles, excluded dirs
	write(t, v, "MOC.md", "# Map of Content\n")
	write(t, v, "_format.md", "# Format\n")
	write(t, v, "CLAUDE.md", "# Claude\n")
	write(t, v, ".obsidian-cli-enabled", "")
	write(t, v, "scripts/x.md", "# Script\n")
	write(t, v, "services/y.md", "# Service\n")
	write(t, v, "archive/old.md", "# Old\n")
	write(t, v, ".git/config.md", "# git\n")

	out, n, err := Generate(v, "2026-06-14")
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("note count = %d, want 3", n)
	}
	got := string(out)

	wantHeader := "# Map of Content\n\n_Auto-generated 2026-06-14 by vkit moc. Do not edit by hand._\n\n"
	if !strings.HasPrefix(got, wantHeader) {
		t.Fatalf("header mismatch:\n%q", got)
	}

	wantLines := []string{
		"- [[infrastructure/nodes]] — Nodes",
		"- [[projects/beta]] — Beta Project",
		"- [[zeta]] — untitled",
	}
	body := strings.TrimPrefix(got, wantHeader)
	gotLines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("lines = %#v want %#v", gotLines, wantLines)
	}
	for i := range wantLines {
		if gotLines[i] != wantLines[i] {
			t.Errorf("line %d = %q want %q", i, gotLines[i], wantLines[i])
		}
	}
}

func TestBuildWritesFile(t *testing.T) {
	v := t.TempDir()
	write(t, v, "a.md", "# A\n")
	n, err := Build(v, "2026-06-14")
	if err != nil || n != 1 {
		t.Fatalf("Build n=%d err=%v", n, err)
	}
	b, err := os.ReadFile(filepath.Join(v, "MOC.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "- [[a]] — A") {
		t.Errorf("MOC.md missing entry:\n%s", b)
	}
}
