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
	fmt.Fprintf(p.writer, "Loading plan files:\n")
}

// Step displays progress for current item: [N/Total] filename (cyan)
func (p *ProgressIndicator) Step(filename string) {
	p.current++
	basename := filepath.Base(filename)
	// Output with cyan ANSI around entire line for visibility
	fmt.Fprintf(p.writer, "\x1b[36m  [%d/%d] %s\x1b[0m\n", p.current, p.totalFiles, basename)
}

// Complete displays success message with green checkmark
func (p *ProgressIndicator) Complete() {
	fmt.Fprintf(p.writer, "\x1b[32m✓\x1b[0m Loaded %d plan files\n", p.totalFiles)
}

// DisplaySingleFile shows simple loading message for single file
func DisplaySingleFile(w io.Writer, filename string) {
	fmt.Fprintf(w, "Loading plan from %s...\n", filename)
}
