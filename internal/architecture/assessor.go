package architecture

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/budget"
	"github.com/harrison/conductor/internal/claude"
	"github.com/harrison/conductor/internal/models"
)

// Assessor evaluates tasks for architectural impact using Claude CLI.
// Embeds claude.Service for CLI invocation with rate limit handling.
type Assessor struct {
	claude.Service
}

// NewAssessor creates an assessor with the specified timeout.
// The timeout parameter controls how long to wait for Claude CLI responses.
// Use config.DefaultTimeoutsConfig().LLM for the standard timeout value.
func NewAssessor(timeout time.Duration, logger budget.WaiterLogger) *Assessor {
	return &Assessor{
		Service: *claude.NewService(timeout, logger),
	}
}

// NewAssessorWithInvoker creates an assessor using an external Invoker.
// This allows sharing a single Invoker across multiple components for consistent
// configuration and rate limit handling. The invoker should already have Timeout
// and Logger configured.
func NewAssessorWithInvoker(inv *claude.Invoker) *Assessor {
	return &Assessor{
		Service: *claude.NewServiceWithInvoker(inv),
	}
}

// Assess evaluates a task against the 6-question architecture framework
func (a *Assessor) Assess(ctx context.Context, task models.Task) (*AssessmentResult, error) {
	prompt := a.buildPrompt(task)

	var result AssessmentResult
	if err := a.InvokeAndParseWithFallback(ctx, prompt, AssessmentSchema(), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (a *Assessor) buildPrompt(task models.Task) string {
	files := strings.Join(task.Files, ", ")
	criteria := strings.Join(task.SuccessCriteria, "\n- ")

	// Use Prompt as the description (primary task content)
	description := task.Prompt
	if len(description) > 500 {
		description = description[:500] + "..." // Truncate long prompts
	}

	return fmt.Sprintf(`<architecture_assessment>
  <task_context>
    <name>%s</name>
    <description>%s</description>
    <files>%s</files>
    <success_criteria>%s</success_criteria>
  </task_context>

  <instructions>
    Analyze this task for architectural impact using the 6-question framework.
    Answer each question with true (YES) or false (NO), providing:
    <requirements>
      <item>Specific examples from THIS task (not generic examples)</item>
      <item>Confidence score (0.0-1.0)</item>
      <item>Brief reasoning</item>
    </requirements>
  </instructions>

  <assessment_questions>
    <question id="1" name="core_infrastructure">
      <prompt>Does this touch core/shared infrastructure?</prompt>
      <yes_examples>caching layers, auth services, database connections, shared utilities</yes_examples>
      <no_examples>single form validation, typo fixes, isolated component changes</no_examples>
    </question>

    <question id="2" name="reuse_concerns">
      <prompt>Are there reuse concerns?</prompt>
      <yes_examples>first-of-kind feature (sets pattern), reusable components, shared libraries</yes_examples>
      <no_examples>one-off fixes, local helpers, single-use utilities</no_examples>
    </question>

    <question id="3" name="new_abstractions">
      <prompt>Does this introduce new abstractions or patterns?</prompt>
      <yes_examples>new base classes, design patterns, error handling strategies, interfaces</yes_examples>
      <no_examples>utility functions, simple helpers, concrete implementations</no_examples>
    </question>

    <question id="4" name="api_contracts">
      <prompt>Are there API contract decisions?</prompt>
      <yes_examples>new endpoints, parameter placement decisions, schema changes, public interfaces</yes_examples>
      <no_examples>internal method changes, local variable renames, private helpers</no_examples>
    </question>

    <question id="5" name="framework_lifecycle">
      <prompt>Does this integrate with framework lifecycle?</prompt>
      <yes_examples>startup hooks, shutdown handlers, middleware registration, plugin systems</yes_examples>
      <no_examples>pure functions, isolated utilities, standalone scripts</no_examples>
    </question>

    <question id="6" name="cross_cutting_concerns">
      <prompt>Are there cross-cutting concerns?</prompt>
      <yes_examples>logging strategies, rate limiting, metrics collection, security policies</yes_examples>
      <no_examples>single component changes, isolated fixes, local error handling</no_examples>
    </question>
  </assessment_questions>

  <decision_rule>
    <condition>If ANY question = true</condition>
    <result>requires_review = true</result>
    <condition>If ALL questions = false</condition>
    <result>requires_review = false, provide skip_justification</result>
  </decision_rule>

  <output_format>Respond with JSON only.</output_format>
</architecture_assessment>`, task.Name, description, files, criteria)
}

// AssessmentSchema returns the JSON schema for Claude response enforcement
func AssessmentSchema() string {
	return `{
  "type": "object",
  "properties": {
    "core_infrastructure": {
      "type": "object",
      "properties": {
        "answer": {"type": "boolean"},
        "confidence": {"type": "number", "minimum": 0, "maximum": 1},
        "reasoning": {"type": "string"},
        "examples": {"type": "string"}
      },
      "required": ["answer", "confidence", "reasoning"]
    },
    "reuse_concerns": {
      "type": "object",
      "properties": {
        "answer": {"type": "boolean"},
        "confidence": {"type": "number", "minimum": 0, "maximum": 1},
        "reasoning": {"type": "string"},
        "examples": {"type": "string"}
      },
      "required": ["answer", "confidence", "reasoning"]
    },
    "new_abstractions": {
      "type": "object",
      "properties": {
        "answer": {"type": "boolean"},
        "confidence": {"type": "number", "minimum": 0, "maximum": 1},
        "reasoning": {"type": "string"},
        "examples": {"type": "string"}
      },
      "required": ["answer", "confidence", "reasoning"]
    },
    "api_contracts": {
      "type": "object",
      "properties": {
        "answer": {"type": "boolean"},
        "confidence": {"type": "number", "minimum": 0, "maximum": 1},
        "reasoning": {"type": "string"},
        "examples": {"type": "string"}
      },
      "required": ["answer", "confidence", "reasoning"]
    },
    "framework_lifecycle": {
      "type": "object",
      "properties": {
        "answer": {"type": "boolean"},
        "confidence": {"type": "number", "minimum": 0, "maximum": 1},
        "reasoning": {"type": "string"},
        "examples": {"type": "string"}
      },
      "required": ["answer", "confidence", "reasoning"]
    },
    "cross_cutting_concerns": {
      "type": "object",
      "properties": {
        "answer": {"type": "boolean"},
        "confidence": {"type": "number", "minimum": 0, "maximum": 1},
        "reasoning": {"type": "string"},
        "examples": {"type": "string"}
      },
      "required": ["answer", "confidence", "reasoning"]
    },
    "requires_review": {"type": "boolean"},
    "overall_confidence": {"type": "number", "minimum": 0, "maximum": 1},
    "summary": {"type": "string"},
    "skip_justification": {"type": "string"}
  },
  "required": ["core_infrastructure", "reuse_concerns", "new_abstractions", "api_contracts", "framework_lifecycle", "cross_cutting_concerns", "requires_review", "overall_confidence", "summary"]
}`
}
