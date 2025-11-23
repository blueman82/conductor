# Cross-File Dependency Implementation Guide

## Quick Reference: What to Implement

This document provides concrete code snippets and implementation patterns for the cross-file dependency feature.

---

## Part 1: Models (internal/models/task.go)

### 1.1 New Type Definitions

Add these types before the Task struct:

```go
// DependencyReference represents a single dependency that can be local or cross-file
type DependencyReference struct {
	// Type distinguishes between local and cross-file references
	// Values: "local" (default) or "cross-file"
	Type string `json:"type" yaml:"type"`

	// For local dependencies (Type="local"):
	// TaskNumber is the number of the task in the same file
	TaskNumber string `json:"task_number,omitempty" yaml:"task_number,omitempty"`

	// For cross-file dependencies (Type="cross-file"):
	// File is the relative or absolute path to the plan file containing the task
	File string `json:"file,omitempty" yaml:"file,omitempty"`

	// Task is the task number in the referenced file (cross-file only)
	Task string `json:"task,omitempty" yaml:"task,omitempty"`
}

// NormalizedDependency represents a fully resolved dependency
// Used internally after all references are converted to absolute paths
type NormalizedDependency struct {
	// LocalTaskNumber is the normalized task number (local reference)
	// Set after cross-file resolution
	LocalTaskNumber string

	// SourceFile is the absolute path to the source file containing this dependency
	// Set for cross-file dependencies; empty for local references
	SourceFile string

	// OriginalFormat tracks the original format for error messages
	OriginalFormat string // "numeric", "structured", "cross-file"
}
```

### 1.2 Task Struct Extensions

Add these fields to the Task struct (around line 13-22):

```go
type Task struct {
	Number        string                 // Task number/identifier
	Name          string                 // Task name/title
	Files         []string               // Files to be modified/created
	DependsOn     []string               // Task numbers this task depends on (local)

	// NEW FIELDS FOR CROSS-FILE SUPPORT
	// DependsOnRaw stores raw dependency data before parsing
	// Used internally during parsing to capture both formats
	// Cleared after normalization for memory efficiency
	DependsOnRaw  []interface{} `json:"-" yaml:"-"`

	// CrossFileDependencies maps cross-file references after resolution
	// Key: "<file>:<task_number>", Value: resolved task number
	// Example: {"/abs/path/plan-01.yaml:2": "2"}
	CrossFileDependencies map[string]string `json:"-" yaml:"-"`

	// DependencyReferences stores the parsed dependency objects
	// Only populated during parsing; converted to DependsOn for execution
	DependencyReferences []DependencyReference `json:"-" yaml:"-"`

	// ... rest of existing fields unchanged ...
	EstimatedTime time.Duration
	Agent         string
	Prompt        string
	WorktreeGroup string
	Status        string
	StartedAt     *time.Time
	CompletedAt   *time.Time
	SourceFile    string
	Metadata      map[string]interface{}
	SuccessCriteria     []string
	TestCommands        []string
	Type                string
	IntegrationCriteria []string
	ExecutionStartTime  time.Time
	ExecutionEndTime    time.Time
	ExecutionDuration   time.Duration
	ExecutedBy          string
	FilesModified       int
	FilesCreated        int
	FilesDeleted        int
}
```

### 1.3 Task Helper Methods

Add these methods to the Task type:

```go
// HasCrossFileDependencies returns true if this task has any cross-file dependencies
func (t *Task) HasCrossFileDependencies() bool {
	return len(t.CrossFileDependencies) > 0
}

// GetAllDependencies returns the union of local and resolved cross-file dependencies
// Used for dependency graph building after merge
func (t *Task) GetAllDependencies() []string {
	result := make([]string, len(t.DependsOn))
	copy(result, t.DependsOn)

	// Add resolved cross-file dependencies (already in DependsOn after merge)
	// This is a helper for clarity; after merge, all deps are in DependsOn
	return result
}

// NormalizedDependsOn returns all dependencies (local + resolved cross-file)
// This is the definitive source for dependency graph building
// Backward compatible: returns same as DependsOn for code using that field
func (t *Task) NormalizedDependsOn() []string {
	return t.DependsOn
}
```

---

## Part 2: YAML Parser (internal/parser/yaml.go)

### 2.1 Modify yamlTask Struct

Update the yamlTask struct to preserve raw dependency data:

```go
// yamlTask represents a single task in the YAML plan
type yamlTask struct {
	TaskNumber          interface{}   `yaml:"task_number"`
	Name                string        `yaml:"name"`
	Files               []string      `yaml:"files"`
	// CHANGED: DependsOn now accepts mixed formats (numeric, structured, cross-file)
	DependsOn           []interface{} `yaml:"depends_on"`
	EstimatedTime       string        `yaml:"estimated_time"`
	Agent               string        `yaml:"agent"`
	WorktreeGroup       string        `yaml:"worktree_group"`
	Status              string        `yaml:"status"`
	CompletedDate       string        `yaml:"completed_date"`
	CompletedAt         string        `yaml:"completed_at"`
	Description         string        `yaml:"description"`
	SuccessCriteria     []string      `yaml:"success_criteria"`
	TestCommands        []string      `yaml:"test_commands"`
	Type                string        `yaml:"type"`
	IntegrationCriteria []string      `yaml:"integration_criteria"`
	// ... rest of fields unchanged ...
}
```

### 2.2 Add Helper Functions

Add these functions to yaml.go:

```go
// isValidTaskNumber checks if a task number string is valid
// Valid: alphanumeric, hyphens, underscores; no special characters
func isValidTaskNumber(taskNum string) bool {
	if taskNum == "" {
		return false
	}

	for _, r := range taskNum {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') || r == '-' || r == '_') {
			return false
		}
	}

	return true
}

// normalizeFilePath resolves a file path relative to the current file's directory
// Handles both relative paths and rejects absolute paths for security
func normalizeFilePath(filePath, currentFile string) (string, error) {
	// Validate file path is not empty
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	// Reject absolute paths for security (relative to current file)
	if filepath.IsAbs(filePath) {
		return "", fmt.Errorf("absolute paths not allowed, use relative paths: %q", filePath)
	}

	// Resolve relative to current file's directory
	dir := filepath.Dir(currentFile)
	resolved := filepath.Join(dir, filePath)

	// Normalize and return
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	return abs, nil
}

// parseDependencies converts raw dependency data to normalized DependsOn list
// Supports: numeric, structured local, cross-file, and mixed formats
func parseDependencies(rawDeps []interface{}, task *models.Task, currentFile string) error {
	var localDeps []string
	crossFileDeps := make(map[string]string)

	for i, dep := range rawDeps {
		switch d := dep.(type) {
		// Case 1 & 2: Numeric or simple task reference
		case int, int64, float32, float64, string:
			depStr, err := convertToString(d)
			if err != nil {
				return fmt.Errorf("depends_on[%d]: invalid numeric reference: %w", i, err)
			}

			// Validate numeric format
			if !isValidTaskNumber(depStr) {
				return fmt.Errorf("depends_on[%d]: invalid task number format %q", i, depStr)
			}
			localDeps = append(localDeps, depStr)

		// Case 3 & 4: Structured dependency (map)
		case map[string]interface{}:
			// Check if it's a cross-file reference
			if file, hasFile := d["file"].(string); hasFile {
				// Cross-file dependency format: {file: "plan-01.yaml", task: ...}
				if taskRef, hasTask := d["task"]; hasTask {
					taskStr, err := convertToString(taskRef)
					if err != nil {
						return fmt.Errorf("depends_on[%d]: cross-file reference has invalid task: %w", i, err)
					}

					if !isValidTaskNumber(taskStr) {
						return fmt.Errorf("depends_on[%d]: cross-file reference has invalid task number %q", i, taskStr)
					}

					// Normalize file path (relative to current file's directory)
					normalizedFile, err := normalizeFilePath(file, currentFile)
					if err != nil {
						return fmt.Errorf("depends_on[%d]: invalid cross-file reference path: %w", i, err)
					}

					// Store cross-file dependency
					// Key format: "normalized-file-path:task-number"
					key := fmt.Sprintf("%s:%s", normalizedFile, taskStr)
					crossFileDeps[key] = "" // Value will be filled during merge
				} else {
					return fmt.Errorf("depends_on[%d]: cross-file reference missing required 'task' field", i)
				}
			} else if taskRef, hasTask := d["task"].(interface{}); hasTask {
				// Local structured format: {task: 2}
				taskStr, err := convertToString(taskRef)
				if err != nil {
					return fmt.Errorf("depends_on[%d]: structured reference has invalid task: %w", i, err)
				}

				if !isValidTaskNumber(taskStr) {
					return fmt.Errorf("depends_on[%d]: structured reference has invalid task number %q", i, taskStr)
				}
				localDeps = append(localDeps, taskStr)
			} else {
				return fmt.Errorf("depends_on[%d]: unrecognized structured dependency format, expected {task: N} or {file: 'F', task: N}", i)
			}

		default:
			return fmt.Errorf("depends_on[%d]: unsupported type %T", i, d)
		}
	}

	task.DependsOn = localDeps
	task.CrossFileDependencies = crossFileDeps
	task.DependsOnRaw = rawDeps // Keep raw for debugging
	return nil
}
```

### 2.3 Update YAMLParser.Parse()

Replace the dependency parsing section (around line 153-160):

```go
// In YAMLParser.Parse(), around line 147-160:

for i, yt := range yp.Plan.Tasks {
	taskNum, err := convertToString(yt.TaskNumber)
	if err != nil {
		return nil, fmt.Errorf("task %d: invalid task_number: %w", i+1, err)
	}

	// NEW: Parse mixed dependency formats
	if err := parseDependencies(yt.DependsOn, &task, plan.FilePath); err != nil {
		return nil, fmt.Errorf("task %s: %w", taskNum, err)
	}

	// REMOVE old code:
	// dependsOn := make([]string, 0, len(yt.DependsOn))
	// for j, dep := range yt.DependsOn {
	//   depStr, err := convertToString(dep)
	//   ...
	// }
	// task.DependsOn = dependsOn

	task := models.Task{
		Number:              taskNum,
		Name:                yt.Name,
		Files:               yt.Files,
		// DependsOn is now set by parseDependencies()
		Agent:               yt.Agent,
		WorktreeGroup:       yt.WorktreeGroup,
		Status:              yt.Status,
		SuccessCriteria:     yt.SuccessCriteria,
		TestCommands:        yt.TestCommands,
		Type:                yt.Type,
		IntegrationCriteria: yt.IntegrationCriteria,
	}

	// ... rest of task construction ...
}
```

---

## Part 3: Plan Merging (internal/parser/parser.go)

### 3.1 Update MergePlans Function

Replace the existing MergePlans function:

```go
// MergePlans combines multiple plans into a single plan
// while preserving all task dependencies and resolving cross-file references.
//
// Process:
// 1. Validate no duplicate task numbers across all files
// 2. Build file-to-task mapping for cross-file reference resolution
// 3. Validate all cross-file references can be resolved
// 4. Convert cross-file references to local task numbers
// 5. Return merged plan ready for execution
func MergePlans(plans ...*models.Plan) (*models.Plan, error) {
	if len(plans) == 0 {
		return &models.Plan{Tasks: []models.Task{}}, nil
	}

	// Step 1: Collect all tasks and build dedup map
	seen := make(map[string]bool)
	var mergedTasks []models.Task

	// Map of "file:task_number" -> actual task number
	// Used for resolving cross-file references
	fileToTask := make(map[string]string)

	for _, plan := range plans {
		if plan == nil {
			continue
		}

		for _, task := range plan.Tasks {
			if seen[task.Number] {
				return nil, fmt.Errorf("duplicate task number: %s", task.Number)
			}
			seen[task.Number] = true

			// Set SourceFile to track which plan file this task comes from
			task.SourceFile = plan.FilePath

			// Build mapping for cross-file reference resolution
			// Key format: "absolute-file-path:task-number"
			fileToTaskKey := fmt.Sprintf("%s:%s", plan.FilePath, task.Number)
			fileToTask[fileToTaskKey] = task.Number

			mergedTasks = append(mergedTasks, task)
		}
	}

	// Step 2: Validate and resolve cross-file dependencies
	for i, task := range mergedTasks {
		for crossFileRef := range task.CrossFileDependencies {
			// crossFileRef format: "/absolute/path/plan-01.yaml:2"

			if taskNum, exists := fileToTask[crossFileRef]; !exists {
				// Parse reference for better error message
				parts := strings.Split(crossFileRef, ":")
				if len(parts) == 2 {
					return nil, fmt.Errorf("task %s (%s): cross-file reference to task %s in %s not found",
						task.Number, task.Name, parts[1], filepath.Base(parts[0]))
				}
				return nil, fmt.Errorf("task %s (%s): invalid cross-file reference %q",
					task.Number, task.Name, crossFileRef)
			} else {
				// Resolve: add cross-file task to DependsOn
				mergedTasks[i].DependsOn = append(mergedTasks[i].DependsOn, taskNum)

				// Update CrossFileDependencies with resolved value
				mergedTasks[i].CrossFileDependencies[crossFileRef] = taskNum
			}
		}
	}

	// Step 3: Build result plan
	var result *models.Plan
	for _, plan := range plans {
		if plan != nil {
			result = &models.Plan{
				Name:           plan.Name,
				DefaultAgent:   plan.DefaultAgent,
				QualityControl: plan.QualityControl,
				WorktreeGroups: plan.WorktreeGroups,
				Tasks:          mergedTasks,
				FilePath:       plan.FilePath,
				FileToTaskMap:  fileToTask, // Store mapping for reference
			}
			break
		}
	}

	if result == nil {
		result = &models.Plan{Tasks: mergedTasks}
	}

	return result, nil
}
```

---

## Part 4: Graph Validation (internal/executor/graph.go)

### 4.1 Update ValidateTasks

Update the ValidateTasks function to use normalized dependencies:

```go
// ValidateTasks checks that all task dependencies are valid
func ValidateTasks(tasks []models.Task) error {
	taskMap := make(map[string]bool)
	for _, task := range tasks {
		if task.Number == "" {
			return fmt.Errorf("task has empty task number")
		}
		if taskMap[task.Number] {
			return fmt.Errorf("task %s: duplicate task number", task.Number)
		}
		taskMap[task.Number] = true
	}

	// Validate all dependencies (including resolved cross-file deps)
	for _, task := range tasks {
		// Use NormalizedDependsOn() to get all dependencies
		allDeps := task.NormalizedDependsOn()
		for _, dep := range allDeps {
			if !taskMap[dep] {
				return fmt.Errorf("task %s (%s): depends on non-existent task %s",
					task.Number, task.Name, dep)
			}
		}
	}

	return nil
}
```

### 4.2 Update BuildDependencyGraph

Update to use NormalizedDependsOn():

```go
// BuildDependencyGraph constructs a dependency graph from a list of tasks
// Handles both local and resolved cross-file dependencies
func BuildDependencyGraph(tasks []models.Task) *DependencyGraph {
	g := &DependencyGraph{
		Tasks:    make(map[string]*models.Task),
		Edges:    make(map[string][]string),
		InDegree: make(map[string]int),
		Groups:   make(map[int]string),
	}

	// Build task map and initialize in-degree
	for i := range tasks {
		g.Tasks[tasks[i].Number] = &tasks[i]
		g.InDegree[tasks[i].Number] = 0

		// Tag node with group info
		if taskNum, err := strconv.Atoi(tasks[i].Number); err == nil {
			g.Groups[taskNum] = tasks[i].WorktreeGroup
		}
	}

	// Build edges using normalized dependency list
	for _, task := range tasks {
		// Use NormalizedDependsOn() to get all dependencies (local + resolved cross-file)
		allDeps := task.NormalizedDependsOn()

		for _, dep := range allDeps {
			// Validate that dependency exists
			if _, exists := g.Tasks[dep]; !exists {
				// Skip invalid dependencies - caught by validation
				continue
			}
			// dep -> task (dep must complete before task)
			g.Edges[dep] = append(g.Edges[dep], task.Number)
			g.InDegree[task.Number]++
		}
	}

	return g
}
```

---

## Part 5: Test Data Examples

### 5.1 Test Plan Files

**File: testdata/split-plan-cross-file-1.yaml**
```yaml
conductor:
  default_agent: golang-pro

plan:
  metadata:
    feature_name: "Foundation Tasks"
  tasks:
    - task_number: 1
      name: "Setup Database"
      files: [internal/db/setup.go]
      depends_on: []

    - task_number: 2
      name: "Create Migrations"
      files: [migrations/001_init.sql]
      depends_on: [1]
```

**File: testdata/split-plan-cross-file-2.yaml**
```yaml
plan:
  metadata:
    feature_name: "Feature Tasks"
  tasks:
    - task_number: 3
      name: "User API"
      files: [internal/api/users.go]
      depends_on:
        - file: "split-plan-cross-file-1.yaml"
          task: 2
        - 2  # This would typically reference a local task, or be ignored if not in same file

    - task_number: 4
      name: "Product API"
      files: [internal/api/products.go]
      depends_on:
        - file: "split-plan-cross-file-1.yaml"
          task: 1
        - 3
```

### 5.2 Test Cases

Create new test file: `internal/parser/cross_file_yaml_test.go`

```go
package parser

import (
	"testing"
	"github.com/harrison/conductor/internal/models"
)

func TestParseDependencies_NumericOnly(t *testing.T) {
	task := &models.Task{
		Number: "2",
		CrossFileDependencies: make(map[string]string),
	}
	rawDeps := []interface{}{1, 2, 3}

	err := parseDependencies(rawDeps, task, "/home/user/plan-01.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(task.DependsOn) != 3 {
		t.Errorf("expected 3 local deps, got %d", len(task.DependsOn))
	}
	if len(task.CrossFileDependencies) != 0 {
		t.Errorf("expected 0 cross-file deps, got %d", len(task.CrossFileDependencies))
	}
}

func TestParseDependencies_MixedFormat(t *testing.T) {
	task := &models.Task{
		Number: "3",
		CrossFileDependencies: make(map[string]string),
	}
	rawDeps := []interface{}{
		1, // numeric
		map[string]interface{}{
			"file": "plan-01.yaml",
			"task": 2,
		}, // cross-file
	}

	err := parseDependencies(rawDeps, task, "/home/user/plans/plan-02.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(task.DependsOn) != 1 || task.DependsOn[0] != "1" {
		t.Errorf("expected local dep [1], got %v", task.DependsOn)
	}
	if len(task.CrossFileDependencies) != 1 {
		t.Errorf("expected 1 cross-file dep, got %d", len(task.CrossFileDependencies))
	}
}

func TestParseDependencies_InvalidTaskNumber(t *testing.T) {
	task := &models.Task{
		Number: "2",
		CrossFileDependencies: make(map[string]string),
	}
	rawDeps := []interface{}{
		map[string]interface{}{
			"task": "2@invalid",
		},
	}

	err := parseDependencies(rawDeps, task, "/home/user/plan.yaml")
	if err == nil {
		t.Fatal("expected error for invalid task number")
	}
	if !strings.Contains(err.Error(), "invalid task number") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNormalizeFilePath_Relative(t *testing.T) {
	got, err := normalizeFilePath("plan-02.yaml", "/home/user/plans/plan-01.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "/home/user/plans/plan-02.yaml"
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestNormalizeFilePath_RejectAbsolute(t *testing.T) {
	_, err := normalizeFilePath("/absolute/path.yaml", "/home/user/plan.yaml")
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
	if !strings.Contains(err.Error(), "absolute paths not allowed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMergePlans_CrossFileResolution(t *testing.T) {
	// Plan 1: Task 1, 2
	plan1 := &models.Plan{
		FilePath: "/home/user/plan-01.yaml",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", DependsOn: []string{}},
			{Number: "2", Name: "Task 2", DependsOn: []string{"1"}},
		},
	}

	// Plan 2: Task 3 with cross-file dep to Task 2
	plan2 := &models.Plan{
		FilePath: "/home/user/plan-02.yaml",
		Tasks: []models.Task{
			{
				Number: "3",
				Name: "Task 3",
				DependsOn: []string{},
				CrossFileDependencies: map[string]string{
					"/home/user/plan-01.yaml:2": "",
				},
			},
		},
	}

	merged, err := MergePlans(plan1, plan2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find task 3
	var task3 *models.Task
	for i := range merged.Tasks {
		if merged.Tasks[i].Number == "3" {
			task3 = &merged.Tasks[i]
			break
		}
	}

	if task3 == nil {
		t.Fatal("task 3 not found after merge")
	}

	// Check that cross-file dep was resolved
	if len(task3.DependsOn) != 1 || task3.DependsOn[0] != "2" {
		t.Errorf("expected DependsOn=[2], got %v", task3.DependsOn)
	}
}

func TestMergePlans_UnresolvedCrossFile(t *testing.T) {
	plan1 := &models.Plan{
		FilePath: "/home/user/plan-01.yaml",
		Tasks: []models.Task{
			{Number: "1", Name: "Task 1", DependsOn: []string{}},
		},
	}

	plan2 := &models.Plan{
		FilePath: "/home/user/plan-02.yaml",
		Tasks: []models.Task{
			{
				Number: "2",
				Name: "Task 2",
				DependsOn: []string{},
				CrossFileDependencies: map[string]string{
					"/home/user/plan-01.yaml:999": "", // Non-existent task
				},
			},
		},
	}

	_, err := MergePlans(plan1, plan2)
	if err == nil {
		t.Fatal("expected error for unresolved cross-file ref")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}
```

---

## Part 6: Summary Checklist

### Before Implementation
- [ ] Review this design document thoroughly
- [ ] Understand the backward compatibility approach
- [ ] Review existing code structure for integration points

### Implementation Order
- [ ] Add types and structs to models/task.go
- [ ] Add helper methods to Task type
- [ ] Add helper functions to parser/yaml.go
- [ ] Update dependency parsing in YAMLParser.Parse()
- [ ] Update MergePlans() in parser/parser.go
- [ ] Update ValidateTasks() in executor/graph.go
- [ ] Update BuildDependencyGraph() in executor/graph.go
- [ ] Create test files with sample plans
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Test error messages for clarity
- [ ] Update documentation in CLAUDE.md

### Testing Validation
- [ ] Backward compatibility: existing plans still work
- [ ] Numeric dependencies: `depends_on: [2, 3, 4]`
- [ ] Mixed format: `depends_on: [2, {file: "...", task: 3}]`
- [ ] Error cases: invalid paths, missing tasks, cycles
- [ ] Single-file plans: cross-file refs are warned but not enforced
- [ ] Multi-file merge: cross-file refs are resolved correctly

---

## Implementation Notes

1. **Memory Efficiency**: DependsOnRaw is cleared after parsing. CrossFileDependencies is merged into DependsOn during MergePlans(), so it's not needed in execution.

2. **Backward Compatibility**: The DependsOn field is never changed in format; it always contains string task numbers. New code paths only activate when parsing structured dependencies.

3. **Error Messages**: Each error includes:
   - What went wrong (e.g., "invalid task number format")
   - The specific input (e.g., "2@invalid")
   - What's valid (e.g., "Valid task numbers contain only alphanumeric characters, hyphens, and underscores")

4. **Performance**: File path normalization happens once during parsing. Cross-file resolution is O(1) lookup during merge.

5. **Testing**: Each function has clear unit tests with success and failure cases.
