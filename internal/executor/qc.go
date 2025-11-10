package executor

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

// QualityController manages quality control reviews using Claude Code agents
type QualityController struct {
	Invoker     InvokerInterface
	ReviewAgent string // Agent name to use for reviews (e.g., "quality-control")
	MaxRetries  int    // Maximum number of retry attempts for RED responses
}

// InvokerInterface defines the interface for agent invocation
type InvokerInterface interface {
	Invoke(ctx context.Context, task models.Task) (*agent.InvocationResult, error)
}

// ReviewResult captures the result of a quality control review
type ReviewResult struct {
	Flag     string // "GREEN", "RED", or "YELLOW"
	Feedback string // Detailed feedback from the reviewer
}

// NewQualityController creates a new QualityController with default settings
func NewQualityController(invoker InvokerInterface) *QualityController {
	return &QualityController{
		Invoker:     invoker,
		ReviewAgent: "quality-control",
		MaxRetries:  2,
	}
}

// BuildReviewPrompt creates a comprehensive review prompt for the QC agent
func (qc *QualityController) BuildReviewPrompt(task models.Task, output string) string {
	return fmt.Sprintf(`Review the following task execution:

Task: %s

Output:
%s

Provide quality control review in this format:
Quality Control: [GREEN/RED/YELLOW]

Feedback: [your detailed feedback]

Respond with GREEN if all requirements met, RED if needs rework, YELLOW if minor issues.
`, task.Name, output)
}

// ParseReviewResponse extracts the QC flag and feedback from agent output
func ParseReviewResponse(output string) (flag string, feedback string) {
	// Parse "Quality Control: GREEN/RED/YELLOW" using regex
	flagRegex := regexp.MustCompile(`Quality Control:\s*(GREEN|RED|YELLOW)`)
	matches := flagRegex.FindStringSubmatch(output)
	if len(matches) > 1 {
		flag = matches[1]
	}

	// Extract feedback after the flag line
	// Split on newline after the Quality Control line
	parts := strings.SplitN(output, "\n", 2)
	if len(parts) > 1 {
		feedback = strings.TrimSpace(parts[1])
	}

	return flag, feedback
}

// Review executes a quality control review of a task output
func (qc *QualityController) Review(ctx context.Context, task models.Task, output string) (*ReviewResult, error) {
	// Build the review prompt
	prompt := qc.BuildReviewPrompt(task, output)

	// Create a review task for the invoker
	reviewTask := models.Task{
		Number: task.Number,
		Name:   fmt.Sprintf("QC Review: %s", task.Name),
		Prompt: prompt,
		Agent:  qc.ReviewAgent,
	}

	// Invoke the QC agent
	result, err := qc.Invoker.Invoke(ctx, reviewTask)
	if err != nil {
		return nil, fmt.Errorf("QC review failed: %w", err)
	}

	// Parse the review response
	flag, feedback := ParseReviewResponse(result.Output)

	return &ReviewResult{
		Flag:     flag,
		Feedback: feedback,
	}, nil
}

// ShouldRetry determines if a task should be retried based on the QC result
func (qc *QualityController) ShouldRetry(result *ReviewResult, currentAttempt int) bool {
	// Only retry if the flag is RED
	if result.Flag != models.StatusRed {
		return false
	}

	// Check if we haven't exceeded max retries
	return currentAttempt < qc.MaxRetries
}
