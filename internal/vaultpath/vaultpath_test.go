package vaultpath

import (
	"os"
	"path/filepath"
	"sort"
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

// TestResolveArgWins: an explicit arg beats every other source. The result is
// absolute and existence is not required.
func TestResolveArgWins(t *testing.T) {
	t.Setenv("VKIT_VAULT", "/env/vault")
	got, err := Resolve("arg/vault", "flag/vault")
	if err != nil {
		t.Fatal(err)
	}
	wantAbs, _ := filepath.Abs("arg/vault")
	if got != wantAbs {
		t.Errorf("Resolve arg = %q want %q", got, wantAbs)
	}
}

// TestResolveFlagBeatsEnv: with no arg, the --vault flag beats $VKIT_VAULT.
func TestResolveFlagBeatsEnv(t *testing.T) {
	t.Setenv("VKIT_VAULT", "/env/vault")
	got, err := Resolve("", "flag/vault")
	if err != nil {
		t.Fatal(err)
	}
	wantAbs, _ := filepath.Abs("flag/vault")
	if got != wantAbs {
		t.Errorf("Resolve flag = %q want %q", got, wantAbs)
	}
}

// TestResolveEnvBeatsWalkAndHome: with no arg or flag, $VKIT_VAULT is used even
// when the cwd is inside a marked vault.
func TestResolveEnvBeatsWalkUp(t *testing.T) {
	v := t.TempDir()
	write(t, v, Marker, "# Format\n")
	chdir(t, v)
	t.Setenv("VKIT_VAULT", filepath.Join(v, "envsub"))

	got, err := Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(filepath.Join(v, "envsub"))
	if got != want {
		t.Errorf("Resolve env = %q want %q", got, want)
	}
}

// TestResolveFallsBackToCWD: with no arg/flag/env, Resolve returns the current
// working directory.
func TestResolveFallsBackToCWD(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	t.Setenv("VKIT_VAULT", "")

	got, err := Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	wantReal, _ := filepath.EvalSymlinks(dir)
	gotReal, _ := filepath.EvalSymlinks(got)
	if gotReal != wantReal {
		t.Errorf("Resolve CWD fallback = %q want %q", gotReal, wantReal)
	}
}

func TestIsVault(t *testing.T) {
	v := t.TempDir()
	if IsVault(v) {
		t.Error("empty dir reported as vault")
	}
	write(t, v, Marker, "# Format\n")
	if !IsVault(v) {
		t.Error("dir with marker not reported as vault")
	}
}

func TestIsExcludedDir(t *testing.T) {
	cases := map[string]bool{
		".git":           true,
		".claude":        true,
		"scripts":        true,
		"services":       true,
		"archive":        true,
		".obsidian":      true, // any dotdir
		".hidden":        true,
		"projects":       false,
		"infrastructure": false,
		"notes":          false,
	}
	for base, want := range cases {
		if got := IsExcludedDir(base); got != want {
			t.Errorf("IsExcludedDir(%q)=%v want %v", base, got, want)
		}
	}
}

func TestIsNote(t *testing.T) {
	cases := map[string]bool{
		"projects/a.md":        true,
		"decisions/2026-06.md": true,
		"zeta.md":              true,
		"notes/file.txt":       false, // not markdown
		".obsidian/x.md":       false, // dot segment
		"projects/.hidden.md":  false, // dotfile basename
		"scripts/x.md":         false, // excluded dir
		"services/y.md":        false, // excluded dir
		"archive/old.md":       false, // excluded dir
		".git/config.md":       false, // excluded dir + dot
		"MOC.md":               false, // skip name
		"CLAUDE.md":            false, // skip name
		"AGENTS.md":            false, // skip name
		"_format.md":           false, // skip name
		"a/MOC.md":             false, // skip name in subdir
	}
	for rel, want := range cases {
		if got := IsNote(rel); got != want {
			t.Errorf("IsNote(%q)=%v want %v", rel, got, want)
		}
	}
}

func TestClassifyFile(t *testing.T) {
	t.Run("normal note passes", func(t *testing.T) {
		if !ClassifyFile("projects/a.md", ClassifyOpts{}) {
			t.Error("expected true for a regular note")
		}
	})
	t.Run("README.md blocked when SkipREADME=true", func(t *testing.T) {
		if ClassifyFile("README.md", ClassifyOpts{SkipREADME: true}) {
			t.Error("expected false for README.md with SkipREADME=true")
		}
		if ClassifyFile("subdir/README.md", ClassifyOpts{SkipREADME: true}) {
			t.Error("expected false for subdir/README.md with SkipREADME=true")
		}
	})
	t.Run("README.md passes when SkipREADME=false", func(t *testing.T) {
		if !ClassifyFile("README.md", ClassifyOpts{SkipREADME: false}) {
			t.Error("expected true for README.md with SkipREADME=false")
		}
	})
	t.Run("non-md file blocked", func(t *testing.T) {
		if ClassifyFile("notes/file.txt", ClassifyOpts{}) {
			t.Error("expected false for non-markdown file")
		}
	})
}

// TestWalkNotes: WalkNotes visits exactly the note-eligible files, skipping
// excluded dirs, dotdirs, dotfiles, non-markdown, and meta files.
func TestWalkNotes(t *testing.T) {
	v := t.TempDir()
	write(t, v, "projects/beta.md", "body")
	write(t, v, "infrastructure/nodes.md", "body")
	write(t, v, "zeta.md", "body")
	// excluded:
	write(t, v, "MOC.md", "x")
	write(t, v, "_format.md", "x")
	write(t, v, "CLAUDE.md", "x")
	write(t, v, "AGENTS.md", "x")
	write(t, v, "notes/file.txt", "x")
	write(t, v, ".obsidian/conf.md", "x")
	write(t, v, "scripts/x.md", "x")
	write(t, v, "services/y.md", "x")
	write(t, v, "archive/old.md", "x")
	write(t, v, ".git/config.md", "x")

	var got []string
	if err := WalkNotes(v, func(rel string) error {
		got = append(got, rel)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	want := []string{"infrastructure/nodes.md", "projects/beta.md", "zeta.md"}
	if len(got) != len(want) {
		t.Fatalf("WalkNotes visited %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("note[%d]=%q want %q", i, got[i], want[i])
		}
	}
}

// chdir changes into dir for the duration of the test, restoring the prior cwd
// on cleanup. t.Chdir is 1.24+, so do it manually for the go 1.22 module.
func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}
