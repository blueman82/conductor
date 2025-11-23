# Cross-File Dependency Examples and Usage Guide

**Purpose**: Practical examples and usage patterns for Conductor's cross-file dependency feature.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Basic Examples](#basic-examples)
3. [Advanced Patterns](#advanced-patterns)
4. [Common Issues](#common-issues)
5. [Testing Guide](#testing-guide)

---

## Quick Start

### The Problem

You have a large implementation plan with 20+ tasks. Keeping everything in one file becomes unwieldy:

```markdown
# 20-Task Monolithic Plan

## Task 1: Setup...
## Task 2: Features...
## Task 3: More Features...
... 17 more tasks ...
```

**Better**: Split into logical files with clear dependencies.

### The Solution

Create separate plan files for logical phases:

```
plans/
  plan-01-foundation.yaml    # Infrastructure, DB setup (Tasks 1-3)
  plan-02-services.yaml      # Business logic (Tasks 4-6)
  plan-03-integration.yaml   # Wiring components (Tasks 7-9)
  plan-04-deployment.yaml    # Docker, CI/CD (Tasks 10-12)
```

Express cross-file dependencies explicitly:

```yaml
# plan-02-services.yaml
tasks:
  - task_number: 4
    depends_on:
      - file: "plan-01-foundation.yaml"   # Reference another file
        task: 2                            # And its task
```

Run them together with Conductor:

```bash
conductor run plans/plan-*.yaml
```

---

## Basic Examples

### Example 1: Two-Phase Plan (Linear)

**Scenario**: Setup phase must complete before features phase.

#### plan-01-setup.yaml

```yaml
conductor:
  default_agent: godev
  quality_control:
    enabled: true
    review_agent: code-reviewer

plan:
  metadata:
    feature_name: "Setup Phase"
  tasks:
    - task_number: 1
      name: "Initialize Repository"
      files: [go.mod, main.go]
      depends_on: []
      estimated_time: "15m"
      description: "Initialize Go module and entry point"

    - task_number: 2
      name: "Setup CLI Framework"
      files: [cmd/root.go]
      depends_on: [1]           # Local dependency (same file)
      estimated_time: "30m"
      description: "Add Cobra CLI framework"
```

#### plan-02-features.yaml

```yaml
conductor:
  default_agent: godev

plan:
  metadata:
    feature_name: "Features Phase"
  tasks:
    - task_number: 3
      name: "Implement Feature"
      files: [internal/feature/feature.go]
      depends_on:
        - file: "plan-01-setup.yaml"     # Cross-file dependency!
          task: 2                        # Task 2 from setup must complete first
      estimated_time: "1h"
      description: "Feature implementation"

    - task_number: 4
      name: "Add Tests"
      files: [internal/feature/feature_test.go]
      depends_on: [3]                    # Local dependency (same file)
      estimated_time: "45m"
      description: "Test suite for feature"
```

#### Execution Plan

```
Wave 1: [1]              # Task 1 from plan-01 (no dependencies)
Wave 2: [2]              # Task 2 from plan-01 (depends on 1)
Wave 3: [3]              # Task 3 from plan-02 (depends on plan-01:Task 2)
Wave 4: [4]              # Task 4 from plan-02 (depends on 3)
```

#### Running

```bash
# Single command loads both files and executes in correct order
conductor run plan-01-setup.yaml plan-02-features.yaml

# Or using glob pattern
conductor run "*.yaml"

# Dry-run to see execution plan first
conductor run --dry-run plan-*.yaml

# Validate dependencies before running
conductor validate plan-*.yaml
```

---

### Example 2: Diamond Dependency Pattern

**Scenario**: Two services depend on shared foundation, then integration task depends on both.

#### plan-01-foundation.yaml

```yaml
plan:
  metadata:
    feature_name: "Foundation"
  tasks:
    - task_number: 1
      name: "Database Setup"
      files: [internal/db/db.go]
      depends_on: []
      estimated_time: "30m"
      description: "Initialize database"
```

#### plan-02-services.yaml

```yaml
plan:
  metadata:
    feature_name: "Services"
  tasks:
    - task_number: 2
      name: "Auth Service"
      files: [internal/auth/auth.go]
      depends_on:
        - file: "plan-01-foundation.yaml"
          task: 1
      estimated_time: "1h"
      description: "Auth service using database"

    - task_number: 3
      name: "API Service"
      files: [internal/api/api.go]
      depends_on:
        - file: "plan-01-foundation.yaml"
          task: 1
      estimated_time: "1h"
      description: "API service using database"
```

#### plan-03-integration.yaml

```yaml
plan:
  metadata:
    feature_name: "Integration"
  tasks:
    - task_number: 4
      name: "Wire Services"
      files: [cmd/main.go]
      depends_on:
        - file: "plan-02-services.yaml"
          task: 2
        - file: "plan-02-services.yaml"
          task: 3
      estimated_time: "30m"
      description: "Integrate auth and API"
```

#### Execution Plan

```
Wave 1: [1]              # Task 1 (foundation, no deps)
Wave 2: [2, 3]           # Tasks 2, 3 (both depend on 1, can run in parallel)
Wave 3: [4]              # Task 4 (depends on 2 and 3)
```

**Key Insight**: Tasks 2 and 3 execute in parallel (Wave 2) even though they're in different files, because they're independent!

---

### Example 3: Cross-Directory References

**Scenario**: Plans are organized in subdirectories.

```
implementation/
  plans/
    plan-01-setup.yaml
    services/
      plan-02-auth.yaml
      plan-03-api.yaml
    deployment/
      plan-04-deploy.yaml
```

#### plan-03-api.yaml (in services/)

```yaml
plan:
  metadata:
    feature_name: "API Service"
  tasks:
    - task_number: 3
      name: "API Implementation"
      files: [internal/api/api.go]
      depends_on:
        - file: "../plan-02-auth.yaml"   # Go up one directory
          task: 2
        - file: "../../plan-01-setup.yaml"  # Go up two directories
          task: 1
      estimated_time: "1h"
      description: "API depending on auth and foundation"
```

**Note**: File paths are relative to the current plan file's directory.

#### Validation

```bash
# Validate from implementation/ directory
cd implementation
conductor validate plans/plan-*.yaml plans/services/*.yaml plans/deployment/*.yaml

# Or with glob (if supported)
conductor validate "plans/**/*.yaml"
```

---

## Advanced Patterns

### Pattern 1: Multi-Stage Build

**Scenario**: Development -> Testing -> Staging -> Production pipeline.

```
plans/
  01-development.yaml  # Dev environment (5 tasks)
  02-testing.yaml      # Test pipeline (4 tasks)
  03-staging.yaml      # Staging deployment (3 tasks)
  04-production.yaml   # Prod deployment (2 tasks)
```

Each stage depends on previous completion:

```yaml
# 02-testing.yaml
tasks:
  - task_number: 6
    name: "Run Integration Tests"
    depends_on:
      - file: "01-development.yaml"
        task: 5            # Development phase must be complete
    description: "Test integration with development code"
```

```yaml
# 03-staging.yaml
tasks:
  - task_number: 9
    name: "Deploy to Staging"
    depends_on:
      - file: "02-testing.yaml"
        task: 8            # All tests must pass
    description: "Deploy tested code to staging"
```

**Validation**: Each file can be validated independently, but execution enforces the full pipeline order.

---

### Pattern 2: Conditional Feature Branches

**Scenario**: Core features everyone implements, optional features some teams do.

```
plans/
  core/
    plan-01-api.yaml        # Required for all
    plan-02-auth.yaml       # Required for all
  optional/
    plan-03-cache.yaml      # Optional caching layer
    plan-04-monitoring.yaml # Optional observability
```

Teams can pick and choose:

```bash
# Minimal: Just core
conductor run core/*.yaml

# With caching: Core + cache
conductor run core/*.yaml optional/plan-03-cache.yaml

# Full features: Everything
conductor run "**/*.yaml"
```

**Cross-file refs allow clear dependencies**:

```yaml
# optional/plan-04-monitoring.yaml
tasks:
  - task_number: 11
    name: "Setup Monitoring"
    depends_on:
      - file: "../core/plan-01-api.yaml"
        task: 2            # Monitoring depends on API setup
    description: "Add observability to API"
```

---

### Pattern 3: Refactoring Monolith

**Scenario**: Breaking down a monolithic plan into services.

**Before**: One 30-task plan

```markdown
# Huge Plan
## Task 1: ...
## Task 2: ...
...
## Task 30: ...
```

**After**: Logical split

```
refactor/
  plan-01-extract-db.yaml     # Extract DB layer (5 tasks)
  plan-02-extract-auth.yaml   # Extract auth (4 tasks)
  plan-03-extract-api.yaml    # Extract API (6 tasks)
  plan-04-extract-cache.yaml  # Extract cache (3 tasks)
  plan-05-integration.yaml    # Rewire (2 tasks)
```

With explicit dependencies:

```yaml
# plan-03-extract-api.yaml
tasks:
  - task_number: 10
    name: "Extract API Layer"
    depends_on:
      - file: "plan-01-extract-db.yaml"
        task: 3            # DB must be extracted first
      - file: "plan-02-extract-auth.yaml"
        task: 7            # Auth must be extracted first
    description: "Extract API layer from monolith"
```

**Benefit**: Clear visibility into what depends on what during refactoring.

---

## Common Issues

### Issue 1: "Task Not Found" Error

**Error Message**:
```
Error: Task "5" references missing task in "plan-02-features.yaml"
Available tasks: 1, 2, 3
```

**Cause**: Typo in file name or task number.

**Fix**:
```yaml
# WRONG
depends_on:
  - file: "plan-02-feature.yaml"  # Typo: feature vs features
    task: 5

# RIGHT
depends_on:
  - file: "plan-02-features.yaml"  # Correct file name
    task: 3                        # Correct task number
```

**Prevention**:
- Double-check file names match exactly
- Use `conductor validate` before running
- Plan file names should be obvious (use prefixes like `plan-01-`, `plan-02-`)

---

### Issue 2: "File Not Found" Error

**Error Message**:
```
Error: Cross-file dependency in plan-02-features.yaml
File not found: "subfolder/plan-01.yaml"
```

**Cause**: File path is wrong or file hasn't been created yet.

**Fix**:
```yaml
# If files are in same directory
depends_on:
  - file: "plan-01-setup.yaml"
    task: 1

# If files are in subdirectory
depends_on:
  - file: "subfolder/plan-01-setup.yaml"
    task: 1

# If files are in parent directory
depends_on:
  - file: "../plan-01-setup.yaml"
    task: 1
```

**Prevention**:
- Use `pwd` to check current directory
- Use absolute paths in validation: `conductor validate /full/path/to/plan-*.yaml`
- Files must exist before validation

---

### Issue 3: Circular Dependency

**Error Message**:
```
Error: Circular dependency detected
Cycle: plan-01.yaml:Task 1 -> plan-02.yaml:Task 2 -> plan-01.yaml:Task 1
```

**Cause**: Dependency creates a loop.

**Bad Design**:
```yaml
# plan-01.yaml, Task 1
depends_on:
  - file: "plan-02.yaml"
    task: 2

# plan-02.yaml, Task 2
depends_on:
  - file: "plan-01.yaml"
    task: 1
```

**Fix**: Identify and remove the circular reference.

```yaml
# plan-01.yaml, Task 1 (no cross-file deps)
depends_on: []

# plan-02.yaml, Task 2 (depends on Task 1 only)
depends_on:
  - file: "plan-01.yaml"
    task: 1
```

**Prevention**:
- Think about it: Does A need to wait for B? And does B need to wait for A?
- If yes, you have a circular dependency
- Refactor to break the cycle (usually one task doesn't really need the other)

---

### Issue 4: Single File with Cross-File References

**Problem**: Testing a single plan file that has cross-file references.

**Context**: You're working on `plan-02-features.yaml` which references `plan-01-setup.yaml`.

```bash
# This fails if plan-01-setup.yaml doesn't exist
conductor validate plan-02-features.yaml
```

**Solution**: Validate the full set of files together.

```bash
# Validate all files together
conductor validate plan-01-setup.yaml plan-02-features.yaml

# Or from a directory
conductor validate plans/
```

**Note**: Single-file validation (v1) doesn't check cross-file refs. Multi-file validation (v2) does.

---

## Testing Guide

### Running the Test Fixtures

Test fixtures are in: `internal/parser/testdata/cross-file-fixtures/`

```bash
# Test fixture 1: Linear chain (simple)
conductor validate \
  internal/parser/testdata/cross-file-fixtures/split-plan-linear/plan-01-setup.yaml \
  internal/parser/testdata/cross-file-fixtures/split-plan-linear/plan-02-features.yaml

# Test fixture 2: Diamond pattern (parallel execution)
conductor validate \
  internal/parser/testdata/cross-file-fixtures/split-plan-diamond/plan-01-foundation.yaml \
  internal/parser/testdata/cross-file-fixtures/split-plan-diamond/plan-02-services.yaml \
  internal/parser/testdata/cross-file-fixtures/split-plan-diamond/plan-03-integration.yaml

# Test fixture 3: Complex with subdirectories
conductor validate \
  internal/parser/testdata/cross-file-fixtures/split-plan-complex/plan-01-foundation.yaml \
  internal/parser/testdata/cross-file-fixtures/split-plan-complex/features/plan-02-auth.yaml \
  internal/parser/testdata/cross-file-fixtures/split-plan-complex/features/plan-03-api.yaml \
  internal/parser/testdata/cross-file-fixtures/split-plan-complex/deployment/plan-04-deploy.yaml

# Test fixture 4: Markdown format (if supported)
conductor validate \
  internal/parser/testdata/cross-file-fixtures/split-plan-markdown/plan-01-setup.md \
  internal/parser/testdata/cross-file-fixtures/split-plan-markdown/plan-02-features.md

# Test fixture 5: Mixed numeric and cross-file
conductor validate \
  internal/parser/testdata/cross-file-fixtures/split-plan-mixed/plan-01.yaml \
  internal/parser/testdata/cross-file-fixtures/split-plan-mixed/plan-02.yaml
```

### Running Unit Tests

```bash
# Run all parser tests
go test ./internal/parser/ -v

# Run specific test
go test ./internal/parser/ -run TestParseMixedDependencies -v

# Run with coverage
go test ./internal/parser/ -cover

# Run integration tests
go test ./internal/executor/ -v -run TestCalculateWaves
```

### Creating Your Own Test Plan

**Step 1**: Create a directory structure

```
my-test-plans/
  plan-01-base.yaml
  plan-02-features.yaml
```

**Step 2**: Write plan files with cross-file deps

```yaml
# plan-01-base.yaml
plan:
  tasks:
    - task_number: 1
      name: "Setup"
      depends_on: []

# plan-02-features.yaml
plan:
  tasks:
    - task_number: 2
      name: "Feature"
      depends_on:
        - file: "plan-01-base.yaml"
          task: 1
```

**Step 3**: Validate

```bash
conductor validate my-test-plans/plan-*.yaml
```

**Step 4**: Check the dependency graph

```bash
# Should show 2 waves: Wave 1: [1], Wave 2: [2]
conductor validate --verbose my-test-plans/plan-*.yaml
```

---

## Best Practices

### Naming Convention

Use clear file names with numeric prefixes:

```
GOOD:
  plan-01-foundation.yaml    # Clear order
  plan-02-services.yaml
  plan-03-integration.yaml

BAD:
  setup.yaml                 # Unclear order
  features.yaml
  final.yaml
```

### Directory Organization

```
project/
  plans/
    plan-01-foundation.yaml
    services/
      plan-02-auth.yaml
      plan-03-api.yaml
    infrastructure/
      plan-04-docker.yaml
      plan-05-k8s.yaml
```

### Dependency Style

Prefer explicit naming in complex plans:

```yaml
# Good for clear intent
depends_on:
  - file: "plan-02-services.yaml"
    task: 3

# Less clear
depends_on: [3]  # Is this Task 3 in current file or another file?
```

### Documentation

Add comments in cross-file deps:

```yaml
tasks:
  - task_number: 5
    name: "Setup Monitoring"
    depends_on:
      - file: "plan-02-services.yaml"  # Must have services first
        task: 4                        # Task 4: API service implementation
    description: "Add observability to API from plan-02"
```

---

## Migration Guide

### From Single Large Plan to Multiple Files

**Before**:
```
single-plan.yaml  # 30 tasks in one file
```

**After**:
```
plans/
  plan-01-foundation.yaml  # Tasks 1-5
  plan-02-services.yaml    # Tasks 6-15 (depends on foundation)
  plan-03-integration.yaml # Tasks 16-25 (depends on services)
  plan-04-deployment.yaml  # Tasks 26-30 (depends on integration)
```

**Steps**:
1. Open single-plan.yaml
2. Identify logical groupings (foundation, features, integration, etc.)
3. Create new files for each group
4. Copy relevant tasks to new files
5. Update task numbers if needed (or keep original)
6. Add cross-file dependencies to link files
7. Test with `conductor validate plan-*.yaml`
8. Run with `conductor run plan-*.yaml`

**Example Mapping**:
```yaml
# OLD: single-plan.yaml with Tasks 1-30
# NEW: split into 4 files

# plan-01-foundation.yaml
tasks: [1, 2, 3, 4, 5]

# plan-02-services.yaml
tasks: [6, 7, 8, ..., 15]
tasks[*].depends_on:        # Add cross-file refs
  - file: "plan-01-foundation.yaml"
    task: 5

# plan-03-integration.yaml
tasks: [16, 17, 18, ..., 25]
tasks[*].depends_on:        # Add cross-file refs
  - file: "plan-02-services.yaml"
    task: 15

# plan-04-deployment.yaml
tasks: [26, 27, 28, 29, 30]
tasks[*].depends_on:        # Add cross-file refs
  - file: "plan-03-integration.yaml"
    task: 25
```

---

## Summary

Cross-file dependencies enable:

1. **Modular Plans**: Large plans split into logical files
2. **Clear Dependencies**: Explicit cross-file references
3. **Parallel Execution**: Tasks in different files can run in parallel if independent
4. **Better Organization**: Large projects become manageable
5. **Team Collaboration**: Teams can work on different files independently

**Remember**:
- Numeric-only depends_on = local to current file
- Cross-file = explicit `{file: "...", task: X}` format
- `conductor validate` checks cross-file refs
- `conductor run` respects cross-file dependencies
- Relative paths are relative to the current file

---

## Next Steps

1. Read [CROSS_FILE_DEPENDENCY_TEST_PLAN.md](./CROSS_FILE_DEPENDENCY_TEST_PLAN.md) for testing details
2. Review test fixtures in `internal/parser/testdata/cross-file-fixtures/`
3. Run test fixtures with `conductor validate`
4. Create your own split plan using the patterns above
5. Run with `conductor run` and verify execution order

