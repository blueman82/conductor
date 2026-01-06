package models

import "time"

// Task execution status constants
const (
	StatusGreen  = "GREEN"  // Task completed successfully
	StatusYellow = "YELLOW" // Task completed with warnings
	StatusRed    = "RED"    // Task failed quality control
	StatusFailed = "FAILED" // Task failed to execute
)

// ExecutionAttempt represents a single execution attempt (for retry tracking)
type ExecutionAttempt struct {
	Attempt     int    // Attempt number (1-indexed)
	Agent       string // Agent used for this attempt
	AgentOutput string // Raw JSON output from agent
	QCFeedback  string // Raw JSON output from QC review
	Verdict     string // QC verdict: "GREEN", "RED", "YELLOW"
	Duration    time.Duration
}

// TaskResult represents the result of executing a single task
type TaskResult struct {
	Task             Task               // The task that was executed
	Status           string             // Status: "GREEN", "RED", "TIMEOUT", "FAILED"
	Output           string             // Captured output from agent
	Error            error              // Error if execution failed
	Duration         time.Duration      // Time taken to execute
	RetryCount       int                // Number of retries attempted
	ReviewFeedback   string             // Feedback from QC review
	ExecutionHistory []ExecutionAttempt // Detailed history of all attempts
	SessionID        string             // Claude CLI session ID (for rate limit recovery)
}

// ExecutionResult represents the aggregate result of executing a plan
type ExecutionResult struct {
	TotalTasks      int            `json:"total_tasks" yaml:"total_tasks"`             // Total number of tasks
	Completed       int            `json:"completed" yaml:"completed"`                 // Number of completed tasks
	Failed          int            `json:"failed" yaml:"failed"`                       // Number of failed tasks
	Duration        time.Duration  `json:"duration" yaml:"duration"`                   // Total execution time
	FailedTasks     []TaskResult   `json:"failed_tasks" yaml:"failed_tasks"`           // Details of failed tasks
	StatusBreakdown map[string]int `json:"status_breakdown" yaml:"status_breakdown"`   // Count by QC status (GREEN/YELLOW/RED)
	AgentUsage      map[string]int `json:"agent_usage" yaml:"agent_usage"`             // Count by agent name
	TotalFiles      int            `json:"total_files" yaml:"total_files"`             // Count of unique files modified
	AvgTaskDuration time.Duration  `json:"avg_task_duration" yaml:"avg_task_duration"` // Average duration per task

	// LOC tracking aggregates (v3.4+)
	TotalLinesAdded   int `json:"total_lines_added" yaml:"total_lines_added"`
	TotalLinesDeleted int `json:"total_lines_deleted" yaml:"total_lines_deleted"`
}

// calculateMetricsFromResults calculates all metrics from a slice of TaskResults.
// This is a private helper function that consolidates metric calculation logic
// to eliminate duplication between NewExecutionResult and CalculateMetrics.
func (er *ExecutionResult) calculateMetricsFromResults(results []TaskResult) {
	// Initialize all status keys to ensure they exist even with zero values
	er.StatusBreakdown[StatusGreen] = 0
	er.StatusBreakdown[StatusYellow] = 0
	er.StatusBreakdown[StatusRed] = 0

	// Reset counters
	er.Completed = 0
	er.Failed = 0
	er.TotalLinesAdded = 0
	er.TotalLinesDeleted = 0

	// Track unique files using a map (set)
	uniqueFiles := make(map[string]bool)

	// Process all results to calculate metrics
	for _, result := range results {
		// Count statuses
		if result.Status != "" {
			er.StatusBreakdown[result.Status]++
		}

		// Track agent usage (count empty agents too)
		if result.Task.Agent != "" {
			er.AgentUsage[result.Task.Agent]++
		} else {
			// Count tasks with no agent
			er.AgentUsage[""]++
		}

		// Collect unique files
		for _, file := range result.Task.Files {
			uniqueFiles[file] = true
		}

		// Aggregate LOC metrics
		er.TotalLinesAdded += result.Task.LinesAdded
		er.TotalLinesDeleted += result.Task.LinesDeleted

		// Track completed/failed
		if result.Status == StatusRed || result.Status == StatusFailed {
			er.Failed++
			// Only append to FailedTasks if it's initialized (not nil)
			if er.FailedTasks != nil {
				er.FailedTasks = append(er.FailedTasks, result)
			}
		} else {
			er.Completed++
		}
	}

	// Calculate total unique files
	er.TotalFiles = len(uniqueFiles)

	// Calculate average task duration
	if len(results) > 0 {
		totalDur := time.Duration(0)
		for _, result := range results {
			totalDur += result.Duration
		}
		er.AvgTaskDuration = totalDur / time.Duration(len(results))
	}

	// Remove empty agent entry if it has zero count
	if er.AgentUsage[""] == 0 {
		delete(er.AgentUsage, "")
	}
}

// NewExecutionResult creates a new ExecutionResult with calculated metrics
func NewExecutionResult(results []TaskResult, success bool, totalDuration time.Duration) *ExecutionResult {
	er := &ExecutionResult{
		TotalTasks:      len(results),
		Duration:        totalDuration,
		FailedTasks:     []TaskResult{},
		StatusBreakdown: make(map[string]int),
		AgentUsage:      make(map[string]int),
	}

	// Use the consolidated helper to calculate all metrics
	er.calculateMetricsFromResults(results)

	return er
}

// CalculateMetrics updates the result with calculated metrics (used for existing results)
func (er *ExecutionResult) CalculateMetrics(results []TaskResult) {
	// Clear and reinitialize maps
	er.StatusBreakdown = make(map[string]int)
	er.AgentUsage = make(map[string]int)

	// Use the consolidated helper to calculate all metrics
	er.calculateMetricsFromResults(results)
}
