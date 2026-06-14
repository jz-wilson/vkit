package scaffold

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// TestLineDiffCounts characterizes the LCS-based added/removed counts for the
// four canonical cases: identical, all-added, all-removed, and mixed.
func TestLineDiffCounts(t *testing.T) {
	cases := []struct {
		name        string
		cur, kit    string
		wantAdded   int
		wantRemoved int
	}{
		{
			name: "identical",
			cur:  "a\nb\nc\n",
			kit:  "a\nb\nc\n",
		},
		{
			name:      "all added (cur empty)",
			cur:       "",
			kit:       "a\nb\nc\n",
			wantAdded: 3,
		},
		{
			name:        "all removed (kit empty)",
			cur:         "a\nb\nc\n",
			kit:         "",
			wantRemoved: 3,
		},
		{
			name:        "mixed: one changed line",
			cur:         "a\nb\nc\n",
			kit:         "a\nX\nc\n",
			wantAdded:   1,
			wantRemoved: 1,
		},
		{
			name:        "mixed: insert in middle",
			cur:         "a\nc\n",
			kit:         "a\nb\nc\n",
			wantAdded:   1,
			wantRemoved: 0,
		},
		{
			name:        "mixed: delete from middle",
			cur:         "a\nb\nc\n",
			kit:         "a\nc\n",
			wantAdded:   0,
			wantRemoved: 1,
		},
		{
			name:        "both empty",
			cur:         "",
			kit:         "",
			wantAdded:   0,
			wantRemoved: 0,
		},
		{
			name:        "trailing newline ignored (no diff)",
			cur:         "a\nb",
			kit:         "a\nb\n",
			wantAdded:   0,
			wantRemoved: 0,
		},
		{
			name:        "full replacement",
			cur:         "x\ny\n",
			kit:         "a\nb\nc\n",
			wantAdded:   3,
			wantRemoved: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			added, removed, hunk := lineDiff(tc.cur, tc.kit)
			if added != tc.wantAdded {
				t.Errorf("added=%d want %d", added, tc.wantAdded)
			}
			if removed != tc.wantRemoved {
				t.Errorf("removed=%d want %d", removed, tc.wantRemoved)
			}
			// The hunk's own +/- line counts must agree with the returned counts.
			gotAdded, gotRemoved := countHunk(hunk)
			if gotAdded != added {
				t.Errorf("hunk + lines=%d, but added=%d", gotAdded, added)
			}
			if gotRemoved != removed {
				t.Errorf("hunk - lines=%d, but removed=%d", gotRemoved, removed)
			}
		})
	}
}

// countHunk tallies the + and - prefixed lines in a unified hunk string.
func countHunk(hunk string) (added, removed int) {
	for _, line := range strings.Split(hunk, "\n") {
		switch {
		case strings.HasPrefix(line, "+ "):
			added++
		case strings.HasPrefix(line, "- "):
			removed++
		}
	}
	return added, removed
}

// TestLineDiffHunkFormat characterizes the unified-style hunk output: context
// lines are prefixed with two spaces, removals with "- ", additions with "+ ".
func TestLineDiffHunkFormat(t *testing.T) {
	_, _, hunk := lineDiff("a\nb\nc\n", "a\nX\nc\n")
	want := "  a\n- b\n+ X\n  c\n"
	if hunk != want {
		t.Errorf("hunk=%q want %q", hunk, want)
	}
}

// TestDelta characterizes the "+added −removed" string (note: the separator is
// the U+2212 minus sign, not an ASCII hyphen).
func TestDelta(t *testing.T) {
	cases := []struct {
		name     string
		cur, kit string
		want     string
	}{
		{"identical", "a\nb\n", "a\nb\n", "+0 −0"},
		{"all added", "", "a\nb\nc\n", "+3 −0"},
		{"all removed", "a\nb\nc\n", "", "+0 −3"},
		{"mixed", "a\nb\nc\n", "a\nX\nc\n", "+1 −1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := delta(tc.cur, tc.kit); got != tc.want {
				t.Errorf("delta=%q want %q", got, tc.want)
			}
		})
	}
}

// TestSplitLines characterizes the trailing-newline-stripping splitter, which
// treats "" and "\n" as zero lines.
func TestSplitLines(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single newline", "\n", nil},
		{"one line no newline", "a", []string{"a"}},
		{"one line trailing newline", "a\n", []string{"a"}},
		{"multi", "a\nb\nc\n", []string{"a", "b", "c"}},
		{"blank line preserved internally", "a\n\nb\n", []string{"a", "", "b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitLines(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("splitLines(%q)=%#v want %#v", tc.in, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("line %d = %q want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestPromptDiffBranch drives the interactive prompt's [d]iff branch via the
// injectable in/out readers, then quits. It characterizes that selecting "d"
// renders a diff hunk for each changed template before returning to the menu.
func TestPromptDiffBranch(t *testing.T) {
	v := dirtyVault(t)
	var out bytes.Buffer
	// "d" prints diffs, then "q" quits with no changes.
	res, err := Update(v, ModePrompt, false, strings.NewReader("d\nq\n"), &out, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != "quit" {
		t.Fatalf("res=%+v", res)
	}
	s := out.String()
	if !strings.Contains(s, "--- diff: CLAUDE.md (current vs kit) ---") {
		t.Errorf("diff header missing:\n%s", s)
	}
	// The custom content should appear as a removed line in the hunk.
	if !strings.Contains(s, "- # my custom claude") {
		t.Errorf("expected removed line for custom CLAUDE.md:\n%s", s)
	}
	// quitting changes nothing.
	if read(t, filepath.Join(v, "CLAUDE.md")) != "# my custom claude\n" {
		t.Error("CLAUDE.md modified after [d]iff + [q]uit")
	}
}

// TestCustomizeStreamBufferingQuirk characterizes a LATENT BUFFERING QUIRK in
// the production code, NOT a regression introduced here.
//
// promptMenu wraps `in` in its own bufio.NewReader; customize() then wraps the
// SAME underlying `in` in a SECOND, independent bufio.NewReader. When the whole
// script is supplied as one buffered stream (a pipe, or strings.Reader in a
// test), promptMenu's bufio reader reads ahead and swallows the per-file answers
// while consuming the "c\n" menu choice. customize()'s fresh reader then sees
// EOF, so every ask() returns false and every file is declined — regardless of
// the y/N bytes that followed "c".
//
// With a real interactive TTY this does not bite: each ReadString('\n') returns
// as soon as the user presses Enter, so promptMenu's reader does not race ahead
// of the customize prompts. The quirk is observable only with buffered input.
//
// This test pins the current (buffered) behavior: "c\ny\ny\ny\n" still declines
// everything, leaving the changed tier2 file as a Keep.
func TestCustomizeStreamBufferingQuirk(t *testing.T) {
	v := dirtyVault(t)
	var out bytes.Buffer
	res, err := Update(v, ModePrompt, false, strings.NewReader("c\ny\ny\ny\n"), &out, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != "customize" {
		t.Fatalf("res=%+v", res)
	}
	// Despite three "y" answers, the second bufio reader sees EOF -> all declined.
	if res.Tool != 0 || res.New != 0 || res.Over != 0 || res.Keep != 1 {
		t.Fatalf("res=%+v want Tool=0 New=0 Over=0 Keep=1 (buffering quirk)", res)
	}
	// .gitignore untouched (declined), CLAUDE.md untouched (declined, Keep).
	if strings.Contains(read(t, filepath.Join(v, ".gitignore")), ".moc-stamp") {
		t.Error(".gitignore refreshed despite buffering quirk declining it")
	}
	if read(t, filepath.Join(v, "CLAUDE.md")) != "# my custom claude\n" {
		t.Error("CLAUDE.md modified despite buffering quirk declining it")
	}
	if exists2(filepath.Join(v, "CLAUDE.md.bak")) {
		t.Error("no .bak should exist when everything is declined")
	}
	if exists2(filepath.Join(v, ".claude", "commands", "note.md")) {
		t.Error("note.md created despite buffering quirk declining it")
	}
}

// TestCustomizeDeclineKeeps characterizes that the changed tier2 file is counted
// as Keep and left untouched when customize declines it. (Per the buffering
// quirk above, the trailing "n" answers are actually swallowed, so this is the
// same code path as TestCustomizeStreamBufferingQuirk reached via explicit n's.)
func TestCustomizeDeclineKeeps(t *testing.T) {
	v := dirtyVault(t)
	var out bytes.Buffer
	res, err := Update(v, ModePrompt, false, strings.NewReader("c\nn\nn\nn\n"), &out, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Tool != 0 || res.New != 0 || res.Over != 0 || res.Keep != 1 {
		t.Fatalf("res=%+v want Tool=0 New=0 Over=0 Keep=1", res)
	}
	if read(t, filepath.Join(v, "CLAUDE.md")) != "# my custom claude\n" {
		t.Error("CLAUDE.md modified despite declining")
	}
	if exists2(filepath.Join(v, "CLAUDE.md.bak")) {
		t.Error("declining created a .bak")
	}
}

// TestPromptInvalidThenQuit characterizes the default branch of the menu: an
// unrecognized choice reprints the hint and loops until a valid choice.
func TestPromptInvalidThenQuit(t *testing.T) {
	v := dirtyVault(t)
	var out bytes.Buffer
	res, err := Update(v, ModePrompt, false, strings.NewReader("z\nq\n"), &out, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != "quit" {
		t.Fatalf("res=%+v", res)
	}
	if !strings.Contains(out.String(), "pick a, s, c, d, or q.") {
		t.Errorf("missing invalid-choice hint:\n%s", out.String())
	}
}

// TestPromptEOFQuits characterizes that an empty/EOF reader defaults to quit.
func TestPromptEOFQuits(t *testing.T) {
	v := dirtyVault(t)
	var out bytes.Buffer
	res, err := Update(v, ModePrompt, false, strings.NewReader(""), &out, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != "quit" {
		t.Fatalf("res=%+v", res)
	}
}
