// Package watcher ports watch.sh: it rebuilds MOC.md whenever a note changes,
// using an fsnotify backend (recursive — new dirs are added as they appear) with
// a zero-dependency mtime-polling fallback.
package watcher

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"vkit/internal/moc"
	"vkit/internal/vaultpath"

	"github.com/fsnotify/fsnotify"
)

// ignoreRe matches paths that must not trigger a rebuild — dotpaths, the
// scripts/ and services/ dirs, MOC.md itself, and editor temp/swap files. This
// mirrors watch.sh's IGNORE regex.
var ignoreRe = regexp.MustCompile(`(/\.|/scripts/|/services/|MOC\.md|\.tmp$|\.swp$|~$)`)

func ignored(path string) bool {
	return ignoreRe.MatchString(filepath.ToSlash(path))
}

// Watch blocks, rebuilding the MOC on changes until the process is killed. If
// poll is true, or fsnotify cannot start, it uses the polling backend.
func Watch(vault string, poll bool, interval time.Duration) error {
	if !poll {
		if err := watchFsnotify(vault); err != nil {
			fmt.Fprintf(os.Stderr, "watch: fsnotify unavailable (%v) — falling back to polling\n", err)
			return watchPoll(vault, interval)
		}
		return nil
	}
	return watchPoll(vault, interval)
}

func watchFsnotify(vault string) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	addDir := func(dir string) {
		_ = w.Add(dir)
	}
	// Add every non-ignored dir under the vault.
	err = filepath.WalkDir(vault, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != vault && ignored(path+"/") {
				return filepath.SkipDir
			}
			addDir(path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Println("watch: fsnotify backend")
	var mu sync.Mutex
	var timer *time.Timer
	rebuild := func() {
		mu.Lock()
		defer mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(300*time.Millisecond, func() {
			if _, err := moc.Build(vault, vaultpath.Today()); err != nil {
				fmt.Fprintf(os.Stderr, "watch: rebuild failed: %v\n", err)
			}
		})
	}

	for {
		select {
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			// Newly-created dirs must be added so the watch stays recursive.
			if ev.Op&fsnotify.Create != 0 {
				if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() && !ignored(ev.Name+"/") {
					addDir(ev.Name)
				}
			}
			if !ignored(ev.Name) {
				rebuild()
			}
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "watch: %v\n", err)
		}
	}
}

func watchPoll(vault string, interval time.Duration) error {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	fmt.Printf("watch: polling backend (every %s)\n", interval)
	last := newestNote(vault)
	for {
		time.Sleep(interval)
		n := newestNote(vault)
		if n.After(last) {
			if _, err := moc.Build(vault, vaultpath.Today()); err != nil {
				fmt.Fprintf(os.Stderr, "watch: rebuild failed: %v\n", err)
			}
			last = n
		}
	}
}

// newestNote returns the most recent mtime among non-ignored .md files.
func newestNote(vault string) time.Time {
	var newest time.Time
	_ = filepath.WalkDir(vault, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != vault && ignored(path+"/") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" || ignored(path) {
			return nil
		}
		if fi, err := d.Info(); err == nil && fi.ModTime().After(newest) {
			newest = fi.ModTime()
		}
		return nil
	})
	return newest
}
