// Package filelock provides file locking and atomic write operations for safe
// concurrent file access across multiple goroutines and processes.
package filelock

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// FileLock wraps a flock file lock for coordinating access to files.
type FileLock struct {
	flock *flock.Flock
	path  string
}

// NewFileLock creates a new file lock for the given path.
// The lock file will be created at the specified path.
func NewFileLock(path string) *FileLock {
	return &FileLock{
		flock: flock.New(path),
		path:  path,
	}
}

// Lock acquires an exclusive lock on the file, blocking until the lock is available.
// Returns an error if the lock cannot be acquired.
func (fl *FileLock) Lock() error {
	err := fl.flock.Lock()
	if err != nil {
		return fmt.Errorf("failed to acquire lock on %s: %w", fl.path, err)
	}
	return nil
}

// TryLock attempts to acquire an exclusive lock on the file without blocking.
// Returns true if the lock was acquired, false if the lock is held by another process.
// Returns an error if the lock operation fails.
func (fl *FileLock) TryLock() (bool, error) {
	acquired, err := fl.flock.TryLock()
	if err != nil {
		return false, fmt.Errorf("failed to try lock on %s: %w", fl.path, err)
	}
	return acquired, nil
}

// Unlock releases the lock.
// Returns an error if the unlock operation fails.
func (fl *FileLock) Unlock() error {
	err := fl.flock.Unlock()
	if err != nil {
		return fmt.Errorf("failed to release lock on %s: %w", fl.path, err)
	}
	return nil
}

// AtomicWrite writes data to a file atomically using a temp file and rename strategy.
// This ensures that readers never see partial writes, even if the write is interrupted.
//
// The process:
// 1. Create a temporary file in the same directory as the target
// 2. Write content to the temporary file
// 3. Rename the temporary file to the target path (atomic operation)
//
// If the operation fails at any point, the original file (if it exists) remains unchanged.
func AtomicWrite(path string, data []byte) error {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create temporary file in same directory as target
	// This ensures the temp file is on the same filesystem, making rename atomic
	tempFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Ensure temp file is cleaned up on error
	defer func() {
		if tempFile != nil {
			tempFile.Close()
			os.Remove(tempPath)
		}
	}()

	// Write data to temp file
	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close temp file
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set correct permissions (0644 = rw-r--r--)
	if err := os.Chmod(tempPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename: this is the key operation that makes the write atomic
	// On Unix systems, rename is atomic within the same filesystem
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file to %s: %w", path, err)
	}

	// Success - prevent cleanup of temp file since it's now renamed
	tempFile = nil

	return nil
}

// LockAndWrite acquires a lock, performs an atomic write, and releases the lock.
// This is a convenience function for the common pattern of locking before writing.
//
// The lock path is derived by appending ".lock" to the target path.
// Example: writing to "plan.md" uses lock file "plan.md.lock"
func LockAndWrite(path string, data []byte) error {
	lockPath := path + ".lock"
	lock := NewFileLock(lockPath)

	// Acquire lock
	if err := lock.Lock(); err != nil {
		return err
	}
	defer lock.Unlock()

	// Perform atomic write while holding lock
	return AtomicWrite(path, data)
}
