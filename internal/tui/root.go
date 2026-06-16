package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jz-wilson/vkit/internal/watcher"
)

// panelID identifies one of the three focusable panels and defines their cycle
// order under the tab key.
type panelID int

const (
	navPanel panelID = iota
	workspacePanel
	logPanel
	panelCount // sentinel: number of panels
)

func (id panelID) String() string {
	switch id {
	case navPanel:
		return "nav"
	case workspacePanel:
		return "workspace"
	case logPanel:
		return "log"
	default:
		return "unknown"
	}
}

// RootModel is the top-level Bubble Tea model. It owns the three panels, tracks
// which one has focus, and routes messages. All state transitions are pure:
// Update returns a new RootModel rather than mutating the receiver.
type RootModel struct {
	vault         string
	focus         panelID
	nav           NavPanel
	work          WorkspacePanel
	log           LogPanel
	watch         watcher.EventSource // optional live-validation source
	status        string              // transient status line (MOC/validate)
	width, height int
	quitting      bool
}

// NewRootModel builds the initial model for the given vault root with focus on
// the nav panel. It is pure (no filesystem watchers); attach a live watcher with
// WithWatcher.
func NewRootModel(vault string) RootModel {
	m := RootModel{
		vault: vault,
		focus: navPanel,
		nav:   NewNavPanel(vault),
		work:  NewWorkspacePanel(vault),
		log:   NewLogPanel(),
	}
	return m.syncFocus()
}

// WithWatcher attaches an EventSource so file changes trigger live revalidation.
// The model takes ownership: the source is consumed via listenCmd and closed by
// the program's Run wrapper.
func (m RootModel) WithWatcher(src watcher.EventSource) RootModel {
	m.watch = src
	return m
}

// syncFocus pushes the model's focus state down into each panel so exactly one
// panel reports focused.
func (m RootModel) syncFocus() RootModel {
	m.nav = m.nav.Focused(m.focus == navPanel)
	m.work = m.work.Focused(m.focus == workspacePanel)
	m.log = m.log.Focused(m.focus == logPanel)
	return m
}

// cycleFocus advances focus by delta (wrapping) and re-syncs the panels.
func (m RootModel) cycleFocus(delta int) RootModel {
	n := int(panelCount)
	m.focus = panelID(((int(m.focus)+delta)%n + n) % n)
	return m.syncFocus()
}

// layout recomputes panel sizes from the current terminal dimensions: nav on
// the left, workspace over log on the right, with one line reserved for help.
func (m RootModel) layout() RootModel {
	if m.width <= 0 || m.height <= 0 {
		return m
	}
	navW := m.width / 4
	if navW < 24 {
		navW = 24
	}
	if navW > m.width {
		navW = m.width
	}
	rightW := m.width - navW
	bodyH := m.height - 1 // reserve the help bar
	if bodyH < 1 {
		bodyH = 1
	}
	workH := bodyH * 2 / 3
	if workH < 1 {
		workH = 1
	}
	logH := bodyH - workH
	if logH < 1 {
		logH = 1
	}

	m.nav = m.nav.Resize(navW, bodyH)
	m.work = m.work.Resize(rightW, workH)
	m.log = m.log.Resize(rightW, logH)
	return m
}

// Init primes the workspace with the starting note, runs a first whole-vault
// validation, and (if a watcher is attached) starts listening for file changes
// so live validation begins immediately.
func (m RootModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.nav.SelectCmd(), validateVaultCmd(m.vault)}
	if m.watch != nil {
		cmds = append(cmds, listenCmd(m.watch))
	}
	return tea.Batch(cmds...)
}

// Update is the central message router.
func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.layout(), nil

	case tea.KeyMsg:
		// While the nav filter is active, the list owns every keystroke (so the
		// query can contain 'q', 'tab', etc.) except the hard ctrl+c quit.
		if m.focus == navPanel && m.nav.Filtering() {
			if msg.Type == tea.KeyCtrlC {
				m.quitting = true
				return m, tea.Quit
			}
			return m.updateFocused(msg)
		}
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "tab":
			return m.cycleFocus(1), nil
		case "shift+tab":
			return m.cycleFocus(-1), nil
		case "v":
			return m, m.validateSelectedCmd()
		case "m":
			return m, rebuildMocCmd(m.vault)
		}
		// Other keys go only to the focused panel.
		return m.updateFocused(msg)

	case FileChangedMsg:
		// A note changed on disk: re-validate the vault and re-arm the listener.
		return m.broadcast(msg, listenCmd(m.watch), validateVaultCmd(m.vault))

	case watchClosedMsg:
		// The watcher stopped; do not re-arm. Surface it once.
		m.status = "watcher stopped"
		return m.broadcast(msg)

	case ValidationDoneMsg:
		m.status = validationStatus(msg)
		return m.broadcast(msg)

	case MocRebuiltMsg:
		if msg.Err != nil {
			m.status = "MOC rebuild failed: " + msg.Err.Error()
		} else {
			m.status = fmt.Sprintf("MOC rebuilt — %d notes", msg.Count)
		}
		return m.broadcast(msg)
	}

	// Any other message is broadcast so async results reach whichever panel cares.
	return m.broadcast(msg)
}

// validateSelectedCmd validates the highlighted note, or the whole vault if no
// note is selected.
func (m RootModel) validateSelectedCmd() tea.Cmd {
	rel := m.nav.Selected()
	if rel == "" {
		return validateVaultCmd(m.vault)
	}
	return validateFileCmd(m.vault, rel)
}

// validationStatus summarizes a validation result for the status line.
func validationStatus(msg ValidationDoneMsg) string {
	scope := msg.Rel
	if scope == "" {
		scope = "vault"
	}
	switch {
	case msg.Err != nil:
		return "validation error"
	case len(msg.Problems) == 0:
		return "✓ " + scope + " clean"
	case len(msg.Problems) == 1:
		return fmt.Sprintf("1 problem in %s", scope)
	default:
		return fmt.Sprintf("%d problems in %s", len(msg.Problems), scope)
	}
}

// updateFocused routes a message to the currently focused panel only.
func (m RootModel) updateFocused(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focus {
	case navPanel:
		m.nav, cmd = m.nav.Update(msg)
	case workspacePanel:
		m.work, cmd = m.work.Update(msg)
	case logPanel:
		m.log, cmd = m.log.Update(msg)
	}
	return m, cmd
}

// broadcast sends a message to all three panels, batching their commands with
// any extra commands the caller supplies (e.g. re-arming the watcher).
func (m RootModel) broadcast(msg tea.Msg, extra ...tea.Cmd) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, 3+len(extra))
	var c tea.Cmd
	m.nav, c = m.nav.Update(msg)
	cmds = append(cmds, c)
	m.work, c = m.work.Update(msg)
	cmds = append(cmds, c)
	m.log, c = m.log.Update(msg)
	cmds = append(cmds, c)
	cmds = append(cmds, extra...)
	return m, tea.Batch(cmds...)
}

// View composes the three panels and the status/help bar.
func (m RootModel) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return "initializing vkit ui..."
	}
	right := lipgloss.JoinVertical(lipgloss.Left, m.work.View(), m.log.View())
	body := lipgloss.JoinHorizontal(lipgloss.Top, m.nav.View(), right)
	return body + "\n" + m.statusBar()
}

// statusBar renders the keybinding hints plus the transient status message.
func (m RootModel) statusBar() string {
	help := lipgloss.NewStyle().Foreground(colorBlur).
		Render("tab cycle | j/k move | v validate | m rebuild MOC | q quit")
	if m.status == "" {
		return help
	}
	return help + lipgloss.NewStyle().Foreground(colorTitle).Render("   "+m.status)
}
