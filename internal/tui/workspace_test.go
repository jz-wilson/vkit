package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// runCmd executes a command and returns its message (nil-safe, non-batched).
func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestWorkspaceLoadsNoteOnFileSelected(t *testing.T) {
	vault := t.TempDir()
	body := "# Alpha\n\nUNIQUE_BODY_TOKEN line one\nmore text\n"
	if err := os.WriteFile(filepath.Join(vault, "alpha.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	p := NewWorkspacePanel(vault).Resize(60, 20)

	// Selecting a note returns a load command.
	p, cmd := p.Update(FileSelectedMsg{Rel: "alpha.md"})
	msg := runCmd(cmd)
	loaded, ok := msg.(noteLoadedMsg)
	if !ok {
		t.Fatalf("expected noteLoadedMsg, got %T", msg)
	}
	if loaded.err != nil {
		t.Fatalf("unexpected load error: %v", loaded.err)
	}

	// Feeding the result back renders the body and shows the path in the title.
	p, _ = p.Update(loaded)
	view := p.View()
	if !strings.Contains(view, "UNIQUE_BODY_TOKEN") {
		t.Fatalf("view missing note body; got:\n%s", view)
	}
	if !strings.Contains(view, "alpha.md") {
		t.Fatalf("view missing note path in title")
	}
}

func TestWorkspaceReportsLoadError(t *testing.T) {
	p := NewWorkspacePanel(t.TempDir()).Resize(60, 20)
	p, cmd := p.Update(FileSelectedMsg{Rel: "missing.md"})
	loaded, ok := runCmd(cmd).(noteLoadedMsg)
	if !ok || loaded.err == nil {
		t.Fatalf("expected a load error for a missing note")
	}
	p, _ = p.Update(loaded)
	if !strings.Contains(p.View(), "error loading") {
		t.Fatalf("view should surface the load error")
	}
}

func TestWorkspaceIgnoresReselectingSameNote(t *testing.T) {
	vault := t.TempDir()
	if err := os.WriteFile(filepath.Join(vault, "a.md"), []byte("# A\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := NewWorkspacePanel(vault).Resize(60, 20)
	p, cmd := p.Update(FileSelectedMsg{Rel: "a.md"})
	p, _ = p.Update(runCmd(cmd).(noteLoadedMsg))

	// Re-selecting the same note must not trigger another load.
	_, cmd2 := p.Update(FileSelectedMsg{Rel: "a.md"})
	if cmd2 != nil {
		t.Fatalf("re-selecting the current note should not reload")
	}
}

func TestWorkspaceScrollsOnlyWhenFocused(t *testing.T) {
	vault := t.TempDir()
	var sb strings.Builder
	sb.WriteString("# Long\n")
	for i := 0; i < 200; i++ {
		sb.WriteString("line\n")
	}
	if err := os.WriteFile(filepath.Join(vault, "long.md"), []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	p := NewWorkspacePanel(vault).Resize(40, 12)
	p, cmd := p.Update(FileSelectedMsg{Rel: "long.md"})
	p, _ = p.Update(runCmd(cmd).(noteLoadedMsg))

	// Unfocused: a down key should not scroll.
	blurred := p.Focused(false)
	blurred, _ = blurred.Update(tea.KeyMsg{Type: tea.KeyDown})
	if blurred.vp.YOffset != 0 {
		t.Fatalf("unfocused panel scrolled (YOffset=%d)", blurred.vp.YOffset)
	}

	// Focused: a down key should scroll.
	focused := p.Focused(true)
	focused, _ = focused.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if focused.vp.YOffset == 0 {
		t.Fatalf("focused panel did not scroll on page-down")
	}
}
