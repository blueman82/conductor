package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseExportFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		want    string
		wantErr bool
	}{
		{
			name:    "json format",
			format:  "json",
			want:    "json",
			wantErr: false,
		},
		{
			name:    "markdown format",
			format:  "markdown",
			want:    "markdown",
			wantErr: false,
		},
		{
			name:    "md alias",
			format:  "md",
			want:    "markdown",
			wantErr: false,
		},
		{
			name:    "uppercase JSON",
			format:  "JSON",
			want:    "json",
			wantErr: false,
		},
		{
			name:    "mixed case Markdown",
			format:  "MarkDown",
			want:    "markdown",
			wantErr: false,
		},
		{
			name:    "format with spaces",
			format:  "  json  ",
			want:    "json",
			wantErr: false,
		},
		{
			name:    "invalid format",
			format:  "xml",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty format",
			format:  "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseExportFormat(tt.format)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseExportFormat() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("parseExportFormat() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("parseExportFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetExportPath(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
		checks  func(t *testing.T, result string)
	}{
		{
			name:    "absolute path",
			path:    filepath.Join(tempDir, "export.json"),
			wantErr: false,
			checks: func(t *testing.T, result string) {
				if !filepath.IsAbs(result) {
					t.Errorf("getExportPath() result should be absolute, got %q", result)
				}
			},
		},
		{
			name:    "relative path",
			path:    "export.json",
			wantErr: false,
			checks: func(t *testing.T, result string) {
				if !filepath.IsAbs(result) {
					t.Errorf("getExportPath() result should be absolute, got %q", result)
				}
				if !strings.HasSuffix(result, "export.json") {
					t.Errorf("getExportPath() result should end with 'export.json', got %q", result)
				}
			},
		},
		{
			name:    "nested path",
			path:    filepath.Join(tempDir, "subdir", "nested", "export.json"),
			wantErr: false,
			checks: func(t *testing.T, result string) {
				if !filepath.IsAbs(result) {
					t.Errorf("getExportPath() result should be absolute, got %q", result)
				}
			},
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			checks:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getExportPath(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("getExportPath() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("getExportPath() unexpected error: %v", err)
				return
			}

			if tt.checks != nil {
				tt.checks(t, got)
			}
		})
	}
}

func TestGetExportPath_HomeDirectory(t *testing.T) {
	// Test home directory expansion
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory, skipping test")
	}

	tests := []struct {
		name     string
		path     string
		wantPath string
	}{
		{
			name:     "home directory with tilde",
			path:     "~/export.json",
			wantPath: filepath.Join(homeDir, "export.json"),
		},
		{
			name:     "home directory with nested path",
			path:     "~/Documents/exports/data.json",
			wantPath: filepath.Join(homeDir, "Documents/exports/data.json"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getExportPath(tt.path)
			if err != nil {
				t.Errorf("getExportPath() unexpected error: %v", err)
				return
			}

			if got != tt.wantPath {
				t.Errorf("getExportPath() = %q, want %q", got, tt.wantPath)
			}
		})
	}
}

func TestHandleExportCommand_FormatValidation(t *testing.T) {
	// Test that format validation works
	tests := []struct {
		name       string
		format     string
		shouldFail bool
	}{
		{
			name:       "valid json",
			format:     "json",
			shouldFail: false,
		},
		{
			name:       "valid markdown",
			format:     "markdown",
			shouldFail: false,
		},
		{
			name:       "valid md alias",
			format:     "md",
			shouldFail: false,
		},
		{
			name:       "valid csv",
			format:     "csv",
			shouldFail: false,
		},
		{
			name:       "invalid format",
			format:     "xml",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseExportFormat(tt.format)
			if tt.shouldFail && err == nil {
				t.Errorf("parseExportFormat() expected error for invalid format %q", tt.format)
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("parseExportFormat() unexpected error for valid format %q: %v", tt.format, err)
			}
		})
	}
}
