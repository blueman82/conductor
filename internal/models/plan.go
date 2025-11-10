package models

// Plan represents an implementation plan with tasks and configuration
type Plan struct {
	Name           string               // Plan name
	Tasks          []Task               // List of tasks to execute
	Waves          []Wave               // Execution waves (grouped tasks)
	DefaultAgent   string               // Default agent to use if not specified in task
	QualityControl QualityControlConfig // QC configuration
	FilePath       string               // Original file path (for updates)
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
