# Cross-File Dependencies Reference

Complete guide for handling dependencies across split plan files.

## Overview

When splitting a large plan into multiple files, tasks in one file may depend on tasks in another file. Conductor supports explicit cross-file dependency notation to handle this scenario.

**When you need cross-file dependencies:**
- Task in `plan-02-integration.yaml` depends on Task 2 from `plan-01-foundation.yaml`
- Multi-file orchestration with clear file-to-file dependencies
- Complex features split across infrastructure, feature, and testing phases

## Syntax

### Same-File Dependencies (Numeric)

Within the same file, use simple numeric references:

```yaml
# In any file
tasks:
  - task_number: 5
    depends_on:
      - 2    # Same file - Task 2
      - 3    # Same file - Task 3
```

### Cross-File Dependencies (Object Notation)

When depending on a task in a different file, use object notation:

```yaml
# In plan-02-integration.yaml
tasks:
  - task_number: 5
    depends_on:
      - 4    # Same file - numeric
      - file: "plan-01-foundation.yaml"    # Different file
        task: 2
      - file: "plan-01-foundation.yaml"    # Multiple from same file
        task: 3
```

### Mixed Dependencies

Tasks can depend on both same-file and cross-file tasks:

```yaml
depends_on:
  - 2                              # Same file
  - 4                              # Same file
  - file: "plan-01-foundation.yaml"
    task: 1
  - file: "plan-03-testing.yaml"
    task: 7
```

## File Reference Format

### Valid Formats

```yaml
# Relative path from plan directory
file: "plan-01-foundation.yaml"

# With explicit extension
file: "plan-02-integration.yaml"

# Task number reference (required)
task: 2
```

### File Path Rules

1. **Use relative paths** - Paths relative to split plan directory
2. **Include extension** - Always include `.yaml` or `.yml`
3. **No directory prefixes** - Files should be in the same directory
4. **Consistent naming** - Use plan directory organization:
   ```
   docs/plans/feature-name/
   ├── plan-01-foundation.yaml
   ├── plan-02-integration.yaml
   └── plan-03-testing.yaml
   ```

## Complete Examples

### Example 1: Two-File Backend Plan

**plan-01-foundation.yaml** (Tasks 1-8):
```yaml
plan:
  metadata:
    feature_name: "Backend Implementation"
  tasks:
    - task_number: 1
      name: "Setup PostgreSQL"
      depends_on: []

    - task_number: 2
      name: "Create User Table"
      depends_on: [1]

    - task_number: 3
      name: "Setup Redis Cache"
      depends_on: []

    - task_number: 4
      name: "Create Sessions Table"
      depends_on: [1]

    - task_number: 5
      name: "Setup JWT Library"
      depends_on: []

    # ... more foundation tasks
```

**plan-02-integration.yaml** (Tasks 6-12):
```yaml
plan:
  metadata:
    feature_name: "Backend Implementation"
  tasks:
    - task_number: 6
      name: "Implement Auth Service"
      depends_on:
        - file: "plan-01-foundation.yaml"
          task: 5      # Depends on JWT Library setup
        - file: "plan-01-foundation.yaml"
          task: 4      # Depends on Sessions table

    - task_number: 7
      name: "Implement User API"
      depends_on:
        - file: "plan-01-foundation.yaml"
          task: 2      # Depends on User table
        - 6            # Depends on Auth service (same file)

    - task_number: 8
      name: "Add Caching Layer"
      depends_on:
        - file: "plan-01-foundation.yaml"
          task: 3      # Depends on Redis setup
        - 7            # Depends on User API (same file)

    # ... more integration tasks
```

**Execution Flow:**

1. **Wave 1:** Tasks 1, 3, 5 from foundation (no dependencies)
2. **Wave 2:** Tasks 2, 4 from foundation (depend on 1); Task 6 from integration (cross-file depends on 5, 4)
3. **Wave 3:** Task 7 from integration (depends on 6)
4. **Wave 4:** Task 8 from integration (cross-file depends on 3, same-file depends on 7)

### Example 2: Three-File Microservices

**plan-01-auth-service.yaml**:
```yaml
tasks:
  - task_number: 1
    name: "Setup Auth Service"
    depends_on: []

  - task_number: 2
    name: "Implement Login Endpoint"
    depends_on: [1]

  - task_number: 3
    name: "Add Token Refresh"
    depends_on: [2]
```

**plan-02-api-service.yaml**:
```yaml
tasks:
  - task_number: 4
    name: "Setup API Server"
    depends_on: []

  - task_number: 5
    name: "Setup Database"
    depends_on: [4]

  - task_number: 6
    name: "Add Auth Middleware"
    depends_on:
      - 5
      - file: "plan-01-auth-service.yaml"
        task: 3    # Cross-file: Auth service must be complete

  - task_number: 7
    name: "Implement User CRUD"
    depends_on:
      - 5
      - 6
```

**plan-03-deployment.yaml**:
```yaml
tasks:
  - task_number: 8
    name: "Docker Configuration"
    depends_on: []

  - task_number: 9
    name: "Kubernetes Manifests"
    depends_on:
      - 8
      - file: "plan-02-api-service.yaml"
        task: 7    # Cross-file: API must be ready
      - file: "plan-01-auth-service.yaml"
        task: 3    # Cross-file: Auth service must be ready

  - task_number: 10
    name: "Deploy to Production"
    depends_on:
      - 9
```

**Execution with cross-file constraints:**
- Auth service (plan 1) builds independently
- API service (plan 2) waits for auth to complete
- Deployment (plan 3) waits for both services

## Validation Checklist

Before finalizing cross-file dependencies:

### Syntax Validation
- [ ] All cross-file references use `{file: "...", task: N}` format
- [ ] File paths are relative (no directory prefixes)
- [ ] File names include `.yaml` or `.yml` extension
- [ ] Task numbers are valid integers
- [ ] Files exist in the split plan directory

### Reference Validation
- [ ] All referenced files exist
- [ ] All referenced task numbers exist in their files
- [ ] No typos in file names
- [ ] No circular dependencies across files
- [ ] All files parse as valid YAML

### Logical Validation
- [ ] Dependencies make architectural sense
- [ ] Dependent task comes AFTER its dependencies in the plan
- [ ] No task depends on itself (directly or indirectly)
- [ ] Cross-file deps only for actual cross-file relationships
- [ ] Same-file deps use numeric notation (not cross-file)

### File Organization
- [ ] Files are in same directory (no nested subdirectories)
- [ ] File names follow pattern: `plan-NN-description.yaml`
- [ ] Plan directory structure is clear to reader
- [ ] Each file has valid `conductor` and `plan` root sections

## Best Practices

### DO Use Cross-File Dependencies For:

1. **Phase transitions** - Foundation → Integration → Testing
   ```yaml
   # In integration phase file
   depends_on:
     - file: "plan-01-foundation.yaml"
       task: 5    # Foundation task must complete first
   ```

2. **Microservice coordination** - Coordinating multiple services
   ```yaml
   depends_on:
     - file: "plan-02-user-service.yaml"
       task: 3    # User service ready before integrating
   ```

3. **Infrastructure dependencies** - Deployment depends on features
   ```yaml
   depends_on:
     - file: "plan-02-features.yaml"
       task: 12   # Features implemented before deployment
   ```

### DON'T Use Cross-File Dependencies For:

1. **Same-file dependencies** - Use numeric notation instead
   ```yaml
   # WRONG
   depends_on:
     - file: "plan-01-foundation.yaml"
       task: 2    # If in same file, use numeric

   # CORRECT
   depends_on:
     - 2
   ```

2. **Weak relationships** - Only genuine sequential dependencies
   ```yaml
   # AVOID if optional
   depends_on:
     - file: "plan-02-other.yaml"
       task: 5    # Should this really block execution?
   ```

3. **Circular patterns** - Avoid cycles across files
   ```yaml
   # AVOID - creates circular dependency
   # plan-01: Task 1 depends on (plan-02: Task 3)
   # plan-02: Task 3 depends on (plan-01: Task 1)
   ```

## Validation in Conductor

### Single File Plan
```bash
conductor validate docs/plans/feature-slug.yaml
```

**Checks:**
- YAML syntax
- Numeric references exist
- No cycles

### Multi-File Plan
```bash
conductor validate docs/plans/feature-slug/
```

**Checks:**
- Each file parses as valid YAML
- All numeric references exist within files
- All cross-file references:
  - File exists
  - Target task exists in file
  - No forward references (dependency before dependee)
  - No circular dependencies (including cross-file)

## File Organization Examples

### Well-Organized (Conductor Can Validate)
```
docs/plans/ecommerce-backend/
├── plan-01-database.yaml       (Tasks 1-5)
├── plan-02-authentication.yaml (Tasks 6-10, depends on plan-01)
├── plan-03-api-layer.yaml      (Tasks 11-15, depends on plan-01, plan-02)
└── plan-04-deployment.yaml     (Tasks 16-18, depends on plan-01, 02, 03)
```

File naming makes dependencies clear:
- Foundation (database) comes first
- Auth builds on database
- API builds on both
- Deployment builds on all

### Poorly Organized (Hard to Validate)
```
docs/plans/features/
├── a.yaml   (Tasks 1-5)
├── b.yaml   (Tasks 6-10, depends on a, c, e)
├── c.yaml   (Tasks 11-15, depends on a, d)
├── d.yaml   (Tasks 16-20, depends on a, b, c)
├── e.yaml   (Tasks 21-25, depends on a, b, d)
```

Too many cross-file dependencies make execution harder to understand.

## Troubleshooting

### Error: "File not found"

```
conductor validate docs/plans/feature/
Error: Referenced file 'plan-01-foundation.yaml' not found
```

**Solution:**
- Check file exists in split plan directory
- Verify spelling and extension
- Ensure relative path is correct

### Error: "Task number does not exist"

```
Error: plan-01-foundation.yaml does not contain task 25
```

**Solution:**
- Verify task number exists in referenced file
- Check task numbering is correct
- Ensure file contains the task

### Error: "Circular dependency"

```
Error: Circular dependency detected: Task 5 → Task 8 → Task 5
```

**Solution:**
- Review cross-file references
- Eliminate cycle by removing or reordering dependencies
- Consider if dependencies are actually necessary

### Warning: "Forward reference"

```
Warning: Task 15 in plan-02-integration.yaml depends on Task 20
(Task 20 appears later in the plan)
```

**Solution:**
- Reorder tasks so dependencies come before dependents
- Or split into more files to make ordering clearer

## Summary

**Cross-file dependencies enable:**
- ✅ Logical phase separation (foundation → integration → testing)
- ✅ Clear architectural boundaries
- ✅ Parallel development of independent modules
- ✅ Coordinated execution across multiple files

**Key syntax:**
- Same file: Use numeric `- 2`
- Different file: Use object `- {file: "...", task: 2}`

**Always validate:**
- Single file: `conductor validate plan.yaml`
- Multi-file: `conductor validate docs/plans/feature/`
