package executor

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/harrison/conductor/internal/models"
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
//
// Parameters:
//   - ctx: Context for cancellation
//   - task: Task being executed (must not be nil, Metadata will be initialized if nil)
func (h *RollbackHook) PreTask(ctx context.Context, task *models.Task) error {
	if h == nil || h.Checkpointer == nil || task == nil {
		return nil // Graceful degradation
	}

	// Skip if rollback is not enabled
	if h.Manager == nil || !h.Manager.Enabled() {
		return nil
	}

	// Parse task number for checkpoint naming
	taskNumber := parseTaskNumberForRollback(task.Number)

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

	// Initialize metadata if nil and store checkpoint info for PostTask retrieval
	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}
	task.Metadata["rollback_checkpoint"] = checkpoint

	if h.Logger != nil {
		h.Logger.Infof("Rollback: Checkpoint created '%s' (commit: %s)",
			checkpoint.BranchName, checkpoint.CommitHash)
	}

	return nil
}

// parseTaskNumberForRollback extracts task number from string (e.g., "1", "1.1", "task-1").
func parseTaskNumberForRollback(taskNumberStr string) int {
	// Try direct integer parse first
	if n, err := strconv.Atoi(taskNumberStr); err == nil {
		return n
	}

	// Handle formats like "1.1", "1.2" - take the integer part
	if idx := strings.Index(taskNumberStr, "."); idx > 0 {
		if n, err := strconv.Atoi(taskNumberStr[:idx]); err == nil {
			return n
		}
	}

	// Handle formats like "task-1" - extract trailing number
	for i := len(taskNumberStr) - 1; i >= 0; i-- {
		if taskNumberStr[i] >= '0' && taskNumberStr[i] <= '9' {
			end := i + 1
			start := i
			for start > 0 && taskNumberStr[start-1] >= '0' && taskNumberStr[start-1] <= '9' {
				start--
			}
			if n, err := strconv.Atoi(taskNumberStr[start:end]); err == nil {
				return n
			}
		}
	}

	// Fallback: use 0 for unknown formats
	return 0
}

// PostTask evaluates rollback decision and performs rollback if needed.
// Retrieves CheckpointInfo from task.Metadata["rollback_checkpoint"].
// On success, cleans up the checkpoint branch.
// Errors log warning but do not block execution (graceful degradation).
//
// Parameters:
//   - ctx: Context for cancellation
//   - task: Task that was executed (must not be nil)
//   - verdict: QC verdict string (GREEN, YELLOW, RED)
//   - attempt: Current attempt number (1-indexed)
//   - maxRetries: Maximum retry attempts allowed
//   - success: Whether the task completed successfully
func (h *RollbackHook) PostTask(ctx context.Context, task *models.Task,
	verdict string, attempt int, maxRetries int, success bool) error {
	if h == nil || h.Manager == nil || task == nil {
		return nil // Graceful degradation
	}

	// Skip if rollback is not enabled
	if !h.Manager.Enabled() {
		return nil
	}

	// Retrieve checkpoint from task metadata
	var checkpoint *CheckpointInfo
	if task.Metadata != nil {
		if cp, ok := task.Metadata["rollback_checkpoint"].(*CheckpointInfo); ok {
			checkpoint = cp
		}
	}

	// No checkpoint stored - nothing to do
	if checkpoint == nil {
		if h.Logger != nil {
			h.Logger.Warnf("Rollback: No checkpoint found for task %s - skipping rollback evaluation", task.Number)
		}
		return nil
	}

	// Evaluate rollback decision
	shouldRollback := h.Manager.ShouldRollback(verdict, attempt, maxRetries)

	if shouldRollback {
		if h.Logger != nil {
			h.Logger.Infof("Rollback: Task %s triggered rollback (verdict=%s, attempt=%d, maxRetries=%d)",
				task.Number, verdict, attempt, maxRetries)
		}

		// Perform rollback
		if err := h.Manager.PerformRollback(ctx, checkpoint); err != nil {
			if h.Logger != nil {
				h.Logger.Warnf("Rollback: Failed to rollback task %s: %v", task.Number, err)
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
				h.Logger.Warnf("Rollback: Failed to cleanup checkpoint for task %s: %v", task.Number, err)
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
