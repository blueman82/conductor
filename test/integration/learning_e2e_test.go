package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/harrison/conductor/internal/learning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_FailingTaskWithAgentSwap tests DB storage for failed/successful attempts
func TestE2E_FailingTaskWithAgentSwap(t *testing.T) {
	t.Skip("TODO: Requires full QC integration - see internal/executor/learning_integration_test.go for comprehensive tests")
}

// TestE2E_MultipleRetriesWithLearning tests multiple retry attempts with learning data accumulation
func TestE2E_MultipleRetriesWithLearning(t *testing.T) {
	t.Skip("TODO: Requires full QC integration - see internal/executor/learning_integration_test.go for comprehensive tests")
}

// TestE2E_LearningDisabledNoAgentSwap verifies no agent swapping when learning disabled
func TestE2E_LearningDisabledNoAgentSwap(t *testing.T) {
	t.Skip("TODO: Requires full QC integration - see internal/executor/learning_integration_test.go for comprehensive tests")
}

// TestE2E_CrossRunLearning verifies learning persists across multiple conductor runs
func TestE2E_CrossRunLearning(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "learning.db")
	planPath := filepath.Join(tmpDir, "plan.md")

	// Run 1: Record a failure
	{
		store, err := learning.NewStore(dbPath)
		require.NoError(t, err)

		exec := &learning.TaskExecution{
			PlanFile:     planPath,
			RunNumber:    1,
			TaskNumber:   "1",
			TaskName:     "Persistent task",
			Agent:        "agent-a",
			Prompt:       "Do something",
			Success:      false,
			Output:       "compilation_error",
			ErrorMessage: "Task failed",
			DurationSecs: 30,
		}
		err = store.RecordExecution(context.Background(), exec)
		require.NoError(t, err)

		store.Close()
	}

	// Run 2: Verify persistence
	{
		store, err := learning.NewStore(dbPath)
		require.NoError(t, err)
		defer store.Close()

		// Verify Run 1 data persists
		history, err := store.GetExecutionHistory(context.Background(), planPath, "1")
		require.NoError(t, err)
		require.Len(t, history, 1, "Should find Run 1 execution")
		assert.False(t, history[0].Success)
		assert.Equal(t, "agent-a", history[0].Agent)

		// Verify analysis suggests different agent
		analysis, err := store.AnalyzeFailures(context.Background(), planPath, "1", 1)
		require.NoError(t, err)
		assert.True(t, analysis.ShouldTryDifferentAgent)
		assert.NotEqual(t, "agent-a", analysis.SuggestedAgent)
	}
}
