package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jz-wilson/vkit/internal/validate"
)

// LogPanel lists lint/validation problems produced by the validate engine. It
// ingests ValidationDoneMsg values (emitted by validateVaultCmd/validateFileCmd
// in response to the 'v' key or a watcher event) and renders them.
type LogPanel struct {
	focused       bool
	width, height int
	checked       bool               // a validation run has completed
	scope         string             // what was validated ("vault" or a rel path)
	problems      []validate.Problem // problems from the most recent run
	loadErr       error              // engine error, if any
}

// NewLogPanel returns an empty log panel.
func NewLogPanel() LogPanel { return LogPanel{} }

// Focused returns a copy with the focus flag set.
func (p LogPanel) Focused(b bool) LogPanel { p.focused = b; return p }

// Resize returns a copy sized to w x h cells.
func (p LogPanel) Resize(w, h int) LogPanel { p.width, p.height = w, h; return p }

// Update ingests validation results.
func (p LogPanel) Update(msg tea.Msg) (LogPanel, tea.Cmd) {
	if m, ok := msg.(ValidationDoneMsg); ok {
		p.checked = true
		p.scope = m.Rel
		if p.scope == "" {
			p.scope = "vault"
		}
		p.loadErr = m.Err
		p.problems = m.Problems
	}
	return p, nil
}

// Problems exposes the most recent problem set (used by the root status line).
func (p LogPanel) Problems() []validate.Problem { return p.problems }

func (p LogPanel) title() string {
	switch {
	case !p.checked:
		return "Validation"
	case p.loadErr != nil:
		return "Validation !"
	case len(p.problems) == 0:
		return "Validation ✓"
	default:
		return fmt.Sprintf("Validation (%d)", len(p.problems))
	}
}

func (p LogPanel) body() string {
	switch {
	case !p.checked:
		return "(no checks run yet — press v)"
	case p.loadErr != nil:
		return errStyle.Render("validation error: " + p.loadErr.Error())
	case len(p.problems) == 0:
		return okStyle.Render("✓ no problems") + " (" + p.scope + ")"
	}
	var b strings.Builder
	for _, pr := range p.problems {
		fmt.Fprintf(&b, "%s\n    %s\n", errStyle.Render(pr.File), pr.Msg)
	}
	return strings.TrimRight(b.String(), "\n")
}

// View renders the problem list inside the panel chrome.
func (p LogPanel) View() string {
	return renderPanel(p.title(), p.body(), p.focused, p.width, p.height)
}
