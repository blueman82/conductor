package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ScanOptions configures the directory scanning behavior
type ScanOptions struct {
	// Pattern is a regex pattern to match filenames (without extension)
	Pattern string
	// Extensions is a list of file extensions to include (e.g., ".md", ".yaml")
	Extensions []string
	// Recursive enables recursive directory scanning
	Recursive bool
	// ExcludeDirs is a list of directory names to exclude (e.g., ".git", "node_modules")
	ExcludeDirs []string
	// MaxDepth limits recursion depth (0 = unlimited, 1 = current dir only)
	MaxDepth int
}

// ScanResult contains the results of a directory scan
type ScanResult struct {
	// Files contains the absolute paths of all matched files
	Files []string
	// Errors contains any errors encountered during scanning
	Errors []error
}

// ScanDirectory scans a directory for files matching the provided options
func ScanDirectory(dir string, opts ScanOptions) (*ScanResult, error) {
	// Validate directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", dir)
	}

	result := &ScanResult{
		Files:  make([]string, 0),
		Errors: make([]error, 0),
	}

	// Compile pattern if provided
	var patternRegex *regexp.Regexp
	if opts.Pattern != "" {
		patternRegex, err = regexp.Compile(opts.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern: %w", err)
		}
	}

	// Create extension map for fast lookup
	extMap := make(map[string]bool)
	for _, ext := range opts.Extensions {
		// Ensure extensions start with a dot
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extMap[strings.ToLower(ext)] = true
	}

	// Create excluded dirs map
	excludeMap := make(map[string]bool)
	for _, dir := range opts.ExcludeDirs {
		excludeMap[dir] = true
	}

	// Walk the directory
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("error accessing %s: %w", path, err))
			return nil // Continue walking
		}

		// Skip the root directory itself
		if path == dir {
			return nil
		}

		// Handle directories
		if d.IsDir() {
			// Check if we should exclude this directory
			if excludeMap[d.Name()] || strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}

			// Check recursion settings
			if !opts.Recursive {
				return filepath.SkipDir
			}

			// Check max depth
			if opts.MaxDepth > 0 {
				relPath, _ := filepath.Rel(dir, path)
				depth := strings.Count(relPath, string(filepath.Separator)) + 1
				if depth >= opts.MaxDepth {
					return filepath.SkipDir
				}
			}

			return nil
		}

		// Process files
		filename := d.Name()

		// Check extension if specified
		if len(extMap) > 0 {
			ext := strings.ToLower(filepath.Ext(filename))
			if !extMap[ext] {
				return nil
			}
		}

		// Check pattern if specified
		if patternRegex != nil {
			// Remove extension for pattern matching
			nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
			if !patternRegex.MatchString(nameWithoutExt) {
				return nil
			}
		}

		// Convert to absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to resolve path %s: %w", path, err))
			return nil
		}

		result.Files = append(result.Files, absPath)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Sort files for consistent output
	sort.Strings(result.Files)

	return result, nil
}
