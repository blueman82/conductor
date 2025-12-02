package executor

import (
	"fmt"
	"sync"
	"time"
)

// generateSessionID creates a unique session identifier with timestamp format
// Format: session-YYYYMMDD-HHMMSS
// Example: session-20251112-143045
func generateSessionID() string {
	now := time.Now()
	return fmt.Sprintf("session-%s", now.Format("20060102-150405"))
}

// =============================================================================
// Session-Level Package Lock Manager (v2.9+)
// =============================================================================

// SessionPackageLockManager provides session-scoped package locking.
// It tracks which packages are in-flight across all tasks in a session
// and prevents concurrent modifications to the same Go package.
type SessionPackageLockManager struct {
	mu       sync.Mutex
	packages map[string]*packageHold // package path -> hold info
}

// packageHold tracks a package lock hold
type packageHold struct {
	taskNumber string
	heldSince  time.Time
}

// NewSessionPackageLockManager creates a new session-level package lock manager.
func NewSessionPackageLockManager() *SessionPackageLockManager {
	return &SessionPackageLockManager{
		packages: make(map[string]*packageHold),
	}
}

// TryLock attempts to acquire locks for all packages.
// Returns (true, release) if all packages are available.
// Returns (false, nil) if any package is held by another task.
// Thread-safe: uses atomic all-or-nothing acquisition.
func (sm *SessionPackageLockManager) TryLock(taskNumber string, packages []string) (bool, func()) {
	if len(packages) == 0 {
		return true, func() {}
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if all packages are available
	for _, pkg := range packages {
		if hold, exists := sm.packages[pkg]; exists && hold.taskNumber != taskNumber {
			return false, nil
		}
	}

	// All available - acquire all
	now := time.Now()
	for _, pkg := range packages {
		sm.packages[pkg] = &packageHold{
			taskNumber: taskNumber,
			heldSince:  now,
		}
	}

	// Return release function
	return true, func() {
		sm.mu.Lock()
		defer sm.mu.Unlock()
		for _, pkg := range packages {
			if hold, exists := sm.packages[pkg]; exists && hold.taskNumber == taskNumber {
				delete(sm.packages, pkg)
			}
		}
	}
}

// IsPackageHeld returns true if the package is currently held by any task.
func (sm *SessionPackageLockManager) IsPackageHeld(pkg string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	_, exists := sm.packages[pkg]
	return exists
}

// GetPackageHolder returns the task number holding the package, or empty string.
func (sm *SessionPackageLockManager) GetPackageHolder(pkg string) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if hold, exists := sm.packages[pkg]; exists {
		return hold.taskNumber
	}
	return ""
}

// GetHeldPackages returns a list of all currently held packages with their holders.
func (sm *SessionPackageLockManager) GetHeldPackages() map[string]string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	result := make(map[string]string)
	for pkg, hold := range sm.packages {
		result[pkg] = hold.taskNumber
	}
	return result
}

// Clear releases all package holds. Use during session cleanup.
func (sm *SessionPackageLockManager) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.packages = make(map[string]*packageHold)
}
