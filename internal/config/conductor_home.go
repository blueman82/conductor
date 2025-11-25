package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// buildTimeRepoRoot is set by cmd during initialization
// It contains the conductor repository root injected at build time
var buildTimeRepoRoot string

// SetBuildTimeRepoRoot sets the conductor repository root (injected at build time)
// This should be called early during app initialization (from cmd)
func SetBuildTimeRepoRoot(root string) {
	buildTimeRepoRoot = root
}

// GetConductorHome returns the conductor home directory
// Priority order:
//  1. CONDUCTOR_HOME environment variable (if set)
//  2. Conductor repository root (injected at build time)
//
// If neither is available, returns an error (NO fallback to cwd)
// The directory is created if it doesn't exist
func GetConductorHome() (string, error) {
	return GetConductorHomeWithRoot(buildTimeRepoRoot)
}

// GetConductorHomeWithRoot is the testable version that accepts the build-time root
// Priority order:
//  1. CONDUCTOR_HOME environment variable (if set)
//  2. Build-time injected conductor repository root
//
// Returns error if neither is available
func GetConductorHomeWithRoot(buildTimeRoot string) (string, error) {
	// Try env var first
	if home := os.Getenv("CONDUCTOR_HOME"); home != "" {
		// Ensure directory exists
		if err := os.MkdirAll(home, 0755); err != nil {
			return "", fmt.Errorf("create conductor home directory: %w", err)
		}
		return home, nil
	}

	// Try build-time injected root
	if buildTimeRoot != "" {
		conductorHome := filepath.Join(buildTimeRoot, ".conductor")
		// Ensure directory exists
		if err := os.MkdirAll(conductorHome, 0755); err != nil {
			return "", fmt.Errorf("create conductor home directory: %w", err)
		}
		return conductorHome, nil
	}

	// No configuration available
	return "", fmt.Errorf("conductor home not configured: set CONDUCTOR_HOME environment variable or rebuild with conductor repo path injected")
}

// findConductorRepoRoot finds the conductor repository root by looking for
// go.mod containing the conductor module path, or a .conductor-root marker
func findConductorRepoRoot() (string, error) {
	// Start from current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	current := cwd
	for {
		// First check for .conductor-root marker file (highest priority)
		markerPath := filepath.Join(current, ".conductor-root")
		if _, err := os.Stat(markerPath); err == nil {
			return current, nil
		}

		// Check for go.mod with conductor module path
		goModPath := filepath.Join(current, "go.mod")
		if data, err := os.ReadFile(goModPath); err == nil {
			// Check if this go.mod contains the conductor module path
			if contains(string(data), "github.com/harrison/conductor") {
				return current, nil
			}
		}

		// Move up one directory
		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			break
		}
		current = parent
	}

	return "", fmt.Errorf("conductor repository root not found (looking for .conductor-root or go.mod with github.com/harrison/conductor)")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && indexOf(s, substr) >= 0)
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// GetLearningDBPath returns the absolute path to the learning database
// Priority order:
//  1. Config file's learning.db_path (if absolute path)
//  2. Config file's learning.db_path resolved relative to cwd (if relative)
//  3. Falls back to $CONDUCTOR_HOME/learning/executions.db
func GetLearningDBPath() (string, error) {
	return GetLearningDBPathWithRoot(buildTimeRepoRoot)
}

// GetLearningDBPathWithRoot is the testable version that accepts the build-time root
// When buildTimeRoot is provided (for testing), it's used directly without config lookup
func GetLearningDBPathWithRoot(buildTimeRoot string) (string, error) {
	// If buildTimeRoot is explicitly provided (testing), use it directly
	if buildTimeRoot != "" {
		home, err := GetConductorHomeWithRoot(buildTimeRoot)
		if err != nil {
			return "", err
		}
		dbPath := filepath.Join(home, "learning", "executions.db")
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("create learning directory: %w", err)
		}
		return dbPath, nil
	}

	// Check CONDUCTOR_HOME env var first
	if envHome := os.Getenv("CONDUCTOR_HOME"); envHome != "" {
		dbPath := filepath.Join(envHome, "learning", "executions.db")
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("create learning directory: %w", err)
		}
		return dbPath, nil
	}

	// Try to load config to get learning.db_path
	cwd, _ := os.Getwd()
	configPath := filepath.Join(cwd, ".conductor", "config.yaml")

	// Check if config file exists and has db_path set
	if cfg, err := LoadConfig(configPath); err == nil && cfg.Learning.DBPath != "" {
		dbPath := cfg.Learning.DBPath

		// Expand ~ to home directory
		if len(dbPath) > 0 && dbPath[0] == '~' {
			if home, err := os.UserHomeDir(); err == nil {
				dbPath = filepath.Join(home, dbPath[1:])
			}
		}

		// If absolute path, use it directly
		if filepath.IsAbs(dbPath) {
			// Ensure parent directory exists
			dir := filepath.Dir(dbPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", fmt.Errorf("create learning directory: %w", err)
			}
			return dbPath, nil
		}

		// Relative path - resolve from cwd
		absPath := filepath.Join(cwd, dbPath)
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("create learning directory: %w", err)
		}
		return absPath, nil
	}

	// No config found - return error
	return "", fmt.Errorf("learning database path not configured: set CONDUCTOR_HOME or create .conductor/config.yaml with learning.db_path")
}

// GetLearningDir returns the learning directory path
func GetLearningDir() (string, error) {
	return GetLearningDirWithRoot(buildTimeRepoRoot)
}

// GetLearningDirWithRoot is the testable version that accepts the build-time root
func GetLearningDirWithRoot(buildTimeRoot string) (string, error) {
	home, err := GetConductorHomeWithRoot(buildTimeRoot)
	if err != nil {
		return "", err
	}

	learningDir := filepath.Join(home, "learning")

	// Ensure directory exists
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		return "", fmt.Errorf("create learning directory: %w", err)
	}

	return learningDir, nil
}
