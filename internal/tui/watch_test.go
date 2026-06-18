package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jz-wilson/vkit/internal/watcher"
)

func TestListenCmdEmitsFileChanged(t *testing.T) {
	src := watcher.NewFakeSource()
	src.Send("/vault/a.md")
	msg := listenCmd(src)()
	fc, ok := msg.(FileChangedMsg)
	if !ok || fc.Path != "/vault/a.md" {
		t.Fatalf("listenCmd = %#v, want FileChangedMsg{/vault/a.md}", msg)
	}
}

func TestListenCmdReportsClose(t *testing.T) {
	src := watcher.NewFakeSource()
	_ = src.Close()
	if _, ok := listenCmd(src)().(watchClosedMsg); !ok {
		t.Fatalf("closed source should yield watchClosedMsg")
	}
}

// invalidVault writes a note that fails validation (no frontmatter, no H1).
func invalidVault(t *testing.T) string {
	t.Helper()
	v := t.TempDir()
	if err := os.WriteFile(filepath.Join(v, "bad.md"), []byte("just text\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestValidateVaultCmdFindsProblems(t *testing.T) {
	v := invalidVault(t)
	msg := validateVaultCmd(v)()
	done, ok := msg.(ValidationDoneMsg)
	if !ok || done.Err != nil {
		t.Fatalf("expected ValidationDoneMsg, got %#v", msg)
	}
	if len(done.Problems) == 0 {
		t.Fatalf("expected problems for an invalid note")
	}
}

func TestFileChangeReArmsAndRevalidates(t *testing.T) {
	v := invalidVault(t)
	src := watcher.NewFakeSource()
	defer src.Close()

	var model tea.Model = NewRootModel(v).WithWatcher(src)
	model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// A file-change message must produce work (re-arm + revalidate).
	model, cmd := model.Update(FileChangedMsg{Path: filepath.Join(v, "bad.md")})
	if cmd == nil {
		t.Fatalf("FileChangedMsg should return a command (re-arm + validate)")
	}

	// Drive the validation directly and feed the result back into the model.
	done := validateVaultCmd(v)().(ValidationDoneMsg)
	model, _ = model.Update(done)
	r := model.(RootModel)
	if len(r.log.Problems()) == 0 {
		t.Fatalf("log panel did not receive validation problems")
	}
	if !strings.Contains(r.View(), "bad.md") {
		t.Fatalf("view should list the failing note")
	}
}

func TestKeyVValidatesSelectedNote(t *testing.T) {
	v := invalidVault(t)
	var model tea.Model = NewRootModel(v)
	model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	if cmd == nil {
		t.Fatalf("'v' should trigger validation")
	}
	done, ok := cmd().(ValidationDoneMsg)
	if !ok || done.Rel != "bad.md" || len(done.Problems) == 0 {
		t.Fatalf("'v' did not validate the selected note: %#v", done)
	}
}
