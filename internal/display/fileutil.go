package display

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/harrison/conductor/internal/fileutil"
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
	// Use fileutil.ScanDirectory to find files matching numbered pattern
	opts := fileutil.ScanOptions{
		Pattern:    `^\d+-.*`,                                     // Match files starting with digits followed by dash
		Extensions: []string{".md", ".markdown", ".yaml", ".yml"}, // Valid plan file extensions
		Recursive:  false,                                         // Only scan top level
	}

	result, err := fileutil.ScanDirectory(dirPath, opts)
	if err != nil {
		return nil, err
	}

	// Convert absolute paths to basenames to maintain API compatibility
	numberedFiles := make([]string, 0, len(result.Files))
	for _, absPath := range result.Files {
		basename := filepath.Base(absPath)
		// Double-check with IsNumberedFile for extra validation
		if IsNumberedFile(basename) {
			numberedFiles = append(numberedFiles, basename)
		}
	}

	return numberedFiles, nil
}
