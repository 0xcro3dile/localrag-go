package filewatcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xcro3dile/localrag-go/internal/domain/ports"
)

func TestFSNotifyWatcher_Creation(t *testing.T) {
	watcher, err := NewFSNotifyWatcher([]string{".txt", ".pdf"})
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer watcher.Stop()
}

func TestFSNotifyWatcher_DefaultExtensions(t *testing.T) {
	watcher, _ := NewFSNotifyWatcher(nil)
	defer watcher.Stop()

	if len(watcher.extensions) != 3 {
		t.Errorf("expected 3 default extensions, got %d", len(watcher.extensions))
	}
}

func TestFSNotifyWatcher_WatchDirectory(t *testing.T) {
	dir, _ := os.MkdirTemp("", "watcher-test-*")
	defer os.RemoveAll(dir)

	watcher, _ := NewFSNotifyWatcher([]string{".txt"})
	defer watcher.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events, err := watcher.Watch(ctx, dir)
	if err != nil {
		t.Fatalf("watch failed: %v", err)
	}

	// Create a file
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hi"), 0644)
	}()

	select {
	case event := <-events:
		if event.Operation != ports.FileCreated {
			t.Errorf("expected create event, got %v", event.Operation)
		}
	case <-ctx.Done():
		t.Error("timeout waiting for event")
	}
}

func TestFSNotifyWatcher_FiltersByExtension(t *testing.T) {
	dir, _ := os.MkdirTemp("", "watcher-test-*")
	defer os.RemoveAll(dir)

	watcher, _ := NewFSNotifyWatcher([]string{".txt"})
	defer watcher.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	events, _ := watcher.Watch(ctx, dir)

	// Create non-matching file
	os.WriteFile(filepath.Join(dir, "test.json"), []byte("{}"), 0644)

	select {
	case <-events:
		t.Error("should not receive event for .json")
	case <-time.After(300 * time.Millisecond):
		// Expected - no event
	}
}

func TestFSNotifyWatcher_Stop(t *testing.T) {
	watcher, _ := NewFSNotifyWatcher(nil)
	err := watcher.Stop()
	if err != nil {
		t.Errorf("stop failed: %v", err)
	}
}
