package executor

import (
	"context"

	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/models"
)

// LIPCollectorHook wraps learning.Store to provide LIP event collection for task execution.
// This is a thin adapter layer that:
// - Records NEW event types (test/build results) via RecordTestResult/RecordBuildResult
// - Does NOT duplicate existing behavioral tracking (tool/file events in RecordSessionMetrics)
// - Creates knowledge graph edges for task relationships in post-task hook
type LIPCollectorHook struct {
	store  *learning.Store
	kg     *learning.SQLiteKnowledgeGraph
	logger RuntimeEnforcementLogger
}

// NewLIPCollectorHook creates a new LIP collector hook.
// Returns nil if store is nil (graceful degradation pattern consistent with other hooks).
func NewLIPCollectorHook(store *learning.Store, logger RuntimeEnforcementLogger) *LIPCollectorHook {
	if store == nil {
		return nil
	}
	return &LIPCollectorHook{
		store:  store,
		kg:     store.NewKnowledgeGraph(),
		logger: logger,
	}
}

// RecordTestResult records a test pass/fail event for a task execution.
// This is a NEW event type that complements existing behavioral tracking.
// Uses Store.RecordTestResult which handles LIP event creation.
func (h *LIPCollectorHook) RecordTestResult(ctx context.Context, taskExecutionID int64, taskNumber string, passed bool, details string) error {
	if h == nil || h.store == nil {
		return nil // Graceful degradation
	}

	if err := h.store.RecordTestResult(ctx, taskExecutionID, taskNumber, passed, details); err != nil {
		if h.logger != nil {
			h.logger.Warnf("LIP: failed to record test result for task %s: %v", taskNumber, err)
		}
		return nil // Graceful degradation - don't fail task on LIP storage error
	}

	return nil
}

// RecordBuildResult records a build success/fail event for a task execution.
// This is a NEW event type that complements existing behavioral tracking.
// Uses Store.RecordBuildResult which handles LIP event creation.
func (h *LIPCollectorHook) RecordBuildResult(ctx context.Context, taskExecutionID int64, taskNumber string, success bool, details string) error {
	if h == nil || h.store == nil {
		return nil // Graceful degradation
	}

	if err := h.store.RecordBuildResult(ctx, taskExecutionID, taskNumber, success, details); err != nil {
		if h.logger != nil {
			h.logger.Warnf("LIP: failed to record build result for task %s: %v", taskNumber, err)
		}
		return nil // Graceful degradation - don't fail task on LIP storage error
	}

	return nil
}

// RecordTestResults records multiple test results from a test run.
// Convenience method that iterates over TestCommandResults.
func (h *LIPCollectorHook) RecordTestResults(ctx context.Context, taskExecutionID int64, taskNumber string, results []TestCommandResult) error {
	if h == nil || h.store == nil || len(results) == 0 {
		return nil // Graceful degradation
	}

	for _, r := range results {
		details := r.Command
		if r.Output != "" && len(r.Output) <= 500 {
			details += ": " + r.Output
		} else if r.Output != "" {
			details += ": " + r.Output[:500] + "..."
		}
		if r.Error != nil {
			details += " [error: " + r.Error.Error() + "]"
		}

		if err := h.RecordTestResult(ctx, taskExecutionID, taskNumber, r.Passed, details); err != nil {
			// Already logged by RecordTestResult
			continue
		}
	}

	return nil
}

// RecordTaskFileRelation creates a knowledge graph edge: task → modifies → file.
// Called in post-task hook to record which files a task modified.
func (h *LIPCollectorHook) RecordTaskFileRelation(ctx context.Context, taskID, filePath string, weight float64) error {
	if h == nil || h.kg == nil {
		return nil // Graceful degradation
	}

	// Ensure task node exists
	taskNode := &learning.KnowledgeNode{
		ID:       taskID,
		NodeType: learning.NodeTypeTask,
	}
	if err := h.kg.AddNode(ctx, taskNode); err != nil {
		// Node may already exist - ignore duplicate errors
		if h.logger != nil {
			h.logger.Warnf("LIP: failed to add task node %s: %v", taskID, err)
		}
	}

	// Ensure file node exists
	fileNode := &learning.KnowledgeNode{
		ID:       filePath,
		NodeType: learning.NodeTypeFile,
		Properties: map[string]interface{}{
			"path": filePath,
		},
	}
	if err := h.kg.AddNode(ctx, fileNode); err != nil {
		// Node may already exist - ignore duplicate errors
	}

	// Create edge: task → modifies → file
	edge := &learning.KnowledgeEdge{
		SourceID: taskID,
		TargetID: filePath,
		EdgeType: learning.EdgeTypeModifies,
		Weight:   weight,
	}
	if err := h.kg.AddEdge(ctx, edge); err != nil {
		if h.logger != nil {
			h.logger.Warnf("LIP: failed to add modifies edge for task %s: %v", taskID, err)
		}
		return nil // Graceful degradation
	}

	return nil
}

// RecordTaskAgentRelation creates a knowledge graph edge based on task outcome.
// task → succeeded_with → agent (for successful tasks)
// task → used_by → agent (for failed tasks, using used_by edge type)
// Called in post-task hook to record agent effectiveness.
func (h *LIPCollectorHook) RecordTaskAgentRelation(ctx context.Context, taskID, agentName string, success bool, weight float64) error {
	if h == nil || h.kg == nil || agentName == "" {
		return nil // Graceful degradation
	}

	// Ensure task node exists
	taskNode := &learning.KnowledgeNode{
		ID:       taskID,
		NodeType: learning.NodeTypeTask,
	}
	if err := h.kg.AddNode(ctx, taskNode); err != nil {
		// Node may already exist
	}

	// Ensure agent node exists
	agentNode := &learning.KnowledgeNode{
		ID:       agentName,
		NodeType: learning.NodeTypeAgent,
		Properties: map[string]interface{}{
			"name": agentName,
		},
	}
	if err := h.kg.AddNode(ctx, agentNode); err != nil {
		// Node may already exist
	}

	// Create edge based on outcome
	edgeType := learning.EdgeTypeSucceededWith
	if !success {
		edgeType = learning.EdgeTypeUsedBy // Records that agent was used (even if failed)
	}

	edge := &learning.KnowledgeEdge{
		SourceID: taskID,
		TargetID: agentName,
		EdgeType: edgeType,
		Weight:   weight,
		Metadata: map[string]interface{}{
			"success": success,
		},
	}
	if err := h.kg.AddEdge(ctx, edge); err != nil {
		if h.logger != nil {
			h.logger.Warnf("LIP: failed to add agent edge for task %s: %v", taskID, err)
		}
		return nil // Graceful degradation
	}

	return nil
}

// PostTaskHook is called after task completion to record knowledge graph relationships.
// Creates edges for:
// - task → modifies → file (for each modified file)
// - task → succeeded_with → agent (or used_by for failures)
func (h *LIPCollectorHook) PostTaskHook(ctx context.Context, task models.Task, result *models.TaskResult, success bool) {
	if h == nil || h.kg == nil {
		return // Graceful degradation
	}

	// Generate task ID from task number (consistent with graph queries)
	taskID := "task:" + task.Number

	// Record agent relationship
	agentName := task.Agent
	if agentName == "" {
		agentName = "default"
	}
	weight := 1.0
	if !success {
		weight = 0.5 // Lower weight for failed attempts
	}
	_ = h.RecordTaskAgentRelation(ctx, taskID, agentName, success, weight)

	// Note: File modification tracking would require access to file operations data
	// from the behavioral collector. This is handled by RecordSessionMetrics which
	// already tracks file_operations. For now, knowledge graph edges for files
	// can be created by querying behavioral_sessions/file_operations and linking
	// them post-hoc, or by enhancing the behavioral collector to call this hook.
}

// GetProgressScore calculates the aggregate progress score for a task execution.
// Wraps Store.CalculateProgress for convenience.
func (h *LIPCollectorHook) GetProgressScore(ctx context.Context, taskExecutionID int64) (learning.ProgressScore, error) {
	if h == nil || h.store == nil {
		return learning.ProgressNone, nil // Graceful degradation
	}

	return h.store.CalculateProgress(ctx, taskExecutionID)
}
