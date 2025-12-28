package models

import (
	"errors"
	"fmt"
)

// CommitSpec represents a commit specification parsed from YAML plans.
// Used for VERIFICATION purposes - checking that agents created the expected commit.
// Agents are instructed to commit via the prompt; conductor only verifies.
//
// YAML structure:
//
//	commit:
//	  type: "feat"      # Conventional commit type
//	  message: "description"
//	  body: "optional extended description"
//	  files:
//	    - "path/to/file.go"
//	    - "path/**/*.go"  # Glob patterns supported
type CommitSpec struct {
	// Type is the conventional commit type (feat, fix, docs, style, refactor, test, chore, etc.)
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Message is the commit message (required when spec is present)
	Message string `yaml:"message" json:"message"`

	// Body is the optional extended commit description
	Body string `yaml:"body,omitempty" json:"body,omitempty"`

	// Files is the list of expected file patterns (supports glob patterns)
	Files []string `yaml:"files,omitempty" json:"files,omitempty"`
}

// Validate checks that the CommitSpec has all required fields.
// Returns an error if Message is empty since commit message is mandatory.
func (c *CommitSpec) Validate() error {
	if c.Message == "" {
		return errors.New("commit message is required")
	}
	return nil
}

// IsEmpty returns true if all fields are zero values.
// Used for backward compatibility when no commit spec is provided.
func (c *CommitSpec) IsEmpty() bool {
	return c.Type == "" && c.Message == "" && c.Body == "" && len(c.Files) == 0
}

// BuildCommitMessage formats the commit message in conventional commit format.
// If Type is present, returns "type: message", otherwise just the message.
func (c *CommitSpec) BuildCommitMessage() string {
	if c.Type != "" {
		return fmt.Sprintf("%s: %s", c.Type, c.Message)
	}
	return c.Message
}

// BuildFullCommitMessage returns the complete commit message including body if present.
// Format: "type: message\n\nbody" or "message\n\nbody" if no type.
func (c *CommitSpec) BuildFullCommitMessage() string {
	msg := c.BuildCommitMessage()
	if c.Body != "" {
		return msg + "\n\n" + c.Body
	}
	return msg
}
