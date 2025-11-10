// Package filelock provides locking and atomic write helpers for safely
// updating plan files from multiple goroutines or processes. Locks are written
// alongside their target using the same path with a `.lock` suffix, e.g.
// `plan.md.lock`, so callers can coordinate access across the filesystem.
package filelock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

// FileLock wraps a flock file lock for coordinating access to files.
type FileLock struct {
	flock *flock.Flock
	path  string

	mu          sync.Mutex
	lastMetrics LockMetrics
	monitor     LockMonitor
}

// NewFileLock creates a new file lock for the given path.
// The lock file will be created at the specified path.
func NewFileLock(path string) *FileLock {
	return &FileLock{
		flock: flock.New(path),
		path:  path,
	}
}

// LockMetrics captures metadata about the most recent lock acquisition attempt.
type LockMetrics struct {
	Attempts     int
	WaitDuration time.Duration
	TimedOut     bool
}

// LockMonitor receives lock acquisition metrics, enabling callers to track
// contention in production environments.
type LockMonitor func(path string, metrics LockMetrics)

// ErrLockTimeout indicates a lock attempt timed out before it could be acquired.
var ErrLockTimeout = errors.New("filelock: lock acquisition timed out")

// Lock acquires an exclusive lock on the file, blocking until the lock is available.
// Returns an error if the lock cannot be acquired.
func (fl *FileLock) Lock() error {
	start := time.Now()
	err := fl.flock.Lock()
	if err != nil {
		return fmt.Errorf("failed to acquire lock on %s: %w", fl.path, err)
	}

	fl.recordMetrics(LockMetrics{Attempts: 1, WaitDuration: time.Since(start)})
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

	if acquired {
		fl.recordMetrics(LockMetrics{Attempts: 1, WaitDuration: 0})
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

// LockWithTimeout attempts to acquire the lock within the provided timeout.
// When timeout is zero or negative it behaves like Lock() and blocks until the
// lock is available. Attempts poll every 50ms, which balances responsiveness
// with low CPU usage. On success the latest metrics can be retrieved via
// LastMetrics. If the timeout is exceeded ErrLockTimeout is returned.
func (fl *FileLock) LockWithTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return fl.Lock()
	}

	start := time.Now()
	deadline := start.Add(timeout)
	attempts := 0

	for {
		attempts++

		acquired, err := fl.flock.TryLock()
		if err != nil {
			return fmt.Errorf("failed to try lock on %s: %w", fl.path, err)
		}

		if acquired {
			fl.recordMetrics(LockMetrics{Attempts: attempts, WaitDuration: time.Since(start)})
			return nil
		}

		if time.Now().After(deadline) {
			fl.recordMetrics(LockMetrics{Attempts: attempts, WaitDuration: time.Since(start), TimedOut: true})
			return fmt.Errorf("timed out acquiring lock on %s after %v: %w", fl.path, timeout, ErrLockTimeout)
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// LastMetrics returns the metrics from the most recent successful or timed-out
// lock attempt.
func (fl *FileLock) LastMetrics() LockMetrics {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	return fl.lastMetrics
}

// SetMonitor registers a callback that receives lock metrics after each
// acquisition attempt. Passing nil removes the monitor.
func (fl *FileLock) SetMonitor(m LockMonitor) {
	fl.mu.Lock()
	fl.monitor = m
	fl.mu.Unlock()
}

func (fl *FileLock) recordMetrics(metrics LockMetrics) {
	fl.mu.Lock()
	fl.lastMetrics = metrics
	monitor := fl.monitor
	fl.mu.Unlock()

	if monitor != nil {
		monitor(fl.path, metrics)
	}
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
//
// The lock file is automatically deleted after the lock is released, ensuring
// that lock files do not persist on the filesystem after the operation completes.
func LockAndWrite(path string, data []byte) error {
	lockPath := path + ".lock"
	lock := NewFileLock(lockPath)

	// Acquire lock
	if err := lock.Lock(); err != nil {
		return err
	}
	defer func() {
		lock.Unlock()
		os.Remove(lockPath)
	}()

	// Perform atomic write while holding lock
	return AtomicWrite(path, data)
}
