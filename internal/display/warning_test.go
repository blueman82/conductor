package display

import (
	"bytes"
	"strings"
	"testing"
)

func TestDisplayWarning_TitleOnly(t *testing.T) {
	var buf bytes.Buffer
	w := Warning{
		Title: "Configuration Missing",
	}

	w.Display(&buf)

	output := buf.String()

	// Should contain yellow color code
	if !strings.Contains(output, "\x1b[33m") {
		t.Error("Expected yellow ANSI color code in output")
	}

	// Should contain warning emoji
	if !strings.Contains(output, "⚠️") {
		t.Error("Expected warning emoji ⚠️ in output")
	}

	// Should contain title
	if !strings.Contains(output, "Configuration Missing") {
		t.Error("Expected title in output")
	}

	// Should end with reset code
	if !strings.Contains(output, "\x1b[0m") {
		t.Error("Expected ANSI reset code in output")
	}
}

func TestDisplayWarning_WithMessage(t *testing.T) {
	var buf bytes.Buffer
	w := Warning{
		Title:   "Deprecated Feature",
		Message: "This feature will be removed in v2.0",
	}

	w.Display(&buf)

	output := buf.String()

	// Should contain title
	if !strings.Contains(output, "Deprecated Feature") {
		t.Error("Expected title in output")
	}

	// Should contain message with indentation
	if !strings.Contains(output, "    This feature will be removed in v2.0") {
		t.Error("Expected indented message in output")
	}

	// Should contain yellow color
	if !strings.Contains(output, "\x1b[33m") {
		t.Error("Expected yellow ANSI color code in output")
	}
}

func TestDisplayWarning_WithFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		wantText string
	}{
		{
			name:     "single file",
			files:    []string{"config.yaml"},
			wantText: "Affected file:",
		},
		{
			name:     "multiple files",
			files:    []string{"config.yaml", "settings.toml", "app.json"},
			wantText: "Affected files:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := Warning{
				Title: "Invalid Configuration",
				Files: tt.files,
			}

			w.Display(&buf)

			output := buf.String()

			// Should use singular/plural correctly
			if !strings.Contains(output, tt.wantText) {
				t.Errorf("Expected %q in output, got: %s", tt.wantText, output)
			}

			// Should list each file with indentation and numbering
			for i, file := range tt.files {
				expected := strings.Repeat(" ", 6) + (string(rune('1'+i))) + ". " + file
				if !strings.Contains(output, expected) {
					t.Errorf("Expected file entry %q in output, got: %s", expected, output)
				}
			}

			// Should contain yellow color
			if !strings.Contains(output, "\x1b[33m") {
				t.Error("Expected yellow ANSI color code in output")
			}
		})
	}
}

func TestDisplayWarning_WithSuggestion(t *testing.T) {
	var buf bytes.Buffer
	w := Warning{
		Title:      "Missing Dependencies",
		Suggestion: "Run 'go mod download' to install dependencies",
	}

	w.Display(&buf)

	output := buf.String()

	// Should contain title
	if !strings.Contains(output, "Missing Dependencies") {
		t.Error("Expected title in output")
	}

	// Should contain suggestion with indentation
	if !strings.Contains(output, "    Run 'go mod download' to install dependencies") {
		t.Error("Expected indented suggestion in output")
	}

	// Should have "Suggestion:" label
	if !strings.Contains(output, "Suggestion:") {
		t.Error("Expected 'Suggestion:' label in output")
	}

	// Should contain yellow color
	if !strings.Contains(output, "\x1b[33m") {
		t.Error("Expected yellow ANSI color code in output")
	}
}

func TestDisplayWarning_Complete(t *testing.T) {
	var buf bytes.Buffer
	w := Warning{
		Title:      "Potential Performance Issue",
		Message:    "Database query is missing an index",
		Files:      []string{"internal/db/users.go", "internal/db/queries.sql"},
		Suggestion: "Add index on users.email column",
	}

	w.Display(&buf)

	output := buf.String()

	// Should contain all components
	components := []string{
		"⚠️",
		"Potential Performance Issue",
		"    Database query is missing an index",
		"    Affected files:",
		"      1. internal/db/users.go",
		"      2. internal/db/queries.sql",
		"    Suggestion:",
		"    Add index on users.email column",
		"\x1b[33m", // Yellow color
		"\x1b[0m",  // Reset
	}

	for _, component := range components {
		if !strings.Contains(output, component) {
			t.Errorf("Expected component %q in output, got: %s", component, output)
		}
	}
}

func TestDisplayWarning_YellowColor(t *testing.T) {
	var buf bytes.Buffer
	w := Warning{
		Title: "Test Warning",
	}

	w.Display(&buf)

	output := buf.String()

	// Should start with yellow color code
	if !strings.HasPrefix(output, "\x1b[33m") {
		t.Error("Expected output to start with yellow ANSI color code \\x1b[33m")
	}

	// Should contain warning emoji
	if !strings.Contains(output, "⚠️") {
		t.Error("Expected warning emoji ⚠️ in output")
	}

	// Should end with reset code
	if !strings.HasSuffix(strings.TrimSpace(output), "\x1b[0m") {
		t.Error("Expected output to end with ANSI reset code \\x1b[0m")
	}

	// Should contain yellow throughout (not reset in the middle)
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		// Each non-empty line should maintain yellow color
		if i == 0 && !strings.HasPrefix(line, "\x1b[33m") {
			t.Error("Expected first line to start with yellow color")
		}
	}
}

func TestWarnNumberedFiles(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		files         []string
		wantTitle     string
		wantFileCount int
	}{
		{
			name:          "single file",
			title:         "File Not Found",
			files:         []string{"missing.go"},
			wantTitle:     "File Not Found",
			wantFileCount: 1,
		},
		{
			name:          "multiple files",
			title:         "Parse Errors",
			files:         []string{"file1.go", "file2.go", "file3.go"},
			wantTitle:     "Parse Errors",
			wantFileCount: 3,
		},
		{
			name:          "empty files list",
			title:         "General Warning",
			files:         []string{},
			wantTitle:     "General Warning",
			wantFileCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := WarnNumberedFiles(tt.title, tt.files)

			// Should set title correctly
			if w.Title != tt.wantTitle {
				t.Errorf("Expected title %q, got %q", tt.wantTitle, w.Title)
			}

			// Should set files correctly
			if len(w.Files) != tt.wantFileCount {
				t.Errorf("Expected %d files, got %d", tt.wantFileCount, len(w.Files))
			}

			// Should preserve file order
			for i, file := range tt.files {
				if w.Files[i] != file {
					t.Errorf("Expected file[%d] to be %q, got %q", i, file, w.Files[i])
				}
			}

			// Should be displayable
			var buf bytes.Buffer
			w.Display(&buf)
			output := buf.String()

			if !strings.Contains(output, tt.wantTitle) {
				t.Errorf("Expected displayable warning with title %q", tt.wantTitle)
			}
		})
	}
}
