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
func ensureMD(relPath string) string {
	if strings.HasSuffix(relPath, ".md") {
		return relPath
	}
	return relPath + ".md"
}

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

// CreateNative drives the official obsidian CLI (Tier A). It is only called when
// native mode is opted into.
func CreateNative(vault, relPath, title string, tags []string, today string) error {
	relPath = ensureMD(relPath)
	if title == "" {
		title = titleFromFilename(relPath)
	}
	content := fmt.Sprintf("# %s\n\n%s", title, bodySkeleton)
	if err := runObsidian(vault, "create", "path="+relPath, "content="+content); err != nil {
		return err
	}
	if err := runObsidian(vault, "property:set", "name=updated", "value="+today, "path="+relPath); err != nil {
		return err
	}
	if len(tags) > 0 {
		if err := runObsidian(vault, "property:set", "name=tags", "type=list", "value="+strings.Join(tags, ","), "path="+relPath); err != nil {
			return err
		}
	}
	return nil
}

func runObsidian(vault string, args ...string) error {
	cmd := exec.Command("obsidian", args...)
	cmd.Dir = vault
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
