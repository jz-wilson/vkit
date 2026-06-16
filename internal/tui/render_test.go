package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// maxLineWidth returns the widest visible line in s (ANSI-aware).
func maxLineWidth(s string) int {
	mx := 0
	for _, line := range strings.Split(s, "\n") {
		if w := lipgloss.Width(line); w > mx {
			mx = w
		}
	}
	return mx
}

// TestRenderPanelNeverExceedsWidth guards against the overflow that made nav
// rows wrap and long workspace titles corrupt the layout: every rendered line
// must fit within the requested box width, even for over-long titles/bodies.
func TestRenderPanelNeverExceedsWidth(t *testing.T) {
	longTitle := "Workspace · agents/claude-code/reports/2026-06-03-antigravity-protocol-verification.md"
	longBody := strings.Repeat("agents/claude-code/reports/a-very-long-note-path-name.md\n", 6)
	for _, w := range []int{12, 20, 40, 80} {
		out := renderPanel(longTitle, longBody, true, w, 8)
		if mx := maxLineWidth(out); mx > w {
			t.Fatalf("w=%d: rendered line width %d exceeds box width %d", w, mx, w)
		}
	}
}

// TestFullViewNeverExceedsTerminalWidth renders the whole model and asserts the
// composed layout fits the terminal and shows all three panels.
func TestFullViewNeverExceedsTerminalWidth(t *testing.T) {
	const termW = 100
	var model tea.Model = NewRootModel("/nonexistent-vault")
	model, _ = model.Update(tea.WindowSizeMsg{Width: termW, Height: 30})
	view := model.(RootModel).View()

	if mx := maxLineWidth(view); mx > termW {
		t.Fatalf("composed view line width %d exceeds terminal width %d", mx, termW)
	}
	for _, title := range []string{"Notes", "Workspace", "Validation"} {
		if !strings.Contains(view, title) {
			t.Fatalf("composed view missing panel %q", title)
		}
	}
}
