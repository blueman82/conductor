package models

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCommitSpec_Validate(t *testing.T) {
	tests := []struct {
		name        string
		spec        CommitSpec
		expectError bool
	}{
		{
			name: "valid with type and message",
			spec: CommitSpec{
				Type:    "feat",
				Message: "add new feature",
			},
			expectError: false,
		},
		{
			name: "valid with message only",
			spec: CommitSpec{
				Message: "update documentation",
			},
			expectError: false,
		},
		{
			name: "valid with all fields",
			spec: CommitSpec{
				Type:    "fix",
				Message: "fix bug in parser",
				Body:    "The parser was not handling edge cases correctly.",
				Files:   []string{"internal/parser/*.go"},
			},
			expectError: false,
		},
		{
			name: "invalid - missing message",
			spec: CommitSpec{
				Type: "feat",
			},
			expectError: true,
		},
		{
			name:        "invalid - empty spec",
			spec:        CommitSpec{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCommitSpec_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		spec     CommitSpec
		expected bool
	}{
		{
			name:     "empty spec",
			spec:     CommitSpec{},
			expected: true,
		},
		{
			name: "has type only",
			spec: CommitSpec{
				Type: "feat",
			},
			expected: false,
		},
		{
			name: "has message only",
			spec: CommitSpec{
				Message: "fix bug",
			},
			expected: false,
		},
		{
			name: "has body only",
			spec: CommitSpec{
				Body: "Extended description",
			},
			expected: false,
		},
		{
			name: "has files only",
			spec: CommitSpec{
				Files: []string{"file.go"},
			},
			expected: false,
		},
		{
			name: "has all fields",
			spec: CommitSpec{
				Type:    "feat",
				Message: "add feature",
				Body:    "Details",
				Files:   []string{"*.go"},
			},
			expected: false,
		},
		{
			name: "empty files slice is still empty",
			spec: CommitSpec{
				Files: []string{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.spec.IsEmpty()
			if result != tt.expected {
				t.Errorf("IsEmpty() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCommitSpec_BuildCommitMessage(t *testing.T) {
	tests := []struct {
		name     string
		spec     CommitSpec
		expected string
	}{
		{
			name: "with type",
			spec: CommitSpec{
				Type:    "feat",
				Message: "add new feature",
			},
			expected: "feat: add new feature",
		},
		{
			name: "without type",
			spec: CommitSpec{
				Message: "update documentation",
			},
			expected: "update documentation",
		},
		{
			name: "fix type",
			spec: CommitSpec{
				Type:    "fix",
				Message: "resolve memory leak",
			},
			expected: "fix: resolve memory leak",
		},
		{
			name: "chore type",
			spec: CommitSpec{
				Type:    "chore",
				Message: "update dependencies",
			},
			expected: "chore: update dependencies",
		},
		{
			name:     "empty message",
			spec:     CommitSpec{},
			expected: "",
		},
		{
			name: "body is ignored",
			spec: CommitSpec{
				Type:    "docs",
				Message: "add README",
				Body:    "This is the body",
			},
			expected: "docs: add README",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.spec.BuildCommitMessage()
			if result != tt.expected {
				t.Errorf("BuildCommitMessage() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestCommitSpec_BuildFullCommitMessage(t *testing.T) {
	tests := []struct {
		name     string
		spec     CommitSpec
		expected string
	}{
		{
			name: "with type and body",
			spec: CommitSpec{
				Type:    "feat",
				Message: "add new feature",
				Body:    "This adds a new feature for X.",
			},
			expected: "feat: add new feature\n\nThis adds a new feature for X.",
		},
		{
			name: "without type but with body",
			spec: CommitSpec{
				Message: "update docs",
				Body:    "Expanded the documentation.",
			},
			expected: "update docs\n\nExpanded the documentation.",
		},
		{
			name: "without body",
			spec: CommitSpec{
				Type:    "fix",
				Message: "fix bug",
			},
			expected: "fix: fix bug",
		},
		{
			name: "message only",
			spec: CommitSpec{
				Message: "quick fix",
			},
			expected: "quick fix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.spec.BuildFullCommitMessage()
			if result != tt.expected {
				t.Errorf("BuildFullCommitMessage() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestCommitSpec_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expected    CommitSpec
		expectError bool
	}{
		{
			name: "full commit spec",
			yaml: `
type: feat
message: add CommitSpec types
body: |
  This adds the foundational types for commit specification.
files:
  - internal/models/commit.go
  - internal/models/commit_test.go
`,
			expected: CommitSpec{
				Type:    "feat",
				Message: "add CommitSpec types",
				Body:    "This adds the foundational types for commit specification.\n",
				Files: []string{
					"internal/models/commit.go",
					"internal/models/commit_test.go",
				},
			},
			expectError: false,
		},
		{
			name: "minimal commit spec",
			yaml: `
message: quick fix
`,
			expected: CommitSpec{
				Message: "quick fix",
			},
			expectError: false,
		},
		{
			name: "with glob patterns",
			yaml: `
type: refactor
message: restructure parser
files:
  - "internal/parser/*.go"
  - "internal/parser/**/*_test.go"
`,
			expected: CommitSpec{
				Type:    "refactor",
				Message: "restructure parser",
				Files: []string{
					"internal/parser/*.go",
					"internal/parser/**/*_test.go",
				},
			},
			expectError: false,
		},
		{
			name:        "empty yaml is valid",
			yaml:        `{}`,
			expected:    CommitSpec{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var spec CommitSpec
			err := yaml.Unmarshal([]byte(tt.yaml), &spec)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if spec.Type != tt.expected.Type {
				t.Errorf("Type = %q, expected %q", spec.Type, tt.expected.Type)
			}
			if spec.Message != tt.expected.Message {
				t.Errorf("Message = %q, expected %q", spec.Message, tt.expected.Message)
			}
			if spec.Body != tt.expected.Body {
				t.Errorf("Body = %q, expected %q", spec.Body, tt.expected.Body)
			}
			if len(spec.Files) != len(tt.expected.Files) {
				t.Errorf("Files length = %d, expected %d", len(spec.Files), len(tt.expected.Files))
			} else {
				for i, f := range spec.Files {
					if f != tt.expected.Files[i] {
						t.Errorf("Files[%d] = %q, expected %q", i, f, tt.expected.Files[i])
					}
				}
			}
		})
	}
}

func TestCommitSpec_BackwardCompatibility(t *testing.T) {
	// Test that tasks without commit specs continue to work
	// This ensures we don't break existing YAML plans

	t.Run("empty commit spec is backward compatible", func(t *testing.T) {
		spec := CommitSpec{}

		if !spec.IsEmpty() {
			t.Error("empty spec should be detected as empty")
		}

		// Validation should fail for empty spec, but IsEmpty() check
		// allows callers to skip validation when no commit is expected
		err := spec.Validate()
		if err == nil {
			t.Error("empty spec should fail validation (message required)")
		}
	})

	t.Run("pointer nil check pattern", func(t *testing.T) {
		// Common usage pattern: check if commit spec is nil or empty
		var specPtr *CommitSpec

		if specPtr != nil && !specPtr.IsEmpty() {
			t.Error("nil pointer should be treated as no commit")
		}

		specPtr = &CommitSpec{}
		if specPtr != nil && !specPtr.IsEmpty() {
			t.Error("empty spec should be treated as no commit")
		}

		specPtr = &CommitSpec{Message: "test"}
		if specPtr == nil || specPtr.IsEmpty() {
			t.Error("non-empty spec should be processed")
		}
	})
}

func TestCommitSpec_ConventionalCommitTypes(t *testing.T) {
	// Verify common conventional commit types work correctly
	types := []string{
		"feat",     // New feature
		"fix",      // Bug fix
		"docs",     // Documentation
		"style",    // Formatting, missing semi-colons, etc
		"refactor", // Code refactoring
		"test",     // Adding tests
		"chore",    // Maintenance
		"perf",     // Performance improvement
		"ci",       // CI/CD changes
		"build",    // Build system changes
		"revert",   // Revert previous commit
	}

	for _, commitType := range types {
		t.Run(commitType, func(t *testing.T) {
			spec := CommitSpec{
				Type:    commitType,
				Message: "test message",
			}

			msg := spec.BuildCommitMessage()
			expected := commitType + ": test message"
			if msg != expected {
				t.Errorf("BuildCommitMessage() = %q, expected %q", msg, expected)
			}
		})
	}
}
