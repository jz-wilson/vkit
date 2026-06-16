package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// asRoot casts a tea.Model back to RootModel for white-box assertions.
func asRoot(t *testing.T, m tea.Model) RootModel {
	t.Helper()
	r, ok := m.(RootModel)
	if !ok {
		t.Fatalf("expected RootModel, got %T", m)
	}
	return r
}

func key(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestNewRootModelStartsFocusedOnNav(t *testing.T) {
	m := NewRootModel("/vault")
	if m.focus != navPanel {
		t.Fatalf("focus = %v, want nav", m.focus)
	}
	if !m.nav.focused || m.work.focused || m.log.focused {
		t.Fatalf("only nav should be focused: nav=%v work=%v log=%v",
			m.nav.focused, m.work.focused, m.log.focused)
	}
}

func TestTabCyclesFocusForward(t *testing.T) {
	m := NewRootModel("/vault")
	want := []panelID{workspacePanel, logPanel, navPanel}
	var model tea.Model = m
	for i, w := range want {
		var cmd tea.Cmd
		model, cmd = model.Update(key("tab"))
		if cmd != nil {
			t.Fatalf("tab #%d returned non-nil cmd", i)
		}
		if got := asRoot(t, model).focus; got != w {
			t.Fatalf("after %d tabs focus = %v, want %v", i+1, got, w)
		}
	}
	// Exactly one panel focused after a full cycle.
	r := asRoot(t, model)
	if !r.nav.focused || r.work.focused || r.log.focused {
		t.Fatalf("focus flags out of sync after cycle")
	}
}

func TestShiftTabCyclesBackward(t *testing.T) {
	var model tea.Model = NewRootModel("/vault")
	model, _ = model.Update(key("shift+tab"))
	if got := asRoot(t, model).focus; got != logPanel {
		t.Fatalf("shift+tab focus = %v, want log", got)
	}
}

func TestQuitKeys(t *testing.T) {
	for _, k := range []string{"q", "ctrl+c"} {
		var model tea.Model = NewRootModel("/vault")
		model, cmd := model.Update(key(k))
		if cmd == nil {
			t.Fatalf("%q returned nil cmd, want tea.Quit", k)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("%q cmd did not produce tea.QuitMsg", k)
		}
		if !asRoot(t, model).quitting {
			t.Fatalf("%q did not set quitting", k)
		}
	}
}

func TestWindowSizePropagatesToPanels(t *testing.T) {
	var model tea.Model = NewRootModel("/vault")
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	r := asRoot(t, model)
	if r.width != 120 || r.height != 40 {
		t.Fatalf("root size = %dx%d, want 120x40", r.width, r.height)
	}
	if r.nav.width <= 0 || r.work.width <= 0 || r.log.width <= 0 {
		t.Fatalf("panel widths not propagated: nav=%d work=%d log=%d",
			r.nav.width, r.work.width, r.log.width)
	}
	// Nav and right column should partition the full width.
	if r.nav.width+r.work.width != 120 {
		t.Fatalf("nav+work width = %d, want 120", r.nav.width+r.work.width)
	}
}

func TestFocusedBorderIsDistinct(t *testing.T) {
	const thickTL = "┏" // ThickBorder top-left (focused)
	const roundTL = "╭" // RoundedBorder top-left (blurred)

	focused := WorkspacePanel{}.Focused(true).Resize(40, 10).View()
	blurred := WorkspacePanel{}.Focused(false).Resize(40, 10).View()

	if !strings.Contains(focused, thickTL) {
		t.Fatalf("focused panel missing thick border %q", thickTL)
	}
	if !strings.Contains(blurred, roundTL) {
		t.Fatalf("blurred panel missing rounded border %q", roundTL)
	}
	if strings.Contains(blurred, thickTL) {
		t.Fatalf("blurred panel should not use the thick border")
	}
}

func TestViewRendersAllPanelTitles(t *testing.T) {
	var model tea.Model = NewRootModel("/vault")
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := asRoot(t, model).View()
	for _, title := range []string{"Notes", "Workspace", "Validation"} {
		if !strings.Contains(view, title) {
			t.Fatalf("view missing panel title %q", title)
		}
	}
	if !strings.Contains(view, "tab cycle") {
		t.Fatalf("view missing help bar")
	}
}
