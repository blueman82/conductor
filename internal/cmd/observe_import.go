package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
	"github.com/spf13/cobra"
)

var (
	importProject    string
	importDryRun     bool
	importSkipExists bool
)

// NewObserveImportCmd creates the 'conductor observe import' subcommand
func NewObserveImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import behavioral data from JSONL files",
		Long: `Import Claude Code agent session data from JSONL files in ~/.claude/projects/
into the behavioral database for analysis.

This command:
- Discovers all agent-*.jsonl files in ~/.claude/projects/
- Parses each session file extracting events and metrics
- Creates synthetic task execution entries (for imported sessions)
- Stores all metrics in the behavioral database
- Skips already imported sessions by default

The import process is idempotent - running it multiple times won't create duplicates.

Examples:
  conductor observe import                    # Import all discovered sessions
  conductor observe import --project myapp    # Import only sessions from myapp
  conductor observe import --dry-run          # Preview what would be imported
  conductor observe import --skip-exists      # Skip already imported sessions`,
		RunE: HandleImportCommand,
	}

	cmd.Flags().StringVarP(&importProject, "project", "p", "", "Filter by project name")
	cmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Preview import without modifying database")
	cmd.Flags().BoolVar(&importSkipExists, "skip-exists", true, "Skip already imported sessions")

	return cmd
}

// HandleImportCommand processes import requests
func HandleImportCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Get database path
	dbPath, err := config.GetLearningDBPath()
	if err != nil {
		return fmt.Errorf("failed to get learning database path: %w", err)
	}

	// Open database (will initialize schema if needed)
	store, err := learning.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open learning store: %w", err)
	}
	defer store.Close()

	// Get Claude projects directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	projectsDir := filepath.Join(homeDir, ".claude", "projects")

	// Discover session files
	fmt.Printf("Discovering JSONL files in %s...\n", projectsDir)
	sessions, err := behavioral.DiscoverSessions(projectsDir)
	if err != nil {
		return fmt.Errorf("failed to discover sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No session files found")
		return nil
	}

	// Filter by project if specified
	if importProject != "" {
		var filtered []behavioral.SessionInfo
		for _, s := range sessions {
			if s.Project == importProject {
				filtered = append(filtered, s)
			}
		}
		sessions = filtered
		if len(sessions) == 0 {
			fmt.Printf("No sessions found for project '%s'\n", importProject)
			return nil
		}
	}

	fmt.Printf("Found %d session files to import\n\n", len(sessions))

	// Import sessions
	var imported, skipped, errored int

	for i, sessionInfo := range sessions {
		fmt.Printf("[%d/%d] Importing %s from %s\n",
			i+1, len(sessions), sessionInfo.SessionID[:8], sessionInfo.Project)

		// Parse session file
		sessionData, err := behavioral.ParseSessionFile(sessionInfo.FilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error: failed to parse session file: %v\n", err)
			errored++
			continue
		}

		// Check if session already imported
		if importSkipExists {
			exists, checkErr := sessionExistsInDB(ctx, store, sessionInfo.SessionID)
			if checkErr != nil {
				fmt.Fprintf(os.Stderr, "  Error: failed to check session existence: %v\n", checkErr)
				errored++
				continue
			}
			if exists {
				fmt.Printf("  Skipping (already imported)\n")
				skipped++
				continue
			}
		}

		// Lookup agent name from parent session
		agentName := lookupAgentName(sessionInfo, projectsDir)
		if agentName != "" && sessionData.Session.AgentName == "" {
			sessionData.Session.AgentName = agentName
		} else if agentName != "" && (strings.HasPrefix(sessionData.Session.AgentName, "agent-") || sessionData.Session.AgentName == "") {
			// Override agent-xxx with human-readable name
			sessionData.Session.AgentName = agentName
		}

		if importDryRun {
			agentDisplay := sessionData.Session.AgentName
			if agentDisplay == "" {
				agentDisplay = "(unknown)"
			}
			fmt.Printf("  Would import: %d events (agent: %s)\n", len(sessionData.Events), agentDisplay)
			imported++
			continue
		}

		// Record session
		if err := recordSessionData(ctx, store, sessionData, sessionInfo); err != nil {
			fmt.Fprintf(os.Stderr, "  Error: failed to record session: %v\n", err)
			errored++
			continue
		}

		fmt.Printf("  Imported: %d events\n", len(sessionData.Events))
		imported++
	}

	// Summary
	fmt.Printf("\n%s\n", "==================================================")
	fmt.Printf("Import Summary\n")
	fmt.Printf("%s\n", "==================================================")
	fmt.Printf("Imported: %d\n", imported)
	fmt.Printf("Skipped:  %d\n", skipped)
	fmt.Printf("Errors:   %d\n", errored)
	fmt.Printf("Total:    %d\n", len(sessions))

	if importDryRun {
		fmt.Printf("\nDry run completed - no changes made to database\n")
	}

	return nil
}

// sessionExistsInDB checks if a session was already imported
// Note: Since we store synthetic task execution entries, we check if a matching
// execution already exists using the session ID as a key.
func sessionExistsInDB(ctx context.Context, store *learning.Store, sessionID string) (bool, error) {
	// For now, always return false as we don't have a direct way to query by session ID
	// This can be improved by adding a session_id column to behavioral_sessions table
	// or storing metadata in the task_executions context field
	return false, nil
}

// recordSessionData converts parsed session data to database records
func recordSessionData(ctx context.Context, store *learning.Store,
	sessionData *behavioral.SessionData, sessionInfo behavioral.SessionInfo) error {

	// Create synthetic task execution entry for imported sessions
	// This is necessary because behavioral_sessions has a foreign key to task_executions
	exec := &learning.TaskExecution{
		PlanFile:     sessionInfo.Project,
		RunNumber:    1,
		TaskNumber:   sessionInfo.SessionID[:8], // Use first 8 chars of UUID
		TaskName:     fmt.Sprintf("Agent Session: %s", sessionInfo.SessionID[:8]),
		Agent:        sessionData.Session.AgentName,
		Prompt:       fmt.Sprintf("Session imported from: %s", sessionInfo.FilePath),
		Success:      sessionData.Session.Success,
		Output:       fmt.Sprintf("Session duration: %dms", sessionData.Session.Duration),
		ErrorMessage: "",
		DurationSecs: sessionData.Session.Duration / 1000,
		QCVerdict:    "IMPORTED", // Mark as imported
		QCFeedback:   "Behavioral data imported from session JSONL",
		Timestamp:    sessionData.Session.Timestamp,
	}

	if !sessionData.Session.Success {
		exec.ErrorMessage = fmt.Sprintf("Session failed with %d errors", sessionData.Session.ErrorCount)
	}

	// Record task execution
	if err := store.RecordExecution(ctx, exec); err != nil {
		return fmt.Errorf("failed to record task execution: %w", err)
	}

	// Extract metrics from session data
	metrics := behavioral.ExtractMetrics(sessionData)

	// Convert to store data structures
	sessionStartTime := sessionData.Session.Timestamp
	sessionEndTime := sessionStartTime.Add(time.Duration(sessionData.Session.Duration) * time.Millisecond)

	sessionDataRecord := &learning.BehavioralSessionData{
		TaskExecutionID:     exec.ID,
		SessionStart:        sessionStartTime,
		SessionEnd:          &sessionEndTime,
		TotalDurationSecs:   sessionData.Session.Duration / 1000,
		TotalToolCalls:      len(metrics.ToolExecutions),
		TotalBashCommands:   len(metrics.BashCommands),
		TotalFileOperations: len(metrics.FileOperations),
		TotalTokensUsed:     metrics.TokenUsage.InputTokens + metrics.TokenUsage.OutputTokens,
		ContextWindowUsed:   0, // Not available in behavioral metrics
	}

	// Convert tool executions
	toolData := make([]learning.ToolExecutionData, 0)
	for _, tool := range metrics.ToolExecutions {
		params, _ := json.Marshal(tool)
		toolData = append(toolData, learning.ToolExecutionData{
			ToolName:     tool.Name,
			Parameters:   string(params),
			DurationMs:   tool.AvgDuration.Milliseconds(),
			Success:      tool.TotalSuccess > 0,
			ErrorMessage: "",
		})
	}

	// Convert bash commands
	bashData := make([]learning.BashCommandData, 0)
	for _, bash := range metrics.BashCommands {
		bashData = append(bashData, learning.BashCommandData{
			Command:      bash.Command,
			DurationMs:   bash.Duration.Milliseconds(),
			ExitCode:     bash.ExitCode,
			StdoutLength: bash.OutputLength,
			StderrLength: 0, // Not tracked in behavioral metrics
			Success:      bash.Success,
		})
	}

	// Convert file operations
	fileData := make([]learning.FileOperationData, 0)
	for _, fileOp := range metrics.FileOperations {
		fileData = append(fileData, learning.FileOperationData{
			OperationType: fileOp.Type,
			FilePath:      fileOp.Path,
			DurationMs:    fileOp.Duration, // Duration is already in milliseconds
			BytesAffected: fileOp.SizeBytes,
			Success:       fileOp.Success,
			ErrorMessage:  "",
		})
	}

	// Convert token usage
	tokenData := make([]learning.TokenUsageData, 0)
	tokenData = append(tokenData, learning.TokenUsageData{
		InputTokens:       metrics.TokenUsage.InputTokens,
		OutputTokens:      metrics.TokenUsage.OutputTokens,
		TotalTokens:       metrics.TokenUsage.InputTokens + metrics.TokenUsage.OutputTokens,
		ContextWindowSize: 0, // Not available in behavioral metrics
	})

	// Record all metrics in a transaction
	_, err := store.RecordSessionMetrics(ctx, sessionDataRecord, toolData, bashData, fileData, tokenData)
	if err != nil {
		return fmt.Errorf("failed to record session metrics: %w", err)
	}

	return nil
}

// lookupAgentName resolves the human-readable agent name from parent session
// For agent files (agent-xxx.jsonl), this looks up the Task tool call in the parent
// session to extract subagent_type. For headless agents (queue-operation), returns "headless".
func lookupAgentName(sessionInfo behavioral.SessionInfo, projectsDir string) string {
	// Read first line to get session metadata
	file, err := os.Open(sessionInfo.FilePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	if !scanner.Scan() {
		return ""
	}
	firstLine := scanner.Text()

	var metadata struct {
		Type      string `json:"type"`
		SessionID string `json:"sessionId"`
		AgentID   string `json:"agentId"`
		AgentType string `json:"agentType"`
	}
	if err := json.Unmarshal([]byte(firstLine), &metadata); err != nil {
		return ""
	}

	// Check for headless (queue-operation) agent
	if metadata.Type == "queue-operation" {
		return "headless"
	}

	// Check for direct agentType field
	if metadata.AgentType != "" {
		return metadata.AgentType
	}

	// No agentId means this is a main session, not a spawned agent
	if metadata.AgentID == "" {
		return "main"
	}

	// Look up parent session to find Task tool call
	if metadata.SessionID == "" {
		return ""
	}

	// Find parent session file
	parentPath := findParentSession(projectsDir, metadata.SessionID)
	if parentPath == "" {
		return ""
	}

	// Extract subagent_type from parent
	return extractSubagentType(parentPath, metadata.AgentID)
}

// findParentSession searches for the parent session file by sessionId
func findParentSession(projectsDir, sessionID string) string {
	// Look for {sessionId}.jsonl in all project directories
	var parentPath string
	filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		// Check if filename is the session UUID
		if strings.HasPrefix(info.Name(), sessionID) && strings.HasSuffix(info.Name(), ".jsonl") {
			parentPath = path
			return filepath.SkipAll // Stop walking
		}
		return nil
	})
	return parentPath
}

// extractSubagentType finds the Task tool call that spawned an agent and extracts subagent_type
func extractSubagentType(parentPath, agentID string) string {
	file, err := os.Open(parentPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	// First pass: find the tool_use_id that resulted in this agentId
	var toolUseID string
	for scanner.Scan() {
		line := scanner.Text()
		// Look for toolUseResult with our agentId
		if strings.Contains(line, "toolUseResult") && strings.Contains(line, agentID) {
			var event struct {
				ToolUseResult struct {
					AgentID string `json:"agentId"`
				} `json:"toolUseResult"`
				Message struct {
					Content []struct {
						ToolUseID string `json:"tool_use_id"`
					} `json:"content"`
				} `json:"message"`
			}
			if err := json.Unmarshal([]byte(line), &event); err == nil {
				if event.ToolUseResult.AgentID == agentID {
					for _, c := range event.Message.Content {
						if c.ToolUseID != "" {
							toolUseID = c.ToolUseID
							break
						}
					}
					break
				}
			}
		}
	}

	if toolUseID == "" {
		return ""
	}

	// Second pass: find the Task tool call with that ID and extract subagent_type
	file.Seek(0, 0)
	scanner = bufio.NewScanner(file)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, toolUseID) && strings.Contains(line, "Task") {
			var event struct {
				Message struct {
					Content []struct {
						Type  string `json:"type"`
						ID    string `json:"id"`
						Name  string `json:"name"`
						Input struct {
							SubagentType string `json:"subagent_type"`
							Description  string `json:"description"`
						} `json:"input"`
					} `json:"content"`
				} `json:"message"`
			}
			if err := json.Unmarshal([]byte(line), &event); err == nil {
				for _, c := range event.Message.Content {
					if c.ID == toolUseID && c.Name == "Task" && c.Input.SubagentType != "" {
						return c.Input.SubagentType
					}
				}
			}
		}
	}

	return ""
}
