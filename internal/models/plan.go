package models

// Plan represents an implementation plan with tasks and configuration
type Plan struct {
	Name              string                 // Plan name
	Tasks             []Task                 // List of tasks to execute
	Waves             []Wave                 // Execution waves (grouped tasks)
	DefaultAgent      string                 // Default agent to use if not specified in task
	QualityControl    QualityControlConfig   // QC configuration
	FilePath          string                 // Original file path (for updates)
	WorktreeGroups    []WorktreeGroup        // Worktree groups for task organization
	FileToTaskMap     map[string][]string    // File path -> list of task numbers mapping
	PlannerCompliance *PlannerComplianceSpec // Runtime enforcement metadata (v2.9+)
	DataFlowRegistry  *DataFlowRegistry      // Data flow registry for runtime enforcement (v2.9+)
}

// DataFlowRegistry captures producers/consumers for runtime enforcement.
// Enables machine-readable data flow validation and documentation targeting.
type DataFlowRegistry struct {
	// Producers maps symbol names to the tasks that produce/define them
	Producers map[string][]DataFlowEntry `yaml:"producers,omitempty" json:"producers,omitempty"`

	// Consumers maps symbol names to the tasks that consume/use them
	Consumers map[string][]DataFlowEntry `yaml:"consumers,omitempty" json:"consumers,omitempty"`

	// DocumentationTargets maps task numbers to their documentation requirements
	DocumentationTargets map[string][]DocumentationTarget `yaml:"documentation_targets,omitempty" json:"documentation_targets,omitempty"`
}

// DataFlowEntry represents a single producer or consumer entry
type DataFlowEntry struct {
	// TaskNumber is the task ID that produces or consumes the symbol
	TaskNumber string `yaml:"task" json:"task"`

	// Symbol is the name of the symbol being produced or consumed
	Symbol string `yaml:"symbol,omitempty" json:"symbol,omitempty"`

	// Description provides context about this data flow
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// PlannerComplianceSpec captures planner metadata for strict runtime enforcement (v2.9+)
// When present, this enables machine-readable validation of plan files.
type PlannerComplianceSpec struct {
	// PlannerVersion identifies the version of the planner that generated this plan.
	// Required when planner_compliance section is present.
	PlannerVersion string `yaml:"planner_version" json:"planner_version"`

	// StrictEnforcement enables strict validation mode.
	// When true, all tasks MUST have runtime_metadata populated.
	// No warn/off modes - either strict (true) or legacy/disabled (false).
	StrictEnforcement bool `yaml:"strict_enforcement" json:"strict_enforcement"`

	// RequiredFeatures lists the planner features that must be validated.
	// Valid values: "dependency_checks", "documentation_targets", "success_criteria"
	RequiredFeatures []string `yaml:"required_features,omitempty" json:"required_features,omitempty"`
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
	Mode             string   // Selection mode: "auto", "explicit", "mixed", or "intelligent"
	ExplicitList     []string // Explicit list of agents (for mode=explicit)
	AdditionalAgents []string // Additional agents (for mode=mixed)
	BlockedAgents    []string // Agents to never use
	// Intelligent selection settings (v2.4+)
	MaxAgents               int  // Maximum number of agents to select (default: 4)
	CacheTTLSeconds         int  // Cache TTL in seconds (default: 3600)
	SelectionTimeoutSeconds int  // Timeout for Claude selection calls (default: 90)
	RequireCodeReview       bool // Always include code-reviewer as baseline (default: true)
}

// QualityControlConfig holds configuration for the QA review process
type QualityControlConfig struct {
	Enabled     bool          // Whether QC is enabled
	ReviewAgent string        // Agent to use for reviews (DEPRECATED: use Agents)
	Agents      QCAgentConfig // Multi-agent QC configuration (v2.2+)
	RetryOnRed  int           // Number of retries on RED status
}

// WorktreeGroup represents a group of related tasks for organization.
// NOTE: Groups are organizational metadata only - NOT used for execution control.
// Conductor uses Task.DependsOn to determine execution order via dependency graph.
type WorktreeGroup struct {
	GroupID        string // Unique identifier for the group
	Description    string // Human-readable description of the group
	ExecutionModel string // DEPRECATED: Not used for execution control (kept for backward compatibility)
	Isolation      string // DEPRECATED: Not used for execution control (kept for backward compatibility)
	Rationale      string // Rationale for the group and its configuration
}
