package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetConductorHome returns the conductor home directory
// Priority order:
//   1. CONDUCTOR_HOME environment variable (if set)
//   2. ~/.conductor (default)
// The directory is created if it doesn't exist
func GetConductorHome() (string, error) {
	// Try env var first
	if home := os.Getenv("CONDUCTOR_HOME"); home != "" {
		return home, nil
	}

	// Fall back to ~/.conductor
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get user home directory: %w", err)
	}

	conductorHome := filepath.Join(homeDir, ".conductor")

	// Ensure directory exists
	if err := os.MkdirAll(conductorHome, 0755); err != nil {
		return "", fmt.Errorf("create conductor home directory: %w", err)
	}

	return conductorHome, nil
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
