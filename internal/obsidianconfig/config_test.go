package obsidianconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func writeAppJSON(t *testing.T, vault, content string) {
	t.Helper()
	dir := filepath.Join(vault, ".obsidian")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRead_folderMode(t *testing.T) {
	v := t.TempDir()
	writeAppJSON(t, v, `{"newFileLocation":"folder","newFileFolderPath":"inbox"}`)
	cfg := Read(v)
	if cfg.NewFileLocation != "folder" {
		t.Errorf("NewFileLocation = %q, want folder", cfg.NewFileLocation)
	}
	if cfg.NewFileFolderPath != "inbox" {
		t.Errorf("NewFileFolderPath = %q, want inbox", cfg.NewFileFolderPath)
	}
}

func TestRead_missingFile(t *testing.T) {
	v := t.TempDir()
	cfg := Read(v)
	if cfg.NewFileLocation != "" || cfg.NewFileFolderPath != "" {
		t.Errorf("expected zero config for missing file, got %+v", cfg)
	}
}

func TestRead_malformedJSON(t *testing.T) {
	v := t.TempDir()
	writeAppJSON(t, v, `{not valid json`)
	cfg := Read(v)
	if cfg.NewFileLocation != "" {
		t.Errorf("expected zero config for malformed JSON, got %+v", cfg)
	}
}

func TestRead_unknownFieldsIgnored(t *testing.T) {
	v := t.TempDir()
	writeAppJSON(t, v, `{"newFileLocation":"root","unknownKey":"value","alsoUnknown":42}`)
	cfg := Read(v)
	if cfg.NewFileLocation != "root" {
		t.Errorf("NewFileLocation = %q, want root", cfg.NewFileLocation)
	}
}

func TestDefaultNoteFolder(t *testing.T) {
	cases := []struct {
		name string
		cfg  AppConfig
		want string
	}{
		{"folder mode with path", AppConfig{NewFileLocation: "folder", NewFileFolderPath: "inbox"}, "inbox"},
		{"folder mode nested", AppConfig{NewFileLocation: "folder", NewFileFolderPath: "notes/inbox"}, "notes/inbox"},
		{"folder mode empty path", AppConfig{NewFileLocation: "folder", NewFileFolderPath: ""}, ""},
		{"root mode", AppConfig{NewFileLocation: "root"}, ""},
		{"current mode (no CLI meaning)", AppConfig{NewFileLocation: "current"}, ""},
		{"zero config", AppConfig{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.cfg.DefaultNoteFolder()
			if got != tc.want {
				t.Errorf("DefaultNoteFolder() = %q, want %q", got, tc.want)
			}
		})
	}
}
