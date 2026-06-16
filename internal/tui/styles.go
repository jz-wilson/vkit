package tui

import "github.com/charmbracelet/lipgloss"

// Panel chrome colors. Focused panels are rendered green; blurred panels gray.
var (
	colorFocus = lipgloss.Color("#22c55e")
	colorBlur  = lipgloss.Color("#6b7280")
	colorTitle = lipgloss.Color("#e5e7eb")
)

// panelStyle returns the border style for a panel. Focused and blurred panels
// use deliberately distinct border designs (thick green vs. rounded gray) so
// the focused pane is obvious at a glance.
func panelStyle(focused bool) lipgloss.Style {
	border := lipgloss.RoundedBorder()
	color := colorBlur
	if focused {
		border = lipgloss.ThickBorder()
		color = colorFocus
	}
	return lipgloss.NewStyle().
		Border(border).
		BorderForeground(color).
		Padding(0, 1)
}

// innerDims returns the usable content area inside a panel of w x h cells, after
// reserving space for the border (2 each axis), horizontal padding (2), and the
// title line (1). Panels size their bubbles components to these dimensions so
// content lines up exactly inside the rendered border.
func innerDims(w, h int) (int, int) {
	iw := w - 4
	if iw < 1 {
		iw = 1
	}
	ih := h - 3
	if ih < 1 {
		ih = 1
	}
	return iw, ih
}

// renderPanel draws a titled, bordered box sized to fit within w x h cells. The
// border style switches on focus. Body is clipped/padded to the inner area.
func renderPanel(title, body string, focused bool, w, h int) string {
	style := panelStyle(focused)
	innerW, innerH := innerDims(w, h)

	head := lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render(title)
	content := lipgloss.NewStyle().Width(innerW).Height(innerH).MaxHeight(innerH).Render(body)
	return style.Width(innerW).Render(head + "\n" + content)
}

// colorError is used for validation problem text.
var colorError = lipgloss.Color("#ef4444")

var (
	okStyle  = lipgloss.NewStyle().Foreground(colorFocus).Bold(true)
	errStyle = lipgloss.NewStyle().Foreground(colorError)
)
