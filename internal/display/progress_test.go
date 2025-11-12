package display

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewProgressIndicator(t *testing.T) {
	tests := []struct {
		name       string
		totalFiles int
		wantPanic  bool
	}{
		{
			name:       "valid total files",
			totalFiles: 3,
			wantPanic:  false,
		},
		{
			name:       "single file",
			totalFiles: 1,
			wantPanic:  false,
		},
		{
			name:       "many files",
			totalFiles: 100,
			wantPanic:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			pi := NewProgressIndicator(&buf, tt.totalFiles)

			if pi == nil {
				t.Error("NewProgressIndicator() returned nil")
			}
			if pi.totalFiles != tt.totalFiles {
				t.Errorf("totalFiles = %d, want %d", pi.totalFiles, tt.totalFiles)
			}
			if pi.current != 0 {
				t.Errorf("current = %d, want 0", pi.current)
			}
		})
	}
}

func TestProgressIndicator_Start(t *testing.T) {
	tests := []struct {
		name       string
		totalFiles int
		wantOutput string
	}{
		{
			name:       "displays header for multiple files",
			totalFiles: 3,
			wantOutput: "Loading plan files:\n",
		},
		{
			name:       "displays header for single file",
			totalFiles: 1,
			wantOutput: "Loading plan files:\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			pi := NewProgressIndicator(&buf, tt.totalFiles)
			pi.Start()

			got := buf.String()
			if got != tt.wantOutput {
				t.Errorf("Start() output = %q, want %q", got, tt.wantOutput)
			}
		})
	}
}

func TestProgressIndicator_Step(t *testing.T) {
	tests := []struct {
		name       string
		totalFiles int
		filename   string
		stepNum    int
		wantFormat string
		wantColor  bool
	}{
		{
			name:       "first step shows [1/3] format",
			totalFiles: 3,
			filename:   "plan1.md",
			stepNum:    1,
			wantFormat: "  [1/3] plan1.md",
			wantColor:  true,
		},
		{
			name:       "second step shows [2/3] format",
			totalFiles: 3,
			filename:   "plan2.md",
			stepNum:    2,
			wantFormat: "  [2/3] plan2.md",
			wantColor:  true,
		},
		{
			name:       "third step shows [3/3] format",
			totalFiles: 3,
			filename:   "plan3.md",
			stepNum:    3,
			wantFormat: "  [3/3] plan3.md",
			wantColor:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			pi := NewProgressIndicator(&buf, tt.totalFiles)

			// Advance to correct step
			for i := 0; i < tt.stepNum; i++ {
				buf.Reset()
				pi.Step(tt.filename)
			}

			got := buf.String()

			// Check format is present
			if !strings.Contains(got, tt.wantFormat) {
				t.Errorf("Step() output missing format %q, got %q", tt.wantFormat, got)
			}

			// Check cyan ANSI color code is present
			if tt.wantColor && !strings.Contains(got, "\x1b[36m") {
				t.Errorf("Step() output missing cyan ANSI color code, got %q", got)
			}

			// Check ANSI reset is present
			if tt.wantColor && !strings.Contains(got, "\x1b[0m") {
				t.Errorf("Step() output missing ANSI reset code, got %q", got)
			}

			// Check newline is present
			if !strings.HasSuffix(got, "\n") {
				t.Errorf("Step() output missing trailing newline, got %q", got)
			}
		})
	}
}

func TestProgressIndicator_StepShowsBasenameOnly(t *testing.T) {
	tests := []struct {
		name     string
		fullPath string
		wantName string
	}{
		{
			name:     "simple filename",
			fullPath: "plan.md",
			wantName: "plan.md",
		},
		{
			name:     "relative path with directory",
			fullPath: "docs/plan.md",
			wantName: "plan.md",
		},
		{
			name:     "absolute path",
			fullPath: "/Users/harrison/Github/conductor/docs/plans/plan.md",
			wantName: "plan.md",
		},
		{
			name:     "nested directories",
			fullPath: "a/b/c/d/plan.yaml",
			wantName: "plan.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			pi := NewProgressIndicator(&buf, 1)
			pi.Step(tt.fullPath)

			got := buf.String()

			// Should contain basename only
			if !strings.Contains(got, tt.wantName) {
				t.Errorf("Step() output missing basename %q, got %q", tt.wantName, got)
			}

			// Should NOT contain full path (unless it's just a filename)
			if tt.fullPath != tt.wantName && strings.Contains(got, tt.fullPath) {
				t.Errorf("Step() output should not contain full path %q, got %q", tt.fullPath, got)
			}
		})
	}
}

func TestProgressIndicator_Complete(t *testing.T) {
	tests := []struct {
		name           string
		totalFiles     int
		wantCheckmark  bool
		wantGreenColor bool
		wantMessage    string
	}{
		{
			name:           "shows success message with checkmark",
			totalFiles:     3,
			wantCheckmark:  true,
			wantGreenColor: true,
			wantMessage:    "Loaded 3 plan files",
		},
		{
			name:           "shows success for single file",
			totalFiles:     1,
			wantCheckmark:  true,
			wantGreenColor: true,
			wantMessage:    "Loaded 1 plan files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			pi := NewProgressIndicator(&buf, tt.totalFiles)
			pi.Complete()

			got := buf.String()

			// Check for checkmark
			if tt.wantCheckmark && !strings.Contains(got, "✓") {
				t.Errorf("Complete() output missing checkmark, got %q", got)
			}

			// Check for success message
			if !strings.Contains(got, tt.wantMessage) {
				t.Errorf("Complete() output missing message %q, got %q", tt.wantMessage, got)
			}

			// Check for green ANSI color code
			if tt.wantGreenColor && !strings.Contains(got, "\x1b[32m") {
				t.Errorf("Complete() output missing green ANSI color code, got %q", got)
			}

			// Check for ANSI reset
			if tt.wantGreenColor && !strings.Contains(got, "\x1b[0m") {
				t.Errorf("Complete() output missing ANSI reset code, got %q", got)
			}

			// Check for newline
			if !strings.HasSuffix(got, "\n") {
				t.Errorf("Complete() output missing trailing newline, got %q", got)
			}
		})
	}
}

func TestProgressIndicator_FullWorkflow(t *testing.T) {
	var buf bytes.Buffer
	pi := NewProgressIndicator(&buf, 3)

	// Start
	pi.Start()
	output := buf.String()
	if !strings.Contains(output, "Loading plan files:") {
		t.Errorf("Start() missing header, got %q", output)
	}

	// Step 1
	buf.Reset()
	pi.Step("docs/plan1.md")
	output = buf.String()
	if !strings.Contains(output, "[1/3]") || !strings.Contains(output, "plan1.md") {
		t.Errorf("Step(1) missing expected format, got %q", output)
	}

	// Step 2
	buf.Reset()
	pi.Step("/Users/harrison/plan2.yaml")
	output = buf.String()
	if !strings.Contains(output, "[2/3]") || !strings.Contains(output, "plan2.yaml") {
		t.Errorf("Step(2) missing expected format, got %q", output)
	}

	// Step 3
	buf.Reset()
	pi.Step("plan3.md")
	output = buf.String()
	if !strings.Contains(output, "[3/3]") || !strings.Contains(output, "plan3.md") {
		t.Errorf("Step(3) missing expected format, got %q", output)
	}

	// Complete
	buf.Reset()
	pi.Complete()
	output = buf.String()
	if !strings.Contains(output, "✓") || !strings.Contains(output, "Loaded 3 plan files") {
		t.Errorf("Complete() missing expected format, got %q", output)
	}
}

func TestProgressIndicator_ANSIColors(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		wantCyan  bool
		wantGreen bool
	}{
		{
			name:      "Step uses cyan color",
			method:    "step",
			wantCyan:  true,
			wantGreen: false,
		},
		{
			name:      "Complete uses green color",
			method:    "complete",
			wantCyan:  false,
			wantGreen: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			pi := NewProgressIndicator(&buf, 1)

			switch tt.method {
			case "step":
				pi.Step("test.md")
			case "complete":
				pi.Complete()
			}

			got := buf.String()

			// Check for cyan ANSI code (36m)
			hasCyan := strings.Contains(got, "\x1b[36m")
			if hasCyan != tt.wantCyan {
				t.Errorf("Cyan ANSI code present = %v, want %v, output = %q", hasCyan, tt.wantCyan, got)
			}

			// Check for green ANSI code (32m)
			hasGreen := strings.Contains(got, "\x1b[32m")
			if hasGreen != tt.wantGreen {
				t.Errorf("Green ANSI code present = %v, want %v, output = %q", hasGreen, tt.wantGreen, got)
			}

			// Both methods should reset color
			if !strings.Contains(got, "\x1b[0m") {
				t.Errorf("Missing ANSI reset code, output = %q", got)
			}
		})
	}
}

func TestDisplaySingleFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantMsg  string
	}{
		{
			name:     "displays single file message with full path",
			filename: "plan.md",
			wantMsg:  "Loading plan from plan.md...",
		},
		{
			name:     "shows full path for single file",
			filename: "/Users/harrison/docs/plan.yaml",
			wantMsg:  "Loading plan from /Users/harrison/docs/plan.yaml...",
		},
		{
			name:     "handles nested paths",
			filename: "a/b/c/implementation.md",
			wantMsg:  "Loading plan from a/b/c/implementation.md...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			DisplaySingleFile(&buf, tt.filename)

			got := buf.String()

			// Check for expected message
			if !strings.Contains(got, tt.wantMsg) {
				t.Errorf("DisplaySingleFile() output = %q, want to contain %q", got, tt.wantMsg)
			}

			// Check for newline
			if !strings.HasSuffix(got, "\n") {
				t.Errorf("DisplaySingleFile() output missing trailing newline, got %q", got)
			}
		})
	}
}
