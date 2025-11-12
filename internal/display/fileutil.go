package display

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// IsNumberedFile checks if filename matches pattern: ^\d+-.*\.(md|markdown|yaml|yml)$
// Pattern validation:
// 1. First character must be digit (0-9)
// 2. Scan for dash, ensuring only digits before it
// 3. Must have dash character
// 4. Must end with valid extension: .md, .markdown, .yaml, .yml (lowercase only)
func IsNumberedFile(filename string) bool {
	// Empty string or too short
	if len(filename) < 5 { // Minimum: "1-a.m" but need valid extension, so really "1-a.md" = 7
		return false
	}

	// Must start with digit
	if !unicode.IsDigit(rune(filename[0])) {
		return false
	}

	// Scan for dash, ensuring only digits before it
	dashIndex := -1
	for i := 1; i < len(filename); i++ {
		if filename[i] == '-' {
			dashIndex = i
			break
		}
		if !unicode.IsDigit(rune(filename[i])) {
			// Non-digit before dash
			return false
		}
	}

	// Must have dash
	if dashIndex == -1 {
		return false
	}

	// Must have something after dash before extension
	if dashIndex == len(filename)-1 {
		return false
	}

	// Check for invalid characters (newline, null byte)
	if strings.ContainsAny(filename, "\n\x00") {
		return false
	}

	// Check extension (case-sensitive, must be lowercase)
	ext := filepath.Ext(filename)
	switch ext {
	case ".md", ".markdown", ".yaml", ".yml":
		// Extension is valid, now check that there's content between dash and extension
		// For "1-.md", dashIndex=1, ext=".md", so filename without ext is "1-"
		// We need at least 1 char after the dash
		filenameWithoutExt := strings.TrimSuffix(filename, ext)
		if len(filenameWithoutExt) <= dashIndex+1 {
			// No content between dash and extension (e.g., "1-.md")
			return false
		}
		return true
	default:
		return false
	}
}

// FindNumberedFiles scans a directory and returns basenames of numbered plan files
// Only scans the immediate directory (not recursive)
// Returns error if path doesn't exist or is not a directory
func FindNumberedFiles(dirPath string) ([]string, error) {
	// Check if path exists and is a directory
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access path: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", dirPath)
	}

	// Read directory entries
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Find numbered files
	var numberedFiles []string
	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		// Check if filename matches numbered pattern
		if IsNumberedFile(entry.Name()) {
			numberedFiles = append(numberedFiles, entry.Name())
		}
	}

	// Return empty slice instead of nil if no files found
	if numberedFiles == nil {
		numberedFiles = []string{}
	}

	return numberedFiles, nil
}
