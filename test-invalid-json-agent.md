---
name: test-invalid-json-agent
description: Test agent that returns invalid JSON with verdict in metadata
tools: []
---

# Test Invalid JSON Agent

This agent deliberately returns invalid JSON on the first invocation to test the retry mechanism.

When reviewing code, respond with this INVALID JSON (verdict in metadata):

```json
{
  "metadata": {
    "verdict": "GREEN"
  },
  "feedback": "Code looks good but JSON is invalid"
}
```

This tests whether the QC system will:
1. Detect the invalid JSON schema
2. Retry with a schema reminder
3. Extract verdict from metadata as fallback

On retry, respond with valid JSON:

```json
{
  "verdict": "GREEN",
  "feedback": "Code looks good with valid JSON schema",
  "issues": [],
  "recommendations": [],
  "should_retry": false,
  "suggested_agent": ""
}
```
