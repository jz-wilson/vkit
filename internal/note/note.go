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

	"github.com/jz-wilson/vkit/internal/obsidianconfig"
)

// EnsureMD appends ".md" to relPath when it has no .md extension.
func EnsureMD(relPath string) string {
	if strings.HasSuffix(relPath, ".md") {
		return relPath
	}
	return relPath + ".md"
}

// DeriveTitle turns "projects/my-cool-note.md" into "My Cool Note".
func DeriveTitle(relPath string) string {
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

// Create scaffolds a note at relPath (relative to vault). It refuses to
// overwrite an existing file. title, if empty, is derived from the filename
// (kebab -> Title Case). tags may be nil.
func Create(vault, relPath, title string, tags []string, today string) error {
	relPath = EnsureMD(relPath)
	if title == "" {
		title = DeriveTitle(relPath)
	}
	full := filepath.Join(vault, relPath)
	if _, err := os.Stat(full); err == nil {
		return fmt.Errorf("%s already exists — refusing to overwrite", relPath)
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "updated: %s\n", today)
	if len(tags) > 0 {
		fmt.Fprintf(&b, "tags: [%s]\n", strings.Join(tags, ", "))
	}
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "# %s\n\n%s\n", title, bodySkeleton)
	return os.WriteFile(full, []byte(b.String()), 0o644)
}

// bodySkeleton is the section scaffold shared by the portable and native paths.
const bodySkeleton = "## Summary\n\n## Notes\n\n## Related"

// Creator is the seam between the note package and its callers.
// Use New() to obtain the appropriate implementation for the current host.
// Create returns the resolved vault-relative path (with .md and any folder
// routing applied) so callers can display exactly where the note landed.
type Creator interface {
	Create(vault, relPath, title string, tags []string, today string) (string, error)
}

type portableCreator struct{}

func (portableCreator) Create(vault, relPath, title string, tags []string, today string) (string, error) {
	// Apply the vault's configured default note folder when the caller gave a
	// bare filename with no directory component. "current" has no CLI meaning
	// so it falls back to root (empty folder = no prefix).
	if filepath.Dir(relPath) == "." {
		if folder := obsidianconfig.Read(vault).DefaultNoteFolder(); folder != "" {
			relPath = folder + "/" + relPath
		}
	}
	return EnsureMD(relPath), Create(vault, relPath, title, tags, today)
}

type nativeCreator struct{}

func (nativeCreator) Create(vault, relPath, title string, tags []string, today string) (string, error) {
	return EnsureMD(relPath), CreateNative(vault, relPath, title, tags, today)
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
	relPath = EnsureMD(relPath)
	if title == "" {
		title = DeriveTitle(relPath)
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
