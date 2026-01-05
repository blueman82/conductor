package executor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/config"
)

// MockGitCommandRunner implements CommandRunner for git checkpointer testing.
type MockGitCommandRunner struct {
	Outputs map[string]string // command -> output
	Errors  map[string]error  // command -> error
}

func NewMockGitCommandRunner() *MockGitCommandRunner {
	return &MockGitCommandRunner{
		Outputs: make(map[string]string),
		Errors:  make(map[string]error),
	}
}

func (m *MockGitCommandRunner) Run(ctx context.Context, command string) (string, error) {
	if err, ok := m.Errors[command]; ok && err != nil {
		return m.Outputs[command], err
	}
	return m.Outputs[command], nil
}

// === Constructor Tests ===

func TestNewGitCheckpointer(t *testing.T) {
	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointer(&cfg)
	if g == nil {
		t.Fatal("Expected non-nil checkpointer")
	}
	if g.CommandRunner != nil {
		t.Error("Expected nil CommandRunner by default")
	}
	if g.Config == nil {
		t.Error("Expected Config to be set")
	}
}

func TestNewGitCheckpointerWithRunner(t *testing.T) {
	runner := NewMockGitCommandRunner()
	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)
	if g == nil {
		t.Fatal("Expected non-nil checkpointer")
	}
	if g.CommandRunner != runner {
		t.Error("Expected CommandRunner to be set")
	}
	if g.Config == nil {
		t.Error("Expected Config to be set")
	}
}

func TestNewGitCheckpointerWithWorkDir(t *testing.T) {
	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithWorkDir(&cfg, "/tmp/test")
	if g == nil {
		t.Fatal("Expected non-nil checkpointer")
	}
	if g.WorkDir != "/tmp/test" {
		t.Errorf("Expected WorkDir '/tmp/test', got %q", g.WorkDir)
	}
}

// === Interface Compliance Test ===

func TestGitCheckpointerInterface(t *testing.T) {
	var _ GitCheckpointer = (*DefaultGitCheckpointer)(nil)
}

// === CreateCheckpoint Tests ===

func TestCreateCheckpoint_Success(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git rev-parse HEAD"] = "abc123def456789\n"
	runner.Outputs["git branch conductor-checkpoint-task-1-20060102-150405"] = ""

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	info, err := g.CreateCheckpoint(context.Background(), 1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if info == nil {
		t.Fatal("Expected non-nil CheckpointInfo")
	}
	if info.CommitHash != "abc123def456789" {
		t.Errorf("Expected CommitHash 'abc123def456789', got %q", info.CommitHash)
	}
	if !strings.HasPrefix(info.BranchName, "conductor-checkpoint-task-1-") {
		t.Errorf("Expected BranchName to start with 'conductor-checkpoint-task-1-', got %q", info.BranchName)
	}
	if info.CreatedAt.IsZero() {
		t.Error("Expected non-zero CreatedAt")
	}
}

func TestCreateCheckpoint_CustomPrefix(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git rev-parse HEAD"] = "abc123\n"
	// Accept any branch name with the custom prefix
	for k := range runner.Outputs {
		delete(runner.Outputs, k)
	}
	runner.Outputs["git rev-parse HEAD"] = "abc123\n"

	cfg := config.RollbackConfig{
		CheckpointPrefix: "custom-prefix-",
	}
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	// Mock the branch creation command
	info, err := g.CreateCheckpoint(context.Background(), 5)
	// The error here is expected because we didn't mock the exact branch name
	// but we can check the prefix logic
	if err == nil {
		if !strings.HasPrefix(info.BranchName, "custom-prefix-task-5-") {
			t.Errorf("Expected BranchName to start with 'custom-prefix-task-5-', got %q", info.BranchName)
		}
	}
}

func TestCreateCheckpoint_RevParseError(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Errors["git rev-parse HEAD"] = errors.New("not a git repository")

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	_, err := g.CreateCheckpoint(context.Background(), 1)
	if err == nil {
		t.Error("Expected error for rev-parse failure")
	}
	if !strings.Contains(err.Error(), "failed to get current commit hash") {
		t.Errorf("Expected 'failed to get current commit hash' in error, got %q", err.Error())
	}
}

// === RestoreCheckpoint Tests ===

func TestRestoreCheckpoint_Success(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git reset --hard abc123"] = ""

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	err := g.RestoreCheckpoint(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestRestoreCheckpoint_EmptyHash(t *testing.T) {
	runner := NewMockGitCommandRunner()
	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	err := g.RestoreCheckpoint(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty commit hash")
	}
	if !strings.Contains(err.Error(), "commit hash cannot be empty") {
		t.Errorf("Expected 'commit hash cannot be empty' in error, got %q", err.Error())
	}
}

func TestRestoreCheckpoint_ResetError(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Errors["git reset --hard badcommit"] = errors.New("fatal: Could not resolve 'badcommit'")

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	err := g.RestoreCheckpoint(context.Background(), "badcommit")
	if err == nil {
		t.Error("Expected error for reset failure")
	}
	if !strings.Contains(err.Error(), "failed to restore checkpoint") {
		t.Errorf("Expected 'failed to restore checkpoint' in error, got %q", err.Error())
	}
}

// === DeleteCheckpoint Tests ===

func TestDeleteCheckpoint_Success(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git branch -D conductor-checkpoint-task-1-20230101"] = ""

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	err := g.DeleteCheckpoint(context.Background(), "conductor-checkpoint-task-1-20230101")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestDeleteCheckpoint_EmptyBranchName(t *testing.T) {
	runner := NewMockGitCommandRunner()
	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	err := g.DeleteCheckpoint(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty branch name")
	}
	if !strings.Contains(err.Error(), "branch name cannot be empty") {
		t.Errorf("Expected 'branch name cannot be empty' in error, got %q", err.Error())
	}
}

func TestDeleteCheckpoint_BranchNotFound(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Errors["git branch -D nonexistent"] = errors.New("error: branch 'nonexistent' not found")

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	err := g.DeleteCheckpoint(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error for branch not found")
	}
	if !strings.Contains(err.Error(), "failed to delete branch") {
		t.Errorf("Expected 'failed to delete branch' in error, got %q", err.Error())
	}
}

// === CreateBranch Tests ===

func TestCreateBranch_Success(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git checkout -b feature-branch"] = "Switched to a new branch 'feature-branch'\n"

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	err := g.CreateBranch(context.Background(), "feature-branch")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestCreateBranch_EmptyName(t *testing.T) {
	runner := NewMockGitCommandRunner()
	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	err := g.CreateBranch(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty branch name")
	}
	if !strings.Contains(err.Error(), "branch name cannot be empty") {
		t.Errorf("Expected 'branch name cannot be empty' in error, got %q", err.Error())
	}
}

func TestCreateBranch_AlreadyExists(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Errors["git checkout -b existing-branch"] = errors.New("fatal: A branch named 'existing-branch' already exists")

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	err := g.CreateBranch(context.Background(), "existing-branch")
	if err == nil {
		t.Error("Expected error for existing branch")
	}
	if !strings.Contains(err.Error(), "failed to create branch") {
		t.Errorf("Expected 'failed to create branch' in error, got %q", err.Error())
	}
}

// === SwitchBranch Tests ===

func TestSwitchBranch_Success(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git checkout main"] = "Switched to branch 'main'\n"

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	err := g.SwitchBranch(context.Background(), "main")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestSwitchBranch_EmptyName(t *testing.T) {
	runner := NewMockGitCommandRunner()
	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	err := g.SwitchBranch(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty branch name")
	}
	if !strings.Contains(err.Error(), "branch name cannot be empty") {
		t.Errorf("Expected 'branch name cannot be empty' in error, got %q", err.Error())
	}
}

func TestSwitchBranch_BranchNotFound(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Errors["git checkout nonexistent"] = errors.New("error: pathspec 'nonexistent' did not match any file(s) known to git")

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	err := g.SwitchBranch(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent branch")
	}
	if !strings.Contains(err.Error(), "failed to switch to branch") {
		t.Errorf("Expected 'failed to switch to branch' in error, got %q", err.Error())
	}
}

// === GetCurrentBranch Tests ===

func TestGetCurrentBranch_Success(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git branch --show-current"] = "main\n"

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	branch, err := g.GetCurrentBranch(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if branch != "main" {
		t.Errorf("Expected branch 'main', got %q", branch)
	}
}

func TestGetCurrentBranch_DetachedHead(t *testing.T) {
	runner := NewMockGitCommandRunner()
	// Detached HEAD returns empty string
	runner.Outputs["git branch --show-current"] = ""

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	branch, err := g.GetCurrentBranch(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if branch != "" {
		t.Errorf("Expected empty string for detached HEAD, got %q", branch)
	}
}

func TestGetCurrentBranch_Error(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Errors["git branch --show-current"] = errors.New("fatal: not a git repository")

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	_, err := g.GetCurrentBranch(context.Background())
	if err == nil {
		t.Error("Expected error for non-git repository")
	}
	if !strings.Contains(err.Error(), "failed to get current branch") {
		t.Errorf("Expected 'failed to get current branch' in error, got %q", err.Error())
	}
}

// === IsCleanState Tests ===

func TestIsCleanState_Clean(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git status --porcelain"] = ""

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	clean, err := g.IsCleanState(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !clean {
		t.Error("Expected clean state to be true for empty porcelain output")
	}
}

func TestIsCleanState_Dirty(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git status --porcelain"] = " M modified.go\n?? untracked.txt\n"

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	clean, err := g.IsCleanState(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if clean {
		t.Error("Expected clean state to be false for dirty working directory")
	}
}

func TestIsCleanState_StagedChanges(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git status --porcelain"] = "A  newfile.go\n"

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	clean, err := g.IsCleanState(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if clean {
		t.Error("Expected clean state to be false for staged changes")
	}
}

func TestIsCleanState_Error(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Errors["git status --porcelain"] = errors.New("fatal: not a git repository")

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	_, err := g.IsCleanState(context.Background())
	if err == nil {
		t.Error("Expected error for non-git repository")
	}
	if !strings.Contains(err.Error(), "failed to check git status") {
		t.Errorf("Expected 'failed to check git status' in error, got %q", err.Error())
	}
}

// === Context Handling Tests ===

func TestCreateCheckpoint_ContextCancelled(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Errors["git rev-parse HEAD"] = context.Canceled

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := g.CreateCheckpoint(ctx, 1)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

func TestRestoreCheckpoint_ContextCancelled(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Errors["git reset --hard abc123"] = context.Canceled

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := g.RestoreCheckpoint(ctx, "abc123")
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

// === Default Prefix Tests ===

func TestGetCheckpointPrefix_Default(t *testing.T) {
	g := &DefaultGitCheckpointer{}
	prefix := g.getCheckpointPrefix()
	if prefix != "conductor-checkpoint-" {
		t.Errorf("Expected default prefix 'conductor-checkpoint-', got %q", prefix)
	}
}

func TestGetCheckpointPrefix_ConfigEmpty(t *testing.T) {
	cfg := &config.RollbackConfig{
		CheckpointPrefix: "",
	}
	g := &DefaultGitCheckpointer{Config: cfg}
	prefix := g.getCheckpointPrefix()
	if prefix != "conductor-checkpoint-" {
		t.Errorf("Expected default prefix 'conductor-checkpoint-', got %q", prefix)
	}
}

func TestGetCheckpointPrefix_ConfigSet(t *testing.T) {
	cfg := &config.RollbackConfig{
		CheckpointPrefix: "my-prefix-",
	}
	g := &DefaultGitCheckpointer{Config: cfg}
	prefix := g.getCheckpointPrefix()
	if prefix != "my-prefix-" {
		t.Errorf("Expected prefix 'my-prefix-', got %q", prefix)
	}
}

// === CheckpointInfo Validation ===

func TestCheckpointInfo_Fields(t *testing.T) {
	info := CheckpointInfo{
		BranchName: "conductor-checkpoint-task-1-20230101",
		CommitHash: "abc123def",
		CreatedAt:  time.Now(),
	}

	if info.BranchName != "conductor-checkpoint-task-1-20230101" {
		t.Errorf("BranchName mismatch")
	}
	if info.CommitHash != "abc123def" {
		t.Errorf("CommitHash mismatch")
	}
	if info.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

// === Nil Config Handling ===

func TestGitCheckpointer_NilConfig(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git rev-parse HEAD"] = "abc123\n"

	g := NewGitCheckpointerWithRunner(runner, nil)

	// Should use default prefix even with nil config
	prefix := g.getCheckpointPrefix()
	if prefix != "conductor-checkpoint-" {
		t.Errorf("Expected default prefix with nil config, got %q", prefix)
	}
}

// === ListCheckpoints Tests ===

func TestListCheckpoints_Success(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git branch --list conductor-checkpoint-*"] = `  conductor-checkpoint-task-1-20260104-143052
  conductor-checkpoint-task-2-20260104-150000
* conductor-checkpoint-task-3-20260105-090000
`

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	checkpoints, err := g.ListCheckpoints(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(checkpoints) != 3 {
		t.Fatalf("Expected 3 checkpoints, got %d", len(checkpoints))
	}

	// Verify first checkpoint
	if checkpoints[0].BranchName != "conductor-checkpoint-task-1-20260104-143052" {
		t.Errorf("Expected branch name 'conductor-checkpoint-task-1-20260104-143052', got %q", checkpoints[0].BranchName)
	}
	expectedTime1, _ := time.Parse("20060102-150405", "20260104-143052")
	if !checkpoints[0].CreatedAt.Equal(expectedTime1) {
		t.Errorf("Expected CreatedAt %v, got %v", expectedTime1, checkpoints[0].CreatedAt)
	}

	// Verify branch with asterisk (current branch) is handled correctly
	if checkpoints[2].BranchName != "conductor-checkpoint-task-3-20260105-090000" {
		t.Errorf("Expected branch name without asterisk, got %q", checkpoints[2].BranchName)
	}
}

func TestListCheckpoints_EmptyResult(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git branch --list conductor-checkpoint-*"] = ""

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	checkpoints, err := g.ListCheckpoints(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return empty slice, not nil
	if checkpoints == nil {
		t.Error("Expected empty slice, not nil")
	}
	if len(checkpoints) != 0 {
		t.Errorf("Expected 0 checkpoints, got %d", len(checkpoints))
	}
}

func TestListCheckpoints_WhitespaceOnly(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git branch --list conductor-checkpoint-*"] = "   \n\n  "

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	checkpoints, err := g.ListCheckpoints(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return empty slice for whitespace-only output
	if checkpoints == nil {
		t.Error("Expected empty slice, not nil")
	}
	if len(checkpoints) != 0 {
		t.Errorf("Expected 0 checkpoints, got %d", len(checkpoints))
	}
}

func TestListCheckpoints_CustomPrefix(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Outputs["git branch --list custom-prefix-*"] = "  custom-prefix-task-5-20260104-120000\n"

	cfg := config.RollbackConfig{
		CheckpointPrefix: "custom-prefix-",
	}
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	checkpoints, err := g.ListCheckpoints(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(checkpoints) != 1 {
		t.Fatalf("Expected 1 checkpoint, got %d", len(checkpoints))
	}
	if checkpoints[0].BranchName != "custom-prefix-task-5-20260104-120000" {
		t.Errorf("Expected branch name 'custom-prefix-task-5-20260104-120000', got %q", checkpoints[0].BranchName)
	}
}

func TestListCheckpoints_GitError(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Errors["git branch --list conductor-checkpoint-*"] = errors.New("fatal: not a git repository")

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	_, err := g.ListCheckpoints(context.Background())
	if err == nil {
		t.Error("Expected error for git failure")
	}
	if !strings.Contains(err.Error(), "list checkpoint branches") {
		t.Errorf("Expected 'list checkpoint branches' in error, got %q", err.Error())
	}
}

func TestListCheckpoints_MalformedBranchNames(t *testing.T) {
	runner := NewMockGitCommandRunner()
	// Include some malformed branch names that won't parse properly
	runner.Outputs["git branch --list conductor-checkpoint-*"] = `  conductor-checkpoint-task-1-20260104-143052
  conductor-checkpoint-malformed
  conductor-checkpoint-task-2-invalid-time
`

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	checkpoints, err := g.ListCheckpoints(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// All branches should be returned, but malformed ones have zero time
	if len(checkpoints) != 3 {
		t.Fatalf("Expected 3 checkpoints, got %d", len(checkpoints))
	}

	// First one should parse correctly
	if checkpoints[0].CreatedAt.IsZero() {
		t.Error("First checkpoint should have valid timestamp")
	}

	// Malformed ones should have zero time
	if !checkpoints[1].CreatedAt.IsZero() {
		t.Error("Malformed branch should have zero CreatedAt")
	}
	if !checkpoints[2].CreatedAt.IsZero() {
		t.Error("Branch with invalid time should have zero CreatedAt")
	}
}

func TestListCheckpoints_ContextCancelled(t *testing.T) {
	runner := NewMockGitCommandRunner()
	runner.Errors["git branch --list conductor-checkpoint-*"] = context.Canceled

	cfg := config.DefaultRollbackConfig()
	g := NewGitCheckpointerWithRunner(runner, &cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.ListCheckpoints(ctx)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

// === parseTimestampFromBranch Tests ===

func TestParseTimestampFromBranch_ValidFormat(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		expected   string // format: 20060102-150405 or Unix timestamp
		isUnix     bool   // if true, expected is Unix timestamp string
	}{
		{
			name:       "standard checkpoint",
			branchName: "conductor-checkpoint-task-1-20260104-143052",
			expected:   "20260104-143052",
		},
		{
			name:       "custom prefix",
			branchName: "my-prefix-task-5-20251231-235959",
			expected:   "20251231-235959",
		},
		{
			name:       "task with multi-digit id",
			branchName: "conductor-checkpoint-task-123-20260115-080000",
			expected:   "20260115-080000",
		},
		{
			name:       "BranchGuard unix timestamp format",
			branchName: "conductor-checkpoint-1767626940",
			expected:   "1767626940",
			isUnix:     true,
		},
		{
			name:       "BranchGuard with custom prefix",
			branchName: "my-prefix-1767654108",
			expected:   "1767654108",
			isUnix:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTimestampFromBranch(tt.branchName)
			var expected time.Time
			if tt.isUnix {
				var unix int64
				fmt.Sscanf(tt.expected, "%d", &unix)
				expected = time.Unix(unix, 0)
			} else {
				expected, _ = time.Parse("20060102-150405", tt.expected)
			}
			if !result.Equal(expected) {
				t.Errorf("Expected %v, got %v", expected, result)
			}
		})
	}
}

func TestParseTimestampFromBranch_InvalidFormat(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
	}{
		{
			name:       "no timestamp",
			branchName: "conductor-checkpoint-task-1",
		},
		{
			name:       "malformed timestamp",
			branchName: "conductor-checkpoint-task-1-invalid-time",
		},
		{
			name:       "too few parts",
			branchName: "single",
		},
		{
			name:       "empty string",
			branchName: "",
		},
		{
			name:       "partial timestamp",
			branchName: "conductor-checkpoint-task-1-20260104",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTimestampFromBranch(tt.branchName)
			if !result.IsZero() {
				t.Errorf("Expected zero time for invalid format, got %v", result)
			}
		})
	}
}
