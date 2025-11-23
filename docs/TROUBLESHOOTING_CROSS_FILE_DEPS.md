# Troubleshooting Guide: Cross-File Dependencies

Common issues when using Conductor's cross-file dependency features and their solutions.

## Table of Contents

- [File Path Issues](#file-path-issues)
- [Task Reference Errors](#task-reference-errors)
- [Circular Dependencies](#circular-dependencies)
- [Validation Failures](#validation-failures)
- [Execution Problems](#execution-problems)
- [Resume Issues](#resume-issues)
- [Performance Issues](#performance-issues)
- [Format Errors](#format-errors)
- [Getting Help](#getting-help)

---

## File Path Issues

### Error: "File not found: services.yaml"

#### Symptoms
```
✗ Validation failed
  Error: File not found: services.yaml
```

#### Causes
1. File doesn't exist in current directory
2. Typo in filename
3. File is in a subdirectory
4. Relative path is wrong
5. File was moved or deleted

#### Solutions

**Step 1: Verify file exists**
```bash
# List files
ls -la services.yaml

# Or with glob pattern
ls -la *.yaml
```

**Step 2: Check file path**
```bash
# Current directory
ls -la services.yaml  # exists?

# Subdirectory
ls -la services/auth.yaml  # exists?

# Absolute path
ls -la /absolute/path/services.yaml  # exists?
```

**Step 3: Verify file is readable**
```bash
# Check permissions
test -r services.yaml && echo "Readable" || echo "Not readable"

# Fix permissions if needed
chmod 644 services.yaml
```

**Step 4: Update cross-file reference**
```yaml
# If file is in subdirectory:
depends_on:
  - file: services/auth.yaml  # Add subdirectory
    task: 1

# If using absolute path:
depends_on:
  - file: /absolute/path/foundation.yaml
    task: 1
```

**Step 5: Re-validate**
```bash
conductor validate services.yaml foundation.yaml
```

---

### Error: "File path too long" or "Invalid file path"

#### Symptoms
```
✗ Validation failed
  Error: Invalid file path: path/to/../../../bad.yaml
```

#### Causes
1. Path with too many `..` references
2. Non-existent intermediate directories
3. Special characters in filename
4. Path traversal attempt

#### Solutions

**Use relative paths:**
```yaml
# Good
depends_on:
  - file: foundation.yaml
    task: 1

# Good (subdirectory)
depends_on:
  - file: services/auth.yaml
    task: 1

# Avoid (confusing)
depends_on:
  - file: ../other/foundation.yaml
    task: 1
```

**Organize files clearly:**
```
conductor/
├── foundation.yaml
├── services/
│   ├── auth.yaml
│   ├── api.yaml
│   └── data.yaml
└── integration/
    └── integration.yaml
```

---

## Task Reference Errors

### Error: "Task 5 not found in foundation.yaml"

#### Symptoms
```
✗ Validation failed
  Cross-file reference error:
  - services.yaml references foundation.yaml#5 but only tasks 1-4 exist
```

#### Causes
1. Task number doesn't exist in referenced file
2. Task was deleted from referenced file
3. Task number is wrong
4. Off-by-one error

#### Solutions

**Step 1: Check tasks in referenced file**
```bash
# View all tasks in foundation.yaml
conductor validate foundation.yaml --verbose

# Or manually inspect
grep -E "id:|name:" foundation.yaml
```

**Step 2: Find correct task number**
```yaml
# foundation.yaml has:
# - id: 1 (Database Setup)
# - id: 2 (Cache Setup)
# - id: 3 (Logging)
# - id: 4 (Monitoring)

# Correct reference in services.yaml:
depends_on:
  - file: foundation.yaml
    task: 4  # Not 5 (doesn't exist)
```

**Step 3: Update all references**
```bash
# Search for incorrect references
grep -n "task: 5" services.yaml

# Update to correct task number
sed -i 's/task: 5/task: 4/g' services.yaml

# Verify
grep -n "task:" services.yaml
```

**Step 4: Re-validate**
```bash
conductor validate foundation.yaml services.yaml
```

---

### Error: "Ambiguous task reference"

#### Symptoms
```
✗ Validation failed
  Multiple tasks with number 1 in foundation.yaml
  (This shouldn't happen - task numbers should be unique per file)
```

#### Causes
1. Duplicate task IDs in same file (parsing error)
2. Task number appears twice in file
3. Malformed YAML causing incorrect parsing

#### Solutions

**Check for duplicate task numbers:**
```bash
# Find duplicates
grep -E "^\s+id:" foundation.yaml | sort | uniq -d
```

**Example (Incorrect):**
```yaml
tasks:
  - id: 1
    name: Setup Database

  - id: 1    # DUPLICATE!
    name: Setup Cache
```

**Fix:**
```yaml
tasks:
  - id: 1
    name: Setup Database

  - id: 2    # Fixed: unique number
    name: Setup Cache
```

**Re-validate:**
```bash
conductor validate foundation.yaml --verbose
```

---

## Circular Dependencies

### Error: "Circular dependency detected"

#### Symptoms
```
✗ Validation failed
  Circular dependency detected:
  Task 1 → Task 2 → Task 3 → Task 1
```

#### Causes
1. Two tasks depend on each other
2. Chain of dependencies that loops back
3. Self-reference (task depends on itself)

#### Solutions

**Step 1: Understand the cycle**
```
Example cycle:
- foundation.yaml#1 depends on services.yaml#2
- services.yaml#2 depends on integration.yaml#3
- integration.yaml#3 depends on foundation.yaml#1
```

**Step 2: Draw the dependency graph**
```
foundation#1 → services#2 → integration#3 → foundation#1  [CYCLE!]
```

**Step 3: Break the cycle**

Identify which dependency is unnecessary:
```yaml
# Option 1: Remove cycle-creating dependency
# foundation.yaml#1
depends_on: []  # Remove: was depending on integration#3

# services.yaml#2
depends_on:
  - file: foundation.yaml
    task: 1

# integration.yaml#3
depends_on:
  - file: services.yaml
    task: 2
```

**Step 4: Restructure if needed**

Sometimes you need to restructure tasks:
```yaml
# Before (circular):
# A → B → C → A

# After (linear):
# A → B → C (C doesn't depend on A)
```

**Step 5: Re-validate**
```bash
conductor validate *.yaml
# Should show no cycles
```

---

### Self-Reference Error

#### Symptoms
```
✗ Validation failed
  Task 1 depends on itself (self-reference)
```

#### Causes
1. Task references itself
2. Copy-paste error

#### Solutions

**Check for self-references:**
```bash
# Find tasks that might reference themselves
grep -n "task: 1" services.yaml  # Look for duplicate references
```

**Example (Incorrect):**
```yaml
# services.yaml
tasks:
  - id: 1
    name: Setup Auth
    depends_on:
      - file: services.yaml  # Same file
        task: 1              # Same task - SELF-REFERENCE!
```

**Fix:**
```yaml
# Change to local reference and use correct task number
tasks:
  - id: 1
    name: Setup Auth
    depends_on:
      - 2  # Reference local task 2 instead

  - id: 2
    name: Integrate Auth
    depends_on:
      - 1
```

---

## Validation Failures

### Error: "Invalid dependency format"

#### Symptoms
```
✗ Validation failed
  Invalid dependency format in services.yaml
  Expected: depends_on as list or object
  Got: string
```

#### Causes
1. Wrong YAML syntax
2. String instead of list
3. Missing required fields (file, task)
4. Incorrect indentation

#### Solutions

**Check YAML syntax:**
```bash
# Validate YAML syntax
yamllint services.yaml

# Or try parsing with Python
python3 -c "import yaml; yaml.safe_load(open('services.yaml'))"
```

**Correct formats:**

Local dependencies (list of numbers):
```yaml
depends_on: [1, 2, 3]
```

Or:
```yaml
depends_on:
  - 1
  - 2
  - 3
```

Cross-file dependencies (list of objects):
```yaml
depends_on:
  - file: foundation.yaml
    task: 1
  - file: services/auth.yaml
    task: 2
```

Mixed:
```yaml
depends_on:
  - 1                      # Local
  - file: foundation.yaml
    task: 2                # Cross-file
```

**Common mistakes:**

❌ Wrong: String instead of list
```yaml
depends_on: "1"
```

✓ Correct:
```yaml
depends_on: [1]
```

❌ Wrong: Missing task field
```yaml
depends_on:
  - file: foundation.yaml
```

✓ Correct:
```yaml
depends_on:
  - file: foundation.yaml
    task: 1
```

❌ Wrong: Wrong indentation
```yaml
tasks:
  - id: 1
  depends_on:     # Wrong level!
    - file: foundation.yaml
      task: 1
```

✓ Correct:
```yaml
tasks:
  - id: 1
    depends_on:   # Correct level (inside task)
      - file: foundation.yaml
        task: 1
```

**Re-validate after fixing:**
```bash
conductor validate services.yaml foundation.yaml --verbose
```

---

## Execution Problems

### Error: "Wave execution failed at task X"

#### Symptoms
```
✗ Task execution failed
  Wave 2, Task 3: Exception during execution
  Error: previous task (foundation.yaml#1) hasn't completed
```

#### Causes
1. Dependency didn't actually complete (marked as complete but failed)
2. Cross-file dependency validation passed but execution shows issue
3. Task status tracking out of sync

#### Solutions

**Step 1: Check dependency completion**
```bash
# Find status of dependency task
grep -A 5 "id: 1" foundation.yaml | grep Status

# Should show: Status: completed
```

**Step 2: Dry-run to verify dependencies**
```bash
# Test without executing
conductor run foundation.yaml services.yaml --dry-run --verbose

# Check wave assignments
# Wave 1: foundation.yaml#1
# Wave 2: foundation.yaml#2
# Wave 3: services.yaml#1 (should wait for foundation#1)
```

**Step 3: Manual task execution**
```bash
# Run just the first wave
conductor run foundation.yaml services.yaml --max-concurrency 1 --verbose

# After first task completes, run rest
conductor run foundation.yaml services.yaml --skip-completed
```

**Step 4: Check plan file syntax**
```bash
# Make sure status is valid
grep "Status:" foundation.yaml
# Should show: Status: completed (or pending)
```

---

### Error: "Dependency not found during execution"

#### Symptoms
```
✗ Execution error
  Task services.yaml#2 requires foundation.yaml#1 but it was not executed
  (This shouldn't happen - validation passed)
```

#### Causes
1. Validation passed but execution found issue
2. File was modified after validation
3. Resume with `--skip-completed` missed dependencies
4. Incomplete task status tracking

#### Solutions

**Step 1: Re-validate**
```bash
# Fresh validation
conductor validate foundation.yaml services.yaml --verbose
```

**Step 2: Check if files were modified**
```bash
# See file modification times
ls -la *.yaml

# Check git status for changes
git status
```

**Step 3: Restart execution from beginning**
```bash
# Clear all completion status and restart
conductor run foundation.yaml services.yaml --no-skip-completed --verbose
```

**Step 4: Check dependency resolution**
```bash
# Verbose output shows dependency resolution
conductor run foundation.yaml services.yaml --verbose --dry-run
```

---

## Resume Issues

### Error: "Resume fails with unknown task state"

#### Symptoms
```
✗ Execution error
  Resuming execution: foundation.yaml#1 status is unknown
  Expected: completed, failed, or pending
```

#### Causes
1. Corrupted status in plan file
2. Status field has invalid value
3. Resume with different file set
4. Plan file manually edited with incorrect status

#### Solutions

**Step 1: Check task status values**
```bash
# Valid statuses
grep "Status:" foundation.yaml
# Should be: pending, in-progress, completed, failed, or skipped
```

**Example (Incorrect status):**
```yaml
- id: 1
  name: Setup Database
  Status: done  # WRONG! Should be "completed"
```

**Fix:**
```yaml
- id: 1
  name: Setup Database
  Status: completed  # Correct
```

**Step 2: Use correct resume command**
```bash
# Always use same file set for resume
conductor run foundation.yaml services.yaml --skip-completed

# Not recommended - different files:
conductor run foundation.yaml --skip-completed
conductor run services.yaml --skip-completed  # Separate runs
```

**Step 3: Full restart if uncertain**
```bash
# Reset and restart
conductor run foundation.yaml services.yaml --no-skip-completed --verbose
```

---

### Error: "Cross-file resume skips wrong tasks"

#### Symptoms
```
✗ Resume error
  Skipping foundation.yaml#1 (completed)
  But services.yaml#1 depends on it!
  (Should have skipped foundation.yaml#1 AND services.yaml#1)
```

#### Causes
1. Resume with incomplete file set
2. Interrupted resume (partial completion)
3. Manual editing of plan files
4. Tasks from different files with same numbering

#### Solutions

**Step 1: Use same file set**
```bash
# Remember which files were used
PLAN_FILES="foundation.yaml services.yaml integration.yaml"

# Initial execution
conductor run $PLAN_FILES

# Resume with EXACT same files
conductor run $PLAN_FILES --skip-completed
```

**Step 2: Check completion status across files**
```bash
# Check each file
echo "=== foundation.yaml ==="
grep "Status:" foundation.yaml

echo "=== services.yaml ==="
grep "Status:" services.yaml

echo "=== integration.yaml ==="
grep "Status:" integration.yaml
```

**Step 3: Full restart if resume is problematic**
```bash
# Option 1: Clear all completion status
conductor run foundation.yaml services.yaml integration.yaml --no-skip-completed

# Option 2: Manually reset status in files
sed -i 's/Status: completed/Status: pending/g' *.yaml
conductor run foundation.yaml services.yaml integration.yaml
```

---

## Performance Issues

### Error: "Excessive validation time for many cross-file references"

#### Symptoms
```
Validating plans... (takes >30 seconds)
Checking cross-file references...
```

#### Causes
1. Many files with many cross-file dependencies
2. Large files being parsed repeatedly
3. Reference validation is O(n*m) for n files with m cross-refs each
4. Network filesystem being used

#### Solutions

**Optimize file organization:**
```bash
# Instead of many scattered files:
services/
├── auth.yaml          # 20 tasks
├── api.yaml           # 20 tasks
├── data.yaml          # 20 tasks
└── cache.yaml         # 20 tasks

# Consolidate by component:
services/
├── foundation.yaml    # Core infrastructure (5 tasks)
├── core.yaml          # Main services (30 tasks)
└── integration.yaml   # Integration (15 tasks)
```

**Reduce cross-file dependencies:**
```bash
# Before: Many cross-file refs
services/auth.yaml    → foundation.yaml#1
services/auth.yaml    → cache.yaml#1
services/auth.yaml    → database.yaml#1
services/api.yaml     → foundation.yaml#1
services/api.yaml     → auth.yaml#1
...

# After: Fewer cross-file refs
foundation.yaml       → (no external deps)
core.yaml             → foundation.yaml#1
integration.yaml      → core.yaml#1, core.yaml#2
```

**Use local concurrency:**
```bash
# Increase concurrency to offset validation overhead
conductor run *.yaml --max-concurrency 10
```

---

## Format Errors

### Error: "Markdown plan with cross-file dependencies not supported"

#### Symptoms
```
✗ Error: Cross-file dependencies in Markdown format not supported
  Use YAML format for cross-file dependencies
```

#### Causes
1. Using Markdown plan with cross-file references
2. Markdown parser doesn't understand cross-file syntax

#### Solutions

**Convert to YAML format:**

Before (Markdown):
```markdown
## Task 2: Auth Service
**Depends on**: foundation.yaml#1

Implementation...
```

After (YAML):
```yaml
tasks:
  - id: 2
    name: Auth Service
    depends_on:
      - file: foundation.yaml
        task: 1
```

**Create YAML version:**
```bash
# Run converter or manually create YAML
conductor validate foundation.md services.md --verbose

# Then create services.yaml with cross-file deps
```

---

### Error: "Invalid YAML: missing file or task in cross-file reference"

#### Symptoms
```
✗ Validation failed
  Invalid cross-file reference: missing "file" or "task" field
  Location: services.yaml, task 2
```

#### Causes
1. Incomplete dependency specification
2. Wrong field names
3. Missing required fields

#### Solutions

**Check field names:**
```yaml
# Wrong field names:
depends_on:
  - filename: foundation.yaml  # Should be "file"
    task_id: 1                 # Should be "task"

# Correct:
depends_on:
  - file: foundation.yaml
    task: 1
```

**Ensure both fields present:**
```yaml
# Wrong: only file
depends_on:
  - file: foundation.yaml

# Wrong: only task
depends_on:
  - task: 1

# Correct: both
depends_on:
  - file: foundation.yaml
    task: 1
```

---

## Getting Help

### Verbose Output for Debugging

```bash
# Get detailed validation output
conductor validate *.yaml --verbose

# Get detailed execution output
conductor run *.yaml --verbose --dry-run

# Check specific cross-file references
grep -r "file:" *.yaml
```

### Collect Debug Information

When reporting issues:

```bash
# Show version
conductor --version

# Show plan structure
conductor validate *.yaml --verbose > validation.log

# Show execution plan
conductor run *.yaml --verbose --dry-run > execution_plan.log

# Attach both files when reporting
```

### Common Debug Commands

```bash
# List all tasks with their dependencies
grep -E "id:|depends_on:" *.yaml

# Find all cross-file references
grep -E "file:" *.yaml

# Check for circular dependencies
conductor validate *.yaml | grep -i circular

# Verify all referenced files exist
grep "file:" *.yaml | sed 's/.*file: //' | sed 's/ .*//' | sort -u | while read f; do test -f "$f" || echo "MISSING: $f"; done

# Verify all referenced tasks exist
grep -A 1 "file:" *.yaml | grep "task:" | while read task; do echo $task; done
```

### Check System State

```bash
# Current working directory
pwd

# Available files
ls -la *.yaml

# Git status (if using git)
git status

# File permissions
ls -l *.yaml

# Disk space
df -h
```

---

## Still Having Issues?

1. **Enable verbose output**: `--verbose` flag shows more details
2. **Validate first**: `conductor validate` before `conductor run`
3. **Test with dry-run**: `--dry-run` simulates without executing
4. **Check documentation**: [Cross-File Dependencies Reference](./CROSS_FILE_DEPENDENCIES.md)
5. **Review examples**: [README Multi-File Plans](../README.md#multi-file-plans)
6. **Check git history**: See what changed in plan files
