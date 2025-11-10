package models

import "time"

// Task execution status constants
const (
	StatusGreen  = "GREEN"  // Task completed successfully
	StatusYellow = "YELLOW" // Task completed with warnings
	StatusRed    = "RED"    // Task failed quality control
	StatusFailed = "FAILED" // Task failed to execute
)

// TaskResult represents the result of executing a single task
type TaskResult struct {
	Task           Task          // The task that was executed
	Status         string        // Status: "GREEN", "RED", "TIMEOUT", "FAILED"
	Output         string        // Captured output from agent
	Error          error         // Error if execution failed
	Duration       time.Duration // Time taken to execute
	RetryCount     int           // Number of retries attempted
	ReviewFeedback string        // Feedback from QC review
}

// ExecutionResult represents the aggregate result of executing a plan
type ExecutionResult struct {
	TotalTasks  int            // Total number of tasks
	Completed   int            // Number of completed tasks
	Failed      int            // Number of failed tasks
	Duration    time.Duration  // Total execution time
	FailedTasks []TaskResult   // Details of failed tasks
}
