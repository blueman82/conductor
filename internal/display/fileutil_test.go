package display

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsNumberedFile_ValidPatterns(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"simple numbered md", "1-setup.md", true},
		{"double digit yaml", "12-api.yaml", true},
		{"triple digit markdown", "123-test.markdown", true},
		{"numbered yml", "5-config.yml", true},
		{"single digit with underscore", "9-my_feature.md", true},
		{"multi digit with spaces in name", "42-feature-name.yaml", true},
		{"leading zero allowed", "01-initial.md", true},
		{"three digit leading zero", "001-start.yml", true},
		{"long number", "9999-large.markdown", true},
		{"single digit md", "3-test.md", true},
		{"double digit yml", "25-service.yml", true},
		{"numbered with dots", "15-v2.0.yaml", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNumberedFile(tt.filename)
			if got != tt.want {
				t.Errorf("IsNumberedFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsNumberedFile_InvalidPatterns(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"plan pattern", "plan-01.md", false},
		{"no dash", "1setup.md", false},
		{"no number", "setup.md", false},
		{"wrong extension", "1-setup.txt", false},
		{"starts with text", "task-1.md", false},
		{"too short", "1.m", false},
		{"no extension", "1-setup", false},
		{"dash only", "-setup.md", false},
		{"negative number", "-1-setup.md", false},
		{"decimal number", "1.5-setup.md", false},
		{"number at end", "setup-1.md", false},
		{"multiple dashes no number first", "plan-01-setup.md", false},
		{"empty string", "", false},
		{"just number", "123", false},
		{"just extension", ".md", false},
		{"space before number", " 1-setup.md", false},
		{"number with space", "1 -setup.md", false},
		{"uppercase extension", "1-setup.MD", false},
		{"double extension", "1-setup.md.txt", false},
		{"README", "README.md", false},
		{"hidden file", ".1-hidden.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNumberedFile(tt.filename)
			if got != tt.want {
				t.Errorf("IsNumberedFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsNumberedFile_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"empty string", "", false},
		{"only dash", "-", false},
		{"only number", "123", false},
		{"only extension", ".md", false},
		{"dash extension only", "-.md", false},
		{"number dash only", "1-", false},
		{"number dash extension", "1-.md", false},
		{"very long number", "12345678901234567890-test.md", true},
		{"special chars after dash", "1-test@file.md", true},
		{"unicode in filename", "1-tëst.md", true},
		{"path separator in name", "1-test/file.md", true},  // basename would be "file.md"
		{"windows path separator", "1-test\\file.md", true}, // basename would handle this
		{"newline in name", "1-test\n.md", false},
		{"null byte", "1-test\x00.md", false},
		{"all valid extensions", "1-test.md", true},
		{"all valid extensions yaml", "1-test.yaml", true},
		{"all valid extensions yml", "1-test.yml", true},
		{"all valid extensions markdown", "1-test.markdown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNumberedFile(tt.filename)
			if got != tt.want {
				t.Errorf("IsNumberedFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestFindNumberedFiles_Directory(t *testing.T) {
	// Create temp directory with numbered files
	tmpDir := t.TempDir()

	// Create test files
	files := []string{
		"1-setup.md",
		"2-database.yaml",
		"3-api.yml",
		"10-deploy.markdown",
		"99-cleanup.md",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", f, err)
		}
	}

	// Test FindNumberedFiles
	got, err := FindNumberedFiles(tmpDir)
	if err != nil {
		t.Fatalf("FindNumberedFiles() error = %v", err)
	}

	// Verify count
	if len(got) != len(files) {
		t.Errorf("FindNumberedFiles() returned %d files, want %d", len(got), len(files))
	}

	// Verify each expected file is present (basenames only)
	foundMap := make(map[string]bool)
	for _, f := range got {
		foundMap[f] = true
	}

	for _, expected := range files {
		if !foundMap[expected] {
			t.Errorf("FindNumberedFiles() missing expected file %q", expected)
		}
	}

	// Verify no unexpected files
	for _, found := range got {
		expectedFile := false
		for _, exp := range files {
			if exp == found {
				expectedFile = true
				break
			}
		}
		if !expectedFile {
			t.Errorf("FindNumberedFiles() returned unexpected file %q", found)
		}
	}
}

func TestFindNumberedFiles_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	got, err := FindNumberedFiles(tmpDir)
	if err != nil {
		t.Fatalf("FindNumberedFiles() error = %v", err)
	}

	if len(got) != 0 {
		t.Errorf("FindNumberedFiles() on empty directory returned %d files, want 0", len(got))
	}

	if got == nil {
		t.Error("FindNumberedFiles() returned nil slice, want empty slice")
	}
}

func TestFindNumberedFiles_MixedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mix of numbered and plan-* files
	allFiles := []struct {
		name       string
		shouldFind bool
	}{
		{"1-setup.md", true},
		{"plan-01-database.md", false},
		{"2-api.yaml", true},
		{"plan-02-frontend.yaml", false},
		{"3-deploy.yml", true},
		{"README.md", false},
		{"10-feature.markdown", true},
		{"notes.txt", false},
		{"task-5.md", false},
		{"setup.md", false},
		{"99-last.md", true},
		{".hidden.md", false},
		{"plan-final.yml", false},
	}

	expectedFiles := []string{}
	for _, f := range allFiles {
		path := filepath.Join(tmpDir, f.name)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", f.name, err)
		}
		if f.shouldFind {
			expectedFiles = append(expectedFiles, f.name)
		}
	}

	got, err := FindNumberedFiles(tmpDir)
	if err != nil {
		t.Fatalf("FindNumberedFiles() error = %v", err)
	}

	// Verify only numbered files returned
	if len(got) != len(expectedFiles) {
		t.Errorf("FindNumberedFiles() returned %d files, want %d", len(got), len(expectedFiles))
		t.Logf("Got files: %v", got)
		t.Logf("Expected files: %v", expectedFiles)
	}

	// Verify correct files found
	foundMap := make(map[string]bool)
	for _, f := range got {
		foundMap[f] = true
	}

	for _, expected := range expectedFiles {
		if !foundMap[expected] {
			t.Errorf("FindNumberedFiles() missing expected numbered file %q", expected)
		}
	}

	// Verify no plan-* or other files included
	for _, found := range got {
		shouldBeFound := false
		for _, expected := range expectedFiles {
			if expected == found {
				shouldBeFound = true
				break
			}
		}
		if !shouldBeFound {
			t.Errorf("FindNumberedFiles() incorrectly included file %q", found)
		}
	}
}

func TestFindNumberedFiles_NonExistentDirectory(t *testing.T) {
	// Test with directory that doesn't exist
	nonExistent := "/path/that/does/not/exist/at/all"

	_, err := FindNumberedFiles(nonExistent)
	if err == nil {
		t.Error("FindNumberedFiles() with non-existent directory should return error")
	}
}

func TestFindNumberedFiles_FileInsteadOfDirectory(t *testing.T) {
	// Create a file, then try to use it as directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "testfile.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := FindNumberedFiles(filePath)
	if err == nil {
		t.Error("FindNumberedFiles() with file instead of directory should return error")
	}
}

func TestFindNumberedFiles_NestedDirectories(t *testing.T) {
	// Test that nested directories don't cause issues
	tmpDir := t.TempDir()

	// Create numbered files in root
	rootFiles := []string{"1-root.md", "2-root.yaml"}
	for _, f := range rootFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create root file: %v", err)
		}
	}

	// Create subdirectory with numbered files (should not be found)
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	subFile := filepath.Join(subDir, "3-nested.md")
	if err := os.WriteFile(subFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	got, err := FindNumberedFiles(tmpDir)
	if err != nil {
		t.Fatalf("FindNumberedFiles() error = %v", err)
	}

	// Should only find root files, not nested
	if len(got) != len(rootFiles) {
		t.Errorf("FindNumberedFiles() returned %d files, want %d (should not include nested)", len(got), len(rootFiles))
	}

	// Verify only root files
	foundMap := make(map[string]bool)
	for _, f := range got {
		foundMap[f] = true
	}

	for _, expected := range rootFiles {
		if !foundMap[expected] {
			t.Errorf("FindNumberedFiles() missing expected root file %q", expected)
		}
	}

	// Make sure nested file not included
	if foundMap["3-nested.md"] {
		t.Error("FindNumberedFiles() should not include files from subdirectories")
	}
}
