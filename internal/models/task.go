package models

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// CrossFileDependency represents a dependency on a task in a different plan file
type CrossFileDependency struct {
	File   string `yaml:"file" json:"file"` // Filename (e.g., "plan-01-foundation.yaml")
	TaskID string `yaml:"task" json:"task"` // Task number in that file
}

// KeyPoint represents a structured implementation key point from the plan
type KeyPoint struct {
	Point     string `yaml:"point,omitempty" json:"point,omitempty"`
	Details   string `yaml:"details,omitempty" json:"details,omitempty"`
	Reference string `yaml:"reference,omitempty" json:"reference,omitempty"`
}

// String returns a standardized string representation of a cross-file dependency
// Format: "file:{filename}:task:{task-id}" (e.g., "file:plan-01-foundation.yaml:task:2")
func (cfd *CrossFileDependency) String() string {
	return fmt.Sprintf("file:%s:task:%s", cfd.File, cfd.TaskID)
}

// Task represents a single task in an implementation plan
type Task struct {
	Number        string                 // Task number/identifier (supports int, float, alphanumeric)
	Name          string                 // Task name/title
	Files         []string               // Files to be modified/created
	DependsOn     []string               // Task numbers this task depends on
	EstimatedTime time.Duration          // Estimated time to complete
	Agent         string                 // Agent to use (optional)
	Prompt        string                 // Full task description/prompt
	WorktreeGroup string                 // Worktree group this task belongs to (optional)
	Status        string                 // Task status: pending, in_progress, completed, skipped
	StartedAt     *time.Time             // Timestamp when task execution started (nil if not started)
	CompletedAt   *time.Time             // Timestamp when task was completed (nil if not completed)
	SourceFile    string                 // Source plan file this task originates from (for multi-file plans)
	Metadata      map[string]interface{} // Additional metadata for hooks and extensions
	KeyPoints     []KeyPoint             `yaml:"key_points,omitempty" json:"key_points,omitempty"` // Structured implementation key points

	// Structured verification (v2.3+)
	SuccessCriteria     []string `yaml:"success_criteria,omitempty" json:"success_criteria,omitempty"`
	TestCommands        []string `yaml:"test_commands,omitempty" json:"test_commands,omitempty"`
	Type                string   `yaml:"type,omitempty" json:"type,omitempty"`                                 // Task type: regular or integration
	IntegrationCriteria []string `yaml:"integration_criteria,omitempty" json:"integration_criteria,omitempty"` // Criteria for integration tasks

	// JSON Schema enforcement (v2.8+)
	JSONSchema string `yaml:"json_schema,omitempty" json:"json_schema,omitempty"` // Custom JSON schema for response validation

	// Execution metadata for enhanced console output
	ExecutionStartTime time.Time     `json:"execution_start_time,omitempty" yaml:"execution_start_time,omitempty"`
	ExecutionEndTime   time.Time     `json:"execution_end_time,omitempty" yaml:"execution_end_time,omitempty"`
	ExecutionDuration  time.Duration `json:"execution_duration,omitempty" yaml:"execution_duration,omitempty"`
	ExecutedBy         string        `json:"executed_by,omitempty" yaml:"executed_by,omitempty"` // Agent name
	FilesModified      int           `json:"files_modified,omitempty" yaml:"files_modified,omitempty"`
	FilesCreated       int           `json:"files_created,omitempty" yaml:"files_created,omitempty"`
	FilesDeleted       int           `json:"files_deleted,omitempty" yaml:"files_deleted,omitempty"`
}

// Validate checks if the task has all required fields
func (t *Task) Validate() error {
	if t.Number == "" {
		return errors.New("task number is required")
	}
	if t.Name == "" {
		return errors.New("task name is required")
	}
	if t.Prompt == "" {
		return errors.New("task prompt is required")
	}
	return nil
}

// IsCompleted returns true if the task status is "completed"
func (t *Task) IsCompleted() bool {
	return t.Status == "completed"
}

// CanSkip returns true if the task can be skipped in execution
// (i.e., it's already completed or marked as skipped)
func (t *Task) CanSkip() bool {
	return t.Status == "completed" || t.Status == "skipped"
}

// CalculateDuration computes the duration between ExecutionStartTime and ExecutionEndTime
// Returns 0 if either timestamp is zero, or if end time is before start time
func (t *Task) CalculateDuration() time.Duration {
	if t.ExecutionStartTime.IsZero() || t.ExecutionEndTime.IsZero() {
		return 0
	}
	duration := t.ExecutionEndTime.Sub(t.ExecutionStartTime)
	// Return negative durations as-is (don't clamp to 0)
	return duration
}

// RecordFileOperation increments the appropriate file operation counter
// Recognizes "modified", "created", and "deleted" operations
// Unknown operations are ignored
func (t *Task) RecordFileOperation(operation string) {
	switch operation {
	case "modified":
		t.FilesModified++
	case "created":
		t.FilesCreated++
	case "deleted":
		t.FilesDeleted++
	}
}

// TotalFileOperations returns the sum of all file operations
func (t *Task) TotalFileOperations() int {
	return t.FilesModified + t.FilesCreated + t.FilesDeleted
}

// GetFormattedDuration returns the ExecutionDuration in human-readable format
func (t *Task) GetFormattedDuration() string {
	return t.ExecutionDuration.String()
}

// IsIntegration returns true if the task type is "integration"
func (t *Task) IsIntegration() bool {
	return t.Type == "integration"
}

// UnmarshalYAML handles custom YAML unmarshaling for Task to support mixed dependency formats
// Supports both:
//   - Numeric-only: depends_on: [1, 2, 3]
//   - Mixed: depends_on: [1, {file: "plan-01.yaml", task: 2}]
func (t *Task) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First, unmarshal into a raw map to handle dependencies specially
	type TaskAlias Task
	raw := struct {
		*TaskAlias
		DependsOn []interface{} `yaml:"depends_on"`
	}{
		TaskAlias: (*TaskAlias)(t),
	}

	if err := unmarshal(&raw); err != nil {
		return err
	}

	// Process depends_on: convert mixed format to normalized strings
	if raw.DependsOn != nil {
		normalized, err := normalizeDependencies(raw.DependsOn)
		if err != nil {
			return err
		}
		t.DependsOn = normalized
	}

	return nil
}

// normalizeDependencies converts a slice of mixed dependency formats into normalized strings
// Supports: integers, floats, strings (numeric), and CrossFileDependency objects
func normalizeDependencies(deps []interface{}) ([]string, error) {
	var normalized []string

	for _, dep := range deps {
		switch v := dep.(type) {
		case int:
			normalized = append(normalized, fmt.Sprintf("%d", v))
		case float64:
			// YAML parses numbers as float64; check if it's a whole number
			if v == float64(int(v)) {
				normalized = append(normalized, fmt.Sprintf("%d", int(v)))
			} else {
				normalized = append(normalized, fmt.Sprintf("%v", v))
			}
		case string:
			normalized = append(normalized, v)
		case map[string]interface{}:
			// Parse cross-file dependency
			cfd, err := parseCrossFileDependencyMap(v)
			if err != nil {
				return nil, err
			}
			normalized = append(normalized, cfd.String())
		default:
			return nil, fmt.Errorf("unsupported dependency format: %T", v)
		}
	}

	return normalized, nil
}

// parseCrossFileDependencyMap converts a map[string]interface{} to CrossFileDependency
func parseCrossFileDependencyMap(m map[string]interface{}) (*CrossFileDependency, error) {
	cfd := &CrossFileDependency{}

	if file, ok := m["file"]; ok {
		if fileStr, ok := file.(string); ok {
			cfd.File = fileStr
		} else {
			return nil, fmt.Errorf("cross-file dependency 'file' must be a string, got %T", file)
		}
	} else {
		return nil, errors.New("cross-file dependency missing required 'file' field")
	}

	if task, ok := m["task"]; ok {
		switch v := task.(type) {
		case int:
			cfd.TaskID = fmt.Sprintf("%d", v)
		case float64:
			if v == float64(int(v)) {
				cfd.TaskID = fmt.Sprintf("%d", int(v))
			} else {
				cfd.TaskID = fmt.Sprintf("%v", v)
			}
		case string:
			cfd.TaskID = v
		default:
			return nil, fmt.Errorf("cross-file dependency 'task' must be int/float/string, got %T", task)
		}
	} else {
		return nil, errors.New("cross-file dependency missing required 'task' field")
	}

	return cfd, nil
}

// NormalizeDependency converts a single dependency (in any format) to standardized string form
// Returns error if the format is invalid
func NormalizeDependency(dep interface{}) (string, error) {
	switch v := dep.(type) {
	case int:
		return fmt.Sprintf("%d", v), nil
	case float64:
		if v == float64(int(v)) {
			return fmt.Sprintf("%d", int(v)), nil
		}
		return fmt.Sprintf("%v", v), nil
	case string:
		return v, nil
	case *CrossFileDependency:
		return v.String(), nil
	case CrossFileDependency:
		return v.String(), nil
	case map[string]interface{}:
		cfd, err := parseCrossFileDependencyMap(v)
		if err != nil {
			return "", err
		}
		return cfd.String(), nil
	default:
		return "", fmt.Errorf("unsupported dependency format: %T", dep)
	}
}

// IsCrossFileDep returns true if the dependency is a cross-file reference
// Cross-file dependencies have the format: "file:{filename}:task:{task-id}"
func IsCrossFileDep(dep string) bool {
	return strings.HasPrefix(dep, "file:") && strings.Contains(dep, ":task:")
}

// ParseCrossFileDep extracts file and task information from a cross-file dependency string
// Returns error if the string is not a valid cross-file dependency
// Expected format: "file:{filename}:task:{task-id}" (e.g., "file:plan-01.yaml:task:2")
func ParseCrossFileDep(dep string) (*CrossFileDependency, error) {
	if !IsCrossFileDep(dep) {
		return nil, fmt.Errorf("not a valid cross-file dependency: %s", dep)
	}

	// Parse format: "file:{filename}:task:{task-id}"
	parts := strings.Split(dep, ":task:")
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed cross-file dependency: %s", dep)
	}

	filePrefix := parts[0]
	taskID := parts[1]

	// Extract filename from "file:{filename}"
	if !strings.HasPrefix(filePrefix, "file:") {
		return nil, fmt.Errorf("cross-file dependency must start with 'file:': %s", dep)
	}

	filename := strings.TrimPrefix(filePrefix, "file:")
	if filename == "" {
		return nil, fmt.Errorf("cross-file dependency has empty filename: %s", dep)
	}

	if taskID == "" {
		return nil, fmt.Errorf("cross-file dependency has empty task ID: %s", dep)
	}

	return &CrossFileDependency{
		File:   filename,
		TaskID: taskID,
	}, nil
}

// HasCyclicDependencies detects circular dependencies in a list of tasks
// using DFS with color marking (white=unvisited, gray=visiting, black=visited)
func HasCyclicDependencies(tasks []Task) bool {
	// Build adjacency list: task number -> list of dependent task numbers
	graph := make(map[string][]string)
	taskMap := make(map[string]bool)

	// Initialize graph and track valid task numbers
	for _, task := range tasks {
		taskMap[task.Number] = true
		graph[task.Number] = []string{}
	}

	// Build edges: if task A depends on B, then B -> A
	for _, task := range tasks {
		for _, dep := range task.DependsOn {
			// Check for self-reference
			if dep == task.Number {
				return true
			}
			// Only add edge if dependency exists
			if taskMap[dep] {
				graph[dep] = append(graph[dep], task.Number)
			}
		}
	}

	// DFS with color marking
	const (
		white = 0 // not visited
		gray  = 1 // currently visiting
		black = 2 // visited
	)

	colors := make(map[string]int)
	for taskNum := range taskMap {
		colors[taskNum] = white
	}

	// DFS function to detect back edges (cycles)
	var dfs func(string) bool
	dfs = func(node string) bool {
		colors[node] = gray

		for _, neighbor := range graph[node] {
			if colors[neighbor] == gray {
				// Back edge found - cycle detected
				return true
			}
			if colors[neighbor] == white && dfs(neighbor) {
				return true
			}
		}

		colors[node] = black
		return false
	}

	// Run DFS from each unvisited node
	for taskNum := range taskMap {
		if colors[taskNum] == white {
			if dfs(taskNum) {
				return true
			}
		}
	}

	return false
}
