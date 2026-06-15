// Package osdetect ports lib/detect-os.sh: it classifies the OS, finds the
// system package manager, probes for a systemd user instance, and reports
// whether Obsidian native mode has been explicitly opted into. Obsidian is
// never auto-probed (that can launch the GUI) — it is opt-in only.
package osdetect

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// lookPath is indirected for testing.
var lookPath = exec.LookPath

// pkgMgrOrder mirrors the bash probe order exactly.
var pkgMgrOrder = []string{"brew", "apt-get", "dnf", "pacman", "zypper", "winget", "choco", "scoop"}

// Info is a snapshot of the host environment for `vkit doctor`.
type Info struct {
	OS             string // macos | linux | wsl | windows | unknown
	PkgMgr         string // one of pkgMgrOrder, or "none"
	SystemdUser    bool
	HasTTY         bool
	ObsidianCLI    bool // opt-in enabled (marker/env-var present)
	ObsidianBinary bool // `obsidian` binary found on PATH
}

// DetectOS returns one of macos/linux/wsl/windows/unknown. On Linux it reads
// /proc/version to distinguish WSL.
func DetectOS() string {
	var proc string
	if b, err := os.ReadFile("/proc/version"); err == nil {
		proc = string(b)
	}
	return detectOSFrom(runtime.GOOS, proc)
}

// detectOSFrom is the pure core, split out for testing.
func detectOSFrom(goos, procVersion string) string {
	switch goos {
	case "darwin":
		return "macos"
	case "linux":
		if isWSL(procVersion) {
			return "wsl"
		}
		return "linux"
	case "windows":
		return "windows"
	default:
		return "unknown"
	}
}

func isWSL(procVersion string) bool {
	p := strings.ToLower(procVersion)
	return strings.Contains(p, "microsoft") || strings.Contains(p, "wsl")
}

// DetectPkgMgr returns the first available package manager, or "none".
func DetectPkgMgr() string {
	return detectPkgMgr(lookPath)
}

func detectPkgMgr(look func(string) (string, error)) string {
	for _, m := range pkgMgrOrder {
		if _, err := look(m); err == nil {
			return m
		}
	}
	return "none"
}

// HasSystemdUser reports whether a real systemd user instance is reachable.
func HasSystemdUser() bool {
	if _, err := lookPath("systemctl"); err != nil {
		return false
	}
	cmd := exec.Command("systemctl", "--user", "show-environment")
	return cmd.Run() == nil
}

// HasTTY reports whether /dev/tty can actually be opened (it may be absent under
// CI / cron / nohup even when a controlling terminal nominally exists).
func HasTTY() bool {
	f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

// ObsidianEnabled reports whether native Obsidian mode is active. It enables
// automatically when the obsidian binary is on PATH; set VAULT_OBSIDIAN_CLI=0
// to disable explicitly.
func ObsidianEnabled(vault string) bool {
	if os.Getenv("VAULT_OBSIDIAN_CLI") == "0" {
		return false
	}
	return obsidianBinaryFound(lookPath)
}

func obsidianBinaryFound(look func(string) (string, error)) bool {
	_, err := look("obsidian")
	return err == nil
}

// Detect builds a full Info snapshot.
func Detect(vault string) Info {
	binary := obsidianBinaryFound(lookPath)
	return Info{
		OS:             DetectOS(),
		PkgMgr:         DetectPkgMgr(),
		SystemdUser:    HasSystemdUser(),
		HasTTY:         HasTTY(),
		ObsidianCLI:    binary && os.Getenv("VAULT_OBSIDIAN_CLI") != "0",
		ObsidianBinary: binary,
	}
}
