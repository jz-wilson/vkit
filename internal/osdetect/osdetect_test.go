package osdetect

import (
	"errors"
	"testing"
)

func TestObsidianBinaryFound(t *testing.T) {
	present := func(name string) (string, error) {
		if name == "obsidian" {
			return "/usr/local/bin/obsidian", nil
		}
		return "", errors.New("not found")
	}
	absent := func(string) (string, error) { return "", errors.New("not found") }

	if !obsidianBinaryFound(present) {
		t.Error("expected binary found when obsidian on PATH")
	}
	if obsidianBinaryFound(absent) {
		t.Error("expected binary not found when obsidian absent")
	}
}

func TestDetectOSFrom(t *testing.T) {
	cases := []struct {
		goos, proc, want string
	}{
		{"darwin", "", "macos"},
		{"linux", "Linux version 5.15.0-generic", "linux"},
		{"linux", "Linux version 5.15.0-microsoft-standard-WSL2", "wsl"},
		{"linux", "Linux version 6.1.0 (wsl build)", "wsl"},
		{"windows", "", "windows"},
		{"plan9", "", "unknown"},
	}
	for _, c := range cases {
		if got := detectOSFrom(c.goos, c.proc); got != c.want {
			t.Errorf("detectOSFrom(%q,%q)=%q want %q", c.goos, c.proc, got, c.want)
		}
	}
}

func TestDetectPkgMgrOrdering(t *testing.T) {
	// Only dnf and apt-get "present"; apt-get ranks first in the probe order.
	present := map[string]bool{"apt-get": true, "dnf": true}
	look := func(name string) (string, error) {
		if present[name] {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
	if got := detectPkgMgr(look); got != "apt-get" {
		t.Errorf("got %q want apt-get", got)
	}

	// brew beats everything.
	present2 := map[string]bool{"brew": true, "apt-get": true}
	look2 := func(name string) (string, error) {
		if present2[name] {
			return "/x/" + name, nil
		}
		return "", errors.New("nope")
	}
	if got := detectPkgMgr(look2); got != "brew" {
		t.Errorf("got %q want brew", got)
	}

	// none present.
	none := func(string) (string, error) { return "", errors.New("nope") }
	if got := detectPkgMgr(none); got != "none" {
		t.Errorf("got %q want none", got)
	}
}
