# Cross-File Dependency Test Fixtures

This directory contains test fixtures demonstrating cross-file dependency support in Conductor's YAML and Markdown parsers.

## Quick Start

### YAML Format

**File**: `cross-file-simple.yaml`
```yaml
tasks:
  - task_number: 2
    depends_on:
      - file: "cross-file-mixed.yaml"
        task: 1
```

**File**: `cross-file-mixed.yaml`
```yaml
tasks:
  - task_number: 3
    depends_on:
      - 2  # Local dependency
      - file: "cross-file-simple.yaml"
        task: 2  # Cross-file dependency
```

### Markdown Format

**File**: `cross-file-simple.md`
```markdown
**Depends on**: file:cross-file-mixed.md:task:1
```

**File**: `cross-file-mixed.md`
```markdown
**Depends on**: Task 2, file:cross-file-simple.md/task:2
```

## Fixture Organization

### Top-Level Fixtures

1. **cross-file-simple.yaml / cross-file-simple.md**
   - Demonstrates basic cross-file dependencies
   - Task 1: Setup Database (no dependencies)
   - Task 2: API Layer (depends on cross-file-mixed task 1)

2. **cross-file-mixed.yaml / cross-file-mixed.md**
   - Demonstrates mixed local and cross-file dependencies
   - Task 1: Implement Authentication (no dependencies)
   - Task 2: Setup Cache Layer (depends on local task 1)
   - Task 3: Wire Everything (depends on local task 2 + cross-file task 2)

### Subdirectory Fixtures

#### cross-file-fixtures/

**split-plan-linear/**
- Linear dependency chain across multiple files
- `plan-01-setup.yaml`: Foundation tasks
- `plan-02-features.yaml`: Feature tasks depending on setup

**split-plan-complex/**
- Complex dependency patterns
- Multiple files with interdependencies
- Demonstrates real-world scenarios

**split-plan-diamond/**
- Diamond dependency pattern
- Two files converging to single task

**split-plan-mixed/**
- Mixed local and cross-file dependencies
- Demonstrates all supported formats

**split-plan-markdown/**
- Markdown format fixtures
- `plan-01-setup.md`: Setup phase
- `plan-02-features.md`: Features phase with cross-file deps

## Dependency Format Reference

### YAML Formats

```yaml
# Numeric (local)
depends_on: [1, 2, 3]

# Cross-file (explicit)
depends_on:
  - file: "plan-01.yaml"
    task: 2

# Mixed
depends_on:
  - 1
  - file: "plan-02.yaml"
    task: 3
  - 2

# Cross-file with float task
depends_on:
  - file: "plan.yaml"
    task: 2.5

# Cross-file with alphanumeric task
depends_on:
  - file: "plan.yaml"
    task: "integration-1"
```

### Markdown Formats

```markdown
# Numeric (local)
**Depends on**: 1, 2, 3

# Cross-file (slash notation)
**Depends on**: file:plan-01.yaml/task:2

# Cross-file (colon notation)
**Depends on**: file:plan-01.yaml:task:2

# Cross-file with spaces
**Depends on**: file:plan-01.yaml / task:2

# Task notation
**Depends on**: Task 1, Task 2

# Mixed
**Depends on**: Task 1, file:plan-02.yaml/task:3, 2
```

## Normalized Representation

All formats normalize to:
```
file:{filename}:task:{task-id}
```

Examples:
- `file:plan-01.yaml:task:2`
- `file:plan-02-auth.yaml:task:integration-1`
- `file:features/plan-03-api.yaml:task:2.5`

## Testing with Fixtures

### Parse a Fixture

```bash
# Test with the conductor binary
./conductor validate internal/parser/testdata/cross-file-simple.yaml

# Or run the full test suite
go test ./internal/parser -v -run "CrossFile"
```

### Key Test Cases

1. **simple_cross-file_dependency**: Single cross-file reference
2. **mixed_numeric_and_cross-file**: Mix of local and cross-file
3. **multiple_cross_file**: Several files referenced
4. **cross_file_with_float_task**: Float task IDs
5. **cross_file_with_string_task**: Alphanumeric task IDs
6. **backward_compatibility**: Existing format still works
7. **error_cases**: Missing required fields

## Error Handling

Invalid cross-file dependencies produce clear error messages:

```
cross-file dependency: missing required 'file' field
cross-file dependency: missing required 'task' field
cross-file dependency 'task' must be int/float/string, got ...
```

## Use Cases

These fixtures demonstrate:

1. **Sequential Dependencies**: Task chain across files
2. **Parallel Setup**: Independent setup tasks that converge
3. **Integration Testing**: Wiring multiple components
4. **Microservices**: Separate service plans with dependencies
5. **Modular Projects**: Large projects split into logical modules

## File Paths

All files in fixtures use relative paths:
- No absolute paths
- No environment-specific paths
- Work in any directory

## Adding New Fixtures

When adding new fixtures:

1. **Keep them small**: 2-4 tasks per file
2. **Clear naming**: `cross-file-{scenario}.{format}`
3. **Include documentation**: Describe the dependency pattern
4. **Add test case**: Update test file to use fixture
5. **Test parsing**: Ensure parser handles it correctly

## Example: Multi-Service Setup

```yaml
# plan-01-database.yaml
tasks:
  - task_number: 1
    name: "Initialize DB"
    depends_on: []

# plan-02-auth.yaml
tasks:
  - task_number: 2
    name: "Setup Auth"
    depends_on:
      - file: "plan-01-database.yaml"
        task: 1

# plan-03-api.yaml
tasks:
  - task_number: 3
    name: "Implement API"
    depends_on:
      - file: "plan-02-auth.yaml"
        task: 2
```

This creates a three-file dependency chain: Database → Auth → API

## Supported Features

✓ Multiple files in single execution
✓ Cross-file task dependencies
✓ Mixed local and cross-file dependencies
✓ Float and string task IDs
✓ Error detection and reporting
✓ Backward compatibility with single-file plans
✓ YAML and Markdown formats
✓ Whitespace-tolerant parsing

## Related Documentation

- See `CROSS_FILE_DEPS_IMPLEMENTATION.md` for implementation details
- See `internal/models/task.go` for data structures
- See parser test files for comprehensive test cases
