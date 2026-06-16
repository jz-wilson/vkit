package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// writeVault lays out a small vault with notes, an excluded dir, and skip files
// so the NavPanel's use of vaultpath.WalkNotes can be exercised end to end.
func writeVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mk := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("_format.md", "marker")      // skip (meta)
	mk("MOC.md", "# Map")           // skip (generated)
	mk("alpha.md", "# Alpha\n")     // note
	mk("beta.md", "# Beta\n")       // note
	mk("sub/gamma.md", "# Gamma\n") // note
	mk("archive/old.md", "# Old\n") // excluded dir
	mk(".git/cfg.md", "# Git\n")    // excluded dotdir
	return root
}

// collectMsgs flattens a (possibly batched, possibly nil) command into the
// concrete messages it would emit.
func collectMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	switch msg := cmd().(type) {
	case nil:
		return nil
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range msg {
			out = append(out, collectMsgs(c)...)
		}
		return out
	default:
		return []tea.Msg{msg}
	}
}

func findFileSelected(msgs []tea.Msg) (FileSelectedMsg, bool) {
	for _, m := range msgs {
		if fs, ok := m.(FileSelectedMsg); ok {
			return fs, true
		}
	}
	return FileSelectedMsg{}, false
}

func TestNavLoadsNotesSortedAndExcludes(t *testing.T) {
	p := NewNavPanel(writeVault(t))
	items := p.list.Items()
	got := make([]string, len(items))
	for i, it := range items {
		got[i] = it.(noteItem).rel
	}
	want := []string{"alpha.md", "beta.md", "sub/gamma.md"}
	if len(got) != len(want) {
		t.Fatalf("items = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("items = %v, want %v", got, want)
		}
	}
}

func TestNavInitialSelectionPrimesWorkspace(t *testing.T) {
	p := NewNavPanel(writeVault(t))
	if p.Selected() != "alpha.md" {
		t.Fatalf("initial selection = %q, want alpha.md", p.Selected())
	}
	msgs := collectMsgs(p.SelectCmd())
	fs, ok := findFileSelected(msgs)
	if !ok || fs.Rel != "alpha.md" {
		t.Fatalf("SelectCmd did not emit FileSelectedMsg{alpha.md}, got %#v", msgs)
	}
}

func TestNavCursorDownEmitsFileSelected(t *testing.T) {
	p := NewNavPanel(writeVault(t)).Resize(30, 12).Focused(true)

	p, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if p.Selected() != "beta.md" {
		t.Fatalf("after 'j' selection = %q, want beta.md", p.Selected())
	}
	fs, ok := findFileSelected(collectMsgs(cmd))
	if !ok || fs.Rel != "beta.md" {
		t.Fatalf("'j' did not emit FileSelectedMsg{beta.md}")
	}

	// Moving back up emits the previous note again.
	p, cmd = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if p.Selected() != "alpha.md" {
		t.Fatalf("after 'k' selection = %q, want alpha.md", p.Selected())
	}
	if fs, ok := findFileSelected(collectMsgs(cmd)); !ok || fs.Rel != "alpha.md" {
		t.Fatalf("'k' did not emit FileSelectedMsg{alpha.md}")
	}
}

func TestNavNoMoveNoEmit(t *testing.T) {
	// At the top, pressing 'k' keeps the selection and should not emit a
	// (redundant) FileSelectedMsg.
	p := NewNavPanel(writeVault(t)).Resize(30, 12).Focused(true)
	p, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if p.Selected() != "alpha.md" {
		t.Fatalf("selection moved unexpectedly: %q", p.Selected())
	}
	if _, ok := findFileSelected(collectMsgs(cmd)); ok {
		t.Fatalf("unexpected FileSelectedMsg when selection did not change")
	}
}

func TestNavEmptyVault(t *testing.T) {
	p := NewNavPanel(filepath.Join(t.TempDir(), "does-not-exist"))
	if len(p.list.Items()) != 0 {
		t.Fatalf("expected no items for missing vault")
	}
	if p.Selected() != "" {
		t.Fatalf("expected empty selection, got %q", p.Selected())
	}
	if p.SelectCmd() != nil {
		t.Fatalf("expected nil SelectCmd for empty vault")
	}
}
