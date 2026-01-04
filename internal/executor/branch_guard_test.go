package executor

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/models"
)

// MockBranchGuardLogger implements RuntimeEnforcementLogger for testing.
type MockBranchGuardLogger struct {
	Warnings []string
	Infos    []string
}

func NewMockBranchGuardLogger() *MockBranchGuardLogger {
	return &MockBranchGuardLogger{
		Warnings: make([]string, 0),
		Infos:    make([]string, 0),
	}
}

func (m *MockBranchGuardLogger) LogTestCommands(entries []models.TestCommandResult)                   {}
func (m *MockBranchGuardLogger) LogCriterionVerifications(entries []models.CriterionVerificationResult) {}
func (m *MockBranchGuardLogger) LogDocTargetVerifications(entries []models.DocTargetResult)           {}
func (m *MockBranchGuardLogger) LogErrorPattern(pattern interface{})                                   {}
func (m *MockBranchGuardLogger) LogDetectedError(detected interface{})                                 {}
func (m *MockBranchGuardLogger) Warnf(format string, args ...interface{})                              { m.Warnings = append(m.Warnings, format) }
func (m *MockBranchGuardLogger) Info(message string)                                                   { m.Infos = append(m.Infos, message) }
func (m *MockBranchGuardLogger) Infof(format string, args ...interface{})                              { m.Infos = append(m.Infos, format) }

// MockGitCheckpointerForGuard implements GitCheckpointer for branch guard testing.
type MockGitCheckpointerForGuard struct {
	CurrentBranch  string
	IsClean        bool
	BranchesCreated []string
	SwitchedTo     []string
	Errors         map[string]error
}

func NewMockGitCheckpointerForGuard() *MockGitCheckpointerForGuard {
	return &MockGitCheckpointerForGuard{
		CurrentBranch:   "main",
		IsClean:         true,
		BranchesCreated: make([]string, 0),
		SwitchedTo:      make([]string, 0),
		Errors:          make(map[string]error),
	}
}

func (m *MockGitCheckpointerForGuard) CreateCheckpoint(ctx context.Context, taskNumber int) (*CheckpointInfo, error) {
	if err, ok := m.Errors["CreateCheckpoint"]; ok && err != nil {
		return nil, err
	}
	return &CheckpointInfo{
		BranchName: "conductor-checkpoint-task-1",
		CommitHash: "abc123",
	}, nil
}

func (m *MockGitCheckpointerForGuard) RestoreCheckpoint(ctx context.Context, commitHash string) error {
	if err, ok := m.Errors["RestoreCheckpoint"]; ok && err != nil {
		return err
	}
	return nil
}

func (m *MockGitCheckpointerForGuard) DeleteCheckpoint(ctx context.Context, branchName string) error {
	if err, ok := m.Errors["DeleteCheckpoint"]; ok && err != nil {
		return err
	}
	return nil
}

func (m *MockGitCheckpointerForGuard) CreateBranch(ctx context.Context, branchName string) error {
	if err, ok := m.Errors["CreateBranch"]; ok && err != nil {
		return err
	}
	m.BranchesCreated = append(m.BranchesCreated, branchName)
	m.CurrentBranch = branchName
	return nil
}

func (m *MockGitCheckpointerForGuard) SwitchBranch(ctx context.Context, branchName string) error {
	if err, ok := m.Errors["SwitchBranch"]; ok && err != nil {
		return err
	}
	m.SwitchedTo = append(m.SwitchedTo, branchName)
	m.CurrentBranch = branchName
	return nil
}

func (m *MockGitCheckpointerForGuard) GetCurrentBranch(ctx context.Context) (string, error) {
	if err, ok := m.Errors["GetCurrentBranch"]; ok && err != nil {
		return "", err
	}
	return m.CurrentBranch, nil
}

func (m *MockGitCheckpointerForGuard) IsCleanState(ctx context.Context) (bool, error) {
	if err, ok := m.Errors["IsCleanState"]; ok && err != nil {
		return false, err
	}
	return m.IsClean, nil
}

func (m *MockGitCheckpointerForGuard) ListCheckpoints(ctx context.Context) ([]CheckpointInfo, error) {
	if err, ok := m.Errors["ListCheckpoints"]; ok && err != nil {
		return nil, err
	}
	return []CheckpointInfo{}, nil
}

// === Constructor Tests ===

func TestNewBranchGuard(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	cfg := config.DefaultRollbackConfig()
	logger := NewMockBranchGuardLogger()

	guard := NewBranchGuard(checkpointer, &cfg, logger, "test-plan.md")

	if guard == nil {
		t.Fatal("Expected non-nil BranchGuard")
	}
	if guard.Checkpointer != checkpointer {
		t.Error("Expected Checkpointer to be set")
	}
	if guard.Config == nil {
		t.Error("Expected Config to be set")
	}
	if guard.Logger == nil {
		t.Error("Expected Logger to be set")
	}
	if guard.PlanName != "test-plan.md" {
		t.Errorf("Expected PlanName 'test-plan.md', got %q", guard.PlanName)
	}
}

func TestNewBranchGuard_NilCheckpointer(t *testing.T) {
	cfg := config.DefaultRollbackConfig()
	logger := NewMockBranchGuardLogger()

	guard := NewBranchGuard(nil, &cfg, logger, "test-plan.md")

	if guard != nil {
		t.Error("Expected nil BranchGuard when checkpointer is nil")
	}
}

func TestNewBranchGuard_NilConfig(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	logger := NewMockBranchGuardLogger()

	guard := NewBranchGuard(checkpointer, nil, logger, "test-plan.md")

	if guard != nil {
		t.Error("Expected nil BranchGuard when config is nil")
	}
}

// === Guard Tests ===

func TestGuard_ProtectedBranch(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	checkpointer.CurrentBranch = "main"
	cfg := config.DefaultRollbackConfig()
	logger := NewMockBranchGuardLogger()

	guard := NewBranchGuard(checkpointer, &cfg, logger, "git-rollback")

	result, err := guard.Guard(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.OriginalBranch != "main" {
		t.Errorf("Expected OriginalBranch 'main', got %q", result.OriginalBranch)
	}
	if !result.WasProtected {
		t.Error("Expected WasProtected to be true for 'main'")
	}
	if result.CheckpointBranch == "" {
		t.Error("Expected CheckpointBranch to be set")
	}
	if result.WorkingBranch == "" {
		t.Error("Expected WorkingBranch to be set for protected branch")
	}
	if !strings.HasPrefix(result.WorkingBranch, "conductor-run/") {
		t.Errorf("Expected WorkingBranch to start with 'conductor-run/', got %q", result.WorkingBranch)
	}

	// Verify branches were created
	if len(checkpointer.BranchesCreated) != 2 {
		t.Errorf("Expected 2 branches created (checkpoint + working), got %d", len(checkpointer.BranchesCreated))
	}

	// Verify warning was logged
	foundWarning := false
	for _, w := range logger.Warnings {
		if strings.Contains(w, "protected branch") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Error("Expected warning about protected branch")
	}
}

func TestGuard_NonProtectedBranch(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	checkpointer.CurrentBranch = "feature/my-feature"
	cfg := config.DefaultRollbackConfig()
	logger := NewMockBranchGuardLogger()

	guard := NewBranchGuard(checkpointer, &cfg, logger, "test-plan")

	result, err := guard.Guard(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.OriginalBranch != "feature/my-feature" {
		t.Errorf("Expected OriginalBranch 'feature/my-feature', got %q", result.OriginalBranch)
	}
	if result.WasProtected {
		t.Error("Expected WasProtected to be false for feature branch")
	}
	if result.CheckpointBranch == "" {
		t.Error("Expected CheckpointBranch to be set even for non-protected branch")
	}
	if result.WorkingBranch != "" {
		t.Errorf("Expected WorkingBranch to be empty for non-protected branch, got %q", result.WorkingBranch)
	}

	// Verify only checkpoint branch was created (no working branch switch)
	if len(checkpointer.BranchesCreated) != 1 {
		t.Errorf("Expected 1 branch created (checkpoint only), got %d", len(checkpointer.BranchesCreated))
	}
}

func TestGuard_DirtyState(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	checkpointer.IsClean = false
	cfg := config.DefaultRollbackConfig()
	logger := NewMockBranchGuardLogger()

	guard := NewBranchGuard(checkpointer, &cfg, logger, "test-plan")

	_, err := guard.Guard(context.Background())
	if err == nil {
		t.Fatal("Expected error for dirty state")
	}
	if !strings.Contains(err.Error(), "uncommitted changes detected") {
		t.Errorf("Expected error about uncommitted changes, got %q", err.Error())
	}
	// Verify actionable suggestions in error
	if !strings.Contains(err.Error(), "git add") {
		t.Error("Expected 'git add' suggestion in error")
	}
	if !strings.Contains(err.Error(), "git stash") {
		t.Error("Expected 'git stash' suggestion in error")
	}
}

func TestGuard_DirtyStateAllowed(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	checkpointer.IsClean = false
	checkpointer.CurrentBranch = "feature/test"
	cfg := config.DefaultRollbackConfig()
	cfg.RequireCleanState = false // Allow dirty state
	logger := NewMockBranchGuardLogger()

	guard := NewBranchGuard(checkpointer, &cfg, logger, "test-plan")

	result, err := guard.Guard(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error when RequireCleanState is false: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestGuard_NilGuard(t *testing.T) {
	var guard *BranchGuard = nil

	result, err := guard.Guard(context.Background())
	if err != nil {
		t.Errorf("Expected nil error for nil guard, got %v", err)
	}
	if result != nil {
		t.Error("Expected nil result for nil guard")
	}
}

// === EnsureSafeState Tests ===

func TestEnsureSafeState_Clean(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	checkpointer.IsClean = true
	cfg := config.DefaultRollbackConfig()

	guard := NewBranchGuard(checkpointer, &cfg, nil, "test")

	err := guard.EnsureSafeState(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error for clean state: %v", err)
	}
}

func TestEnsureSafeState_Dirty(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	checkpointer.IsClean = false
	cfg := config.DefaultRollbackConfig()

	guard := NewBranchGuard(checkpointer, &cfg, nil, "test")

	err := guard.EnsureSafeState(context.Background())
	if err == nil {
		t.Fatal("Expected error for dirty state")
	}

	// Verify all three options are in the error message
	errMsg := err.Error()
	if !strings.Contains(errMsg, "Option 1") || !strings.Contains(errMsg, "git add") {
		t.Error("Expected Option 1 with git add")
	}
	if !strings.Contains(errMsg, "Option 2") || !strings.Contains(errMsg, "git stash") {
		t.Error("Expected Option 2 with git stash")
	}
	if !strings.Contains(errMsg, "Option 3") || !strings.Contains(errMsg, "git checkout") {
		t.Error("Expected Option 3 with git checkout")
	}
}

func TestEnsureSafeState_CheckError(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	checkpointer.Errors["IsCleanState"] = errors.New("not a git repository")
	cfg := config.DefaultRollbackConfig()

	guard := NewBranchGuard(checkpointer, &cfg, nil, "test")

	err := guard.EnsureSafeState(context.Background())
	if err == nil {
		t.Fatal("Expected error for IsCleanState failure")
	}
	if !strings.Contains(err.Error(), "failed to check git state") {
		t.Errorf("Expected 'failed to check git state' in error, got %q", err.Error())
	}
}

// === Protected Branch Detection Tests ===

func TestIsProtectedBranch_Main(t *testing.T) {
	cfg := config.DefaultRollbackConfig()
	guard := &BranchGuard{Config: &cfg}

	if !guard.isProtectedBranch("main") {
		t.Error("Expected 'main' to be protected")
	}
}

func TestIsProtectedBranch_Master(t *testing.T) {
	cfg := config.DefaultRollbackConfig()
	guard := &BranchGuard{Config: &cfg}

	if !guard.isProtectedBranch("master") {
		t.Error("Expected 'master' to be protected")
	}
}

func TestIsProtectedBranch_Develop(t *testing.T) {
	cfg := config.DefaultRollbackConfig()
	guard := &BranchGuard{Config: &cfg}

	if !guard.isProtectedBranch("develop") {
		t.Error("Expected 'develop' to be protected")
	}
}

func TestIsProtectedBranch_Feature(t *testing.T) {
	cfg := config.DefaultRollbackConfig()
	guard := &BranchGuard{Config: &cfg}

	if guard.isProtectedBranch("feature/my-feature") {
		t.Error("Expected 'feature/my-feature' to NOT be protected")
	}
}

func TestIsProtectedBranch_CustomList(t *testing.T) {
	cfg := config.RollbackConfig{
		ProtectedBranches: []string{"production", "staging"},
	}
	guard := &BranchGuard{Config: &cfg}

	if !guard.isProtectedBranch("production") {
		t.Error("Expected 'production' to be protected")
	}
	if !guard.isProtectedBranch("staging") {
		t.Error("Expected 'staging' to be protected")
	}
	if guard.isProtectedBranch("main") {
		t.Error("Expected 'main' to NOT be protected with custom list")
	}
}

func TestIsProtectedBranch_NilConfig(t *testing.T) {
	guard := &BranchGuard{Config: nil}

	// Should use defaults when config is nil
	if !guard.isProtectedBranch("main") {
		t.Error("Expected 'main' to be protected with nil config (defaults)")
	}
}

// === Branch Name Generation Tests ===

func TestSanitizeBranchName_Simple(t *testing.T) {
	result := sanitizeBranchName("my-plan")
	if result != "my-plan" {
		t.Errorf("Expected 'my-plan', got %q", result)
	}
}

func TestSanitizeBranchName_WithExtension(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"plan.md", "plan"},
		{"feature.yaml", "feature"},
		{"task.yml", "task"},
	}

	for _, tc := range tests {
		result := sanitizeBranchName(tc.input)
		if result != tc.expected {
			t.Errorf("sanitizeBranchName(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestSanitizeBranchName_SpecialCharacters(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my plan", "my-plan"},
		{"plan/with/slashes", "plan-with-slashes"},
		{"plan@special!chars", "plan-special-chars"},
		{"--leading-hyphens--", "leading-hyphens"},
	}

	for _, tc := range tests {
		result := sanitizeBranchName(tc.input)
		if result != tc.expected {
			t.Errorf("sanitizeBranchName(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestSanitizeBranchName_Empty(t *testing.T) {
	result := sanitizeBranchName("")
	if !strings.HasPrefix(result, "plan-") {
		t.Errorf("Expected empty input to generate 'plan-*' fallback, got %q", result)
	}
}

func TestGenerateWorkingBranchName_WithPlanName(t *testing.T) {
	cfg := config.DefaultRollbackConfig()
	guard := &BranchGuard{
		Config:   &cfg,
		PlanName: "git-rollback.md",
	}

	name := guard.generateWorkingBranchName()
	if name != "conductor-run/git-rollback" {
		t.Errorf("Expected 'conductor-run/git-rollback', got %q", name)
	}
}

func TestGenerateWorkingBranchName_CustomPrefix(t *testing.T) {
	cfg := config.RollbackConfig{
		WorkingBranchPrefix: "work/",
	}
	guard := &BranchGuard{
		Config:   &cfg,
		PlanName: "my-feature",
	}

	name := guard.generateWorkingBranchName()
	if name != "work/my-feature" {
		t.Errorf("Expected 'work/my-feature', got %q", name)
	}
}

func TestGenerateWorkingBranchName_NoPlanName(t *testing.T) {
	cfg := config.DefaultRollbackConfig()
	guard := &BranchGuard{
		Config:   &cfg,
		PlanName: "",
	}

	name := guard.generateWorkingBranchName()
	if !strings.HasPrefix(name, "conductor-run/") {
		t.Errorf("Expected name to start with 'conductor-run/', got %q", name)
	}
}

// === Prefix Tests ===

func TestBranchGuard_GetCheckpointPrefix_Default(t *testing.T) {
	guard := &BranchGuard{Config: nil}
	prefix := guard.getCheckpointPrefix()
	if prefix != "conductor-checkpoint-" {
		t.Errorf("Expected default prefix 'conductor-checkpoint-', got %q", prefix)
	}
}

func TestBranchGuard_GetCheckpointPrefix_Custom(t *testing.T) {
	cfg := config.RollbackConfig{
		CheckpointPrefix: "cp-",
	}
	guard := &BranchGuard{Config: &cfg}
	prefix := guard.getCheckpointPrefix()
	if prefix != "cp-" {
		t.Errorf("Expected prefix 'cp-', got %q", prefix)
	}
}

func TestBranchGuard_GetWorkingBranchPrefix_Default(t *testing.T) {
	guard := &BranchGuard{Config: nil}
	prefix := guard.getWorkingBranchPrefix()
	if prefix != "conductor-run/" {
		t.Errorf("Expected default prefix 'conductor-run/', got %q", prefix)
	}
}

func TestBranchGuard_GetWorkingBranchPrefix_Custom(t *testing.T) {
	cfg := config.RollbackConfig{
		WorkingBranchPrefix: "wip/",
	}
	guard := &BranchGuard{Config: &cfg}
	prefix := guard.getWorkingBranchPrefix()
	if prefix != "wip/" {
		t.Errorf("Expected prefix 'wip/', got %q", prefix)
	}
}

// === Error Handling Tests ===

func TestGuard_GetCurrentBranchError(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	checkpointer.Errors["GetCurrentBranch"] = errors.New("not a git repository")
	cfg := config.DefaultRollbackConfig()

	guard := NewBranchGuard(checkpointer, &cfg, nil, "test")

	_, err := guard.Guard(context.Background())
	if err == nil {
		t.Fatal("Expected error for GetCurrentBranch failure")
	}
	if !strings.Contains(err.Error(), "failed to get current branch") {
		t.Errorf("Expected 'failed to get current branch' in error, got %q", err.Error())
	}
}

func TestGuard_CreateBranchError(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	checkpointer.Errors["CreateBranch"] = errors.New("branch already exists")
	cfg := config.DefaultRollbackConfig()

	guard := NewBranchGuard(checkpointer, &cfg, nil, "test")

	_, err := guard.Guard(context.Background())
	if err == nil {
		t.Fatal("Expected error for CreateBranch failure")
	}
	if !strings.Contains(err.Error(), "failed to create checkpoint branch") {
		t.Errorf("Expected 'failed to create checkpoint branch' in error, got %q", err.Error())
	}
}

// === BranchGuardResult Tests ===

func TestBranchGuardResult_Fields(t *testing.T) {
	result := BranchGuardResult{
		OriginalBranch:   "main",
		CheckpointBranch: "conductor-checkpoint-123456",
		WorkingBranch:    "conductor-run/my-plan",
		WasProtected:     true,
		CheckpointCommit: "abc123def",
	}

	if result.OriginalBranch != "main" {
		t.Errorf("OriginalBranch mismatch")
	}
	if result.CheckpointBranch != "conductor-checkpoint-123456" {
		t.Errorf("CheckpointBranch mismatch")
	}
	if result.WorkingBranch != "conductor-run/my-plan" {
		t.Errorf("WorkingBranch mismatch")
	}
	if !result.WasProtected {
		t.Error("WasProtected should be true")
	}
	if result.CheckpointCommit != "abc123def" {
		t.Errorf("CheckpointCommit mismatch")
	}
}

// === Context Handling Tests ===

func TestGuard_ContextCancelled(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	checkpointer.Errors["IsCleanState"] = context.Canceled
	cfg := config.DefaultRollbackConfig()

	guard := NewBranchGuard(checkpointer, &cfg, nil, "test")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := guard.Guard(ctx)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

// === Integration Flow Tests ===

func TestGuard_FullProtectedBranchFlow(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	checkpointer.CurrentBranch = "main"
	cfg := config.DefaultRollbackConfig()
	logger := NewMockBranchGuardLogger()

	guard := NewBranchGuard(checkpointer, &cfg, logger, "feature-implementation.yaml")

	result, err := guard.Guard(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify the complete flow
	if result.OriginalBranch != "main" {
		t.Errorf("OriginalBranch: expected 'main', got %q", result.OriginalBranch)
	}
	if !result.WasProtected {
		t.Error("WasProtected should be true")
	}
	if !strings.HasPrefix(result.CheckpointBranch, "conductor-checkpoint-") {
		t.Errorf("CheckpointBranch should start with prefix, got %q", result.CheckpointBranch)
	}
	if result.WorkingBranch != "conductor-run/feature-implementation" {
		t.Errorf("WorkingBranch: expected 'conductor-run/feature-implementation', got %q", result.WorkingBranch)
	}
	if result.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	// Verify logging occurred
	if len(logger.Warnings) == 0 {
		t.Error("Expected warning logs for protected branch")
	}
	if len(logger.Infos) == 0 {
		t.Error("Expected info logs for branch creation")
	}
}

func TestGuard_FullNonProtectedBranchFlow(t *testing.T) {
	checkpointer := NewMockGitCheckpointerForGuard()
	checkpointer.CurrentBranch = "feature/existing-work"
	cfg := config.DefaultRollbackConfig()
	logger := NewMockBranchGuardLogger()

	guard := NewBranchGuard(checkpointer, &cfg, logger, "continue-work")

	result, err := guard.Guard(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify the flow
	if result.OriginalBranch != "feature/existing-work" {
		t.Errorf("OriginalBranch: expected 'feature/existing-work', got %q", result.OriginalBranch)
	}
	if result.WasProtected {
		t.Error("WasProtected should be false for feature branch")
	}
	if result.CheckpointBranch == "" {
		t.Error("CheckpointBranch should be set even for non-protected branch")
	}
	if result.WorkingBranch != "" {
		t.Errorf("WorkingBranch should be empty for non-protected branch, got %q", result.WorkingBranch)
	}

	// No warnings for non-protected branch
	protectedWarnings := 0
	for _, w := range logger.Warnings {
		if strings.Contains(w, "protected branch") {
			protectedWarnings++
		}
	}
	if protectedWarnings > 0 {
		t.Error("Should not warn about protected branch for feature branch")
	}
}
