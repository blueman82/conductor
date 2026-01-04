package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/config"
)

// mockGitCheckpointer implements GitCheckpointer for testing.
type mockGitCheckpointer struct {
	createCheckpointFunc   func(ctx context.Context, taskNumber int) (*CheckpointInfo, error)
	restoreCheckpointFunc  func(ctx context.Context, commitHash string) error
	deleteCheckpointFunc   func(ctx context.Context, branchName string) error
	createBranchFunc       func(ctx context.Context, branchName string) error
	switchBranchFunc       func(ctx context.Context, branchName string) error
	getCurrentBranchFunc   func(ctx context.Context) (string, error)
	isCleanStateFunc       func(ctx context.Context) (bool, error)
	createCheckpointCalled int
	deleteCheckpointCalled int
	restoreCalled          int
}

func (m *mockGitCheckpointer) CreateCheckpoint(ctx context.Context, taskNumber int) (*CheckpointInfo, error) {
	m.createCheckpointCalled++
	if m.createCheckpointFunc != nil {
		return m.createCheckpointFunc(ctx, taskNumber)
	}
	return &CheckpointInfo{
		BranchName: "conductor-checkpoint-task-1-20240101-120000",
		CommitHash: "abc123",
		CreatedAt:  time.Now(),
	}, nil
}

func (m *mockGitCheckpointer) RestoreCheckpoint(ctx context.Context, commitHash string) error {
	m.restoreCalled++
	if m.restoreCheckpointFunc != nil {
		return m.restoreCheckpointFunc(ctx, commitHash)
	}
	return nil
}

func (m *mockGitCheckpointer) DeleteCheckpoint(ctx context.Context, branchName string) error {
	m.deleteCheckpointCalled++
	if m.deleteCheckpointFunc != nil {
		return m.deleteCheckpointFunc(ctx, branchName)
	}
	return nil
}

func (m *mockGitCheckpointer) CreateBranch(ctx context.Context, branchName string) error {
	if m.createBranchFunc != nil {
		return m.createBranchFunc(ctx, branchName)
	}
	return nil
}

func (m *mockGitCheckpointer) SwitchBranch(ctx context.Context, branchName string) error {
	if m.switchBranchFunc != nil {
		return m.switchBranchFunc(ctx, branchName)
	}
	return nil
}

func (m *mockGitCheckpointer) GetCurrentBranch(ctx context.Context) (string, error) {
	if m.getCurrentBranchFunc != nil {
		return m.getCurrentBranchFunc(ctx)
	}
	return "feature-branch", nil
}

func (m *mockGitCheckpointer) IsCleanState(ctx context.Context) (bool, error) {
	if m.isCleanStateFunc != nil {
		return m.isCleanStateFunc(ctx)
	}
	return true, nil
}

// rollbackMockLogger implements RuntimeEnforcementLogger for testing.
type rollbackMockLogger struct {
	infos  []string
	warns  []string
	errors []string
}

func (m *rollbackMockLogger) LogTestCommands(entries []TestCommandResult)                  {}
func (m *rollbackMockLogger) LogCriterionVerifications(entries []CriterionVerificationResult) {}
func (m *rollbackMockLogger) LogDocTargetVerifications(entries []DocTargetResult)          {}
func (m *rollbackMockLogger) LogErrorPattern(pattern interface{})                          {}
func (m *rollbackMockLogger) LogDetectedError(detected interface{})                        {}
func (m *rollbackMockLogger) Warnf(format string, args ...interface{})                     { m.warns = append(m.warns, format) }
func (m *rollbackMockLogger) Info(message string)                                          { m.infos = append(m.infos, message) }
func (m *rollbackMockLogger) Infof(format string, args ...interface{})                     { m.infos = append(m.infos, format) }

func TestNewRollbackHook(t *testing.T) {
	tests := []struct {
		name         string
		manager      *RollbackManager
		checkpointer GitCheckpointer
		wantNil      bool
	}{
		{
			name:         "nil manager returns nil hook",
			manager:      nil,
			checkpointer: &mockGitCheckpointer{},
			wantNil:      true,
		},
		{
			name: "nil checkpointer returns nil hook",
			manager: &RollbackManager{
				Config: &config.RollbackConfig{Enabled: true},
			},
			checkpointer: nil,
			wantNil:      true,
		},
		{
			name: "valid manager and checkpointer returns hook",
			manager: &RollbackManager{
				Config: &config.RollbackConfig{Enabled: true},
			},
			checkpointer: &mockGitCheckpointer{},
			wantNil:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &rollbackMockLogger{}
			hook := NewRollbackHook(tt.manager, tt.checkpointer, logger)
			if tt.wantNil && hook != nil {
				t.Errorf("expected nil hook, got %v", hook)
			}
			if !tt.wantNil && hook == nil {
				t.Error("expected non-nil hook")
			}
		})
	}
}

func TestRollbackHook_PreTask(t *testing.T) {
	tests := []struct {
		name                string
		enabled             bool
		createErr           error
		wantCheckpointCalls int
		wantMetadata        bool
	}{
		{
			name:                "creates checkpoint when enabled",
			enabled:             true,
			createErr:           nil,
			wantCheckpointCalls: 1,
			wantMetadata:        true,
		},
		{
			name:                "skips when disabled",
			enabled:             false,
			createErr:           nil,
			wantCheckpointCalls: 0,
			wantMetadata:        false,
		},
		{
			name:                "handles checkpoint error gracefully",
			enabled:             true,
			createErr:           errors.New("git error"),
			wantCheckpointCalls: 1,
			wantMetadata:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkpointer := &mockGitCheckpointer{
				createCheckpointFunc: func(ctx context.Context, taskNumber int) (*CheckpointInfo, error) {
					if tt.createErr != nil {
						return nil, tt.createErr
					}
					return &CheckpointInfo{
						BranchName: "checkpoint-test",
						CommitHash: "abc123",
						CreatedAt:  time.Now(),
					}, nil
				},
			}

			manager := &RollbackManager{
				Config:       &config.RollbackConfig{Enabled: tt.enabled},
				Checkpointer: checkpointer,
			}

			hook := NewRollbackHook(manager, checkpointer, &rollbackMockLogger{})
			metadata := make(map[string]interface{})

			err := hook.PreTask(context.Background(), 1, metadata)
			if err != nil {
				t.Errorf("PreTask returned error: %v", err)
			}

			if checkpointer.createCheckpointCalled != tt.wantCheckpointCalls {
				t.Errorf("expected %d checkpoint calls, got %d",
					tt.wantCheckpointCalls, checkpointer.createCheckpointCalled)
			}

			_, hasCheckpoint := metadata["rollback_checkpoint"]
			if tt.wantMetadata && !hasCheckpoint {
				t.Error("expected checkpoint in metadata")
			}
			if !tt.wantMetadata && hasCheckpoint {
				t.Error("unexpected checkpoint in metadata")
			}
		})
	}
}

func TestRollbackHook_PreTask_NilHook(t *testing.T) {
	var hook *RollbackHook
	err := hook.PreTask(context.Background(), 1, nil)
	if err != nil {
		t.Errorf("nil hook PreTask should not error, got %v", err)
	}
}

func TestRollbackHook_PostTask(t *testing.T) {
	tests := []struct {
		name              string
		enabled           bool
		mode              config.RollbackMode
		verdict           string
		attempt           int
		maxRetries        int
		success           bool
		wantRestore       bool
		wantDeleteCalls   int
	}{
		{
			name:            "no rollback on GREEN verdict",
			enabled:         true,
			mode:            config.RollbackModeAutoOnRed,
			verdict:         "GREEN",
			attempt:         1,
			maxRetries:      2,
			success:         true,
			wantRestore:     false,
			wantDeleteCalls: 1, // Cleanup on success
		},
		{
			name:            "rollback on RED with auto_on_red mode",
			enabled:         true,
			mode:            config.RollbackModeAutoOnRed,
			verdict:         "RED",
			attempt:         1,
			maxRetries:      2,
			success:         false,
			wantRestore:     true,
			wantDeleteCalls: 1, // Cleanup after rollback
		},
		{
			name:            "no rollback on RED with manual mode",
			enabled:         true,
			mode:            config.RollbackModeManual,
			verdict:         "RED",
			attempt:         1,
			maxRetries:      2,
			success:         false,
			wantRestore:     false,
			wantDeleteCalls: 0,
		},
		{
			name:            "rollback on max retries exhausted with auto_on_max_retries mode",
			enabled:         true,
			mode:            config.RollbackModeAutoOnMaxRetries,
			verdict:         "RED",
			attempt:         3, // > maxRetries (2)
			maxRetries:      2,
			success:         false,
			wantRestore:     true,
			wantDeleteCalls: 1,
		},
		{
			name:            "no rollback before max retries with auto_on_max_retries mode",
			enabled:         true,
			mode:            config.RollbackModeAutoOnMaxRetries,
			verdict:         "RED",
			attempt:         1, // <= maxRetries
			maxRetries:      2,
			success:         false,
			wantRestore:     false,
			wantDeleteCalls: 0,
		},
		{
			name:            "skips when disabled",
			enabled:         false,
			mode:            config.RollbackModeAutoOnRed,
			verdict:         "RED",
			attempt:         1,
			maxRetries:      2,
			success:         false,
			wantRestore:     false,
			wantDeleteCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkpointer := &mockGitCheckpointer{}

			manager := &RollbackManager{
				Config: &config.RollbackConfig{
					Enabled: tt.enabled,
					Mode:    tt.mode,
				},
				Checkpointer: checkpointer,
			}

			hook := NewRollbackHook(manager, checkpointer, &rollbackMockLogger{})

			// Setup metadata with checkpoint
			metadata := map[string]interface{}{
				"rollback_checkpoint": &CheckpointInfo{
					BranchName: "checkpoint-test",
					CommitHash: "abc123",
					CreatedAt:  time.Now(),
				},
			}

			err := hook.PostTask(context.Background(), 1, metadata, tt.verdict, tt.attempt, tt.maxRetries, tt.success)
			if err != nil {
				t.Errorf("PostTask returned error: %v", err)
			}

			if tt.wantRestore && checkpointer.restoreCalled == 0 {
				t.Error("expected restore to be called")
			}
			if !tt.wantRestore && checkpointer.restoreCalled > 0 {
				t.Error("unexpected restore call")
			}

			if checkpointer.deleteCheckpointCalled != tt.wantDeleteCalls {
				t.Errorf("expected %d delete calls, got %d",
					tt.wantDeleteCalls, checkpointer.deleteCheckpointCalled)
			}
		})
	}
}

func TestRollbackHook_PostTask_NoCheckpoint(t *testing.T) {
	checkpointer := &mockGitCheckpointer{}
	manager := &RollbackManager{
		Config:       &config.RollbackConfig{Enabled: true, Mode: config.RollbackModeAutoOnRed},
		Checkpointer: checkpointer,
	}

	hook := NewRollbackHook(manager, checkpointer, &rollbackMockLogger{})

	// Empty metadata - no checkpoint
	metadata := make(map[string]interface{})

	err := hook.PostTask(context.Background(), 1, metadata, "RED", 1, 2, false)
	if err != nil {
		t.Errorf("PostTask without checkpoint should not error, got %v", err)
	}

	// Should not attempt restore without checkpoint
	if checkpointer.restoreCalled > 0 {
		t.Error("should not restore without checkpoint")
	}
}

func TestRollbackHook_PostTask_NilHook(t *testing.T) {
	var hook *RollbackHook
	err := hook.PostTask(context.Background(), 1, nil, "RED", 1, 2, false)
	if err != nil {
		t.Errorf("nil hook PostTask should not error, got %v", err)
	}
}

func TestRollbackHook_Enabled(t *testing.T) {
	tests := []struct {
		name    string
		hook    *RollbackHook
		want    bool
	}{
		{
			name: "nil hook returns false",
			hook: nil,
			want: false,
		},
		{
			name: "nil manager returns false",
			hook: &RollbackHook{
				Manager: nil,
			},
			want: false,
		},
		{
			name: "disabled config returns false",
			hook: &RollbackHook{
				Manager: &RollbackManager{
					Config: &config.RollbackConfig{Enabled: false},
				},
			},
			want: false,
		},
		{
			name: "enabled config returns true",
			hook: &RollbackHook{
				Manager: &RollbackManager{
					Config: &config.RollbackConfig{Enabled: true},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.hook.Enabled()
			if got != tt.want {
				t.Errorf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRollbackHook_PostTask_RestoreError(t *testing.T) {
	checkpointer := &mockGitCheckpointer{
		restoreCheckpointFunc: func(ctx context.Context, commitHash string) error {
			return errors.New("restore failed")
		},
	}

	manager := &RollbackManager{
		Config:       &config.RollbackConfig{Enabled: true, Mode: config.RollbackModeAutoOnRed},
		Checkpointer: checkpointer,
		Logger:       &rollbackMockLogger{},
	}

	hook := NewRollbackHook(manager, checkpointer, &rollbackMockLogger{})

	metadata := map[string]interface{}{
		"rollback_checkpoint": &CheckpointInfo{
			BranchName: "checkpoint-test",
			CommitHash: "abc123",
			CreatedAt:  time.Now(),
		},
	}

	// Should not return error - graceful degradation
	err := hook.PostTask(context.Background(), 1, metadata, "RED", 1, 2, false)
	if err != nil {
		t.Errorf("PostTask with restore error should not fail, got %v", err)
	}
}

func TestRollbackHook_PostTask_DeleteError(t *testing.T) {
	checkpointer := &mockGitCheckpointer{
		deleteCheckpointFunc: func(ctx context.Context, branchName string) error {
			return errors.New("delete failed")
		},
	}

	manager := &RollbackManager{
		Config:       &config.RollbackConfig{Enabled: true, Mode: config.RollbackModeAutoOnRed},
		Checkpointer: checkpointer,
	}

	hook := NewRollbackHook(manager, checkpointer, &rollbackMockLogger{})

	metadata := map[string]interface{}{
		"rollback_checkpoint": &CheckpointInfo{
			BranchName: "checkpoint-test",
			CommitHash: "abc123",
			CreatedAt:  time.Now(),
		},
	}

	// Should not return error - graceful degradation (cleanup failure is non-fatal)
	err := hook.PostTask(context.Background(), 1, metadata, "GREEN", 1, 2, true)
	if err != nil {
		t.Errorf("PostTask with delete error should not fail, got %v", err)
	}
}

func TestRollbackHook_IntegrationScenario(t *testing.T) {
	// Simulate full task lifecycle with checkpoint/rollback

	checkpointer := &mockGitCheckpointer{}
	manager := &RollbackManager{
		Config: &config.RollbackConfig{
			Enabled:          true,
			Mode:             config.RollbackModeAutoOnMaxRetries,
			CheckpointPrefix: "conductor-checkpoint-",
		},
		Checkpointer: checkpointer,
	}

	hook := NewRollbackHook(manager, checkpointer, &rollbackMockLogger{})
	ctx := context.Background()

	// Simulate task metadata
	metadata := make(map[string]interface{})

	// PreTask: Create checkpoint
	if err := hook.PreTask(ctx, 1, metadata); err != nil {
		t.Fatalf("PreTask failed: %v", err)
	}

	// Verify checkpoint was created and stored
	if checkpointer.createCheckpointCalled != 1 {
		t.Error("expected checkpoint to be created in PreTask")
	}
	if _, ok := metadata["rollback_checkpoint"]; !ok {
		t.Error("expected checkpoint to be stored in metadata")
	}

	// Simulate task failure with max retries exhausted
	// attempt=3 > maxRetries=2 triggers rollback
	if err := hook.PostTask(ctx, 1, metadata, "RED", 3, 2, false); err != nil {
		t.Fatalf("PostTask failed: %v", err)
	}

	// Verify rollback was performed
	if checkpointer.restoreCalled != 1 {
		t.Error("expected restore to be called for rollback")
	}

	// Verify checkpoint was cleaned up after rollback
	if checkpointer.deleteCheckpointCalled != 1 {
		t.Error("expected checkpoint to be deleted after rollback")
	}
}

func TestRollbackHook_SuccessfulTaskCleanup(t *testing.T) {
	// Verify checkpoint is cleaned up on successful task completion

	checkpointer := &mockGitCheckpointer{}
	manager := &RollbackManager{
		Config: &config.RollbackConfig{
			Enabled: true,
			Mode:    config.RollbackModeAutoOnRed,
		},
		Checkpointer: checkpointer,
	}

	hook := NewRollbackHook(manager, checkpointer, &rollbackMockLogger{})
	ctx := context.Background()
	metadata := make(map[string]interface{})

	// PreTask: Create checkpoint
	hook.PreTask(ctx, 1, metadata)

	// PostTask with GREEN verdict and success=true
	hook.PostTask(ctx, 1, metadata, "GREEN", 1, 2, true)

	// Should NOT restore (no rollback needed)
	if checkpointer.restoreCalled != 0 {
		t.Error("should not restore on success")
	}

	// Should cleanup checkpoint
	if checkpointer.deleteCheckpointCalled != 1 {
		t.Error("expected checkpoint cleanup on success")
	}
}
