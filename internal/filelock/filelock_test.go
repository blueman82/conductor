package filelock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewFileLock(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	lock := NewFileLock(lockPath)
	if lock == nil {
		t.Fatal("NewFileLock should not return nil")
	}

	if lock.path != lockPath {
		t.Errorf("Expected lock path %s, got %s", lockPath, lock.path)
	}
}

func TestLockUnlock(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	lock := NewFileLock(lockPath)

	// Test lock
	err := lock.Lock()
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Test unlock
	err = lock.Unlock()
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}
}

func TestConcurrentLocking(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	const goroutines = 5
	const iterations = 10

	// Use a file to track counter to test file-based locking
	counterPath := filepath.Join(tmpDir, "counter.txt")
	os.WriteFile(counterPath, []byte("0"), 0644)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				lock := NewFileLock(lockPath)

				err := lock.Lock()
				if err != nil {
					t.Errorf("Failed to acquire lock: %v", err)
					return
				}

				// Critical section - read, increment, write counter file
				data, err := os.ReadFile(counterPath)
				if err != nil {
					t.Errorf("Failed to read counter: %v", err)
					lock.Unlock()
					return
				}

				var counter int
				fmt.Sscanf(string(data), "%d", &counter)
				time.Sleep(1 * time.Millisecond) // Simulate work
				counter++

				err = os.WriteFile(counterPath, []byte(fmt.Sprintf("%d", counter)), 0644)
				if err != nil {
					t.Errorf("Failed to write counter: %v", err)
					lock.Unlock()
					return
				}

				err = lock.Unlock()
				if err != nil {
					t.Errorf("Failed to release lock: %v", err)
					return
				}
			}
		}()
	}

	wg.Wait()

	// Read final counter value
	data, err := os.ReadFile(counterPath)
	if err != nil {
		t.Fatalf("Failed to read final counter: %v", err)
	}

	var finalCounter int
	fmt.Sscanf(string(data), "%d", &finalCounter)

	expected := goroutines * iterations
	if finalCounter != expected {
		t.Errorf("Expected counter %d, got %d (race condition detected)", expected, finalCounter)
	}
}

func TestTryLock(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	lock1 := NewFileLock(lockPath)
	lock2 := NewFileLock(lockPath)

	// First lock should succeed
	acquired, err := lock1.TryLock()
	if err != nil {
		t.Fatalf("TryLock failed: %v", err)
	}
	if !acquired {
		t.Fatal("First TryLock should succeed")
	}

	// Second lock should fail (already locked)
	acquired, err = lock2.TryLock()
	if err != nil {
		t.Fatalf("TryLock failed: %v", err)
	}
	if acquired {
		t.Error("Second TryLock should fail when lock is held")
	}

	// After unlock, should succeed
	err = lock1.Unlock()
	if err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	acquired, err = lock2.TryLock()
	if err != nil {
		t.Fatalf("TryLock failed: %v", err)
	}
	if !acquired {
		t.Error("TryLock should succeed after unlock")
	}

	lock2.Unlock()
}

func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test.txt")

	content := []byte("Hello, World!")

	err := AtomicWrite(targetPath, content)
	if err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// Verify file was written
	readContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("Expected content %q, got %q", string(content), string(readContent))
	}
}

func TestAtomicWriteOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test.txt")

	// Write initial content
	initialContent := []byte("Initial content")
	err := os.WriteFile(targetPath, initialContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	// Overwrite with atomic write
	newContent := []byte("New content")
	err = AtomicWrite(targetPath, newContent)
	if err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// Verify file was overwritten
	readContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readContent) != string(newContent) {
		t.Errorf("Expected content %q, got %q", string(newContent), string(readContent))
	}
}

func TestConcurrentAtomicWrites(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test.txt")

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			content := []byte(string(rune('A' + id)))
			err := AtomicWrite(targetPath, content)
			if err != nil {
				t.Errorf("AtomicWrite failed for goroutine %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// File should exist and contain valid content from one of the writes
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Fatal("File should exist after concurrent writes")
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Content should be one byte (one of the writes succeeded)
	if len(content) != 1 {
		t.Errorf("Expected 1 byte, got %d bytes: %q", len(content), string(content))
	}
}

func TestAtomicWriteWithLock(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test.txt")
	lockPath := filepath.Join(tmpDir, "test.txt.lock")

	lock := NewFileLock(lockPath)

	// Lock first
	err := lock.Lock()
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Write while holding lock
	content := []byte("Locked content")
	err = AtomicWrite(targetPath, content)
	if err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// Unlock
	err = lock.Unlock()
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}

	// Verify content
	readContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("Expected content %q, got %q", string(content), string(readContent))
	}
}

func TestAtomicWritePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test.txt")

	content := []byte("Test content")
	err := AtomicWrite(targetPath, content)
	if err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := info.Mode()
	expectedMode := os.FileMode(0644)

	if mode.Perm() != expectedMode {
		t.Errorf("Expected permissions %v, got %v", expectedMode, mode.Perm())
	}
}

func TestAtomicWriteNoTempFileLeftBehind(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test.txt")

	content := []byte("Test content")
	err := AtomicWrite(targetPath, content)
	if err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// List all files in directory
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	// Should only have the target file, no temp files
	if len(entries) != 1 {
		var files []string
		for _, entry := range entries {
			files = append(files, entry.Name())
		}
		t.Errorf("Expected only 1 file, found %d: %v", len(entries), files)
	}

	if entries[0].Name() != "test.txt" {
		t.Errorf("Expected file test.txt, got %s", entries[0].Name())
	}
}

func TestLockAndWrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test.txt")

	content := []byte("LockAndWrite content")
	err := LockAndWrite(targetPath, content)
	if err != nil {
		t.Fatalf("LockAndWrite failed: %v", err)
	}

	// Verify file was written
	readContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("Expected content %q, got %q", string(content), string(readContent))
	}
}

func TestConcurrentLockAndWrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test.txt")

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			content := []byte(string(rune('A' + id)))
			err := LockAndWrite(targetPath, content)
			if err != nil {
				t.Errorf("LockAndWrite failed for goroutine %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// File should exist and contain valid content from one of the writes
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Fatal("File should exist after concurrent writes")
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Content should be one byte (one of the writes succeeded)
	if len(content) != 1 {
		t.Errorf("Expected 1 byte, got %d bytes: %q", len(content), string(content))
	}
}

func TestAtomicWriteCreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "subdir", "nested", "test.txt")

	content := []byte("Test content")
	err := AtomicWrite(targetPath, content)
	if err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// Verify file was written
	readContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("Expected content %q, got %q", string(content), string(readContent))
	}

	// Verify directory was created
	dirPath := filepath.Join(tmpDir, "subdir", "nested")
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Error("Directory should have been created")
	}
}

func TestLockWithTimeoutSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	holder := NewFileLock(lockPath)
	if err := holder.Lock(); err != nil {
		t.Fatalf("failed to acquire holder lock: %v", err)
	}

	released := make(chan struct{})
	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := holder.Unlock(); err != nil {
			t.Errorf("failed to release holder lock: %v", err)
		}
		close(released)
	}()

	contender := NewFileLock(lockPath)
	start := time.Now()
	if err := contender.LockWithTimeout(500 * time.Millisecond); err != nil {
		t.Fatalf("LockWithTimeout should succeed: %v", err)
	}

	wait := time.Since(start)
	if wait < 90*time.Millisecond {
		t.Fatalf("expected to wait for lock, waited only %v", wait)
	}

	metrics := contender.LastMetrics()
	if metrics.Attempts < 2 {
		t.Fatalf("expected multiple attempts, got %d", metrics.Attempts)
	}
	if metrics.TimedOut {
		t.Fatal("metrics should not report timeout")
	}

	if err := contender.Unlock(); err != nil {
		t.Fatalf("failed to release contender lock: %v", err)
	}

	<-released
}

func TestLockWithTimeoutTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	holder := NewFileLock(lockPath)
	if err := holder.Lock(); err != nil {
		t.Fatalf("failed to acquire holder lock: %v", err)
	}

	contender := NewFileLock(lockPath)
	err := contender.LockWithTimeout(100 * time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("expected ErrLockTimeout, got %v", err)
	}

	metrics := contender.LastMetrics()
	if !metrics.TimedOut {
		t.Fatal("metrics should report timeout")
	}
	if metrics.Attempts == 0 {
		t.Fatal("expected at least one lock attempt")
	}

	if err := holder.Unlock(); err != nil {
		t.Fatalf("failed to release holder lock: %v", err)
	}
}

func TestSetMonitorReceivesMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	lock := NewFileLock(lockPath)

	metricsCh := make(chan LockMetrics, 1)
	lock.SetMonitor(func(path string, metrics LockMetrics) {
		if path != lockPath {
			t.Errorf("unexpected path in monitor: %s", path)
		}
		metricsCh <- metrics
	})

	if err := lock.Lock(); err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	if err := lock.Unlock(); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	select {
	case metrics := <-metricsCh:
		if metrics.Attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", metrics.Attempts)
		}
	case <-time.After(time.Second):
		t.Fatal("monitor did not receive metrics")
	}

	lock.SetMonitor(nil)
}

func TestMonitorReceivesTimeoutMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	holder := NewFileLock(lockPath)
	if err := holder.Lock(); err != nil {
		t.Fatalf("failed to acquire holder lock: %v", err)
	}

	contender := NewFileLock(lockPath)
	metricsCh := make(chan LockMetrics, 1)
	contender.SetMonitor(func(path string, metrics LockMetrics) {
		metricsCh <- metrics
	})

	err := contender.LockWithTimeout(100 * time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	select {
	case metrics := <-metricsCh:
		if !metrics.TimedOut {
			t.Fatal("monitor metrics should indicate timeout")
		}
	case <-time.After(time.Second):
		t.Fatal("monitor did not capture timeout metrics")
	}

	if err := holder.Unlock(); err != nil {
		t.Fatalf("failed to release holder lock: %v", err)
	}
}

func TestLockAndWrite_DeletesLockFile(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test.txt")
	lockPath := targetPath + ".lock"

	content := []byte("test content")
	err := LockAndWrite(targetPath, content)
	if err != nil {
		t.Fatalf("LockAndWrite failed: %v", err)
	}

	// Verify target file was created
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Fatalf("Target file %s was not created", targetPath)
	}

	// Verify lock file was deleted
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Errorf("Lock file %s was not deleted", lockPath)
	}
}

func TestLockAndWrite_DeletesLockFileOnError(t *testing.T) {
	// Skip when running as root since root bypasses permission checks
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	// Create a read-only directory to force write failure
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

	targetPath := filepath.Join(readOnlyDir, "test.txt")
	lockPath := targetPath + ".lock"

	// Create the lock file first to test that it exists before the operation
	// The lock acquisition should succeed, but the write should fail
	content := []byte("test content")
	err := LockAndWrite(targetPath, content)
	if err == nil {
		t.Fatal("Expected LockAndWrite to fail when writing to read-only directory")
	}

	// Verify lock file was deleted even though write failed
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Errorf("Lock file %s was not deleted after error", lockPath)
	}
}

func TestLockAndWrite_ConcurrentDeletesAllLockFiles(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test.txt")
	lockPath := targetPath + ".lock"

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			content := []byte(fmt.Sprintf("content-%d", id))
			err := LockAndWrite(targetPath, content)
			if err != nil {
				t.Errorf("LockAndWrite failed for goroutine %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify target file exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Fatal("Target file should exist after concurrent writes")
	}

	// Verify lock file was deleted
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Errorf("Lock file %s was not deleted after concurrent writes", lockPath)
	}
}
