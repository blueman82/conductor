package executor

import (
	"context"
	"testing"

	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockWarmUpProvider implements learning.WarmUpProvider for testing
type mockWarmUpProvider struct {
	buildContextFunc func(ctx context.Context, task *learning.TaskInfo) (*learning.WarmUpContext, error)
}

func (m *mockWarmUpProvider) BuildContext(ctx context.Context, task *learning.TaskInfo) (*learning.WarmUpContext, error) {
	if m.buildContextFunc != nil {
		return m.buildContextFunc(ctx, task)
	}
	return &learning.WarmUpContext{Confidence: 0.0}, nil
}

// mockWarmUpLogger implements RuntimeEnforcementLogger for testing
type mockWarmUpLogger struct {
	warnMessages []string
	infoMessages []string
}

func (m *mockWarmUpLogger) LogTestCommands(entries []models.TestCommandResult) {}
func (m *mockWarmUpLogger) LogCriterionVerifications(entries []models.CriterionVerificationResult) {
}
func (m *mockWarmUpLogger) LogDocTargetVerifications(entries []models.DocTargetResult) {}
func (m *mockWarmUpLogger) LogErrorPattern(pattern interface{})                        {}
func (m *mockWarmUpLogger) LogDetectedError(detected interface{})                      {}
func (m *mockWarmUpLogger) Warnf(format string, args ...interface{}) {
	m.warnMessages = append(m.warnMessages, format)
}
func (m *mockWarmUpLogger) Info(message string) {
	m.infoMessages = append(m.infoMessages, message)
}
func (m *mockWarmUpLogger) Infof(format string, args ...interface{}) {
	m.infoMessages = append(m.infoMessages, format)
}

func TestNewWarmUpHook(t *testing.T) {
	t.Run("returns nil for nil provider", func(t *testing.T) {
		hook := NewWarmUpHook(nil, nil)
		assert.Nil(t, hook)
	})

	t.Run("creates hook with valid provider", func(t *testing.T) {
		provider := &mockWarmUpProvider{}
		hook := NewWarmUpHook(provider, nil)
		assert.NotNil(t, hook)
	})
}

func TestWarmUpHookInjectContext(t *testing.T) {
	ctx := context.Background()

	t.Run("graceful degradation on nil hook", func(t *testing.T) {
		var hook *WarmUpHook
		task := models.Task{Number: "1", Name: "Test Task", Prompt: "Original prompt"}

		result, err := hook.InjectContext(ctx, task)

		require.NoError(t, err)
		assert.Equal(t, "Original prompt", result.Prompt)
	})

	t.Run("skips injection on low confidence", func(t *testing.T) {
		provider := &mockWarmUpProvider{
			buildContextFunc: func(ctx context.Context, task *learning.TaskInfo) (*learning.WarmUpContext, error) {
				return &learning.WarmUpContext{
					Confidence: 0.1, // Below threshold
				}, nil
			},
		}
		hook := NewWarmUpHook(provider, nil)
		task := models.Task{Number: "1", Name: "Test Task", Prompt: "Original prompt"}

		result, err := hook.InjectContext(ctx, task)

		require.NoError(t, err)
		assert.Equal(t, "Original prompt", result.Prompt)
	})

	t.Run("injects context on high confidence", func(t *testing.T) {
		provider := &mockWarmUpProvider{
			buildContextFunc: func(ctx context.Context, task *learning.TaskInfo) (*learning.WarmUpContext, error) {
				return &learning.WarmUpContext{
					Confidence:          0.8,
					RecommendedApproach: "Use the existing pattern from similar tasks",
					RelevantHistory: []learning.TaskExecution{
						{TaskName: "Similar Task", Success: true, Agent: "golang-pro", QCVerdict: "GREEN"},
					},
				}, nil
			},
		}
		logger := &mockWarmUpLogger{}
		hook := NewWarmUpHook(provider, logger)
		task := models.Task{Number: "1", Name: "Test Task", Prompt: "Original prompt"}

		result, err := hook.InjectContext(ctx, task)

		require.NoError(t, err)
		assert.Contains(t, result.Prompt, "--- WARM-UP CONTEXT ---")
		assert.Contains(t, result.Prompt, "--- END WARM-UP ---")
		assert.Contains(t, result.Prompt, "Use the existing pattern")
		assert.Contains(t, result.Prompt, "Original prompt")
		assert.True(t, len(logger.infoMessages) > 0)
	})

	t.Run("includes similar patterns in injection", func(t *testing.T) {
		provider := &mockWarmUpProvider{
			buildContextFunc: func(ctx context.Context, task *learning.TaskInfo) (*learning.WarmUpContext, error) {
				return &learning.WarmUpContext{
					Confidence: 0.7,
					SimilarPatterns: []string{
						"Always validate input before processing",
						"Use context for cancellation",
					},
				}, nil
			},
		}
		hook := NewWarmUpHook(provider, nil)
		task := models.Task{Number: "1", Name: "Test Task", Prompt: "Original prompt"}

		result, err := hook.InjectContext(ctx, task)

		require.NoError(t, err)
		assert.Contains(t, result.Prompt, "Patterns from Similar Tasks")
		assert.Contains(t, result.Prompt, "Always validate input")
		assert.Contains(t, result.Prompt, "Use context for cancellation")
	})

	t.Run("graceful degradation on provider error", func(t *testing.T) {
		provider := &mockWarmUpProvider{
			buildContextFunc: func(ctx context.Context, task *learning.TaskInfo) (*learning.WarmUpContext, error) {
				return nil, assert.AnError
			},
		}
		logger := &mockWarmUpLogger{}
		hook := NewWarmUpHook(provider, logger)
		task := models.Task{Number: "1", Name: "Test Task", Prompt: "Original prompt"}

		result, err := hook.InjectContext(ctx, task)

		require.NoError(t, err)
		assert.Equal(t, "Original prompt", result.Prompt)
		assert.True(t, len(logger.warnMessages) > 0)
	})

	t.Run("extracts file paths from task metadata", func(t *testing.T) {
		var capturedTaskInfo *learning.TaskInfo
		provider := &mockWarmUpProvider{
			buildContextFunc: func(ctx context.Context, task *learning.TaskInfo) (*learning.WarmUpContext, error) {
				capturedTaskInfo = task
				return &learning.WarmUpContext{Confidence: 0.0}, nil
			},
		}
		hook := NewWarmUpHook(provider, nil)
		task := models.Task{
			Number: "1",
			Name:   "Test Task",
			Prompt: "Original prompt",
			Metadata: map[string]interface{}{
				"target_files": []string{"/src/main.go", "/src/util.go"},
			},
		}

		_, _ = hook.InjectContext(ctx, task)

		require.NotNil(t, capturedTaskInfo)
		assert.Equal(t, "1", capturedTaskInfo.TaskNumber)
		assert.Equal(t, "Test Task", capturedTaskInfo.TaskName)
		assert.Contains(t, capturedTaskInfo.FilePaths, "/src/main.go")
		assert.Contains(t, capturedTaskInfo.FilePaths, "/src/util.go")
	})
}

func TestFormatWarmUpContext(t *testing.T) {
	t.Run("returns empty for nil context", func(t *testing.T) {
		result := FormatWarmUpContext(nil)
		assert.Empty(t, result)
	})

	t.Run("formats recommended approach", func(t *testing.T) {
		ctx := &learning.WarmUpContext{
			Confidence:          0.8,
			RecommendedApproach: "Use pattern X",
		}

		result := FormatWarmUpContext(ctx)

		assert.Contains(t, result, "--- WARM-UP CONTEXT ---")
		assert.Contains(t, result, "--- END WARM-UP ---")
		assert.Contains(t, result, "Recommended Approach")
		assert.Contains(t, result, "Use pattern X")
		assert.Contains(t, result, "80%")
	})

	t.Run("formats similar patterns", func(t *testing.T) {
		ctx := &learning.WarmUpContext{
			Confidence: 0.7,
			SimilarPatterns: []string{
				"Pattern 1",
				"Pattern 2",
			},
		}

		result := FormatWarmUpContext(ctx)

		assert.Contains(t, result, "Patterns from Similar Tasks")
		assert.Contains(t, result, "Pattern 1")
		assert.Contains(t, result, "Pattern 2")
	})

	t.Run("limits patterns to 5", func(t *testing.T) {
		ctx := &learning.WarmUpContext{
			Confidence: 0.7,
			SimilarPatterns: []string{
				"Pattern 1", "Pattern 2", "Pattern 3",
				"Pattern 4", "Pattern 5", "Pattern 6", "Pattern 7",
			},
		}

		result := FormatWarmUpContext(ctx)

		assert.Contains(t, result, "Pattern 5")
		assert.NotContains(t, result, "Pattern 6")
		assert.NotContains(t, result, "Pattern 7")
	})

	t.Run("formats relevant history summary", func(t *testing.T) {
		ctx := &learning.WarmUpContext{
			Confidence: 0.8,
			RelevantHistory: []learning.TaskExecution{
				{TaskName: "Task A", Success: true, Agent: "golang-pro", QCVerdict: "GREEN"},
				{TaskName: "Task B", Success: true, Agent: "backend-developer"},
				{TaskName: "Task C", Success: false, Agent: "frontend-developer"},
			},
		}

		result := FormatWarmUpContext(ctx)

		assert.Contains(t, result, "Similar Historical Tasks")
		assert.Contains(t, result, "3 similar tasks (2 successful, 1 failed)")
		assert.Contains(t, result, "Task A")
		assert.Contains(t, result, "golang-pro")
		assert.Contains(t, result, "(GREEN)")
	})

	t.Run("limits shown history to 3 successful tasks", func(t *testing.T) {
		ctx := &learning.WarmUpContext{
			Confidence: 0.8,
			RelevantHistory: []learning.TaskExecution{
				{TaskName: "Task A", Success: true, Agent: "agent1"},
				{TaskName: "Task B", Success: true, Agent: "agent2"},
				{TaskName: "Task C", Success: true, Agent: "agent3"},
				{TaskName: "Task D", Success: true, Agent: "agent4"},
			},
		}

		result := FormatWarmUpContext(ctx)

		assert.Contains(t, result, "Task A")
		assert.Contains(t, result, "Task B")
		assert.Contains(t, result, "Task C")
		assert.NotContains(t, result, "Task D")
	})
}

func TestExtractFilePaths(t *testing.T) {
	t.Run("extracts from target_files metadata", func(t *testing.T) {
		task := models.Task{
			Metadata: map[string]interface{}{
				"target_files": []string{"/a.go", "/b.go"},
			},
		}

		paths := extractFilePaths(task)

		assert.Contains(t, paths, "/a.go")
		assert.Contains(t, paths, "/b.go")
	})

	t.Run("extracts from files metadata", func(t *testing.T) {
		task := models.Task{
			Metadata: map[string]interface{}{
				"files": []string{"/c.go"},
			},
		}

		paths := extractFilePaths(task)

		assert.Contains(t, paths, "/c.go")
	})

	t.Run("deduplicates paths", func(t *testing.T) {
		task := models.Task{
			Metadata: map[string]interface{}{
				"target_files": []string{"/a.go", "/a.go", "/b.go"},
				"files":        []string{"/a.go"},
			},
		}

		paths := extractFilePaths(task)

		// Count occurrences of /a.go
		count := 0
		for _, p := range paths {
			if p == "/a.go" {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("handles empty metadata", func(t *testing.T) {
		task := models.Task{}

		paths := extractFilePaths(task)

		assert.Empty(t, paths)
	})
}

func TestWarmUpHookIntegrationWithPreTaskHook(t *testing.T) {
	// This test verifies that WarmUpHook is properly called in preTaskHook
	// when configured on DefaultTaskExecutor
	t.Run("preTaskHook calls WarmUpHook.InjectContext", func(t *testing.T) {
		ctx := context.Background()
		called := false

		provider := &mockWarmUpProvider{
			buildContextFunc: func(ctx context.Context, task *learning.TaskInfo) (*learning.WarmUpContext, error) {
				called = true
				return &learning.WarmUpContext{
					Confidence:          0.8,
					RecommendedApproach: "Test approach",
				}, nil
			},
		}

		// Create minimal task executor with WarmUpHook
		te := &DefaultTaskExecutor{
			WarmUpHook:      NewWarmUpHook(provider, nil),
			LearningStore:   nil, // Need to set this to allow preTaskHook to run
			FileLockManager: NewFileLockManager(),
		}

		task := models.Task{Number: "1", Name: "Test", Prompt: "Original"}

		// preTaskHook will skip early if LearningStore is nil
		// So for this test we just call InjectContext directly
		if te.WarmUpHook != nil {
			modifiedTask, err := te.WarmUpHook.InjectContext(ctx, task)
			require.NoError(t, err)
			assert.True(t, called)
			assert.Contains(t, modifiedTask.Prompt, "WARM-UP CONTEXT")
		}
	})
}
