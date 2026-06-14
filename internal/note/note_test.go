package note

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestCreateWritesScaffold: Create writes schema-valid frontmatter, an H1 from
// the explicit title, and the standard body skeleton.
func TestCreateWritesScaffold(t *testing.T) {
	v := t.TempDir()
	if err := Create(v, "projects/foo.md", "My Title", []string{"a", "b"}, "2026-06-14"); err != nil {
		t.Fatal(err)
	}
	got := read(t, filepath.Join(v, "projects", "foo.md"))
	want := "---\nupdated: 2026-06-14\ntags: [a, b]\n---\n\n# My Title\n\n## Summary\n\n## Notes\n\n## Related\n"
	if got != want {
		t.Errorf("Create output:\n%q\nwant:\n%q", got, want)
	}
}

// TestCreateNoTags: with no tags, the tags line is omitted entirely.
func TestCreateNoTags(t *testing.T) {
	v := t.TempDir()
	if err := Create(v, "a.md", "Title", nil, "2026-06-14"); err != nil {
		t.Fatal(err)
	}
	got := read(t, filepath.Join(v, "a.md"))
	if strings.Contains(got, "tags:") {
		t.Errorf("expected no tags line, got:\n%q", got)
	}
	want := "---\nupdated: 2026-06-14\n---\n\n# Title\n\n## Summary\n\n## Notes\n\n## Related\n"
	if got != want {
		t.Errorf("Create output:\n%q\nwant:\n%q", got, want)
	}
}

// TestCreateTitleFromFilename: an empty title is derived from the filename,
// kebab/underscore -> Title Case, with the directory dropped.
func TestCreateTitleFromFilename(t *testing.T) {
	v := t.TempDir()
	if err := Create(v, "projects/my-cool-note.md", "", nil, "2026-06-14"); err != nil {
		t.Fatal(err)
	}
	got := read(t, filepath.Join(v, "projects", "my-cool-note.md"))
	if !strings.Contains(got, "# My Cool Note\n") {
		t.Errorf("derived title missing:\n%q", got)
	}
}

// TestCreateRefusesOverwrite: Create never clobbers an existing file and reports
// the relpath in the error.
func TestCreateRefusesOverwrite(t *testing.T) {
	v := t.TempDir()
	rel := "projects/exists.md"
	full := filepath.Join(v, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("ORIGINAL"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Create(v, rel, "Title", nil, "2026-06-14")
	if err == nil {
		t.Fatal("expected overwrite refusal, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") || !strings.Contains(err.Error(), rel) {
		t.Errorf("error = %q, want mention of %q and 'already exists'", err, rel)
	}
	// original content untouched
	if read(t, full) != "ORIGINAL" {
		t.Error("Create overwrote the existing file")
	}
}

// TestCreateMakesParentDirs: Create creates missing parent directories.
func TestCreateMakesParentDirs(t *testing.T) {
	v := t.TempDir()
	if err := Create(v, "deep/nested/path/note.md", "Title", nil, "2026-06-14"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(v, "deep", "nested", "path", "note.md")); err != nil {
		t.Errorf("nested note not created: %v", err)
	}
}

// TestTitleFromFilename exercises the kebab/underscore -> Title Case derivation
// directly across the relevant cases.
func TestTitleFromFilename(t *testing.T) {
	cases := map[string]string{
		"projects/my-cool-note.md": "My Cool Note",
		"foo.md":                   "Foo",
		"snake_case_note.md":       "Snake Case Note",
		"mixed-snake_case.md":      "Mixed Snake Case",
		"a/b/deep-note.md":         "Deep Note",
		"already Title.md":         "Already Title",
		"-leading-dash.md":         " Leading Dash", // empty first segment -> leading space
		"UPPER-word.md":            "UPPER Word",    // first rune upcased, rest preserved
	}
	for rel, want := range cases {
		if got := titleFromFilename(rel); got != want {
			t.Errorf("titleFromFilename(%q)=%q want %q", rel, got, want)
		}
	}
}

// TestRender exercises the frontmatter/body builder directly.
func TestRender(t *testing.T) {
	got := render("Title", []string{"x", "y"}, "2026-06-14")
	want := "---\nupdated: 2026-06-14\ntags: [x, y]\n---\n\n# Title\n\n## Summary\n\n## Notes\n\n## Related\n"
	if got != want {
		t.Errorf("render:\n%q\nwant:\n%q", got, want)
	}

	gotNoTags := render("Title", nil, "2026-06-14")
	wantNoTags := "---\nupdated: 2026-06-14\n---\n\n# Title\n\n## Summary\n\n## Notes\n\n## Related\n"
	if gotNoTags != wantNoTags {
		t.Errorf("render no-tags:\n%q\nwant:\n%q", gotNoTags, wantNoTags)
	}
}

// TestCreateNativeTitleRouting characterizes the only part of the Tier A native
// path that is unit-testable without an installed `obsidian` binary: the
// empty-title -> titleFromFilename derivation that precedes the exec calls.
//
// LIMITATION: CreateNative invokes the `obsidian` CLI through the unexported
// runObsidian, which calls exec.Command("obsidian", ...) directly. There is no
// injectable exec/lookPath hook (unlike osdetect's detectPkgMgr), so the actual
// exec routing, argument construction, and property:set sequencing cannot be
// tested here without either an installed obsidian binary or a refactor to make
// the exec injectable. Only the shared title-derivation decision is covered.
func TestCreateNativeTitleRouting(t *testing.T) {
	// Same derivation rule the native path applies before shelling out.
	if got := titleFromFilename("projects/native-note.md"); got != "Native Note" {
		t.Errorf("native title derivation = %q want %q", got, "Native Note")
	}
}
