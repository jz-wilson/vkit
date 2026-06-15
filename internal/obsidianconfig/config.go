// Package obsidianconfig reads vault-level Obsidian settings from
// .obsidian/app.json so Tier B commands can respect the user's vault
// configuration without requiring Obsidian to be running.
package obsidianconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppConfig holds the subset of .obsidian/app.json fields that affect Tier B
// note creation. Unknown fields are silently discarded.
type AppConfig struct {
	NewFileLocation   string `json:"newFileLocation"`   // "root" | "current" | "folder"
	NewFileFolderPath string `json:"newFileFolderPath"` // target folder when location == "folder"
}

// Read parses <vault>/.obsidian/app.json and returns the relevant settings.
// Any read or parse error returns a zero AppConfig so callers degrade
// gracefully when the file is absent or malformed.
func Read(vault string) AppConfig {
	b, err := os.ReadFile(filepath.Join(vault, ".obsidian", "app.json"))
	if err != nil {
		return AppConfig{}
	}
	var cfg AppConfig
	_ = json.Unmarshal(b, &cfg)
	return cfg
}

// DefaultNoteFolder returns the configured default folder for new notes, or ""
// when the location is "root", "current", or unconfigured. "current" (the
// folder open in the Obsidian UI) has no CLI equivalent and falls back to root.
func (c AppConfig) DefaultNoteFolder() string {
	if c.NewFileLocation == "folder" && c.NewFileFolderPath != "" {
		return c.NewFileFolderPath
	}
	return ""
}
