package executor

import (
	"context"
	"fmt"
	"time"
)

// SetupHook wraps SetupIntrospector to provide pre-wave project setup.
// This is a thin adapter layer that:
// - Introspects the project to determine required setup commands
// - Runs setup commands before wave execution begins
// - Handles graceful degradation if introspector is unavailable
type SetupHook struct {
	introspector *SetupIntrospector
	logger       RuntimeEnforcementLogger
}

// NewSetupHook creates a new SetupHook.
// Returns nil if introspector is nil (graceful degradation pattern consistent with other hooks).
func NewSetupHook(introspector *SetupIntrospector, logger RuntimeEnforcementLogger) *SetupHook {
	if introspector == nil {
		return nil
	}
	return &SetupHook{
		introspector: introspector,
		logger:       logger,
	}
}

// Setup introspects the project and runs any required setup commands.
// Called before wave execution begins in the orchestrator.
// Errors log warning but do not block execution (graceful degradation).
func (h *SetupHook) Setup(ctx context.Context) error {
	if h == nil || h.introspector == nil {
		return nil // Graceful degradation
	}

	startTime := time.Now()

	if h.logger != nil {
		h.logger.Infof("Setup: Starting project introspection...")
	}

	// Run introspection to determine setup commands
	result, err := h.introspector.Introspect(ctx)
	if err != nil {
		if h.logger != nil {
			h.logger.Warnf("Setup: Introspection failed (continuing without setup): %v", err)
		}
		return nil // Graceful degradation - don't fail the plan on setup error
	}

	introspectDuration := time.Since(startTime)

	if result == nil || len(result.Commands) == 0 {
		if h.logger != nil {
			h.logger.Infof("Setup: No setup commands needed (introspection took %v)", introspectDuration)
		}
		return nil
	}

	if h.logger != nil {
		h.logger.Infof("Setup: Introspection found %d commands (took %v): %s",
			len(result.Commands), introspectDuration, result.Reasoning)
	}

	// Run the setup commands
	commandStartTime := time.Now()
	err = h.introspector.RunSetupCommands(ctx, result)
	commandDuration := time.Since(commandStartTime)

	if err != nil {
		if h.logger != nil {
			h.logger.Warnf("Setup: Commands failed after %v (continuing): %v", commandDuration, err)
		}
		// Note: We return nil here for graceful degradation, but a required command failure
		// is already handled inside RunSetupCommands which returns an error for required failures.
		// The decision to block or continue is made there based on Required field.
		return nil // Graceful degradation - log but don't block plan execution
	}

	totalDuration := time.Since(startTime)
	if h.logger != nil {
		h.logger.Infof("Setup: Completed %d commands successfully (total: %v)",
			len(result.Commands), totalDuration)
	}

	return nil
}

// RuntimeEnforcementLogger is already defined in other executor files.
// This interface provides Infof and Warnf methods for logging.
// See: pattern_integration.go, warmup_hook.go

// Note: The SetupHook follows the established hook patterns:
// - Nil-safety: NewSetupHook returns nil if introspector is nil
// - Graceful degradation: Errors log warnings but don't block execution
// - Timing logs: Reports introspection and command execution durations
// - Consistent interface: Uses RuntimeEnforcementLogger like other hooks

// logSetupCommandOutput formats a setup command result for logging.
func logSetupCommandOutput(cmdIndex, total int, cmd SetupCommand, success bool) string {
	status := "✓"
	if !success {
		status = "✗"
	}
	return fmt.Sprintf("  %s [%d/%d] %s: %s", status, cmdIndex+1, total, cmd.Purpose, cmd.Command)
}
