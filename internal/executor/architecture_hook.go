package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/architecture"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/models"
)

// ArchitectureCheckpointHook handles architecture assessment before task execution
type ArchitectureCheckpointHook struct {
	assessor *architecture.Assessor
	config   *config.ArchitectureConfig
	logger   RuntimeEnforcementLogger
}

// NewArchitectureCheckpointHook creates a new architecture checkpoint hook.
// Returns nil if disabled or assessor is nil (graceful degradation).
func NewArchitectureCheckpointHook(
	assessor *architecture.Assessor,
	cfg *config.ArchitectureConfig,
	logger RuntimeEnforcementLogger,
) *ArchitectureCheckpointHook {
	if assessor == nil || cfg == nil || !cfg.Enabled {
		return nil
	}
	return &ArchitectureCheckpointHook{
		assessor: assessor,
		config:   cfg,
		logger:   logger,
	}
}

// CheckTask runs architecture assessment on a task before execution.
// Returns CheckpointResult with blocking/warning decisions based on mode.
func (h *ArchitectureCheckpointHook) CheckTask(ctx context.Context, task models.Task) (*architecture.CheckpointResult, error) {
	if h == nil || h.assessor == nil || h.config == nil || !h.config.Enabled {
		return &architecture.CheckpointResult{}, nil
	}

	// Run architecture assessment
	assessment, err := h.assessor.Assess(ctx, task)
	if err != nil {
		GracefulWarn(h.logger, "Architecture assessment failed for task %s: %v", task.Number, err)
		return &architecture.CheckpointResult{}, nil // Graceful degradation
	}

	result := &architecture.CheckpointResult{
		Assessment: assessment,
	}

	// Check for low confidence escalation
	if h.config.EscalateOnUncertain && assessment.OverallConfidence < h.config.ConfidenceThreshold {
		result.ShouldEscalate = true
		GracefulWarn(h.logger, "Architecture assessment low confidence (%.0f%%) for task %s - escalating",
			assessment.OverallConfidence*100, task.Number)
	}

	// Handle mode-specific behavior
	switch h.config.Mode {
	case config.ArchitectureModeBlock:
		if assessment.RequiresReview {
			result.ShouldBlock = true
			result.BlockReason = fmt.Sprintf("Architecture review required: %s. Flagged: %s",
				assessment.Summary, strings.Join(assessment.FlaggedQuestions(), ", "))
			GracefulWarn(h.logger, "Architecture checkpoint BLOCKED task %s: %s", task.Number, result.BlockReason)
		}

	case config.ArchitectureModeEscalate:
		if assessment.RequiresReview {
			result.ShouldEscalate = true
			GracefulWarn(h.logger, "Architecture checkpoint ESCALATE for task %s: %s (flagged: %s)",
				task.Number, assessment.Summary, strings.Join(assessment.FlaggedQuestions(), ", "))
		}
		// Always build prompt injection in escalate mode
		result.PromptInjection = h.buildPromptInjection(assessment)
	}

	return result, nil
}

// buildPromptInjection creates architecture context for injection into agent prompt.
// Uses XML format for structured content per Claude 4 best practices.
func (h *ArchitectureCheckpointHook) buildPromptInjection(assessment *architecture.AssessmentResult) string {
	if assessment == nil || !assessment.RequiresReview {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n<architecture_checkpoint>\n")
	sb.WriteString(agent.XMLTag("assessment", assessment.Summary))
	sb.WriteString("\n")

	flagged := assessment.FlaggedQuestions()
	if len(flagged) > 0 {
		sb.WriteString("<architectural_concerns>\n")
		for _, q := range flagged {
			sb.WriteString(agent.XMLTag("concern", q))
			sb.WriteString("\n")
		}
		sb.WriteString("</architectural_concerns>\n")
	}

	// Add specific reasoning for flagged questions
	sb.WriteString("<reasoning>\n")
	if assessment.CoreInfrastructure.Answer {
		sb.WriteString(fmt.Sprintf("<core_infrastructure>%s</core_infrastructure>\n", assessment.CoreInfrastructure.Reasoning))
	}
	if assessment.ReuseConcerns.Answer {
		sb.WriteString(fmt.Sprintf("<reuse_concerns>%s</reuse_concerns>\n", assessment.ReuseConcerns.Reasoning))
	}
	if assessment.NewAbstractions.Answer {
		sb.WriteString(fmt.Sprintf("<new_abstractions>%s</new_abstractions>\n", assessment.NewAbstractions.Reasoning))
	}
	if assessment.APIContracts.Answer {
		sb.WriteString(fmt.Sprintf("<api_contracts>%s</api_contracts>\n", assessment.APIContracts.Reasoning))
	}
	if assessment.FrameworkLifecycle.Answer {
		sb.WriteString(fmt.Sprintf("<framework_lifecycle>%s</framework_lifecycle>\n", assessment.FrameworkLifecycle.Reasoning))
	}
	if assessment.CrossCuttingConcerns.Answer {
		sb.WriteString(fmt.Sprintf("<cross_cutting_concerns>%s</cross_cutting_concerns>\n", assessment.CrossCuttingConcerns.Reasoning))
	}
	sb.WriteString("</reasoning>\n")

	if h.config.RequireJustification {
		sb.WriteString(agent.XMLTag("important", "You must justify any architectural decisions in your output."))
		sb.WriteString("\n")
	}

	sb.WriteString("</architecture_checkpoint>\n")
	return sb.String()
}

// RequireJustification returns whether architectural justification is required
func (h *ArchitectureCheckpointHook) RequireJustification() bool {
	if h == nil || h.config == nil {
		return false
	}
	return h.config.RequireJustification
}
