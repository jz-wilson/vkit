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
	"sync/atomic"
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

// EventSource is the seam that lets tests inject a synchronous fake instead of
// a real filesystem watcher or a polling loop.
type EventSource interface {
	// Events returns a channel that emits the path of each changed file.
	Events() <-chan string
	Close() error
}

// FakeSource is a synchronous EventSource for use in tests. Callers drive
// events via Send; Close closes the channel (idempotent — safe to call multiple times).
type FakeSource struct {
	ch     chan string
	closed atomic.Bool
}

// NewFakeSource returns a FakeSource with a buffered channel.
func NewFakeSource() *FakeSource {
	return &FakeSource{ch: make(chan string, 16)}
}

// Send enqueues path as a changed-file event.
func (f *FakeSource) Send(path string) {
	f.ch <- path
}

// Events implements EventSource.
func (f *FakeSource) Events() <-chan string {
	return f.ch
}

// Close implements EventSource. Safe to call multiple times.
func (f *FakeSource) Close() error {
	if f.closed.CompareAndSwap(false, true) {
		close(f.ch)
	}
	return nil
}

// FsnotifySource implements EventSource using fsnotify.
type FsnotifySource struct {
	vault string
	w     *fsnotify.Watcher
	ch    chan string
	done  chan struct{}
}

// NewFsnotifySource creates an FsnotifySource that watches all non-ignored
// directories under vault. Returns an error if fsnotify cannot be initialised.
func NewFsnotifySource(vault string) (*FsnotifySource, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	addDir := func(dir string) {
		_ = w.Add(dir)
	}
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
		w.Close()
		return nil, err
	}

	src := &FsnotifySource{
		vault: vault,
		w:     w,
		ch:    make(chan string, 64),
		done:  make(chan struct{}),
	}

	go src.pump(addDir)
	return src, nil
}

func (s *FsnotifySource) pump(addDir func(string)) {
	defer close(s.ch)
	for {
		select {
		case <-s.done:
			return
		case ev, ok := <-s.w.Events:
			if !ok {
				return
			}
			// Newly-created dirs must be added so the watch stays recursive.
			if ev.Op&fsnotify.Create != 0 {
				if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() && !ignored(ev.Name+"/") {
					addDir(ev.Name)
				}
			}
			if !ignored(ev.Name) {
				select {
				case s.ch <- ev.Name:
				default:
				}
			}
		case err, ok := <-s.w.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "watch: %v\n", err)
		}
	}
}

// Events implements EventSource.
func (s *FsnotifySource) Events() <-chan string {
	return s.ch
}

// Close implements EventSource.
func (s *FsnotifySource) Close() error {
	close(s.done)
	return s.w.Close()
}

// PollSource implements EventSource using mtime polling.
type PollSource struct {
	vault    string
	interval time.Duration
	ch       chan string
	stop     chan struct{}
}

// NewPollSource creates a PollSource that emits the vault path whenever a newer
// note is detected. A non-positive interval defaults to 5 s.
func NewPollSource(vault string, interval time.Duration) *PollSource {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	src := &PollSource{
		vault:    vault,
		interval: interval,
		ch:       make(chan string, 4),
		stop:     make(chan struct{}),
	}
	go src.poll()
	return src
}

func (s *PollSource) poll() {
	defer close(s.ch)
	last := newestNote(s.vault)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			n := newestNote(s.vault)
			if n.After(last) {
				last = n
				select {
				case s.ch <- s.vault:
				default:
				}
			}
		}
	}
}

// Events implements EventSource.
func (s *PollSource) Events() <-chan string {
	return s.ch
}

// Close implements EventSource.
func (s *PollSource) Close() error {
	close(s.stop)
	return nil
}

// WatchWithSource blocks, rebuilding via rebuild() whenever src emits an event,
// until src's Events channel is closed. The caller retains ownership of src
// and is responsible for closing it. This is the primary entry point for tests
// and any caller that wants to supply its own EventSource.
func WatchWithSource(vault string, src EventSource, rebuild func() error) error {
	for range src.Events() {
		if err := rebuild(); err != nil {
			fmt.Fprintf(os.Stderr, "watch: rebuild failed: %v\n", err)
		}
	}
	return nil
}

// Watch blocks, rebuilding the MOC on changes until the process is killed.
// If poll is true, or fsnotify cannot start, the polling backend is used.
// This preserves the original public signature.
func Watch(vault string, poll bool, interval time.Duration) error {
	defaultRebuild := func() error {
		_, err := moc.Build(vault, vaultpath.Today())
		return err
	}

	if !poll {
		src, err := NewFsnotifySource(vault)
		if err != nil {
			fmt.Fprintf(os.Stderr, "watch: fsnotify unavailable (%v) — falling back to polling\n", err)
			poll = true
		} else {
			fmt.Println("watch: fsnotify backend")
			return watchFsnotifyDebounced(src, defaultRebuild)
		}
	}

	fmt.Printf("watch: polling backend (every %s)\n", interval)
	src := NewPollSource(vault, interval)
	return WatchWithSource(vault, src, defaultRebuild)
}

// watchFsnotifyDebounced wraps WatchWithSource with a 300 ms debounce that
// coalesces rapid bursts of events into a single rebuild call.
func watchFsnotifyDebounced(src EventSource, rebuild func() error) error {
	defer src.Close()
	var mu sync.Mutex
	var timer *time.Timer
	for range src.Events() {
		mu.Lock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(300*time.Millisecond, func() {
			if err := rebuild(); err != nil {
				fmt.Fprintf(os.Stderr, "watch: rebuild failed: %v\n", err)
			}
		})
		mu.Unlock()
	}
	return nil
}

// newestNote returns the mtime of the most recently modified non-ignored .md
// file under vault, or the zero time if no qualifying file is found.
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
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	return newest
}
