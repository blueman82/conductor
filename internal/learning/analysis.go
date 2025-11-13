package learning

import (
	"context"
	"fmt"
	"strings"
)

// FailureAnalysis contains metrics and recommendations from analyzing task execution history
type FailureAnalysis struct {
	TotalAttempts           int
	FailedAttempts          int
	TriedAgents             []string
	CommonPatterns          []string
	SuggestedAgent          string
	SuggestedApproach       string
	ShouldTryDifferentAgent bool
}

// AnalyzeFailures examines execution history to identify patterns and suggest alternative agents
func (s *Store) AnalyzeFailures(ctx context.Context, planFile, taskNumber string) (*FailureAnalysis, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	analysis := &FailureAnalysis{
		TriedAgents:    make([]string, 0),
		CommonPatterns: make([]string, 0),
	}

	// Query execution history for this task
	history, err := s.GetExecutionHistory(ctx, planFile, taskNumber)
	if err != nil {
		return nil, fmt.Errorf("get execution history: %w", err)
	}

	// If no history, return empty analysis
	if len(history) == 0 {
		return analysis, nil
	}

	// Count attempts and failures
	analysis.TotalAttempts = len(history)
	failedOutputs := make([]string, 0)
	triedAgentsMap := make(map[string]bool)

	for _, exec := range history {
		if !exec.Success {
			analysis.FailedAttempts++
			if exec.Output != "" {
				failedOutputs = append(failedOutputs, exec.Output)
			}
		}
		if exec.Agent != "" {
			triedAgentsMap[exec.Agent] = true
		}
	}

	// Convert tried agents map to slice
	for agent := range triedAgentsMap {
		analysis.TriedAgents = append(analysis.TriedAgents, agent)
	}

	// Extract common failure patterns
	analysis.CommonPatterns = extractFailurePatterns(failedOutputs)

	// Determine if we should try a different agent (threshold: 2+ failures)
	if analysis.FailedAttempts >= 2 {
		analysis.ShouldTryDifferentAgent = true

		// Find best alternative agent
		suggestedAgent, reason := s.findBestAlternativeAgent(ctx, analysis.TriedAgents)
		analysis.SuggestedAgent = suggestedAgent

		// Generate approach suggestion
		analysis.SuggestedApproach = generateApproachSuggestion(analysis.CommonPatterns, suggestedAgent)

		// Add reason to approach if available
		if reason != "" {
			analysis.SuggestedApproach += fmt.Sprintf("\n\nReason: %s", reason)
		}
	}

	return analysis, nil
}

// extractFailurePatterns identifies common error patterns from failure outputs
func extractFailurePatterns(outputs []string) []string {
	patternMatches := make(map[string]bool)

	// Pattern keywords for failure analysis - matches task.go keywords
	patternKeywords := map[string][]string{
		"compilation_error": {
			"compilation_error",
			"compilation error", "compilation fail",
			"build fail", "build error", "parse error",
			"code won't compile", "unable to build",
			"compilation failed",
		},
		"test_failure": {
			"test_failure",
			"test fail", "tests fail", "test failure",
			"assertion fail", "verification fail",
			"check fail", "validation fail",
		},
		"dependency_missing": {
			"dependency_missing",
			"dependency", "package not found", "module not found",
			"unable to locate", "missing package",
			"import error", "cannot find module",
		},
		"permission_denied": {
			"permission_denied",
			"permission", "access denied", "forbidden",
			"unauthorized",
		},
		"timeout": {
			"timeout", "deadline", "timed out",
			"request timeout", "execution timeout",
			"deadline exceeded",
		},
		"runtime_error": {
			"runtime_error",
			"runtime error", "panic", "segfault", "nil pointer",
			"null reference", "stack overflow",
			"segmentation fault",
		},
		"syntax_error": {
			"syntax_error",
			"syntax error", "syntax fail",
		},
		"type_error": {
			"type_error",
			"type error", "type mismatch", "type fail",
		},
	}

	// Search for pattern keywords in outputs
	for _, output := range outputs {
		outputLower := strings.ToLower(output)

		// Check each pattern category
		for pattern, keywords := range patternKeywords {
			if patternMatches[pattern] {
				continue // Already found this pattern
			}

			// Check if any keyword matches
			for _, keyword := range keywords {
				if strings.Contains(outputLower, strings.ToLower(keyword)) {
					patternMatches[pattern] = true
					break // Only add pattern once
				}
			}
		}
	}

	// Convert to slice (only patterns that matched)
	result := make([]string, 0, len(patternMatches))
	for pattern := range patternMatches {
		result = append(result, pattern)
	}

	return result
}

// findBestAlternativeAgent suggests an alternative agent based on historical success
func (s *Store) findBestAlternativeAgent(ctx context.Context, triedAgents []string) (string, string) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return "general-purpose", "context cancelled"
	}

	// Query for agents with successful executions, excluding tried agents
	query := `
		SELECT agent, COUNT(*) as success_count
		FROM task_executions
		WHERE success = 1 AND agent IS NOT NULL AND agent != ''
	`

	// Add exclusion for tried agents
	if len(triedAgents) > 0 {
		placeholders := make([]string, len(triedAgents))
		for i := range triedAgents {
			placeholders[i] = "?"
		}
		query += fmt.Sprintf(" AND agent NOT IN (%s)", strings.Join(placeholders, ","))
	}

	query += `
		GROUP BY agent
		HAVING COUNT(*) >= 5
		ORDER BY success_count DESC
		LIMIT 1
	`

	// Execute query
	args := make([]interface{}, len(triedAgents))
	for i, agent := range triedAgents {
		args[i] = agent
	}

	var bestAgent string
	var successCount int
	err := s.db.QueryRow(query, args...).Scan(&bestAgent, &successCount)
	if err != nil {
		// No suitable alternative found, return general-purpose agent
		return "general-purpose", "no statistically significant alternatives found"
	}

	reason := fmt.Sprintf("%d successful executions", successCount)
	return bestAgent, reason
}

// generateApproachSuggestion creates human-readable recommendations based on patterns
func generateApproachSuggestion(patterns []string, alternativeAgent string) string {
	if len(patterns) == 0 {
		return fmt.Sprintf("Try using the %s agent for a different perspective on this task.", alternativeAgent)
	}

	var suggestions []string

	// Generate specific suggestions based on patterns
	patternSuggestions := map[string]string{
		"compilation_error":  "Focus on fixing compilation errors. Review syntax, type definitions, and imports carefully.",
		"test_failure":       "Investigate test failures. Check assertions, test data, and expected vs actual behavior.",
		"dependency_missing": "Resolve missing dependencies. Verify all required packages are installed and versions are compatible.",
		"timeout":            "Address timeout issues. Consider breaking down the task, optimizing performance, or increasing timeout limits.",
		"syntax_error":       "Fix syntax errors. Review language syntax rules and code structure.",
		"type_error":         "Resolve type mismatches. Check type definitions and ensure proper type conversions.",
		"import_error":       "Fix import/module errors. Verify import paths and package availability.",
		"runtime_error":      "Debug runtime errors. Add error handling and validate input data.",
		"permission_denied":  "Resolve permission issues. Check file/directory permissions and access rights.",
		"file_not_found":     "Fix file path issues. Verify file paths are correct and files exist.",
	}

	for _, pattern := range patterns {
		if suggestion, exists := patternSuggestions[pattern]; exists {
			suggestions = append(suggestions, suggestion)
		}
	}

	// Combine suggestions
	result := fmt.Sprintf("Detected patterns: %s\n\n", strings.Join(patterns, ", "))
	if len(suggestions) > 0 {
		result += strings.Join(suggestions, "\n\n")
		result += fmt.Sprintf("\n\nRecommendation: Try using the %s agent, which may handle these issues better.", alternativeAgent)
	} else {
		result += fmt.Sprintf("Try using the %s agent for a different approach.", alternativeAgent)
	}

	return result
}
