// Package filewatcher provides file system monitoring adapters.
// Clean Architecture: Adapter implementing ports.FileWatcher.
package filewatcher

import (
	"context"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/0xcro3dile/localrag-go/internal/domain/ports"
)

// FSNotifyWatcher implements ports.FileWatcher using fsnotify.
type FSNotifyWatcher struct {
	watcher    *fsnotify.Watcher
	extensions []string // File extensions to watch (e.g., ".pdf", ".txt")
}

// NewFSNotifyWatcher creates a new file watcher.
func NewFSNotifyWatcher(extensions []string) (*FSNotifyWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if len(extensions) == 0 {
		extensions = []string{".pdf", ".txt", ".md"}
	}

	return &FSNotifyWatcher{
		watcher:    w,
		extensions: extensions,
	}, nil
}

// Watch starts monitoring the directory and emits events.
func (w *FSNotifyWatcher) Watch(ctx context.Context, dir string) (<-chan ports.FileEvent, error) {
	if err := w.watcher.Add(dir); err != nil {
		return nil, err
	}

	events := make(chan ports.FileEvent, 100)

	go func() {
		defer close(events)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				// Filter by extension
				if !w.isWatchedExtension(event.Name) {
					continue
				}

				var op ports.FileOperation
				switch {
				case event.Op&fsnotify.Create == fsnotify.Create:
					op = ports.FileCreated
				case event.Op&fsnotify.Write == fsnotify.Write:
					op = ports.FileModified
				case event.Op&fsnotify.Remove == fsnotify.Remove:
					op = ports.FileDeleted
				default:
					continue
				}

				select {
				case events <- ports.FileEvent{Path: event.Name, Operation: op}:
				case <-ctx.Done():
					return
				}
			case _, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				// Log errors in production
			}
		}
	}()

	return events, nil
}

// Stop stops the watcher.
func (w *FSNotifyWatcher) Stop() error {
	return w.watcher.Close()
}

// isWatchedExtension checks if the file has a watched extension.
func (w *FSNotifyWatcher) isWatchedExtension(path string) bool {
	ext := filepath.Ext(path)
	for _, e := range w.extensions {
		if ext == e {
			return true
		}
	}
	return false
}
