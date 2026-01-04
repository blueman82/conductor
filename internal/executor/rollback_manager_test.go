package executor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/models"
)

// MockRollbackLogger implements RuntimeEnforcementLogger for rollback testing.
type MockRollbackLogger struct {
	InfoMessages []string
	WarnMessages []string
}

func NewMockRollbackLogger() *MockRollbackLogger {
	return &MockRollbackLogger{
		InfoMessages: []string{},
		WarnMessages: []string{},
	}
}

func (m *MockRollbackLogger) LogTestCommands(entries []models.TestCommandResult)                     {}
func (m *MockRollbackLogger) LogCriterionVerifications(entries []models.CriterionVerificationResult) {}
func (m *MockRollbackLogger) LogDocTargetVerifications(entries []models.DocTargetResult)             {}
func (m *MockRollbackLogger) LogErrorPattern(pattern interface{})                                    {}
func (m *MockRollbackLogger) LogDetectedError(detected interface{})                                  {}

func (m *MockRollbackLogger) Warnf(format string, args ...interface{}) {
	m.WarnMessages = append(m.WarnMessages, format)
}

func (m *MockRollbackLogger) Info(message string) {
	m.InfoMessages = append(m.InfoMessages, message)
}

func (m *MockRollbackLogger) Infof(format string, args ...interface{}) {
	m.InfoMessages = append(m.InfoMessages, format)
}

// MockRollbackCheckpointer implements GitCheckpointer for rollback testing.
type MockRollbackCheckpointer struct {
	RestoreError     error
	RestoreCalls     []string // Stores commit hashes passed to RestoreCheckpoint
	CreateCalls      int
	DeleteCalls      []string
	CreateBranchErr  error
	SwitchBranchErr  error
	CurrentBranch    string
	CleanState       bool
	CleanStateErr    error
}

func NewMockRollbackCheckpointer() *MockRollbackCheckpointer {
	return &MockRollbackCheckpointer{
		RestoreCalls:  []string{},
		DeleteCalls:   []string{},
		CurrentBranch: "main",
		CleanState:    true,
	}
}

func (m *MockRollbackCheckpointer) CreateCheckpoint(ctx context.Context, taskNumber int) (*CheckpointInfo, error) {
	m.CreateCalls++
	return &CheckpointInfo{
		BranchName: "conductor-checkpoint-task-" + string(rune('0'+taskNumber)),
		CommitHash: "abc123",
		CreatedAt:  time.Now(),
	}, nil
}

func (m *MockRollbackCheckpointer) RestoreCheckpoint(ctx context.Context, commitHash string) error {
	m.RestoreCalls = append(m.RestoreCalls, commitHash)
	return m.RestoreError
}

func (m *MockRollbackCheckpointer) DeleteCheckpoint(ctx context.Context, branchName string) error {
	m.DeleteCalls = append(m.DeleteCalls, branchName)
	return nil
}

func (m *MockRollbackCheckpointer) CreateBranch(ctx context.Context, branchName string) error {
	return m.CreateBranchErr
}

func (m *MockRollbackCheckpointer) SwitchBranch(ctx context.Context, branchName string) error {
	return m.SwitchBranchErr
}

func (m *MockRollbackCheckpointer) GetCurrentBranch(ctx context.Context) (string, error) {
	return m.CurrentBranch, nil
}

func (m *MockRollbackCheckpointer) IsCleanState(ctx context.Context) (bool, error) {
	return m.CleanState, m.CleanStateErr
}

// === Constructor Tests ===

func TestNewRollbackManager(t *testing.T) {
	cfg := config.DefaultRollbackConfig()
	checkpointer := NewMockRollbackCheckpointer()
	logger := NewMockRollbackLogger()

	m := NewRollbackManager(&cfg, checkpointer, logger)
	if m == nil {
		t.Fatal("Expected non-nil manager")
	}
	if m.Config == nil {
		t.Error("Expected Config to be set")
	}
	if m.Checkpointer == nil {
		t.Error("Expected Checkpointer to be set")
	}
	if m.Logger == nil {
		t.Error("Expected Logger to be set")
	}
}

func TestNewRollbackManager_NilConfig(t *testing.T) {
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(nil, checkpointer, nil)
	if m != nil {
		t.Error("Expected nil manager when config is nil")
	}
}

func TestNewRollbackManager_NilCheckpointer(t *testing.T) {
	cfg := config.DefaultRollbackConfig()
	m := NewRollbackManager(&cfg, nil, nil)
	if m != nil {
		t.Error("Expected nil manager when checkpointer is nil")
	}
}

func TestNewRollbackManager_NilLogger(t *testing.T) {
	cfg := config.DefaultRollbackConfig()
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)
	if m == nil {
		t.Fatal("Expected non-nil manager even with nil logger")
	}
	if m.Logger != nil {
		t.Error("Expected nil logger")
	}
}

// === ShouldRollback Tests - Manual Mode ===

func TestShouldRollback_ManualMode_NeverRollbacks(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: true,
		Mode:    config.RollbackModeManual,
	}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)

	// Test all verdict combinations - manual mode should never rollback
	testCases := []struct {
		verdict    string
		attempt    int
		maxRetries int
	}{
		{"RED", 1, 2},
		{"RED", 2, 2},
		{"RED", 3, 2},
		{"GREEN", 1, 2},
		{"YELLOW", 1, 2},
	}

	for _, tc := range testCases {
		result := m.ShouldRollback(tc.verdict, tc.attempt, tc.maxRetries)
		if result {
			t.Errorf("Manual mode should never rollback, got true for verdict=%s, attempt=%d, maxRetries=%d",
				tc.verdict, tc.attempt, tc.maxRetries)
		}
	}
}

// === ShouldRollback Tests - Auto On Red Mode ===

func TestShouldRollback_AutoOnRed_RollbacksOnRed(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: true,
		Mode:    config.RollbackModeAutoOnRed,
	}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)

	// Should rollback on RED regardless of attempt number
	if !m.ShouldRollback("RED", 1, 2) {
		t.Error("Expected rollback on RED verdict at attempt 1")
	}
	if !m.ShouldRollback("RED", 2, 2) {
		t.Error("Expected rollback on RED verdict at attempt 2")
	}
	if !m.ShouldRollback("RED", 5, 2) {
		t.Error("Expected rollback on RED verdict at attempt 5")
	}
}

func TestShouldRollback_AutoOnRed_NoRollbackOnGreen(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: true,
		Mode:    config.RollbackModeAutoOnRed,
	}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)

	if m.ShouldRollback("GREEN", 1, 2) {
		t.Error("Should not rollback on GREEN verdict")
	}
	if m.ShouldRollback("GREEN", 5, 2) {
		t.Error("Should not rollback on GREEN verdict even at high attempt")
	}
}

func TestShouldRollback_AutoOnRed_NoRollbackOnYellow(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: true,
		Mode:    config.RollbackModeAutoOnRed,
	}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)

	if m.ShouldRollback("YELLOW", 1, 2) {
		t.Error("Should not rollback on YELLOW verdict")
	}
}

// === ShouldRollback Tests - Auto On Max Retries Mode ===

func TestShouldRollback_AutoOnMaxRetries_NoRollbackBeforeMax(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: true,
		Mode:    config.RollbackModeAutoOnMaxRetries,
	}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)

	// maxRetries=2 means attempts 1, 2, 3 are allowed (1 initial + 2 retries)
	// Should not rollback on attempts 1 and 2
	if m.ShouldRollback("RED", 1, 2) {
		t.Error("Should not rollback on first attempt")
	}
	if m.ShouldRollback("RED", 2, 2) {
		t.Error("Should not rollback on second attempt (first retry)")
	}
}

func TestShouldRollback_AutoOnMaxRetries_RollbackAtMax(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: true,
		Mode:    config.RollbackModeAutoOnMaxRetries,
	}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)

	// maxRetries=2 means attempts 1, 2, 3 are allowed
	// Should rollback when attempt > maxRetries (i.e., attempt 3 > 2)
	if !m.ShouldRollback("RED", 3, 2) {
		t.Error("Should rollback when attempt (3) > maxRetries (2)")
	}
	if !m.ShouldRollback("RED", 4, 2) {
		t.Error("Should rollback when attempt exceeds maxRetries")
	}
}

func TestShouldRollback_AutoOnMaxRetries_NoRollbackOnGreen(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: true,
		Mode:    config.RollbackModeAutoOnMaxRetries,
	}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)

	// Should never rollback on GREEN even if max retries exceeded
	if m.ShouldRollback("GREEN", 5, 2) {
		t.Error("Should not rollback on GREEN verdict")
	}
}

func TestShouldRollback_AutoOnMaxRetries_ZeroRetries(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: true,
		Mode:    config.RollbackModeAutoOnMaxRetries,
	}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)

	// maxRetries=0 means only 1 attempt allowed
	// Should rollback on second attempt (attempt > 0)
	if !m.ShouldRollback("RED", 1, 0) {
		t.Error("Should rollback when attempt (1) > maxRetries (0)")
	}
}

// === ShouldRollback Tests - Disabled Feature ===

func TestShouldRollback_Disabled_NeverRollbacks(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: false, // Feature disabled
		Mode:    config.RollbackModeAutoOnRed,
	}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)

	if m.ShouldRollback("RED", 1, 2) {
		t.Error("Should not rollback when feature is disabled")
	}
}

// === ShouldRollback Tests - Nil Safety ===

func TestShouldRollback_NilManager(t *testing.T) {
	var m *RollbackManager
	if m.ShouldRollback("RED", 1, 2) {
		t.Error("Nil manager should return false")
	}
}

func TestShouldRollback_NilConfig(t *testing.T) {
	m := &RollbackManager{
		Config: nil,
	}
	if m.ShouldRollback("RED", 1, 2) {
		t.Error("Manager with nil config should return false")
	}
}

// === PerformRollback Tests ===

func TestPerformRollback_Success(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: true,
		Mode:    config.RollbackModeAutoOnRed,
	}
	checkpointer := NewMockRollbackCheckpointer()
	logger := NewMockRollbackLogger()
	m := NewRollbackManager(&cfg, checkpointer, logger)

	checkpoint := &CheckpointInfo{
		BranchName: "conductor-checkpoint-task-1",
		CommitHash: "abc123def",
		CreatedAt:  time.Now(),
	}

	err := m.PerformRollback(context.Background(), checkpoint)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify RestoreCheckpoint was called with correct hash
	if len(checkpointer.RestoreCalls) != 1 {
		t.Errorf("Expected 1 restore call, got %d", len(checkpointer.RestoreCalls))
	}
	if checkpointer.RestoreCalls[0] != "abc123def" {
		t.Errorf("Expected commit hash 'abc123def', got %q", checkpointer.RestoreCalls[0])
	}

	// Verify logging
	if len(logger.InfoMessages) < 2 {
		t.Errorf("Expected at least 2 info messages, got %d", len(logger.InfoMessages))
	}
}

func TestPerformRollback_Error(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: true,
		Mode:    config.RollbackModeAutoOnRed,
	}
	checkpointer := NewMockRollbackCheckpointer()
	checkpointer.RestoreError = errors.New("git reset failed")
	logger := NewMockRollbackLogger()
	m := NewRollbackManager(&cfg, checkpointer, logger)

	checkpoint := &CheckpointInfo{
		BranchName: "conductor-checkpoint-task-1",
		CommitHash: "abc123def",
		CreatedAt:  time.Now(),
	}

	err := m.PerformRollback(context.Background(), checkpoint)
	if err == nil {
		t.Fatal("Expected error from restore failure")
	}
	if !strings.Contains(err.Error(), "failed to restore checkpoint") {
		t.Errorf("Expected 'failed to restore checkpoint' in error, got %q", err.Error())
	}

	// Verify warning was logged
	if len(logger.WarnMessages) == 0 {
		t.Error("Expected warning message on failure")
	}
}

func TestPerformRollback_NilCheckpoint(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: true,
	}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)

	err := m.PerformRollback(context.Background(), nil)
	if err == nil {
		t.Error("Expected error for nil checkpoint")
	}
	if !strings.Contains(err.Error(), "checkpoint cannot be nil") {
		t.Errorf("Expected 'checkpoint cannot be nil' in error, got %q", err.Error())
	}
}

func TestPerformRollback_EmptyCommitHash(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: true,
	}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)

	checkpoint := &CheckpointInfo{
		BranchName: "conductor-checkpoint",
		CommitHash: "",
	}

	err := m.PerformRollback(context.Background(), checkpoint)
	if err == nil {
		t.Error("Expected error for empty commit hash")
	}
	if !strings.Contains(err.Error(), "checkpoint commit hash cannot be empty") {
		t.Errorf("Expected 'checkpoint commit hash cannot be empty' in error, got %q", err.Error())
	}
}

func TestPerformRollback_NilManager(t *testing.T) {
	var m *RollbackManager
	checkpoint := &CheckpointInfo{
		CommitHash: "abc123",
	}
	err := m.PerformRollback(context.Background(), checkpoint)
	if err == nil {
		t.Error("Expected error for nil manager")
	}
}

func TestPerformRollback_NilCheckpointer(t *testing.T) {
	m := &RollbackManager{
		Config:       &config.RollbackConfig{Enabled: true},
		Checkpointer: nil,
	}
	checkpoint := &CheckpointInfo{
		CommitHash: "abc123",
	}
	err := m.PerformRollback(context.Background(), checkpoint)
	if err == nil {
		t.Error("Expected error for nil checkpointer")
	}
}

func TestPerformRollback_NoLogger(t *testing.T) {
	cfg := config.RollbackConfig{
		Enabled: true,
	}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil) // nil logger

	checkpoint := &CheckpointInfo{
		BranchName: "conductor-checkpoint",
		CommitHash: "abc123",
	}

	// Should work without error even with nil logger
	err := m.PerformRollback(context.Background(), checkpoint)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// === Enabled/Mode Tests ===

func TestEnabled_True(t *testing.T) {
	cfg := config.RollbackConfig{Enabled: true}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)

	if !m.Enabled() {
		t.Error("Expected Enabled() to return true")
	}
}

func TestEnabled_False(t *testing.T) {
	cfg := config.RollbackConfig{Enabled: false}
	checkpointer := NewMockRollbackCheckpointer()
	m := NewRollbackManager(&cfg, checkpointer, nil)

	if m.Enabled() {
		t.Error("Expected Enabled() to return false")
	}
}

func TestEnabled_NilManager(t *testing.T) {
	var m *RollbackManager
	if m.Enabled() {
		t.Error("Nil manager should return false for Enabled()")
	}
}

func TestMode_ReturnsConfiguredMode(t *testing.T) {
	testCases := []config.RollbackMode{
		config.RollbackModeManual,
		config.RollbackModeAutoOnRed,
		config.RollbackModeAutoOnMaxRetries,
	}

	for _, mode := range testCases {
		cfg := config.RollbackConfig{
			Enabled: true,
			Mode:    mode,
		}
		checkpointer := NewMockRollbackCheckpointer()
		m := NewRollbackManager(&cfg, checkpointer, nil)

		if m.Mode() != mode {
			t.Errorf("Expected mode %q, got %q", mode, m.Mode())
		}
	}
}

func TestMode_NilManager(t *testing.T) {
	var m *RollbackManager
	if m.Mode() != config.RollbackModeManual {
		t.Error("Nil manager should return RollbackModeManual")
	}
}

func TestMode_NilConfig(t *testing.T) {
	m := &RollbackManager{Config: nil}
	if m.Mode() != config.RollbackModeManual {
		t.Error("Manager with nil config should return RollbackModeManual")
	}
}

// === Context Handling Tests ===

func TestPerformRollback_ContextCancelled(t *testing.T) {
	cfg := config.RollbackConfig{Enabled: true}
	checkpointer := NewMockRollbackCheckpointer()
	checkpointer.RestoreError = context.Canceled
	m := NewRollbackManager(&cfg, checkpointer, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checkpoint := &CheckpointInfo{CommitHash: "abc123"}
	err := m.PerformRollback(ctx, checkpoint)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

// === Decision Matrix Integration Tests ===

func TestDecisionMatrix_Comprehensive(t *testing.T) {
	testCases := []struct {
		name       string
		mode       config.RollbackMode
		verdict    string
		attempt    int
		maxRetries int
		expected   bool
	}{
		// Manual mode - never rollbacks
		{"manual_red_first", config.RollbackModeManual, "RED", 1, 2, false},
		{"manual_red_last", config.RollbackModeManual, "RED", 3, 2, false},
		{"manual_green", config.RollbackModeManual, "GREEN", 1, 2, false},

		// Auto on red - rollbacks on any RED
		{"auto_red_first", config.RollbackModeAutoOnRed, "RED", 1, 2, true},
		{"auto_red_middle", config.RollbackModeAutoOnRed, "RED", 2, 2, true},
		{"auto_red_last", config.RollbackModeAutoOnRed, "RED", 3, 2, true},
		{"auto_green", config.RollbackModeAutoOnRed, "GREEN", 1, 2, false},
		{"auto_yellow", config.RollbackModeAutoOnRed, "YELLOW", 1, 2, false},

		// Auto on max retries - rollbacks only when RED and max exceeded
		{"max_red_first", config.RollbackModeAutoOnMaxRetries, "RED", 1, 2, false},
		{"max_red_middle", config.RollbackModeAutoOnMaxRetries, "RED", 2, 2, false},
		{"max_red_at_max", config.RollbackModeAutoOnMaxRetries, "RED", 3, 2, true},
		{"max_red_over_max", config.RollbackModeAutoOnMaxRetries, "RED", 4, 2, true},
		{"max_green_over_max", config.RollbackModeAutoOnMaxRetries, "GREEN", 4, 2, false},
		{"max_yellow_over_max", config.RollbackModeAutoOnMaxRetries, "YELLOW", 4, 2, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.RollbackConfig{
				Enabled: true,
				Mode:    tc.mode,
			}
			checkpointer := NewMockRollbackCheckpointer()
			m := NewRollbackManager(&cfg, checkpointer, nil)

			result := m.ShouldRollback(tc.verdict, tc.attempt, tc.maxRetries)
			if result != tc.expected {
				t.Errorf("ShouldRollback(%q, %d, %d) with mode %q = %v, want %v",
					tc.verdict, tc.attempt, tc.maxRetries, tc.mode, result, tc.expected)
			}
		})
	}
}
