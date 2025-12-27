package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/models"
)

// WarmUpHook wraps learning.WarmUpProvider to provide warm-up context injection for task execution.
// This is a thin adapter layer that:
// - Builds warm-up context from similar successful tasks
// - Injects context into task prompts using standardized format
// - Handles graceful degradation if provider is unavailable
type WarmUpHook struct {
	provider learning.WarmUpProvider
	logger   RuntimeEnforcementLogger
}

// NewWarmUpHook creates a new WarmUpHook.
// Returns nil if provider is nil (graceful degradation pattern consistent with other hooks).
func NewWarmUpHook(provider learning.WarmUpProvider, logger RuntimeEnforcementLogger) *WarmUpHook {
	if provider == nil {
		return nil
	}
	return &WarmUpHook{
		provider: provider,
		logger:   logger,
	}
}

// InjectContext builds warm-up context and injects it into the task prompt.
// Called in preTaskHook AFTER failure analysis but BEFORE agent invocation.
// Returns the modified task with injected warm-up context in the prompt.
func (h *WarmUpHook) InjectContext(ctx context.Context, task models.Task) (models.Task, error) {
	if h == nil || h.provider == nil {
		return task, nil // Graceful degradation
	}

	// Build TaskInfo from models.Task
	taskInfo := &learning.TaskInfo{
		TaskNumber: task.Number,
		TaskName:   task.Name,
		FilePaths:  extractFilePaths(task),
		PlanFile:   task.SourceFile,
	}

	// Build warm-up context
	warmUpCtx, err := h.provider.BuildContext(ctx, taskInfo)
	if err != nil {
		if h.logger != nil {
			h.logger.Warnf("WarmUp: failed to build context for task %s: %v", task.Number, err)
		}
		return task, nil // Graceful degradation - don't fail task on warm-up error
	}

	// Check if we have useful context to inject
	if warmUpCtx == nil || warmUpCtx.Confidence < 0.3 {
		// Low confidence or no context - skip injection
		return task, nil
	}

	// Format and inject warm-up context
	injection := FormatWarmUpContext(warmUpCtx)
	if injection == "" {
		return task, nil
	}

	// Inject into prompt
	task.Prompt = injection + "\n\n" + task.Prompt

	if h.logger != nil {
		h.logger.Infof("WarmUp: Injected context with %.0f%% confidence for task %s", warmUpCtx.Confidence*100, task.Number)
	}

	return task, nil
}

// extractFilePaths extracts file paths from task metadata and structured criteria.
func extractFilePaths(task models.Task) []string {
	var paths []string

	// Extract from Metadata if available
	if task.Metadata != nil {
		if files, ok := task.Metadata["target_files"].([]string); ok {
			paths = append(paths, files...)
		}
		if files, ok := task.Metadata["files"].([]string); ok {
			paths = append(paths, files...)
		}
	}

	// Extract from structured criteria verification commands
	for _, criterion := range task.StructuredCriteria {
		if criterion.Verification != nil && criterion.Verification.Command != "" {
			// Try to extract file paths from verification commands
			cmd := criterion.Verification.Command
			// Look for common file path patterns
			if strings.Contains(cmd, ".go") || strings.Contains(cmd, ".ts") ||
				strings.Contains(cmd, ".js") || strings.Contains(cmd, ".py") {
				// This is a heuristic - could be improved with proper parsing
				words := strings.Fields(cmd)
				for _, word := range words {
					if strings.Contains(word, "/") && !strings.HasPrefix(word, "-") {
						paths = append(paths, word)
					}
				}
			}
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	unique := make([]string, 0, len(paths))
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}

	return unique
}

// FormatWarmUpContext formats a WarmUpContext into prompt injection string.
// Uses the standardized format: --- WARM-UP CONTEXT ---\n[context]\n--- END WARM-UP ---
func FormatWarmUpContext(ctx *learning.WarmUpContext) string {
	if ctx == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("--- WARM-UP CONTEXT ---\n")
	sb.WriteString(fmt.Sprintf("Historical context (confidence: %.0f%%):\n\n", ctx.Confidence*100))

	// Add recommended approach if available
	if ctx.RecommendedApproach != "" {
		sb.WriteString("## Recommended Approach\n")
		sb.WriteString(ctx.RecommendedApproach)
		sb.WriteString("\n\n")
	}

	// Add similar patterns if available
	if len(ctx.SimilarPatterns) > 0 {
		sb.WriteString("## Patterns from Similar Tasks\n")
		for i, pattern := range ctx.SimilarPatterns {
			if i >= 5 {
				break // Limit to 5 patterns to avoid prompt bloat
			}
			sb.WriteString(fmt.Sprintf("- %s\n", pattern))
		}
		sb.WriteString("\n")
	}

	// Add relevant history summary if available
	if len(ctx.RelevantHistory) > 0 {
		sb.WriteString("## Similar Historical Tasks\n")
		successCount := 0
		failCount := 0
		for _, exec := range ctx.RelevantHistory {
			if exec.Success {
				successCount++
			} else {
				failCount++
			}
		}
		sb.WriteString(fmt.Sprintf("Found %d similar tasks (%d successful, %d failed)\n", len(ctx.RelevantHistory), successCount, failCount))

		// Add brief details for top 3 successful tasks
		shown := 0
		for _, exec := range ctx.RelevantHistory {
			if exec.Success && shown < 3 {
				agent := exec.Agent
				if agent == "" {
					agent = "default"
				}
				sb.WriteString(fmt.Sprintf("- \"%s\" succeeded with agent %s", exec.TaskName, agent))
				if exec.QCVerdict == "GREEN" {
					sb.WriteString(" (GREEN)")
				}
				sb.WriteString("\n")
				shown++
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("--- END WARM-UP ---")

	return sb.String()
}
