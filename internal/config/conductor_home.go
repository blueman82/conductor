package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetConductorHome returns the conductor home directory
// Priority order:
//   1. CONDUCTOR_HOME environment variable (if set)
//   2. Conductor repository root (detected by finding go.mod)
//   3. Current working directory (fallback)
// The directory is created if it doesn't exist
func GetConductorHome() (string, error) {
	// Try env var first
	if home := os.Getenv("CONDUCTOR_HOME"); home != "" {
		return home, nil
	}

	// Try to find conductor repo root by looking for go.mod
	repoRoot, err := findConductorRepoRoot()
	if err == nil && repoRoot != "" {
		conductorHome := filepath.Join(repoRoot, ".conductor")
		// Ensure directory exists
		if err := os.MkdirAll(conductorHome, 0755); err != nil {
			return "", fmt.Errorf("create conductor home directory: %w", err)
		}
		return conductorHome, nil
	}

	// Fallback to current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	conductorHome := filepath.Join(cwd, ".conductor")

	// Ensure directory exists
	if err := os.MkdirAll(conductorHome, 0755); err != nil {
		return "", fmt.Errorf("create conductor home directory: %w", err)
	}

	return conductorHome, nil
}

// findConductorRepoRoot finds the conductor repository root by looking for
// the cmd/conductor/main.go file, which uniquely identifies the conductor repo
func findConductorRepoRoot() (string, error) {
	// Start from current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for cmd/conductor/main.go
	current := cwd
	for {
		conductorMainPath := filepath.Join(current, "cmd", "conductor", "main.go")
		if _, err := os.Stat(conductorMainPath); err == nil {
			// Found conductor's main.go - this is the repo root
			return current, nil
		}

		// Move up one directory
		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			break
		}
		current = parent
	}

	return "", fmt.Errorf("conductor repository root not found (looking for cmd/conductor/main.go)")
}

// GetLearningDBPath returns the absolute path to the learning database
// Always returns: $CONDUCTOR_HOME/learning/executions.db
func GetLearningDBPath() (string, error) {
	home, err := GetConductorHome()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, "learning", "executions.db"), nil
}

// GetLearningDir returns the learning directory path
func GetLearningDir() (string, error) {
	home, err := GetConductorHome()
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
