package tui

import (
	"sort"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jz-wilson/vkit/internal/vaultpath"
)

// noteItem adapts a vault-relative note path to the bubbles/list Item
// interface. The relative path is both the label and the filter value.
type noteItem struct{ rel string }

func (i noteItem) Title() string       { return i.rel }
func (i noteItem) Description() string { return "" }
func (i noteItem) FilterValue() string { return i.rel }

// NavPanel browses the vault's notes as an interactive, filterable list backed
// by bubbles/list. Moving the cursor (j/k or arrows) emits a FileSelectedMsg so
// the workspace and validation panels can react. It is updated in the Bubble
// Tea functional style: mutators return a copy.
type NavPanel struct {
	focused       bool
	width, height int
	list          list.Model
	selected      string // rel path of the highlighted note ("" if none)
}

// NewNavPanel builds a NavPanel populated from a single canonical vault walk
// (vaultpath.WalkNotes), so it agrees with moc/validate on what counts as a
// note. A non-existent or empty vault yields an empty list rather than an error.
func NewNavPanel(vault string) NavPanel {
	items := loadNoteItems(vault)

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false // single-line rows
	delegate.SetSpacing(0)
	delegate.SetHeight(1)

	l := list.New(items, delegate, 0, 0)
	l.SetShowTitle(false)     // the panel border supplies the title
	l.SetShowStatusBar(false) // keep the pane compact
	l.SetShowHelp(false)      // help is shown globally in the root help bar
	l.SetShowPagination(true) // page indicator when notes overflow the pane
	l.SetFilteringEnabled(true)

	p := NavPanel{list: l}
	if it, ok := l.SelectedItem().(noteItem); ok {
		p.selected = it.rel
	}
	return p
}

// loadNoteItems walks the vault once and returns its notes as list items,
// byte-sorted for a stable, predictable order (matching the MOC ordering).
func loadNoteItems(vault string) []list.Item {
	var rels []string
	_ = vaultpath.WalkNotes(vault, func(rel string) error {
		rels = append(rels, rel)
		return nil
	})
	sort.Strings(rels)
	items := make([]list.Item, len(rels))
	for i, rel := range rels {
		items[i] = noteItem{rel: rel}
	}
	return items
}

// Selected returns the rel path of the currently highlighted note, or "".
func (p NavPanel) Selected() string { return p.selected }

// Filtering reports whether the list's incremental filter is currently active
// (the user is typing a query). While true the root forwards keystrokes to the
// list instead of treating them as global shortcuts.
func (p NavPanel) Filtering() bool { return p.list.FilterState() == list.Filtering }

// SelectCmd returns a command that emits the current selection as a
// FileSelectedMsg, or nil if nothing is selected. The root uses it on startup
// to prime the workspace with the first note.
func (p NavPanel) SelectCmd() tea.Cmd {
	if p.selected == "" {
		return nil
	}
	rel := p.selected
	return func() tea.Msg { return FileSelectedMsg{Rel: rel} }
}

// Focused returns a copy with the focus flag set.
func (p NavPanel) Focused(b bool) NavPanel { p.focused = b; return p }

// Resize returns a copy sized to w×h cells, sizing the inner list to the panel's
// content area so rows align inside the border.
func (p NavPanel) Resize(w, h int) NavPanel {
	p.width, p.height = w, h
	iw, ih := innerDims(w, h)
	p.list.SetSize(iw, ih)
	return p
}

// Update forwards the message to the inner list and, if the highlighted note
// changed as a result, emits a FileSelectedMsg.
func (p NavPanel) Update(msg tea.Msg) (NavPanel, tea.Cmd) {
	prev := p.selected
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	cmds := []tea.Cmd{cmd}

	if it, ok := p.list.SelectedItem().(noteItem); ok {
		p.selected = it.rel
	} else {
		p.selected = ""
	}
	if p.selected != "" && p.selected != prev {
		cmds = append(cmds, p.SelectCmd())
	}
	return p, tea.Batch(cmds...)
}

// View renders the note list inside the panel chrome.
func (p NavPanel) View() string {
	return renderPanel("Notes", p.list.View(), p.focused, p.width, p.height)
}
