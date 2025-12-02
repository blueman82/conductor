package executor

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/harrison/conductor/internal/models"
)

// =============================================================================
// DetectPackageConflicts Tests
// =============================================================================

func TestDetectPackageConflicts_ConflictingTasks(t *testing.T) {
	// Two tasks modifying same Go package without depends_on relationship
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Task 1",
			Files:  []string{"internal/behavioral/parser.go", "internal/behavioral/parser_test.go"},
		},
		{
			Number: "2",
			Name:   "Task 2",
			Files:  []string{"internal/behavioral/models.go", "internal/behavioral/models_test.go"},
		},
	}

	err := DetectPackageConflicts(tasks)
	if err == nil {
		t.Fatal("expected error for conflicting tasks, got nil")
	}
	if !containsSubstring(err.Error(), "internal/behavioral") {
		t.Errorf("error should mention conflicting package 'internal/behavioral', got: %v", err)
	}
}

func TestDetectPackageConflicts_NoConflict_DifferentPackages(t *testing.T) {
	// Tasks modifying different Go packages
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Task 1",
			Files:  []string{"internal/executor/task.go"},
		},
		{
			Number: "2",
			Name:   "Task 2",
			Files:  []string{"internal/parser/yaml.go"},
		},
	}

	err := DetectPackageConflicts(tasks)
	if err != nil {
		t.Errorf("expected no error for different packages, got: %v", err)
	}
}

func TestDetectPackageConflicts_NoConflict_ExplicitDependsOn(t *testing.T) {
	// Tasks modifying same package but with explicit depends_on
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Task 1",
			Files:  []string{"internal/behavioral/parser.go"},
		},
		{
			Number:    "2",
			Name:      "Task 2",
			DependsOn: []string{"1"}, // Explicit serialization
			Files:     []string{"internal/behavioral/models.go"},
		},
	}

	err := DetectPackageConflicts(tasks)
	if err != nil {
		t.Errorf("expected no error when depends_on serializes tasks, got: %v", err)
	}
}

func TestDetectPackageConflicts_NonGoFiles_Bypass(t *testing.T) {
	// Non-Go files should bypass the guard
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Task 1",
			Files:  []string{"docs/README.md", "docs/guide.md"},
		},
		{
			Number: "2",
			Name:   "Task 2",
			Files:  []string{"docs/api.md"},
		},
	}

	err := DetectPackageConflicts(tasks)
	if err != nil {
		t.Errorf("expected no error for non-Go files, got: %v", err)
	}
}

func TestDetectPackageConflicts_MixedGoAndNonGo(t *testing.T) {
	// Mixed Go and non-Go files - only Go packages checked
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Task 1",
			Files:  []string{"internal/executor/task.go", "docs/README.md"},
		},
		{
			Number: "2",
			Name:   "Task 2",
			Files:  []string{"internal/executor/wave.go", "config.yaml"},
		},
	}

	err := DetectPackageConflicts(tasks)
	if err == nil {
		t.Fatal("expected error for conflicting Go package 'internal/executor', got nil")
	}
}

func TestDetectPackageConflicts_EmptyTasks(t *testing.T) {
	tasks := []models.Task{}

	err := DetectPackageConflicts(tasks)
	if err != nil {
		t.Errorf("expected no error for empty task list, got: %v", err)
	}
}

func TestDetectPackageConflicts_TasksWithNoFiles(t *testing.T) {
	tasks := []models.Task{
		{Number: "1", Name: "Task 1"},
		{Number: "2", Name: "Task 2"},
	}

	err := DetectPackageConflicts(tasks)
	if err != nil {
		t.Errorf("expected no error for tasks without files, got: %v", err)
	}
}

func TestDetectPackageConflicts_TransitiveDependency(t *testing.T) {
	// Task 3 depends on Task 2 which depends on Task 1
	// All touch same package - should be OK due to transitive serialization
	tasks := []models.Task{
		{
			Number: "1",
			Name:   "Task 1",
			Files:  []string{"internal/behavioral/parser.go"},
		},
		{
			Number:    "2",
			Name:      "Task 2",
			DependsOn: []string{"1"},
			Files:     []string{"internal/behavioral/models.go"},
		},
		{
			Number:    "3",
			Name:      "Task 3",
			DependsOn: []string{"2"},
			Files:     []string{"internal/behavioral/types.go"},
		},
	}

	err := DetectPackageConflicts(tasks)
	if err != nil {
		t.Errorf("expected no error for transitively serialized tasks, got: %v", err)
	}
}

func TestDetectPackageConflicts_MultipleConflicts(t *testing.T) {
	// Three tasks all touching same package without serialization
	tasks := []models.Task{
		{Number: "1", Name: "Task 1", Files: []string{"internal/executor/a.go"}},
		{Number: "2", Name: "Task 2", Files: []string{"internal/executor/b.go"}},
		{Number: "3", Name: "Task 3", Files: []string{"internal/executor/c.go"}},
	}

	err := DetectPackageConflicts(tasks)
	if err == nil {
		t.Fatal("expected error for multiple conflicting tasks")
	}
}

func TestDetectPackageConflicts_RootPackageFiles(t *testing.T) {
	// Files in root package - now excluded from package guard
	tasks := []models.Task{
		{Number: "1", Name: "Task 1", Files: []string{"main.go"}},
		{Number: "2", Name: "Task 2", Files: []string{"version.go"}},
	}

	err := DetectPackageConflicts(tasks)
	if err != nil {
		t.Errorf("root package files should be excluded from guard, got: %v", err)
	}
}

// =============================================================================
// PackageGuard Runtime Enforcement Tests
// =============================================================================

func TestNewPackageGuard(t *testing.T) {
	guard := NewPackageGuard()
	if guard == nil {
		t.Fatal("NewPackageGuard returned nil")
	}
}

func TestPackageGuard_AcquireRelease(t *testing.T) {
	guard := NewPackageGuard()
	ctx := context.Background()
	taskNum := "1"
	pkgs := []string{"internal/executor", "internal/parser"}

	// Acquire should succeed
	release, err := guard.Acquire(ctx, taskNum, pkgs)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	if release == nil {
		t.Fatal("Acquire returned nil release function")
	}

	// Release
	release()
}

func TestPackageGuard_ConcurrentSamePackage_Blocked(t *testing.T) {
	guard := NewPackageGuard()
	ctx := context.Background()

	// Task 1 acquires internal/executor
	release1, err := guard.Acquire(ctx, "1", []string{"internal/executor"})
	if err != nil {
		t.Fatalf("Task 1 acquire failed: %v", err)
	}

	// Task 2 tries to acquire same package - should fail immediately with TryAcquire
	acquired, _ := guard.TryAcquire("2", []string{"internal/executor"})
	if acquired {
		t.Error("expected TryAcquire to return false when package is held")
	}

	// Release task 1
	release1()

	// Now task 2 should succeed
	release2, err := guard.Acquire(ctx, "2", []string{"internal/executor"})
	if err != nil {
		t.Fatalf("Task 2 acquire after release failed: %v", err)
	}
	release2()
}

func TestPackageGuard_DifferentPackages_NotBlocked(t *testing.T) {
	guard := NewPackageGuard()
	ctx := context.Background()

	var wg sync.WaitGroup
	errors := make(chan error, 2)

	// Task 1 acquires internal/executor
	wg.Add(1)
	go func() {
		defer wg.Done()
		release, err := guard.Acquire(ctx, "1", []string{"internal/executor"})
		if err != nil {
			errors <- err
			return
		}
		defer release()
	}()

	// Task 2 acquires internal/parser (different package) - should not block
	wg.Add(1)
	go func() {
		defer wg.Done()
		release, err := guard.Acquire(ctx, "2", []string{"internal/parser"})
		if err != nil {
			errors <- err
			return
		}
		defer release()
	}()

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPackageGuard_ContextCancellation(t *testing.T) {
	guard := NewPackageGuard()

	// Task 1 holds the lock
	ctx1 := context.Background()
	release1, _ := guard.Acquire(ctx1, "1", []string{"internal/executor"})
	defer release1()

	// Task 2 tries with cancelled context
	ctx2, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := guard.Acquire(ctx2, "2", []string{"internal/executor"})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

// =============================================================================
// GetGoPackage Tests
// =============================================================================

func TestGetGoPackage(t *testing.T) {
	tests := []struct {
		file     string
		expected string
	}{
		{"internal/executor/task.go", "internal/executor"},
		{"internal/parser/yaml.go", "internal/parser"},
		{"main.go", ""}, // Root-level Go file - excluded
		{"cmd/conductor/main.go", "cmd/conductor"},
		{"README.md", ""},           // Non-Go file
		{"docs/guide.md", ""},       // Non-Go file
		{"config.yaml", ""},         // Non-Go file
		{"internal/test_data/", ""}, // Directory, not file
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			pkg := GetGoPackage(tt.file)
			if pkg != tt.expected {
				t.Errorf("GetGoPackage(%q) = %q, want %q", tt.file, pkg, tt.expected)
			}
		})
	}
}

// =============================================================================
// GetTaskPackages Tests
// =============================================================================

func TestGetTaskPackages(t *testing.T) {
	task := models.Task{
		Number: "1",
		Files: []string{
			"internal/executor/task.go",
			"internal/executor/task_test.go",
			"internal/parser/yaml.go",
			"docs/README.md",
		},
	}

	pkgs := GetTaskPackages(task)

	// Should have 2 unique packages
	if len(pkgs) != 2 {
		t.Errorf("expected 2 packages, got %d: %v", len(pkgs), pkgs)
	}

	// Check both packages present
	hasExecutor := false
	hasParser := false
	for _, pkg := range pkgs {
		if pkg == "internal/executor" {
			hasExecutor = true
		}
		if pkg == "internal/parser" {
			hasParser = true
		}
	}

	if !hasExecutor {
		t.Error("expected internal/executor package")
	}
	if !hasParser {
		t.Error("expected internal/parser package")
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// EnforcePackageIsolation Tests
// =============================================================================

// mockGitRunner simulates git diff output for testing
type mockGitRunner struct {
	diffOutput   string
	cachedOutput string
	err          error
}

func (m *mockGitRunner) Run(ctx context.Context, command string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if strings.Contains(command, "--cached") {
		return m.cachedOutput, nil
	}
	return m.diffOutput, nil
}

func TestEnforcePackageIsolation_NoViolation(t *testing.T) {
	runner := &mockGitRunner{
		diffOutput: "internal/executor/task.go\ninternal/executor/task_test.go\n",
	}
	task := models.Task{
		Number: "1",
		Files:  []string{"internal/executor/task.go", "internal/executor/task_test.go"},
	}

	result, err := EnforcePackageIsolation(context.Background(), runner, task)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Passed {
		t.Error("expected Passed to be true")
	}
	if len(result.UndeclaredFiles) > 0 {
		t.Errorf("expected no undeclared files, got: %v", result.UndeclaredFiles)
	}
}

func TestEnforcePackageIsolation_Violation(t *testing.T) {
	// Modified a file not in declared list
	runner := &mockGitRunner{
		diffOutput: "internal/executor/task.go\ninternal/parser/yaml.go\n",
	}
	task := models.Task{
		Number: "1",
		Files:  []string{"internal/executor/task.go"},
	}

	result, err := EnforcePackageIsolation(context.Background(), runner, task)
	if err == nil {
		t.Fatal("expected error for violation")
	}
	if result.Passed {
		t.Error("expected Passed to be false")
	}
	if len(result.UndeclaredFiles) != 1 || result.UndeclaredFiles[0] != "internal/parser/yaml.go" {
		t.Errorf("expected undeclared file 'internal/parser/yaml.go', got: %v", result.UndeclaredFiles)
	}
	if result.Remediation == "" {
		t.Error("expected remediation message")
	}
}

func TestEnforcePackageIsolation_PackageLevel(t *testing.T) {
	// Modified another file in the same declared package - should be OK
	runner := &mockGitRunner{
		diffOutput: "internal/executor/task.go\ninternal/executor/wave.go\n",
	}
	task := models.Task{
		Number: "1",
		Files:  []string{"internal/executor/task.go"}, // Only declared task.go, but wave.go is same package
	}

	result, err := EnforcePackageIsolation(context.Background(), runner, task)
	if err != nil {
		t.Fatalf("expected no error for same package, got: %v", err)
	}
	if !result.Passed {
		t.Error("expected Passed to be true for same package modifications")
	}
}

func TestEnforcePackageIsolation_NoFiles(t *testing.T) {
	runner := &mockGitRunner{
		diffOutput: "internal/executor/task.go\n",
	}
	task := models.Task{
		Number: "1",
		Files:  []string{}, // No files declared
	}

	result, err := EnforcePackageIsolation(context.Background(), runner, task)
	if err != nil {
		t.Fatalf("expected no error for task with no files, got: %v", err)
	}
	if !result.Passed {
		t.Error("expected Passed for task with no declared files")
	}
}

func TestEnforcePackageIsolation_NoDiff(t *testing.T) {
	runner := &mockGitRunner{
		diffOutput: "", // No modifications
	}
	task := models.Task{
		Number: "1",
		Files:  []string{"internal/executor/task.go"},
	}

	result, err := EnforcePackageIsolation(context.Background(), runner, task)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Passed {
		t.Error("expected Passed when no files modified")
	}
}

func TestEnforcePackageIsolation_StagedChanges(t *testing.T) {
	// Staged changes in undeclared file should trigger violation
	runner := &mockGitRunner{
		diffOutput:   "internal/executor/task.go\n",
		cachedOutput: "internal/parser/yaml.go\n", // Staged change outside scope
	}
	task := models.Task{
		Number: "1",
		Files:  []string{"internal/executor/task.go"},
	}

	result, err := EnforcePackageIsolation(context.Background(), runner, task)
	if err == nil {
		t.Fatal("expected error for staged undeclared file")
	}
	if result.Passed {
		t.Error("expected Passed to be false")
	}
}

func TestEnforcePackageIsolation_NonGoFiles(t *testing.T) {
	// Non-Go files must be explicitly declared
	runner := &mockGitRunner{
		diffOutput: "docs/README.md\nconfig.yaml\n",
	}
	task := models.Task{
		Number: "1",
		Files:  []string{"docs/README.md"}, // Only README declared
	}

	result, err := EnforcePackageIsolation(context.Background(), runner, task)
	if err == nil {
		t.Fatal("expected error for undeclared non-Go file")
	}
	if len(result.UndeclaredFiles) != 1 || result.UndeclaredFiles[0] != "config.yaml" {
		t.Errorf("expected undeclared 'config.yaml', got: %v", result.UndeclaredFiles)
	}
}

func TestParseGitDiffOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single file",
			input:    "internal/executor/task.go\n",
			expected: []string{"internal/executor/task.go"},
		},
		{
			name:     "multiple files",
			input:    "a.go\nb.go\nc.go\n",
			expected: []string{"a.go", "b.go", "c.go"},
		},
		{
			name:     "empty",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "  \n\t\n  \n",
			expected: nil,
		},
		{
			name:     "trailing newline",
			input:    "file.go\n",
			expected: []string{"file.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGitDiffOutput(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d files, got %d: %v", len(tt.expected), len(result), result)
			}
			for i := range result {
				if i < len(tt.expected) && result[i] != tt.expected[i] {
					t.Errorf("file %d: expected %q, got %q", i, tt.expected[i], result[i])
				}
			}
		})
	}
}
