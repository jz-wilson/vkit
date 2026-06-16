package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// noteLoadedMsg is the async result of reading a note's raw bytes off disk. It
// is produced by loadNoteCmd so the file read never blocks Update/View.
type noteLoadedMsg struct {
	rel     string
	content string
	err     error
}

// WorkspacePanel renders the selected note's raw contents in a scrolling
// viewport. It reacts to FileSelectedMsg by loading the note asynchronously and
// scrolls (j/k, arrows, page keys) while focused. Updated in the functional
// style: mutators return a copy.
type WorkspacePanel struct {
	focused       bool
	width, height int
	vault         string
	rel           string // currently displayed note ("" if none)
	vp            viewport.Model
}

// NewWorkspacePanel builds a workspace bound to a vault root. The viewport is
// constructed via viewport.New so it carries the default scroll keymap.
func NewWorkspacePanel(vault string) WorkspacePanel {
	return WorkspacePanel{vault: vault, vp: viewport.New(0, 0)}
}

// loadNoteCmd reads vault/rel off disk and reports it as a noteLoadedMsg.
func loadNoteCmd(vault, rel string) tea.Cmd {
	return func() tea.Msg {
		b, err := os.ReadFile(filepath.Join(vault, filepath.FromSlash(rel)))
		if err != nil {
			return noteLoadedMsg{rel: rel, err: err}
		}
		return noteLoadedMsg{rel: rel, content: string(b)}
	}
}

// Focused returns a copy with the focus flag set.
func (p WorkspacePanel) Focused(b bool) WorkspacePanel { p.focused = b; return p }

// Resize returns a copy sized to w x h cells, sizing the viewport to the inner
// content area.
func (p WorkspacePanel) Resize(w, h int) WorkspacePanel {
	p.width, p.height = w, h
	iw, ih := innerDims(w, h)
	p.vp.Width = iw
	p.vp.Height = ih
	return p
}

// Update loads notes on selection, ingests load results, and forwards other
// messages to the viewport for scrolling (only while focused).
func (p WorkspacePanel) Update(msg tea.Msg) (WorkspacePanel, tea.Cmd) {
	switch msg := msg.(type) {
	case FileSelectedMsg:
		if msg.Rel == p.rel {
			return p, nil // already showing this note
		}
		return p, loadNoteCmd(p.vault, msg.Rel)

	case noteLoadedMsg:
		p.rel = msg.rel
		if msg.err != nil {
			p.vp.SetContent(fmt.Sprintf("error loading %s:\n\n%v", msg.rel, msg.err))
		} else {
			p.vp.SetContent(msg.content)
		}
		p.vp.GotoTop()
		return p, nil
	}

	if p.focused {
		var cmd tea.Cmd
		p.vp, cmd = p.vp.Update(msg)
		return p, cmd
	}
	return p, nil
}

// View renders the note body (or a placeholder) inside the panel chrome, with
// the note's relative path in the title.
func (p WorkspacePanel) View() string {
	title := "Workspace"
	var body string
	if p.rel == "" {
		body = "(no note selected)"
	} else {
		title = "Workspace · " + p.rel
		body = p.vp.View()
	}
	return renderPanel(title, body, p.focused, p.width, p.height)
}
