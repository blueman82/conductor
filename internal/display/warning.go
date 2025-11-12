package display

import (
	"fmt"
	"io"
	"strings"
)

// Warning represents a user-facing warning message
type Warning struct {
	Title      string   // Main warning title
	Message    string   // Detailed explanation (optional)
	Files      []string // Related files (optional)
	Suggestion string   // Action to take (optional)
}

// Display shows a formatted warning in yellow
func (w Warning) Display(out io.Writer) {
	var b strings.Builder

	// Start with yellow color, emoji, and title
	b.WriteString("\x1b[33m")
	b.WriteString("⚠️  Warning: ")
	b.WriteString(w.Title)
	b.WriteString("\n")

	// Add message with 4-space indent if present
	if w.Message != "" {
		b.WriteString("    ")
		b.WriteString(w.Message)
		b.WriteString("\n")
	}

	// Add files with proper singular/plural and indentation
	if len(w.Files) > 0 {
		b.WriteString("    ")
		if len(w.Files) == 1 {
			b.WriteString("Affected file:\n")
		} else {
			b.WriteString("Affected files:\n")
		}

		for i, file := range w.Files {
			b.WriteString("      ")
			b.WriteString(fmt.Sprintf("%d. %s", i+1, file))
			b.WriteString("\n")
		}
	}

	// Add suggestion with 4-space indent if present
	if w.Suggestion != "" {
		b.WriteString("    Suggestion:\n")
		b.WriteString("    ")
		b.WriteString(w.Suggestion)
		b.WriteString("\n")
	}

	// End with reset code
	b.WriteString("\x1b[0m")

	// Write final output
	fmt.Fprint(out, b.String())
}

// WarnNumberedFiles creates a warning for numbered plan files
func WarnNumberedFiles(title string, files []string) Warning {
	return Warning{
		Title: title,
		Files: files,
	}
}
