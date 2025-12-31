package agent

import (
	"strings"
	"testing"
)

func TestXMLTag(t *testing.T) {
	tests := []struct {
		name     string
		tagName  string
		content  string
		expected string
	}{
		{
			name:     "simple content",
			tagName:  "test",
			content:  "content",
			expected: "<test>content</test>",
		},
		{
			name:     "empty content",
			tagName:  "empty",
			content:  "",
			expected: "<empty></empty>",
		},
		{
			name:     "content with spaces",
			tagName:  "message",
			content:  "hello world",
			expected: "<message>hello world</message>",
		},
		{
			name:     "multiline content",
			tagName:  "block",
			content:  "line1\nline2",
			expected: "<block>line1\nline2</block>",
		},
		{
			name:     "content with special chars",
			tagName:  "code",
			content:  "if (x > 0) { return true; }",
			expected: "<code>if (x > 0) { return true; }</code>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := XMLTag(tt.tagName, tt.content)
			if result != tt.expected {
				t.Errorf("XMLTag(%q, %q) = %q, want %q", tt.tagName, tt.content, result, tt.expected)
			}
		})
	}
}

func TestXMLSection(t *testing.T) {
	tests := []struct {
		name    string
		tagName string
		content string
	}{
		{
			name:    "simple section",
			tagName: "section",
			content: "content here",
		},
		{
			name:    "multiline section",
			tagName: "code",
			content: "line1\nline2\nline3",
		},
		{
			name:    "section with leading/trailing whitespace",
			tagName: "trimmed",
			content: "  content with spaces  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := XMLSection(tt.tagName, tt.content)

			// Verify starts with opening tag
			if !strings.HasPrefix(result, "<"+tt.tagName+">") {
				t.Errorf("XMLSection should start with opening tag <%s>, got: %s", tt.tagName, result[:min(len(result), 50)])
			}

			// Verify ends with closing tag
			if !strings.HasSuffix(result, "</"+tt.tagName+">") {
				t.Errorf("XMLSection should end with closing tag </%s>, got: %s", tt.tagName, result[max(0, len(result)-50):])
			}

			// Verify contains newlines (for proper formatting)
			if !strings.Contains(result, "\n") {
				t.Error("XMLSection should contain newlines for proper formatting")
			}

			// Verify content is trimmed
			trimmedContent := strings.TrimSpace(tt.content)
			if !strings.Contains(result, trimmedContent) {
				t.Errorf("XMLSection should contain trimmed content %q", trimmedContent)
			}
		})
	}
}

func TestXMLList(t *testing.T) {
	tests := []struct {
		name      string
		listName  string
		items     []string
		wantItems []string
	}{
		{
			name:      "simple list",
			listName:  "items",
			items:     []string{"a", "b", "c"},
			wantItems: []string{"<item>a</item>", "<item>b</item>", "<item>c</item>"},
		},
		{
			name:      "empty list",
			listName:  "empty",
			items:     []string{},
			wantItems: []string{},
		},
		{
			name:      "single item",
			listName:  "single",
			items:     []string{"only"},
			wantItems: []string{"<item>only</item>"},
		},
		{
			name:      "items with spaces",
			listName:  "commands",
			items:     []string{"go test ./...", "go build -o main"},
			wantItems: []string{"<item>go test ./...</item>", "<item>go build -o main</item>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := XMLList(tt.listName, tt.items)

			// Verify starts with list tag
			if !strings.HasPrefix(result, "<"+tt.listName+">") {
				t.Errorf("XMLList should start with opening tag <%s>", tt.listName)
			}

			// Verify ends with closing tag
			if !strings.HasSuffix(result, "</"+tt.listName+">") {
				t.Errorf("XMLList should end with closing tag </%s>", tt.listName)
			}

			// Verify all items are wrapped correctly
			for _, expectedItem := range tt.wantItems {
				if !strings.Contains(result, expectedItem) {
					t.Errorf("XMLList should contain %q, got: %s", expectedItem, result)
				}
			}

			// Verify item count matches
			itemCount := strings.Count(result, "<item>")
			if itemCount != len(tt.items) {
				t.Errorf("XMLList should contain %d items, found %d", len(tt.items), itemCount)
			}
		})
	}
}

func TestEnhancePromptForClaude4(t *testing.T) {
	prompt := "test prompt content"
	result := EnhancePromptForClaude4(prompt)

	// Verify all Claude 4 enhancement sections are present
	requiredSections := []struct {
		openTag  string
		closeTag string
	}{
		{"<context_awareness>", "</context_awareness>"},
		{"<thinking_guidance>", "</thinking_guidance>"},
		{"<anti_hallucination>", "</anti_hallucination>"},
	}

	for _, section := range requiredSections {
		if !strings.Contains(result, section.openTag) {
			t.Errorf("EnhancePromptForClaude4 missing opening tag: %s", section.openTag)
		}
		if !strings.Contains(result, section.closeTag) {
			t.Errorf("EnhancePromptForClaude4 missing closing tag: %s", section.closeTag)
		}
	}

	// Verify original prompt is preserved
	if !strings.Contains(result, prompt) {
		t.Errorf("EnhancePromptForClaude4 should preserve original prompt %q", prompt)
	}

	// Verify prompt comes after enhancements
	enhancementEnd := strings.LastIndex(result, "</anti_hallucination>")
	promptStart := strings.Index(result, prompt)
	if promptStart < enhancementEnd {
		t.Error("Original prompt should come after enhancement tags")
	}

	// Verify specific guidance content is present
	guidanceChecks := []string{
		"context window will be automatically managed",
		"carefully reflect on their quality",
		"NEVER speculate about code you have not read",
		"Use the Read tool",
	}

	for _, check := range guidanceChecks {
		if !strings.Contains(result, check) {
			t.Errorf("EnhancePromptForClaude4 missing guidance content: %q", check)
		}
	}
}

func TestEnhancePromptForClaude4_EmptyPrompt(t *testing.T) {
	result := EnhancePromptForClaude4("")

	// Even with empty prompt, should have all enhancement sections
	if !strings.Contains(result, "<context_awareness>") {
		t.Error("EnhancePromptForClaude4 should include context_awareness even with empty prompt")
	}
	if !strings.Contains(result, "<thinking_guidance>") {
		t.Error("EnhancePromptForClaude4 should include thinking_guidance even with empty prompt")
	}
	if !strings.Contains(result, "<anti_hallucination>") {
		t.Error("EnhancePromptForClaude4 should include anti_hallucination even with empty prompt")
	}
}

func TestEnhancePromptForClaude4_MultilinePrompt(t *testing.T) {
	prompt := `This is a multiline prompt.
It has several lines.
And various content.`

	result := EnhancePromptForClaude4(prompt)

	// Verify entire multiline prompt is preserved
	if !strings.Contains(result, prompt) {
		t.Error("EnhancePromptForClaude4 should preserve entire multiline prompt")
	}
}

func TestXMLSection_Format(t *testing.T) {
	// Test the exact format: <name>\ncontent\n</name>
	result := XMLSection("test", "content")
	expected := "<test>\ncontent\n</test>"

	if result != expected {
		t.Errorf("XMLSection format incorrect:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestXMLTag_NoMarkdown(t *testing.T) {
	result := XMLTag("header", "Some content")

	// Verify no Markdown formatting
	if strings.Contains(result, "##") {
		t.Error("XMLTag should not contain Markdown headers (##)")
	}
	if strings.Contains(result, "**") {
		t.Error("XMLTag should not contain Markdown bold (**)")
	}
	if strings.Contains(result, "```") {
		t.Error("XMLTag should not contain Markdown code blocks (```)")
	}
}

func TestXMLSection_NoMarkdown(t *testing.T) {
	result := XMLSection("section", "Content with stuff")

	// Verify no Markdown formatting
	if strings.Contains(result, "##") {
		t.Error("XMLSection should not contain Markdown headers (##)")
	}
	if strings.Contains(result, "**") {
		t.Error("XMLSection should not contain Markdown bold (**)")
	}
}

func TestXMLList_NoMarkdown(t *testing.T) {
	items := []string{"item1", "item2", "item3"}
	result := XMLList("list", items)

	// Verify no Markdown formatting
	if strings.Contains(result, "##") {
		t.Error("XMLList should not contain Markdown headers (##)")
	}
	if strings.Contains(result, "- ") && !strings.Contains(result, "<item>") {
		t.Error("XMLList should use XML tags, not Markdown bullet points")
	}
}

// min and max helper functions for string slicing safety
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
