package behavioral

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ProjectInfo represents metadata about a Claude Code project
type ProjectInfo struct {
	Name         string // Project name (directory name)
	Path         string // Full path to project directory
	SessionCount int    // Number of JSONL session files
	TotalSize    int64  // Total size of all JSONL files in bytes
}

// ProjectStats represents aggregate statistics for a project
type ProjectStats struct {
	TotalSessions int
	SuccessRate   float64
	TotalSize     int64
	LastModified  string
	ErrorRate     float64
}

// ListProjects scans ~/.claude/projects and returns information about each project
func ListProjects() ([]ProjectInfo, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	return ListProjectsWithBaseDir(filepath.Join(homeDir, ".claude", "projects"))
}

// ListProjectsWithBaseDir scans the specified base directory and returns information about each project
func ListProjectsWithBaseDir(baseDir string) ([]ProjectInfo, error) {
	// Expand ~ in path
	projectsDir := expandHomePath(baseDir)

	// Check if projects directory exists
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		return []ProjectInfo{}, nil
	}

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	var projects []ProjectInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectPath := filepath.Join(projectsDir, entry.Name())
		info := ProjectInfo{
			Name: entry.Name(),
			Path: projectPath,
		}

		// Count JSONL files and calculate total size
		sessionFiles, err := filepath.Glob(filepath.Join(projectPath, "*.jsonl"))
		if err != nil {
			continue // Skip projects with glob errors
		}

		info.SessionCount = len(sessionFiles)

		for _, file := range sessionFiles {
			fileInfo, err := os.Stat(file)
			if err == nil {
				info.TotalSize += fileInfo.Size()
			}
		}

		projects = append(projects, info)
	}

	// Sort by name
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}

// GetProjectStats returns aggregate statistics for a specific project
func GetProjectStats(projectName string) (*ProjectStats, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	return GetProjectStatsWithBaseDir(projectName, filepath.Join(homeDir, ".claude", "projects"))
}

// GetProjectStatsWithBaseDir returns aggregate statistics for a specific project using the given base directory
func GetProjectStatsWithBaseDir(projectName, baseDir string) (*ProjectStats, error) {
	projectPath := filepath.Join(expandHomePath(baseDir), projectName)

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("project not found: %s", projectName)
	}

	sessionFiles, err := filepath.Glob(filepath.Join(projectPath, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob session files: %w", err)
	}

	stats := &ProjectStats{
		TotalSessions: len(sessionFiles),
	}

	var lastModTime int64
	for _, file := range sessionFiles {
		fileInfo, err := os.Stat(file)
		if err == nil {
			stats.TotalSize += fileInfo.Size()
			modTime := fileInfo.ModTime().Unix()
			if modTime > lastModTime {
				lastModTime = modTime
				stats.LastModified = fileInfo.ModTime().Format("2006-01-02 15:04:05")
			}
		}
	}

	// TODO: Parse JSONL files to compute success/error rates
	// For now, return placeholder values
	stats.SuccessRate = 0.0
	stats.ErrorRate = 0.0

	return stats, nil
}

// expandHomePath expands ~ to the user's home directory
func expandHomePath(path string) string {
	if len(path) == 0 {
		return path
	}
	if path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[1:])
	}
	return path
}
