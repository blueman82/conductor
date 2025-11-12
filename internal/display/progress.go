package display

import (
	"fmt"
	"io"
	"path/filepath"
)

// ProgressIndicator manages multi-step progress display with ANSI colors
type ProgressIndicator struct {
	writer     io.Writer
	totalFiles int
	current    int
}

// NewProgressIndicator creates a new progress indicator
func NewProgressIndicator(w io.Writer, total int) *ProgressIndicator {
	return &ProgressIndicator{
		writer:     w,
		totalFiles: total,
		current:    0,
	}
}

// Start displays the header message
func (p *ProgressIndicator) Start() {
	fmt.Fprintf(p.writer, "Loading %d plan files...\n", p.totalFiles)
}

// Step displays progress for current item: [N/Total] filename (blue)
func (p *ProgressIndicator) Step(filename string) {
	p.current++
	basename := filepath.Base(filename)
	// Output with blue ANSI around entire line for visibility
	fmt.Fprintf(p.writer, "\x1b[34m  [%d/%d] %s\x1b[0m\n", p.current, p.totalFiles, basename)
}

// Complete displays success message with green checkmark
func (p *ProgressIndicator) Complete() {
	fmt.Fprintf(p.writer, "\x1b[32m✓\x1b[0m Successfully loaded %d plan files\n", p.totalFiles)
}

// DisplaySingleFile shows simple loading message for single file
func DisplaySingleFile(w io.Writer, filename string) {
	basename := filepath.Base(filename)
	fmt.Fprintf(w, "Loading plan file: %s...\n", basename)
}
