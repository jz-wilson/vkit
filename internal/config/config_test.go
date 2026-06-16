package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeSettings(t *testing.T, dir string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestContentDirs_default_when_no_file(t *testing.T) {
	vault := t.TempDir()
	got := ContentDirs(vault)
	if !reflect.DeepEqual(got, DefaultContentDirs) {
		t.Errorf("got %v, want %v", got, DefaultContentDirs)
	}
}

func TestContentDirs_custom_key(t *testing.T) {
	vault := t.TempDir()
	writeSettings(t, vault, map[string]any{
		"vkit": map[string]any{
			"contentDirs": []string{"foo", "bar"},
		},
	})
	got := ContentDirs(vault)
	want := []string{"foo", "bar"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestContentDirs_absent_key_returns_defaults(t *testing.T) {
	vault := t.TempDir()
	writeSettings(t, vault, map[string]any{"permissions": map[string]any{}})
	got := ContentDirs(vault)
	if !reflect.DeepEqual(got, DefaultContentDirs) {
		t.Errorf("got %v, want %v", got, DefaultContentDirs)
	}
}

func TestContentDirs_malformed_json_returns_defaults(t *testing.T) {
	vault := t.TempDir()
	p := filepath.Join(vault, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{bad json}"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ContentDirs(vault)
	if !reflect.DeepEqual(got, DefaultContentDirs) {
		t.Errorf("got %v, want %v", got, DefaultContentDirs)
	}
}
