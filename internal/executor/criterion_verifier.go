package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	sb.WriteString("## CRITERION VERIFICATION RESULTS\n\n")

	allPassed := true
	for _, r := range results {
		if !r.Passed {
			allPassed = false
		}

		status := "✅ PASS"
		if !r.Passed {
			status = "❌ FAIL"
		}

		sb.WriteString(fmt.Sprintf("### Criterion %d: %s [%s]\n", r.Index, r.Criterion, status))

		if r.Description != "" {
			sb.WriteString(fmt.Sprintf("*%s*\n", r.Description))
		}

		sb.WriteString(fmt.Sprintf("**Command:** `%s`\n", r.Command))

		if r.Expected != "" {
			sb.WriteString(fmt.Sprintf("**Expected:** `%s`\n", r.Expected))
		}

		if r.Output != "" {
			sb.WriteString(fmt.Sprintf("**Output:**\n```\n%s\n```\n", strings.TrimSpace(r.Output)))
		}

		if r.Error != nil {
			sb.WriteString(fmt.Sprintf("**Error:** %v\n", r.Error))
		}

		sb.WriteString(fmt.Sprintf("**Duration:** %v\n\n", r.Duration.Round(time.Millisecond)))
	}

	passCount := 0
	for _, r := range results {
		if r.Passed {
			passCount++
		}
	}

	if allPassed {
		sb.WriteString(fmt.Sprintf("**Summary:** All %d criterion verifications passed ✅\n", len(results)))
	} else {
		sb.WriteString(fmt.Sprintf("**Summary:** %d/%d criterion verifications passed ⚠️\n", passCount, len(results)))
	}

	return sb.String()
}
