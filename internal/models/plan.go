package models

// Plan represents an implementation plan with tasks and configuration
type Plan struct {
	Name           string               // Plan name
	Tasks          []Task               // List of tasks to execute
	Waves          []Wave               // Execution waves (grouped tasks)
	DefaultAgent   string               // Default agent to use if not specified in task
	QualityControl QualityControlConfig // QC configuration
	FilePath       string               // Original file path (for updates)
	WorktreeGroups []WorktreeGroup      // Worktree groups for task organization
	FileToTaskMap  map[string][]string  // File path -> list of task numbers mapping
}

// Wave represents a group of tasks that can be executed in parallel
type Wave struct {
	Name           string              // Wave name (e.g., "Wave 1")
	TaskNumbers    []string            // Task numbers in this wave
	MaxConcurrency int                 // Maximum concurrent tasks in this wave
	GroupInfo      map[string][]string // group ID -> list of task numbers in this wave
}

// QCAgentConfig represents multi-agent QC configuration (v2.2+)
type QCAgentConfig struct {
	Mode             string   // Selection mode: "auto", "explicit", or "mixed"
	ExplicitList     []string // Explicit list of agents (for mode=explicit)
	AdditionalAgents []string // Additional agents (for mode=mixed)
	BlockedAgents    []string // Agents to never use
}

// QualityControlConfig holds configuration for the QA review process
type QualityControlConfig struct {
	Enabled     bool          // Whether QC is enabled
	ReviewAgent string        // Agent to use for reviews (DEPRECATED: use Agents)
	Agents      QCAgentConfig // Multi-agent QC configuration (v2.2+)
	RetryOnRed  int           // Number of retries on RED status
}

// WorktreeGroup represents a group of related tasks for organization and execution control
type WorktreeGroup struct {
	GroupID        string // Unique identifier for the group
	Description    string // Human-readable description of the group
	ExecutionModel string // Execution model (e.g., "parallel", "sequential")
	Isolation      string // Isolation level (e.g., "none", "weak", "strong")
	Rationale      string // Rationale for the group and its configuration
}
