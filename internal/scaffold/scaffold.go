// Package scaffold installs the embedded vault tree and runs the eval-first
// update flow ported from setup.sh. The template tree is carried in the binary
// via go:embed, so there are no external script files to copy.
package scaffold

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:templates
var templates embed.FS

// Tier classification, relative to the vault root. Scripts are gone (the binary
// IS the scripts), so TIER1 is just the services + .gitignore tooling.
var tier1Files = []string{
	"services/README.md",
	"services/com.vault.watch.plist",
	"services/vault-watch.service",
	".gitignore",
}

var tier2Files = []string{
	"_format.md",
	"CLAUDE.md",
	".githooks/pre-commit",
	".claude/settings.json",
	".claude/commands/note.md",
	".claude/commands/rename.md",
	".claude/commands/sync.md",
}

// contentDirs are created (empty) on a fresh install.
var contentDirs = []string{"decisions", "infrastructure", "projects", "reference", "archive"}

// Mode pre-answers the update prompt.
type Mode int

const (
	ModePrompt Mode = iota // interactive (or quit if no tty)
	ModeForce              // apply all
	ModeKeep               // safe: tooling + new templates, keep changed
)

// Eval is the phase-1 analysis: what an update would change, written nowhere.
type Eval struct {
	T1Change []string // tooling absent or differing -> would refresh
	T2New    []string // templates absent -> would add
	T2Change []string // templates present & differing -> would overwrite
}

// Empty reports whether the vault already matches the kit.
func (e Eval) Empty() bool {
	return len(e.T1Change)+len(e.T2New)+len(e.T2Change) == 0
}

// Result summarizes an applied (or skipped) update.
type Result struct {
	AlreadyMatches        bool
	DryRun                bool
	Action                string // apply | safe | customize | quit
	Tool, New, Over, Keep int
}

// templateBytes reads an embedded template by its vault-relative path.
func templateBytes(rel string) ([]byte, error) {
	return templates.ReadFile("templates/" + rel)
}

// copyTemplate writes an embedded template into the vault, creating parents.
func copyTemplate(vault, rel string) error {
	data, err := templateBytes(rel)
	if err != nil {
		return err
	}
	dst := filepath.Join(vault, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if rel == ".githooks/pre-commit" {
		mode = 0o755
	}
	return os.WriteFile(dst, data, mode)
}

// Install lays down the full tree for a fresh vault.
func Install(vault string) error {
	if err := os.MkdirAll(vault, 0o755); err != nil {
		return err
	}
	err := fs.WalkDir(templates, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, "templates/")
		return copyTemplate(vault, rel)
	})
	if err != nil {
		return err
	}
	for _, dir := range contentDirs {
		if err := os.MkdirAll(filepath.Join(vault, dir), 0o755); err != nil {
			return err
		}
	}
	return nil
}

// differs reports whether the vault file is absent or unequal to the template.
func differs(vault, rel string) bool {
	want, err := templateBytes(rel)
	if err != nil {
		return false
	}
	got, err := os.ReadFile(filepath.Join(vault, filepath.FromSlash(rel)))
	if err != nil {
		return true // absent
	}
	return !bytes.Equal(got, want)
}

func exists(vault, rel string) bool {
	_, err := os.Stat(filepath.Join(vault, filepath.FromSlash(rel)))
	return err == nil
}

// Analyze runs phase 1 (eval only, change nothing).
func Analyze(vault string) Eval {
	var e Eval
	for _, f := range tier1Files {
		if differs(vault, f) {
			e.T1Change = append(e.T1Change, f)
		}
	}
	for _, f := range tier2Files {
		if !exists(vault, f) {
			e.T2New = append(e.T2New, f)
		} else if differs(vault, f) {
			e.T2Change = append(e.T2Change, f)
		}
	}
	return e
}

// printEval renders the phase-1 report.
func printEval(out io.Writer, vault string, e Eval) {
	fmt.Fprintf(out, "\nEval — planned changes for %s:\n", vault)
	if len(e.T1Change) > 0 {
		fmt.Fprintln(out, "  Tooling to refresh:")
		for _, f := range e.T1Change {
			fmt.Fprintf(out, "    ~ %s\n", f)
		}
	}
	if len(e.T2New) > 0 {
		fmt.Fprintln(out, "  Templates to add (new):")
		for _, f := range e.T2New {
			fmt.Fprintf(out, "    + %s\n", f)
		}
	}
	if len(e.T2Change) > 0 {
		fmt.Fprintln(out, "  Templates you changed (overwrite drops a .bak):")
		for _, f := range e.T2Change {
			cur, _ := os.ReadFile(filepath.Join(vault, filepath.FromSlash(f)))
			kit, _ := templateBytes(f)
			fmt.Fprintf(out, "    ! %-26s (%s)\n", f, delta(string(cur), string(kit)))
		}
	}
	fmt.Fprintln(out)
}

// Update runs the full eval-first flow. in/out drive the interactive prompt;
// hasTTY decides whether prompting is possible at all.
func Update(vault string, mode Mode, dryRun bool, in io.Reader, out io.Writer, hasTTY bool) (Result, error) {
	e := Analyze(vault)
	if e.Empty() {
		fmt.Fprintf(out, "==> vault already matches the kit — nothing to update.\n")
		return Result{AlreadyMatches: true}, nil
	}
	printEval(out, vault, e)

	if dryRun {
		fmt.Fprintln(out, "==> dry-run: no changes written.")
		return Result{DryRun: true}, nil
	}

	r := bufio.NewReader(in)
	action := decideAction(mode, e, vault, r, out, hasTTY)
	res := Result{Action: action}
	switch action {
	case "apply":
		res.Tool, res.New, res.Over = applyCopies(vault, e.T1Change), applyCopies(vault, e.T2New), applyOverwrite(vault, e.T2Change, out)
	case "safe":
		res.Tool, res.New = applyCopies(vault, e.T1Change), applyCopies(vault, e.T2New)
		res.Keep = len(e.T2Change)
	case "customize":
		res = customize(vault, e, r, out)
		res.Action = "customize"
	case "quit":
		fmt.Fprintln(out, "==> no changes made.")
	}
	return res, nil
}

func decideAction(mode Mode, e Eval, vault string, r *bufio.Reader, out io.Writer, hasTTY bool) string {
	switch mode {
	case ModeForce:
		return "apply"
	case ModeKeep:
		return "safe"
	default: // prompt
		if !hasTTY {
			fmt.Fprintln(out, "==> non-interactive, no flag — changing nothing. Re-run with --force (all) or --keep (tooling + new templates only).")
			return "quit"
		}
		return promptMenu(e, vault, r, out)
	}
}

func promptMenu(e Eval, vault string, r *bufio.Reader, out io.Writer) string {
	for {
		fmt.Fprint(out, "Apply? [a]ll / [s]afe (skip your changed templates) / [c]ustomize / [d]iff / [q]uit  (default: quit): ")
		line, err := r.ReadString('\n')
		if err != nil && line == "" {
			return "quit"
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "a":
			return "apply"
		case "s":
			return "safe"
		case "c":
			return "customize"
		case "", "q":
			return "quit"
		case "d":
			printDiffs(e, vault, out)
		default:
			fmt.Fprintln(out, "  pick a, s, c, d, or q.")
		}
	}
}

func printDiffs(e Eval, vault string, out io.Writer) {
	for _, f := range e.T2Change {
		fmt.Fprintf(out, "--- diff: %s (current vs kit) ---\n", f)
		cur, _ := os.ReadFile(filepath.Join(vault, filepath.FromSlash(f)))
		kit, _ := templateBytes(f)
		_, _, hunk := lineDiff(string(cur), string(kit))
		fmt.Fprint(out, hunk)
	}
}

// applyCopies copies each template into the vault, returning the success count.
func applyCopies(vault string, files []string) int {
	n := 0
	for _, f := range files {
		if copyTemplate(vault, f) == nil {
			n++
		}
	}
	return n
}

func applyOverwrite(vault string, files []string, out io.Writer) int {
	n := 0
	for _, f := range files {
		overwriteWithBak(vault, f, out)
		n++
	}
	return n
}

func overwriteWithBak(vault, rel string, out io.Writer) {
	dst := filepath.Join(vault, filepath.FromSlash(rel))
	if data, err := os.ReadFile(dst); err == nil {
		_ = os.WriteFile(dst+".bak", data, 0o644)
	}
	_ = copyTemplate(vault, rel)
	fmt.Fprintf(out, "    overwrote %s (backup: %s.bak)\n", rel, rel)
}

func customize(vault string, e Eval, r *bufio.Reader, out io.Writer) Result {
	ask := func(label string) bool {
		fmt.Fprintf(out, "  apply %s? [y/N]: ", label)
		line, _ := r.ReadString('\n')
		s := strings.ToLower(strings.TrimSpace(line))
		return s == "y"
	}
	var res Result
	for _, f := range e.T1Change {
		if ask(f + " (tooling)") {
			if copyTemplate(vault, f) == nil {
				res.Tool++
			}
		}
	}
	for _, f := range e.T2New {
		if ask(f + " (new)") {
			if copyTemplate(vault, f) == nil {
				res.New++
			}
		}
	}
	for _, f := range e.T2Change {
		if ask(f + " (overwrite, .bak)") {
			overwriteWithBak(vault, f, out)
			res.Over++
		} else {
			res.Keep++
		}
	}
	return res
}
