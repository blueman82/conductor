# Phase 2A: Multi-File Plans & Objective Plan Splitting Guide

**Latest Update**: 2025-11-11
**Status**: ✅ Fully Implemented and Tested
**Test Coverage**: 37 integration tests, 100% backward compatible

---

## Overview

Phase 2A extends Conductor v1 with the ability to split large implementation plans across multiple files while maintaining full dependency tracking and execution orchestration.

**Key Benefits:**
- 📦 Break monolithic plans into focused, manageable files
- 🔗 Maintain full dependency graphs across files
- 👥 Enable team collaboration on different plan segments
- 📊 Better plan organization and version control
- ⚡ Support partial plan re-execution on split plans

---

## Core Concepts

### 1. Multi-File Plan Loading

Conductor automatically loads, validates, and merges multiple plan files:

```bash
# Load and execute multiple plan files
conductor run phase1-setup.md phase2-implementation.md phase3-testing.md

# Validate split plans before execution
conductor validate setup.md features.md deployment.md
```

**Format Auto-Detection**: Each file's format is detected independently:
- File `setup.md` → Markdown parser
- File `features.yaml` → YAML parser
- File `deploy.yaml` → YAML parser
- Mixed formats work seamlessly

### 2. Objective Plan Splitting

Split a plan objectively by logical boundaries rather than arbitrary task counts:

**Good Split**:
```
frontend.md       → All UI/frontend tasks
backend.md        → Server/backend tasks
deployment.md     → Infrastructure/ops tasks
```

**Avoid**:
```
plan-part1.md     → Tasks 1-7
plan-part2.md     → Tasks 8-14  (arbitrary split)
```

### 3. Worktree Groups

Group related tasks and define execution rules:

```markdown
## Task 1: Database Setup
**WorktreeGroup**: infrastructure

Initialize and seed database.

## Task 2: API Server
**WorktreeGroup**: backend-core

Start API server.

## Task 3: User Service
**WorktreeGroup**: backend-features

Build user management API.
```

**Group Definition** (in YAML plan):
```yaml
worktree_groups:
  - group_id: infrastructure
    description: Infrastructure and database setup
    execution_model: sequential
    isolation: strong
    rationale: State-dependent, must execute in order

  - group_id: backend-features
    description: API feature implementations
    execution_model: parallel
    isolation: weak
    rationale: Independent features can execute in parallel
```

### 4. FileToTaskMap

Conductor automatically tracks which file each task originated from:

```
FileToTaskMap:
  "setup.md" → [1, 2, 3, 4]
  "features.md" → [5, 6, 7, 8]
  "deploy.md" → [9, 10]
```

**Benefits:**
- Resume on specific file segments
- Partial re-execution support
- Better logging and tracking
- File-aware task reporting

---

## Usage Examples

### Example 1: Basic Split Plan

**setup.md**:
```markdown
# Backend Setup Plan

## Task 1: Initialize Database
**Files**: infrastructure/database.sql
**Depends on**: None

Create PostgreSQL database and schema.

## Task 2: Setup Redis Cache
**Files**: infrastructure/cache-config.yaml
**Depends on**: Task 1

Configure Redis for caching.
```

**features.md**:
```markdown
# Backend Features Plan

## Task 3: Implement Auth Service
**Files**: internal/auth/auth.go
**Depends on**: Task 1, Task 2

Add JWT authentication.

## Task 4: Create User API
**Files**: internal/api/users.go
**Depends on**: Task 3

Implement user CRUD endpoints.
```

**Execution**:
```bash
# Validate all files together
conductor validate setup.md features.md

# Dry-run to test without execution
conductor run setup.md features.md --dry-run

# Execute with verbose output
conductor run setup.md features.md --verbose

# Resume with skip-completed
conductor run setup.md features.md --skip-completed
```

### Example 2: Microservices Architecture

**auth-service.md** (6 tasks):
```markdown
# Authentication Service

## Task 1: Database Setup
**WorktreeGroup**: auth-infra

Initialize auth database.

## Task 2: JWT Implementation
**WorktreeGroup**: auth-core
**Depends on**: Task 1

Implement JWT token handling.

## Task 3: OAuth Integration
**WorktreeGroup**: auth-features
**Depends on**: Task 2

Add OAuth2 provider support.
```

**api-service.md** (8 tasks):
```markdown
# Main API Service

## Task 4: API Framework
**WorktreeGroup**: api-core

Set up HTTP server.

## Task 5: User Endpoints
**WorktreeGroup**: api-features
**Depends on**: Task 4

Create user management endpoints.

## Task 6: Product Endpoints
**WorktreeGroup**: api-features
**Depends on**: Task 4

Create product catalog endpoints.
```

**deployment.md** (5 tasks):
```markdown
# Deployment & Infrastructure

## Task 7: Terraform Setup
**WorktreeGroup**: deployment

Configure infrastructure.

## Task 8: Docker Build
**WorktreeGroup**: deployment
**Depends on**: Task 7

Build container images.

## Task 9: Deploy to K8s
**WorktreeGroup**: deployment
**Depends on**: Task 8

Deploy to Kubernetes cluster.
```

**Execution**:
```bash
# Execute all services together
conductor run auth-service.md api-service.md deployment.md

# With max concurrency per group
conductor run *.md --max-concurrency 4

# Monitor with file-aware logging
conductor run *.md --verbose --log-dir ./logs
```

### Example 3: Phased Delivery

**phase1-foundation.md**:
```markdown
# Phase 1: Foundation (Setup & Core)

## Task 1: Project Structure
**Depends on**: None

Initialize project layout.

## Task 2: Build Pipeline
**Depends on**: Task 1

Set up CI/CD.

## Task 3: Core Library
**Depends on**: Task 1

Create base utilities.
```

**phase2-features.md**:
```markdown
# Phase 2: Features (Implementation)

## Task 4: Feature A
**Depends on**: Task 3

Implement first feature.

## Task 5: Feature B
**Depends on**: Task 3

Implement second feature.

## Task 6: Feature C
**Depends on**: Task 4, Task 5

Integrate features.
```

**phase3-polish.md**:
```markdown
# Phase 3: Polish (Testing & Optimization)

## Task 7: Performance Optimization
**Depends on**: Task 6

Optimize critical paths.

## Task 8: Integration Testing
**Depends on**: Task 6

Test feature integration.

## Task 9: Release Preparation
**Depends on**: Task 7, Task 8

Prepare for release.
```

**Execution Strategy**:
```bash
# Phase 1: Foundation
conductor run phase1-foundation.md --verbose

# Phase 2: Features (depends on Phase 1)
conductor run phase2-features.md --skip-completed

# Phase 3: Polish (depends on Phase 1 & 2)
conductor run phase3-polish.md --skip-completed

# Or execute all at once with natural ordering
conductor run phase1-*.md phase2-*.md phase3-*.md
```

---

## Best Practices

### 📋 File Organization

1. **One Unit Per File**
   - Each file represents a cohesive feature/module/service
   - Aim for 5-20 tasks per file
   - Related tasks stay together

2. **Clear Naming**
   ```
   ✅ backend-setup.md          # Clear, specific
   ✅ frontend-features.md       # Clear, specific
   ✅ deployment-k8s.md          # Clear, specific

   ❌ part1.md, part2.md         # Vague
   ❌ tasks-1-5.md               # Arbitrary split
   ```

3. **File Order Matters**
   - List files in execution order when possible
   - Dependencies override file order, but order aids understanding
   - Reduce cross-file dependencies

### 🔗 Dependency Management

1. **Minimize Cross-File Dependencies**
   ```markdown
   ✅ Good: Task 3 (api.md) depends on Task 1,2 (db.md)
      One clear dependency boundary

   ❌ Avoid: Task 3 (a.md) → Task 5 (b.md) → Task 7 (c.md) → Task 4 (a.md)
      Tangled cross-file dependencies
   ```

2. **Clear Dependency Declarations**
   ```markdown
   ## Task 3: User API
   **Depends on**: Task 1, Task 2

   Create user endpoints (depends on DB setup and Cache setup).
   ```

3. **Validate Cross-File Links**
   ```bash
   # Always validate split plans together
   conductor validate setup.md features.md deploy.md
   ```

### 👥 Worktree Groups

1. **Group by Execution Needs**
   ```yaml
   worktree_groups:
     # Sequential: State-dependent tasks
     - group_id: database
       execution_model: sequential
       isolation: strong
       rationale: Database setup and migrations must execute in order

     # Parallel: Independent tasks
     - group_id: services
       execution_model: parallel
       isolation: weak
       rationale: Microservices can be built independently

     # Optional: Complex isolation
     - group_id: deployment
       execution_model: sequential
       isolation: strong
       rationale: Infrastructure changes require careful ordering
   ```

2. **Isolation Levels**
   - `none` - No isolation (default, allows any overlap)
   - `weak` - Weak isolation (can share resources)
   - `strong` - Strong isolation (no resource sharing, may require separate worktrees)

3. **Execution Models**
   - `parallel` - Run all tasks in group in parallel (when dependencies allow)
   - `sequential` - Run tasks in group one at a time

### 🧪 Testing Split Plans

1. **Validation First**
   ```bash
   # Validate all files together to catch cross-file issues
   conductor validate setup.md api.md frontend.md deploy.md
   ```

2. **Dry-Run Before Execution**
   ```bash
   # Test without actually executing tasks
   conductor run setup.md api.md --dry-run --verbose
   ```

3. **Check Dependency Graph**
   ```bash
   # Verbose output shows task execution order and dependencies
   conductor run *.md --verbose --dry-run
   ```

4. **Incremental Validation**
   ```bash
   # Validate and run incrementally
   conductor validate setup.md && conductor run setup.md
   conductor validate api.md && conductor run api.md --skip-completed
   ```

---

## Merging Strategy

Conductor uses these rules when merging multiple plan files:

1. **Task Number Preservation**
   - Task 1 from setup.md stays as Task 1
   - Task 1 from features.md becomes Task N (renumbered)
   - Automatic renumbering to maintain uniqueness

2. **Dependency Resolution**
   - Cross-file dependencies maintained
   - Cycle detection works across files
   - Dependency graph validated before execution

3. **Deduplication**
   - Duplicate task names not allowed (error if same task in multiple files)
   - File origins preserved for logging and resume

4. **WorktreeGroup Merging**
   - All groups from all files merged
   - Group IDs must be unique across files
   - Execution respects all group constraints

---

## Advanced Patterns

### Pattern 1: Optional Features

**core-plan.md** (required):
```markdown
## Task 1: Database
## Task 2: API Server
## Task 3: Auth
```

**optional-features.md** (optional):
```markdown
## Task 4: Analytics
**Depends on**: Task 2

Add analytics tracking.

## Task 5: Admin Panel
**Depends on**: Task 2, Task 3

Create admin interface.
```

**Execution**:
```bash
# Run core only
conductor run core-plan.md

# Run with optional features
conductor run core-plan.md optional-features.md

# Skip optional features if they fail
conductor run core-plan.md optional-features.md --retry-failed
```

### Pattern 2: Conditional Deployment

**app-plan.md** (always):
```markdown
## Task 1-5: Build application
```

**deploy-staging.md** (for staging):
```markdown
## Task 6: Deploy to Staging
**Depends on**: Task 5
```

**deploy-prod.md** (for production):
```markdown
## Task 7: Deploy to Production
**Depends on**: Task 5
```

**Execution**:
```bash
# Build only
conductor run app-plan.md

# Build and deploy to staging
conductor run app-plan.md deploy-staging.md

# Build and deploy to production
conductor run app-plan.md deploy-prod.md
```

### Pattern 3: Multi-Service Deployment

**shared-infra.md** (prerequisite):
```markdown
## Task 1-3: Shared infrastructure
```

**service-auth.md** (first service):
```markdown
## Task 4-6: Auth service
**Depends on**: Task 3
```

**service-api.md** (second service):
```markdown
## Task 7-10: API service
**Depends on**: Task 3
```

**service-frontend.md** (third service):
```markdown
## Task 11-13: Frontend
**Depends on**: Task 10 (API ready)
```

**Execution**:
```bash
# Build everything together
conductor run shared-infra.md service-auth.md service-api.md service-frontend.md

# Or in phases (if needed)
conductor run shared-infra.md
conductor run shared-infra.md service-auth.md service-api.md --skip-completed
conductor run shared-infra.md service-auth.md service-api.md service-frontend.md --skip-completed
```

---

## Troubleshooting

### Issue: Cross-File Dependencies Not Found

```bash
conductor validate setup.md features.md

# Error: Task 5 depends on Task 8, not found in setup.md
```

**Solution**: Ensure all task numbers are referenced correctly:
```bash
# Check task numbers
grep "^## Task" setup.md features.md

# Update references if needed
```

### Issue: WorktreeGroup Not Found

```bash
# Error: Task 3 references unknown group "backend-infra"
```

**Solution**: Define group in plan's YAML metadata:
```yaml
worktree_groups:
  - group_id: backend-infra
    description: Backend infrastructure
    execution_model: sequential
    isolation: strong
```

### Issue: Dependency Cycle Across Files

```bash
# Error: Circular dependency detected: Task 1 → Task 4 → Task 2 → Task 1
```

**Solution**: Refactor cross-file dependencies:
```markdown
# Move Task 2 to features.md to avoid cycle
# Ensure: setup.md → features.md (one-way dependency)
```

### Issue: File Format Mismatch

```bash
conductor run setup.md features.yaml deploy.yml
# Works! Auto-detects each file's format
```

---

## Migration from Single-File Plans

Splitting an existing plan is straightforward:

**Before** (single-plan.md with 15 tasks):
```markdown
## Task 1: Setup
## Task 2: Setup
## Task 3: Setup
## Task 4: Feature A
## Task 5: Feature A
## Task 6: Feature B
## Task 7: Feature B
## Task 8: Testing
# ... 15 tasks total
```

**After** (split into focused files):

setup.md (Tasks 1-3):
```markdown
## Task 1: Setup
## Task 2: Setup
## Task 3: Setup
```

features.md (Tasks 4-7, renumbered):
```markdown
## Task 4: Feature A
**Depends on**: Task 3

## Task 5: Feature A
**Depends on**: Task 4

## Task 6: Feature B
**Depends on**: Task 3

## Task 7: Feature B
**Depends on**: Task 6
```

testing.md (Tasks 8-15, renumbered):
```markdown
## Task 8: Testing
**Depends on**: Task 7
# ... rest of tests
```

**Execution** (unchanged):
```bash
# Just add all files
conductor run setup.md features.md testing.md
```

---

## Performance Considerations

### Large-Scale Plans

Tested and verified with:
- ✅ 10 plan files
- ✅ 20+ tasks per file
- ✅ Complex cross-file dependencies
- ✅ 37 integration tests pass

### Optimization Tips

1. **Keep Files Smaller**
   - 5-20 tasks per file (ideal)
   - Easier to understand and debug

2. **Minimize Cross-File Dependencies**
   - Each dependency adds coordination overhead
   - Batch related files together

3. **Use Worktree Groups**
   - Groups enable better parallelization
   - Strong isolation for independence

4. **Parallel Execution**
   ```bash
   # Increase max concurrency for better parallelization
   conductor run *.md --max-concurrency 5
   ```

---

## Reference

### CLI Commands

```bash
# Validate split plans
conductor validate file1.md file2.yaml file3.md

# Run split plans
conductor run file1.md file2.md file3.md [FLAGS]

# With common flags
conductor run *.md --max-concurrency 4 --verbose
conductor run *.md --dry-run
conductor run *.md --skip-completed --retry-failed
```

### Task Metadata

```markdown
## Task N: Task Name
**Files**: file1.go, file2.go
**Depends on**: Task X, Task Y
**Estimated time**: 30m
**Agent**: agent-name
**WorktreeGroup**: group-id

Task description and implementation notes.
```

### YAML Plan Format

```yaml
plan:
  metadata:
    name: "Plan Name"
    created: "2025-11-11"

  worktree_groups:
    - group_id: group1
      description: "Group description"
      execution_model: parallel
      isolation: weak

  tasks:
    - number: 1
      name: "Task 1"
      files: [file1.go]
      depends_on: []
      worktree_group: group1
```

---

## See Also

- [Plan Format Guide](./plan-format.md) - Detailed format specifications
- [Usage Guide](./usage.md) - CLI command reference
- [CLAUDE.md](../CLAUDE.md) - Project architecture and implementation details
