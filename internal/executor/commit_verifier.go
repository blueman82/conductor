package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// CommitVerification holds the result of verifying a commit was created.
type CommitVerification struct {
	// Found indicates whether a matching commit was found in the git log.
	Found bool

	// CommitHash is the hash of the matching commit (short form), empty if not found.
	CommitHash string

	// FullHash is the full commit hash, empty if not found.
	FullHash string

	// Message is the actual commit message found.
	Message string

	// Mismatch describes why verification failed (empty if Found is true).
	// Examples: "no matching commit found", "commit message prefix mismatch", etc.
	Mismatch string

	// Duration is how long the verification took.
	Duration time.Duration
}

// CommitVerifier verifies that an expected commit was created by an agent.
// This is for VERIFICATION only - agents commit via git commands in their prompts.
// Conductor only verifies the commit exists after agent completion.
type CommitVerifier interface {
	// Verify checks if a commit matching the spec exists in recent git history.
	// Uses read-only git commands (safe, idempotent).
	// Returns CommitVerification with Found=true if a matching commit exists.
	Verify(ctx context.Context, spec *models.CommitSpec, workDir string) (*CommitVerification, error)
}

// GitLogVerifier implements CommitVerifier using git log --grep.
type GitLogVerifier struct {
	// CommandRunner for executing git commands (optional, uses exec.Command if nil)
	CommandRunner CommandRunner

	// LookbackCommits is how many commits to search (default 10)
	LookbackCommits int
}

// NewGitLogVerifier creates a GitLogVerifier with default settings.
func NewGitLogVerifier() *GitLogVerifier {
	return &GitLogVerifier{
		LookbackCommits: 10,
	}
}

// NewGitLogVerifierWithRunner creates a GitLogVerifier with a custom command runner.
// Useful for testing.
func NewGitLogVerifierWithRunner(runner CommandRunner) *GitLogVerifier {
	return &GitLogVerifier{
		CommandRunner:   runner,
		LookbackCommits: 10,
	}
}

// Verify checks if a commit matching the spec exists in recent git history.
// Uses git log --oneline --grep to search for commits with matching messages.
func (v *GitLogVerifier) Verify(ctx context.Context, spec *models.CommitSpec, workDir string) (*CommitVerification, error) {
	start := time.Now()

	// Validate spec
	if spec == nil {
		return &CommitVerification{
			Found:    false,
			Mismatch: "commit spec is nil",
			Duration: time.Since(start),
		}, nil
	}

	if spec.IsEmpty() {
		return &CommitVerification{
			Found:    false,
			Mismatch: "commit spec is empty",
			Duration: time.Since(start),
		}, nil
	}

	// Build expected commit message for search
	expectedMessage := spec.BuildCommitMessage()
	if expectedMessage == "" {
		return &CommitVerification{
			Found:    false,
			Mismatch: "expected commit message is empty",
			Duration: time.Since(start),
		}, nil
	}

	// Prepare lookback limit
	lookback := v.LookbackCommits
	if lookback <= 0 {
		lookback = 10
	}

	// Execute git log --grep
	var output string
	var err error

	if v.CommandRunner != nil {
		// Use injected runner (for testing)
		cmd := fmt.Sprintf("git log --oneline --grep=%q -n %d", expectedMessage, lookback)
		output, err = v.CommandRunner.Run(ctx, cmd)
	} else {
		// Use exec.Command directly
		cmd := exec.CommandContext(ctx, "git", "log", "--oneline",
			fmt.Sprintf("--grep=%s", expectedMessage),
			"-n", fmt.Sprintf("%d", lookback))
		if workDir != "" {
			cmd.Dir = workDir
		}
		outputBytes, execErr := cmd.CombinedOutput()
		output = string(outputBytes)
		err = execErr
	}

	duration := time.Since(start)

	if err != nil {
		// Git error (not in repo, etc.)
		return &CommitVerification{
			Found:    false,
			Mismatch: fmt.Sprintf("git log failed: %v", err),
			Duration: duration,
		}, nil
	}

	// Parse git log output
	// Format: <short-hash> <message>
	output = strings.TrimSpace(output)
	if output == "" {
		return &CommitVerification{
			Found:    false,
			Mismatch: fmt.Sprintf("no commit found matching %q", expectedMessage),
			Duration: duration,
		}, nil
	}

	// Parse first matching line
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return &CommitVerification{
			Found:    false,
			Mismatch: fmt.Sprintf("no commit found matching %q", expectedMessage),
			Duration: duration,
		}, nil
	}

	firstLine := strings.TrimSpace(lines[0])
	parts := strings.SplitN(firstLine, " ", 2)
	if len(parts) < 2 {
		return &CommitVerification{
			Found:    false,
			Mismatch: fmt.Sprintf("unexpected git log output format: %q", firstLine),
			Duration: duration,
		}, nil
	}

	shortHash := parts[0]
	actualMessage := parts[1]

	// Get full hash for the commit
	fullHash := v.getFullHash(ctx, shortHash, workDir)

	return &CommitVerification{
		Found:      true,
		CommitHash: shortHash,
		FullHash:   fullHash,
		Message:    actualMessage,
		Duration:   duration,
	}, nil
}

// getFullHash retrieves the full commit hash for a short hash.
func (v *GitLogVerifier) getFullHash(ctx context.Context, shortHash, workDir string) string {
	if v.CommandRunner != nil {
		cmd := fmt.Sprintf("git rev-parse %s", shortHash)
		output, err := v.CommandRunner.Run(ctx, cmd)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(output)
	}

	cmd := exec.CommandContext(ctx, "git", "rev-parse", shortHash)
	if workDir != "" {
		cmd.Dir = workDir
	}
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// VerifyExact checks for an exact commit message match (stricter than grep).
// Uses grep to find candidates, then verifies exact match.
func (v *GitLogVerifier) VerifyExact(ctx context.Context, spec *models.CommitSpec, workDir string) (*CommitVerification, error) {
	// First do a grep-based search
	result, err := v.Verify(ctx, spec, workDir)
	if err != nil {
		return result, err
	}

	if !result.Found {
		return result, nil
	}

	// Now verify exact match
	expectedMessage := spec.BuildCommitMessage()
	if result.Message != expectedMessage {
		return &CommitVerification{
			Found:      false,
			CommitHash: result.CommitHash,
			Message:    result.Message,
			Mismatch:   fmt.Sprintf("commit message mismatch: expected %q, got %q", expectedMessage, result.Message),
			Duration:   result.Duration,
		}, nil
	}

	return result, nil
}
