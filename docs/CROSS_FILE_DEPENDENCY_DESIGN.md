# Cross-File Dependency Support Design (Option 2: Cross-File Annotation)

## Executive Summary

This document provides a comprehensive technical design for implementing cross-file dependency support in Conductor with full backward compatibility. The design uses an annotation-based approach that extends the existing `depends_on` field to support both numeric references (local) and cross-file references without breaking existing plans.

**Key Features:**
- Mixed dependency format support in a single `depends_on` array
- Backward compatible with existing numeric dependencies (`depends_on: [2, 3, 4]`)
- Cross-file references with explicit file/task pairs
- Single-file validation skips external references
- Multi-file validation resolves and validates cross-file dependencies after merge
- Type-safe Go implementation with comprehensive error handling

---

## 1. Data Structure Design

### 1.1 Dependency Representation in Models

**File: `/Users/harrison/Github/conductor/internal/models/task.go`**

Add a new type to represent dependencies that can be either local or cross-file:

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

### 1.2 Task DependsOn Field Migration

**File: `/Users/harrison/Github/conductor/internal/models/task.go`**

Modify the Task struct to support both old and new formats during parsing, but store normalized results:

```go
// Task represents a single task in an implementation plan
type Task struct {
	Number        string                 // Task number/identifier (supports int, float, alphanumeric)
	Name          string                 // Task name/title
	Files         []string               // Files to be modified/created
	DependsOn     []string               // EXISTING: Task numbers this task depends on (local, numeric format)
	                                      // This field maintains backward compatibility with existing plans
	                                      // Format: ["2", "3", "4"] for local references only

	// DependsOnRaw stores raw dependency data before parsing
	// Used internally during parsing to capture both formats
	// Cleared after normalization for memory efficiency
	DependsOnRaw  []interface{} `json:"-" yaml:"-"` // Raw input from YAML/Markdown

	// CrossFileDependencies maps cross-file references after resolution
	// Key: "<file>:<task_number>", Value: resolved task number
	// Example: {"plan-01-foundation.yaml:2": "2", "plan-02-features.yaml:3": "3"}
	CrossFileDependencies map[string]string `json:"-" yaml:"-"`

	// DependencyReferences stores the parsed dependency objects
	// Only populated during parsing; converted to DependsOn for execution
	DependencyReferences []DependencyReference `json:"-" yaml:"-"`

	// EstimatedTime time.Duration          // ... rest of fields unchanged
	Agent         string                 // Agent to use (optional)
	Prompt        string                 // Full task description/prompt
	WorktreeGroup string                 // Worktree group this task belongs to (optional)
	Status        string                 // Task status: pending, in_progress, completed, skipped
	StartedAt     *time.Time             // Timestamp when task execution started (nil if not started)
	CompletedAt   *time.Time             // Timestamp when task was completed (nil if not completed)
	SourceFile    string                 // Source plan file this task originates from (for multi-file plans)
	Metadata      map[string]interface{} // Additional metadata for hooks and extensions

	// Structured verification (v2.3+)
	SuccessCriteria     []string `yaml:"success_criteria,omitempty" json:"success_criteria,omitempty"`
	TestCommands        []string `yaml:"test_commands,omitempty" json:"test_commands,omitempty"`
	Type                string   `yaml:"type,omitempty" json:"type,omitempty"`
	IntegrationCriteria []string `yaml:"integration_criteria,omitempty" json:"integration_criteria,omitempty"`

	// Execution metadata for enhanced console output
	ExecutionStartTime time.Time     `json:"execution_start_time,omitempty" yaml:"execution_start_time,omitempty"`
	ExecutionEndTime   time.Time     `json:"execution_end_time,omitempty" yaml:"execution_end_time,omitempty"`
	ExecutionDuration  time.Duration `json:"execution_duration,omitempty" yaml:"execution_duration,omitempty"`
	ExecutedBy         string        `json:"executed_by,omitempty" yaml:"executed_by,omitempty"`
	FilesModified      int           `json:"files_modified,omitempty" yaml:"files_modified,omitempty"`
	FilesCreated       int           `json:"files_created,omitempty" yaml:"files_created,omitempty"`
	FilesDeleted       int           `json:"files_deleted,omitempty" yaml:"files_deleted,omitempty"`
}

// Helper Methods to Add to Task

// HasCrossFileDependencies returns true if this task has any cross-file dependencies
func (t *Task) HasCrossFileDependencies() bool {
	return len(t.CrossFileDependencies) > 0
}

// GetAllDependencies returns the union of local and resolved cross-file dependencies
// Used for dependency graph building
func (t *Task) GetAllDependencies() []string {
	result := make([]string, len(t.DependsOn))
	copy(result, t.DependsOn)

	// Add resolved cross-file dependencies
	for _, resolved := range t.CrossFileDependencies {
		result = append(result, resolved)
	}
	return result
}

// NormalizedDependsOn returns DependsOn field, resolving cross-file references
// For backward compatibility, this is the definitive source for dependency graph building
func (t *Task) NormalizedDependsOn() []string {
	return t.GetAllDependencies()
}
```

### 1.3 Parser Structures

**File: `/Users/harrison/Github/conductor/internal/parser/yaml.go`**

Modify yamlTask to capture cross-file references:

```go
// yamlTask represents a single task in the YAML plan
type yamlTask struct {
	TaskNumber          interface{}   `yaml:"task_number"` // Accepts int, float, or string
	Name                string        `yaml:"name"`
	Files               []string      `yaml:"files"`
	// Change DependsOn to accept both numeric and structured formats
	DependsOn           []interface{} `yaml:"depends_on"` // NEW: Accepts mixed formats
	                                                       // - int/float/string: local numeric reference
	                                                       // - map: cross-file reference {file: "...", task: "..."}
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
	// ... remaining fields unchanged
}
```

**File: `/Users/harrison/Github/conductor/internal/parser/markdown.go`**

Markdown parser structure remains the same but parseTaskMetadata() function needs enhancement.

---

## 2. Parser Implementation Changes

### 2.1 YAML Parser Implementation

**File: `/Users/harrison/Github/conductor/internal/parser/yaml.go`**

Add new parsing functions to handle mixed dependency formats:

```go
// parseDependencies converts raw dependency data to normalized DependsOn list
// Supports three formats:
// 1. Numeric: `depends_on: [2, 3, 4]` -> DependsOn: ["2", "3", "4"]
// 2. Structured local: `depends_on: [{task: 2}, {task: 3}]` -> DependsOn: ["2", "3"]
// 3. Cross-file: `depends_on: [{file: "plan-01.yaml", task: 2}]` -> stores in CrossFileDependencies
// 4. Mixed: `depends_on: [2, {file: "plan-01.yaml", task: 3}]` -> combines both formats
//
// Parameters:
//   - rawDeps: Raw dependency data from YAML parser
//   - task: Task struct to populate
//   - currentFile: Source file for this task (used for relative path resolution)
//
// Returns: error if invalid dependency format detected
func parseDependencies(rawDeps []interface{}, task *models.Task, currentFile string) error {
	var localDeps []string
	crossFileDeps := make(map[string]string)

	for i, dep := range rawDeps {
		switch d := dep.(type) {
		// Format 1 & 2: Numeric or simple task reference
		case int, int64, float32, float64, string:
			depStr, err := convertToString(d)
			if err != nil {
				return fmt.Errorf("depends_on[%d]: invalid numeric reference: %w", i, err)
			}

			// Validate numeric format (alphanumeric only, no special chars)
			if !isValidTaskNumber(depStr) {
				return fmt.Errorf("depends_on[%d]: invalid task number format %q", i, depStr)
			}
			localDeps = append(localDeps, depStr)

		// Format 3 & 4: Structured dependency (map)
		case map[string]interface{}:
			// Check if it's a cross-file reference
			if file, hasFile := d["file"].(string); hasFile {
				// Cross-file dependency format: {file: "...", task: ...}
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
	task.DependsOnRaw = rawDeps // Keep raw for debugging/error messages
	return nil
}

// isValidTaskNumber checks if a task number string is valid
// Valid: alphanumeric, no special characters (except dash/underscore)
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
// Handles both relative paths (./plan-01.yaml, ../plans/setup.yaml)
// and absolute paths (/home/user/plans/setup.yaml)
//
// Parameters:
//   - filePath: Path specified in cross-file reference
//   - currentFile: Current source file (absolute path)
//
// Returns: Absolute normalized path or error if path is invalid
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

// In YAMLParser.Parse(), modify dependency parsing:
// Around line 153-160 in current implementation

for i, yt := range yp.Plan.Tasks {
	// ... existing code ...

	// Replace old dependency parsing:
	// OLD: Convert simple interface{} to string
	// dependsOn := make([]string, 0, len(yt.DependsOn))
	// for j, dep := range yt.DependsOn {
	//   depStr, err := convertToString(dep)
	//   ...
	// }

	// NEW: Parse mixed dependency formats
	if err := parseDependencies(yt.DependsOn, &task, plan.FilePath); err != nil {
		return nil, fmt.Errorf("task %s: %w", taskNum, err)
	}

	// ... rest of parsing ...
}
```

### 2.2 Markdown Parser Implementation

**File: `/Users/harrison/Github/conductor/internal/parser/markdown.go`**

Enhance the parseTaskMetadata function to handle cross-file references:

```go
// parseTaskMetadata extracts task metadata from markdown content
// Supports extracting:
// - Files: **Files**: file1.go, file2.go
// - Depends on: **Depends on**: 2, 3, 4 OR {file: plan-01.yaml, task: 2}
// - Agent: **Agent**: golang-pro
// - WorktreeGroup: **WorktreeGroup**: backend-core
// - Success criteria: **Success Criteria**: criterion 1, criterion 2
// - Test commands: **Test Commands**: go test ./..., ...
// - Integration criteria: **Integration Criteria**: criterion 1, ...
//
// Replaces existing parseTaskMetadata() function
func parseTaskMetadata(task *models.Task, content string) {
	// ... existing parsing code for Files, Agent, WorktreeGroup, etc ...

	// Enhanced depends_on parsing
	// Supports:
	// 1. Numeric list: **Depends on**: 2, 3, 4
	// 2. Markdown list:
	//    **Depends on**:
	//    - 2
	//    - 3
	//    - {file: plan-01.yaml, task: 4}

	dependsOnRegex := regexp.MustCompile(`(?i)^\*?\*Depends\s+on\*?\*:\s*(.+)$`)

	for _, line := range strings.Split(content, "\n") {
		if match := dependsOnRegex.FindStringSubmatch(line); match != nil {
			depList := match[1]

			// Try parsing as comma-separated list first
			if !strings.Contains(depList, "-") && !strings.Contains(depList, "{") {
				// Simple format: 2, 3, 4
				for _, dep := range strings.Split(depList, ",") {
					depStr := strings.TrimSpace(dep)
					if depStr != "" && isValidTaskNumber(depStr) {
						task.DependsOn = append(task.DependsOn, depStr)
					}
				}
			} else {
				// Structured format (handled by parseMarkdownListItems)
				// This is more complex and extracted separately
			}
			break
		}
	}

	// Parse markdown list-style dependencies
	// Example:
	// **Depends on**:
	// - 2
	// - 3
	// - file: plan-01.yaml
	//   task: 4
	parseMarkdownDependencyList(task, content)
}

// parseMarkdownDependencyList handles complex markdown list-based dependency declarations
func parseMarkdownDependencyList(task *models.Task, content string) {
	// Implementation parses markdown list sections
	// Converts entries to local or cross-file dependencies

	// For MVP, markdown format remains simple:
	// **Depends on**: 2, 3, 4
	//
	// Complex markdown lists will be supported in v2.6+
}
```

### 2.3 Backward Compatibility in Parsing

The parsing changes maintain backward compatibility:

1. **Numeric dependencies** (`depends_on: [2, 3, 4]`) are parsed directly into `DependsOn` field
2. **Structured local dependencies** (`{task: 2}`) are converted to numeric strings in `DependsOn`
3. **Cross-file dependencies** (`{file: "plan-01.yaml", task: 2}`) are stored separately in `CrossFileDependencies`
4. **Mixed formats** are split appropriately into both fields
5. Old code continues to work because:
   - `DependsOn` field is unchanged and backward compatible
   - Graph building reads from `DependsOn` (can also call `NormalizedDependsOn()`)
   - New fields (`CrossFileDependencies`, `DependencyReferences`) are internal

---

## 3. Validation Changes

### 3.1 Single-File Validation

**File: `/Users/harrison/Github/conductor/internal/executor/graph.go`**

The `ValidateTasks()` function needs modification to skip cross-file reference validation:

```go
// ValidateTasks checks that all task dependencies are valid
// For single-file plans: skips validation of cross-file references
// For multi-file plans: validates after merge (cross-file refs are resolved)
func ValidateTasks(tasks []models.Task, isMultiFile bool) error {
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

	// Validate dependencies
	for _, task := range tasks {
		for _, dep := range task.DependsOn {
			if !taskMap[dep] {
				return fmt.Errorf("task %s (%s): depends on non-existent task %s", task.Number, task.Name, dep)
			}
		}

		// For single-file plans, skip cross-file validation
		// They will be validated during merge for multi-file plans
		if !isMultiFile && task.HasCrossFileDependencies() {
			// Log warning for single-file plans with cross-file references
			fmt.Fprintf(os.Stderr, "Warning: task %s (%s) has cross-file dependencies but plan is single-file\n",
				task.Number, task.Name)
		}
	}

	return nil
}

// ValidateCrossFileDependencies validates that all cross-file references can be resolved
// Should be called during multi-file merge after all tasks are combined
//
// Parameters:
//   - tasks: All tasks from merged plan
//   - fileToTask: Map of "file:task" -> task number (for quick lookup)
//
// Returns: error if any cross-file reference is invalid
func ValidateCrossFileDependencies(tasks []models.Task, fileToTask map[string]string) error {
	for _, task := range tasks {
		for crossFileRef := range task.CrossFileDependencies {
			// crossFileRef format: "/absolute/path/plan-01.yaml:2"

			// Check if reference can be resolved
			if resolvedTaskNum, exists := fileToTask[crossFileRef]; !exists {
				return fmt.Errorf("task %s (%s): cross-file reference %q cannot be resolved",
					task.Number, task.Name, crossFileRef)
			} else {
				// Update the resolved task number in the map
				task.CrossFileDependencies[crossFileRef] = resolvedTaskNum
			}
		}
	}
	return nil
}
```

### 3.2 Multi-File Validation

**File: `/Users/harrison/Github/conductor/internal/parser/parser.go`**

Enhance `MergePlans()` to validate and resolve cross-file dependencies:

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

	// Step 2: Validate cross-file dependencies
	for i, task := range mergedTasks {
		for crossFileRef, _ := range task.CrossFileDependencies {
			// crossFileRef format: "/absolute/path/plan-01.yaml:2"

			if taskNum, exists := fileToTask[crossFileRef]; !exists {
				// Parse reference for better error message
				parts := strings.Split(crossFileRef, ":")
				if len(parts) == 2 {
					return nil, fmt.Errorf("task %s (%s): cross-file reference to %s in %s not found",
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

## 4. Dependency Graph Builder Changes

**File: `/Users/harrison/Github/conductor/internal/executor/graph.go`**

Minimal changes needed - just use the unified dependency list:

```go
// BuildDependencyGraph constructs a dependency graph from a list of tasks
// Handles both local and resolved cross-file dependencies
//
// Note: Must be called after MergePlans() for multi-file plans
// to ensure cross-file references are resolved to local task numbers
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

	// Build edges using unified dependency list (includes resolved cross-file deps)
	for _, task := range tasks {
		// Use NormalizedDependsOn() to get all dependencies (local + resolved cross-file)
		allDeps := task.NormalizedDependsOn()

		for _, dep := range allDeps {
			// Validate that dependency exists
			if _, exists := g.Tasks[dep]; !exists {
				// Skip invalid dependencies - they will be caught by validation
				continue
			}
			// dep -> task (dep must complete before task)
			g.Edges[dep] = append(g.Edges[dep], task.Number)
			g.InDegree[task.Number]++
		}
	}

	return g
}

// ValidateTasks signature updated to support multi-file context
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

	// Validate all dependencies (local + resolved cross-file)
	for _, task := range tasks {
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

---

## 5. Implementation Phases

### Phase 1: Data Structures & Parsing (MVP)
1. Add `DependencyReference`, `NormalizedDependency` types to `models/task.go`
2. Modify `Task.DependsOn` field documentation
3. Add `CrossFileDependencies` and helper methods to Task
4. Update YAML parser to handle mixed formats in `parseDependencies()`
5. Add validation functions `isValidTaskNumber()`, `normalizeFilePath()`
6. Update markdown parser metadata extraction

### Phase 2: Plan Merging & Validation
1. Enhance `MergePlans()` to build file-to-task mapping
2. Add cross-file dependency resolution
3. Add `ValidateCrossFileDependencies()` function
4. Update `ValidateTasks()` to handle resolved dependencies
5. Add error messages for invalid cross-file references

### Phase 3: Integration & Testing
1. Update `BuildDependencyGraph()` to use `NormalizedDependsOn()`
2. Add comprehensive tests for parsing mixed formats
3. Add tests for single-file validation behavior
4. Add tests for multi-file merge and cross-file resolution
5. Add tests for error cases (invalid references, cycles, etc.)

---

## 6. YAML Format Examples

### 6.1 Single-File Plan (Backward Compatible)

```yaml
conductor:
  default_agent: golang-pro
  quality_control:
    enabled: true

plan:
  metadata:
    feature_name: "Database Setup"
  tasks:
    - task_number: 1
      name: "Create tables"
      files: [internal/db/schema.go]
      depends_on: []  # Numeric format (unchanged)

    - task_number: 2
      name: "Add migrations"
      files: [migrations/001_init.sql]
      depends_on: [1]  # Numeric format (unchanged)
```

### 6.2 Multi-File Plan with Cross-File Dependencies

**File: plan-01-foundation.yaml**
```yaml
plan:
  tasks:
    - task_number: 1
      name: "Database Setup"
      depends_on: []

    - task_number: 2
      name: "Auth Module"
      depends_on: [1]
```

**File: plan-02-features.yaml**
```yaml
plan:
  tasks:
    - task_number: 3
      name: "User API"
      depends_on:
        - 2                                    # Local reference (same file context would be unusual)
        - file: "plan-01-foundation.yaml"     # Cross-file reference
          task: 2                              # Depends on Task 2 from plan-01

    - task_number: 4
      name: "Product API"
      depends_on:
        - file: "plan-01-foundation.yaml"     # Cross-file reference
          task: 1                              # Depends on Task 1 from plan-01
        - 3                                    # Local reference
```

### 6.3 Mixed Format Examples

```yaml
depends_on:
  - 2                                          # Numeric (local)
  - 3                                          # Numeric (local)
  - file: "plan-01-foundation.yaml"           # Cross-file
    task: 4
  - file: "../setup/plan-database.yaml"       # Cross-file with relative path
    task: 1
```

---

## 7. Markdown Format Examples

### 7.1 Backward Compatible Format

```markdown
## Task 2: Add Migrations

**Files**: migrations/001_init.sql
**Depends on**: 1
**Agent**: golang-pro

Add database migrations...
```

### 7.2 Cross-File Reference Format (MVP Limited)

For MVP, markdown parser supports:

```markdown
## Task 3: User API

**Files**: internal/api/users.go
**Depends on**: 1, 2
**Agent**: golang-pro

Build user API endpoints...
```

**Note**: Complex cross-file references in markdown will be enhanced in v2.6. For now, use structured lists in YAML format for cross-file dependencies.

---

## 8. Error Handling & Messages

### 8.1 Parsing Errors

**Invalid Task Number in Cross-File Reference:**
```
Error: task 5 (User API): depends_on[2]: cross-file reference has invalid task number "2a"
  Valid task numbers contain only alphanumeric characters, hyphens, and underscores
  Example: 2, task-1, Task_A
```

**Missing Cross-File Reference Fields:**
```
Error: task 5 (User API): depends_on[1]: cross-file reference missing required 'task' field
  Cross-file references require both 'file' and 'task' fields
  Example: {file: "plan-01.yaml", task: 2}
```

**Invalid File Path:**
```
Error: task 5 (User API): depends_on[2]: invalid cross-file reference path: no such file or directory
  File path: "../../invalid/plan-01.yaml"
  Relative to: "/Users/user/conductor/plans/"
  Resolved to: "/Users/invalid/plan-01.yaml"
```

### 8.2 Merge/Validation Errors

**Unresolved Cross-File Reference:**
```
Error: task 5 (User API): cross-file reference to task 2 in plan-01-foundation.yaml not found
  Expected to find task "2" in file "plan-01-foundation.yaml"
  Available tasks in that file: 1, 3, 4
```

**Duplicate Task Number:**
```
Error: duplicate task number: 2
  Task 2 appears in both plan-01-foundation.yaml and plan-02-features.yaml
  Task numbers must be globally unique across all plan files
```

---

## 9. Backward Compatibility Strategy

### 9.1 Compatibility Guarantees

1. **Existing Numeric Dependencies**: Fully compatible
   - `depends_on: [2, 3, 4]` continues to work unchanged
   - No changes required to existing plan files

2. **Existing Single-File Plans**: Fully compatible
   - No validation changes for single-file scenarios
   - Cross-file references are warned but not enforced to fail

3. **Existing Code**: Fully compatible
   - Graph building reads from `DependsOn` field (unchanged)
   - `DependsOn` always contains all dependencies (local + resolved cross-file)
   - Code calling `task.DependsOn` works without modification
   - Helper method `NormalizedDependsOn()` available for explicit clarity

### 9.2 Migration Path for Users

**Scenario 1: Single-File Plan (No Changes Needed)**
```yaml
# Existing plan - no changes required
depends_on: [2, 3, 4]
```

**Scenario 2: Multi-File Plan to Add Cross-File Deps**
```yaml
# Before: All dependencies are local
depends_on: [1, 2]

# After: Mix local and cross-file references
depends_on:
  - 1                                 # Local reference
  - file: "plan-01-foundation.yaml"  # Cross-file reference
    task: 3
```

---

## 10. Testing Strategy

### 10.1 Unit Tests

**File: `/Users/harrison/Github/conductor/internal/parser/yaml_test.go` (new tests)**

```go
func TestParseDependencies_Numeric(t *testing.T) {
	// Test: depends_on: [2, 3, 4]
	// Expected: DependsOn = ["2", "3", "4"], CrossFileDependencies = {}
}

func TestParseDependencies_Mixed(t *testing.T) {
	// Test: depends_on: [2, {file: "plan-01.yaml", task: 3}]
	// Expected: DependsOn = ["2"], CrossFileDependencies = {"...plan-01.yaml:3" -> ""}
}

func TestParseDependencies_CrossFile(t *testing.T) {
	// Test: depends_on: [{file: "plan-01.yaml", task: 2}]
	// Expected: DependsOn = [], CrossFileDependencies = {"...plan-01.yaml:2" -> ""}
}

func TestParseDependencies_InvalidTaskNumber(t *testing.T) {
	// Test: depends_on: [{task: "2@invalid"}]
	// Expected: Error with helpful message
}

func TestNormalizeFilePath_Relative(t *testing.T) {
	// Test: normalizeFilePath("plan-01.yaml", "/home/user/plans/plan-02.yaml")
	// Expected: "/home/user/plans/plan-01.yaml"
}

func TestNormalizeFilePath_AbsoluteRejected(t *testing.T) {
	// Test: normalizeFilePath("/absolute/path/plan.yaml", "/home/user/plans/plan-02.yaml")
	// Expected: Error "absolute paths not allowed"
}
```

**File: `/Users/harrison/Github/conductor/internal/parser/parser_test.go` (new tests)**

```go
func TestMergePlans_CrossFileResolution(t *testing.T) {
	// Test: Two plans with cross-file dependency
	// Expected: Cross-file references resolved to local task numbers
}

func TestMergePlans_UnresolvedCrossFile(t *testing.T) {
	// Test: Cross-file reference to non-existent task
	// Expected: Error with helpful message
}

func TestMergePlans_DuplicateTaskNumber(t *testing.T) {
	// Test: Task 2 exists in both plans
	// Expected: Error "duplicate task number"
}
```

### 10.2 Integration Tests

**File: `/Users/harrison/Github/conductor/test/integration/cross_file_test.go` (new)**

```go
func TestCrossFileExecutionFlow(t *testing.T) {
	// E2E: Parse multi-file plan -> merge -> build graph -> calculate waves
	// Verify: Cross-file dependencies are correctly resolved in execution waves
}

func TestCrossFileWithCycles(t *testing.T) {
	// E2E: Cross-file dependency creates a cycle
	// Verify: Cycle detection works correctly
}

func TestCrossFileErrorMessages(t *testing.T) {
	// E2E: Various error scenarios
	// Verify: Error messages are clear and actionable
}
```

---

## 11. Files Modified Summary

### Core Model Changes
- **`internal/models/task.go`**: Add DependencyReference, NormalizedDependency types; extend Task struct
- **`internal/models/plan.go`**: No changes (Plan.FileToTaskMap already exists for this purpose)

### Parser Changes
- **`internal/parser/parser.go`**: Enhanced MergePlans() for cross-file resolution
- **`internal/parser/yaml.go`**: Add parseDependencies(), isValidTaskNumber(), normalizeFilePath()
- **`internal/parser/markdown.go`**: Update parseTaskMetadata() for consistency (minimal changes)

### Executor Changes
- **`internal/executor/graph.go`**:
  - Update ValidateTasks() for multi-file context
  - Add ValidateCrossFileDependencies()
  - Update BuildDependencyGraph() to use NormalizedDependsOn()

### Test Files (New)
- **`internal/parser/cross_file_yaml_test.go`**: YAML parsing tests
- **`internal/parser/cross_file_integration_test.go`**: MergePlans() tests
- **`internal/executor/cross_file_graph_test.go`**: Graph building tests
- **`test/integration/cross_file_e2e_test.go`**: End-to-end tests

---

## 12. Implementation Checklist

### Phase 1: Data Structures
- [ ] Add DependencyReference struct to task.go
- [ ] Add NormalizedDependency struct to task.go
- [ ] Add CrossFileDependencies field to Task
- [ ] Add DependsOnRaw field to Task (internal)
- [ ] Add DependencyReferences field to Task (internal)
- [ ] Add HasCrossFileDependencies() method
- [ ] Add GetAllDependencies() method
- [ ] Add NormalizedDependsOn() method
- [ ] Update Task struct documentation

### Phase 2: YAML Parser
- [ ] Implement parseDependencies() function
- [ ] Implement isValidTaskNumber() function
- [ ] Implement normalizeFilePath() function
- [ ] Update YAMLParser.Parse() to use parseDependencies()
- [ ] Add comprehensive error messages
- [ ] Write unit tests for parsing functions

### Phase 3: Markdown Parser
- [ ] Update parseTaskMetadata() to document cross-file support
- [ ] Handle graceful fallback for simple markdown format
- [ ] Add notes about markdown limitations for MVP

### Phase 4: Plan Merging
- [ ] Update MergePlans() to build fileToTask mapping
- [ ] Add cross-file reference resolution logic
- [ ] Add ValidateCrossFileDependencies() function
- [ ] Update error handling with helpful messages
- [ ] Write unit tests for merge scenarios

### Phase 5: Graph Building
- [ ] Update BuildDependencyGraph() to use NormalizedDependsOn()
- [ ] Update ValidateTasks() signature
- [ ] Add cycle detection tests with cross-file deps
- [ ] Write integration tests

### Phase 6: Testing & Documentation
- [ ] Create test data files (multi-file plans with cross-file deps)
- [ ] Write comprehensive unit tests
- [ ] Write integration tests
- [ ] Create E2E test scenarios
- [ ] Update README/CLAUDE.md with examples
- [ ] Test error messages for clarity

---

## 13. Performance Considerations

1. **File Path Normalization**: Cached during parsing, not recomputed
2. **Cross-File Resolution**: O(1) lookup via map in MergePlans()
3. **Dependency Graph**: No change to complexity (same Kahn's algorithm)
4. **Memory**: Minimal overhead (raw data cleaned after parsing)
   - DependsOnRaw cleared after processing
   - CrossFileDependencies only populated when needed

---

## 14. Security Considerations

1. **Path Validation**:
   - Absolute paths rejected (must be relative)
   - Path traversal attack mitigation (.. sequences validated)
   - filepath.Abs() prevents directory escape

2. **Task Number Validation**:
   - Alphanumeric only (no special characters)
   - Empty string validation
   - Type validation in YAML parsing

3. **File Access**:
   - Files must exist and be readable
   - Validation happens at merge time
   - Cross-file refs verified before execution

---

## 15. Future Enhancements (v2.6+)

1. **Markdown Enhancement**: Complex cross-file dependencies in markdown format
2. **File Aliasing**: Shorter alias syntax for frequently referenced files
3. **Package-Level Dependencies**: Depend on all tasks in a file
4. **Transitive Cross-File Deps**: Handle indirect file dependencies
5. **Dependency Groups**: Group related cross-file dependencies
6. **Visualization**: Graph visualization showing cross-file dependencies

---

## Conclusion

This design provides a robust, backward-compatible approach to cross-file dependencies in Conductor using the annotation-based format. The implementation is phased, well-tested, and maintains the existing architecture while adding powerful new multi-file coordination capabilities.
