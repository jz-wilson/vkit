// Package style provides shared lipgloss styles and print helpers for vkit commands.
package style

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	green = lipgloss.Color("#22c55e")
	red   = lipgloss.Color("#ef4444")
	gray  = lipgloss.Color("#6b7280")
)

var (
	StyleSuccess = lipgloss.NewStyle().Foreground(green)
	StyleError   = lipgloss.NewStyle().Foreground(red)
	StyleLabel   = lipgloss.NewStyle().Bold(true)
	StyleDim     = lipgloss.NewStyle().Foreground(gray)
)

// Check returns a colored ✓ (green) or ✗ (red).
func Check(ok bool) string {
	if ok {
		return StyleSuccess.Render("✓")
	}
	return StyleError.Render("✗")
}

// Row renders a bold fixed-width label followed by a value.
func Row(label, value string) string {
	return fmt.Sprintf("  %s  %s", StyleLabel.Render(fmt.Sprintf("%-12s", label)), value)
}

// Section renders a section header: emoji + bold title.
func Section(emoji, title string) string {
	return emoji + "  " + StyleLabel.Render(title)
}

// Line renders a leading emoji + plain text on one line.
func Line(emoji, text string) string {
	return emoji + "  " + text
}

// Dim renders s in gray.
func Dim(s string) string {
	return StyleDim.Render(s)
}

// OK renders a green ✓ + text.
func OK(text string) string {
	return StyleSuccess.Render("✓") + "  " + text
}

// Fail renders a red ✗ + text.
func Fail(text string) string {
	return StyleError.Render("✗") + "  " + text
}

// Step renders a 2-space-indented per-substep line with a ✓ or ✗ prefix.
func Step(ok bool, msg string) string {
	return "  " + Check(ok) + "  " + msg
}

// Summary renders a dim footer with parts joined by " · ", 2-space indent.
func Summary(parts ...string) string {
	return StyleDim.Render("  " + strings.Join(parts, " · "))
}
