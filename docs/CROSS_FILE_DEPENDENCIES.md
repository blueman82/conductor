# Cross-File Dependencies Reference

Conductor v2.5+ supports explicit cross-file dependency notation, enabling complex multi-file orchestration with dependencies that span across multiple plan files.

## Table of Contents

- [Overview](#overview)
- [Syntax Guide](#syntax-guide)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Validation and Errors](#validation-and-errors)
- [Backward Compatibility](#backward-compatibility)
- [Migration Guide](#migration-guide)
- [Troubleshooting](#troubleshooting)

---

## Overview

Cross-file dependencies allow tasks in one plan file to explicitly depend on tasks in other plan files. This enables:

- **Modular Planning**: Split large implementations into logical components
- **Clear Dependencies**: Explicitly declare which files depend on which
- **Automatic Validation**: Conductor validates all cross-file links
- **Resumable Execution**: Continue interrupted multi-file executions seamlessly

### When to Use Cross-File Dependencies

Use cross-file dependencies when:
- You have logically separate plan files that need strict ordering
- Task execution order matters between files
- You want explicit, queryable dependency relationships
- You're building complex microservices or multi-component systems

### When to Use Implicit Ordering

Omit cross-file dependencies when:
- Files are completely independent
- Logical grouping alone is sufficient
- You prefer simpler, flatter dependency graphs

---

## Syntax Guide

### Local Dependencies (Single File)

Reference tasks in the same file by task number only:

```yaml
tasks:
  - id: 1
    name: Setup Database
    depends_on: []

  - id: 2
    name: Create Tables
    depends_on: [1]  # Reference local task by number
```

```markdown
## Task 2: Create Tables
**Depends on**: Task 1
```

### Cross-File Dependencies (v2.5+)

Reference tasks in other files using explicit file notation:

#### YAML Format

```yaml
tasks:
  - id: 3
    name: Auth Service
    depends_on:
      - 1                    # Local task (same file)
      - file: foundation.yaml
        task: 2              # Task 2 from foundation.yaml
      - file: setup/db.yaml
        task: 5              # Task 5 from setup/db.yaml
```

#### Markdown Format (Pseudo-Syntax)

Currently cross-file dependencies are best expressed in YAML format. For Markdown plans, use a comment section:

```markdown
## Task 3: Auth Service
**Depends on**: Task 1 (this file), foundation.yaml#2, setup/db.yaml#5

Implementation details...
```

### Reference Styles

**Long Form** (Explicit file notation):
```yaml
depends_on:
  - file: authentication.yaml
    task: 4
```

**Shorthand** (For local references only):
```yaml
depends_on: [1, 2, 3]
```

**Mixed** (Both local and cross-file):
```yaml
depends_on:
  - 1                    # Local: Task 1
  - 2                    # Local: Task 2
  - file: auth.yaml
    task: 3              # Cross-file: Task 3 from auth.yaml
```

---

## Examples

### Example 1: Three-File Backend Setup

**File 1: foundation.yaml** (Core infrastructure)
```yaml
plan:
  name: Foundation Tasks
  tasks:
    - id: 1
      name: Initialize Database
      files: [infrastructure/db.tf]
      depends_on: []

    - id: 2
      name: Setup Redis Cache
      files: [infrastructure/redis.tf]
      depends_on: [1]  # Depends on Task 1 in same file
```

**File 2: services.yaml** (Service implementations)
```yaml
plan:
  name: Service Implementation
  tasks:
    - id: 3
      name: Create Auth Service
      files: [internal/auth/auth.go]
      depends_on:
        - file: foundation.yaml
          task: 1      # Must wait for database setup

    - id: 4
      name: Create API Service
      files: [internal/api/api.go]
      depends_on:
        - file: foundation.yaml
          task: 2      # Needs Redis
        - 3            # Needs Auth Service (same file)
```

**File 3: integration.yaml** (Integration tasks)
```yaml
plan:
  name: Integration Tasks
  tasks:
    - id: 5
      name: Connect Services
      files: [internal/connector/connector.go]
      depends_on:
        - file: services.yaml
          task: 3      # Needs Auth Service
        - file: services.yaml
          task: 4      # Needs API Service
```

**Execution Order:**
```
Wave 1: foundation.yaml#1
Wave 2: foundation.yaml#2
Wave 3: services.yaml#3
Wave 4: services.yaml#4
Wave 5: integration.yaml#5
```

**Execute Together:**
```bash
conductor run foundation.yaml services.yaml integration.yaml
conductor validate foundation.yaml services.yaml integration.yaml
```

### Example 2: Microservices with Shared Dependencies

**File 1: shared.yaml** (Shared modules)
```yaml
plan:
  name: Shared Infrastructure
  tasks:
    - id: 1
      name: Database Layer
      files: [shared/db/db.go]
      depends_on: []

    - id: 2
      name: Cache Layer
      files: [shared/cache/cache.go]
      depends_on: [1]

    - id: 3
      name: Logger Service
      files: [shared/logging/logger.go]
      depends_on: []
```

**File 2: auth-service.yaml**
```yaml
plan:
  name: Authentication Service
  tasks:
    - id: 4
      name: Auth Module
      files: [services/auth/auth.go]
      depends_on:
        - file: shared.yaml
          task: 1    # Uses database
        - file: shared.yaml
          task: 3    # Uses logging

    - id: 5
      name: Auth Tests
      files: [services/auth/auth_test.go]
      depends_on: [4]
```

**File 3: api-service.yaml**
```yaml
plan:
  name: API Service
  tasks:
    - id: 6
      name: API Router
      files: [services/api/router.go]
      depends_on:
        - file: shared.yaml
          task: 2    # Uses cache
        - file: shared.yaml
          task: 3    # Uses logging

    - id: 7
      name: Integrate Auth
      files: [services/api/middleware.go]
      depends_on:
        - 6         # Needs router
        - file: auth-service.yaml
          task: 4   # Needs auth module
```

**Execute All at Once:**
```bash
conductor run shared.yaml auth-service.yaml api-service.yaml --max-concurrency 4
```

### Example 3: Frontend + Backend Integration

**File 1: backend.yaml**
```yaml
plan:
  name: Backend Implementation
  tasks:
    - id: 1
      name: Database Schema
      depends_on: []

    - id: 2
      name: API Endpoints
      depends_on: [1]
```

**File 2: frontend.yaml**
```yaml
plan:
  name: Frontend Implementation
  tasks:
    - id: 3
      name: API Client Library
      depends_on:
        - file: backend.yaml
          task: 2    # Must have API endpoints available

    - id: 4
      name: UI Components
      depends_on: [3]
```

**File 3: integration.yaml**
```yaml
plan:
  name: End-to-End Integration
  tasks:
    - id: 5
      name: E2E Tests
      depends_on:
        - file: backend.yaml
          task: 2
        - file: frontend.yaml
          task: 4
```

---

## Best Practices

### 1. Keep Files Logically Grouped

**Good:**
```
foundation.yaml        # Core infrastructure
services/auth.yaml     # Auth service only
services/api.yaml      # API service only
```

**Avoid:**
```
mixed.yaml            # Database setup, auth, API, and tests all together
```

### 2. Minimize Cross-File Dependencies

**Good:** 2-3 dependencies between files
```yaml
depends_on:
  - file: foundation.yaml
    task: 1
```

**Avoid:** Many cross-file dependencies creating tightly coupled files
```yaml
depends_on:
  - file: file1.yaml
    task: 1
  - file: file2.yaml
    task: 2
  - file: file3.yaml
    task: 3
  - file: file4.yaml
    task: 4
```

### 3. Use Clear File Names

**Good:**
- `foundation.yaml` - Core infrastructure
- `services-auth.yaml` - Auth service
- `integration.yaml` - Integration points
- `deployment.yaml` - Deployment tasks

**Avoid:**
- `plan1.yaml`, `plan2.yaml` (unclear purpose)
- `tasks.yaml` (too generic)

### 4. Document Integration Points

Add comments explaining why cross-file dependencies exist:

```yaml
tasks:
  - id: 5
    name: Create API Handler
    files: [internal/api/handler.go]
    # MUST WAIT FOR:
    # - foundation.yaml#1: Database initialization
    # - services.yaml#2: Auth module (for middleware)
    depends_on:
      - file: foundation.yaml
        task: 1
      - file: services.yaml
        task: 2
```

### 5. Validate Before Execution

Always validate before running multi-file plans:

```bash
conductor validate foundation.yaml services.yaml integration.yaml
```

Conductor will check:
- All cross-file references exist
- No circular dependencies
- Task numbers are valid
- File paths are readable

### 6. Document File Execution Order

When files have strict ordering, document it:

```bash
# Strict execution order required:
# 1. foundation.yaml  - Core infrastructure
# 2. services.yaml    - Service implementations
# 3. integration.yaml - Integration tasks

conductor run foundation.yaml services.yaml integration.yaml
```

---

## Validation and Errors

### Conductor Validation Process

When loading multi-file plans, Conductor:

1. **Loads each file** independently
2. **Merges plans** into unified task graph
3. **Validates all references**:
   - Check file existence (readable)
   - Check task numbers exist in referenced files
   - Check for circular dependencies (intra-file and cross-file)
4. **Builds dependency graph** across all files
5. **Calculates execution waves** respecting all dependencies

### Common Errors and Solutions

#### Error: "File not found: services.yaml"

**Cause:** Referenced file doesn't exist

**Solution:**
```bash
# Check file exists
ls -la services.yaml

# Verify file path is relative to current directory
conductor validate services.yaml

# Use absolute paths if needed
conductor run /absolute/path/foundation.yaml /absolute/path/services.yaml
```

#### Error: "Task 4 not found in foundation.yaml"

**Cause:** Referenced task doesn't exist in the file

**Cause Examples:**
- Task ID is wrong (`task: 4` but only tasks 1-3 exist)
- Task was deleted but dependency wasn't updated
- Typo in file name (e.g., `foundaton.yaml` instead of `foundation.yaml`)

**Solution:**
```yaml
# Verify task exists in foundation.yaml
# Then update reference:
- file: foundation.yaml
  task: 3  # Corrected task number
```

#### Error: "Circular dependency detected: 1 → foundation.yaml#2 → services.yaml#3 → 1"

**Cause:** Tasks form a circular dependency chain

**Example (Incorrect):**
```yaml
# foundation.yaml
tasks:
  - id: 1
    depends_on:
      - file: services.yaml
        task: 2

# services.yaml
tasks:
  - id: 2
    depends_on:
      - file: foundation.yaml
        task: 1  # CIRCULAR!
```

**Solution:**
```yaml
# Restructure to break cycle
# foundation.yaml
tasks:
  - id: 1
    depends_on: []  # No dependency

# services.yaml
tasks:
  - id: 2
    depends_on:
      - file: foundation.yaml
        task: 1  # One-way dependency
```

#### Error: "Invalid dependency format: depends_on should be list of numbers or file references"

**Cause:** Malformed dependency syntax

**Incorrect:**
```yaml
depends_on: "1"                    # String instead of list
depends_on:                        # Missing task field
  - file: foundation.yaml
```

**Correct:**
```yaml
depends_on:
  - 1                              # List of integers
  - file: foundation.yaml
    task: 2                        # Both file and task required
```

---

## Backward Compatibility

### Existing Plans Work Unchanged

All existing plans (v2.0 - v2.4) continue to work without modification:

**Single-File Plans (unchanged):**
```yaml
depends_on: [1, 2, 3]  # Still works
```

**Multi-File Plans (implicit ordering, unchanged):**
```bash
conductor run file1.yaml file2.yaml file3.yaml  # Still works
```

### Mixing Old and New Syntax

You can mix local and cross-file dependencies:

```yaml
tasks:
  - id: 5
    depends_on:
      - 1                          # Local (old syntax)
      - 2                          # Local (old syntax)
      - file: services.yaml        # Cross-file (new syntax)
        task: 3
```

### Migration Path

**No migration required** for existing plans. You can:
1. Keep existing multi-file setups as-is
2. Add explicit cross-file dependencies when needed
3. Incrementally adopt new syntax

---

## Migration Guide

### Converting Implicit to Explicit Dependencies

If you're currently using multiple files with implicit ordering:

**Before (Implicit):**
```bash
# Order matters, but not explicit
conductor run foundation.yaml services.yaml integration.yaml
```

**After (Explicit, v2.5+):**

foundation.yaml:
```yaml
tasks:
  - id: 1
    name: Setup
    depends_on: []
```

services.yaml:
```yaml
tasks:
  - id: 2
    name: Service A
    depends_on:
      - file: foundation.yaml
        task: 1
```

integration.yaml:
```yaml
tasks:
  - id: 3
    name: Integrate
    depends_on:
      - file: services.yaml
        task: 2
```

**Benefits of Explicit:**
- Self-documenting dependency relationships
- Works even if file order changes
- Better error messages (shows which file is missing)
- Resumable execution tracks across files

### Detecting Missing Dependencies

Use validation to find undeclared dependencies:

```bash
# Run validation with verbose output
conductor validate foundation.yaml services.yaml integration.yaml --verbose

# Check for missing cross-file dependencies
# If execution fails, update depends_on with explicit file references
```

### Gradual Adoption

You don't need to convert everything at once:

**Stage 1: Keep implicit ordering**
```bash
conductor run file1.yaml file2.yaml file3.yaml  # Still works
```

**Stage 2: Add explicit dependencies to new files**
```yaml
depends_on:
  - file: file1.yaml
    task: 1
```

**Stage 3: Update existing files incrementally**

Convert old syntax:
```yaml
# Old (implicit file ordering)
depends_on: [1, 2]

# New (explicit file references if needed)
depends_on:
  - 1  # Local
  - file: foundation.yaml
    task: 2
```

---

## Troubleshooting

### File Path Issues

**Problem:** "File not found" error

**Debug Steps:**
```bash
# 1. Check if file exists
ls -la foundation.yaml

# 2. Verify file is readable
test -r foundation.yaml && echo "readable"

# 3. Try absolute path
conductor validate /absolute/path/foundation.yaml

# 4. Check for typos in file references
grep "file:" services.yaml
```

### Task Number Issues

**Problem:** "Task X not found in file.yaml"

**Debug Steps:**
```bash
# 1. List all tasks in the file
conductor validate foundation.yaml --verbose

# 2. Check task IDs are correct
grep -E "^\s+(id|task):" foundation.yaml

# 3. Verify cross-file references match actual task numbers
# In services.yaml:
# - file: foundation.yaml
#   task: 2   <- verify task 2 exists in foundation.yaml
```

### Dependency Resolution

**Problem:** Unexpected execution order

**Debug Steps:**
```bash
# 1. Validate plans to see dependency graph
conductor validate *.yaml --verbose

# 2. Check for implicit dependencies (task ordering)
# Conductore calculates waves based on explicit depends_on

# 3. Verify no circular dependencies
# If validation passes, graph is acyclic
```

### Resumable Execution

**Problem:** Resume with `--skip-completed` misses tasks from other files

**Solution:**
```bash
# When using --skip-completed with multi-file plans:
# 1. Conductor tracks completion per file
# 2. Ensure all files are passed in same order

# Correct:
conductor run foundation.yaml services.yaml --skip-completed

# Also correct (different files):
conductor run foundation.yaml services.yaml integration.yaml --skip-completed

# Problem: Different file sets on resume
conductor run foundation.yaml --skip-completed  # Misses services.yaml tasks!
```

**Best Practice:**
```bash
# Always pass all files in same order
PLAN_FILES="foundation.yaml services.yaml integration.yaml"
conductor run $PLAN_FILES
conductor run $PLAN_FILES --skip-completed  # Resume with same files
```

---

## Related Documentation

- **[CLAUDE.md Multi-File Plans](../CLAUDE.md#multi-file-plans)** - Architecture and implementation details
- **[README.md Multi-File Plans](../README.md#multi-file-plans)** - User guide
- **[Migration Guide](#migration-guide)** - Converting existing plans
- **[Troubleshooting Guide](#troubleshooting)** - Common issues and solutions
