package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/harrison/conductor/internal/estimation"
	"github.com/harrison/conductor/internal/models"
)

// EstimationHook estimates human developer time before task execution
// and calculates speedup ratio after completion.
// This hook follows the same graceful degradation pattern as LOCTrackerHook,
// allowing execution to continue even if estimation fails.
type EstimationHook struct {
	Enabled   bool
	Estimator *estimation.Estimator
	Logger    RuntimeEnforcementLogger
}

// NewEstimationHook creates a new estimation hook.
// Returns nil if not enabled (graceful disable pattern consistent with other hooks).
func NewEstimationHook(enabled bool, estimator *estimation.Estimator, logger RuntimeEnforcementLogger) *EstimationHook {
	if !enabled || estimator == nil {
		return nil
	}
	return &EstimationHook{
		Enabled:   enabled,
		Estimator: estimator,
		Logger:    logger,
	}
}

// PreTask estimates human developer time before task execution.
// Stores the estimate in task.HumanEstimateSecs and task.HumanEstimateSource.
// Errors log warning but do not block execution (graceful degradation).
func (h *EstimationHook) PreTask(ctx context.Context, task *models.Task) error {
	if h == nil || !h.Enabled || h.Estimator == nil {
		return nil
	}

	estimateSecs, source, err := h.Estimator.EstimateTask(ctx, task)
	if err != nil {
		GracefulWarn(h.Logger, "Estimation: Failed to estimate human time for task %s: %v", task.Number, err)
		return nil // Graceful degradation
	}

	task.HumanEstimateSecs = estimateSecs
	task.HumanEstimateSource = source

	// Format duration for logging
	humanDuration := time.Duration(estimateSecs) * time.Second
	GracefulInfo(h.Logger, "Estimation: Task %s - Human estimate: %s", task.Number, formatEstimationDuration(humanDuration))

	return nil
}

// PostTask logs the speedup ratio after task completion.
// Uses task.HumanEstimateSecs and task.ExecutionDuration to calculate speedup.
// Errors log warning but do not block execution (graceful degradation).
func (h *EstimationHook) PostTask(ctx context.Context, task *models.Task) error {
	if h == nil || !h.Enabled {
		return nil
	}

	speedup := task.CalculateSpeedup()
	if speedup == 0 {
		return nil // No estimate or no duration
	}

	humanDuration := task.GetHumanEstimate()
	GracefulInfo(h.Logger, "Estimation: Task %s - %.1fx faster than human (human: %s, actual: %s)",
		task.Number, speedup, formatEstimationDuration(humanDuration), formatEstimationDuration(task.ExecutionDuration))

	return nil
}

// formatEstimationDuration formats a duration in a human-readable way
func formatEstimationDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs > 0 {
			return fmt.Sprintf("%dm%ds", mins, secs)
		}
		return fmt.Sprintf("%dm", mins)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins > 0 {
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
	return fmt.Sprintf("%dh", hours)
}
