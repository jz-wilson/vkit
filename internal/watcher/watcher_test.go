package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, dir, rel, content string) string {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// chtime sets both atime and mtime of p to t.
func chtime(t *testing.T, p string, mt time.Time) {
	t.Helper()
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}
}

// TestIgnored characterizes the IGNORE regex parity with watch.sh. The dotpath,
// scripts/, and services/ sub-patterns each require a *leading slash* ("/\.",
// "/scripts/", "/services/"), so they only match within a rooted path — exactly
// how the watcher invokes ignored() with absolute WalkDir/fsnotify paths. The
// MOC.md, *.tmp, *.swp, and ~ patterns are unanchored / end-anchored and match
// even in a bare relative path. These cases capture that current behavior; a
// bare relative "scripts/x.md" is NOT ignored because it lacks a leading slash.
func TestIgnored(t *testing.T) {
	cases := map[string]bool{
		// absolute-style paths (the real call shape):
		"/vault/projects/a.md":        false,
		"/vault/infrastructure/x.md":  false,
		"/vault/decisions/2026-06.md": false,
		"/vault/a.md":                 false,
		"/vault/MOC.md":               true,
		"/vault/.obsidian/app.json":   true,
		"/vault/.git/config":          true,
		"/vault/.hidden.md":           true,
		"/vault/scripts/build.sh":     true,
		"/vault/scripts/x.md":         true,
		"/vault/services/s.md":        true,
		"/vault/services/y.md":        true,
		"/vault/projects/.draft.md":   true,
		// end-anchored editor temp/swap patterns match regardless of slash:
		"note.tmp": true,
		"note.swp": true,
		"note.md~": true,
		"MOC.md":   true, // MOC\.md is unanchored
		// bare relative dot/scripts/services lack a leading slash -> NOT ignored:
		"scripts/build.sh":   false,
		"services/s.md":      false,
		".hidden.md":         false,
		".obsidian/app.json": false,
		// .bak is not in the IGNORE set:
		"/vault/projects/note.md.bak": false,
	}
	for path, want := range cases {
		if got := ignored(path); got != want {
			t.Errorf("ignored(%q)=%v want %v", path, got, want)
		}
	}
}

// TestNewestNote characterizes that newestNote returns the most recent mtime
// among non-ignored .md files, skipping ignored dirs/files and non-.md files.
func TestNewestNote(t *testing.T) {
	v := t.TempDir()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// non-ignored notes with ascending mtimes; the newest must win.
	a := write(t, v, "projects/a.md", "a")
	b := write(t, v, "infrastructure/b.md", "b")
	newest := write(t, v, "decisions/2026-06.md", "c")
	chtime(t, a, base)
	chtime(t, b, base.Add(1*time.Hour))
	chtime(t, newest, base.Add(2*time.Hour))

	// ignored content that is *newer* must NOT win.
	moc := write(t, v, "MOC.md", "moc")
	chtime(t, moc, base.Add(10*time.Hour))
	script := write(t, v, "scripts/x.md", "x")
	chtime(t, script, base.Add(11*time.Hour))
	svc := write(t, v, "services/y.md", "y")
	chtime(t, svc, base.Add(12*time.Hour))
	dot := write(t, v, ".obsidian/cache.md", "z")
	chtime(t, dot, base.Add(13*time.Hour))
	txt := write(t, v, "notes.txt", "t")
	chtime(t, txt, base.Add(14*time.Hour))
	tmp := write(t, v, "draft.tmp", "tmp")
	chtime(t, tmp, base.Add(15*time.Hour))

	got := newestNote(v)
	want := base.Add(2 * time.Hour)
	if !got.Equal(want) {
		t.Errorf("newestNote=%v want %v (ignored files must not win)", got, want)
	}
}

// TestNewestNoteEmpty characterizes that an empty vault (no .md notes) yields the
// zero time.
func TestNewestNoteEmpty(t *testing.T) {
	v := t.TempDir()
	// only ignored / non-md content present.
	write(t, v, "MOC.md", "moc")
	write(t, v, "notes.txt", "t")
	if got := newestNote(v); !got.IsZero() {
		t.Errorf("newestNote=%v want zero time", got)
	}
}

// TestNewestNoteSkipsIgnoredDirTree characterizes that an entire ignored dir is
// pruned (SkipDir): a deeply-nested newer note under scripts/ does not win.
func TestNewestNoteSkipsIgnoredDirTree(t *testing.T) {
	v := t.TempDir()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	keep := write(t, v, "projects/a.md", "a")
	chtime(t, keep, base)

	nested := write(t, v, "scripts/sub/deep/x.md", "x")
	chtime(t, nested, base.Add(5*time.Hour))

	got := newestNote(v)
	if !got.Equal(base) {
		t.Errorf("newestNote=%v want %v (nested ignored-dir note must not win)", got, base)
	}
}
