package behavioral

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileOp represents the type of file operation
type FileOp int

const (
	// FileCreated indicates a new file was created
	FileCreated FileOp = iota
	// FileWritten indicates a file was written to
	FileWritten
	// FileRemoved indicates a file was removed
	FileRemoved
)

// String returns a human-readable representation of the file operation
func (op FileOp) String() string {
	switch op {
	case FileCreated:
		return "created"
	case FileWritten:
		return "written"
	case FileRemoved:
		return "removed"
	default:
		return "unknown"
	}
}

// FileEvent represents a file system event for a watched file
type FileEvent struct {
	Path      string    // Absolute path to the file
	Op        FileOp    // Type of operation
	Timestamp time.Time // When the event occurred
}

// FileWatcher watches directories for JSONL file changes
type FileWatcher struct {
	watcher *fsnotify.Watcher
	events  chan FileEvent
	errors  chan error
	done    chan struct{}
	rootDir string
	pattern string // e.g., "*.jsonl"

	mu            sync.Mutex
	debounceDelay time.Duration
	debounceMap   map[string]*time.Timer
	closed        bool
}

// DefaultDebounceDelay is the default delay for coalescing rapid writes
const DefaultDebounceDelay = 100 * time.Millisecond

// NewFileWatcher creates a new FileWatcher for the given root directory and pattern
func NewFileWatcher(rootDir, pattern string) (*FileWatcher, error) {
	// Expand ~ to home directory
	if strings.HasPrefix(rootDir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		rootDir = filepath.Join(home, rootDir[1:])
	}

	// Clean the path
	rootDir = filepath.Clean(rootDir)

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fw := &FileWatcher{
		watcher:       watcher,
		events:        make(chan FileEvent, 100),
		errors:        make(chan error, 10),
		done:          make(chan struct{}),
		rootDir:       rootDir,
		pattern:       pattern,
		debounceDelay: DefaultDebounceDelay,
		debounceMap:   make(map[string]*time.Timer),
	}

	// Add the root directory and all subdirectories
	if err := fw.addRecursive(rootDir); err != nil {
		watcher.Close()
		return nil, err
	}

	// Start the event processing goroutine
	go fw.processEvents()

	return fw, nil
}

// addRecursive adds the directory and all its subdirectories to the watcher
func (fw *FileWatcher) addRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// If the directory doesn't exist, that's ok - skip it
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if info.IsDir() {
			if err := fw.watcher.Add(path); err != nil {
				// Ignore permission errors for directories we can't access
				if os.IsPermission(err) {
					return nil
				}
				return err
			}
		}

		return nil
	})
}

// processEvents processes fsnotify events and converts them to FileEvents
func (fw *FileWatcher) processEvents() {
	for {
		select {
		case <-fw.done:
			return
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			select {
			case fw.errors <- err:
			default:
				// Error channel full, drop the error
			}
		}
	}
}

// handleEvent processes a single fsnotify event
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	path := event.Name

	// Handle new directories - add them to the watcher
	if event.Has(fsnotify.Create) {
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			// Add the new directory and any subdirectories
			if err := fw.addRecursive(path); err != nil {
				select {
				case fw.errors <- err:
				default:
				}
			}
		}
	}

	// Check if the file matches our pattern
	if !fw.matchesPattern(path) {
		return
	}

	// Convert fsnotify operation to FileOp
	var op FileOp
	switch {
	case event.Has(fsnotify.Create):
		op = FileCreated
	case event.Has(fsnotify.Write):
		op = FileWritten
	case event.Has(fsnotify.Remove):
		op = FileRemoved
	case event.Has(fsnotify.Rename):
		// Treat rename as remove (file moved away)
		op = FileRemoved
	default:
		// Ignore chmod events
		return
	}

	// Debounce write events
	if op == FileWritten {
		fw.debounce(path, op)
	} else {
		// For create/remove, send immediately
		fw.sendEvent(path, op)
	}
}

// matchesPattern checks if the file path matches the configured pattern
func (fw *FileWatcher) matchesPattern(path string) bool {
	if fw.pattern == "" {
		return true
	}

	// Match against filename only
	filename := filepath.Base(path)
	matched, err := filepath.Match(fw.pattern, filename)
	if err != nil {
		return false
	}
	return matched
}

// debounce coalesces rapid writes for the same file
func (fw *FileWatcher) debounce(path string, op FileOp) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return
	}

	// Cancel existing timer if any
	if timer, exists := fw.debounceMap[path]; exists {
		timer.Stop()
	}

	// Create new timer
	fw.debounceMap[path] = time.AfterFunc(fw.debounceDelay, func() {
		fw.mu.Lock()
		delete(fw.debounceMap, path)
		fw.mu.Unlock()

		fw.sendEvent(path, op)
	})
}

// sendEvent sends a FileEvent to the events channel
func (fw *FileWatcher) sendEvent(path string, op FileOp) {
	event := FileEvent{
		Path:      path,
		Op:        op,
		Timestamp: time.Now(),
	}

	select {
	case fw.events <- event:
	case <-fw.done:
	default:
		// Events channel full, drop the event
	}
}

// Events returns the channel for receiving file events
func (fw *FileWatcher) Events() <-chan FileEvent {
	return fw.events
}

// Errors returns the channel for receiving errors
func (fw *FileWatcher) Errors() <-chan error {
	return fw.errors
}

// Close stops the file watcher and releases resources
func (fw *FileWatcher) Close() error {
	fw.mu.Lock()
	if fw.closed {
		fw.mu.Unlock()
		return nil
	}
	fw.closed = true

	// Cancel all pending debounce timers
	for _, timer := range fw.debounceMap {
		timer.Stop()
	}
	fw.debounceMap = nil
	fw.mu.Unlock()

	// Signal done to stop the event processing goroutine
	close(fw.done)

	// Close the underlying watcher
	return fw.watcher.Close()
}

// RootDir returns the root directory being watched
func (fw *FileWatcher) RootDir() string {
	return fw.rootDir
}

// Pattern returns the file pattern being matched
func (fw *FileWatcher) Pattern() string {
	return fw.pattern
}

// SetDebounceDelay sets the debounce delay for coalescing rapid writes
// This should only be called before the watcher starts receiving events
func (fw *FileWatcher) SetDebounceDelay(delay time.Duration) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.debounceDelay = delay
}
