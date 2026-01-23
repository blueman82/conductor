package executor

import (
	"context"
	"os/exec"
	"strconv"
	"strings"

	"github.com/harrison/conductor/internal/models"
)

// LOCMetrics holds lines of code change data.
type LOCMetrics struct {
	LinesAdded   int
	LinesDeleted int
	FileCount    int
}

// LOCTrackerHook captures LOC changes during task execution.
// This hook follows the same graceful degradation pattern as RollbackHook,
// allowing execution to continue even if git operations fail.
type LOCTrackerHook struct {
	Enabled bool
	WorkDir string
	Logger  RuntimeEnforcementLogger
}

// NewLOCTrackerHook creates a new LOC tracker hook.
// Returns nil if not enabled (graceful disable pattern consistent with other hooks).
func NewLOCTrackerHook(enabled bool, workDir string, logger RuntimeEnforcementLogger) *LOCTrackerHook {
	if !enabled {
		return nil
	}
	return &LOCTrackerHook{
		Enabled: enabled,
		WorkDir: workDir,
		Logger:  logger,
	}
}

// PreTask captures the baseline commit hash before task execution.
// Stores the commit hash in task.Metadata["loc_baseline_commit"] for PostTask retrieval.
// Errors log warning but do not block execution (graceful degradation).
func (h *LOCTrackerHook) PreTask(ctx context.Context, task *models.Task) error {
	if h == nil || !h.Enabled {
		return nil
	}

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	if h.WorkDir != "" {
		cmd.Dir = h.WorkDir
	}
	output, err := cmd.Output()
	if err != nil {
		GracefulWarn(h.Logger, "LOC: Failed to capture baseline commit: %v", err)
		return nil // Graceful degradation
	}

	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}
	task.Metadata["loc_baseline_commit"] = strings.TrimSpace(string(output))
	return nil
}

// PostTask calculates LOC changes from baseline and updates task fields.
// Uses git diff --numstat to calculate lines added/deleted.
// Errors log warning but do not block execution (graceful degradation).
func (h *LOCTrackerHook) PostTask(ctx context.Context, task *models.Task) (*LOCMetrics, error) {
	if h == nil || !h.Enabled || task == nil {
		return nil, nil
	}

	var baselineCommit string
	if task.Metadata != nil {
		if bc, ok := task.Metadata["loc_baseline_commit"].(string); ok {
			baselineCommit = bc
		}
	}

	if baselineCommit == "" {
		return nil, nil
	}

	cmd := exec.CommandContext(ctx, "git", "diff", "--numstat", baselineCommit+"..HEAD")
	if h.WorkDir != "" {
		cmd.Dir = h.WorkDir
	}
	output, err := cmd.Output()
	if err != nil {
		GracefulWarn(h.Logger, "LOC: Failed to calculate diff: %v", err)
		return nil, nil
	}

	metrics := h.parseNumstat(string(output))

	task.LinesAdded = metrics.LinesAdded
	task.LinesDeleted = metrics.LinesDeleted

	GracefulInfo(h.Logger, "LOC: Task %s changed +%d/-%d lines across %d files",
		task.Number, metrics.LinesAdded, metrics.LinesDeleted, metrics.FileCount)

	return metrics, nil
}

// parseNumstat parses git diff --numstat output into LOCMetrics.
// Format: <added>\t<deleted>\t<filename>
// Binary files show "-" for added/deleted counts.
func (h *LOCTrackerHook) parseNumstat(output string) *LOCMetrics {
	metrics := &LOCMetrics{}
	output = strings.TrimSpace(output)
	if output == "" {
		return metrics
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		// Skip binary files (shown as "-")
		if parts[0] != "-" {
			if added, err := strconv.Atoi(parts[0]); err == nil {
				metrics.LinesAdded += added
			}
		}
		if parts[1] != "-" {
			if deleted, err := strconv.Atoi(parts[1]); err == nil {
				metrics.LinesDeleted += deleted
			}
		}
		metrics.FileCount++
	}

	return metrics
}
