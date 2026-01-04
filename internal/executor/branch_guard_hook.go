package executor

import (
	"context"
)

// BranchGuardHook wraps BranchGuard for orchestrator-level branch protection.
// This is a thin adapter layer that:
// - Calls BranchGuard.Guard() at the start of orchestrator execution
// - Runs BEFORE SetupHook to ensure branch safety before any other operations
// - Handles graceful degradation if guard is unavailable
//
// The hook provides plan-level branch protection by:
// - Checking for dirty state and blocking with actionable error
// - Creating checkpoint branches for recovery
// - Switching to working branches when on protected branches (main/master/develop)
type BranchGuardHook struct {
	guard  *BranchGuard
	logger RuntimeEnforcementLogger
}

// NewBranchGuardHook creates a new BranchGuardHook.
// Returns nil if guard is nil (graceful degradation pattern consistent with other hooks).
func NewBranchGuardHook(guard *BranchGuard, logger RuntimeEnforcementLogger) *BranchGuardHook {
	if guard == nil {
		return nil
	}
	return &BranchGuardHook{
		guard:  guard,
		logger: logger,
	}
}

// Guard performs branch safety checks and creates checkpoints before plan execution.
// Called at the very start of orchestrator Execute(), BEFORE setupHook.
// Returns BranchGuardResult for orchestrator logging and potential recovery.
// Errors are returned to block execution (e.g., dirty state with require_clean_state: true).
func (h *BranchGuardHook) Guard(ctx context.Context) (*BranchGuardResult, error) {
	if h == nil || h.guard == nil {
		return nil, nil // Graceful degradation
	}

	if h.logger != nil {
		h.logger.Infof("Branch Guard: Checking branch safety...")
	}

	// Delegate to BranchGuard.Guard() which handles:
	// 1. Dirty state check (if require_clean_state is true)
	// 2. Checkpoint creation for all branches
	// 3. Working branch creation for protected branches
	result, err := h.guard.Guard(ctx)
	if err != nil {
		// Return error to block execution (e.g., dirty state)
		return nil, err
	}

	// Log result summary
	if result != nil && h.logger != nil {
		if result.WasProtected {
			h.logger.Infof("Branch Guard: Protected branch detected, created working branch '%s'",
				result.WorkingBranch)
		} else {
			h.logger.Infof("Branch Guard: Checkpoint created '%s' on branch '%s'",
				result.CheckpointBranch, result.OriginalBranch)
		}
	}

	return result, nil
}

// Note: The BranchGuardHook follows the established hook patterns:
// - Nil-safety: NewBranchGuardHook returns nil if guard is nil
// - Graceful degradation: Nil hook returns nil result (no-op)
// - Timing: Runs BEFORE SetupHook in orchestrator.Execute()
// - Consistent interface: Uses RuntimeEnforcementLogger like other hooks
