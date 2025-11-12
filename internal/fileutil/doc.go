// Package fileutil provides centralized file system scanning and pattern matching utilities.
//
// This package serves as a single source of truth for all file operations in Conductor,
// offering robust directory traversal with flexible filtering, pattern matching, and
// error-tolerant scanning capabilities.
//
// # Purpose
//
// The fileutil package is designed for:
//   - Directory traversal with recursive and depth-limited scanning
//   - File filtering by extension and regex patterns
//   - Pattern matching on filenames without extensions
//   - Directory exclusion (hidden dirs, .git, node_modules, etc.)
//   - Error-tolerant scanning that collects non-fatal errors
//
// # Key Features
//
//   - Recursive directory scanning with configurable depth limits
//   - Case-insensitive file extension filtering
//   - Regex pattern matching on filenames (without extension)
//   - Automatic exclusion of hidden directories (starting with ".")
//   - Configurable directory exclusion (e.g., .git, node_modules)
//   - Sorted, deterministic output (alphabetically sorted file paths)
//   - Error tolerance (non-fatal errors collected, scanning continues)
//   - Absolute path resolution for all matched files
//   - Standard library only (no external dependencies)
//
// # Main Components
//
// ScanOptions - Configuration struct for directory scanning:
//   - Pattern: Regex pattern to match filenames (without extension)
//   - Extensions: List of file extensions to include (case-insensitive, e.g., ".md", ".yaml")
//   - Recursive: Enable/disable subdirectory traversal
//   - ExcludeDirs: Directory names to skip (e.g., ".git", "node_modules")
//   - MaxDepth: Limit recursion depth (0 = unlimited, 1 = current dir only)
//
// ScanResult - Results of directory scan:
//   - Files: Absolute paths of all matched files (sorted alphabetically)
//   - Errors: Non-fatal errors encountered during scan
//
// # ScanDirectory() - Main scanning function that walks directories with the provided options
//
// # Usage Examples
//
// Basic recursive scanning (all files):
//
//	result, err := fileutil.ScanDirectory("/path/to/dir", fileutil.ScanOptions{
//	    Recursive: true,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, file := range result.Files {
//	    fmt.Println(file)
//	}
//
// Extension filtering (Markdown and YAML files):
//
//	result, err := fileutil.ScanDirectory("/path/to/plans", fileutil.ScanOptions{
//	    Extensions: []string{".md", ".yaml", ".yml"},
//	    Recursive:  true,
//	})
//
// Pattern matching (files starting with "plan-"):
//
//	result, err := fileutil.ScanDirectory("/path/to/dir", fileutil.ScanOptions{
//	    Pattern:    "^plan-.*",
//	    Extensions: []string{".md"},
//	    Recursive:  true,
//	})
//
// Numbered file detection (e.g., "01-setup.md", "02-backend.md"):
//
//	result, err := fileutil.ScanDirectory("/path/to/dir", fileutil.ScanOptions{
//	    Pattern:   "^\\d+",  // Starts with digits
//	    Recursive: false,    // Current directory only
//	    MaxDepth:  1,
//	})
//
// Combined options (pattern + extensions + depth limit):
//
//	result, err := fileutil.ScanDirectory("/path/to/docs", fileutil.ScanOptions{
//	    Pattern:     "^task-",
//	    Extensions:  []string{".md", ".yaml"},
//	    Recursive:   true,
//	    MaxDepth:    2,
//	    ExcludeDirs: []string{"examples", "drafts"},
//	})
//
// Directory exclusion (skip .git, node_modules, vendor):
//
//	result, err := fileutil.ScanDirectory("/path/to/project", fileutil.ScanOptions{
//	    Extensions:  []string{".go"},
//	    Recursive:   true,
//	    ExcludeDirs: []string{".git", "node_modules", "vendor"},
//	})
//
// Error handling (check for non-fatal errors):
//
//	result, err := fileutil.ScanDirectory("/path/to/dir", fileutil.ScanOptions{
//	    Recursive: true,
//	})
//	if err != nil {
//	    log.Fatalf("Fatal error: %v", err)
//	}
//	if len(result.Errors) > 0 {
//	    log.Printf("Encountered %d non-fatal errors:", len(result.Errors))
//	    for _, err := range result.Errors {
//	        log.Printf("  - %v", err)
//	    }
//	}
//
// # Design Principles
//
// Single Source of Truth:
// This package centralizes all file system operations to avoid duplicated logic
// across the codebase. Any file scanning functionality should use this package
// rather than implementing custom filepath.Walk logic.
//
// Performance-Oriented:
//   - Sorted output ensures deterministic results for testing and consistency
//   - Case-insensitive extension matching via maps for O(1) lookups
//   - Fast directory exclusion using hash maps
//   - Efficient regex compilation (once per scan)
//
// Error Tolerance:
// The scanner collects non-fatal errors (e.g., permission denied on a subdirectory)
// and continues scanning. Only fatal errors (e.g., root directory doesn't exist,
// invalid regex pattern) cause immediate failure.
//
// Standard Library Only:
// The package uses only Go's standard library (os, path/filepath, regexp, strings, sort)
// with no external dependencies, ensuring minimal overhead and maximum compatibility.
//
// Auto-Exclusion of Hidden Directories:
// Directories starting with "." (e.g., .git, .cache) are automatically skipped
// during recursive scans to avoid scanning hidden system directories.
//
// # Performance Characteristics
//
// Sorted Output:
// All file paths are sorted alphabetically before being returned, ensuring
// deterministic output across runs and platforms. This is critical for testing
// and consistent behavior.
//
// Case-Insensitive Extension Matching:
// Extensions are normalized to lowercase for matching, allowing users to specify
// ".MD", ".md", or "md" and match all variants (file.md, file.MD, file.Md).
//
// Fast Directory Exclusion:
// Excluded directories are stored in a map for O(1) lookup time, making exclusion
// checks efficient even with large exclusion lists.
//
// Memory Efficient:
// Files are collected in a slice but only store absolute paths (strings), not
// full file metadata, keeping memory usage low even when scanning large directories.
//
// # Common Use Cases in Conductor
//
// Plan File Discovery:
// Scanning for .md and .yaml plan files in a directory for multi-file plan execution.
//
// Numbered File Detection:
// Detecting numbered files (e.g., 01-setup.md, 02-backend.md) to warn users about
// potential execution order confusion with multi-file plans.
//
// Worktree Scanning:
// Finding all relevant files within a worktree while excluding .git, node_modules,
// and other non-essential directories.
//
// Test Fixture Discovery:
// Locating test fixtures in testdata/ directories during test execution.
package fileutil
