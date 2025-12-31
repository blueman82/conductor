package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/harrison/conductor/internal/agent"
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

// RequireJustification returns whether STOP justification is required for custom implementations.
// Returns false if the hook or config is nil (graceful degradation).
func (h *PatternIntelligenceHook) RequireJustification() bool {
	if h == nil || h.config == nil {
		return false
	}
	return h.config.RequireJustification
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
// Uses XML format for structured content per Claude 4 best practices.
func (h *PatternIntelligenceHook) buildPromptInjection(stopResult *pattern.STOPResult, dupResult *pattern.DuplicateResult) string {
	if !h.config.InjectIntoPrompt {
		return ""
	}

	var sb strings.Builder

	// Add STOP analysis if available
	if stopResult != nil && stopResult.Confidence >= h.config.MinConfidence {
		sb.WriteString("\n<pattern_intelligence>\n")

		// Search results - SimilarPatterns uses PatternMatch type
		if len(stopResult.Search.SimilarPatterns) > 0 || len(stopResult.Search.RelatedFiles) > 0 {
			sb.WriteString("<similar_patterns>\n")
			for i, p := range stopResult.Search.SimilarPatterns {
				if i >= h.config.MaxPatternsPerTask {
					break
				}
				sb.WriteString(fmt.Sprintf("<pattern name=\"%s\" path=\"%s\" similarity=\"%.0f%%\">%s</pattern>\n",
					p.Name, p.FilePath, p.Similarity*100, p.Description))
			}
			if len(stopResult.Search.RelatedFiles) > 0 {
				maxFiles := h.config.MaxRelatedFiles
				if maxFiles > len(stopResult.Search.RelatedFiles) {
					maxFiles = len(stopResult.Search.RelatedFiles)
				}
				sb.WriteString(agent.XMLTag("related_files", strings.Join(stopResult.Search.RelatedFiles[:maxFiles], ", ")))
				sb.WriteString("\n")
			}
			sb.WriteString("</similar_patterns>\n")
		}

		// Think analysis
		if stopResult.Think.ComplexityScore > 0 {
			sb.WriteString(fmt.Sprintf("<analysis complexity=\"%d\" effort=\"%s\">\n",
				stopResult.Think.ComplexityScore, stopResult.Think.EstimatedEffort))
			if len(stopResult.Think.ApproachSuggestions) > 0 {
				sb.WriteString("<suggestions>\n")
				for _, suggestion := range stopResult.Think.ApproachSuggestions {
					sb.WriteString(agent.XMLTag("item", suggestion))
					sb.WriteString("\n")
				}
				sb.WriteString("</suggestions>\n")
			}
			if len(stopResult.Think.RiskFactors) > 0 {
				sb.WriteString("<risks>\n")
				for _, rf := range stopResult.Think.RiskFactors {
					sb.WriteString(fmt.Sprintf("<risk severity=\"%s\" mitigation=\"%s\">%s</risk>\n",
						rf.Severity, rf.Mitigation, rf.Name))
				}
				sb.WriteString("</risks>\n")
			}
			sb.WriteString("</analysis>\n")
		}

		// Outline steps - uses OutlineStep type
		if len(stopResult.Outline.Steps) > 0 {
			sb.WriteString("<implementation_steps>\n")
			for _, step := range stopResult.Outline.Steps {
				if step.TestStrategy != "" {
					sb.WriteString(fmt.Sprintf("<step order=\"%d\" test=\"%s\">%s</step>\n",
						step.Order, step.TestStrategy, step.Description))
				} else {
					sb.WriteString(fmt.Sprintf("<step order=\"%d\">%s</step>\n",
						step.Order, step.Description))
				}
			}
			sb.WriteString("</implementation_steps>\n")
		}

		// Prove verification
		if len(stopResult.Prove.TestCommands) > 0 {
			sb.WriteString("<verification_commands>\n")
			for _, cmd := range stopResult.Prove.TestCommands {
				sb.WriteString(agent.XMLTag("command", cmd))
				sb.WriteString("\n")
			}
			sb.WriteString("</verification_commands>\n")
		}

		// Recommendations
		if len(stopResult.Recommendations) > 0 {
			sb.WriteString("<recommendations>\n")
			for _, rec := range stopResult.Recommendations {
				sb.WriteString(agent.XMLTag("item", rec))
				sb.WriteString("\n")
			}
			sb.WriteString("</recommendations>\n")
		}

		sb.WriteString("</pattern_intelligence>\n")
	}

	// Add duplicate warning if applicable
	if dupResult != nil && dupResult.IsDuplicate {
		if sb.Len() == 0 {
			sb.WriteString("\n<pattern_intelligence>\n")
		} else {
			// Remove closing tag to add duplicate section inside
			content := sb.String()
			content = strings.TrimSuffix(content, "</pattern_intelligence>\n")
			sb.Reset()
			sb.WriteString(content)
		}
		sb.WriteString(fmt.Sprintf("<duplicate_warning similarity=\"%.0f%%\">\n",
			dupResult.SimilarityScore*100))
		sb.WriteString("<similar_tasks>\n")
		for _, dup := range dupResult.DuplicateOf {
			sb.WriteString(fmt.Sprintf("<task name=\"%s\" similarity=\"%.0f%%\"/>\n",
				dup.TaskName, dup.SimilarityScore*100))
		}
		sb.WriteString("</similar_tasks>\n")
		if len(dupResult.OverlapAreas) > 0 {
			sb.WriteString(agent.XMLTag("overlap_areas", strings.Join(dupResult.OverlapAreas, ", ")))
			sb.WriteString("\n")
		}
		sb.WriteString(agent.XMLTag("action", "Review existing implementations before proceeding"))
		sb.WriteString("\n")
		sb.WriteString("</duplicate_warning>\n")
		sb.WriteString("</pattern_intelligence>\n")
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
