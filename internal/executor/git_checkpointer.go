package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/config"
)

// CheckpointInfo holds information about a created checkpoint.
type CheckpointInfo struct {
	// BranchName is the name of the checkpoint branch.
	BranchName string

	// CommitHash is the commit hash at the time of checkpoint creation.
	CommitHash string

	// CreatedAt is the timestamp when the checkpoint was created.
	CreatedAt time.Time
}

// GitCheckpointer provides git branch operations for checkpoint/rollback functionality.
// Both BranchGuard (plan-level) and RollbackHook (task-level) use this interface.
type GitCheckpointer interface {
	// CreateCheckpoint creates a checkpoint branch for a task.
	// Returns CheckpointInfo with branch name, commit hash, and creation time.
	CreateCheckpoint(ctx context.Context, taskNumber int) (*CheckpointInfo, error)

	// RestoreCheckpoint restores the working directory to a specific commit.
	// Uses git reset --hard to restore the state.
	RestoreCheckpoint(ctx context.Context, commitHash string) error

	// DeleteCheckpoint deletes a checkpoint branch.
	// Uses git branch -D to force delete the branch.
	DeleteCheckpoint(ctx context.Context, branchName string) error

	// CreateBranch creates a new branch and switches to it.
	// Uses git checkout -b to create and switch.
	CreateBranch(ctx context.Context, branchName string) error

	// SwitchBranch switches to an existing branch.
	// Uses git checkout to switch branches.
	SwitchBranch(ctx context.Context, branchName string) error

	// GetCurrentBranch returns the name of the current branch.
	// Uses git branch --show-current.
	GetCurrentBranch(ctx context.Context) (string, error)

	// IsCleanState checks if the working directory has no uncommitted changes.
	// Returns true if git status --porcelain returns empty output.
	IsCleanState(ctx context.Context) (bool, error)
}

// DefaultGitCheckpointer implements GitCheckpointer using git commands.
type DefaultGitCheckpointer struct {
	// CommandRunner for executing git commands (optional, uses exec.Command if nil)
	CommandRunner CommandRunner

	// Config contains rollback configuration
	Config *config.RollbackConfig

	// WorkDir is the working directory for git commands (empty = current dir)
	WorkDir string
}

// NewGitCheckpointer creates a DefaultGitCheckpointer with default settings.
func NewGitCheckpointer(cfg *config.RollbackConfig) *DefaultGitCheckpointer {
	return &DefaultGitCheckpointer{
		Config: cfg,
	}
}

// NewGitCheckpointerWithRunner creates a DefaultGitCheckpointer with a custom command runner.
// Useful for testing.
func NewGitCheckpointerWithRunner(runner CommandRunner, cfg *config.RollbackConfig) *DefaultGitCheckpointer {
	return &DefaultGitCheckpointer{
		CommandRunner: runner,
		Config:        cfg,
	}
}

// NewGitCheckpointerWithWorkDir creates a DefaultGitCheckpointer with a specified working directory.
func NewGitCheckpointerWithWorkDir(cfg *config.RollbackConfig, workDir string) *DefaultGitCheckpointer {
	return &DefaultGitCheckpointer{
		Config:  cfg,
		WorkDir: workDir,
	}
}

// CreateCheckpoint creates a checkpoint branch for a task.
func (g *DefaultGitCheckpointer) CreateCheckpoint(ctx context.Context, taskNumber int) (*CheckpointInfo, error) {
	// Get current commit hash
	commitHash, err := g.runCommand(ctx, "git", "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get current commit hash: %w", err)
	}
	commitHash = strings.TrimSpace(commitHash)

	// Generate checkpoint branch name
	timestamp := time.Now().Format("20060102-150405")
	prefix := g.getCheckpointPrefix()
	branchName := fmt.Sprintf("%stask-%d-%s", prefix, taskNumber, timestamp)

	// Create checkpoint branch (without switching to it)
	_, err = g.runCommand(ctx, "git", "branch", branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkpoint branch %s: %w", branchName, err)
	}

	return &CheckpointInfo{
		BranchName: branchName,
		CommitHash: commitHash,
		CreatedAt:  time.Now(),
	}, nil
}

// RestoreCheckpoint restores the working directory to a specific commit.
func (g *DefaultGitCheckpointer) RestoreCheckpoint(ctx context.Context, commitHash string) error {
	// Validate commit hash is not empty
	if commitHash == "" {
		return fmt.Errorf("commit hash cannot be empty")
	}

	// Use git reset --hard to restore to the checkpoint
	_, err := g.runCommand(ctx, "git", "reset", "--hard", commitHash)
	if err != nil {
		return fmt.Errorf("failed to restore checkpoint %s: %w", commitHash, err)
	}

	return nil
}

// DeleteCheckpoint deletes a checkpoint branch.
func (g *DefaultGitCheckpointer) DeleteCheckpoint(ctx context.Context, branchName string) error {
	// Validate branch name is not empty
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Use git branch -D to force delete the branch
	_, err := g.runCommand(ctx, "git", "branch", "-D", branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}

	return nil
}

// CreateBranch creates a new branch and switches to it.
func (g *DefaultGitCheckpointer) CreateBranch(ctx context.Context, branchName string) error {
	// Validate branch name is not empty
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Use git checkout -b to create and switch to the new branch
	_, err := g.runCommand(ctx, "git", "checkout", "-b", branchName)
	if err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	return nil
}

// SwitchBranch switches to an existing branch.
func (g *DefaultGitCheckpointer) SwitchBranch(ctx context.Context, branchName string) error {
	// Validate branch name is not empty
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Use git checkout to switch to the branch
	_, err := g.runCommand(ctx, "git", "checkout", branchName)
	if err != nil {
		return fmt.Errorf("failed to switch to branch %s: %w", branchName, err)
	}

	return nil
}

// GetCurrentBranch returns the name of the current branch.
func (g *DefaultGitCheckpointer) GetCurrentBranch(ctx context.Context) (string, error) {
	output, err := g.runCommand(ctx, "git", "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// IsCleanState checks if the working directory has no uncommitted changes.
func (g *DefaultGitCheckpointer) IsCleanState(ctx context.Context) (bool, error) {
	output, err := g.runCommand(ctx, "git", "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	// If output is empty, the working directory is clean
	return strings.TrimSpace(output) == "", nil
}

// runCommand executes a git command and returns the output.
func (g *DefaultGitCheckpointer) runCommand(ctx context.Context, name string, args ...string) (string, error) {
	if g.CommandRunner != nil {
		// Use injected runner (for testing)
		// Build full command string for the runner
		cmd := name
		for _, arg := range args {
			cmd += " " + arg
		}
		return g.CommandRunner.Run(ctx, cmd)
	}

	// Use exec.Command directly
	cmd := exec.CommandContext(ctx, name, args...)
	if g.WorkDir != "" {
		cmd.Dir = g.WorkDir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%w: %s", err, string(output))
	}
	return string(output), nil
}

// getCheckpointPrefix returns the checkpoint prefix from config or default.
func (g *DefaultGitCheckpointer) getCheckpointPrefix() string {
	if g.Config != nil && g.Config.CheckpointPrefix != "" {
		return g.Config.CheckpointPrefix
	}
	return "conductor-checkpoint-"
}

// Ensure DefaultGitCheckpointer implements GitCheckpointer
var _ GitCheckpointer = (*DefaultGitCheckpointer)(nil)
