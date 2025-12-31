package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

// RunCriterionVerifications executes optional verification commands for each success criterion.
// Unlike test commands, verification failures do NOT block execution - they feed into QC prompt.
// Returns nil error even if some verifications fail (QC decides final verdict).
// Only returns error for context cancellation/timeout.
func RunCriterionVerifications(ctx context.Context, runner CommandRunner, task models.Task) ([]CriterionVerificationResult, error) {
	// No structured criteria - nothing to do
	if len(task.StructuredCriteria) == 0 {
		return nil, nil
	}

	var results []CriterionVerificationResult

	for i, criterion := range task.StructuredCriteria {
		// Skip criteria without verification blocks
		if criterion.Verification == nil {
			continue
		}

		// Check context before running
		if ctx.Err() != nil {
			return results, ctx.Err()
		}

		verification := criterion.Verification
		start := time.Now()
		output, err := runner.Run(ctx, verification.Command)
		duration := time.Since(start)

		// Determine if passed
		passed := err == nil

		// If expected output is specified, also check for match
		if passed && verification.Expected != "" {
			trimmedOutput := strings.TrimSpace(output)
			trimmedExpected := strings.TrimSpace(verification.Expected)
			passed = trimmedOutput == trimmedExpected
		}

		result := CriterionVerificationResult{
			Index:       i,
			Criterion:   criterion.Criterion,
			Command:     verification.Command,
			Output:      output,
			Expected:    verification.Expected,
			Error:       err,
			Passed:      passed,
			Duration:    duration,
			Description: verification.Description,
		}
		results = append(results, result)

		// NOTE: We do NOT return on failure - continue to verify all criteria
		// Verification failures feed into QC prompt for final judgment
	}

	return results, nil
}

// FormatCriterionResults formats criterion verification results for injection into QC prompt.
// Returns empty string if no results.
func FormatCriterionResults(results []CriterionVerificationResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<criterion_verification_results>\n")

	allPassed := true
	passCount := 0
	for _, r := range results {
		if !r.Passed {
			allPassed = false
		} else {
			passCount++
		}

		status := "passed"
		if !r.Passed {
			status = "failed"
		}

		sb.WriteString(fmt.Sprintf("<criterion index=\"%d\" status=\"%s\">\n", r.Index, status))
		sb.WriteString(agent.XMLTag("text", r.Criterion))
		sb.WriteString("\n")

		if r.Description != "" {
			sb.WriteString(agent.XMLTag("description", r.Description))
			sb.WriteString("\n")
		}

		sb.WriteString(agent.XMLTag("command", r.Command))
		sb.WriteString("\n")

		if r.Expected != "" {
			sb.WriteString(agent.XMLTag("expected", r.Expected))
			sb.WriteString("\n")
		}

		if r.Output != "" {
			sb.WriteString(agent.XMLSection("output", strings.TrimSpace(r.Output)))
			sb.WriteString("\n")
		}

		if r.Error != nil {
			sb.WriteString(agent.XMLTag("error", fmt.Sprintf("%v", r.Error)))
			sb.WriteString("\n")
		}

		sb.WriteString(agent.XMLTag("duration", r.Duration.Round(time.Millisecond).String()))
		sb.WriteString("\n")
		sb.WriteString("</criterion>\n")
	}

	if allPassed {
		sb.WriteString(fmt.Sprintf("<summary>All %d criterion verifications passed</summary>\n", len(results)))
	} else {
		sb.WriteString(fmt.Sprintf("<summary>%d/%d criterion verifications passed</summary>\n", passCount, len(results)))
	}

	sb.WriteString("</criterion_verification_results>\n")
	return sb.String()
}
