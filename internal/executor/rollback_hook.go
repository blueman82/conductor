package executor

import (
	"context"
	"fmt"
)

// RollbackHook wraps RollbackManager and GitCheckpointer for task-level checkpoint/rollback.
// This is a thin adapter layer that:
// - Creates checkpoints before agent invocation (PreTask)
// - Evaluates rollback decisions after QC review (PostTask)
// - Handles graceful degradation if manager or checkpointer is unavailable
//
// RollbackHook operates within the working branch created by BranchGuard and provides
// fine-grained task-level rollback capability.
type RollbackHook struct {
	Manager      *RollbackManager
	Checkpointer GitCheckpointer
	Logger       RuntimeEnforcementLogger
}

// NewRollbackHook creates a new RollbackHook.
// Returns nil if manager or checkpointer is nil (graceful degradation pattern consistent with other hooks).
func NewRollbackHook(manager *RollbackManager, checkpointer GitCheckpointer, logger RuntimeEnforcementLogger) *RollbackHook {
	if manager == nil || checkpointer == nil {
		return nil
	}
	return &RollbackHook{
		Manager:      manager,
		Checkpointer: checkpointer,
		Logger:       logger,
	}
}

// PreTask creates a checkpoint before agent invocation.
// Stores CheckpointInfo in task.Metadata["rollback_checkpoint"] for PostTask retrieval.
// Errors log warning but do not block execution (graceful degradation).
func (h *RollbackHook) PreTask(ctx context.Context, taskNumber int, metadata map[string]interface{}) error {
	if h == nil || h.Checkpointer == nil {
		return nil // Graceful degradation
	}

	// Skip if rollback is not enabled
	if h.Manager == nil || !h.Manager.Enabled() {
		return nil
	}

	if h.Logger != nil {
		h.Logger.Infof("Rollback: Creating checkpoint for task %d", taskNumber)
	}

	// Create checkpoint branch
	checkpoint, err := h.Checkpointer.CreateCheckpoint(ctx, taskNumber)
	if err != nil {
		if h.Logger != nil {
			h.Logger.Warnf("Rollback: Failed to create checkpoint for task %d: %v", taskNumber, err)
		}
		// Non-fatal: continue without checkpoint (graceful degradation)
		return nil
	}

	// Store checkpoint info in task metadata for PostTask retrieval
	if metadata != nil {
		metadata["rollback_checkpoint"] = checkpoint
	}

	if h.Logger != nil {
		h.Logger.Infof("Rollback: Checkpoint created '%s' (commit: %s)",
			checkpoint.BranchName, checkpoint.CommitHash)
	}

	return nil
}

// PostTask evaluates rollback decision and performs rollback if needed.
// Retrieves CheckpointInfo from task.Metadata["rollback_checkpoint"].
// On success, cleans up the checkpoint branch.
// Errors log warning but do not block execution (graceful degradation).
//
// Parameters:
//   - ctx: Context for cancellation
//   - taskNumber: Task number for logging
//   - metadata: Task metadata containing rollback_checkpoint
//   - verdict: QC verdict string (GREEN, YELLOW, RED)
//   - attempt: Current attempt number (1-indexed)
//   - maxRetries: Maximum retry attempts allowed
//   - success: Whether the task completed successfully
func (h *RollbackHook) PostTask(ctx context.Context, taskNumber int, metadata map[string]interface{},
	verdict string, attempt int, maxRetries int, success bool) error {
	if h == nil || h.Manager == nil {
		return nil // Graceful degradation
	}

	// Skip if rollback is not enabled
	if !h.Manager.Enabled() {
		return nil
	}

	// Retrieve checkpoint from metadata
	var checkpoint *CheckpointInfo
	if metadata != nil {
		if cp, ok := metadata["rollback_checkpoint"].(*CheckpointInfo); ok {
			checkpoint = cp
		}
	}

	// No checkpoint stored - nothing to do
	if checkpoint == nil {
		if h.Logger != nil {
			h.Logger.Warnf("Rollback: No checkpoint found for task %d - skipping rollback evaluation", taskNumber)
		}
		return nil
	}

	// Evaluate rollback decision
	shouldRollback := h.Manager.ShouldRollback(verdict, attempt, maxRetries)

	if shouldRollback {
		if h.Logger != nil {
			h.Logger.Infof("Rollback: Task %d triggered rollback (verdict=%s, attempt=%d, maxRetries=%d)",
				taskNumber, verdict, attempt, maxRetries)
		}

		// Perform rollback
		if err := h.Manager.PerformRollback(ctx, checkpoint); err != nil {
			if h.Logger != nil {
				h.Logger.Warnf("Rollback: Failed to rollback task %d: %v", taskNumber, err)
			}
			// Don't return error - graceful degradation
			return nil
		}

		// Delete checkpoint branch after successful rollback (cleanup)
		if err := h.deleteCheckpoint(ctx, checkpoint); err != nil {
			if h.Logger != nil {
				h.Logger.Warnf("Rollback: Failed to delete checkpoint branch after rollback: %v", err)
			}
		}

		return nil
	}

	// Task succeeded (no rollback needed) - cleanup checkpoint branch
	if success {
		if err := h.deleteCheckpoint(ctx, checkpoint); err != nil {
			if h.Logger != nil {
				h.Logger.Warnf("Rollback: Failed to cleanup checkpoint for task %d: %v", taskNumber, err)
			}
		} else if h.Logger != nil {
			h.Logger.Infof("Rollback: Cleaned up checkpoint '%s' after successful task", checkpoint.BranchName)
		}
	}

	return nil
}

// deleteCheckpoint deletes a checkpoint branch.
func (h *RollbackHook) deleteCheckpoint(ctx context.Context, checkpoint *CheckpointInfo) error {
	if h.Checkpointer == nil || checkpoint == nil {
		return fmt.Errorf("checkpointer or checkpoint is nil")
	}
	return h.Checkpointer.DeleteCheckpoint(ctx, checkpoint.BranchName)
}

// Enabled returns whether the rollback hook is enabled and active.
func (h *RollbackHook) Enabled() bool {
	if h == nil || h.Manager == nil {
		return false
	}
	return h.Manager.Enabled()
}
