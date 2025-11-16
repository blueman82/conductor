package models

import "fmt"

// AgentResponse represents structured JSON output from an agent execution
type AgentResponse struct {
	Status   string                 `json:"status"`         // "success" or "failed"
	Summary  string                 `json:"summary"`        // Brief description
	Output   string                 `json:"output"`         // Full execution output
	Errors   []string               `json:"errors"`         // Error messages
	Files    []string               `json:"files_modified"` // Modified file paths
	Metadata map[string]interface{} `json:"metadata"`       // Additional data
}

// Validate checks if required fields are present
func (r *AgentResponse) Validate() error {
	if r.Status == "" {
		return fmt.Errorf("status is required")
	}
	if r.Status != "success" && r.Status != "failed" {
		return fmt.Errorf("status must be 'success' or 'failed'")
	}
	return nil
}

// Issue represents a specific issue found during QC review
type Issue struct {
	Severity    string `json:"severity"`    // "critical", "warning", "info"
	Description string `json:"description"` // Issue description
	Location    string `json:"location"`    // File:line or component
}

// QCResponse represents structured JSON output from QC review
type QCResponse struct {
	Verdict         string   `json:"verdict"`         // "GREEN", "RED", "YELLOW"
	Feedback        string   `json:"feedback"`        // Detailed review feedback
	Issues          []Issue  `json:"issues"`          // Specific issues found
	Recommendations []string `json:"recommendations"` // Suggested improvements
	ShouldRetry     bool     `json:"should_retry"`    // Whether to retry
	SuggestedAgent  string   `json:"suggested_agent"` // Alternative agent suggestion
}

// Validate checks if required fields are present
func (r *QCResponse) Validate() error {
	if r.Verdict == "" {
		return fmt.Errorf("verdict is required")
	}
	validVerdicts := map[string]bool{"GREEN": true, "RED": true, "YELLOW": true}
	if !validVerdicts[r.Verdict] {
		return fmt.Errorf("verdict must be GREEN, RED, or YELLOW")
	}
	return nil
}
