package executor

import (
	"context"
	"fmt"

	"github.com/harrison/conductor/internal/config"
)

// RollbackManager encapsulates task-level rollback decision logic.
// It evaluates whether to rollback based on config mode, QC verdict,
// and retry attempt numbers.
//
// Decision matrix:
// | Mode                 | Verdict | Attempt >= Max | Rollback? |
// |----------------------|---------|----------------|-----------|
// | manual               | any     | any            | NO        |
// | auto_on_red          | RED     | any            | YES       |
// | auto_on_red          | GREEN   | any            | NO        |
// | auto_on_max_retries  | RED     | NO             | NO        |
// | auto_on_max_retries  | RED     | YES            | YES       |
// | auto_on_max_retries  | GREEN   | any            | NO        |
type RollbackManager struct {
	// Config contains rollback mode and settings.
	Config *config.RollbackConfig

	// Checkpointer provides git operations for rollback.
	Checkpointer GitCheckpointer

	// Logger for console output.
	Logger RuntimeEnforcementLogger
}

// NewRollbackManager creates a new RollbackManager.
// Returns nil if config or checkpointer is nil (graceful degradation).
func NewRollbackManager(cfg *config.RollbackConfig, checkpointer GitCheckpointer, logger RuntimeEnforcementLogger) *RollbackManager {
	if cfg == nil || checkpointer == nil {
		return nil
	}
	return &RollbackManager{
		Config:       cfg,
		Checkpointer: checkpointer,
		Logger:       logger,
	}
}

// ShouldRollback evaluates whether to perform a rollback based on the decision matrix.
//
// Parameters:
//   - verdict: QC verdict string (GREEN, YELLOW, RED)
//   - attempt: Current attempt number (1-indexed)
//   - maxRetries: Maximum retry attempts allowed (from QualityControlConfig.RetryOnRed)
//
// Returns true if rollback should be performed based on config mode and conditions.
func (m *RollbackManager) ShouldRollback(verdict string, attempt int, maxRetries int) bool {
	if m == nil || m.Config == nil {
		return false
	}

	// Check if rollback feature is enabled
	if !m.Config.Enabled {
		return false
	}

	switch m.Config.Mode {
	case config.RollbackModeManual:
		// Manual mode never auto-rollbacks
		return false

	case config.RollbackModeAutoOnRed:
		// Auto-rollback on any RED verdict
		return verdict == "RED"

	case config.RollbackModeAutoOnMaxRetries:
		// Auto-rollback only when RED and max retries exhausted
		// Max retries exhausted means: attempt >= maxRetries + 1
		// Because attempt 1 is the initial try, retries are 2 through maxRetries+1
		if verdict != "RED" {
			return false
		}
		// attempt is the current attempt (1-indexed)
		// maxRetries is the number of retries allowed after initial failure
		// So total attempts allowed = 1 (initial) + maxRetries
		return attempt > maxRetries

	default:
		// Unknown mode, default to no rollback
		return false
	}
}

// PerformRollback restores the working directory to the checkpoint state.
// Logs the rollback action and commit hash being restored to.
//
// Parameters:
//   - ctx: Context for cancellation
//   - checkpoint: CheckpointInfo containing the commit hash to restore
//
// Returns error if rollback fails.
func (m *RollbackManager) PerformRollback(ctx context.Context, checkpoint *CheckpointInfo) error {
	if m == nil || m.Checkpointer == nil {
		return fmt.Errorf("rollback manager not initialized")
	}

	if checkpoint == nil {
		return fmt.Errorf("checkpoint cannot be nil")
	}

	if checkpoint.CommitHash == "" {
		return fmt.Errorf("checkpoint commit hash cannot be empty")
	}

	// Log the rollback action
	if m.Logger != nil {
		m.Logger.Infof("Rollback: Restoring to checkpoint %s (commit: %s)",
			checkpoint.BranchName, checkpoint.CommitHash)
	}

	// Perform the actual rollback via GitCheckpointer
	if err := m.Checkpointer.RestoreCheckpoint(ctx, checkpoint.CommitHash); err != nil {
		if m.Logger != nil {
			m.Logger.Warnf("Rollback: Failed to restore checkpoint: %v", err)
		}
		return fmt.Errorf("failed to restore checkpoint: %w", err)
	}

	if m.Logger != nil {
		m.Logger.Infof("Rollback: Successfully restored to commit %s", checkpoint.CommitHash)
	}

	return nil
}

// Enabled returns whether rollback functionality is enabled.
func (m *RollbackManager) Enabled() bool {
	if m == nil || m.Config == nil {
		return false
	}
	return m.Config.Enabled
}

// Mode returns the current rollback mode.
func (m *RollbackManager) Mode() config.RollbackMode {
	if m == nil || m.Config == nil {
		return config.RollbackModeManual
	}
	return m.Config.Mode
}
