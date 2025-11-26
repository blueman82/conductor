package behavioral

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// SessionInfo contains metadata about a discovered session file
type SessionInfo struct {
	Project   string    `json:"project"`    // Project name from directory path
	Filename  string    `json:"filename"`   // Base filename (agent-{uuid}.jsonl)
	SessionID string    `json:"session_id"` // Extracted UUID from filename
	CreatedAt time.Time `json:"created_at"` // File modification time
	FileSize  int64     `json:"file_size"`  // File size in bytes
	FilePath  string    `json:"file_path"`  // Full absolute path to file
}

var (
	// Pattern for agent session files: agent-{short-id}.jsonl (e.g., agent-000bd1b9.jsonl)
	agentFilePattern = regexp.MustCompile(`^agent-([0-9a-fA-F]+)\.jsonl$`)
	// Pattern for main session files: {uuid}.jsonl (e.g., 003c44c7-e568-46e5-8f00-c234961493cc.jsonl)
	uuidFilePattern = regexp.MustCompile(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\.jsonl$`)
)

// DiscoverSessions scans the base directory for all Claude session JSONL files
// Returns a sorted list of SessionInfo (newest first)
func DiscoverSessions(baseDir string) ([]SessionInfo, error) {
	// Expand ~ to home directory if present
	expandedDir, err := expandHomeDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to expand home directory: %w", err)
	}

	// Verify base directory exists
	if _, err := os.Stat(expandedDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("base directory does not exist: %s", expandedDir)
	}

	var sessions []SessionInfo

	// Walk the directory tree
	err = filepath.Walk(expandedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log warning but continue traversal
			fmt.Fprintf(os.Stderr, "Warning: error accessing path %s: %v\n", path, err)
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if this is a valid session file
		if !isValidSessionFile(info.Name()) {
			return nil
		}

		// Parse session info from path
		sessionInfo, err := parseSessionPath(path, info)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse session path %s: %v\n", path, err)
			return nil
		}

		sessions = append(sessions, *sessionInfo)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory tree: %w", err)
	}

	// Sort by creation time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

// DiscoverProjectSessions discovers sessions for a specific project
func DiscoverProjectSessions(baseDir, project string) ([]SessionInfo, error) {
	// Expand ~ to home directory if present
	expandedDir, err := expandHomeDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to expand home directory: %w", err)
	}

	projectDir := filepath.Join(expandedDir, project)

	// Verify project directory exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("project directory does not exist: %s", projectDir)
	}

	var sessions []SessionInfo

	// Read project directory
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read project directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if this is a valid session file
		if !isValidSessionFile(entry.Name()) {
			continue
		}

		fullPath := filepath.Join(projectDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get file info for %s: %v\n", fullPath, err)
			continue
		}

		// Parse session info from path
		sessionInfo, err := parseSessionPath(fullPath, info)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse session path %s: %v\n", fullPath, err)
			continue
		}

		sessions = append(sessions, *sessionInfo)
	}

	// Sort by creation time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

// isValidSessionFile checks if filename matches the agent-{uuid}.jsonl pattern
func isValidSessionFile(filename string) bool {
	// Match either agent-{id}.jsonl or {uuid}.jsonl patterns
	return agentFilePattern.MatchString(filename) || uuidFilePattern.MatchString(filename)
}

// extractSessionID extracts the session ID from a session filename
// Handles both agent-{id}.jsonl and {uuid}.jsonl patterns
func extractSessionID(filename string) (string, error) {
	// Try agent pattern first (agent-000bd1b9.jsonl)
	if matches := agentFilePattern.FindStringSubmatch(filename); len(matches) == 2 {
		return matches[1], nil
	}
	// Try UUID pattern ({uuid}.jsonl)
	if matches := uuidFilePattern.FindStringSubmatch(filename); len(matches) == 2 {
		return matches[1], nil
	}
	return "", fmt.Errorf("filename does not match session pattern: %s", filename)
}

// parseSessionPath parses a full file path into SessionInfo
func parseSessionPath(path string, info os.FileInfo) (*SessionInfo, error) {
	// Extract session ID from filename
	sessionID, err := extractSessionID(info.Name())
	if err != nil {
		return nil, err
	}

	// Extract project name from directory structure
	// Expected structure: {baseDir}/projects/{project}/agent-{uuid}.jsonl
	// or: {baseDir}/{project}/agent-{uuid}.jsonl
	dir := filepath.Dir(path)
	project := filepath.Base(dir)

	// Handle case where parent is "projects" directory
	parentDir := filepath.Dir(dir)
	if filepath.Base(parentDir) == "projects" {
		// Already have project name from dir
	} else if strings.HasSuffix(parentDir, ".claude") {
		// Direct child of .claude directory
		project = filepath.Base(dir)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &SessionInfo{
		Project:   project,
		Filename:  info.Name(),
		SessionID: sessionID,
		CreatedAt: info.ModTime(),
		FileSize:  info.Size(),
		FilePath:  absPath,
	}, nil
}

// expandHomeDir expands ~ to the user's home directory
func expandHomeDir(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	if path == "~" {
		return homeDir, nil
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:]), nil
	}

	return path, nil
}
