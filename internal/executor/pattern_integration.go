package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/models"
	"github.com/harrison/conductor/internal/pattern"
)

// PatternIntelligenceHook handles Pattern Intelligence integration for task execution.
// Implements STOP protocol (Search/Think/Outline/Prove) from CLAWED methodology
// and duplicate detection to prevent redundant work.
type PatternIntelligenceHook struct {
	pi     pattern.PatternIntelligence
	config *config.PatternConfig
	logger RuntimeEnforcementLogger
}

// NewPatternIntelligenceHook creates a new PatternIntelligenceHook.
// Returns nil if Pattern Intelligence is disabled or pi is nil (graceful degradation).
func NewPatternIntelligenceHook(pi pattern.PatternIntelligence, cfg *config.PatternConfig, logger RuntimeEnforcementLogger) *PatternIntelligenceHook {
	if pi == nil || cfg == nil || !cfg.Enabled {
		return nil
	}
	return &PatternIntelligenceHook{
		pi:     pi,
		config: cfg,
		logger: logger,
	}
}

// PreTaskCheckResult contains the result of a pre-task pattern check.
type PreTaskCheckResult struct {
	// ShouldBlock indicates if task execution should be blocked
	ShouldBlock bool

	// BlockReason explains why task is blocked (if applicable)
	BlockReason string

	// PromptInjection contains STOP protocol context to inject into agent prompt
	PromptInjection string

	// Recommendations are suggestions for the executing agent
	Recommendations []string

	// STOPResult contains the full STOP protocol analysis (if available)
	STOPResult *pattern.STOPResult

	// DuplicateResult contains duplicate detection results (if available)
	DuplicateResult *pattern.DuplicateResult
}

// CheckTask runs Pattern Intelligence analysis on a task before execution.
// Implements pre-task hook logic with mode handling (block, warn, suggest).
// Returns PreTaskCheckResult with recommendations and potential blocking.
//
// Mode handling:
//   - block: Returns ShouldBlock=true if duplicate detected above threshold
//   - warn: Logs warning but allows execution to proceed
//   - suggest: Injects STOP context into prompt without blocking
func (h *PatternIntelligenceHook) CheckTask(ctx context.Context, task models.Task) (*PreTaskCheckResult, error) {
	// Graceful degradation on nil hook or disabled PI
	if h == nil || h.pi == nil || h.config == nil || !h.config.Enabled {
		return &PreTaskCheckResult{ShouldBlock: false}, nil
	}

	// Run Pattern Intelligence check
	stopResult, dupResult, err := h.pi.CheckTask(ctx, task)
	if err != nil {
		// Graceful degradation: log warning but don't block execution
		if h.logger != nil {
			h.logger.Warnf("Pattern Intelligence check failed for task %s: %v", task.Number, err)
		}
		return &PreTaskCheckResult{ShouldBlock: false}, nil
	}

	result := &PreTaskCheckResult{
		STOPResult:      stopResult,
		DuplicateResult: dupResult,
	}

	// Get combined check result for mode handling
	checkResult := pattern.GetCheckResult(stopResult, dupResult, h.config)
	if checkResult != nil {
		result.Recommendations = checkResult.Suggestions
	}

	// Handle mode-specific behavior
	switch h.config.Mode {
	case config.PatternModeBlock:
		if checkResult != nil && checkResult.ShouldBlock {
			result.ShouldBlock = true
			result.BlockReason = checkResult.BlockReason
			if h.logger != nil {
				h.logger.Warnf("Pattern Intelligence BLOCKED task %s: %s", task.Number, checkResult.BlockReason)
			}
		}

	case config.PatternModeWarn:
		if dupResult != nil && dupResult.IsDuplicate {
			if h.logger != nil {
				h.logger.Warnf("Pattern Intelligence WARNING for task %s: potential duplicate (%.0f%% similarity)",
					task.Number, dupResult.SimilarityScore*100)
			}
		}
		// Always build prompt injection in warn mode
		result.PromptInjection = h.buildPromptInjection(stopResult, dupResult)

	case config.PatternModeSuggest:
		// In suggest mode, always inject STOP context into prompt
		result.PromptInjection = h.buildPromptInjection(stopResult, dupResult)
		if h.logger != nil && len(result.Recommendations) > 0 {
			h.logger.Infof("Pattern Intelligence suggestions for task %s: %d recommendations",
				task.Number, len(result.Recommendations))
		}
	}

	return result, nil
}

// buildPromptInjection creates STOP protocol context for injection into agent prompt.
// Returns empty string if no relevant context is available.
func (h *PatternIntelligenceHook) buildPromptInjection(stopResult *pattern.STOPResult, dupResult *pattern.DuplicateResult) string {
	if !h.config.InjectIntoPrompt {
		return ""
	}

	var sb strings.Builder

	// Add STOP analysis if available
	if stopResult != nil && stopResult.Confidence >= h.config.MinConfidence {
		sb.WriteString("\n---\n## PATTERN INTELLIGENCE CONTEXT\n\n")

		// Search results - SimilarPatterns uses PatternMatch type
		if len(stopResult.Search.SimilarPatterns) > 0 || len(stopResult.Search.RelatedFiles) > 0 {
			sb.WriteString("### Similar Patterns Found\n")
			for i, p := range stopResult.Search.SimilarPatterns {
				if i >= h.config.MaxPatternsPerTask {
					break
				}
				sb.WriteString(fmt.Sprintf("- **%s** (%s): %.0f%% similar - %s\n",
					p.Name, p.FilePath, p.Similarity*100, p.Description))
			}
			if len(stopResult.Search.RelatedFiles) > 0 {
				sb.WriteString("\n**Related Files**: ")
				maxFiles := h.config.MaxRelatedFiles
				if maxFiles > len(stopResult.Search.RelatedFiles) {
					maxFiles = len(stopResult.Search.RelatedFiles)
				}
				sb.WriteString(strings.Join(stopResult.Search.RelatedFiles[:maxFiles], ", "))
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}

		// Think analysis
		if stopResult.Think.ComplexityScore > 0 {
			sb.WriteString(fmt.Sprintf("### Analysis (Complexity: %d/10, Effort: %s)\n",
				stopResult.Think.ComplexityScore, stopResult.Think.EstimatedEffort))
			for _, suggestion := range stopResult.Think.ApproachSuggestions {
				sb.WriteString(fmt.Sprintf("- %s\n", suggestion))
			}
			for _, rf := range stopResult.Think.RiskFactors {
				sb.WriteString(fmt.Sprintf("- ⚠️ Risk [%s]: %s - Mitigation: %s\n",
					rf.Severity, rf.Name, rf.Mitigation))
			}
			sb.WriteString("\n")
		}

		// Outline steps - uses OutlineStep type
		if len(stopResult.Outline.Steps) > 0 {
			sb.WriteString("### Suggested Implementation Steps\n")
			for _, step := range stopResult.Outline.Steps {
				sb.WriteString(fmt.Sprintf("%d. %s\n", step.Order, step.Description))
				if step.TestStrategy != "" {
					sb.WriteString(fmt.Sprintf("   - Test: %s\n", step.TestStrategy))
				}
			}
			sb.WriteString("\n")
		}

		// Prove verification
		if len(stopResult.Prove.TestCommands) > 0 {
			sb.WriteString("### Verification Commands\n")
			for _, cmd := range stopResult.Prove.TestCommands {
				sb.WriteString(fmt.Sprintf("- `%s`\n", cmd))
			}
			sb.WriteString("\n")
		}

		// Recommendations
		if len(stopResult.Recommendations) > 0 {
			sb.WriteString("### Recommendations\n")
			for _, rec := range stopResult.Recommendations {
				sb.WriteString(fmt.Sprintf("- %s\n", rec))
			}
		}
	}

	// Add duplicate warning if applicable
	if dupResult != nil && dupResult.IsDuplicate {
		if sb.Len() == 0 {
			sb.WriteString("\n---\n## PATTERN INTELLIGENCE CONTEXT\n\n")
		}
		sb.WriteString("### ⚠️ Potential Duplicate Detected\n")
		sb.WriteString(fmt.Sprintf("This task has %.0f%% similarity to existing patterns.\n",
			dupResult.SimilarityScore*100))
		for _, dup := range dupResult.DuplicateOf {
			sb.WriteString(fmt.Sprintf("- Similar to: %s (%.0f%% match)\n",
				dup.TaskName, dup.SimilarityScore*100))
		}
		if len(dupResult.OverlapAreas) > 0 {
			sb.WriteString(fmt.Sprintf("Overlap areas: %s\n", strings.Join(dupResult.OverlapAreas, ", ")))
		}
		sb.WriteString("\n**Action**: Review existing implementations before proceeding.\n")
	}

	if sb.Len() > 0 {
		sb.WriteString("\n---\n")
	}

	return sb.String()
}

// RecordSuccess records a successful task execution in the pattern library.
// Called after QC GREEN verdict to store patterns for future reference.
// Gracefully handles nil hook or errors.
func (h *PatternIntelligenceHook) RecordSuccess(ctx context.Context, task models.Task, agent string) error {
	if h == nil || h.pi == nil || h.config == nil || !h.config.Enabled {
		return nil
	}

	// Type assert to get concrete implementation with RecordSuccess method
	if pi, ok := h.pi.(*pattern.PatternIntelligenceImpl); ok {
		if err := pi.RecordSuccess(ctx, task, agent); err != nil {
			if h.logger != nil {
				h.logger.Warnf("Failed to record pattern for task %s: %v", task.Number, err)
			}
			// Graceful degradation - don't fail task on pattern storage error
			return nil
		}
	}

	return nil
}

// ApplyPromptInjection applies Pattern Intelligence context to a task's prompt.
// Returns the modified task with STOP context injected if applicable.
func ApplyPromptInjection(task models.Task, result *PreTaskCheckResult) models.Task {
	if result == nil || result.PromptInjection == "" {
		return task
	}

	// Inject Pattern Intelligence context at the end of the prompt
	task.Prompt = task.Prompt + result.PromptInjection
	return task
}
