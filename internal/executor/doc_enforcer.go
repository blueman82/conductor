package executor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/harrison/conductor/internal/models"
)

// Documentation enforcement errors
var (
	// ErrDocSectionNotFound indicates the expected documentation section was not found.
	ErrDocSectionNotFound = errors.New("documentation section not found")

	// ErrDocFileNotFound indicates the documentation file does not exist.
	ErrDocFileNotFound = errors.New("documentation file not found")
)

// VerifyDocumentationTargets checks that all documentation targets in a task exist and contain
// the expected sections. This ensures agents edit the exact sections specified in the plan.
//
// Returns results for each target (even on failure) and only returns error for context cancellation.
// Individual target failures are recorded in DocTargetResult.Error.
func VerifyDocumentationTargets(ctx context.Context, task models.Task) ([]DocTargetResult, error) {
	// No metadata or no targets - nothing to do
	if task.RuntimeMetadata == nil {
		return nil, nil
	}

	targets := task.RuntimeMetadata.DocumentationTargets
	if len(targets) == 0 {
		return nil, nil
	}

	var results []DocTargetResult

	for _, target := range targets {
		// Check context before processing
		if ctx.Err() != nil {
			return results, ctx.Err()
		}

		result := verifyDocTarget(target)
		results = append(results, result)
	}

	return results, nil
}

// verifyDocTarget checks a single documentation target.
func verifyDocTarget(target models.DocumentationTarget) DocTargetResult {
	result := DocTargetResult{
		Location: target.Location,
		Section:  target.Section,
	}

	// Check if file exists
	if _, err := os.Stat(target.Location); os.IsNotExist(err) {
		result.Passed = false
		result.Error = fmt.Errorf("%w: %s", ErrDocFileNotFound, target.Location)
		return result
	}

	// Open and scan file for section
	file, err := os.Open(target.Location)
	if err != nil {
		result.Passed = false
		result.Error = fmt.Errorf("failed to open documentation file: %w", err)
		return result
	}
	defer file.Close()

	// Scan file line by line looking for the section
	scanner := bufio.NewScanner(file)
	lineNum := 0
	found := false
	var contentLines []string

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check if this line contains the section heading
		if strings.Contains(line, target.Section) {
			found = true
			// Capture a few lines after the heading for context
			contentLines = append(contentLines, line)
			for i := 0; i < 3 && scanner.Scan(); i++ {
				contentLines = append(contentLines, scanner.Text())
			}
			break
		}
	}

	if err := scanner.Err(); err != nil {
		result.Passed = false
		result.Error = fmt.Errorf("error reading documentation file: %w", err)
		return result
	}

	if !found {
		result.Passed = false
		result.Error = fmt.Errorf("%w: section %q not found in %s", ErrDocSectionNotFound, target.Section, target.Location)
		return result
	}

	result.Passed = true
	result.LineNumber = lineNum
	result.Content = strings.Join(contentLines, "\n")
	return result
}

// FormatDocTargetResults formats documentation target verification results for injection into QC prompt.
// Returns empty string if no results.
func FormatDocTargetResults(results []DocTargetResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## DOCUMENTATION TARGET VERIFICATION\n\n")

	allPassed := true
	for _, r := range results {
		if !r.Passed {
			allPassed = false
		}

		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}

		if r.LineNumber > 0 {
			sb.WriteString(fmt.Sprintf("### %s:%d `%s` [%s]\n", r.Location, r.LineNumber, r.Section, status))
		} else {
			sb.WriteString(fmt.Sprintf("### %s `%s` [%s]\n", r.Location, r.Section, status))
		}

		if r.Content != "" {
			sb.WriteString("**Content found:**\n```\n")
			sb.WriteString(r.Content)
			sb.WriteString("\n```\n")
		}

		if r.Error != nil {
			sb.WriteString(fmt.Sprintf("**Error:** %v\n", r.Error))
		}

		sb.WriteString("\n")
	}

	passCount := 0
	for _, r := range results {
		if r.Passed {
			passCount++
		}
	}

	if allPassed {
		sb.WriteString(fmt.Sprintf("**Summary:** All %d documentation targets verified\n", len(results)))
	} else {
		sb.WriteString(fmt.Sprintf("**Summary:** %d/%d documentation targets passed\n", passCount, len(results)))
	}

	return sb.String()
}

// HasDocumentationTargets returns true if the task has documentation targets to verify.
func HasDocumentationTargets(task models.Task) bool {
	return task.RuntimeMetadata != nil && len(task.RuntimeMetadata.DocumentationTargets) > 0
}

// IsDocumentationTask returns true if the task type is "documentation".
func IsDocumentationTask(task models.Task) bool {
	return task.Type == "documentation"
}
