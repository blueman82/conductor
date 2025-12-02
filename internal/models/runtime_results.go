package models

import "time"

// TestCommandResult holds the result of a single test command execution.
type TestCommandResult struct {
	Command  string
	Output   string
	Error    error
	Passed   bool
	Duration time.Duration
}

// CriterionVerificationResult holds the result of a single criterion verification.
type CriterionVerificationResult struct {
	Index       int
	Criterion   string
	Command     string
	Output      string
	Expected    string
	Error       error
	Passed      bool
	Duration    time.Duration
	Description string
}

// DocTargetResult holds the result of a single documentation target verification.
type DocTargetResult struct {
	Location   string // File path
	Section    string // Section heading to verify
	LineNumber int    // Line number where section was found (0 if not found)
	Passed     bool
	Content    string // Content found near the section (for context)
	Error      error
}
