package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DefaultContentDirs is the canonical default slice shared by all callers.
var DefaultContentDirs = []string{"decisions", "infrastructure", "projects", "reference", "archive"}

type vkitSection struct {
	ContentDirs []string `json:"contentDirs"`
}

type settingsFile struct {
	Vkit vkitSection `json:"vkit"`
}

// ContentDirs returns the content directory list for the given vault.
// It reads .claude/settings.json at the vault root and returns the
// vkit.contentDirs value if present and non-empty; otherwise DefaultContentDirs.
// Malformed JSON or a missing file silently returns the default.
func ContentDirs(vault string) []string {
	data, err := os.ReadFile(filepath.Join(vault, ".claude", "settings.json"))
	if err != nil {
		return DefaultContentDirs
	}
	var s settingsFile
	if err := json.Unmarshal(data, &s); err != nil {
		return DefaultContentDirs
	}
	if len(s.Vkit.ContentDirs) == 0 {
		return DefaultContentDirs
	}
	return s.Vkit.ContentDirs
}
