package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jz-wilson/vkit/internal/validate"
	"github.com/jz-wilson/vkit/internal/vaultpath"
	"github.com/jz-wilson/vkit/internal/watcher"
)

// listenCmd bridges a watcher.EventSource into the Bubble Tea loop: it blocks on
// a single event read and reports it as a FileChangedMsg, then the model
// re-issues listenCmd to wait for the next one. This keeps the channel read off
// the Update/View goroutine so the UI never blocks on the filesystem. When the
// source's channel closes it reports watchClosedMsg and the model stops
// re-arming.
func listenCmd(src watcher.EventSource) tea.Cmd {
	return func() tea.Msg {
		path, ok := <-src.Events()
		if !ok {
			return watchClosedMsg{}
		}
		return FileChangedMsg{Path: path}
	}
}

// validateVaultCmd validates every note in the vault and reports the result.
func validateVaultCmd(vault string) tea.Cmd {
	return func() tea.Msg {
		var rels []string
		_ = vaultpath.WalkNotes(vault, func(rel string) error {
			if !validate.ShouldSkip(rel) {
				rels = append(rels, rel)
			}
			return nil
		})
		probs, err := validate.Files(vault, rels)
		return ValidationDoneMsg{Rel: "", Problems: probs, Err: err}
	}
}

// validateFileCmd validates a single note (the 'v' keybinding on the highlighted
// note).
func validateFileCmd(vault, rel string) tea.Cmd {
	return func() tea.Msg {
		probs, err := validate.Files(vault, []string{rel})
		return ValidationDoneMsg{Rel: rel, Problems: probs, Err: err}
	}
}
