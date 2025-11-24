// Package behavioral provides data structures and utilities for extracting
// behavioral metrics from Claude Code agent session JSONL files.
//
// This package contains core models representing:
//   - Agent session metadata and execution results
//   - Tool execution statistics and performance metrics
//   - Bash command tracking and success rates
//   - File operation patterns and sizes
//   - Token usage and cost calculations
//
// The models mirror the JSONL schema from ~/.claude/projects/*/agent-*.jsonl
// and provide validation methods and calculation helpers for metrics aggregation.
//
// Example usage:
//
//	session := &behavioral.Session{
//	    ID:        "session-123",
//	    Project:   "my-project",
//	    Timestamp: time.Now(),
//	    Status:    "completed",
//	}
//	if err := session.Validate(); err != nil {
//	    log.Fatal(err)
//	}
//
//	metrics := &behavioral.BehavioralMetrics{}
//	metrics.AggregateMetrics([]behavioral.Session{*session})
package behavioral
