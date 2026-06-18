// Package tui implements the interactive multi-pane `vkit ui` terminal app on
// top of charmbracelet/bubbletea. The RootModel owns three panels — a note
// navigator, a workspace viewport, and a validation log — and brokers all
// cross-panel communication through the tea.Msg types defined in this file so
// modules stay decoupled and the rendering thread never blocks on I/O.
package tui

import "github.com/jz-wilson/vkit/internal/validate"

// FileSelectedMsg is emitted by the NavPanel whenever the highlighted note
// changes. The WorkspacePanel loads the note body in response; the RootModel
// may also kick off a validation run for the selected file.
type FileSelectedMsg struct{ Rel string }

// ValidationTriggeredMsg requests a compliance check. An empty Rel means
// "validate the whole vault"; a non-empty Rel scopes the check to one note.
type ValidationTriggeredMsg struct{ Rel string }

// ValidationDoneMsg carries the outcome of a validation run back to the
// LogPanel. It is produced asynchronously by a tea.Cmd so the engine call
// never blocks Update/View.
type ValidationDoneMsg struct {
	Rel      string
	Problems []validate.Problem
	Err      error
}

// FileChangedMsg is delivered by the watcher bridge each time the underlying
// watcher.EventSource reports a changed file on disk. The RootModel reacts by
// re-validating and, in later phases, refreshing the workspace.
type FileChangedMsg struct{ Path string }

// watchClosedMsg signals that the watcher's event channel was closed (the
// EventSource shut down). The bridge stops re-arming itself once this is seen.
type watchClosedMsg struct{}

// errMsg wraps a background error so it can be surfaced in the UI without
// panicking the program loop.
type errMsg struct{ err error }
