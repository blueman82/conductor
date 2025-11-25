package behavioral

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// SessionData represents complete session data including all events
type SessionData struct {
	Session Session `json:"session"` // Session metadata
	Events  []Event `json:"events"`  // All events from the session
}

// Event is the interface for all event types
type Event interface {
	GetType() string
	GetTimestamp() time.Time
	Validate() error
}

// BaseEvent contains common fields for all events
type BaseEvent struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

// GetType returns event type
func (e *BaseEvent) GetType() string {
	return e.Type
}

// GetTimestamp returns event timestamp
func (e *BaseEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

// ToolCallEvent represents tool invocation events
type ToolCallEvent struct {
	BaseEvent
	ToolName   string                 `json:"tool_name"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     string                 `json:"result"`
	Success    bool                   `json:"success"`
	Duration   int64                  `json:"duration"` // milliseconds
	Error      string                 `json:"error,omitempty"`
}

// Validate checks if tool call event is valid
func (e *ToolCallEvent) Validate() error {
	if e.Type == "" {
		return errors.New("event type is required")
	}
	if e.ToolName == "" {
		return errors.New("tool name is required")
	}
	if e.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	return nil
}

// BashCommandEvent represents bash command execution events
type BashCommandEvent struct {
	BaseEvent
	Command      string `json:"command"`
	ExitCode     int    `json:"exit_code"`
	Output       string `json:"output"`
	OutputLength int    `json:"output_length"`
	Duration     int64  `json:"duration"` // milliseconds
	Success      bool   `json:"success"`
}

// Validate checks if bash command event is valid
func (e *BashCommandEvent) Validate() error {
	if e.Type == "" {
		return errors.New("event type is required")
	}
	if e.Command == "" {
		return errors.New("command is required")
	}
	if e.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	return nil
}

// FileOperationEvent represents file operation events
type FileOperationEvent struct {
	BaseEvent
	Operation string `json:"operation"` // read, write, edit, delete
	Path      string `json:"path"`
	Success   bool   `json:"success"`
	SizeBytes int64  `json:"size_bytes"`
	Duration  int64  `json:"duration"` // milliseconds
	Error     string `json:"error,omitempty"`
}

// Validate checks if file operation event is valid
func (e *FileOperationEvent) Validate() error {
	if e.Type == "" {
		return errors.New("event type is required")
	}
	if e.Operation == "" {
		return errors.New("operation is required")
	}
	if e.Path == "" {
		return errors.New("file path is required")
	}
	if e.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	if e.SizeBytes < 0 {
		return errors.New("file size cannot be negative")
	}
	return nil
}

// TokenUsageEvent represents token consumption events
type TokenUsageEvent struct {
	BaseEvent
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	ModelName    string  `json:"model_name"`
}

// Validate checks if token usage event is valid
func (e *TokenUsageEvent) Validate() error {
	if e.Type == "" {
		return errors.New("event type is required")
	}
	if e.InputTokens < 0 {
		return errors.New("input tokens cannot be negative")
	}
	if e.OutputTokens < 0 {
		return errors.New("output tokens cannot be negative")
	}
	if e.CostUSD < 0 {
		return errors.New("cost cannot be negative")
	}
	if e.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	return nil
}

// ParseSessionFile parses a JSONL session file and returns structured session data
// Gracefully handles malformed lines by skipping them with warning logs
func ParseSessionFile(filepath string) (*SessionData, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	sessionData := &SessionData{
		Events: make([]Event, 0),
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer size for long lines (some JSONL lines can be very long)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max line size
	lineNum := 0
	sessionSeen := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if line == "" {
			continue
		}

		// Check if this is explicit session metadata (test format)
		if isSessionMetadata(line) {
			if err := parseSessionMetadata(line, &sessionData.Session); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse session metadata at line %d: %v\n", lineNum, err)
			} else {
				sessionSeen = true
			}
			continue
		}

		// Don't try to parse every line as session metadata - only session start lines contain
		// session metadata. Event lines like tool_call with "success":true would corrupt the
		// session.Success flag.

		events, err := parseEventLine(line)
		if err != nil {
			// Log warning but continue parsing - graceful degradation
			fmt.Fprintf(os.Stderr, "Warning: skipping malformed event at line %d: %v\n", lineNum, err)
			continue
		}

		// Skip nil/empty events (known non-event types like summary, file-history-snapshot)
		if len(events) == 0 {
			continue
		}

		sessionData.Events = append(sessionData.Events, events...)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading session file: %w", err)
	}

	// For real Claude sessions, session ID might be in any line with sessionId field
	// Be lenient - only error if we have absolutely no session info
	if !sessionSeen && sessionData.Session.ID == "" {
		// If file is completely empty, that's OK (return empty session)
		// But if we parsed events without session metadata, that's an error
		if len(sessionData.Events) > 0 {
			return nil, fmt.Errorf("events found but no session metadata provided")
		}
		// Empty file is OK
		return sessionData, nil
	}

	return sessionData, nil
}

// parseEventLine parses a single JSONL line into multiple Events
// Supports both simplified test format and actual Claude Code JSONL format
// Returns ALL events from a line (multiple tool_use blocks + token_usage)
func parseEventLine(line string) ([]Event, error) {
	// First parse to get event type
	var base BaseEvent
	if err := json.Unmarshal([]byte(line), &base); err != nil {
		return nil, fmt.Errorf("failed to parse base event: %w", err)
	}

	// Route to specific event type based on type field
	switch base.Type {
	case "tool_call", "tool_execution":
		var event ToolCallEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("failed to parse tool call event: %w", err)
		}
		if err := event.Validate(); err != nil {
			return nil, err
		}
		return []Event{&event}, nil

	case "bash_command":
		var event BashCommandEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("failed to parse bash command event: %w", err)
		}
		if err := event.Validate(); err != nil {
			return nil, err
		}
		return []Event{&event}, nil

	case "file_operation":
		var event FileOperationEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("failed to parse file operation event: %w", err)
		}
		if err := event.Validate(); err != nil {
			return nil, err
		}
		return []Event{&event}, nil

	case "token_usage":
		var event TokenUsageEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("failed to parse token usage event: %w", err)
		}
		if err := event.Validate(); err != nil {
			return nil, err
		}
		return []Event{&event}, nil

	case "assistant", "user":
		// Real Claude Code JSONL format - extract ALL nested events
		return parseClaudeCodeEvents(line, base.Type)

	case "summary", "file-history-snapshot", "queue-operation", "init", "system":
		// Known non-event types in Claude Code JSONL - skip silently
		return nil, nil

	default:
		// Unknown event types - skip with warning
		return nil, fmt.Errorf("unknown event type: %s", base.Type)
	}
}

// ClaudeCodeEvent represents the actual Claude Code JSONL format
type ClaudeCodeEvent struct {
	Type       string          `json:"type"`
	Timestamp  string          `json:"timestamp"`
	SessionID  string          `json:"sessionId"`
	AgentID    string          `json:"agentId,omitempty"`
	AgentType  string          `json:"agentType,omitempty"`
	IsSidechain bool           `json:"isSidechain,omitempty"`
	Message    json.RawMessage `json:"message"`
}

// ClaudeMessage represents the message structure in Claude Code JSONL
type ClaudeMessage struct {
	Role    string            `json:"role"`
	Model   string            `json:"model,omitempty"`
	Content json.RawMessage   `json:"content"`
	Usage   *ClaudeUsage      `json:"usage,omitempty"`
}

// ClaudeUsage represents token usage in Claude Code format
type ClaudeUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

// ToolUseBlock represents a tool_use block in Claude content
type ToolUseBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// parseClaudeCodeEvents extracts ALL events from actual Claude Code JSONL format
// Returns multiple events: all tool_use blocks + token_usage if present
func parseClaudeCodeEvents(line string, eventType string) ([]Event, error) {
	var event ClaudeCodeEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return nil, fmt.Errorf("failed to parse Claude Code event: %w", err)
	}

	// Parse timestamp
	ts, _ := time.Parse(time.RFC3339, event.Timestamp)

	// Parse message
	var msg ClaudeMessage
	if err := json.Unmarshal(event.Message, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	var events []Event

	// For assistant messages, extract ALL tool calls AND token usage
	if eventType == "assistant" {
		// Extract ALL tool_use blocks from content
		toolEvents := extractAllToolsFromContent(msg.Content, ts, msg.Model)
		events = append(events, toolEvents...)

		// Also extract token usage (always include if present)
		if msg.Usage != nil && (msg.Usage.InputTokens > 0 || msg.Usage.OutputTokens > 0) {
			events = append(events, &TokenUsageEvent{
				BaseEvent: BaseEvent{
					Type:      "token_usage",
					Timestamp: ts,
				},
				InputTokens:  msg.Usage.InputTokens,
				OutputTokens: msg.Usage.OutputTokens,
				ModelName:    msg.Model,
			})
		}
	}

	// For user messages, extract tool_result information
	if eventType == "user" {
		// User messages contain tool results - extract them for completion tracking
		toolResults := extractToolResultsFromContent(msg.Content, ts)
		events = append(events, toolResults...)
	}

	return events, nil
}

// extractAllToolsFromContent extracts ALL tool_use blocks from message content
// Returns slice of Events for all tool invocations in this message
func extractAllToolsFromContent(content json.RawMessage, ts time.Time, model string) []Event {
	var events []Event

	// Try to parse as array of content blocks
	var blocks []json.RawMessage
	if err := json.Unmarshal(content, &blocks); err != nil {
		return events
	}

	for _, block := range blocks {
		var toolBlock ToolUseBlock
		if err := json.Unmarshal(block, &toolBlock); err != nil {
			continue
		}

		if toolBlock.Type == "tool_use" && toolBlock.Name != "" {
			// Convert input to map
			var params map[string]interface{}
			json.Unmarshal(toolBlock.Input, &params)

			// Check if this is a Bash command - extract command details
			if toolBlock.Name == "Bash" {
				if cmd, ok := params["command"].(string); ok {
					events = append(events, &BashCommandEvent{
						BaseEvent: BaseEvent{
							Type:      "bash_command",
							Timestamp: ts,
						},
						Command: cmd,
						Success: true, // Will be updated from tool_result
					})
					continue
				}
			}

			// Check if this is a file operation (Read, Write, Edit)
			if toolBlock.Name == "Read" || toolBlock.Name == "Write" || toolBlock.Name == "Edit" {
				filePath := ""
				if fp, ok := params["file_path"].(string); ok {
					filePath = fp
				} else if fp, ok := params["path"].(string); ok {
					filePath = fp
				}
				if filePath != "" {
					events = append(events, &FileOperationEvent{
						BaseEvent: BaseEvent{
							Type:      "file_operation",
							Timestamp: ts,
						},
						Operation: toolBlock.Name,
						Path:      filePath,
						Success:   true, // Will be updated from tool_result
					})
					continue
				}
			}

			// Generic tool call
			events = append(events, &ToolCallEvent{
				BaseEvent: BaseEvent{
					Type:      "tool_call",
					Timestamp: ts,
				},
				ToolName:   toolBlock.Name,
				Parameters: params,
				Success:    true, // Default to success
			})
		}
	}

	return events
}

// ToolResultBlock represents a tool_result block in Claude content
type ToolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

// extractToolResultsFromContent extracts tool_result blocks from user messages
// This provides completion status for tool calls
func extractToolResultsFromContent(content json.RawMessage, ts time.Time) []Event {
	var events []Event

	// Try to parse as array of content blocks
	var blocks []json.RawMessage
	if err := json.Unmarshal(content, &blocks); err != nil {
		return events
	}

	for _, block := range blocks {
		var resultBlock ToolResultBlock
		if err := json.Unmarshal(block, &resultBlock); err != nil {
			continue
		}

		if resultBlock.Type == "tool_result" {
			// Record tool result as an event for tracking success/failure
			errorMsg := ""
			if resultBlock.IsError {
				errorMsg = resultBlock.Content
			}
			events = append(events, &ToolCallEvent{
				BaseEvent: BaseEvent{
					Type:      "tool_result",
					Timestamp: ts,
				},
				ToolName: resultBlock.ToolUseID, // Use tool_use_id to correlate
				Result:   resultBlock.Content,
				Success:  !resultBlock.IsError,
				Error:    errorMsg,
			})
		}
	}

	return events
}

// isSessionMetadata checks if the line contains ONLY session metadata (not event data)
// Supports both test format (session_start/session_metadata) and real Claude Code format
func isSessionMetadata(line string) bool {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return false
	}
	// Test format - explicit metadata types
	eventType, ok := raw["type"].(string)
	if ok && (eventType == "session_start" || eventType == "session_metadata") {
		return true
	}
	// Don't treat Claude "assistant"/"user" events as metadata - they contain actual event data
	if ok && (eventType == "assistant" || eventType == "user") {
		return false
	}
	return false
}

// parseSessionMetadata extracts session metadata from a session event line
// Supports both test format and real Claude Code format
func parseSessionMetadata(line string, session *Session) error {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return fmt.Errorf("failed to parse session metadata: %w", err)
	}

	// Extract session ID (test format: session_id/id, real: sessionId)
	// Only set if not already set (we scan all lines)
	if session.ID == "" {
		if id, ok := raw["session_id"].(string); ok {
			session.ID = id
		} else if id, ok := raw["id"].(string); ok {
			session.ID = id
		} else if id, ok := raw["sessionId"].(string); ok {
			// Real Claude Code format
			session.ID = id
		}
	}

	// Extract project (test format: project, real: extract from cwd)
	// Only set if not already set
	if session.Project == "" {
		if project, ok := raw["project"].(string); ok {
			session.Project = project
		} else if cwd, ok := raw["cwd"].(string); ok {
			// Real Claude Code format - extract project from cwd path
			session.Project = extractProjectFromPath(cwd)
		}
	}

	if timestampStr, ok := raw["timestamp"].(string); ok {
		if ts, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			session.Timestamp = ts
		}
	}

	if status, ok := raw["status"].(string); ok {
		session.Status = status
	} else {
		// Default status for real Claude Code sessions
		session.Status = "active"
	}

	// Extract agent name - agentType (human-readable) takes precedence over agentId
	if agentName, ok := raw["agent_name"].(string); ok && session.AgentName == "" {
		session.AgentName = agentName
	}
	// agentType is the human-readable name - always use it if found (overrides agent-xxx)
	if agentType, ok := raw["agentType"].(string); ok && agentType != "" {
		session.AgentName = agentType
	}
	// Fallback to agentId only if no agent name yet
	if session.AgentName == "" {
		if agentID, ok := raw["agentId"].(string); ok {
			session.AgentName = "agent-" + agentID
		}
	}

	if duration, ok := raw["duration"].(float64); ok {
		session.Duration = int64(duration)
	}

	if success, ok := raw["success"].(bool); ok {
		session.Success = success
	}
	// Don't set a default - only update if the field exists

	if errorCount, ok := raw["error_count"].(float64); ok {
		session.ErrorCount = int(errorCount)
	}

	// Skip validation for real Claude sessions (they may have minimal metadata)
	if session.ID != "" {
		return nil
	}
	return session.Validate()
}

// extractProjectFromPath extracts project name from cwd path
func extractProjectFromPath(cwd string) string {
	// Get base directory name
	if cwd == "" {
		return ""
	}
	// Find last path component
	for i := len(cwd) - 1; i >= 0; i-- {
		if cwd[i] == '/' {
			return cwd[i+1:]
		}
	}
	return cwd
}
