package parser

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
//  1. Auto-detects the format from the file extension
//  2. Opens the file
//  3. Calls the appropriate parser
//  4. Stores the original file path in plan.FilePath
//
// This is the recommended way to parse plan files from disk.
func ParseFile(path string) (*models.Plan, error) {
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
