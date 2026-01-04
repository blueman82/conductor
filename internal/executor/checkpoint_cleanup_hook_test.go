package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/config"
)

// mockCheckpointerForCleanup implements GitCheckpointer for cleanup hook tests.
type mockCheckpointerForCleanup struct {
	checkpoints     []CheckpointInfo
	listError       error
	deleteErrors    map[string]error // branchName -> error
	deletedBranches []string
}

func newMockCheckpointerForCleanup() *mockCheckpointerForCleanup {
	return &mockCheckpointerForCleanup{
		deleteErrors:    make(map[string]error),
		deletedBranches: make([]string, 0),
	}
}

func (m *mockCheckpointerForCleanup) ListCheckpoints(ctx context.Context) ([]CheckpointInfo, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	return m.checkpoints, nil
}

func (m *mockCheckpointerForCleanup) DeleteCheckpoint(ctx context.Context, branchName string) error {
	if err, ok := m.deleteErrors[branchName]; ok {
		return err
	}
	m.deletedBranches = append(m.deletedBranches, branchName)
	return nil
}

func (m *mockCheckpointerForCleanup) CreateCheckpoint(ctx context.Context, taskNumber int) (*CheckpointInfo, error) {
	return nil, nil
}

func (m *mockCheckpointerForCleanup) RestoreCheckpoint(ctx context.Context, commitHash string) error {
	return nil
}

func (m *mockCheckpointerForCleanup) CreateBranch(ctx context.Context, branchName string) error {
	return nil
}

func (m *mockCheckpointerForCleanup) SwitchBranch(ctx context.Context, branchName string) error {
	return nil
}

func (m *mockCheckpointerForCleanup) GetCurrentBranch(ctx context.Context) (string, error) {
	return "", nil
}

func (m *mockCheckpointerForCleanup) IsCleanState(ctx context.Context) (bool, error) {
	return true, nil
}

// cleanupMockLogger implements RuntimeEnforcementLogger for cleanup hook tests.
type cleanupMockLogger struct {
	infos    []string
	warnings []string
}

func newMockCleanupLogger() *cleanupMockLogger {
	return &cleanupMockLogger{
		infos:    make([]string, 0),
		warnings: make([]string, 0),
	}
}

func (m *cleanupMockLogger) LogTestCommands(entries []TestCommandResult)                    {}
func (m *cleanupMockLogger) LogCriterionVerifications(entries []CriterionVerificationResult) {}
func (m *cleanupMockLogger) LogDocTargetVerifications(entries []DocTargetResult)            {}
func (m *cleanupMockLogger) LogErrorPattern(pattern interface{})                            {}
func (m *cleanupMockLogger) LogDetectedError(detected interface{})                          {}
func (m *cleanupMockLogger) Info(message string)                                            { m.infos = append(m.infos, message) }
func (m *cleanupMockLogger) Infof(format string, args ...interface{})                       { m.infos = append(m.infos, format) }
func (m *cleanupMockLogger) Warnf(format string, args ...interface{})                       { m.warnings = append(m.warnings, format) }

// === Constructor Tests ===

func TestNewCheckpointCleanupHook_Success(t *testing.T) {
	checkpointer := newMockCheckpointerForCleanup()
	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}

	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)
	if hook == nil {
		t.Fatal("Expected non-nil hook")
	}
	if hook.now == nil {
		t.Error("Expected now function to be set")
	}
}

func TestNewCheckpointCleanupHook_NilCheckpointer(t *testing.T) {
	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}

	hook := NewCheckpointCleanupHook(nil, cfg, nil)
	if hook != nil {
		t.Error("Expected nil hook when checkpointer is nil")
	}
}

func TestNewCheckpointCleanupHook_NilConfig(t *testing.T) {
	checkpointer := newMockCheckpointerForCleanup()

	hook := NewCheckpointCleanupHook(checkpointer, nil, nil)
	if hook != nil {
		t.Error("Expected nil hook when config is nil")
	}
}

func TestNewCheckpointCleanupHook_BothNil(t *testing.T) {
	hook := NewCheckpointCleanupHook(nil, nil, nil)
	if hook != nil {
		t.Error("Expected nil hook when both checkpointer and config are nil")
	}
}

func TestNewCheckpointCleanupHook_DefaultClock(t *testing.T) {
	checkpointer := newMockCheckpointerForCleanup()
	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}

	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)
	if hook == nil {
		t.Fatal("Expected non-nil hook")
	}

	// Verify the now function is set and returns a reasonable time
	now := hook.now()
	if now.IsZero() {
		t.Error("Expected now() to return non-zero time")
	}

	// Verify it's close to current time (within 1 second)
	diff := time.Since(now)
	if diff < 0 || diff > time.Second {
		t.Errorf("Expected now() to return current time, got diff: %v", diff)
	}
}

func TestNewCheckpointCleanupHook_NilLogger(t *testing.T) {
	checkpointer := newMockCheckpointerForCleanup()
	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}

	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)
	if hook == nil {
		t.Fatal("Expected non-nil hook even with nil logger")
	}
	if hook.logger != nil {
		t.Error("Expected nil logger")
	}
}

// === Nil Receiver Tests ===

func TestCheckpointCleanupHook_Cleanup_NilReceiver(t *testing.T) {
	var hook *CheckpointCleanupHook = nil
	deleted, err := hook.Cleanup(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted, got %d", deleted)
	}
}

func TestCheckpointCleanupHook_Enabled_NilReceiver(t *testing.T) {
	var hook *CheckpointCleanupHook = nil
	if hook.Enabled() {
		t.Error("Expected nil hook to be disabled")
	}
}

// === Cleanup Early Return Tests ===

func TestCheckpointCleanupHook_Cleanup_RollbackDisabled(t *testing.T) {
	checkpointer := newMockCheckpointerForCleanup()
	cfg := &config.RollbackConfig{Enabled: false, KeepCheckpointDays: 7}

	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)

	deleted, err := hook.Cleanup(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted, got %d", deleted)
	}
}

func TestCheckpointCleanupHook_Cleanup_ZeroKeepDays(t *testing.T) {
	checkpointer := newMockCheckpointerForCleanup()
	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 0}

	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)

	deleted, err := hook.Cleanup(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted, got %d", deleted)
	}
}

func TestCheckpointCleanupHook_Cleanup_NegativeKeepDays(t *testing.T) {
	checkpointer := newMockCheckpointerForCleanup()
	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: -1}

	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)

	deleted, err := hook.Cleanup(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted, got %d", deleted)
	}
}

// === List Error Test ===

func TestCheckpointCleanupHook_Cleanup_ListError(t *testing.T) {
	checkpointer := newMockCheckpointerForCleanup()
	checkpointer.listError = errors.New("git error")
	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}

	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)

	_, err := hook.Cleanup(context.Background())
	if err == nil {
		t.Error("Expected error when list fails")
	}
}

// === Empty Checkpoints Test ===

func TestCheckpointCleanupHook_Cleanup_NoCheckpoints(t *testing.T) {
	checkpointer := newMockCheckpointerForCleanup()
	checkpointer.checkpoints = []CheckpointInfo{}
	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}

	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)

	deleted, err := hook.Cleanup(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted, got %d", deleted)
	}
}

// === Core Cleanup Logic Tests with Injected Clock ===

func TestCheckpointCleanupHook_Cleanup_DeletesStaleCheckpoints(t *testing.T) {
	// Fixed "now" time for deterministic testing
	fixedNow := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	checkpointer := newMockCheckpointerForCleanup()
	checkpointer.checkpoints = []CheckpointInfo{
		{BranchName: "conductor-checkpoint-task-1-20260103-120000", CreatedAt: time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC)}, // 7 days old - exactly at cutoff
		{BranchName: "conductor-checkpoint-task-2-20260101-120000", CreatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}, // 9 days old - stale
		{BranchName: "conductor-checkpoint-task-3-20260109-120000", CreatedAt: time.Date(2026, 1, 9, 12, 0, 0, 0, time.UTC)}, // 1 day old - fresh
	}

	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}
	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)
	hook.now = func() time.Time { return fixedNow }

	deleted, err := hook.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should delete 2 stale branches (7 days old and 9 days old)
	if deleted != 2 {
		t.Errorf("Expected 2 deleted, got %d", deleted)
	}

	// Verify correct branches were deleted
	if len(checkpointer.deletedBranches) != 2 {
		t.Errorf("Expected 2 branches deleted, got %d", len(checkpointer.deletedBranches))
	}
	expectedDeleted := map[string]bool{
		"conductor-checkpoint-task-1-20260103-120000": true, // at cutoff
		"conductor-checkpoint-task-2-20260101-120000": true, // stale
	}
	for _, branch := range checkpointer.deletedBranches {
		if !expectedDeleted[branch] {
			t.Errorf("Unexpected branch deleted: %s", branch)
		}
	}
}

func TestCheckpointCleanupHook_Cleanup_AllFresh(t *testing.T) {
	fixedNow := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	checkpointer := newMockCheckpointerForCleanup()
	checkpointer.checkpoints = []CheckpointInfo{
		{BranchName: "conductor-checkpoint-task-1-20260109-120000", CreatedAt: time.Date(2026, 1, 9, 12, 0, 0, 0, time.UTC)},  // 1 day old
		{BranchName: "conductor-checkpoint-task-2-20260108-120000", CreatedAt: time.Date(2026, 1, 8, 12, 0, 0, 0, time.UTC)},  // 2 days old
		{BranchName: "conductor-checkpoint-task-3-20260105-120000", CreatedAt: time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)},  // 5 days old
		{BranchName: "conductor-checkpoint-task-4-20260104-120000", CreatedAt: time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC)},  // 6 days old (within retention)
	}

	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}
	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)
	hook.now = func() time.Time { return fixedNow }

	deleted, err := hook.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if deleted != 0 {
		t.Errorf("Expected 0 deleted (all fresh), got %d", deleted)
	}
	if len(checkpointer.deletedBranches) != 0 {
		t.Errorf("Expected no branches deleted, got %v", checkpointer.deletedBranches)
	}
}

func TestCheckpointCleanupHook_Cleanup_AllStale(t *testing.T) {
	fixedNow := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	checkpointer := newMockCheckpointerForCleanup()
	checkpointer.checkpoints = []CheckpointInfo{
		{BranchName: "conductor-checkpoint-task-1-20251220-120000", CreatedAt: time.Date(2025, 12, 20, 12, 0, 0, 0, time.UTC)}, // 21 days old
		{BranchName: "conductor-checkpoint-task-2-20251225-120000", CreatedAt: time.Date(2025, 12, 25, 12, 0, 0, 0, time.UTC)}, // 16 days old
	}

	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}
	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)
	hook.now = func() time.Time { return fixedNow }

	deleted, err := hook.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if deleted != 2 {
		t.Errorf("Expected 2 deleted (all stale), got %d", deleted)
	}
}

// === Zero Time (Invalid Timestamp) Tests ===

func TestCheckpointCleanupHook_Cleanup_SkipsZeroTime(t *testing.T) {
	fixedNow := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	checkpointer := newMockCheckpointerForCleanup()
	checkpointer.checkpoints = []CheckpointInfo{
		{BranchName: "conductor-checkpoint-task-1-malformed", CreatedAt: time.Time{}},                                          // Zero time - skip
		{BranchName: "conductor-checkpoint-task-2-20260101-120000", CreatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)},   // 9 days old - delete
	}

	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}
	logger := newMockCleanupLogger()
	hook := NewCheckpointCleanupHook(checkpointer, cfg, logger)
	hook.now = func() time.Time { return fixedNow }

	deleted, err := hook.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should only delete the valid stale one
	if deleted != 1 {
		t.Errorf("Expected 1 deleted, got %d", deleted)
	}

	// Verify warning was logged for zero time
	if len(logger.warnings) == 0 {
		t.Error("Expected warning for unparseable timestamp")
	}
}

// === Graceful Degradation on Delete Error ===

func TestCheckpointCleanupHook_Cleanup_ContinuesOnDeleteError(t *testing.T) {
	fixedNow := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	checkpointer := newMockCheckpointerForCleanup()
	checkpointer.checkpoints = []CheckpointInfo{
		{BranchName: "conductor-checkpoint-task-1-20260101-120000", CreatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}, // Will fail
		{BranchName: "conductor-checkpoint-task-2-20260102-120000", CreatedAt: time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)}, // Will succeed
		{BranchName: "conductor-checkpoint-task-3-20260103-120000", CreatedAt: time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC)}, // Will succeed
	}
	checkpointer.deleteErrors["conductor-checkpoint-task-1-20260101-120000"] = errors.New("permission denied")

	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}
	logger := newMockCleanupLogger()
	hook := NewCheckpointCleanupHook(checkpointer, cfg, logger)
	hook.now = func() time.Time { return fixedNow }

	deleted, err := hook.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Expected no error (graceful degradation), got %v", err)
	}

	// Should delete 2 out of 3 (one failed)
	if deleted != 2 {
		t.Errorf("Expected 2 deleted, got %d", deleted)
	}

	// Verify warning was logged for failed deletion
	if len(logger.warnings) == 0 {
		t.Error("Expected warning for failed deletion")
	}
}

// === Enabled Tests ===

func TestCheckpointCleanupHook_Enabled_True(t *testing.T) {
	checkpointer := newMockCheckpointerForCleanup()
	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}

	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)
	if !hook.Enabled() {
		t.Error("Expected hook to be enabled")
	}
}

func TestCheckpointCleanupHook_Enabled_FalseWhenDisabled(t *testing.T) {
	checkpointer := newMockCheckpointerForCleanup()
	cfg := &config.RollbackConfig{Enabled: false, KeepCheckpointDays: 7}

	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)
	if hook.Enabled() {
		t.Error("Expected hook to be disabled")
	}
}

func TestCheckpointCleanupHook_Enabled_FalseWhenZeroKeepDays(t *testing.T) {
	checkpointer := newMockCheckpointerForCleanup()
	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 0}

	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)
	if hook.Enabled() {
		t.Error("Expected hook to be disabled with zero keep days")
	}
}

// === Context Cancellation Test ===

func TestCheckpointCleanupHook_Cleanup_ContextCancelled(t *testing.T) {
	checkpointer := newMockCheckpointerForCleanup()
	checkpointer.listError = context.Canceled
	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}

	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := hook.Cleanup(ctx)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

// === Different Keep Days Configuration ===

func TestCheckpointCleanupHook_Cleanup_CustomKeepDays(t *testing.T) {
	fixedNow := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	checkpointer := newMockCheckpointerForCleanup()
	checkpointer.checkpoints = []CheckpointInfo{
		{BranchName: "conductor-checkpoint-task-1-20260108-120000", CreatedAt: time.Date(2026, 1, 8, 12, 0, 0, 0, time.UTC)}, // 2 days old
		{BranchName: "conductor-checkpoint-task-2-20260107-120000", CreatedAt: time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)}, // 3 days old
	}

	// Only keep for 2 days
	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 2}
	hook := NewCheckpointCleanupHook(checkpointer, cfg, nil)
	hook.now = func() time.Time { return fixedNow }

	deleted, err := hook.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Both should be stale (2 and 3 days old, cutoff is 2 days)
	if deleted != 2 {
		t.Errorf("Expected 2 deleted with 2-day retention, got %d", deleted)
	}
}

// === Integration Style Test ===

func TestCheckpointCleanupHook_FullWorkflow(t *testing.T) {
	// Simulate a real workflow with mixed checkpoints
	fixedNow := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

	checkpointer := newMockCheckpointerForCleanup()
	checkpointer.checkpoints = []CheckpointInfo{
		// Fresh checkpoints (should keep)
		{BranchName: "conductor-checkpoint-task-10-20260114-090000", CreatedAt: time.Date(2026, 1, 14, 9, 0, 0, 0, time.UTC)},  // 1 day old
		{BranchName: "conductor-checkpoint-task-11-20260113-150000", CreatedAt: time.Date(2026, 1, 13, 15, 0, 0, 0, time.UTC)}, // ~2 days old

		// Stale checkpoints (should delete)
		{BranchName: "conductor-checkpoint-task-5-20260105-120000", CreatedAt: time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)},   // 10 days old
		{BranchName: "conductor-checkpoint-task-3-20260101-080000", CreatedAt: time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)},    // 14 days old
		{BranchName: "conductor-checkpoint-task-1-20251230-230000", CreatedAt: time.Date(2025, 12, 30, 23, 0, 0, 0, time.UTC)}, // 16 days old

		// Malformed (should skip with warning)
		{BranchName: "conductor-checkpoint-task-99-invalid", CreatedAt: time.Time{}},
	}

	cfg := &config.RollbackConfig{Enabled: true, KeepCheckpointDays: 7}
	logger := newMockCleanupLogger()
	hook := NewCheckpointCleanupHook(checkpointer, cfg, logger)
	hook.now = func() time.Time { return fixedNow }

	deleted, err := hook.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should delete 3 stale, skip 1 malformed, keep 2 fresh
	if deleted != 3 {
		t.Errorf("Expected 3 deleted, got %d", deleted)
	}

	// Verify info logs were generated
	if len(logger.infos) < 2 {
		t.Errorf("Expected at least 2 info logs, got %d", len(logger.infos))
	}

	// Verify warning for malformed
	if len(logger.warnings) != 1 {
		t.Errorf("Expected 1 warning for malformed timestamp, got %d", len(logger.warnings))
	}
}
