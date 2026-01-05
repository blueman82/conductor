package agent

import (
	"fmt"
	"strings"
)

// claude4Enhancements contains Claude 4-specific prompt patterns
// per Anthropic best practices documentation
const claude4Enhancements = `<context_awareness>
Your context window will be automatically managed. Do not stop tasks early
due to token budget concerns. Complete tasks fully and persist progress.
</context_awareness>

<thinking_guidance>
After receiving tool results, carefully reflect on their quality and
determine optimal next steps before proceeding. Use your thinking to
plan and iterate based on new information.
</thinking_guidance>

<anti_hallucination>
NEVER speculate about code you have not read. Use the Read tool to
examine files BEFORE making claims about code. If unsure, investigate
first rather than guessing.
</anti_hallucination>

<parallel_tool_calls>
When multiple independent tool operations are needed (e.g., reading several
files, running multiple searches), execute them in parallel rather than
sequentially. Only serialize operations that have dependencies.
</parallel_tool_calls>
`

// XMLTag wraps content in XML tags: <name>content</name>
func XMLTag(name, content string) string {
	return fmt.Sprintf("<%s>%s</%s>", name, content, name)
}

// XMLSection creates a section with proper formatting
// Output: <name>\ncontent\n</name>
func XMLSection(name, content string) string {
	return fmt.Sprintf("<%s>\n%s\n</%s>", name, strings.TrimSpace(content), name)
}

// XMLList creates an XML list with item elements
// Output: <name>\n<item>a</item>\n<item>b</item>\n</name>
func XMLList(name string, items []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<%s>\n", name))
	for _, item := range items {
		sb.WriteString(fmt.Sprintf("<item>%s</item>\n", item))
	}
	sb.WriteString(fmt.Sprintf("</%s>", name))
	return sb.String()
}

// EnhancePromptForClaude4 prepends Claude 4 enhancements to any prompt
func EnhancePromptForClaude4(prompt string) string {
	return claude4Enhancements + "\n" + prompt
}
