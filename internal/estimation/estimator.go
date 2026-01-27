// Package estimation provides human time estimation for task execution.
// It uses Claude haiku to estimate how long a senior developer would take
// to complete a task manually, enabling speedup ratio calculation.
package estimation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/budget"
	"github.com/harrison/conductor/internal/claude"
	"github.com/harrison/conductor/internal/models"
)

// EstimateResponse contains Claude's human time estimate
type EstimateResponse struct {
	EstimateMinutes int    `json:"estimate_minutes"`
	Reasoning       string `json:"reasoning"`
	Confidence      string `json:"confidence"` // high, medium, low
}

// Estimator estimates human developer time for tasks using Claude haiku.
// Embeds claude.Service for CLI invocation with rate limit handling.
type Estimator struct {
	claude.Service
}

// NewEstimator creates an estimator with the specified timeout.
// The timeout parameter controls how long to wait for Claude CLI responses.
// Use config.DefaultTimeoutsConfig().LLM for the standard timeout value.
func NewEstimator(timeout time.Duration, logger budget.WaiterLogger) *Estimator {
	return &Estimator{
		Service: *claude.NewService(timeout, logger),
	}
}

// NewEstimatorWithInvoker creates an estimator using an external Invoker.
// This allows sharing a single Invoker across multiple components for consistent
// configuration and rate limit handling.
func NewEstimatorWithInvoker(inv *claude.Invoker) *Estimator {
	return &Estimator{
		Service: *claude.NewServiceWithInvoker(inv),
	}
}

// EstimateTask estimates how long a human developer would take to complete the task.
// Returns the estimate in seconds and the source identifier ("claude-haiku").
// Returns (0, "", error) if estimation fails.
func (e *Estimator) EstimateTask(ctx context.Context, task *models.Task) (int64, string, error) {
	prompt := e.buildPrompt(task)

	var result EstimateResponse
	if err := e.InvokeAndParse(ctx, prompt, EstimationSchema(), &result); err != nil {
		return 0, "", fmt.Errorf("estimation failed: %w", err)
	}

	// Convert minutes to seconds
	estimateSecs := int64(result.EstimateMinutes * 60)

	return estimateSecs, "claude-haiku", nil
}

// buildPrompt constructs the estimation prompt from task details
func (e *Estimator) buildPrompt(task *models.Task) string {
	var sb strings.Builder

	sb.WriteString("Estimate how long a senior software developer would take to complete this task manually.\n\n")

	sb.WriteString(fmt.Sprintf("Task: %s\n", task.Name))

	if task.Prompt != "" {
		// Truncate very long prompts to avoid excessive token usage
		prompt := task.Prompt
		if len(prompt) > 2000 {
			prompt = prompt[:2000] + "..."
		}
		sb.WriteString(fmt.Sprintf("Description: %s\n", prompt))
	}

	if len(task.Files) > 0 {
		sb.WriteString(fmt.Sprintf("Files: %s\n", strings.Join(task.Files, ", ")))
	}

	if len(task.SuccessCriteria) > 0 {
		sb.WriteString(fmt.Sprintf("Success Criteria: %s\n", strings.Join(task.SuccessCriteria, "; ")))
	}

	sb.WriteString("\nConsider the full development cycle:\n")
	sb.WriteString("- Reading and understanding existing code\n")
	sb.WriteString("- Planning the implementation approach\n")
	sb.WriteString("- Writing the implementation\n")
	sb.WriteString("- Testing and debugging\n")
	sb.WriteString("- Self-review and refinement\n")
	sb.WriteString("\nProvide your estimate in minutes. Respond with JSON only.")

	return sb.String()
}

// EstimationSchema returns the JSON schema for response enforcement
func EstimationSchema() string {
	return `{
		"type": "object",
		"properties": {
			"estimate_minutes": {
				"type": "integer",
				"minimum": 1,
				"description": "Estimated time in minutes for a senior developer"
			},
			"reasoning": {
				"type": "string",
				"description": "Brief explanation of the estimate"
			},
			"confidence": {
				"type": "string",
				"enum": ["high", "medium", "low"],
				"description": "Confidence level in the estimate"
			}
		},
		"required": ["estimate_minutes", "reasoning", "confidence"]
	}`
}
