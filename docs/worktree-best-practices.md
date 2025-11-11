# Worktree Group Best Practices

**Last Updated**: 2025-11-11
**Status**: ✅ Phase 2A Feature

Worktree groups provide a mechanism to organize and control task execution in Conductor split plans. This guide explains best practices for defining, using, and managing worktree groups.

---

## Quick Reference

| Concept | Use Case | Example |
|---------|----------|---------|
| **ExecutionModel: sequential** | State-dependent tasks | Database migrations, setup |
| **ExecutionModel: parallel** | Independent tasks | API endpoints, services |
| **Isolation: strong** | No resource sharing | Infrastructure, database |
| **Isolation: weak** | Can share resources | API implementations |
| **Isolation: none** | Maximum parallelization | Utility tasks, tests |

---

## Fundamental Concepts

### ExecutionModel

Defines how tasks in a group should be executed:

**Sequential**: Tasks execute one after another
```yaml
execution_model: sequential
```
- Task 1 completes → Task 2 starts → Task 3 starts
- Guarantees ordering within group
- Best for state-dependent operations

**Parallel**: Tasks execute concurrently (when dependencies allow)
```yaml
execution_model: parallel
```
- Task 1, 2, 3 can start together
- Limited by task dependencies
- Best for independent operations

### Isolation

Defines whether tasks can share resources:

**Strong**: No shared resources
```yaml
isolation: strong
```
- Each task has independent environment
- May require separate worktrees/directories
- Best for infrastructure and ops tasks

**Weak**: Can share some resources
```yaml
isolation: weak
```
- Tasks can use shared database, cache
- Tasks can modify shared state
- Best for application features

**None**: Maximum resource sharing
```yaml
isolation: none
```
- Default, no isolation enforced
- Tasks can freely interact
- Best for utility and helper tasks

---

## Group Definition Patterns

### Pattern 1: Infrastructure Groups

```yaml
worktree_groups:
  - group_id: database
    description: Database schema and migrations
    execution_model: sequential
    isolation: strong
    rationale: |
      Database migrations must execute in order. Each migration
      builds on the previous state. Strong isolation prevents
      concurrent schema modifications.

  - group_id: cache-layer
    description: Cache infrastructure setup
    execution_model: sequential
    isolation: strong
    depends_on: database
    rationale: |
      Cache layer depends on database being ready.
      Sequential execution ensures proper initialization.
```

**Key Points:**
- `sequential` ensures ordering
- `strong` isolation prevents resource conflicts
- Clear dependencies between groups
- Detailed rationale explains why

### Pattern 2: Feature Groups

```yaml
worktree_groups:
  - group_id: user-service
    description: User management API and logic
    execution_model: parallel
    isolation: weak
    depends_on: database
    rationale: |
      User endpoints (get, create, update, delete) are independent
      and can be implemented in parallel. Weak isolation allows
      shared database access (already configured and safe).

  - group_id: product-service
    description: Product catalog API and logic
    execution_model: parallel
    isolation: weak
    depends_on: database
    rationale: |
      Product endpoints are independent. Weak isolation enables
      efficient shared resource usage.

  - group_id: order-service
    description: Order processing and fulfillment
    execution_model: parallel
    isolation: weak
    depends_on: user-service, product-service
    rationale: |
      Orders require both users and products to exist.
      Parallel execution optimizes order processing implementations.
```

**Key Points:**
- `parallel` allows concurrent feature development
- `weak` isolation is appropriate for API features
- Dependencies between groups clearly specified
- Parallelization improves team velocity

### Pattern 3: Testing Groups

```yaml
worktree_groups:
  - group_id: unit-tests
    description: Unit test suite
    execution_model: parallel
    isolation: none
    depends_on: []
    rationale: |
      Unit tests are independent and can run in parallel.
      No isolation needed (no state mutation).

  - group_id: integration-tests
    description: Integration test suite
    execution_model: sequential
    isolation: strong
    depends_on: unit-tests
    rationale: |
      Integration tests may conflict if run simultaneously.
      Strong isolation and sequential execution ensure
      test isolation and reliable results.

  - group_id: e2e-tests
    description: End-to-end testing
    execution_model: sequential
    isolation: strong
    depends_on: integration-tests
    rationale: |
      E2E tests require full environment setup.
      Sequential execution prevents interference.
```

**Key Points:**
- Different test types have different parallelization strategies
- `unit-tests` are highly parallelizable
- `integration-tests` require careful isolation
- `e2e-tests` require sequential execution

---

## Task Assignment Patterns

### Pattern 1: Single Task per Group

Assign individual critical tasks to their own group:

```markdown
## Task 1: Initialize Database Schema
**WorktreeGroup**: database-schema

Create base database schema.

## Task 2: Create Initial Users
**WorktreeGroup**: database-init-data
**Depends on**: Task 1

Insert seed data.
```

**When to Use:**
- Task is critical and shouldn't be parallelized
- Task has strict ordering requirements
- Task conflicts with other operations

### Pattern 2: Related Tasks in Group

Group related tasks that naturally belong together:

```markdown
## Task 4: Implement GET /users
**WorktreeGroup**: user-api

Fetch users endpoint.

## Task 5: Implement POST /users
**WorktreeGroup**: user-api
**Depends on**: Task 3

Create user endpoint.

## Task 6: Implement PUT /users/:id
**WorktreeGroup**: user-api
**Depends on**: Task 3

Update user endpoint.

## Task 7: Implement DELETE /users/:id
**WorktreeGroup**: user-api
**Depends on**: Task 3

Delete user endpoint.
```

**Advantages:**
- Logically grouped related functionality
- Can be developed by single team member
- Clear scope and responsibility
- Can be parallelized internally (when dependencies allow)

### Pattern 3: Zero Dependencies Within Group

Keep task dependencies outside the group:

```markdown
## Task 10: Validate User Input
**WorktreeGroup**: input-validation
**Depends on**: Task 3 (auth service)

Implement input validation middleware.

## Task 11: Sanitize User Data
**WorktreeGroup**: input-validation
**Depends on**: Task 3 (auth service)

Implement data sanitization.

## Task 12: Log All Requests
**WorktreeGroup**: input-validation
**Depends on**: Task 3 (auth service)

Add request logging.
```

**Benefits:**
- All tasks have same external dependencies
- Clear separation of concerns
- Easy to understand group purpose
- Can be fully parallelized

---

## Practical Examples

### Example 1: E-Commerce Backend

```yaml
worktree_groups:
  # Foundation Layer
  - group_id: foundation
    description: Database and basic infrastructure
    execution_model: sequential
    isolation: strong
    rationale: Foundation must be established first

  # Service Layer
  - group_id: catalog-service
    description: Product catalog functionality
    execution_model: parallel
    isolation: weak
    depends_on: foundation
    rationale: Catalog features are independent

  - group_id: user-service
    description: User management functionality
    execution_model: parallel
    isolation: weak
    depends_on: foundation
    rationale: User features are independent

  - group_id: order-service
    description: Order processing
    execution_model: parallel
    isolation: weak
    depends_on: catalog-service, user-service
    rationale: Orders need both catalog and users

  # Testing Layer
  - group_id: tests
    description: Comprehensive testing
    execution_model: sequential
    isolation: strong
    depends_on: order-service
    rationale: Tests run after all services complete

  # Deployment Layer
  - group_id: deployment
    description: Containerization and deployment
    execution_model: sequential
    isolation: strong
    depends_on: tests
    rationale: Deploy after all tests pass
```

**Execution Flow:**
1. **foundation** (sequential) → All tasks must complete
2. **catalog-service**, **user-service** (parallel) → Run together
3. **order-service** (parallel) → After 2 and 3
4. **tests** (sequential) → After all services
5. **deployment** (sequential) → Final stage

### Example 2: Microservices Platform

```yaml
worktree_groups:
  - group_id: shared-infra
    description: Shared infrastructure (K8s, networking)
    execution_model: sequential
    isolation: strong
    rationale: Infrastructure must be set up first and serially

  - group_id: auth-service
    description: Authentication microservice
    execution_model: parallel
    isolation: weak
    depends_on: shared-infra
    rationale: Auth endpoints can be built independently

  - group_id: api-gateway
    description: API gateway service
    execution_model: parallel
    isolation: weak
    depends_on: shared-infra
    rationale: Gateway components are independent

  - group_id: data-service
    description: Data and analytics service
    execution_model: parallel
    isolation: weak
    depends_on: shared-infra
    rationale: Data processing is independent

  - group_id: integration
    description: Service integration and testing
    execution_model: sequential
    isolation: strong
    depends_on: auth-service, api-gateway, data-service
    rationale: Integration tests must verify all services together

  - group_id: deployment
    description: Deployment to production
    execution_model: sequential
    isolation: strong
    depends_on: integration
    rationale: Deployment is final and must be serial
```

### Example 3: Data Platform

```yaml
worktree_groups:
  # Data Layer
  - group_id: data-infrastructure
    description: Databases, data warehouses, data lakes
    execution_model: sequential
    isolation: strong
    rationale: Data infrastructure must be initialized in order

  # ETL/ELT Layer
  - group_id: data-ingestion
    description: Data ingestion pipelines
    execution_model: parallel
    isolation: weak
    depends_on: data-infrastructure
    rationale: Independent data sources can be ingested in parallel

  # Processing Layer
  - group_id: data-processing
    description: Data transformation and processing
    execution_model: parallel
    isolation: weak
    depends_on: data-ingestion
    rationale: Transformations are independent

  # Analytics Layer
  - group_id: analytics
    description: Analytics and reporting
    execution_model: parallel
    isolation: weak
    depends_on: data-processing
    rationale: Reports and dashboards are independent

  # ML Layer
  - group_id: machine-learning
    description: ML model development
    execution_model: parallel
    isolation: weak
    depends_on: data-processing
    rationale: Models can be developed independently

  # Deployment Layer
  - group_id: deployment
    description: Deploy to production
    execution_model: sequential
    isolation: strong
    depends_on: analytics, machine-learning
    rationale: Deployment must be coordinated and serial
```

---

## Decision Matrix

**Use this matrix to decide group configuration:**

```
QUESTION 1: Can tasks run simultaneously?
  ├─ NO (state-dependent)
  │  └─ execution_model: sequential
  └─ YES (independent)
     └─ execution_model: parallel

QUESTION 2: Will tasks share resources?
  ├─ YES, safely (shared DB, cache)
  │  └─ isolation: weak
  ├─ NO (separate environments)
  │  └─ isolation: strong
  └─ NONE (utility tasks)
     └─ isolation: none

QUESTION 3: Does group depend on others?
  ├─ YES → List as depends_on
  └─ NO → depends_on: []
```

### Decision Examples

**Database Migration Tasks**
- Can tasks run simultaneously? **NO** → sequential
- Will they share resources? **YES** → weak
- Decision: `sequential + weak`

**Parallel API Endpoints**
- Can tasks run simultaneously? **YES** → parallel
- Will they share resources? **YES** → weak
- Decision: `parallel + weak`

**Infrastructure Setup**
- Can tasks run simultaneously? **NO** → sequential
- Will they share resources? **NO** → strong
- Decision: `sequential + strong`

**Utility/Helper Tasks**
- Can tasks run simultaneously? **YES** → parallel
- Will they share resources? **OPTIONAL** → none
- Decision: `parallel + none`

---

## Anti-Patterns to Avoid

### ❌ Anti-Pattern 1: Over-Grouping

**Bad:**
```yaml
worktree_groups:
  - group_id: task-1
    # Just one task
  - group_id: task-2
    # Just one task
  - group_id: task-3
    # Just one task
  # ... etc
```

**Why it's bad:**
- Defeats the purpose of groups
- Creates clutter
- No real organization

**Good:**
```yaml
worktree_groups:
  - group_id: api-endpoints
    description: All API endpoints
    # 5-10 related tasks
```

### ❌ Anti-Pattern 2: Circular Dependencies

**Bad:**
```yaml
worktree_groups:
  - group_id: service-a
    depends_on: service-b

  - group_id: service-b
    depends_on: service-a
```

**Why it's bad:**
- Creates circular dependency
- Neither can execute
- Plan validation will fail

**Good:**
```yaml
worktree_groups:
  - group_id: shared-infra
    depends_on: []

  - group_id: service-a
    depends_on: shared-infra

  - group_id: service-b
    depends_on: shared-infra
```

### ❌ Anti-Pattern 3: Mixed Models in Group

**Bad:**
```markdown
## Task 1: Sequential DB Migration
**WorktreeGroup**: mixed-group

Migrate schema.

## Task 2: Parallel API Endpoint
**WorktreeGroup**: mixed-group

Create API endpoint.
```

With `execution_model: parallel`, Task 1 might run before Task 2 completes, breaking dependencies.

**Good:**
```yaml
worktree_groups:
  - group_id: database
    execution_model: sequential

  - group_id: api
    execution_model: parallel
    depends_on: database
```

### ❌ Anti-Pattern 4: Inconsistent Isolation

**Bad:**
```markdown
## Task 1: Modify Schema
**WorktreeGroup**: db-ops
**Isolation**: strong

## Task 2: Add Data
**WorktreeGroup**: db-ops
**Isolation**: weak
```

**Why it's bad:**
- Inconsistent isolation within group
- Confusing for team
- May cause unexpected conflicts

**Good:**
```yaml
worktree_groups:
  - group_id: db-schema
    isolation: strong

  - group_id: db-data
    isolation: strong
    depends_on: db-schema
```

---

## Checklist for Group Definition

When defining worktree groups, verify:

- ✅ **Clear Purpose**: Group has single, clear purpose
- ✅ **Consistent Model**: All tasks use same execution model
- ✅ **Consistent Isolation**: All tasks have same isolation level
- ✅ **Dependencies**: All `depends_on` references exist
- ✅ **No Cycles**: No circular dependencies between groups
- ✅ **Rationale**: Each group has clear rationale
- ✅ **Task Assignments**: All tasks assigned to appropriate groups
- ✅ **Testing**: Validated before execution

---

## Team Collaboration with Groups

### Group Ownership

```yaml
worktree_groups:
  - group_id: frontend
    description: Frontend UI components
    owner: "Frontend Team"
    execution_model: parallel

  - group_id: backend-api
    description: Backend API services
    owner: "Backend Team"
    execution_model: parallel

  - group_id: infrastructure
    description: Infrastructure and DevOps
    owner: "DevOps Team"
    execution_model: sequential
```

### Parallel Development

With well-defined groups and clear dependencies, teams can work in parallel:

```
Time →
Frontend Team    ███████████
Backend Team                ████████████
DevOps Team                              ████
```

### Merging Plans

When teams complete their groups, merge plans:

```bash
# Team A creates frontend.md
# Team B creates backend.md
# DevOps creates infrastructure.md

# Merge and execute together
conductor run infrastructure.md frontend.md backend.md
```

---

## Validation and Testing

### Validate Group Definitions

```bash
# Check for circular dependencies
conductor validate --verbose *.md

# Shows group dependencies
# Verifies all task group assignments
# Detects group reference errors
```

### Test Group Execution

```bash
# Dry-run to verify execution order
conductor run *.md --dry-run --verbose

# Check parallel execution potential
# Verify group isolation assumptions
# Review execution timeline
```

### Monitor Execution

```bash
# Execute with verbose logging
conductor run *.md --verbose

# Logs show group execution
# Tracks task completion per group
# Reports any group-related issues
```

---

## Migration Guide

### Migrating from Single Plan to Split with Groups

**Step 1: Identify Natural Groups**
```
Existing 20-task plan:
  - Tasks 1-3: Database setup
  - Tasks 4-8: API features
  - Tasks 9-15: Testing
  - Tasks 16-20: Deployment
```

**Step 2: Define Group Structure**
```yaml
worktree_groups:
  - group_id: database
    execution_model: sequential

  - group_id: api
    execution_model: parallel
    depends_on: database

  - group_id: testing
    execution_model: sequential
    depends_on: api

  - group_id: deployment
    execution_model: sequential
    depends_on: testing
```

**Step 3: Split into Files**
- `database-setup.md` → database group tasks
- `api-features.md` → api group tasks
- `testing.md` → testing group tasks
- `deployment.md` → deployment group tasks

**Step 4: Update Task Assignments**
```markdown
## Task 1: Initialize Schema
**WorktreeGroup**: database
```

**Step 5: Validate and Execute**
```bash
conductor validate *.md
conductor run *.md --dry-run --verbose
conductor run *.md
```

---

## References

- [Phase 2A Guide](./phase-2a-guide.md) - Complete Phase 2A documentation
- [Plan Format Guide](./plan-format.md) - Plan format specifications
- [CLAUDE.md](../CLAUDE.md) - Project architecture
