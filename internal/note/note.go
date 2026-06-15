// Package note ports the /note command's Tier B (portable) scaffold and a thin
// Tier A (native Obsidian) shell. Tier B writes schema-valid frontmatter and the
// standard body skeleton; Tier A drives the official `obsidian` CLI.
package note

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ensureMD appends ".md" to relPath when it has no .md extension, so callers
// can pass bare stems ("projects/alpha") or full names ("projects/alpha.md").
// EnsureMD appends ".md" to relPath when it has no .md extension.
func EnsureMD(relPath string) string {
	if strings.HasSuffix(relPath, ".md") {
		return relPath
	}
	return relPath + ".md"
}

func ensureMD(relPath string) string { return EnsureMD(relPath) }

// Create scaffolds a note at relPath (relative to vault). It refuses to
// overwrite an existing file. title, if empty, is derived from the filename
// (kebab -> Title Case). tags may be nil.
func Create(vault, relPath, title string, tags []string, today string) error {
	relPath = ensureMD(relPath)
	if title == "" {
		title = titleFromFilename(relPath)
	}
	full := filepath.Join(vault, relPath)
	if _, err := os.Stat(full); err == nil {
		return fmt.Errorf("%s already exists — refusing to overwrite", relPath)
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(render(title, tags, today)), 0o644)
}

func render(title string, tags []string, today string) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "updated: %s\n", today)
	if len(tags) > 0 {
		fmt.Fprintf(&b, "tags: [%s]\n", strings.Join(tags, ", "))
	}
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "# %s\n\n%s\n", title, bodySkeleton)
	return b.String()
}

// bodySkeleton is the section scaffold shared by the portable and native paths.
const bodySkeleton = "## Summary\n\n## Notes\n\n## Related"

// titleFromFilename turns "projects/my-cool-note.md" into "My Cool Note".
func titleFromFilename(relPath string) string {
	base := filepath.Base(relPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.ReplaceAll(base, "_", "-")
	parts := strings.Split(base, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// Creator is the seam between the note package and its callers.
// Use New() to obtain the appropriate implementation for the current host.
type Creator interface {
	Create(vault, relPath, title string, tags []string, today string) error
}

type portableCreator struct{}

func (portableCreator) Create(vault, relPath, title string, tags []string, today string) error {
	return Create(vault, relPath, title, tags, today)
}

type nativeCreator struct{}

func (nativeCreator) Create(vault, relPath, title string, tags []string, today string) error {
	return CreateNative(vault, relPath, title, tags, today)
}

// New returns the appropriate Creator for the current host.
// nativeCreator is returned when the obsidian binary is on PATH and
// VAULT_OBSIDIAN_CLI is not "0"; otherwise portableCreator is returned.
func New(vault string) Creator {
	if os.Getenv("VAULT_OBSIDIAN_CLI") == "0" {
		return portableCreator{}
	}
	if _, err := exec.LookPath("obsidian"); err == nil {
		return nativeCreator{}
	}
	return portableCreator{}
}

type execFunc func(vault string, args ...string) error

// createNative is the testable core of CreateNative — accepts an exec hook so
// tests can inject a fake without spawning a real obsidian process.
func createNative(vault, relPath, title string, tags []string, today string, exec execFunc) error {
	relPath = ensureMD(relPath)
	if title == "" {
		title = titleFromFilename(relPath)
	}
	content := fmt.Sprintf("# %s\n\n%s", title, bodySkeleton)
	if err := exec(vault, "create", "path="+relPath, "content="+content); err != nil {
		return err
	}
	if err := exec(vault, "property:set", "name=updated", "value="+today, "path="+relPath); err != nil {
		return err
	}
	if len(tags) > 0 {
		if err := exec(vault, "property:set", "name=tags", "type=list", "value="+strings.Join(tags, ","), "path="+relPath); err != nil {
			return err
		}
	}
	return nil
}

// CreateNative drives the official obsidian CLI (Tier A). It is only called when
// native mode is opted into.
func CreateNative(vault, relPath, title string, tags []string, today string) error {
	return createNative(vault, relPath, title, tags, today, runObsidian)
}

func runObsidian(vault string, args ...string) error {
	cmd := exec.Command("obsidian", args...)
	cmd.Dir = vault
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
