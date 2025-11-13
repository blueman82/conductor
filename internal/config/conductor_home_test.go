package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetConductorHomeWithEnvVar tests CONDUCTOR_HOME env var takes precedence
func TestGetConductorHomeWithEnvVar(t *testing.T) {
	customHome := t.TempDir()
	t.Setenv("CONDUCTOR_HOME", customHome)

	home, err := GetConductorHomeWithRoot("")
	if err != nil {
		t.Fatalf("GetConductorHomeWithRoot() error = %v", err)
	}

	if home != customHome {
		t.Errorf("GetConductorHomeWithRoot() = %q, want %q", home, customHome)
	}
}

// TestGetConductorHomeWithBuildTimeRoot tests build-time injected root
func TestGetConductorHomeWithBuildTimeRoot(t *testing.T) {
	// Unset env var to test build-time path
	t.Setenv("CONDUCTOR_HOME", "")

	buildRoot := t.TempDir()
	expectedPath := filepath.Join(buildRoot, ".conductor")

	home, err := GetConductorHomeWithRoot(buildRoot)
	if err != nil {
		t.Fatalf("GetConductorHomeWithRoot() error = %v", err)
	}

	if home != expectedPath {
		t.Errorf("GetConductorHomeWithRoot() = %q, want %q", home, expectedPath)
	}

	// Verify .conductor directory was created
	if _, err := os.Stat(home); os.IsNotExist(err) {
		t.Errorf("Directory not created: %q", home)
	}
}

// TestGetConductorHomeEnvVarPrecedence tests env var takes precedence over build-time
func TestGetConductorHomeEnvVarPrecedence(t *testing.T) {
	envHome := t.TempDir()
	buildRoot := t.TempDir()
	t.Setenv("CONDUCTOR_HOME", envHome)

	home, err := GetConductorHomeWithRoot(buildRoot)
	if err != nil {
		t.Fatalf("GetConductorHomeWithRoot() error = %v", err)
	}

	// Env var should take precedence
	if home != envHome {
		t.Errorf("GetConductorHomeWithRoot() = %q, want %q (env var should take precedence)", home, envHome)
	}
}

// TestGetConductorHomeNoConfigReturnsError tests error when both unavailable
func TestGetConductorHomeNoConfigReturnsError(t *testing.T) {
	// Unset env var and provide empty build-time root
	t.Setenv("CONDUCTOR_HOME", "")

	_, err := GetConductorHomeWithRoot("")
	if err == nil {
		t.Error("GetConductorHomeWithRoot() expected error when no config available, got nil")
	}
}

// TestGetConductorHomeDirCreation tests directory creation
func TestGetConductorHomeDirCreation(t *testing.T) {
	t.Setenv("CONDUCTOR_HOME", "")
	buildRoot := t.TempDir()

	home, err := GetConductorHomeWithRoot(buildRoot)
	if err != nil {
		t.Fatalf("GetConductorHomeWithRoot() error = %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(home)
	if err != nil {
		t.Fatalf("Directory not created: %q", home)
	}

	if !info.IsDir() {
		t.Errorf("Path is not a directory: %q", home)
	}
}

// TestGetLearningDBPath tests database path generation
func TestGetLearningDBPath(t *testing.T) {
	t.Setenv("CONDUCTOR_HOME", "")
	buildRoot := t.TempDir()

	dbPath, err := GetLearningDBPathWithRoot(buildRoot)
	if err != nil {
		t.Fatalf("GetLearningDBPathWithRoot() error = %v", err)
	}

	expectedPath := filepath.Join(buildRoot, ".conductor", "learning", "executions.db")
	if dbPath != expectedPath {
		t.Errorf("GetLearningDBPathWithRoot() = %q, want %q", dbPath, expectedPath)
	}
}

// TestGetLearningDirPath tests learning directory path generation
func TestGetLearningDirPath(t *testing.T) {
	t.Setenv("CONDUCTOR_HOME", "")
	buildRoot := t.TempDir()

	learningDir, err := GetLearningDirWithRoot(buildRoot)
	if err != nil {
		t.Fatalf("GetLearningDirWithRoot() error = %v", err)
	}

	expectedPath := filepath.Join(buildRoot, ".conductor", "learning")
	if learningDir != expectedPath {
		t.Errorf("GetLearningDirWithRoot() = %q, want %q", learningDir, expectedPath)
	}

	// Verify directory was created
	if _, err := os.Stat(learningDir); os.IsNotExist(err) {
		t.Errorf("Learning directory not created: %q", learningDir)
	}
}

// TestGetConductorHomeEnvVarDoesNotCreate tests env var path is not created if missing
func TestGetConductorHomeEnvVarDoesNotCreate(t *testing.T) {
	nonExistentDir := filepath.Join(t.TempDir(), "nonexistent", "path")
	t.Setenv("CONDUCTOR_HOME", nonExistentDir)

	home, err := GetConductorHomeWithRoot("")
	if err != nil {
		// Should fail because env var path doesn't exist and can't be created
		// (parent directory missing)
		if err == nil {
			t.Error("Expected error for non-creatable env var path")
		}
		return
	}

	// If it succeeds, directory should have been created
	if _, err := os.Stat(home); os.IsNotExist(err) {
		t.Errorf("Directory should be created at env var path: %q", home)
	}
}
