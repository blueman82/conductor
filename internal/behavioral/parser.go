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
	lineNum := 0
	sessionSeen := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if line == "" {
			continue
		}

		// Check if this is session metadata
		if isSessionMetadata(line) {
			if err := parseSessionMetadata(line, &sessionData.Session); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse session metadata at line %d: %v\n", lineNum, err)
			} else {
				sessionSeen = true
			}
			continue
		}

		event, err := parseEventLine(line)
		if err != nil {
			// Log warning but continue parsing - graceful degradation
			fmt.Fprintf(os.Stderr, "Warning: skipping malformed event at line %d: %v\n", lineNum, err)
			continue
		}

		sessionData.Events = append(sessionData.Events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading session file: %w", err)
	}

	// Validate session metadata was found
	if !sessionSeen || sessionData.Session.ID == "" {
		return nil, errors.New("session metadata not found in JSONL file")
	}

	return sessionData, nil
}

// parseEventLine parses a single JSONL line into an Event
func parseEventLine(line string) (Event, error) {
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
		return &event, nil

	case "bash_command":
		var event BashCommandEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("failed to parse bash command event: %w", err)
		}
		if err := event.Validate(); err != nil {
			return nil, err
		}
		return &event, nil

	case "file_operation":
		var event FileOperationEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("failed to parse file operation event: %w", err)
		}
		if err := event.Validate(); err != nil {
			return nil, err
		}
		return &event, nil

	case "token_usage":
		var event TokenUsageEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("failed to parse token usage event: %w", err)
		}
		if err := event.Validate(); err != nil {
			return nil, err
		}
		return &event, nil

	default:
		return nil, fmt.Errorf("unknown event type: %s", base.Type)
	}
}

// isSessionMetadata checks if the line contains session metadata
func isSessionMetadata(line string) bool {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return false
	}
	eventType, ok := raw["type"].(string)
	return ok && (eventType == "session_start" || eventType == "session_metadata")
}

// parseSessionMetadata extracts session metadata from a session event line
func parseSessionMetadata(line string, session *Session) error {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return fmt.Errorf("failed to parse session metadata: %w", err)
	}

	// Extract fields from the raw map
	if id, ok := raw["session_id"].(string); ok {
		session.ID = id
	} else if id, ok := raw["id"].(string); ok {
		session.ID = id
	}

	if project, ok := raw["project"].(string); ok {
		session.Project = project
	}

	if timestampStr, ok := raw["timestamp"].(string); ok {
		if ts, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			session.Timestamp = ts
		}
	}

	if status, ok := raw["status"].(string); ok {
		session.Status = status
	}

	if agentName, ok := raw["agent_name"].(string); ok {
		session.AgentName = agentName
	}

	if duration, ok := raw["duration"].(float64); ok {
		session.Duration = int64(duration)
	}

	if success, ok := raw["success"].(bool); ok {
		session.Success = success
	}

	if errorCount, ok := raw["error_count"].(float64); ok {
		session.ErrorCount = int(errorCount)
	}

	return session.Validate()
}
