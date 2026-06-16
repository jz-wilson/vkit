package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jz-wilson/vkit/internal/watcher"
)

// Run launches the interactive TUI for the given vault in the alternate screen.
// Live validation is on by default: it attaches an fsnotify watcher, falling
// back to mtime polling if fsnotify is unavailable. The watcher is closed when
// the program exits.
func Run(vault string) error {
	model := NewRootModel(vault)

	src, err := watcher.NewFsnotifySource(vault)
	var es watcher.EventSource = src
	if err != nil {
		fmt.Fprintf(os.Stderr, "vkit ui: fsnotify unavailable (%v) — using polling\n", err)
		es = watcher.NewPollSource(vault, 0)
	}
	model = model.WithWatcher(es)
	defer es.Close()

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
