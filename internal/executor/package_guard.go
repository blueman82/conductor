package executor

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/harrison/conductor/internal/models"
)

// =============================================================================
// Package Conflict Detection (Validation Time)
// =============================================================================

// DetectPackageConflicts inspects tasks and detects when multiple tasks modify
// the same Go package without explicit dependency serialization.
// Returns error with actionable suggestions if conflicts detected.
func DetectPackageConflicts(tasks []models.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	// Build package -> tasks mapping
	pkgToTasks := make(map[string][]string) // package -> list of task numbers
	for _, task := range tasks {
		for _, pkg := range GetTaskPackages(task) {
			pkgToTasks[pkg] = append(pkgToTasks[pkg], task.Number)
		}
	}

	// Build dependency graph for serialization check
	// deps[taskNum] = set of tasks that taskNum depends on (directly or transitively)
	deps := buildTransitiveDeps(tasks)

	// Check for conflicts: multiple tasks touching same package without serialization
	var conflicts []string
	for pkg, taskNums := range pkgToTasks {
		if len(taskNums) < 2 {
			continue
		}

		// Check if all pairs are serialized (one depends on the other transitively)
		conflicting := findUnserializedPairs(taskNums, deps)
		if len(conflicting) > 0 {
			conflicts = append(conflicts, fmt.Sprintf(
				"package %q: tasks %v modify same package without serialization. "+
					"Add depends_on to serialize or split tasks.",
				pkg, conflicting))
		}
	}

	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		return fmt.Errorf("package conflicts detected:\n  - %s", strings.Join(conflicts, "\n  - "))
	}

	return nil
}

// buildTransitiveDeps builds transitive dependency map for all tasks.
// Returns map[taskNum] -> set of all tasks that taskNum depends on (transitively).
func buildTransitiveDeps(tasks []models.Task) map[string]map[string]bool {
	// Build direct deps
	directDeps := make(map[string][]string)
	for _, task := range tasks {
		directDeps[task.Number] = task.DependsOn
	}

	// Build transitive closure
	transitive := make(map[string]map[string]bool)
	for _, task := range tasks {
		transitive[task.Number] = make(map[string]bool)
		visited := make(map[string]bool)
		var dfs func(string)
		dfs = func(taskNum string) {
			if visited[taskNum] {
				return
			}
			visited[taskNum] = true
			for _, dep := range directDeps[taskNum] {
				transitive[task.Number][dep] = true
				dfs(dep)
			}
		}
		dfs(task.Number)
	}

	return transitive
}

// findUnserializedPairs finds pairs of tasks that are not serialized via depends_on.
func findUnserializedPairs(taskNums []string, deps map[string]map[string]bool) []string {
	var unserialized []string
	seen := make(map[string]bool)

	for i := 0; i < len(taskNums); i++ {
		for j := i + 1; j < len(taskNums); j++ {
			t1, t2 := taskNums[i], taskNums[j]

			// Check if either depends on the other (directly or transitively)
			if deps[t1][t2] || deps[t2][t1] {
				continue // Serialized
			}

			// Not serialized - record conflict
			pair := fmt.Sprintf("%s,%s", t1, t2)
			if !seen[pair] {
				seen[pair] = true
				unserialized = append(unserialized, pair)
			}
		}
	}

	return unserialized
}

// =============================================================================
// Package Guard (Runtime Enforcement)
// =============================================================================

// PackageGuard provides runtime enforcement of package isolation.
// It ensures no two tasks modify the same Go package concurrently.
type PackageGuard struct {
	mu      sync.Mutex
	locks   map[string]*packageLock // package path -> lock info
	waiters map[string][]chan struct{}
}

type packageLock struct {
	holder   string // task number holding the lock
	acquired bool
}

// NewPackageGuard creates a new PackageGuard for runtime enforcement.
func NewPackageGuard() *PackageGuard {
	return &PackageGuard{
		locks:   make(map[string]*packageLock),
		waiters: make(map[string][]chan struct{}),
	}
}

// Acquire attempts to acquire locks for all packages needed by a task.
// Blocks until all packages are available or context is cancelled.
// Returns a release function that must be called when task completes.
func (pg *PackageGuard) Acquire(ctx context.Context, taskNum string, packages []string) (func(), error) {
	if len(packages) == 0 {
		return func() {}, nil
	}

	// Sort packages to prevent deadlock (consistent ordering)
	sortedPkgs := make([]string, len(packages))
	copy(sortedPkgs, packages)
	sort.Strings(sortedPkgs)

	// Acquire each package lock in order
	acquired := make([]string, 0, len(sortedPkgs))
	for _, pkg := range sortedPkgs {
		if err := pg.acquireOne(ctx, taskNum, pkg); err != nil {
			// Release any already acquired
			for _, acq := range acquired {
				pg.releaseOne(taskNum, acq)
			}
			return nil, err
		}
		acquired = append(acquired, pkg)
	}

	// Return release function
	return func() {
		for _, pkg := range acquired {
			pg.releaseOne(taskNum, pkg)
		}
	}, nil
}

// TryAcquire attempts to acquire locks without blocking.
// Returns (true, release) if successful, (false, nil) if any package is held.
func (pg *PackageGuard) TryAcquire(taskNum string, packages []string) (bool, func()) {
	if len(packages) == 0 {
		return true, func() {}
	}

	pg.mu.Lock()
	defer pg.mu.Unlock()

	// Check if all packages are available
	for _, pkg := range packages {
		if lock, exists := pg.locks[pkg]; exists && lock.acquired {
			return false, nil
		}
	}

	// All available - acquire all
	for _, pkg := range packages {
		pg.locks[pkg] = &packageLock{holder: taskNum, acquired: true}
	}

	return true, func() {
		pg.mu.Lock()
		defer pg.mu.Unlock()
		for _, pkg := range packages {
			delete(pg.locks, pkg)
			pg.notifyWaiters(pkg)
		}
	}
}

// IsHeld returns true if the package is currently held by any task.
func (pg *PackageGuard) IsHeld(pkg string) bool {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	lock, exists := pg.locks[pkg]
	return exists && lock.acquired
}

// GetHolder returns the task number holding the package lock, or empty if not held.
func (pg *PackageGuard) GetHolder(pkg string) string {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	if lock, exists := pg.locks[pkg]; exists && lock.acquired {
		return lock.holder
	}
	return ""
}

func (pg *PackageGuard) acquireOne(ctx context.Context, taskNum, pkg string) error {
	pg.mu.Lock()

	// Check if available
	if lock, exists := pg.locks[pkg]; !exists || !lock.acquired {
		pg.locks[pkg] = &packageLock{holder: taskNum, acquired: true}
		pg.mu.Unlock()
		return nil
	}

	// Need to wait - create waiter channel
	waiter := make(chan struct{}, 1)
	pg.waiters[pkg] = append(pg.waiters[pkg], waiter)
	pg.mu.Unlock()

	// Wait for release or context cancellation
	select {
	case <-ctx.Done():
		// Remove ourselves from waiters
		pg.mu.Lock()
		waiters := pg.waiters[pkg]
		for i, w := range waiters {
			if w == waiter {
				pg.waiters[pkg] = append(waiters[:i], waiters[i+1:]...)
				break
			}
		}
		pg.mu.Unlock()
		return ctx.Err()

	case <-waiter:
		// We were notified - try to acquire
		pg.mu.Lock()
		if lock, exists := pg.locks[pkg]; !exists || !lock.acquired {
			pg.locks[pkg] = &packageLock{holder: taskNum, acquired: true}
			pg.mu.Unlock()
			return nil
		}
		pg.mu.Unlock()
		// Someone else got it - retry
		return pg.acquireOne(ctx, taskNum, pkg)
	}
}

func (pg *PackageGuard) releaseOne(taskNum, pkg string) {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	if lock, exists := pg.locks[pkg]; exists && lock.holder == taskNum {
		delete(pg.locks, pkg)
		pg.notifyWaiters(pkg)
	}
}

func (pg *PackageGuard) notifyWaiters(pkg string) {
	if waiters, exists := pg.waiters[pkg]; exists && len(waiters) > 0 {
		// Notify first waiter
		select {
		case waiters[0] <- struct{}{}:
		default:
		}
		pg.waiters[pkg] = waiters[1:]
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// GetGoPackage extracts the Go package path from a file path.
// Returns empty string for non-Go files or root-level Go files.
// Root-level files (e.g., "main.go", "setup.go") are excluded from
// package conflict detection to avoid false positives in test fixtures.
func GetGoPackage(file string) string {
	// Only process .go files
	if !strings.HasSuffix(file, ".go") {
		return ""
	}

	// Get directory (package path)
	dir := filepath.Dir(file)
	if dir == "" || dir == "/" || dir == "." {
		// Exclude root-level Go files from package guard
		// These are often test fixtures or standalone scripts
		return ""
	}

	// Normalize path
	dir = filepath.Clean(dir)
	if dir == "." {
		return ""
	}

	return dir
}

// GetTaskPackages extracts unique Go package paths from a task's files.
func GetTaskPackages(task models.Task) []string {
	pkgSet := make(map[string]bool)
	for _, file := range task.Files {
		pkg := GetGoPackage(file)
		if pkg != "" {
			pkgSet[pkg] = true
		}
	}

	packages := make([]string, 0, len(pkgSet))
	for pkg := range pkgSet {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)
	return packages
}

// ValidatePackageConflictsInWave checks a single wave for package conflicts.
// Used by wave builder to inject synthetic dependencies.
func ValidatePackageConflictsInWave(tasks []models.Task) error {
	return DetectPackageConflicts(tasks)
}

// =============================================================================
// EnforcePackageIsolation (Runtime Git Diff Validation)
// =============================================================================

// ErrPackageIsolationViolation indicates modified files outside declared scope.
var ErrPackageIsolationViolation = fmt.Errorf("package isolation violation")

// PackageIsolationResult contains the result of package isolation enforcement.
type PackageIsolationResult struct {
	Passed           bool
	DeclaredFiles    []string
	ModifiedFiles    []string
	UndeclaredFiles  []string
	DeclaredPackages []string
	ModifiedPackages []string
	Remediation      string
}

// EnforcePackageIsolation verifies that git diff shows only modifications
// within the task's declared files/packages. Returns error with remediation
// if files were modified outside the declared scope.
//
// Parameters:
//   - ctx: Context for command execution
//   - runner: CommandRunner for executing git commands
//   - task: Task with declared Files to check against
//
// Returns nil if all modifications are within declared scope.
// Returns ErrPackageIsolationViolation (wrapped) if violations detected.
func EnforcePackageIsolation(ctx context.Context, runner CommandRunner, task models.Task) (*PackageIsolationResult, error) {
	result := &PackageIsolationResult{
		DeclaredFiles:    task.Files,
		DeclaredPackages: GetTaskPackages(task),
	}

	// No declared files - skip enforcement (can't validate without declarations)
	if len(task.Files) == 0 {
		result.Passed = true
		return result, nil
	}

	// Run git diff --name-only to get modified files
	output, err := runner.Run(ctx, "git diff --name-only HEAD")
	if err != nil {
		// Git command failed - check if we're in a git repo
		if strings.Contains(output, "not a git repository") {
			result.Passed = true // Not a git repo - skip enforcement
			return result, nil
		}
		// Other git error - surface it
		return result, fmt.Errorf("git diff failed: %w", err)
	}

	// Also check staged changes
	stagedOutput, err := runner.Run(ctx, "git diff --name-only --cached")
	if err == nil && stagedOutput != "" {
		output = output + "\n" + stagedOutput
	}

	// Parse modified files from git output
	modifiedFiles := parseGitDiffOutput(output)
	result.ModifiedFiles = modifiedFiles

	if len(modifiedFiles) == 0 {
		result.Passed = true
		return result, nil
	}

	// Build set of declared files for quick lookup
	declaredSet := make(map[string]bool)
	for _, f := range task.Files {
		declaredSet[filepath.Clean(f)] = true
	}

	// Build set of declared packages
	declaredPkgSet := make(map[string]bool)
	for _, pkg := range result.DeclaredPackages {
		declaredPkgSet[pkg] = true
	}

	// Check each modified file
	var undeclared []string
	modifiedPkgs := make(map[string]bool)

	for _, modFile := range modifiedFiles {
		normalizedMod := filepath.Clean(modFile)

		// Check if file is explicitly declared
		if declaredSet[normalizedMod] {
			continue
		}

		// Check if file is in a declared package (for Go files)
		modPkg := GetGoPackage(modFile)
		if modPkg != "" {
			modifiedPkgs[modPkg] = true
			if declaredPkgSet[modPkg] {
				continue // Package is declared
			}
		}

		// File is outside declared scope
		undeclared = append(undeclared, modFile)
	}

	// Collect modified packages for reporting
	for pkg := range modifiedPkgs {
		result.ModifiedPackages = append(result.ModifiedPackages, pkg)
	}
	sort.Strings(result.ModifiedPackages)

	result.UndeclaredFiles = undeclared

	if len(undeclared) > 0 {
		result.Passed = false
		result.Remediation = buildRemediationMessage(task, undeclared)
		return result, fmt.Errorf("%w: task %s modified files outside declared scope: %v. %s",
			ErrPackageIsolationViolation, task.Number, undeclared, result.Remediation)
	}

	result.Passed = true
	return result, nil
}

// parseGitDiffOutput parses git diff --name-only output into file paths.
func parseGitDiffOutput(output string) []string {
	var files []string
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			files = append(files, trimmed)
		}
	}
	return files
}

// buildRemediationMessage creates an actionable remediation message.
func buildRemediationMessage(task models.Task, undeclaredFiles []string) string {
	var sb strings.Builder
	sb.WriteString("Remediation options:\n")
	sb.WriteString("  1. Add missing files to task.Files:\n")
	for _, f := range undeclaredFiles {
		sb.WriteString(fmt.Sprintf("     - %s\n", f))
	}
	sb.WriteString("  2. Revert unintended changes: git checkout -- <file>\n")
	sb.WriteString("  3. Split task into multiple tasks with proper file declarations\n")
	return sb.String()
}
