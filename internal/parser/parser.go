package parser

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/harrison/conductor/internal/fileutil"
	"github.com/harrison/conductor/internal/models"
)

// Format represents the format of a plan file
type Format int

const (
	// FormatUnknown represents an unknown or unsupported file format
	FormatUnknown Format = iota
	// FormatMarkdown represents a Markdown (.md, .markdown) plan file
	FormatMarkdown
	// FormatYAML represents a YAML (.yaml, .yml) plan file
	FormatYAML
)

// String returns the string representation of the Format
func (f Format) String() string {
	switch f {
	case FormatMarkdown:
		return "markdown"
	case FormatYAML:
		return "yaml"
	default:
		return "unknown"
	}
}

// Parser is the interface that all plan parsers must implement
type Parser interface {
	// Parse reads from an io.Reader and returns a parsed Plan
	Parse(r io.Reader) (*models.Plan, error)
}

// DetectFormat automatically detects the plan format based on file extension
// Supported extensions:
//   - .md, .markdown -> FormatMarkdown
//   - .yaml, .yml -> FormatYAML
//   - all others -> FormatUnknown
func DetectFormat(filename string) Format {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".md", ".markdown":
		return FormatMarkdown
	case ".yaml", ".yml":
		return FormatYAML
	default:
		return FormatUnknown
	}
}

// NewParser creates a new parser instance for the specified format
// Returns an error if the format is unknown or unsupported
func NewParser(format Format) (Parser, error) {
	switch format {
	case FormatMarkdown:
		return NewMarkdownParser(), nil
	case FormatYAML:
		return NewYAMLParser(), nil
	default:
		return nil, fmt.Errorf("unsupported format: %v", format)
	}
}

// ParseFile is a convenience function that:
//  1. Detects if input is a directory (multi-file plan) or file
//  2. For directories, calls ParseDirectory to load split plans
//  3. For files, auto-detects format, opens it, and parses
//  4. Stores the original file path in plan.FilePath
//
// This is the recommended way to parse plan files from disk.
func ParseFile(path string) (*models.Plan, error) {
	// Check if path is a directory
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to access path: %w", err)
	}

	if info.IsDir() {
		return ParseDirectory(path)
	}

	// Auto-detect format from extension
	format := DetectFormat(path)
	if format == FormatUnknown {
		return nil, fmt.Errorf("unknown file format: %s (supported: .md, .markdown, .yaml, .yml)", path)
	}

	// Create appropriate parser
	parser, err := NewParser(format)
	if err != nil {
		return nil, err
	}

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Parse the file
	plan, err := parser.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	// Store the original file path for later updates
	// Convert to absolute path for consistency
	absPath, err := filepath.Abs(path)
	if err != nil {
		// If we can't get absolute path, use the original
		absPath = path
	}
	plan.FilePath = absPath

	return plan, nil
}

// IsSplitPlan detects if a directory contains a split plan
// Returns true if numbered files (1-*.md, 2-*.yaml, etc.) are found
func IsSplitPlan(dirname string) bool {
	entries, err := os.ReadDir(dirname)
	if err != nil {
		return false
	}

	pattern := regexp.MustCompile(`^\d+-`)
	for _, entry := range entries {
		if !entry.IsDir() && pattern.MatchString(entry.Name()) && DetectFormat(entry.Name()) != FormatUnknown {
			return true
		}
	}
	return false
}

// ParseDirectory loads all numbered plan files from a directory
// and merges them into a single plan.
func ParseDirectory(dirname string) (*models.Plan, error) {
	info, err := os.Stat(dirname)
	if err != nil {
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", dirname)
	}

	entries, err := os.ReadDir(dirname)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	type planFile struct {
		index int
		path  string
		name  string
	}

	var planFiles []planFile
	pattern := regexp.MustCompile(`^(\d+)-`)

	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		match := pattern.FindStringSubmatch(entry.Name())
		if match == nil || DetectFormat(entry.Name()) == FormatUnknown {
			continue
		}

		var index int
		fmt.Sscanf(match[1], "%d", &index)
		planFiles = append(planFiles, planFile{index, filepath.Join(dirname, entry.Name()), entry.Name()})
	}

	if len(planFiles) == 0 {
		return &models.Plan{Name: filepath.Base(dirname), Tasks: []models.Task{}}, nil
	}

	sort.Slice(planFiles, func(i, j int) bool { return planFiles[i].index < planFiles[j].index })

	var plans []*models.Plan
	for _, pf := range planFiles {
		plan, err := parseFile(pf.path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", pf.name, err)
		}
		plans = append(plans, plan)
	}

	merged, err := MergePlans(plans...)
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(dirname)
	if err != nil {
		absPath = dirname
	}
	merged.FilePath = absPath
	return merged, nil
}

// parseFile is the internal implementation that parses a single file
func parseFile(path string) (*models.Plan, error) {
	format := DetectFormat(path)
	if format == FormatUnknown {
		return nil, fmt.Errorf("unknown file format: %s", path)
	}

	parser, err := NewParser(format)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	plan, err := parser.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}
	return plan, nil
}

// MergePlans combines multiple plans into a single plan
// while preserving all task dependencies.
func MergePlans(plans ...*models.Plan) (*models.Plan, error) {
	if len(plans) == 0 {
		return &models.Plan{Tasks: []models.Task{}}, nil
	}

	seen := make(map[string]bool)
	var mergedTasks []models.Task

	for _, plan := range plans {
		if plan == nil {
			continue
		}
		for _, task := range plan.Tasks {
			if seen[task.Number] {
				return nil, fmt.Errorf("duplicate task number: %s", task.Number)
			}
			seen[task.Number] = true
			// Set SourceFile to track which plan file this task comes from
			task.SourceFile = plan.FilePath
			mergedTasks = append(mergedTasks, task)
		}
	}

	var result *models.Plan
	for _, plan := range plans {
		if plan != nil {
			result = &models.Plan{
				Name:           plan.Name,
				DefaultAgent:   plan.DefaultAgent,
				QualityControl: plan.QualityControl,
				WorktreeGroups: plan.WorktreeGroups,
				Tasks:          mergedTasks,
				FilePath:       plan.FilePath,
			}
			break
		}
	}

	if result == nil {
		result = &models.Plan{Tasks: mergedTasks}
	}
	return result, nil
}

// FilterPlanFiles accepts an array of file and/or directory paths and returns
// a deduplicated list of absolute file paths that match the plan-* pattern.
//
// Pattern matching rules:
//   - Files MUST start with "plan-" prefix
//   - Files MUST have extension: .md, .markdown, .yaml, or .yml
//   - Examples: plan-01-setup.md, plan-02-features.yaml, plan-database.yml
//
// For directories, recursively scans for files matching the pattern.
// For files, checks if filename matches the pattern.
//
// Returns error if:
//   - No paths provided (empty input array)
//   - Any path doesn't exist
//   - No valid plan files are found
func FilterPlanFiles(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("no paths provided")
	}

	// Use map for deduplication
	planFiles := make(map[string]bool)

	// Regex pattern for plan-* files with valid extensions
	planPattern := regexp.MustCompile(`^plan-.*\.(md|markdown|yaml|yml)$`)

	// Configure scan options for fileutil
	opts := fileutil.ScanOptions{
		Pattern:    "^plan-.*", // Match plan-* prefix
		Extensions: []string{".md", ".markdown", ".yaml", ".yml"},
		Recursive:  true, // Recursively scan directories
	}

	for _, path := range paths {
		// Convert to absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path %q: %w", path, err)
		}

		// Check if path exists
		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("path %q does not exist", absPath)
			}
			return nil, fmt.Errorf("failed to access path %q: %w", absPath, err)
		}

		if info.IsDir() {
			// Scan directory for plan files using fileutil
			result, err := fileutil.ScanDirectory(absPath, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to scan directory %q: %w", absPath, err)
			}
			// Add all matched files to our deduplication map
			for _, file := range result.Files {
				planFiles[file] = true
			}
		} else {
			// Single file - add directly if it matches criteria
			filename := filepath.Base(absPath)
			if planPattern.MatchString(filename) {
				planFiles[absPath] = true
			}
		}
	}

	// Check if we found any plan files
	if len(planFiles) == 0 {
		return nil, fmt.Errorf("no plan files found matching pattern plan-*.{md,markdown,yaml,yml}")
	}

	// Convert map to sorted slice for consistent output
	result := make([]string, 0, len(planFiles))
	for path := range planFiles {
		result = append(result, path)
	}
	sort.Strings(result)

	return result, nil
}
