package learning

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestStoreWithHistory creates a test store and populates it with execution history
func setupTestStoreWithHistory(t *testing.T, failureCount int, agent string) *Store {
	t.Helper()
	store := setupTestStore(t)

	// Create execution history with specified number of failures
	for i := 0; i < failureCount; i++ {
		exec := &TaskExecution{
			TaskNumber:   "Task 1",
			TaskName:     "Test Task",
			Agent:        agent,
			Prompt:       "Test prompt",
			Success:      false,
			Output:       "compilation_error: undefined reference to 'foo'",
			ErrorMessage: "Build failed",
			DurationSecs: 30,
		}
		require.NoError(t, store.RecordExecution(exec))
	}

	return store
}

func TestAnalyzeFailures_NoHistory(t *testing.T) {
	t.Run("returns empty analysis when no history exists", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		ctx := context.Background()
		analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 1")
		require.NoError(t, err)
		require.NotNil(t, analysis)

		assert.Equal(t, 0, analysis.TotalAttempts)
		assert.Equal(t, 0, analysis.FailedAttempts)
		assert.False(t, analysis.ShouldTryDifferentAgent)
		assert.Empty(t, analysis.TriedAgents)
		assert.Empty(t, analysis.CommonPatterns)
	})
}

func TestAnalyzeFailures_OneFailure(t *testing.T) {
	t.Run("does not suggest different agent after single failure", func(t *testing.T) {
		store := setupTestStoreWithHistory(t, 1, "golang-pro")
		defer store.Close()

		ctx := context.Background()
		analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 1")
		require.NoError(t, err)
		require.NotNil(t, analysis)

		assert.Equal(t, 1, analysis.TotalAttempts)
		assert.Equal(t, 1, analysis.FailedAttempts)
		assert.False(t, analysis.ShouldTryDifferentAgent)
		assert.Contains(t, analysis.TriedAgents, "golang-pro")
	})
}

func TestAnalyzeFailures_TwoFailures(t *testing.T) {
	t.Run("suggests different agent after two failures", func(t *testing.T) {
		store := setupTestStoreWithHistory(t, 2, "golang-pro")
		defer store.Close()

		ctx := context.Background()
		analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 1")
		require.NoError(t, err)
		require.NotNil(t, analysis)

		assert.Equal(t, 2, analysis.TotalAttempts)
		assert.Equal(t, 2, analysis.FailedAttempts)
		assert.True(t, analysis.ShouldTryDifferentAgent)
		assert.Contains(t, analysis.TriedAgents, "golang-pro")
		assert.NotEmpty(t, analysis.SuggestedAgent)
	})
}

func TestAnalyzeFailures_MixedResults(t *testing.T) {
	t.Run("handles mixed success and failure results", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// Add 2 failures and 1 success
		executions := []*TaskExecution{
			{TaskNumber: "Task 1", TaskName: "Test", Agent: "golang-pro", Prompt: "test", Success: false, Output: "test_failure", DurationSecs: 30},
			{TaskNumber: "Task 1", TaskName: "Test", Agent: "golang-pro", Prompt: "test", Success: false, Output: "compilation_error", DurationSecs: 30},
			{TaskNumber: "Task 1", TaskName: "Test", Agent: "golang-pro", Prompt: "test", Success: true, Output: "success", DurationSecs: 30},
		}
		for _, exec := range executions {
			require.NoError(t, store.RecordExecution(exec))
		}

		ctx := context.Background()
		analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 1")
		require.NoError(t, err)

		assert.Equal(t, 3, analysis.TotalAttempts)
		assert.Equal(t, 2, analysis.FailedAttempts)
	})
}

func TestAnalyzeFailures_CommonPatterns(t *testing.T) {
	tests := []struct {
		name            string
		outputs         []string
		expectedPattern string
	}{
		{
			name: "detects compilation errors",
			outputs: []string{
				"compilation_error: undefined reference",
				"compilation_error: missing semicolon",
			},
			expectedPattern: "compilation_error",
		},
		{
			name: "detects test failures",
			outputs: []string{
				"test_failure: TestFoo failed",
				"test_failure: TestBar failed",
			},
			expectedPattern: "test_failure",
		},
		{
			name: "detects dependency issues",
			outputs: []string{
				"dependency_missing: package not found",
				"dependency_missing: module not installed",
			},
			expectedPattern: "dependency_missing",
		},
		{
			name: "detects timeout issues",
			outputs: []string{
				"timeout: operation exceeded 5 minutes",
				"timeout: command did not complete",
			},
			expectedPattern: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupTestStore(t)
			defer store.Close()

			// Add executions with specified outputs
			for _, output := range tt.outputs {
				exec := &TaskExecution{
					TaskNumber: "Task 1",
					TaskName:   "Test",
					Agent:      "golang-pro",
					Prompt:     "test",
					Success:    false,
					Output:     output,
				}
				require.NoError(t, store.RecordExecution(exec))
			}

			ctx := context.Background()
			analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 1")
			require.NoError(t, err)

			assert.Contains(t, analysis.CommonPatterns, tt.expectedPattern)
		})
	}
}

func TestFindBestAlternativeAgent(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Store)
		triedAgents []string
		wantAgent   string
		wantReason  string
	}{
		{
			name: "suggests agent with high success rate",
			setup: func(s *Store) {
				// Record successful executions with test-automator
				for i := 0; i < 10; i++ {
					exec := &TaskExecution{
						TaskNumber: "Task 2",
						TaskName:   "Other task",
						Agent:      "test-automator",
						Prompt:     "test",
						Success:    true,
					}
					require.NoError(t, s.RecordExecution(exec))
				}
			},
			triedAgents: []string{"golang-pro"},
			wantAgent:   "test-automator",
			wantReason:  "10 successful executions",
		},
		{
			name: "excludes already tried agents",
			setup: func(s *Store) {
				// Record successes with both agents
				agents := []string{"golang-pro", "test-automator"}
				for _, agent := range agents {
					for i := 0; i < 5; i++ {
						exec := &TaskExecution{
							TaskNumber: "Task 2",
							TaskName:   "Other task",
							Agent:      agent,
							Prompt:     "test",
							Success:    true,
						}
						require.NoError(t, s.RecordExecution(exec))
					}
				}
			},
			triedAgents: []string{"golang-pro"},
			wantAgent:   "test-automator",
		},
		{
			name:        "returns general-purpose when no alternatives",
			setup:       func(s *Store) {},
			triedAgents: []string{"golang-pro", "test-automator"},
			wantAgent:   "general-purpose",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupTestStore(t)
			defer store.Close()

			if tt.setup != nil {
				tt.setup(store)
			}

			ctx := context.Background()
			agent, reason := store.findBestAlternativeAgent(ctx, tt.triedAgents)
			assert.Equal(t, tt.wantAgent, agent)
			if tt.wantReason != "" {
				assert.Contains(t, reason, tt.wantReason)
			}
		})
	}
}

func TestExtractFailurePatterns(t *testing.T) {
	tests := []struct {
		name     string
		outputs  []string
		expected []string
	}{
		{
			name: "extracts single pattern",
			outputs: []string{
				"compilation_error: undefined reference",
			},
			expected: []string{"compilation_error"},
		},
		{
			name: "extracts multiple patterns",
			outputs: []string{
				"compilation_error: undefined",
				"test_failure: failed",
				"compilation_error: syntax",
			},
			expected: []string{"compilation_error", "test_failure"},
		},
		{
			name: "handles no recognizable patterns",
			outputs: []string{
				"some generic error message",
				"another unstructured error",
			},
			expected: []string{},
		},
		{
			name: "deduplicates patterns",
			outputs: []string{
				"timeout: exceeded limit",
				"timeout: took too long",
				"timeout: cancelled",
			},
			expected: []string{"timeout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := extractFailurePatterns(tt.outputs)

			// Check that all expected patterns are present
			for _, expected := range tt.expected {
				assert.Contains(t, patterns, expected)
			}

			// Check that we don't have extra patterns
			if len(tt.expected) == 0 {
				assert.Empty(t, patterns)
			}
		})
	}
}

func TestGenerateApproachSuggestion(t *testing.T) {
	tests := []struct {
		name              string
		patterns          []string
		alternativeAgent  string
		expectedContains  []string
	}{
		{
			name:             "suggests approach for compilation errors",
			patterns:         []string{"compilation_error"},
			alternativeAgent: "golang-pro",
			expectedContains: []string{"compilation", "golang-pro"},
		},
		{
			name:             "suggests approach for test failures",
			patterns:         []string{"test_failure"},
			alternativeAgent: "test-automator",
			expectedContains: []string{"test", "test-automator"},
		},
		{
			name:             "suggests approach for dependency issues",
			patterns:         []string{"dependency_missing"},
			alternativeAgent: "general-purpose",
			expectedContains: []string{"dependencies", "general-purpose"},
		},
		{
			name:             "handles multiple patterns",
			patterns:         []string{"compilation_error", "test_failure"},
			alternativeAgent: "golang-pro",
			expectedContains: []string{"compilation", "test"},
		},
		{
			name:             "provides default suggestion",
			patterns:         []string{},
			alternativeAgent: "general-purpose",
			expectedContains: []string{"general-purpose"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := generateApproachSuggestion(tt.patterns, tt.alternativeAgent)
			assert.NotEmpty(t, suggestion)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, suggestion, expected)
			}
		})
	}
}

func TestAnalyzeFailures_AllAgentsTried(t *testing.T) {
	t.Run("handles case where all agents have been tried", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// Simulate trying many different agents
		agents := []string{"golang-pro", "test-automator", "backend-developer", "general-purpose"}
		for _, agent := range agents {
			exec := &TaskExecution{
				TaskNumber: "Task 1",
				TaskName:   "Test",
				Agent:      agent,
				Prompt:     "test",
				Success:    false,
				Output:     "error",
			}
			require.NoError(t, store.RecordExecution(exec))
		}

		ctx := context.Background()
		analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 1")
		require.NoError(t, err)

		assert.Equal(t, 4, len(analysis.TriedAgents))
		assert.True(t, analysis.ShouldTryDifferentAgent)
		// Should still suggest an agent (possibly one already tried)
		assert.NotEmpty(t, analysis.SuggestedAgent)
	})
}

func TestAnalyzeFailures_InsufficientData(t *testing.T) {
	t.Run("handles insufficient data gracefully", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// Add only 2 executions from different agents (not enough for statistical significance)
		executions := []*TaskExecution{
			{TaskNumber: "Other", TaskName: "Test", Agent: "agent-a", Prompt: "test", Success: true},
			{TaskNumber: "Other", TaskName: "Test", Agent: "agent-b", Prompt: "test", Success: true},
		}
		for _, exec := range executions {
			require.NoError(t, store.RecordExecution(exec))
		}

		// Now analyze a task with 2 failures
		for i := 0; i < 2; i++ {
			exec := &TaskExecution{
				TaskNumber: "Task 1",
				TaskName:   "Test",
				Agent:      "golang-pro",
				Prompt:     "test",
				Success:    false,
			}
			require.NoError(t, store.RecordExecution(exec))
		}

		ctx := context.Background()
		analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 1")
		require.NoError(t, err)

		// Should still provide a suggestion, even if not statistically significant
		assert.True(t, analysis.ShouldTryDifferentAgent)
		assert.NotEmpty(t, analysis.SuggestedAgent)
	})
}

func TestAnalyzeFailures_ConflictingPatterns(t *testing.T) {
	t.Run("handles conflicting patterns in execution history", func(t *testing.T) {
		store := setupTestStore(t)
		defer store.Close()

		// Add executions with different patterns
		outputs := []string{
			"compilation_error: syntax error",
			"test_failure: assertion failed",
			"timeout: exceeded limit",
			"compilation_error: type mismatch",
		}
		for _, output := range outputs {
			exec := &TaskExecution{
				TaskNumber: "Task 1",
				TaskName:   "Test",
				Agent:      "golang-pro",
				Prompt:     "test",
				Success:    false,
				Output:     output,
			}
			require.NoError(t, store.RecordExecution(exec))
		}

		ctx := context.Background()
		analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 1")
		require.NoError(t, err)

		// Should identify multiple patterns
		assert.GreaterOrEqual(t, len(analysis.CommonPatterns), 2)
		assert.Contains(t, analysis.CommonPatterns, "compilation_error")
	})
}

func TestAnalyzeFailures_ContextCancellation(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		store := setupTestStoreWithHistory(t, 2, "golang-pro")
		defer store.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := store.AnalyzeFailures(ctx, "plan.md", "Task 1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context")
	})
}
