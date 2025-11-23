# Cross-File Dependency Examples

Complete practical examples showing how to use cross-file dependencies in Conductor.

---

## Example 1: Simple Two-File Plan

A basic microservices setup with foundation tasks and feature tasks.

### Plan File 1: plan-01-foundation.yaml

```yaml
conductor:
  default_agent: golang-pro
  quality_control:
    enabled: true
    retry_on_red: 2

plan:
  metadata:
    feature_name: "Foundation Setup"
    created: "2025-11-23"
  tasks:
    - task_number: 1
      name: "Initialize PostgreSQL Database"
      files:
        - internal/db/postgres.go
        - migrations/001_init_schema.sql
      depends_on: []
      agent: golang-pro
      estimated_time: 30m
      description: |
        Set up PostgreSQL database connection pool and initialization.
        Create connection string from environment variables.
      test_commands:
        - "go test ./internal/db -v"

    - task_number: 2
      name: "Create User Table"
      files:
        - migrations/002_create_users.sql
      depends_on: [1]
      agent: golang-pro
      estimated_time: 15m
      description: |
        Create the users table with email as unique identifier.
        Include timestamps and password hash field.
      test_commands:
        - "psql -d conductor_test -c '\\dt users'"
```

### Plan File 2: plan-02-features.yaml

```yaml
plan:
  metadata:
    feature_name: "User API Features"
    created: "2025-11-23"
  tasks:
    - task_number: 3
      name: "Implement User Create Endpoint"
      files:
        - internal/api/handlers/user.go
        - internal/api/handlers/user_test.go
      depends_on:
        - file: "plan-01-foundation.yaml"
          task: 2
      agent: golang-pro
      estimated_time: 45m
      description: |
        Implement POST /users endpoint to create new users.
        Validate email format and hash passwords using bcrypt.
        Return 201 Created with user ID on success.
      success_criteria:
        - "Endpoint accepts POST /users with email and password"
        - "Returns 201 Created on success"
        - "Returns 400 Bad Request for invalid email"
        - "Password is hashed before storage"
      test_commands:
        - "go test ./internal/api/handlers -run TestCreateUser -v"

    - task_number: 4
      name: "Implement User Get Endpoint"
      files:
        - internal/api/handlers/user.go
        - internal/api/handlers/user_test.go
      depends_on:
        - file: "plan-01-foundation.yaml"
          task: 2
        - 3
      agent: golang-pro
      estimated_time: 30m
      description: |
        Implement GET /users/:id endpoint to retrieve user by ID.
        Return 404 if user not found.
      success_criteria:
        - "Endpoint accepts GET /users/:id"
        - "Returns 200 OK with user data"
        - "Returns 404 Not Found for invalid user ID"
      test_commands:
        - "go test ./internal/api/handlers -run TestGetUser -v"
```

**Execution Order (Calculated by Conductor):**
- Wave 1: Task 1
- Wave 2: Task 2
- Wave 3: Task 3
- Wave 4: Task 4

---

## Example 2: Mixed Format Dependencies

Showing numeric, structured local, and cross-file references in one task.

```yaml
tasks:
  - task_number: 5
    name: "Integration Task"
    files: [internal/integration.go]
    depends_on:
      - 2                                   # Numeric local reference
      - file: "plan-01-foundation.yaml"    # Cross-file reference
        task: 1
      - file: "../setup/plan-core.yaml"    # Cross-file with relative path
        task: 3
    agent: golang-pro
    description: |
      This task depends on:
      - Task 2 from the same file (numeric)
      - Task 1 from plan-01-foundation.yaml (cross-file)
      - Task 3 from ../setup/plan-core.yaml (cross-file with path)
```

---

## Example 3: Complex Microservices

A realistic three-file plan showing infrastructure, services, and integration.

### File: plan-01-infrastructure.yaml

```yaml
plan:
  metadata:
    feature_name: "Infrastructure Setup"
  tasks:
    - task_number: 1
      name: "Setup Docker Environment"
      files: [docker-compose.yaml, Dockerfile]
      depends_on: []
      estimated_time: 20m

    - task_number: 2
      name: "Initialize PostgreSQL"
      files: [migrations/001_init.sql]
      depends_on: [1]
      estimated_time: 15m

    - task_number: 3
      name: "Setup Redis Cache"
      files: [docker-compose.yaml]
      depends_on: [1]
      estimated_time: 10m
```

### File: plan-02-services.yaml

```yaml
plan:
  metadata:
    feature_name: "Core Services"
  tasks:
    - task_number: 4
      name: "Database Connection Service"
      files: [internal/db/service.go]
      depends_on:
        - file: "plan-01-infrastructure.yaml"
          task: 2
      estimated_time: 30m

    - task_number: 5
      name: "Cache Connection Service"
      files: [internal/cache/service.go]
      depends_on:
        - file: "plan-01-infrastructure.yaml"
          task: 3
      estimated_time: 20m

    - task_number: 6
      name: "Configuration Manager"
      files: [internal/config/manager.go]
      depends_on:
        - file: "plan-01-infrastructure.yaml"
          task: 1
      estimated_time: 25m
```

### File: plan-03-integration.yaml

```yaml
plan:
  metadata:
    feature_name: "Integration"
  tasks:
    - task_number: 7
      name: "Wire Services Together"
      type: integration
      files: [cmd/api/main.go]
      depends_on:
        - file: "plan-02-services.yaml"
          task: 4
        - file: "plan-02-services.yaml"
          task: 5
        - file: "plan-02-services.yaml"
          task: 6
      estimated_time: 40m
      success_criteria:
        - "Services are initialized in correct order"
        - "Database service is ready before use"
        - "Cache service is ready before use"
      integration_criteria:
        - "Database service receives config from Configuration Manager"
        - "Cache service receives config from Configuration Manager"
        - "All services start without errors"
      test_commands:
        - "go test ./cmd/api -v"

    - task_number: 8
      name: "Write End-to-End Tests"
      files: [test/e2e/main_test.go]
      depends_on:
        - 7
        - file: "plan-01-infrastructure.yaml"
          task: 1
      estimated_time: 45m
      test_commands:
        - "go test ./test/e2e -v"
```

**Execution Waves:**
- Wave 1: Task 1
- Wave 2: Tasks 2, 3
- Wave 3: Tasks 4, 5, 6
- Wave 4: Task 7
- Wave 5: Task 8

---

## Error Handling Examples

### Unresolved Cross-File Reference

**Error:**
```
Error: task 5 (User API): cross-file reference to task 2 in plan-01-foundation.yaml not found
Available tasks in plan-01-foundation.yaml: 1, 3, 4
```

**Fix:**
```yaml
# Before (incorrect)
depends_on:
  - file: "plan-01-foundation.yaml"
    task: 2  # Task 2 doesn't exist

# After (correct)
depends_on:
  - file: "plan-01-foundation.yaml"
    task: 3  # Task 3 exists
```

### Invalid File Path

**Error:**
```
Error: task 5 (User API): invalid cross-file reference path: absolute paths not allowed
```

**Fix:**
```yaml
# Before (incorrect)
depends_on:
  - file: "/absolute/path/plan-01.yaml"
    task: 2

# After (correct)
depends_on:
  - file: "plan-01.yaml"
    task: 2
```

### Invalid Task Number

**Error:**
```
Error: task 5 (User API): cross-file reference has invalid task number "2@invalid"
Valid task numbers contain only alphanumeric characters, hyphens, and underscores
```

**Fix:**
```yaml
# Before (incorrect)
depends_on:
  - file: "plan-01.yaml"
    task: 2@invalid

# After (correct)
depends_on:
  - file: "plan-01.yaml"
    task: 2
```

---

## Best Practices

**1. File Organization**
```
plans/
  plan-01-foundation.yaml
  plan-02-features.yaml
  plan-03-integration.yaml
```

**2. Dependency Documentation**
```yaml
- task_number: 5
  name: "Feature Task"
  depends_on:
    - file: "plan-01-foundation.yaml"
      task: 2      # Requires database setup from foundation
    - 3            # Requires configuration from same file
  description: |
    Build feature that uses database from Task 2 (foundation)
    and configuration from Task 3 (local).
```

**3. Testing**
```bash
# Validate all plans
conductor validate plan-*.yaml

# Dry run to see execution order
conductor run plan-*.yaml --dry-run

# Verify cross-file resolution
conductor validate plan-01-foundation.yaml plan-02-features.yaml --verbose
```

---

## Backward Compatibility

Existing plans continue to work without any changes:

```yaml
# This format is still fully supported
depends_on: [1, 2, 3]

# Can now also use mixed format
depends_on:
  - 1
  - 2
  - file: "other-plan.yaml"
    task: 3
```

---

## Summary

Cross-file dependencies enable:
- Splitting large plans into logical files
- Clear organization by feature/layer
- Parallel development across teams
- Reusable foundational plans
- Better code review and testing workflows
