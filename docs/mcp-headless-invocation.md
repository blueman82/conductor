# MCP Headless Invocation Guide

**Date**: November 10, 2025
**Author**: Research documentation from Claude Code experimentation
**Purpose**: Document how to invoke Model Context Protocol (MCP) servers in headless/non-interactive mode

## Executive Summary

MCP servers can be invoked from the command line in headless mode using the `claude` CLI with specific flags. The critical requirement is using `--mcp-config` to explicitly load MCP server configurations, as auto-loading from default locations does NOT work in print/headless mode.

## Key Discovery

**MCP servers do NOT auto-load in headless mode.** You must explicitly specify the configuration file using `--mcp-config`.

## Working Command Pattern

```bash
claude -p "<your prompt>" \
  --mcp-config ~/.claude/config/mcp.json \
  --allowedTools "mcp__<server-name>" \
  --output-format json \
  --max-turns 5
```

## Critical Flags Explained

### `--mcp-config <path>`
**Required for headless MCP invocation**

- Explicitly loads MCP servers from specified JSON configuration file
- Without this flag, MCP servers will NOT be available in print mode
- Can load multiple config files: `--mcp-config file1.json file2.json`
- Use `--strict-mcp-config` to ONLY load servers from specified configs (ignore all others)

### `--allowedTools <tools>`
**Controls which MCP tools are accessible**

- Format: Space-separated or comma-separated list
- Tool naming: `mcp__<servername>` or `mcp__<servername>__<toolname>`
- Wildcards NOT supported - must list explicitly
- Example: `--allowedTools "mcp__ai-counsel,mcp__slack__send_message"`

### `-p` or `--print`
**Non-interactive execution mode**

- Execute prompt and exit (don't enter interactive REPL)
- Essential for automation and scripting
- Returns output to stdout

### `--output-format <format>`
**Controls output structure**

- Options: `text`, `json`, `stream-json`
- `json` provides structured output with metadata (cost, tokens, session ID)
- `text` provides raw output only

### `--max-turns <number>`
**Limits agentic execution rounds**

- Default: varies by mode
- For multi-step MCP workflows, set appropriately (5-10 recommended)
- Prevents infinite loops in agent reasoning

## Real-World Example: AI Counsel MCP Server

### Configuration File
Located at `~/.claude/config/mcp.json`:

```json
{
  "mcpServers": {
    "ai-counsel": {
      "type": "stdio",
      "command": "/Users/harrison/Github/ai-counsel/.venv/bin/python",
      "args": ["/Users/harrison/Github/ai-counsel/server.py"],
      "env": {}
    }
  }
}
```

### Successful Invocation
```bash
claude -p 'Use the deliberate tool from ai-counsel to analyze: Should we use REST or GraphQL? Participants: claude sonnet and gemini flash, 2 rounds, conference mode' \
  --mcp-config /Users/harrison/.claude/config/mcp.json \
  --allowedTools "mcp__ai-counsel" \
  --output-format json \
  --max-turns 5
```

### Execution Results
- **Duration**: ~178 seconds (3 minutes)
- **MCP Server**: Successfully loaded and invoked
- **Models Used**: Claude Sonnet 4.5 & Gemini 2.0 Flash
- **Rounds**: 2 rounds of deliberation
- **Convergence**: 91.4% similarity (high consensus)
- **Transcript**: Auto-saved to disk
- **Output**: Full JSON with structured results

### Output Structure (JSON format)
```json
{
  "type": "result",
  "subtype": "success",
  "is_error": false,
  "duration_ms": 178524,
  "num_turns": 2,
  "result": "<deliberation summary and findings>",
  "session_id": "e313fe28-ff0a-48ca-9238-0c259e38118f",
  "total_cost_usd": 0.058765,
  "usage": {
    "input_tokens": 4,
    "cache_creation_input_tokens": 36640,
    "cache_read_input_tokens": 29270,
    "output_tokens": 1150
  }
}
```

## MCP Configuration File Format

### Stdio Transport (Local Processes)
```json
{
  "mcpServers": {
    "server-name": {
      "type": "stdio",
      "command": "/path/to/executable",
      "args": ["arg1", "arg2"],
      "env": {
        "API_KEY": "value",
        "OTHER_VAR": "value"
      }
    }
  }
}
```

### HTTP Transport (Remote Servers)
```json
{
  "mcpServers": {
    "server-name": {
      "type": "http",
      "url": "https://api.example.com/mcp",
      "headers": {
        "Authorization": "Bearer token"
      }
    }
  }
}
```

### SSE Transport (Deprecated but Supported)
```json
{
  "mcpServers": {
    "server-name": {
      "type": "sse",
      "url": "https://api.example.com/sse",
      "headers": {
        "Authorization": "Bearer token"
      }
    }
  }
}
```

## MCP Tool Naming Convention

Tools from MCP servers use this naming pattern:

### Full Tool Reference
```
mcp__<servername>__<toolname>
```

Example: `mcp__ai-counsel__deliberate`

### Server Wildcard
```
mcp__<servername>
```

Grants access to ALL tools from that server.

Example: `mcp__ai-counsel` (allows all ai-counsel tools)

## Advanced Usage Patterns

### Multiple MCP Servers
```bash
claude -p "Your prompt" \
  --mcp-config ~/.claude/config/mcp.json \
  --allowedTools "mcp__ai-counsel,mcp__github,mcp__slack" \
  --output-format json
```

### Combining with Other Tools
```bash
claude -p "Your prompt" \
  --allowedTools "Bash,Read,Write,mcp__datadog" \
  --mcp-config monitoring-tools.json \
  --output-format json
```

### Strict MCP Mode
```bash
# ONLY use servers from specified config, ignore all others
claude -p "Your prompt" \
  --strict-mcp-config \
  --mcp-config /path/to/specific-config.json \
  --allowedTools "mcp__specific-server" \
  --output-format json
```

### Permission Management
```bash
# Skip permission prompts (use carefully!)
claude -p "Your prompt" \
  --mcp-config ~/.claude/config/mcp.json \
  --dangerously-skip-permissions \
  --output-format json
```

### Debugging MCP Issues
```bash
# Enable debug mode for troubleshooting
claude -p "Your prompt" \
  --mcp-config ~/.claude/config/mcp.json \
  --debug \
  --output-format json
```

## Common Patterns for Automation

### SRE Incident Response
```bash
claude -p "Incident: Payment API returning 500 errors (Severity: high)" \
  --append-system-prompt "You are an SRE expert. Diagnose the issue, assess impact, and provide immediate action items." \
  --mcp-config monitoring-tools.json \
  --allowedTools "Bash,Read,WebSearch,mcp__datadog" \
  --output-format json
```

### Security Review with MCP Integration
```bash
gh pr diff 123 | claude -p \
  --append-system-prompt "Review for vulnerabilities and compliance issues." \
  --mcp-config security-tools.json \
  --allowedTools "Read,Grep,WebSearch,mcp__snyk" \
  --output-format json
```

### Data Analysis with MCP Tools
```bash
claude -p "Analyze sales data for Q4 trends" \
  --mcp-config analytics-tools.json \
  --allowedTools "mcp__postgres,mcp__tableau" \
  --output-format json \
  --max-turns 10
```

## Troubleshooting

### Issue: MCP Server Not Available
**Symptom**: Error message "mcp__servername tool is not currently available"

**Solutions**:
1. ✅ Add `--mcp-config <path>` flag (most common fix)
2. ✅ Verify JSON syntax: `cat ~/.claude/config/mcp.json | jq .`
3. ✅ Check file permissions: `ls -la ~/.claude/config/mcp.json`
4. ✅ Test server manually: Run the command/executable directly
5. ✅ Use `--debug` flag to see detailed error messages

### Issue: Tool Permission Denied
**Symptom**: Tool available but execution blocked

**Solutions**:
1. Add tool to `--allowedTools` list
2. Use correct tool naming: `mcp__servername__toolname`
3. Consider `--dangerously-skip-permissions` for automation (use carefully)

### Issue: Timeout During Execution
**Symptom**: Long-running MCP operations fail

**Solutions**:
1. Increase `--max-turns` limit (default may be too low)
2. Set environment variable: `export MCP_TOOL_TIMEOUT=120000` (2 minutes)
3. Check MCP server logs for issues

### Issue: JSON Parse Errors
**Symptom**: Cannot parse MCP configuration file

**Solutions**:
1. Validate JSON: `cat config.json | jq .`
2. Check for trailing commas (invalid in JSON)
3. Verify quotes are properly escaped
4. Use a JSON linter

## Environment Variables

### MCP_TIMEOUT
Server startup timeout in milliseconds
```bash
export MCP_TIMEOUT=30000
```

### MAX_MCP_OUTPUT_TOKENS
Maximum output token limit (default: 25,000)
```bash
export MAX_MCP_OUTPUT_TOKENS=50000
```

### MCP_TOOL_TIMEOUT
Individual tool call timeout
```bash
export MCP_TOOL_TIMEOUT=60000
```

## Configuration Scopes

MCP servers can be configured at different scopes:

### User Scope (Global)
- **Location**: `~/.claude/config/mcp.json`
- **Usage**: Servers available across all projects
- **Management**: `claude mcp add --scope user <name> <command>`

### Project Scope (Shared via Git)
- **Location**: `.mcp.json` in project root
- **Usage**: Team-shared MCP servers
- **Management**: `claude mcp add --scope project <name> <command>`
- **Note**: Requires user approval on first use

### Local Scope (Private)
- **Location**: Project-specific settings in `~/.claude/`
- **Usage**: Default for project-specific servers
- **Management**: `claude mcp add --scope local <name> <command>`

## Best Practices

### 1. Always Specify MCP Config in Headless Mode
❌ **Wrong** (will fail):
```bash
claude -p "Use mcp tool" --output-format json
```

✅ **Correct**:
```bash
claude -p "Use mcp tool" --mcp-config ~/.claude/config/mcp.json --output-format json
```

### 2. Use Specific Tool Allowlists
❌ **Too Permissive**:
```bash
--allowedTools "mcp__ai-counsel"  # Grants ALL tools
```

✅ **More Secure**:
```bash
--allowedTools "mcp__ai-counsel__deliberate"  # Only specific tool
```

### 3. Set Appropriate Max Turns
```bash
# For simple queries
--max-turns 2

# For complex multi-step workflows
--max-turns 10

# For autonomous agents
--max-turns 20
```

### 4. Use JSON Output for Automation
```bash
# Structured output for parsing
--output-format json

# Then parse in scripts
result=$(claude -p "..." --output-format json)
cost=$(echo "$result" | jq -r '.total_cost_usd')
```

### 5. Handle Timeouts Gracefully
```bash
# Set reasonable timeout for entire operation
--timeout 5m

# Set environment variables for MCP-specific timeouts
export MCP_TIMEOUT=30000
export MCP_TOOL_TIMEOUT=60000
```

## Integration with CI/CD

### GitHub Actions Example
```yaml
- name: Run MCP Analysis
  run: |
    claude -p "Analyze codebase for security issues" \
      --mcp-config .github/mcp-tools.json \
      --allowedTools "mcp__snyk,mcp__sonarqube" \
      --output-format json \
      --max-turns 5 > analysis.json

- name: Parse Results
  run: |
    jq '.result' analysis.json > summary.md
```

### GitLab CI Example
```yaml
mcp-security-scan:
  script:
    - export MCP_TIMEOUT=60000
    - claude -p "Security scan" \
        --mcp-config security-tools.json \
        --allowedTools "mcp__security-scanner" \
        --output-format json \
        --dangerously-skip-permissions > report.json
```

## Comparison: Interactive vs Headless Mode

| Feature | Interactive Mode | Headless Mode |
|---------|-----------------|---------------|
| MCP Auto-loading | ✅ Yes (from default configs) | ❌ No (requires `--mcp-config`) |
| Tool Management | `/mcp` command | `--allowedTools` flag |
| Output Format | Human-readable | JSON/Text via `--output-format` |
| Authentication | OAuth flows supported | Pre-configured credentials |
| Permission Prompts | Interactive approval | `--dangerously-skip-permissions` |
| Use Case | Development, exploration | Automation, CI/CD, scripts |

## Resources

- **Official Docs**: https://code.claude.com/docs/en/mcp
- **CLI Reference**: https://code.claude.com/docs/en/cli-reference.md
- **MCP Specification**: Model Context Protocol standard
- **Available Servers**: 40+ including GitHub, Sentry, Notion, Linear, Datadog

## Changelog

- **2025-11-10**: Initial documentation created from experimentation with AI Counsel MCP server
- Validated headless invocation with real multi-model deliberation
- Confirmed `--mcp-config` requirement for headless mode
- Documented successful 3-minute deliberation with Claude Sonnet & Gemini

## Notes for Conductor Integration

This documentation is particularly relevant for Conductor's potential integration with MCP servers. Key considerations:

1. **Task Execution**: Conductor could invoke MCP tools as part of task execution
2. **Agent Enhancement**: Agents could leverage MCP servers for specialized capabilities
3. **Quality Control**: MCP servers could provide additional validation/review
4. **Parallel Execution**: MCP invocations could run in parallel within waves
5. **Configuration**: Plan files could specify MCP servers per task

Example conductor integration:
```markdown
## Task 5: Security Analysis
**Agent**: security-expert
**MCP Tools**: mcp__snyk, mcp__sonarqube
**Estimated time**: 10 minutes

Analyze codebase for security vulnerabilities using MCP security tools.
```
