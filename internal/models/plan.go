package models

// Plan represents an implementation plan with tasks and configuration
type Plan struct {
	Name            string               // Plan name
	Tasks           []Task               // List of tasks to execute
	Waves           []Wave               // Execution waves (grouped tasks)
	DefaultAgent    string               // Default agent to use if not specified in task
	QualityControl  QualityControlConfig // QC configuration
	FilePath        string               // Original file path (for updates)
	WorktreeGroups  []WorktreeGroup      // Worktree groups for task organization
}

// Wave represents a group of tasks that can be executed in parallel
type Wave struct {
	Name           string   // Wave name (e.g., "Wave 1")
	TaskNumbers    []string // Task numbers in this wave
	MaxConcurrency int      // Maximum concurrent tasks in this wave
}

// QualityControlConfig holds configuration for the QA review process
type QualityControlConfig struct {
	Enabled     bool   // Whether QC is enabled
	ReviewAgent string // Agent to use for reviews
	RetryOnRed  int    // Number of retries on RED status
}

// WorktreeGroup represents a group of related tasks for organization and execution control
type WorktreeGroup struct {
	GroupID        string // Unique identifier for the group
	Description    string // Human-readable description of the group
	ExecutionModel string // Execution model (e.g., "parallel", "sequential")
	Isolation      string // Isolation level (e.g., "none", "weak", "strong")
	Rationale      string // Rationale for the group and its configuration
}
