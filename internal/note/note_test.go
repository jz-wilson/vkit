package note

import (
	"errors"
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

// TestCreateAppendsExtension: passing a bare stem ("projects/alpha") creates
// "projects/alpha.md" — the .md suffix is appended automatically.
func TestCreateAppendsExtension(t *testing.T) {
	v := t.TempDir()
	if err := Create(v, "projects/alpha", "Alpha", nil, "2026-06-14"); err != nil {
		t.Fatal(err)
	}
	mdPath := filepath.Join(v, "projects", "alpha.md")
	if _, err := os.Stat(mdPath); err != nil {
		t.Fatalf("expected %s to exist, got: %v", mdPath, err)
	}
	// bare stem must NOT exist alongside the .md
	if _, err := os.Stat(filepath.Join(v, "projects", "alpha")); err == nil {
		t.Error("extensionless file created alongside .md — should only create .md")
	}
	got := read(t, mdPath)
	if !strings.Contains(got, "# Alpha\n") {
		t.Errorf("title not in scaffold:\n%q", got)
	}
}

// TestCreateAppendsExtensionOverwriteGuard: stem + .md forms resolve to the same
// file; a second call with the stem is refused correctly.
func TestCreateAppendsExtensionOverwriteGuard(t *testing.T) {
	v := t.TempDir()
	if err := Create(v, "notes/foo.md", "Foo", nil, "2026-06-14"); err != nil {
		t.Fatal(err)
	}
	// passing the bare stem should hit the overwrite guard on the .md file
	err := Create(v, "notes/foo", "Foo Again", nil, "2026-06-14")
	if err == nil {
		t.Fatal("expected overwrite refusal for stem when .md exists, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, want 'already exists'", err)
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

// TestCreateNativeCallSequence verifies that createNative issues exec calls in
// order: create → property:set updated → property:set tags (when tags present).
func TestCreateNativeCallSequence(t *testing.T) {
	type call struct {
		vault string
		args  []string
	}
	var calls []call
	fake := func(vault string, args ...string) error {
		calls = append(calls, call{vault, args})
		return nil
	}
	if err := createNative("/vault", "projects/note.md", "My Note", []string{"go", "tdd"}, "2026-06-15", fake); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 {
		t.Fatalf("expected 3 exec calls, got %d", len(calls))
	}
	if calls[0].args[0] != "create" {
		t.Errorf("first call = %q, want create", calls[0].args[0])
	}
	if calls[1].args[0] != "property:set" || calls[1].args[1] != "name=updated" {
		t.Errorf("second call = %v, want property:set updated", calls[1].args)
	}
	if calls[2].args[0] != "property:set" || calls[2].args[1] != "name=tags" {
		t.Errorf("third call = %v, want property:set tags", calls[2].args)
	}
}

// TestCreateNativeNoTagsSkipsTagCall: nil tags → only 2 exec calls.
func TestCreateNativeNoTagsSkipsTagCall(t *testing.T) {
	var count int
	fake := func(_ string, _ ...string) error { count++; return nil }
	if err := createNative("/vault", "note.md", "Title", nil, "2026-06-15", fake); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2 exec calls, got %d", count)
	}
}

// TestCreateNativeEnsuresMD: bare stem → exec args contain path with .md.
func TestCreateNativeEnsuresMD(t *testing.T) {
	var firstArgs []string
	fake := func(_ string, args ...string) error {
		if firstArgs == nil {
			firstArgs = args
		}
		return nil
	}
	if err := createNative("/vault", "projects/alpha", "Alpha", nil, "2026-06-15", fake); err != nil {
		t.Fatal(err)
	}
	for _, a := range firstArgs {
		if strings.HasPrefix(a, "path=") && !strings.HasSuffix(a, ".md") {
			t.Errorf("exec arg %q lacks .md extension", a)
		}
	}
}

// TestCreateNativeExecFailure: first exec error → returned immediately, no
// subsequent calls.
func TestCreateNativeExecFailure(t *testing.T) {
	var count int
	fake := func(_ string, _ ...string) error {
		count++
		return errors.New("obsidian not found")
	}
	if err := createNative("/vault", "note.md", "Title", nil, "2026-06-15", fake); err == nil {
		t.Fatal("expected error, got nil")
	}
	if count != 1 {
		t.Errorf("expected 1 exec call before error, got %d", count)
	}
}

// TestPortableCreatorRespectsVaultDefaultFolder: when .obsidian/app.json
// sets newFileLocation=folder, portableCreator.Create routes bare-stem notes
// into the configured folder automatically.
func TestPortableCreatorRespectsVaultDefaultFolder(t *testing.T) {
	t.Setenv("VAULT_OBSIDIAN_CLI", "0")
	v := t.TempDir()

	// write .obsidian/app.json pointing new notes to "inbox"
	obsDir := filepath.Join(v, ".obsidian")
	if err := os.MkdirAll(obsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(obsDir, "app.json"),
		[]byte(`{"newFileLocation":"folder","newFileFolderPath":"inbox"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	creator := New(v)
	if _, err := creator.Create(v, "bare-note.md", "Bare", nil, "2026-06-15"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	// note must land in inbox/, not vault root
	if _, err := os.Stat(filepath.Join(v, "inbox", "bare-note.md")); err != nil {
		t.Error("note not routed to inbox/ as configured in app.json")
	}
	if _, err := os.Stat(filepath.Join(v, "bare-note.md")); err == nil {
		t.Error("note created at vault root — should be in inbox/")
	}
}

// TestPortableCreatorExplicitPathNotRerouted: an explicit path with a folder
// component bypasses vault default routing regardless of app.json.
func TestPortableCreatorExplicitPathNotRerouted(t *testing.T) {
	t.Setenv("VAULT_OBSIDIAN_CLI", "0")
	v := t.TempDir()

	obsDir := filepath.Join(v, ".obsidian")
	if err := os.MkdirAll(obsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(obsDir, "app.json"),
		[]byte(`{"newFileLocation":"folder","newFileFolderPath":"inbox"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	creator := New(v)
	if _, err := creator.Create(v, "projects/explicit.md", "Explicit", nil, "2026-06-15"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(v, "projects", "explicit.md")); err != nil {
		t.Error("explicit path not respected")
	}
	// must not appear under inbox/
	if _, err := os.Stat(filepath.Join(v, "inbox", "explicit.md")); err == nil {
		t.Error("explicit path was rerouted to inbox — should not happen")
	}
}

func TestNew_returnsPortableWhenDisabled(t *testing.T) {
	t.Setenv("VAULT_OBSIDIAN_CLI", "0")
	v := t.TempDir()
	creator := New(v)
	// Verify we get a portableCreator by exercising its Create path — it must
	// write the file to disk (nativeCreator would invoke the obsidian binary).
	if _, err := creator.Create(v, "test-new.md", "Test New", nil, "2026-06-15"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	mdPath := filepath.Join(v, "test-new.md")
	if _, err := os.Stat(mdPath); err != nil {
		t.Fatalf("expected file on disk at %s, got: %v", mdPath, err)
	}
}
