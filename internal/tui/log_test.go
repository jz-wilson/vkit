package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/jz-wilson/vkit/internal/validate"
)

func TestLogIngestsAndRendersProblems(t *testing.T) {
	p := NewLogPanel().Resize(60, 12)
	p, _ = p.Update(ValidationDoneMsg{
		Rel: "notes/a.md",
		Problems: []validate.Problem{
			{File: "notes/a.md", Msg: "missing 'updated:' frontmatter"},
		},
	})
	view := p.View()
	if !strings.Contains(view, "notes/a.md") || !strings.Contains(view, "missing 'updated:'") {
		t.Fatalf("log view missing problem detail:\n%s", view)
	}
	if !strings.Contains(view, "Validation (1)") {
		t.Fatalf("log title should show problem count:\n%s", view)
	}
	if got := p.Problems(); len(got) != 1 {
		t.Fatalf("Problems() = %d, want 1", len(got))
	}
}

func TestLogCleanState(t *testing.T) {
	p := NewLogPanel().Resize(60, 12)
	p, _ = p.Update(ValidationDoneMsg{Rel: "", Problems: nil})
	view := p.View()
	if !strings.Contains(view, "no problems") || !strings.Contains(view, "Validation ✓") {
		t.Fatalf("clean state not rendered:\n%s", view)
	}
}

func TestLogErrorState(t *testing.T) {
	p := NewLogPanel().Resize(60, 12)
	p, _ = p.Update(ValidationDoneMsg{Err: errors.New("boom")})
	if !strings.Contains(p.View(), "validation error") {
		t.Fatalf("error state not surfaced")
	}
}
