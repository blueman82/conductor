package behavioral

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LiveWatcher watches JSONL files for new content and emits events
type LiveWatcher struct {
	rootDir         string
	project         string
	positions       map[string]int64 // file -> byte offset
	seenFiles       map[string]bool
	eventChan       chan Event
	pollInterval    time.Duration
	initialScanDone bool // tracks if initial scan is complete
	mu              sync.RWMutex
}

// NewLiveWatcher creates a new file watcher for JSONL files
func NewLiveWatcher(rootDir, project string) *LiveWatcher {
	return &LiveWatcher{
		rootDir:      rootDir,
		project:      project,
		positions:    make(map[string]int64),
		seenFiles:    make(map[string]bool),
		eventChan:    make(chan Event, 1000),
		pollInterval: 500 * time.Millisecond,
	}
}

// SetPollInterval sets the polling interval for file checks
func (w *LiveWatcher) SetPollInterval(d time.Duration) {
	w.pollInterval = d
}

// Events returns the channel of parsed events
func (w *LiveWatcher) Events() <-chan Event {
	return w.eventChan
}

// Start begins watching for new JSONL content
// Blocks until context is cancelled
func (w *LiveWatcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	defer close(w.eventChan)

	// Initial scan - skip to end of all existing files
	if err := w.scanFiles(ctx); err != nil {
		return fmt.Errorf("initial scan failed: %w", err)
	}

	// Mark initial scan complete - new files discovered after this will be shown from beginning
	w.mu.Lock()
	w.initialScanDone = true
	w.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.scanFiles(ctx); err != nil {
				// Log but continue - don't fail on transient errors
				fmt.Fprintf(os.Stderr, "Warning: scan error: %v\n", err)
			}
		}
	}
}

// scanFiles discovers and processes all matching JSONL files
func (w *LiveWatcher) scanFiles(ctx context.Context) error {
	pattern := w.buildPattern()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob failed: %w", err)
	}

	for _, file := range matches {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		w.mu.Lock()
		isNewFile := !w.seenFiles[file]
		initialDone := w.initialScanDone
		if isNewFile {
			w.seenFiles[file] = true
			if !initialDone {
				// During initial scan: skip to end of existing files (don't show historical)
				info, err := os.Stat(file)
				if err == nil {
					w.positions[file] = info.Size()
				} else {
					w.positions[file] = 0
				}
			} else {
				// After initial scan: new file discovered, show from beginning
				w.positions[file] = 0

				// Emit session start event for new files discovered after initial scan
				w.emitSessionStart(ctx, file)
			}
		}
		w.mu.Unlock()

		if err := w.processFile(ctx, file); err != nil {
			// Log but continue with other files
			fmt.Fprintf(os.Stderr, "Warning: failed to process %s: %v\n", file, err)
		}
	}

	return nil
}

// emitSessionStart sends a SessionStartEvent for a newly discovered file
func (w *LiveWatcher) emitSessionStart(ctx context.Context, filePath string) {
	// Extract session ID from filename (e.g., "agent-abc123.jsonl" -> "abc123" or "uuid.jsonl" -> "uuid")
	filename := filepath.Base(filePath)
	agentID := filename[:len(filename)-6] // Remove .jsonl
	if len(agentID) > 6 && agentID[:6] == "agent-" {
		agentID = agentID[6:]
	}

	// Extract project name from path
	projectName := filepath.Base(filepath.Dir(filePath))
	projectDir := filepath.Dir(filePath)

	// Try to detect agent type
	agentType := w.detectAgentType(filePath, projectDir, agentID)

	event := &SessionStartEvent{
		BaseEvent: BaseEvent{
			Type:      "session_start",
			Timestamp: time.Now(),
		},
		SessionID:   agentID,
		AgentType:   agentType,
		ProjectName: projectName,
		FilePath:    filePath,
	}

	select {
	case <-ctx.Done():
		return
	case w.eventChan <- event:
	}
}

// detectAgentType tries to find the agent type from the file or parent session
func (w *LiveWatcher) detectAgentType(filePath, projectDir, agentID string) string {
	// Read first line of agent file
	f, err := os.Open(filePath)
	if err != nil {
		return "unknown"
	}
	defer f.Close()

	buf := make([]byte, 8192)
	n, _ := f.Read(buf)
	if n == 0 {
		return "unknown"
	}

	// Find first complete JSON line
	lineEnd := 0
	for i := 0; i < n; i++ {
		if buf[i] == '\n' {
			lineEnd = i
			break
		}
	}
	if lineEnd == 0 {
		lineEnd = n
	}

	var data map[string]interface{}
	if err := json.Unmarshal(buf[:lineEnd], &data); err != nil {
		return "unknown"
	}

	// Check for agentType field directly
	if at, ok := data["agentType"].(string); ok && at != "" {
		return at
	}

	// Check if this is a headless session (queue-operation)
	if t, ok := data["type"].(string); ok && t == "queue-operation" {
		return "headless"
	}

	// Try to look up parent session for subagent_type
	parentSessionID, ok := data["sessionId"].(string)
	if !ok || parentSessionID == "" {
		return "unknown"
	}

	// Find parent session file
	parentPath := filepath.Join(projectDir, parentSessionID+".jsonl")
	parentFile, err := os.Open(parentPath)
	if err != nil {
		return "unknown"
	}
	defer parentFile.Close()

	// Search parent file for Task tool call that spawned this agent
	// Look for lines containing this agentId in toolUseResult
	scanner := bufio.NewScanner(parentFile)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for long lines

	for scanner.Scan() {
		line := scanner.Bytes()

		// Quick check - does this line mention our agent ID?
		if !bytes.Contains(line, []byte(agentID)) {
			continue
		}

		var parentData map[string]interface{}
		if err := json.Unmarshal(line, &parentData); err != nil {
			continue
		}

		// Check for toolUseResult with our agentId
		if result, ok := parentData["toolUseResult"].(map[string]interface{}); ok {
			if resultAgentID, ok := result["agentId"].(string); ok && resultAgentID == agentID {
				// Found the result - now find the tool_use_id
				if toolUseID, ok := result["tool_use_id"].(string); ok {
					// Search for the original Task call with this ID
					agentType := w.findTaskSubagentType(parentPath, toolUseID)
					if agentType != "" {
						return agentType
					}
				}
			}
		}
	}

	return "unknown"
}

// findTaskSubagentType searches a session file for a Task tool call with the given ID
func (w *LiveWatcher) findTaskSubagentType(sessionPath, toolUseID string) string {
	f, err := os.Open(sessionPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()

		// Quick check for the tool use ID
		if !bytes.Contains(line, []byte(toolUseID)) {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal(line, &data); err != nil {
			continue
		}

		// Look for message.content[] with the Task tool
		msg, ok := data["message"].(map[string]interface{})
		if !ok {
			continue
		}

		content, ok := msg["content"].([]interface{})
		if !ok {
			continue
		}

		for _, item := range content {
			toolUse, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			// Check if this is our Task tool call
			if id, ok := toolUse["id"].(string); ok && id == toolUseID {
				if name, ok := toolUse["name"].(string); ok && name == "Task" {
					if input, ok := toolUse["input"].(map[string]interface{}); ok {
						if subagentType, ok := input["subagent_type"].(string); ok {
							return subagentType
						}
					}
				}
			}
		}
	}

	return ""
}

// buildPattern constructs the glob pattern for JSONL files
func (w *LiveWatcher) buildPattern() string {
	if w.project != "" {
		// Project-specific: rootDir/project/*.jsonl (handles both agent-*.jsonl and *.jsonl)
		return filepath.Join(w.rootDir, w.project, "*.jsonl")
	}
	// All projects: rootDir/*/*.jsonl
	return filepath.Join(w.rootDir, "*", "*.jsonl")
}

// processFile reads new content from a single file
func (w *LiveWatcher) processFile(ctx context.Context, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open failed: %w", err)
	}
	defer f.Close()

	// Get file size
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat failed: %w", err)
	}

	w.mu.RLock()
	pos := w.positions[path]
	w.mu.RUnlock()

	// Nothing new to read
	if info.Size() <= pos {
		return nil
	}

	// Seek to last position
	if pos > 0 {
		if _, err := f.Seek(pos, io.SeekStart); err != nil {
			return fmt.Errorf("seek failed: %w", err)
		}
	}

	// Read new content
	newContent := make([]byte, info.Size()-pos)
	n, err := f.Read(newContent)
	if err != nil && err != io.EOF {
		return fmt.Errorf("read failed: %w", err)
	}

	// Update position
	w.mu.Lock()
	w.positions[path] = pos + int64(n)
	w.mu.Unlock()

	// Parse lines and emit events
	return w.parseAndEmit(ctx, newContent)
}

// parseAndEmit parses JSONL lines and sends events to channel
func (w *LiveWatcher) parseAndEmit(ctx context.Context, data []byte) error {
	// Split by newlines
	start := 0
	for i, b := range data {
		if b == '\n' {
			line := string(data[start:i])
			start = i + 1

			if line == "" {
				continue
			}

			events, err := parseEventLine(line)
			if err != nil {
				// Skip malformed lines
				continue
			}

			for _, event := range events {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case w.eventChan <- event:
				}
			}
		}
	}

	// Handle trailing content (incomplete line - ignore for now, will be picked up next read)
	return nil
}

// GetPositions returns a copy of current file positions (for testing/debugging)
func (w *LiveWatcher) GetPositions() map[string]int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	result := make(map[string]int64, len(w.positions))
	for k, v := range w.positions {
		result[k] = v
	}
	return result
}

// GetSeenFiles returns a copy of seen files (for testing/debugging)
func (w *LiveWatcher) GetSeenFiles() map[string]bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	result := make(map[string]bool, len(w.seenFiles))
	for k, v := range w.seenFiles {
		result[k] = v
	}
	return result
}
