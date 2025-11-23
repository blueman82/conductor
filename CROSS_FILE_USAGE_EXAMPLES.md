# Cross-File Dependency Usage Examples

This document provides practical examples of using cross-file dependencies in Conductor plans.

## Basic Example: Two-File Plan

### File 1: setup.yaml
```yaml
conductor:
  default_agent: golang-pro

plan:
  metadata:
    feature_name: "Infrastructure Setup"
  tasks:
    - task_number: 1
      name: "Initialize Database"
      files:
        - internal/db/schema.sql
        - internal/db/migrations.go
      depends_on: []
      estimated_time: "30m"
      description: "Create database schema and migration infrastructure"

    - task_number: 2
      name: "Setup Connection Pool"
      files:
        - internal/db/pool.go
      depends_on: [1]
      estimated_time: "20m"
      description: "Configure database connection pooling"
```

### File 2: features.yaml
```yaml
conductor:
  default_agent: golang-pro

plan:
  metadata:
    feature_name: "Application Features"
  tasks:
    - task_number: 3
      name: "Implement User Model"
      files:
        - internal/models/user.go
        - internal/models/user_test.go
      depends_on:
        - file: "setup.yaml"
          task: 2  # Depends on connection pool setup
      estimated_time: "1h"
      description: "Create user model with database operations"

    - task_number: 4
      name: "Add Authentication"
      files:
        - internal/auth/jwt.go
        - internal/auth/middleware.go
      depends_on:
        - 3  # Local dependency
        - file: "setup.yaml"
          task: 1  # Also verify base DB setup
      estimated_time: "1h30m"
      description: "Implement JWT authentication with user model"
```

**Execution Order**:
1. setup.yaml task 1 (Initialize Database)
2. setup.yaml task 2 (Setup Connection Pool) - depends on 1
3. features.yaml task 3 (User Model) - depends on setup.yaml task 2
4. features.yaml task 4 (Authentication) - depends on task 3 and setup.yaml task 1

## Microservices Example: Three Services

### auth-service-plan.yaml
```yaml
plan:
  metadata:
    feature_name: "Authentication Service"
  tasks:
    - task_number: 1
      name: "Implement JWT Provider"
      files: [internal/jwt/provider.go]
      depends_on: []
      estimated_time: "1h"
      agent: golang-pro

    - task_number: 2
      name: "Add OAuth2 Support"
      files: [internal/oauth/handler.go]
      depends_on: [1]
      estimated_time: "1h30m"
      agent: golang-pro
```

### api-service-plan.yaml
```yaml
plan:
  metadata:
    feature_name: "API Service"
  tasks:
    - task_number: 1
      name: "Setup REST Framework"
      files: [internal/api/server.go]
      depends_on:
        - file: "auth-service-plan.yaml"
          task: 2  # Need OAuth2 from auth service
      estimated_time: "1h"

    - task_number: 2
      name: "Implement API Routes"
      files: [internal/api/routes.go]
      depends_on:
        - 1  # Local dependency
        - file: "auth-service-plan.yaml"
          task: 2  # Need OAuth2
      estimated_time: "1h30m"
```

### gateway-plan.yaml
```yaml
plan:
  metadata:
    feature_name: "API Gateway"
  tasks:
    - task_number: 1
      name: "Setup Gateway"
      files: [internal/gateway/server.go]
      depends_on:
        - file: "api-service-plan.yaml"
          task: 2
        - file: "auth-service-plan.yaml"
          task: 2
      estimated_time: "1h"
      description: "Wire auth and API services into gateway"
```

**Execution Flow**:
```
Auth Service:
  - JWT Provider
  - OAuth2 Support

API Service (depends on Auth):
  - REST Framework
  - Routes

Gateway (depends on Auth + API):
  - Gateway Setup
```

## Complex Example: Diamond Pattern

### foundation.yaml
```yaml
plan:
  metadata:
    feature_name: "Foundation"
  tasks:
    - task_number: 1
      name: "Database Setup"
      files: [internal/db/init.go]
      depends_on: []
      estimated_time: "30m"
```

### service-a.yaml
```yaml
plan:
  metadata:
    feature_name: "Service A"
  tasks:
    - task_number: 1
      name: "Service A Implementation"
      files: [internal/service_a/handler.go]
      depends_on:
        - file: "foundation.yaml"
          task: 1
      estimated_time: "1h"
```

### service-b.yaml
```yaml
plan:
  metadata:
    feature_name: "Service B"
  tasks:
    - task_number: 1
      name: "Service B Implementation"
      files: [internal/service_b/handler.go]
      depends_on:
        - file: "foundation.yaml"
          task: 1
      estimated_time: "1h"
```

### integration.yaml
```yaml
plan:
  metadata:
    feature_name: "Integration"
  tasks:
    - task_number: 1
      name: "Wire Services Together"
      files: [cmd/main.go]
      depends_on:
        - file: "service-a.yaml"
          task: 1
        - file: "service-b.yaml"
          task: 1
      estimated_time: "30m"
```

**Diamond Pattern**:
```
        Foundation
           / \
          /   \
    Service A  Service B
          \   /
           \ /
       Integration
```

## Markdown Examples

### setup.md
```markdown
## Task 1: Setup Cache

**File(s)**: `internal/cache/redis.go`
**Depends on**: None
**Estimated time**: 45m
**Agent**: golang-pro

Configure Redis cache.

## Task 2: Setup Database

**File(s)**: `internal/db/postgres.go`
**Depends on**: Task 1
**Estimated time**: 1h
**Agent**: golang-pro

Configure PostgreSQL database.
```

### features.md
```markdown
## Task 3: Implement User Service

**File(s)**: `internal/user/service.go`
**Depends on**: file:setup.md:task:2
**Estimated time**: 2h
**Agent**: golang-pro

Create user service using cache and database.

## Task 4: Add User API

**File(s)**: `internal/api/users.go`
**Depends on**: Task 3, file:setup.md/task:1
**Estimated time**: 1h30m
**Agent**: golang-pro

Create API endpoints for user service.
```

## Running Multi-File Plans

### Command Line
```bash
# Validate all files together
conductor validate setup.yaml features.yaml

# Run with specific ordering
conductor run setup.yaml features.yaml --max-concurrency 2

# Dry run first
conductor run setup.yaml features.yaml --dry-run

# With specific timeout
conductor run auth-plan.yaml api-plan.yaml gateway-plan.yaml --timeout 30m
```

### Configuration File

Create `.conductor/config.yaml`:
```yaml
max_concurrency: 3
timeout: 1h
skip_completed: true
retry_failed: true

quality_control:
  enabled: true
  review_agent: code-reviewer

learning:
  enabled: true
  enhance_prompts: true
  swap_during_retries: true
```

## File Organization Best Practices

### For Microservices
```
plans/
├── auth-service-plan.yaml
├── api-service-plan.yaml
├── database-plan.yaml
└── deployment-plan.yaml
```

### For Features
```
plans/
├── foundation/
│   ├── 01-database.yaml
│   └── 02-cache.yaml
├── features/
│   ├── 01-auth.yaml
│   ├── 02-users.yaml
│   └── 03-api.yaml
└── integration/
    └── 01-gateway.yaml
```

### For Teams
```
plans/
├── backend/
│   ├── foundation.yaml
│   └── features.yaml
├── frontend/
│   └── components.yaml
└── integration/
    └── full-stack.yaml
```

## Dependency Validation

Conductor automatically validates:

1. **File References**: All referenced files must exist
2. **Task Numbers**: All referenced task numbers must exist
3. **Circular Dependencies**: Detects and reports cycles
4. **Type Compatibility**: Validates task IDs are int/float/string

Example error output:
```
Error: File 'auth-plan.yaml' not found (referenced by api-plan.yaml:task:1)
Error: Task 5 not found in auth-plan.yaml (referenced by api-plan.yaml:task:2)
Error: Circular dependency detected: plan-a.yaml:task:1 → plan-b.yaml:task:2 → plan-a.yaml:task:1
```

## Advanced Patterns

### Shared Infrastructure

```yaml
# infrastructure.yaml
tasks:
  - task_number: 1
    name: "Setup VPC"
    depends_on: []

  - task_number: 2
    name: "Setup Load Balancer"
    depends_on: [1]

# service-1.yaml
tasks:
  - task_number: 1
    name: "Deploy Service 1"
    depends_on:
      - file: "infrastructure.yaml"
        task: 2

# service-2.yaml
tasks:
  - task_number: 1
    name: "Deploy Service 2"
    depends_on:
      - file: "infrastructure.yaml"
        task: 2
```

### Progressive Integration

```yaml
# phase-1-foundation.yaml
tasks:
  - task_number: 1
    name: "Core Models"
    depends_on: []

# phase-2-services.yaml
tasks:
  - task_number: 1
    name: "Service A"
    depends_on:
      - file: "phase-1-foundation.yaml"
        task: 1

  - task_number: 2
    name: "Service B"
    depends_on:
      - file: "phase-1-foundation.yaml"
        task: 1

# phase-3-integration.yaml
tasks:
  - task_number: 1
    name: "Integration"
    depends_on:
      - file: "phase-2-services.yaml"
        task: 1
      - file: "phase-2-services.yaml"
        task: 2
```

## Tips and Tricks

### 1. Use Descriptive File Names
```
Good:  plan-01-database.yaml, plan-02-auth.yaml
Bad:   plan1.yaml, plan2.yaml
```

### 2. Group Related Tasks
```yaml
# Keep related tasks in same file when possible
tasks:
  - task_number: 1
    name: "User Model"
    depends_on: [database]

  - task_number: 2
    name: "User Repository"
    depends_on: [1]

  - task_number: 3
    name: "User Service"
    depends_on: [2]
```

### 3. Document Dependencies Clearly
```markdown
## Task 3: Implement API

**Depends on**: file:setup-database.yaml:task:2

This task requires the database connection pool from the setup phase
because it needs to execute database queries through the pool.
```

### 4. Use Consistent Task Numbering
```
Avoid:  Task 1 in file A, Task 5 in file B
Better: Task 1-2 in file A, Task 3-4 in file B
```

### 5. Validate Before Running
```bash
# Always validate first
conductor validate plan-1.yaml plan-2.yaml plan-3.yaml

# Then run with dry-run
conductor run plan-1.yaml plan-2.yaml plan-3.yaml --dry-run

# Finally execute
conductor run plan-1.yaml plan-2.yaml plan-3.yaml
```

## Troubleshooting

### Issue: "File not found" error
**Solution**: Verify file paths are correct and files exist in the directory

### Issue: Circular dependency detected
**Solution**: Review dependency chain and break the cycle

### Issue: Task executes out of order
**Solution**: Check dependencies are declared correctly, use `--dry-run` to preview

### Issue: Cross-file dependency not recognized
**Solution**: Verify format: `file:name.yaml:task:N` or `{file: "name.yaml", task: N}`

## Examples Repository

See `internal/parser/testdata/` for complete working examples:
- `cross-file-simple.yaml` / `cross-file-simple.md`
- `cross-file-mixed.yaml` / `cross-file-mixed.md`
- `cross-file-fixtures/` for complex patterns

## Related Documentation

- See `CROSS_FILE_DEPS_IMPLEMENTATION.md` for technical details
- See `PARSER_CHANGES_SUMMARY.md` for code changes
- See CLAUDE.md for architecture overview
