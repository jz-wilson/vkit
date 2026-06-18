// Package vaultpath resolves the vault root and holds the shared path-exclusion
// rules used by the validate and watcher packages so they all agree on what
// counts as a note.
package vaultpath

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Marker is the file whose presence identifies a directory as a vault root.
const Marker = "_format.md"

// DateFormat is the vault's canonical date format (the `updated:` field and the
// MOC header). Today returns the current date in it.
const DateFormat = "2006-01-02"

// Today returns the current date in the vault's canonical format.
func Today() string { return time.Now().Format(DateFormat) }

// excludedDirs are directory names skipped entirely when walking for notes.
var excludedDirs = map[string]bool{
	".git":     true,
	".claude":  true,
	"scripts":  true,
	"services": true,
	"archive":  true,
}

// skipNames are markdown files that are never treated as notes (generated or
// meta files).
var skipNames = map[string]bool{
	"MOC.md":     true, // legacy generated file; kept here so old vaults still exclude it
	"CLAUDE.md":  true,
	"AGENTS.md":  true,
	"_format.md": true,
}

// Resolve picks the vault root from, in order: an explicit arg, the --vault
// flag, $VKIT_VAULT, then the current working directory. The returned path is
// absolute. It does not require the path to exist (callers check separately).
func Resolve(arg, flag string) (string, error) {
	if arg != "" {
		return filepath.Abs(arg)
	}
	if flag != "" {
		return filepath.Abs(flag)
	}
	if env := os.Getenv("VKIT_VAULT"); env != "" {
		return filepath.Abs(env)
	}
	return os.Getwd()
}

// IsVault reports whether path looks like a vault root (has the marker).
func IsVault(path string) bool {
	_, err := os.Stat(filepath.Join(path, Marker))
	return err == nil
}

// IsExcludedDir reports whether a directory base name should be skipped during a
// note walk: the named tooling/meta dirs, or any dotdir.
func IsExcludedDir(base string) bool {
	if excludedDirs[base] {
		return true
	}
	return strings.HasPrefix(base, ".")
}

// IsNote reports whether a path (relative to the vault root, slash-separated) is
// a markdown note that belongs in the index. It applies the same dir, dotfile,
// and meta-file exclusions the bash kit used.
func IsNote(rel string) bool {
	rel = filepath.ToSlash(rel)
	if !strings.HasSuffix(rel, ".md") {
		return false
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == "" {
			continue
		}
		if strings.HasPrefix(seg, ".") {
			return false
		}
	}
	// any segment that is an excluded dir
	parts := strings.Split(rel, "/")
	for _, seg := range parts[:len(parts)-1] {
		if excludedDirs[seg] {
			return false
		}
	}
	base := parts[len(parts)-1]
	if skipNames[base] {
		return false
	}
	return true
}

// ClassifyOpts configures optional filtering on top of the base IsNote check.
type ClassifyOpts struct {
	// SkipREADME causes README.md files to be excluded even when they would
	// otherwise pass IsNote (e.g. validate skips them; moc does not).
	SkipREADME bool
}

// ClassifyFile returns true when path is a note that should be processed under
// the given options. It returns false if IsNote returns false, or if
// opts.SkipREADME is true and the basename is "README.md".
func ClassifyFile(path string, opts ClassifyOpts) bool {
	if !IsNote(path) {
		return false
	}
	if opts.SkipREADME && filepath.Base(filepath.ToSlash(path)) == "README.md" {
		return false
	}
	return true
}

// WalkNotes is the canonical vault note walk: it descends from the vault root,
// skips excluded/dot dirs, and calls fn once per note-eligible file with its
// slash-separated relative path. moc, validate, and any other note sweep share
// this so they cannot drift on what counts as a note.
func WalkNotes(vault string, fn func(rel string) error) error {
	return filepath.WalkDir(vault, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == vault {
			return nil
		}
		if d.IsDir() {
			if IsExcludedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(vault, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if IsNote(rel) {
			return fn(rel)
		}
		return nil
	})
}
