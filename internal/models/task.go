package models

import (
	"errors"
	"time"
)

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

	// Structured verification (v2.3+)
	SuccessCriteria     []string `yaml:"success_criteria,omitempty" json:"success_criteria,omitempty"`
	TestCommands        []string `yaml:"test_commands,omitempty" json:"test_commands,omitempty"`
	Type                string   `yaml:"type,omitempty" json:"type,omitempty"`                               // Task type: regular or integration
	IntegrationCriteria []string `yaml:"integration_criteria,omitempty" json:"integration_criteria,omitempty"` // Criteria for integration tasks

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
