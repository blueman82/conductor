# Conductor Troubleshooting Guide

Common issues, solutions, and debugging techniques for Conductor.

## Table of Contents

- [Common Errors](#common-errors)
- [Validation Issues](#validation-issues)
- [Execution Problems](#execution-problems)
- [Agent Issues](#agent-issues)
- [Debug Mode](#debug-mode)
- [Getting Help](#getting-help)

## Common Errors

### "Circular dependency detected"

**Symptoms:**
```
Error: Circular dependency detected in task graph
Cycle: Task 1 → Task 2 → Task 3 → Task 1
```

**Cause:**
Tasks have circular dependencies where A depends on B, B depends on C, and C depends on A.

**Solution:**
1. Review dependency chain in error message
2. Identify which dependency to remove
3. Restructure tasks to break the cycle

**Example Fix:**

Before (circular):
```markdown
## Task 1: A
**Depends on**: Task 3

## Task 2: B
**Depends on**: Task 1

## Task 3: C
**Depends on**: Task 2
```

After (fixed):
```markdown
## Task 1: A
**Depends on**: None

## Task 2: B
**Depends on**: Task 1

## Task 3: C
**Depends on**: Task 2
```

### "Task depends on non-existent task"

**Symptoms:**
```
Error: Task 2 depends on non-existent Task 99
```

**Cause:**
Task references dependency that doesn't exist in plan.

**Solution:**
1. Check task numbers in plan
2. Verify dependency format: "Task N" (capital T, space, number)
3. Ensure referenced task is defined

### "File modified by multiple tasks"

**Symptoms:**
```
Warning: File main.go modified by Task 1 and Task 3
```

**Cause:**
Multiple tasks specify the same file in their `File(s)` metadata.

**Solutions:**

**Option 1: Add Dependencies**
Make tasks sequential by adding dependency:
```markdown
## Task 1: Initial Implementation
**File(s)**: main.go

## Task 3: Enhancement
**File(s)**: main.go
**Depends on**: Task 1
```

**Option 2: Split Files**
Use different files for different tasks:
```markdown
## Task 1: Core Logic
**File(s)**: core.go

## Task 3: Helper Functions
**File(s)**: helpers.go
```

### "Unknown agent"

**Symptoms:**
```
Error: Agent 'custom-agent' not found in ~/.claude/agents/
```

**Cause:**
Referenced agent doesn't exist in agent directory.

**Solution:**

1. Check agent directory:
```bash
ls -la ~/.claude/agents/
```

2. Verify agent file exists:
```bash
ls ~/.claude/agents/custom-agent.md
```

3. Use existing agent or create missing agent:
```markdown
## Task 1: Implementation
**Agent**: code-implementation  # Use existing agent
```

### "Context deadline exceeded" / "Timeout"

**Symptoms:**
```
Error: Task 1 execution timeout: context deadline exceeded
```

**Cause:**
Task exceeded timeout limit.

**Solution:**

**Option 1: Increase Timeout**
```bash
conductor run plan.md --timeout 1h

# Or via config
cat > .conductor/config.yaml << EOF
execution_timeout: 1h
task_timeout: 15m
EOF
```

**Option 2: Split Large Tasks**
Break complex tasks into smaller subtasks.

## Validation Issues

### "Invalid plan format"

**Symptoms:**
```
Error: Failed to parse plan file: invalid format
```

**Cause:**
Plan file doesn't match Markdown or YAML format specification.

**Solution:**

1. **Check file extension**:
   - Markdown: `.md` or `.markdown`
   - YAML: `.yaml` or `.yml`

2. **Validate Markdown format**:
```markdown
# Plan Title (required H1)

## Task 1: Task Name (required H2 with "Task N:" prefix)
**File(s)**: files.go (metadata with bold keys)

Task description.
```

3. **Validate YAML format**:
```yaml
plan:                    # Required root key
  name: Plan Title       # Required
  tasks:                 # Required
    - id: 1              # Required
      name: Task Name    # Required
```

4. **Use validate command**:
```bash
conductor validate plan.md
```

### "No tasks found"

**Symptoms:**
```
Error: No tasks found in plan file
```

**Cause:**
Plan file has no properly formatted tasks.

**Solution:**

1. **Markdown**: Ensure H2 headings with "Task N:" prefix:
```markdown
## Task 1: First Task
```
NOT:
```markdown
## First Task (missing "Task 1:" prefix)
```

2. **YAML**: Ensure tasks array not empty:
```yaml
plan:
  name: Plan
  tasks:
    - id: 1
      name: First Task
```

## Execution Problems

### "Quality control failed" / "RED verdict"

**Symptoms:**
```
Task 1 received RED verdict from quality control
Retry 1/2...
Task 1 received RED verdict from quality control
Retry 2/2...
Error: Task 1 failed after 2 retries
```

**Cause:**
Task output didn't meet quality standards.

**Solution:**

1. **Check task logs**:
```bash
cat .conductor/logs/task-1-*.log
```

2. **Review QC feedback**:
```
Quality Control: RED

Feedback: Implementation incomplete. Missing error handling
for database connection failures.
```

3. **Fix underlying issue**:
   - Clarify task requirements
   - Improve task description
   - Add code examples
   - Simplify complex tasks

4. **Adjust retry limit**:
```yaml
# .conductor/config.yaml
max_retries: 3
```

### "Claude CLI not found"

**Symptoms:**
```
Error: Failed to execute claude command: executable file not found in $PATH
```

**Cause:**
Claude Code CLI not installed or not in PATH.

**Solution:**

1. **Verify installation**:
```bash
which claude
claude --version
```

2. **Install Claude Code CLI**:
Follow instructions at https://claude.ai/code

3. **Add to PATH**:
```bash
export PATH="$PATH:/path/to/claude"
echo 'export PATH="$PATH:/path/to/claude"' >> ~/.bashrc
```

4. **Verify authentication**:
```bash
claude auth status
```

### "Permission denied"

**Symptoms:**
```
Error: Failed to write to plan file: permission denied
Error: Failed to create log directory: permission denied
```

**Cause:**
Insufficient permissions for file operations.

**Solution:**

1. **Check file permissions**:
```bash
ls -la plan.md
ls -la .conductor/
```

2. **Fix permissions**:
```bash
chmod 644 plan.md
chmod 755 .conductor/
chmod 755 .conductor/logs/
```

3. **Use custom log directory**:
```bash
conductor run plan.md --log-dir ~/conductor-logs
```

### "File lock timeout"

**Symptoms:**
```
Error: Failed to acquire file lock after 30s
```

**Cause:**
Another process holds lock on plan file.

**Solution:**

1. **Check for running instances**:
```bash
ps aux | grep conductor
```

2. **Remove stale lock**:
```bash
rm .conductor/.lock
```

3. **Increase lock timeout**:
```yaml
# .conductor/config.yaml
lock_timeout: 1m
```

4. **Wait and retry**:
```bash
conductor run plan.md
```

## Agent Issues

### "Agent file not readable"

**Symptoms:**
```
Warning: Agent file ~/.claude/agents/custom-agent.md not readable
```

**Cause:**
Agent file has incorrect permissions.

**Solution:**

```bash
chmod 644 ~/.claude/agents/custom-agent.md
```

### "No agents discovered"

**Symptoms:**
```
Warning: No agents found in ~/.claude/agents/
```

**Cause:**
No agent files in expected locations.

**Solution:**

1. **Check agent directory**:
```bash
ls -la ~/.claude/agents/
```

2. **Create agents** (if missing):
```bash
mkdir -p ~/.claude/agents/
cp /path/to/agent.md ~/.claude/agents/
```

## Debug Mode

### Enabling Verbose Logging

```bash
# Console output
conductor run plan.md --verbose

# File logging (always enabled)
cat .conductor/logs/conductor-*.log
```

### Examining Task Output

```bash
# View specific task log
cat .conductor/logs/task-1-setup-*.log

# View all task logs
ls -l .conductor/logs/task-*.log

# Search for errors
grep -i error .conductor/logs/*.log

# Search for RED verdicts
grep "RED" .conductor/logs/*.log
```

### Testing Individual Tasks

Run Claude CLI directly to test task execution:

```bash
# Test task execution
claude -p "Your task prompt here" \
  --settings '{"disableAllHooks": true}' \
  --output-format json

# With specific agent
claude -p "use the code-implementation subagent to: implement feature" \
  --settings '{"disableAllHooks": true}' \
  --output-format json
```

### Dry Run Testing

Test execution plan without running tasks:

```bash
# See wave structure
conductor run plan.md --dry-run

# Check concurrency limits
conductor run plan.md --dry-run --max-concurrency 5

# Check dependency resolution
conductor validate plan.md
```

## Performance Tuning

### Slow Execution

**Symptoms:**
- Execution takes much longer than expected
- Tasks running sequentially when they should be parallel

**Solutions:**

1. **Check wave grouping**:
```bash
conductor validate plan.md
```

2. **Increase concurrency**:
```bash
conductor run plan.md --max-concurrency 5
```

3. **Reduce dependencies**:
Remove unnecessary dependencies to enable more parallelism.

4. **Optimize task sizes**:
- Split large tasks (>15 min) into smaller tasks
- Combine tiny tasks (<2 min) into larger tasks

### High Resource Usage

**Solutions:**

1. **Reduce concurrency**:
```bash
conductor run plan.md --max-concurrency 2
```

2. **Run fewer intensive tasks in parallel**:
Restructure plan to avoid parallel resource-intensive tasks.

## Getting Help

### Documentation

- [README](../README.md) - Project overview
- [Usage Guide](usage.md) - CLI reference
- [Plan Format Guide](plan-format.md) - Format specifications

### GitHub

- **Issues**: https://github.com/harrison/conductor/issues
- **Discussions**: https://github.com/harrison/conductor/discussions
- **Source Code**: https://github.com/harrison/conductor

### Reporting Bugs

When reporting issues, include:

1. **Conductor version**:
```bash
conductor --version
```

2. **Error message** (full output):
```bash
conductor run plan.md 2>&1 | tee error.log
```

3. **Plan file** (sanitized if needed)

4. **Environment**:
   - OS: macOS, Linux, etc.
   - Go version: `go version`
   - Claude CLI version: `claude --version`

### Common Questions

**Q: Can I run multiple plans simultaneously?**

A: Not recommended. File locking prevents concurrent updates to same plan.

**Q: Can I pause/resume execution?**

A: Not currently supported. Execution is atomic.

**Q: How do I skip completed tasks?**

A: Conductor automatically skips tasks marked as completed in plan file.

**Q: Can I run specific tasks only?**

A: Not directly. Create new plan with only desired tasks.

**Q: How do I handle secrets in plans?**

A: Do not include secrets in plan files. Use environment variables or configuration files.

## See Also

- [Usage Guide](usage.md) - CLI commands and execution
- [Plan Format Guide](plan-format.md) - Plan format specifications
- [README](../README.md) - Project overview
