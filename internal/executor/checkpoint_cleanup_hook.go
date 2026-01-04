package executor

import (
	"context"
	"time"

	"github.com/harrison/conductor/internal/config"
)

// CheckpointCleanupHook provides age-based cleanup of stale checkpoint branches.
// This hook scans for checkpoint branches older than the configured retention period
// and deletes them to prevent branch accumulation over time.
//
// The hook follows conductor's standard hook patterns:
// - Constructor validates deps, returns nil if missing (graceful degradation)
// - Nil-receiver safety on all methods
// - Injected clock (now func) for deterministic testing
// - Individual deletion failures don't stop cleanup
type CheckpointCleanupHook struct {
	checkpointer GitCheckpointer
	config       *config.RollbackConfig
	logger       RuntimeEnforcementLogger
	now          func() time.Time // Injected for testing, defaults to time.Now
}

// NewCheckpointCleanupHook creates a new CheckpointCleanupHook.
// Returns nil if checkpointer or config is nil (graceful degradation pattern
// consistent with NewSetupHook, NewRollbackHook).
func NewCheckpointCleanupHook(
	checkpointer GitCheckpointer,
	cfg *config.RollbackConfig,
	logger RuntimeEnforcementLogger,
) *CheckpointCleanupHook {
	if checkpointer == nil || cfg == nil {
		return nil
	}
	return &CheckpointCleanupHook{
		checkpointer: checkpointer,
		config:       cfg,
		logger:       logger,
		now:          time.Now,
	}
}

// Cleanup scans for checkpoint branches older than KeepCheckpointDays and deletes them.
// Returns the number of successfully deleted branches.
//
// Behavior:
// - Returns (0, nil) if hook is nil, rollback disabled, or KeepCheckpointDays <= 0
// - Continues deleting remaining branches even if individual deletions fail
// - Logs warnings for deletion failures via logger.Warnf
//
// Parameters:
//   - ctx: Context for cancellation
//
// Returns:
//   - int: Number of successfully deleted checkpoint branches
//   - error: Only returned for listing errors; individual deletion failures are logged
func (h *CheckpointCleanupHook) Cleanup(ctx context.Context) (int, error) {
	// Nil-receiver safety
	if h == nil {
		return 0, nil
	}

	// Skip if rollback not enabled or no retention period configured
	if !h.config.Enabled || h.config.KeepCheckpointDays <= 0 {
		return 0, nil
	}

	// List all checkpoint branches
	checkpoints, err := h.checkpointer.ListCheckpoints(ctx)
	if err != nil {
		return 0, err
	}

	if len(checkpoints) == 0 {
		return 0, nil
	}

	// Calculate cutoff time using injected clock
	cutoff := h.now().AddDate(0, 0, -h.config.KeepCheckpointDays)

	if h.logger != nil {
		h.logger.Infof("CheckpointCleanup: Scanning %d checkpoint branches (cutoff: %s)",
			len(checkpoints), cutoff.Format("2006-01-02"))
	}

	// Filter and delete stale checkpoints
	deleted := 0
	for _, checkpoint := range checkpoints {
		// Skip checkpoints with invalid timestamps (zero time)
		if checkpoint.CreatedAt.IsZero() {
			if h.logger != nil {
				h.logger.Warnf("CheckpointCleanup: Skipping branch '%s' with unparseable timestamp",
					checkpoint.BranchName)
			}
			continue
		}

		// Skip checkpoints newer than cutoff
		if checkpoint.CreatedAt.After(cutoff) {
			continue
		}

		// Delete stale checkpoint
		if err := h.checkpointer.DeleteCheckpoint(ctx, checkpoint.BranchName); err != nil {
			// Log warning but continue with remaining branches (graceful degradation)
			if h.logger != nil {
				h.logger.Warnf("CheckpointCleanup: Failed to delete stale branch '%s': %v",
					checkpoint.BranchName, err)
			}
			continue
		}

		deleted++
		if h.logger != nil {
			h.logger.Infof("CheckpointCleanup: Deleted stale branch '%s' (created: %s)",
				checkpoint.BranchName, checkpoint.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	}

	if h.logger != nil && deleted > 0 {
		h.logger.Infof("CheckpointCleanup: Completed - deleted %d stale checkpoint branches", deleted)
	}

	return deleted, nil
}

// Enabled returns whether the cleanup hook is enabled and can perform cleanup.
func (h *CheckpointCleanupHook) Enabled() bool {
	if h == nil || h.config == nil {
		return false
	}
	return h.config.Enabled && h.config.KeepCheckpointDays > 0
}
