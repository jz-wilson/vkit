package tui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jz-wilson/vkit/internal/watcher"
)

// TestProgramRunsAndQuitsHeadless drives the real Bubble Tea program with a
// fake watcher and a scripted 'q' keypress, exercising Init (initial validation
// + watcher listen goroutine), the update loop, and clean shutdown — all
// without a TTY.
func TestProgramRunsAndQuitsHeadless(t *testing.T) {
	vault := writeVault(t) // helper from nav_test.go
	src := watcher.NewFakeSource()
	defer src.Close()

	model := NewRootModel(vault).WithWatcher(src)
	p := tea.NewProgram(
		model,
		tea.WithInput(bytes.NewReader([]byte("q"))),
		tea.WithoutRenderer(),
	)

	done := make(chan error, 1)
	go func() {
		_, err := p.Run()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("program returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		p.Kill()
		t.Fatal("program did not quit on 'q' within 5s")
	}
}
