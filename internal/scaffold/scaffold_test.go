package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
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

func exists2(p string) bool { _, err := os.Stat(p); return err == nil }

// dirtyVault installs a fresh vault then mutates it so an update has work to do:
// a changed tier2 (CLAUDE.md), a changed tier1 (.gitignore), and a missing
// tier2 (note.md).
func dirtyVault(t *testing.T) string {
	t.Helper()
	v := t.TempDir()
	if err := Install(v); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v, "CLAUDE.md"), []byte("# my custom claude\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v, ".gitignore"), []byte("# custom ignore\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(v, ".claude", "commands", "note.md")); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestAnalyze(t *testing.T) {
	v := dirtyVault(t)
	e := Analyze(v)
	if got := strings.Join(e.T1Change, ","); got != ".gitignore" {
		t.Errorf("T1Change=%v", e.T1Change)
	}
	if got := strings.Join(e.T2New, ","); got != ".claude/commands/note.md" {
		t.Errorf("T2New=%v", e.T2New)
	}
	if got := strings.Join(e.T2Change, ","); got != "CLAUDE.md" {
		t.Errorf("T2Change=%v", e.T2Change)
	}
}

func TestAlreadyMatches(t *testing.T) {
	v := t.TempDir()
	if err := Install(v); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	res, err := Update(v, ModeKeep, false, strings.NewReader(""), &out, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.AlreadyMatches {
		t.Errorf("expected AlreadyMatches, got %+v / %q", res, out.String())
	}
}

func TestDryRun(t *testing.T) {
	v := dirtyVault(t)
	var out bytes.Buffer
	res, err := Update(v, ModeForce, true, strings.NewReader(""), &out, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun {
		t.Fatalf("expected DryRun, got %+v", res)
	}
	// nothing written: CLAUDE.md still custom, note.md still missing, no .bak.
	if read(t, filepath.Join(v, "CLAUDE.md")) != "# my custom claude\n" {
		t.Error("CLAUDE.md was modified during dry-run")
	}
	if exists2(filepath.Join(v, ".claude", "commands", "note.md")) {
		t.Error("note.md was created during dry-run")
	}
	if exists2(filepath.Join(v, "CLAUDE.md.bak")) {
		t.Error("dry-run created a .bak")
	}
}

func TestKeep(t *testing.T) {
	v := dirtyVault(t)
	var out bytes.Buffer
	res, err := Update(v, ModeKeep, false, strings.NewReader(""), &out, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != "safe" || res.Tool != 1 || res.New != 1 || res.Keep != 1 || res.Over != 0 {
		t.Fatalf("res=%+v", res)
	}
	// tooling refreshed:
	if !strings.Contains(read(t, filepath.Join(v, ".gitignore")), ".moc-stamp") {
		t.Error(".gitignore not refreshed")
	}
	// new template added:
	if !exists2(filepath.Join(v, ".claude", "commands", "note.md")) {
		t.Error("note.md not added")
	}
	// changed template kept, no .bak:
	if read(t, filepath.Join(v, "CLAUDE.md")) != "# my custom claude\n" {
		t.Error("CLAUDE.md was overwritten in keep mode")
	}
	if exists2(filepath.Join(v, "CLAUDE.md.bak")) {
		t.Error("keep mode created a .bak")
	}
}

func TestForce(t *testing.T) {
	v := dirtyVault(t)
	var out bytes.Buffer
	res, err := Update(v, ModeForce, false, strings.NewReader(""), &out, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != "apply" || res.Tool != 1 || res.New != 1 || res.Over != 1 {
		t.Fatalf("res=%+v", res)
	}
	// .bak holds the old custom content:
	if read(t, filepath.Join(v, "CLAUDE.md.bak")) != "# my custom claude\n" {
		t.Error("CLAUDE.md.bak missing or wrong")
	}
	// CLAUDE.md now matches the template:
	want, _ := templateBytes("CLAUDE.md")
	if read(t, filepath.Join(v, "CLAUDE.md")) != string(want) {
		t.Error("CLAUDE.md not overwritten with template")
	}
	// after force, vault matches the kit:
	if !Analyze(v).Empty() {
		t.Error("vault still differs after force")
	}
}

func TestInstallTree(t *testing.T) {
	v := t.TempDir()
	if err := Install(v); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"_format.md", "CLAUDE.md", "MOC.md", ".gitignore",
		".githooks/pre-commit", ".claude/settings.json", ".claude/commands/note.md",
		"services/README.md"} {
		if !exists2(filepath.Join(v, filepath.FromSlash(f))) {
			t.Errorf("missing %s", f)
		}
	}
	for _, d := range contentDirs {
		if !exists2(filepath.Join(v, d)) {
			t.Errorf("missing content dir %s", d)
		}
	}
	// hook is executable (Windows has no Unix exec bit — skip the mode check there)
	fi, err := os.Stat(filepath.Join(v, ".githooks", "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && fi.Mode()&0o100 == 0 {
		t.Error("pre-commit not executable")
	}
}
