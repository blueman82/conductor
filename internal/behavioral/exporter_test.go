package behavioral

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJSONExporter_Export(t *testing.T) {
	tests := []struct {
		name    string
		metrics *BehavioralMetrics
		pretty  bool
		wantErr bool
	}{
		{
			name: "valid metrics with pretty printing",
			metrics: &BehavioralMetrics{
				TotalSessions:   10,
				SuccessRate:     0.9,
				AverageDuration: 5 * time.Second,
				TotalCost:       10.50,
				ErrorRate:       0.1,
				TotalErrors:     1,
			},
			pretty:  true,
			wantErr: false,
		},
		{
			name: "valid metrics without pretty printing",
			metrics: &BehavioralMetrics{
				TotalSessions: 5,
				SuccessRate:   1.0,
				ErrorRate:     0.0,
			},
			pretty:  false,
			wantErr: false,
		},
		{
			name:    "nil metrics",
			metrics: nil,
			pretty:  true,
			wantErr: true,
		},
		{
			name: "invalid metrics - negative sessions",
			metrics: &BehavioralMetrics{
				TotalSessions: -1,
			},
			pretty:  true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := &JSONExporter{Pretty: tt.pretty}
			result, err := exporter.Export(tt.metrics)

			if tt.wantErr {
				if err == nil {
					t.Errorf("JSONExporter.Export() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("JSONExporter.Export() unexpected error: %v", err)
				return
			}

			// Validate JSON structure
			var parsed BehavioralMetrics
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Errorf("JSONExporter.Export() produced invalid JSON: %v", err)
			}

			// Check pretty printing
			if tt.pretty && !strings.Contains(result, "\n") {
				t.Errorf("JSONExporter.Export() with Pretty=true should contain newlines")
			}
		})
	}
}

func TestMarkdownExporter_Export(t *testing.T) {
	tests := []struct {
		name             string
		metrics          *BehavioralMetrics
		includeTimestamp bool
		wantErr          bool
		wantContains     []string
	}{
		{
			name: "complete metrics",
			metrics: &BehavioralMetrics{
				TotalSessions:   10,
				SuccessRate:     0.85,
				ErrorRate:       0.15,
				TotalErrors:     2,
				AverageDuration: 3 * time.Minute,
				TotalCost:       5.25,
				TokenUsage: TokenUsage{
					InputTokens:  1000,
					OutputTokens: 500,
					CostUSD:      5.25,
					ModelName:    "claude-sonnet-4-5",
				},
				ToolExecutions: []ToolExecution{
					{Name: "Read", Count: 50, SuccessRate: 1.0, ErrorRate: 0.0, AvgDuration: 100 * time.Millisecond},
				},
			},
			includeTimestamp: true,
			wantErr:          false,
			wantContains: []string{
				"# Behavioral Metrics Report",
				"## Summary",
				"**Total Sessions**: 10",
				"**Success Rate**: 85.00%",
				"## Token Usage",
				"**Input Tokens**: 1000",
				"## Tool Executions",
				"| Read |",
			},
		},
		{
			name: "minimal metrics",
			metrics: &BehavioralMetrics{
				TotalSessions: 1,
				SuccessRate:   1.0,
				ErrorRate:     0.0,
			},
			includeTimestamp: false,
			wantErr:          false,
			wantContains: []string{
				"# Behavioral Metrics Report",
				"## Summary",
			},
		},
		{
			name:    "nil metrics",
			metrics: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := &MarkdownExporter{IncludeTimestamp: tt.includeTimestamp}
			result, err := exporter.Export(tt.metrics)

			if tt.wantErr {
				if err == nil {
					t.Errorf("MarkdownExporter.Export() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("MarkdownExporter.Export() unexpected error: %v", err)
				return
			}

			// Check for expected content
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("MarkdownExporter.Export() missing expected content: %q", want)
				}
			}

			// Check timestamp inclusion
			if tt.includeTimestamp && !strings.Contains(result, "**Generated**:") {
				t.Errorf("MarkdownExporter.Export() with IncludeTimestamp=true should include timestamp")
			}
		})
	}
}

func TestExportToFile(t *testing.T) {
	tempDir := t.TempDir()

	metrics := &BehavioralMetrics{
		TotalSessions: 5,
		SuccessRate:   0.8,
		ErrorRate:     0.2,
	}

	tests := []struct {
		name    string
		path    string
		format  string
		wantErr bool
	}{
		{
			name:    "export JSON",
			path:    filepath.Join(tempDir, "export.json"),
			format:  "json",
			wantErr: false,
		},
		{
			name:    "export Markdown",
			path:    filepath.Join(tempDir, "export.md"),
			format:  "markdown",
			wantErr: false,
		},
		{
			name:    "export with md alias",
			path:    filepath.Join(tempDir, "export2.md"),
			format:  "md",
			wantErr: false,
		},
		{
			name:    "unsupported format",
			path:    filepath.Join(tempDir, "export.txt"),
			format:  "txt",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			format:  "json",
			wantErr: true,
		},
		{
			name:    "nested directory creation",
			path:    filepath.Join(tempDir, "subdir", "nested", "export.json"),
			format:  "json",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExportToFile(metrics, tt.path, tt.format)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExportToFile() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ExportToFile() unexpected error: %v", err)
				return
			}

			// Verify file exists and is readable
			content, err := os.ReadFile(tt.path)
			if err != nil {
				t.Errorf("ExportToFile() failed to read exported file: %v", err)
				return
			}

			if len(content) == 0 {
				t.Errorf("ExportToFile() produced empty file")
			}

			// Verify content format
			switch tt.format {
			case "json":
				var parsed BehavioralMetrics
				if err := json.Unmarshal(content, &parsed); err != nil {
					t.Errorf("ExportToFile() produced invalid JSON: %v", err)
				}
			case "markdown", "md":
				if !strings.HasPrefix(string(content), "# Behavioral Metrics Report") {
					t.Errorf("ExportToFile() markdown missing expected header")
				}
			}
		})
	}
}

func TestExportToString(t *testing.T) {
	metrics := &BehavioralMetrics{
		TotalSessions: 3,
		SuccessRate:   0.7,
		ErrorRate:     0.3,
	}

	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{
			name:    "export to JSON string",
			format:  "json",
			wantErr: false,
		},
		{
			name:    "export to Markdown string",
			format:  "markdown",
			wantErr: false,
		},
		{
			name:    "export with md alias",
			format:  "md",
			wantErr: false,
		},
		{
			name:    "unsupported format",
			format:  "xml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExportToString(metrics, tt.format)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExportToString() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ExportToString() unexpected error: %v", err)
				return
			}

			if len(result) == 0 {
				t.Errorf("ExportToString() produced empty string")
			}
		})
	}
}

func TestExportToFile_NilMetrics(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "test.json")

	err := ExportToFile(nil, path, "json")
	if err == nil {
		t.Errorf("ExportToFile() with nil metrics should return error")
	}
}

func TestExportToString_NilMetrics(t *testing.T) {
	_, err := ExportToString(nil, "json")
	if err == nil {
		t.Errorf("ExportToString() with nil metrics should return error")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "needs truncation",
			input:  "hello world this is a long string",
			maxLen: 10,
			want:   "hello w...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 5,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.want {
				t.Errorf("truncateString() = %q, want %q", result, tt.want)
			}
			if len(result) > tt.maxLen {
				t.Errorf("truncateString() result length %d exceeds maxLen %d", len(result), tt.maxLen)
			}
		})
	}
}

func TestCSVExporter_Export(t *testing.T) {
	tests := []struct {
		name        string
		metrics     *BehavioralMetrics
		wantErr     bool
		wantContains []string
	}{
		{
			name: "complete metrics",
			metrics: &BehavioralMetrics{
				TotalSessions:   10,
				SuccessRate:     0.85,
				ErrorRate:       0.15,
				TotalErrors:     2,
				AverageDuration: 3 * time.Minute,
				TotalCost:       5.25,
				TokenUsage: TokenUsage{
					InputTokens:  1000,
					OutputTokens: 500,
					CostUSD:      5.25,
					ModelName:    "claude-sonnet-4-5",
				},
				AgentPerformance: map[string]int{"agent1": 5, "agent2": 3},
				ToolExecutions: []ToolExecution{
					{Name: "Read", Count: 50, SuccessRate: 1.0, ErrorRate: 0.0, AvgDuration: 100 * time.Millisecond},
				},
				BashCommands: []BashCommand{
					{Command: "ls -la", ExitCode: 0, Success: true, Duration: time.Second, OutputLength: 100},
				},
				FileOperations: []FileOperation{
					{Type: "read", Path: "/test/file.txt", Success: true, SizeBytes: 1024, Duration: 50},
				},
			},
			wantErr: false,
			wantContains: []string{
				"Type,Name,Value",
				"Summary,TotalSessions,10",
				"Summary,SuccessRate,0.8500",
				"TokenUsage,InputTokens,1000",
				"AgentPerformance,agent1,5",
				"ToolExecution,Read,",
				"BashCommand,ls -la,",
				"FileOperation,/test/file.txt,",
			},
		},
		{
			name: "minimal metrics",
			metrics: &BehavioralMetrics{
				TotalSessions: 1,
				SuccessRate:   1.0,
				ErrorRate:     0.0,
			},
			wantErr: false,
			wantContains: []string{
				"Type,Name,Value",
				"Summary,TotalSessions,1",
				"Summary,SuccessRate,1.0000",
			},
		},
		{
			name:    "nil metrics",
			metrics: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := &CSVExporter{}
			result, err := exporter.Export(tt.metrics)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CSVExporter.Export() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("CSVExporter.Export() unexpected error: %v", err)
				return
			}

			// Check for expected content
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("CSVExporter.Export() missing expected content: %q", want)
				}
			}
		})
	}
}

func TestEscapeCSV(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no special chars",
			input: "simple text",
			want:  "simple text",
		},
		{
			name:  "contains comma",
			input: "value,with,commas",
			want:  "\"value,with,commas\"",
		},
		{
			name:  "contains quotes",
			input: "value\"with\"quotes",
			want:  "\"value\"\"with\"\"quotes\"",
		},
		{
			name:  "contains newline",
			input: "value\nwith\nnewline",
			want:  "\"value\nwith\nnewline\"",
		},
		{
			name:  "contains carriage return",
			input: "value\rwith\rreturn",
			want:  "\"value\rwith\rreturn\"",
		},
		{
			name:  "mixed special chars",
			input: "value,with\"mixed\nchars",
			want:  "\"value,with\"\"mixed\nchars\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeCSV(tt.input)
			if result != tt.want {
				t.Errorf("escapeCSV() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestExportToFile_CSV(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "export.csv")

	metrics := &BehavioralMetrics{
		TotalSessions: 10,
		SuccessRate:   0.8,
		ErrorRate:     0.2,
	}

	err := ExportToFile(metrics, outputPath, "csv")
	if err != nil {
		t.Errorf("ExportToFile() unexpected error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("ExportToFile() did not create output file")
	}

	// Verify file contents
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Errorf("Failed to read output file: %v", err)
	}

	contentStr := string(content)
	if !strings.HasPrefix(contentStr, "Type,Name,Value\n") {
		t.Errorf("ExportToFile() incorrect CSV header")
	}

	if !strings.Contains(contentStr, "Summary,TotalSessions,10") {
		t.Errorf("ExportToFile() incorrect file content")
	}
}

func TestExportToString_CSV(t *testing.T) {
	metrics := &BehavioralMetrics{
		TotalSessions: 5,
		SuccessRate:   0.9,
		ErrorRate:     0.1,
	}

	result, err := ExportToString(metrics, "csv")
	if err != nil {
		t.Errorf("ExportToString() unexpected error: %v", err)
	}

	if !strings.HasPrefix(result, "Type,Name,Value\n") {
		t.Errorf("ExportToString() incorrect CSV format")
	}

	if !strings.Contains(result, "Summary,TotalSessions,5") {
		t.Errorf("ExportToString() missing expected data")
	}
}

func TestMarkdownExporter_ComplexMetrics(t *testing.T) {
	metrics := &BehavioralMetrics{
		TotalSessions:   100,
		SuccessRate:     0.95,
		ErrorRate:       0.05,
		TotalErrors:     5,
		AverageDuration: 2 * time.Minute,
		TotalCost:       25.75,
		TokenUsage: TokenUsage{
			InputTokens:  50000,
			OutputTokens: 25000,
			CostUSD:      25.75,
			ModelName:    "claude-sonnet-4-5",
		},
		AgentPerformance: map[string]int{
			"backend-developer":  30,
			"frontend-developer": 25,
			"test-automator":     20,
		},
		ToolExecutions: []ToolExecution{
			{Name: "Read", Count: 200, SuccessRate: 0.99, ErrorRate: 0.01, AvgDuration: 50 * time.Millisecond},
			{Name: "Write", Count: 50, SuccessRate: 0.98, ErrorRate: 0.02, AvgDuration: 100 * time.Millisecond},
		},
		BashCommands: []BashCommand{
			{Command: "go test ./...", ExitCode: 0, Success: true, Duration: 5 * time.Second, OutputLength: 1024},
			{Command: "go build", ExitCode: 1, Success: false, Duration: 2 * time.Second, OutputLength: 512},
		},
		FileOperations: []FileOperation{
			{Type: "read", Path: "/src/main.go", Success: true, SizeBytes: 2048, Duration: 10},
			{Type: "write", Path: "/src/test.go", Success: true, SizeBytes: 1024, Duration: 20},
		},
	}

	exporter := &MarkdownExporter{IncludeTimestamp: true}
	result, err := exporter.Export(metrics)

	if err != nil {
		t.Fatalf("MarkdownExporter.Export() unexpected error: %v", err)
	}

	// Verify all sections are present
	expectedSections := []string{
		"# Behavioral Metrics Report",
		"## Summary",
		"## Token Usage",
		"## Agent Performance",
		"## Tool Executions",
		"## Bash Commands",
		"## File Operations",
	}

	for _, section := range expectedSections {
		if !strings.Contains(result, section) {
			t.Errorf("MarkdownExporter.Export() missing section: %q", section)
		}
	}

	// Verify specific data points
	expectedData := []string{
		"**Total Sessions**: 100",
		"**Success Rate**: 95.00%",
		"**Total Cost**: $25.7500",
		"backend-developer",
		"| Read |",
		"| Write |",
		"`go test ./...`",
		"`/src/main.go`",
	}

	for _, data := range expectedData {
		if !strings.Contains(result, data) {
			t.Errorf("MarkdownExporter.Export() missing data: %q", data)
		}
	}
}
