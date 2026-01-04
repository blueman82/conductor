package executor

import (
	"context"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/models"
)

// branchGuardMockLogger implements RuntimeEnforcementLogger for testing.
type branchGuardMockLogger struct {
	infos  []string
	warns  []string
	errors []string
}

func (m *branchGuardMockLogger) LogTestCommands(entries []models.TestCommandResult) {}
func (m *branchGuardMockLogger) LogCriterionVerifications(entries []models.CriterionVerificationResult) {
}
func (m *branchGuardMockLogger) LogDocTargetVerifications(entries []models.DocTargetResult) {}
func (m *branchGuardMockLogger) LogErrorPattern(pattern interface{})                        {}
func (m *branchGuardMockLogger) LogDetectedError(detected interface{})                      {}
func (m *branchGuardMockLogger) Warnf(format string, args ...interface{}) {
	m.warns = append(m.warns, format)
}
func (m *branchGuardMockLogger) Info(message string) {
	m.infos = append(m.infos, message)
}
func (m *branchGuardMockLogger) Infof(format string, args ...interface{}) {
	m.infos = append(m.infos, format)
}

func newBranchGuardMockLogger() *branchGuardMockLogger {
	return &branchGuardMockLogger{
		infos:  []string{},
		warns:  []string{},
		errors: []string{},
	}
}

// branchGuardMockCheckpointer implements GitCheckpointer for testing.
type branchGuardMockCheckpointer struct {
	currentBranch        string
	isClean              bool
	isCleanError         error
	getCurrentBranchErr  error
	createBranchErr      error
	switchBranchErr      error
	checkpointCreated    string
	createCheckpointErr  error
	deleteCheckpointErr  error
	createCheckpointFunc func(ctx context.Context, taskNumber int) (*CheckpointInfo, error)
	branchesCreated      []string
	switchedBranches     []string
}

func (m *branchGuardMockCheckpointer) CreateCheckpoint(ctx context.Context, taskNumber int) (*CheckpointInfo, error) {
	if m.createCheckpointFunc != nil {
		return m.createCheckpointFunc(ctx, taskNumber)
	}
	if m.createCheckpointErr != nil {
		return nil, m.createCheckpointErr
	}
	return &CheckpointInfo{
		BranchName: "conductor-checkpoint-task-1-20240101-120000",
		CommitHash: "abc123",
		CreatedAt:  time.Now(),
	}, nil
}

func (m *branchGuardMockCheckpointer) RestoreCheckpoint(ctx context.Context, commitHash string) error {
	return nil
}

func (m *branchGuardMockCheckpointer) DeleteCheckpoint(ctx context.Context, branchName string) error {
	return m.deleteCheckpointErr
}

func (m *branchGuardMockCheckpointer) CreateBranch(ctx context.Context, branchName string) error {
	if m.createBranchErr != nil {
		return m.createBranchErr
	}
	m.branchesCreated = append(m.branchesCreated, branchName)
	return nil
}

func (m *branchGuardMockCheckpointer) SwitchBranch(ctx context.Context, branchName string) error {
	if m.switchBranchErr != nil {
		return m.switchBranchErr
	}
	m.switchedBranches = append(m.switchedBranches, branchName)
	return nil
}

func (m *branchGuardMockCheckpointer) GetCurrentBranch(ctx context.Context) (string, error) {
	if m.getCurrentBranchErr != nil {
		return "", m.getCurrentBranchErr
	}
	return m.currentBranch, nil
}

func (m *branchGuardMockCheckpointer) IsCleanState(ctx context.Context) (bool, error) {
	if m.isCleanError != nil {
		return false, m.isCleanError
	}
	return m.isClean, nil
}

func TestNewBranchGuardHook_NilGuard(t *testing.T) {
	// Test that nil guard returns nil hook (graceful degradation)
	hook := NewBranchGuardHook(nil, nil)
	if hook != nil {
		t.Error("expected nil hook when guard is nil")
	}
}

func TestNewBranchGuardHook_WithGuard(t *testing.T) {
	// Create a minimal BranchGuard for testing
	cfg := &config.RollbackConfig{
		Enabled:             true,
		ProtectedBranches:   []string{"main", "master"},
		WorkingBranchPrefix: "conductor-run/",
		CheckpointPrefix:    "conductor-checkpoint-",
	}
	checkpointer := &branchGuardMockCheckpointer{
		currentBranch: "feature/test",
		isClean:       true,
	}
	logger := newBranchGuardMockLogger()
	guard := NewBranchGuard(checkpointer, cfg, logger, "test-plan.yaml")

	hook := NewBranchGuardHook(guard, logger)
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}
	if hook.guard != guard {
		t.Error("guard not set correctly")
	}
	if hook.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestBranchGuardHook_Guard_NilHook(t *testing.T) {
	// Test that nil hook returns nil (graceful degradation)
	var hook *BranchGuardHook
	result, err := hook.Guard(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

func TestBranchGuardHook_Guard_NilGuard(t *testing.T) {
	// Test that hook with nil guard returns nil (graceful degradation)
	hook := &BranchGuardHook{
		guard:  nil,
		logger: newBranchGuardMockLogger(),
	}
	result, err := hook.Guard(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

func TestBranchGuardHook_Guard_Success_NotProtected(t *testing.T) {
	// Test successful guard on non-protected branch
	logger := newBranchGuardMockLogger()

	cfg := &config.RollbackConfig{
		Enabled:             true,
		RequireCleanState:   true,
		ProtectedBranches:   []string{"main", "master"},
		WorkingBranchPrefix: "conductor-run/",
		CheckpointPrefix:    "conductor-checkpoint-",
	}
	checkpointer := &branchGuardMockCheckpointer{
		currentBranch: "feature/test",
		isClean:       true,
	}

	guard := NewBranchGuard(checkpointer, cfg, logger, "test-plan.yaml")
	hook := NewBranchGuardHook(guard, logger)
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}

	result, err := hook.Guard(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.WasProtected {
		t.Error("expected WasProtected to be false for feature branch")
	}
	if result.OriginalBranch != "feature/test" {
		t.Errorf("expected OriginalBranch 'feature/test', got %q", result.OriginalBranch)
	}

	// Verify logging
	if len(logger.infos) < 2 {
		t.Errorf("expected at least 2 info messages, got %d", len(logger.infos))
	}
}

func TestBranchGuardHook_Guard_Success_Protected(t *testing.T) {
	// Test successful guard on protected branch (creates working branch)
	logger := newBranchGuardMockLogger()

	cfg := &config.RollbackConfig{
		Enabled:             true,
		RequireCleanState:   true,
		ProtectedBranches:   []string{"main", "master"},
		WorkingBranchPrefix: "conductor-run/",
		CheckpointPrefix:    "conductor-checkpoint-",
	}
	checkpointer := &branchGuardMockCheckpointer{
		currentBranch: "main",
		isClean:       true,
	}

	guard := NewBranchGuard(checkpointer, cfg, logger, "test-plan.yaml")
	hook := NewBranchGuardHook(guard, logger)
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}

	result, err := hook.Guard(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.WasProtected {
		t.Error("expected WasProtected to be true for main branch")
	}
	if result.OriginalBranch != "main" {
		t.Errorf("expected OriginalBranch 'main', got %q", result.OriginalBranch)
	}
	if result.WorkingBranch == "" {
		t.Error("expected non-empty WorkingBranch for protected branch")
	}

	// Verify logging includes protected branch message
	foundProtectedLog := false
	for _, msg := range logger.infos {
		if msg == "Branch Guard: Protected branch detected, created working branch '%s'" {
			foundProtectedLog = true
			break
		}
	}
	if !foundProtectedLog {
		t.Error("expected log message about protected branch")
	}
}

func TestBranchGuardHook_Guard_Error_DirtyState(t *testing.T) {
	// Test that dirty state error is returned (blocks execution)
	logger := newBranchGuardMockLogger()

	cfg := &config.RollbackConfig{
		Enabled:             true,
		RequireCleanState:   true,
		ProtectedBranches:   []string{"main", "master"},
		WorkingBranchPrefix: "conductor-run/",
		CheckpointPrefix:    "conductor-checkpoint-",
	}
	checkpointer := &branchGuardMockCheckpointer{
		currentBranch: "feature/test",
		isClean:       false, // Dirty state
	}

	guard := NewBranchGuard(checkpointer, cfg, logger, "test-plan.yaml")
	hook := NewBranchGuardHook(guard, logger)
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}

	result, err := hook.Guard(context.Background())
	if err == nil {
		t.Error("expected error for dirty state")
	}
	if result != nil {
		t.Error("expected nil result on error")
	}
}

func TestBranchGuardHook_Guard_NilLogger(t *testing.T) {
	// Test that nil logger doesn't cause panic
	cfg := &config.RollbackConfig{
		Enabled:             true,
		RequireCleanState:   true,
		ProtectedBranches:   []string{"main", "master"},
		WorkingBranchPrefix: "conductor-run/",
		CheckpointPrefix:    "conductor-checkpoint-",
	}
	checkpointer := &branchGuardMockCheckpointer{
		currentBranch: "feature/test",
		isClean:       true,
	}

	// Use a logger for BranchGuard but nil for hook
	bgLogger := newBranchGuardMockLogger()
	guard := NewBranchGuard(checkpointer, cfg, bgLogger, "test-plan.yaml")

	hook := NewBranchGuardHook(guard, nil) // nil logger for hook
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}

	// Should not panic
	result, err := hook.Guard(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestBranchGuardHook_Guard_ContextCancellation(t *testing.T) {
	// Test that context cancellation is handled without panic
	logger := newBranchGuardMockLogger()

	cfg := &config.RollbackConfig{
		Enabled:             true,
		RequireCleanState:   true,
		ProtectedBranches:   []string{"main", "master"},
		WorkingBranchPrefix: "conductor-run/",
		CheckpointPrefix:    "conductor-checkpoint-",
	}
	checkpointer := &branchGuardMockCheckpointer{
		currentBranch: "feature/test",
		isClean:       true,
	}

	guard := NewBranchGuard(checkpointer, cfg, logger, "test-plan.yaml")
	hook := NewBranchGuardHook(guard, logger)
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should not panic - graceful handling
	_, err := hook.Guard(ctx)
	// Either an error or nil is acceptable, the key is no panic
	_ = err
}

func TestBranchGuardHook_Guard_CleanStateNotRequired(t *testing.T) {
	// Test that dirty state is allowed when RequireCleanState is false
	logger := newBranchGuardMockLogger()

	cfg := &config.RollbackConfig{
		Enabled:             true,
		RequireCleanState:   false, // Not required
		ProtectedBranches:   []string{"main", "master"},
		WorkingBranchPrefix: "conductor-run/",
		CheckpointPrefix:    "conductor-checkpoint-",
	}
	checkpointer := &branchGuardMockCheckpointer{
		currentBranch: "feature/test",
		isClean:       false, // Dirty state - but allowed
	}

	guard := NewBranchGuard(checkpointer, cfg, logger, "test-plan.yaml")
	hook := NewBranchGuardHook(guard, logger)
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}

	result, err := hook.Guard(context.Background())
	if err != nil {
		t.Errorf("expected nil error when RequireCleanState is false, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
