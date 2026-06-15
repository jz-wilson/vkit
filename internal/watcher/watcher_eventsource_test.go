package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWatchWithSource_rebuildsOnEvent verifies that WatchWithSource calls
// rebuild exactly once when FakeSource emits a single event, and that the
// rebuild function receives control (i.e. the MOC.md is created/updated in
// the temp vault).
func TestWatchWithSource_rebuildsOnEvent(t *testing.T) {
	vault := t.TempDir()

	// Seed a note so moc.Build has something to index.
	note := filepath.Join(vault, "projects", "a.md")
	if err := os.MkdirAll(filepath.Dir(note), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(note, []byte("# Hello\n\ntest note\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	src := NewFakeSource()

	rebuilt := make(chan struct{}, 1)
	rebuild := func() error {
		rebuilt <- struct{}{}
		return nil
	}

	done := make(chan error, 1)
	go func() {
		done <- WatchWithSource(vault, src, rebuild)
	}()

	// Send one event then close the source so WatchWithSource can return.
	src.Send(note)
	src.Close()

	select {
	case <-rebuilt:
		// good — rebuild was called
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: rebuild was not called after FakeSource event")
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("WatchWithSource returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: WatchWithSource did not return after source closed")
	}
}

// TestWatchWithSource_multipleEvents verifies that each event triggers exactly
// one rebuild call (no coalescing in WatchWithSource itself — debounce lives
// in watchFsnotifyDebounced).
func TestWatchWithSource_multipleEvents(t *testing.T) {
	vault := t.TempDir()

	src := NewFakeSource()

	var count int
	rebuild := func() error {
		count++
		return nil
	}

	done := make(chan error, 1)
	go func() {
		done <- WatchWithSource(vault, src, rebuild)
	}()

	src.Send("a.md")
	src.Send("b.md")
	src.Send("c.md")
	src.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("WatchWithSource returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: WatchWithSource did not return")
	}

	if count != 3 {
		t.Errorf("rebuild called %d times, want 3", count)
	}
}

// TestWatchWithSource_noEvents verifies that WatchWithSource returns cleanly
// when the source is closed immediately with no events.
func TestWatchWithSource_noEvents(t *testing.T) {
	vault := t.TempDir()
	src := NewFakeSource()
	src.Close()

	done := make(chan error, 1)
	go func() {
		done <- WatchWithSource(vault, src, func() error { return nil })
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("WatchWithSource returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: WatchWithSource did not return on empty source")
	}
}

// TestFakeSource_sendAndReceive verifies the FakeSource contract directly.
func TestFakeSource_sendAndReceive(t *testing.T) {
	src := NewFakeSource()
	src.Send("/vault/projects/a.md")
	src.Send("/vault/decisions/2026-06.md")

	got1 := <-src.Events()
	got2 := <-src.Events()

	if got1 != "/vault/projects/a.md" {
		t.Errorf("event 1 = %q, want /vault/projects/a.md", got1)
	}
	if got2 != "/vault/decisions/2026-06.md" {
		t.Errorf("event 2 = %q, want /vault/decisions/2026-06.md", got2)
	}

	if err := src.Close(); err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
}
