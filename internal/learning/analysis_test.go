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
	ctx := context.Background()
	store := setupTestStore(t)

	// Create execution history with specified number of failures
	for i := 0; i < failureCount; i++ {
		exec := &TaskExecution{
			PlanFile:     "plan.md",
			TaskNumber:   "Task 1",
			TaskName:     "Test Task",
			Agent:        agent,
			Prompt:       "Test prompt",
			Success:      false,
			Output:       "compilation_error: undefined reference to 'foo'",
			ErrorMessage: "Build failed",
			DurationSecs: 30,
		}
		require.NoError(t, store.RecordExecution(ctx, exec))
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
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Add 2 failures and 1 success
		executions := []*TaskExecution{
			{PlanFile: "plan.md", TaskNumber: "Task 1", TaskName: "Test", Agent: "golang-pro", Prompt: "test", Success: false, Output: "test_failure", DurationSecs: 30},
			{PlanFile: "plan.md", TaskNumber: "Task 1", TaskName: "Test", Agent: "golang-pro", Prompt: "test", Success: false, Output: "compilation_error", DurationSecs: 30},
			{PlanFile: "plan.md", TaskNumber: "Task 1", TaskName: "Test", Agent: "golang-pro", Prompt: "test", Success: true, Output: "success", DurationSecs: 30},
		}
		for _, exec := range executions {
			require.NoError(t, store.RecordExecution(ctx, exec))
		}

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
			ctx := context.Background()
			store := setupTestStore(t)
			defer store.Close()

			// Add executions with specified outputs
			for _, output := range tt.outputs {
				exec := &TaskExecution{
					PlanFile:   "plan.md",
					TaskNumber: "Task 1",
					TaskName:   "Test",
					Agent:      "golang-pro",
					Prompt:     "test",
					Success:    false,
					Output:     output,
				}
				require.NoError(t, store.RecordExecution(ctx, exec))
			}

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
				ctx := context.Background()
				// Record successful executions with test-automator
				for i := 0; i < 10; i++ {
					exec := &TaskExecution{
						PlanFile:   "plan.md",
						TaskNumber: "Task 2",
						TaskName:   "Other task",
						Agent:      "test-automator",
						Prompt:     "test",
						Success:    true,
					}
					require.NoError(t, s.RecordExecution(ctx, exec))
				}
			},
			triedAgents: []string{"golang-pro"},
			wantAgent:   "test-automator",
			wantReason:  "10 successful executions",
		},
		{
			name: "excludes already tried agents",
			setup: func(s *Store) {
				ctx := context.Background()
				// Record successes with both agents
				agents := []string{"golang-pro", "test-automator"}
				for _, agent := range agents {
					for i := 0; i < 5; i++ {
						exec := &TaskExecution{
							PlanFile:   "plan.md",
							TaskNumber: "Task 2",
							TaskName:   "Other task",
							Agent:      agent,
							Prompt:     "test",
							Success:    true,
						}
						require.NoError(t, s.RecordExecution(ctx, exec))
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
		name             string
		patterns         []string
		alternativeAgent string
		expectedContains []string
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
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Simulate trying many different agents
		agents := []string{"golang-pro", "test-automator", "backend-developer", "general-purpose"}
		for _, agent := range agents {
			exec := &TaskExecution{
				PlanFile:   "plan.md",
				TaskNumber: "Task 1",
				TaskName:   "Test",
				Agent:      agent,
				Prompt:     "test",
				Success:    false,
				Output:     "error",
			}
			require.NoError(t, store.RecordExecution(ctx, exec))
		}

		analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 1")
		require.NoError(t, err)

		assert.Equal(t, 4, len(analysis.TriedAgents))
		assert.True(t, analysis.ShouldTryDifferentAgent)
		// Should still suggest an agent (possibly one already tried)
		assert.NotEmpty(t, analysis.SuggestedAgent)
	})
}

// TestAnalyzeFailures_ExpandedKeywordMatching verifies expanded keywords are detected in analysis
func TestAnalyzeFailures_ExpandedKeywordMatching(t *testing.T) {
	tests := []struct {
		name             string
		outputs          []string
		expectedPatterns []string
	}{
		{
			name: "Build fail detected in analysis",
			outputs: []string{
				"Build failed during compilation phase",
				"Unable to build the project successfully",
			},
			expectedPatterns: []string{"compilation_error"},
		},
		{
			name: "Parse error detected in analysis",
			outputs: []string{
				"Parse error in source file",
				"Parser failed to process input",
			},
			expectedPatterns: []string{"compilation_error"},
		},
		{
			name: "Assertion fail detected in analysis",
			outputs: []string{
				"Assertion failure in test suite",
				"Multiple assertion fail messages",
			},
			expectedPatterns: []string{"test_failure"},
		},
		{
			name: "Verification fail detected in analysis",
			outputs: []string{
				"Verification failed for expected outcome",
				"Check failed during validation",
			},
			expectedPatterns: []string{"test_failure"},
		},
		{
			name: "Missing package detected in analysis",
			outputs: []string{
				"Missing package in dependencies",
				"Unable to locate required module",
			},
			expectedPatterns: []string{"dependency_missing"},
		},
		{
			name: "Import error detected in analysis",
			outputs: []string{
				"Import error for required module",
				"Cannot find module in registry",
			},
			expectedPatterns: []string{"dependency_missing"},
		},
		{
			name: "Segmentation fault detected in analysis",
			outputs: []string{
				"Segmentation fault occurred during runtime",
				"Nil pointer dereference detected",
			},
			expectedPatterns: []string{"runtime_error"},
		},
		{
			name: "Stack overflow detected in analysis",
			outputs: []string{
				"Stack overflow error in recursive call",
				"Null reference exception thrown",
			},
			expectedPatterns: []string{"runtime_error"},
		},
		{
			name: "Deadline exceeded detected in analysis",
			outputs: []string{
				"Deadline exceeded for operation",
				"Request timed out after waiting",
			},
			expectedPatterns: []string{"timeout"},
		},
		{
			name: "Execution timeout detected in analysis",
			outputs: []string{
				"Execution timeout reached",
				"Operation timed out",
			},
			expectedPatterns: []string{"timeout"},
		},
		{
			name: "Access denied detected in analysis",
			outputs: []string{
				"Access denied to protected resource",
				"Forbidden operation attempted",
			},
			expectedPatterns: []string{"permission_denied"},
		},
		{
			name: "Unauthorized detected in analysis",
			outputs: []string{
				"Unauthorized access attempt",
				"Permission denied for file system",
			},
			expectedPatterns: []string{"permission_denied"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := extractFailurePatterns(tt.outputs)

			for _, expected := range tt.expectedPatterns {
				found := false
				for _, p := range patterns {
					if p == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected pattern %q in %v", expected, patterns)
				}
			}
		})
	}
}

// TestAnalyzeFailures_KeywordCombinations verifies multiple patterns detected and deduplicated
func TestAnalyzeFailures_KeywordCombinations(t *testing.T) {
	tests := []struct {
		name             string
		outputs          []string
		expectedPatterns []string
		minPatternCount  int
	}{
		{
			name: "Multiple different error types",
			outputs: []string{
				"Build error occurred during compilation",
				"Assertion fail in unit tests",
				"Missing package dependency",
			},
			expectedPatterns: []string{"compilation_error", "test_failure", "dependency_missing"},
			minPatternCount:  3,
		},
		{
			name: "Same pattern from multiple outputs - deduplication",
			outputs: []string{
				"Build fail in first attempt",
				"Build error in second attempt",
				"Unable to build in third attempt",
			},
			expectedPatterns: []string{"compilation_error"},
			minPatternCount:  1,
		},
		{
			name: "Runtime and permission errors together",
			outputs: []string{
				"Segmentation fault in process",
				"Access denied to file",
				"Nil pointer dereference",
				"Forbidden operation",
			},
			expectedPatterns: []string{"runtime_error", "permission_denied"},
			minPatternCount:  2,
		},
		{
			name: "Timeout and dependency errors mixed",
			outputs: []string{
				"Deadline exceeded waiting for response",
				"Import error for module",
				"Request timed out",
				"Cannot find module",
			},
			expectedPatterns: []string{"timeout", "dependency_missing"},
			minPatternCount:  2,
		},
		{
			name: "All error types in comprehensive failure",
			outputs: []string{
				"Build fail detected",
				"Verification fail in tests",
				"Missing package xyz",
				"Segmentation fault",
				"Deadline exceeded",
				"Access denied",
			},
			expectedPatterns: []string{
				"compilation_error",
				"test_failure",
				"dependency_missing",
				"runtime_error",
				"timeout",
				"permission_denied",
			},
			minPatternCount: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := extractFailurePatterns(tt.outputs)

			// Check that all expected patterns are present
			for _, expected := range tt.expectedPatterns {
				found := false
				for _, p := range patterns {
					if p == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected pattern %q in %v", expected, patterns)
				}
			}

			// Check minimum pattern count (deduplication should work)
			if len(patterns) < tt.minPatternCount {
				t.Errorf("expected at least %d patterns, got %d: %v",
					tt.minPatternCount, len(patterns), patterns)
			}
		})
	}
}

// TestAnalyzeFailures_HistoricalPatternRecovery verifies pattern extraction from historical data
func TestAnalyzeFailures_HistoricalPatternRecovery(t *testing.T) {
	t.Run("recovers patterns from multiple failed execution history", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Simulate 3 failed runs with different expanded keyword errors
		executions := []struct {
			output string
		}{
			{output: "Build error occurred during first attempt"},
			{output: "Parse error found in second attempt"},
			{output: "Unable to build in third attempt"},
		}

		for i, exec := range executions {
			taskExec := &TaskExecution{
				PlanFile:     "plan.md",
				RunNumber:    i + 1,
				TaskNumber:   "Task 1",
				TaskName:     "Build Project",
				Agent:        "golang-pro",
				Prompt:       "Build the project",
				Success:      false,
				Output:       exec.output,
				ErrorMessage: "compilation failed",
				DurationSecs: 30,
			}
			require.NoError(t, store.RecordExecution(ctx, taskExec))
		}

		// Analyze should detect compilation_error pattern from all 3 outputs
		analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 1")
		require.NoError(t, err)

		assert.Equal(t, 3, analysis.TotalAttempts)
		assert.Equal(t, 3, analysis.FailedAttempts)

		// Should detect compilation_error pattern from expanded keywords
		found := false
		for _, pattern := range analysis.CommonPatterns {
			if pattern == "compilation_error" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected compilation_error pattern in %v", analysis.CommonPatterns)
		}
	})

	t.Run("recovers mixed patterns from diverse failure history", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Simulate failures with various error types using expanded keywords
		executions := []struct {
			output string
		}{
			{output: "Build fail - compilation phase failed"},
			{output: "Assertion fail in unit test suite"},
			{output: "Missing package required for build"},
			{output: "Segmentation fault during runtime"},
			{output: "Deadline exceeded waiting for response"},
			{output: "Access denied to configuration file"},
		}

		for i, exec := range executions {
			taskExec := &TaskExecution{
				PlanFile:     "plan.md",
				RunNumber:    i + 1,
				TaskNumber:   "Task 2",
				TaskName:     "Complex Task",
				Agent:        "backend-developer",
				Prompt:       "Execute complex task",
				Success:      false,
				Output:       exec.output,
				ErrorMessage: "task failed",
				DurationSecs: 45,
			}
			require.NoError(t, store.RecordExecution(ctx, taskExec))
		}

		// Analyze should detect all pattern types
		analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 2")
		require.NoError(t, err)

		assert.Equal(t, 6, analysis.TotalAttempts)
		assert.Equal(t, 6, analysis.FailedAttempts)

		// Should detect at least 5 different patterns (could be 6 if permission_denied counted separately)
		expectedPatterns := []string{
			"compilation_error",
			"test_failure",
			"dependency_missing",
			"runtime_error",
			"timeout",
		}

		for _, expected := range expectedPatterns {
			found := false
			for _, pattern := range analysis.CommonPatterns {
				if pattern == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected pattern %q in %v from historical analysis", expected, analysis.CommonPatterns)
			}
		}

		// Should have at least 5 common patterns
		if len(analysis.CommonPatterns) < 5 {
			t.Errorf("expected at least 5 common patterns, got %d: %v",
				len(analysis.CommonPatterns), analysis.CommonPatterns)
		}
	})

	t.Run("handles case where only some outputs have keywords", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Mix of outputs with and without expanded keywords
		executions := []struct {
			output string
		}{
			{output: "Something went wrong"},             // No specific keyword
			{output: "Build error in compilation phase"}, // Has keyword
			{output: "Generic error message"},            // No specific keyword
			{output: "Unable to build the project"},      // Has keyword
			{output: "An unexpected error occurred"},     // No specific keyword
			{output: "Parse error in source file"},       // Has keyword
		}

		for i, exec := range executions {
			taskExec := &TaskExecution{
				PlanFile:     "plan.md",
				RunNumber:    i + 1,
				TaskNumber:   "Task 3",
				TaskName:     "Partial Pattern Task",
				Agent:        "general-purpose",
				Prompt:       "Execute task",
				Success:      false,
				Output:       exec.output,
				ErrorMessage: "task failed",
				DurationSecs: 20,
			}
			require.NoError(t, store.RecordExecution(ctx, taskExec))
		}

		// Analyze should still detect compilation_error from the outputs that have keywords
		analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 3")
		require.NoError(t, err)

		assert.Equal(t, 6, analysis.TotalAttempts)
		assert.Equal(t, 6, analysis.FailedAttempts)

		// Should detect compilation_error from 3 outputs with keywords
		found := false
		for _, pattern := range analysis.CommonPatterns {
			if pattern == "compilation_error" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected compilation_error pattern even with partial keyword coverage, got %v",
				analysis.CommonPatterns)
		}
	})
}

func TestAnalyzeFailures_InsufficientData(t *testing.T) {
	t.Run("handles insufficient data gracefully", func(t *testing.T) {
		ctx := context.Background()
		store := setupTestStore(t)
		defer store.Close()

		// Add only 2 executions from different agents (not enough for statistical significance)
		executions := []*TaskExecution{
			{PlanFile: "plan.md", TaskNumber: "Other", TaskName: "Test", Agent: "agent-a", Prompt: "test", Success: true},
			{PlanFile: "plan.md", TaskNumber: "Other", TaskName: "Test", Agent: "agent-b", Prompt: "test", Success: true},
		}
		for _, exec := range executions {
			require.NoError(t, store.RecordExecution(ctx, exec))
		}

		// Now analyze a task with 2 failures
		for i := 0; i < 2; i++ {
			exec := &TaskExecution{
				PlanFile:   "plan.md",
				TaskNumber: "Task 1",
				TaskName:   "Test",
				Agent:      "golang-pro",
				Prompt:     "test",
				Success:    false,
			}
			require.NoError(t, store.RecordExecution(ctx, exec))
		}

		analysis, err := store.AnalyzeFailures(ctx, "plan.md", "Task 1")
		require.NoError(t, err)

		// Should still provide a suggestion, even if not statistically significant
		assert.True(t, analysis.ShouldTryDifferentAgent)
		assert.NotEmpty(t, analysis.SuggestedAgent)
	})
}

func TestAnalyzeFailures_ConflictingPatterns(t *testing.T) {
	t.Run("handles conflicting patterns in execution history", func(t *testing.T) {
		ctx := context.Background()
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
				PlanFile:   "plan.md",
				TaskNumber: "Task 1",
				TaskName:   "Test",
				Agent:      "golang-pro",
				Prompt:     "test",
				Success:    false,
				Output:     output,
			}
			require.NoError(t, store.RecordExecution(ctx, exec))
		}

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
