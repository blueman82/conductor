# Integration Tasks Architecture

Integration tasks wire together previously completed components. They require dual-level validation: component-level and cross-component integration verification.

## Component vs Integration Tasks

### Component Task

**Characteristics:**
- Single module/feature responsibility
- Focused `success_criteria`
- Minimal dependencies (0-2)
- Creates new functionality

**Example:**
```yaml
- task_number: 1
  name: "Implement JWT Validation Module"
  type: component  # Optional - default behavior
  files:
    - internal/auth/jwt.go
    - internal/auth/jwt_test.go
  depends_on: []
  success_criteria:
    - "ValidateJWT function implemented"
    - "Supports HS256 algorithm"
    - "Unit tests achieve 90% coverage"
```

### Integration Task

**Characteristics:**
- Wires multiple components together
- `type: integration` marker (REQUIRED)
- Both `success_criteria` AND `integration_criteria`
- Multiple dependencies (typically 3+)
- Modifies existing files to connect components

**Example:**
```yaml
- task_number: 5
  name: "Wire Auth Module to API Router"
  type: integration  # REQUIRED marker
  files:
    - internal/auth/jwt.go
    - internal/api/router.go
    - cmd/main.go
  depends_on: [1, 2, 3, 4]

  # Component-level validation
  success_criteria:
    - "Router accepts GET /api/users request"
    - "Auth middleware has correct signature"
    - "Error responses include HTTP status codes"

  # Cross-component validation
  integration_criteria:
    - "Auth middleware executes BEFORE route handlers"
    - "Unauthenticated requests receive 401 Unauthorized"
    - "Authenticated requests proceed to handler"
    - "Database connection from Task 2 used in handler"
    - "Transaction commits after response sent"

  description: |
    Wire the JWT auth module to the API router.

    Takes completed auth module and connects to API router,
    ensuring authentication middleware properly integrated
    before route handling.
```

## When to Generate Integration Tasks

### Yes - Generate Integration Task

**1. Multi-component plans (3+ independent modules):**
```
Task 1-3: Auth system components
Task 4-6: API router components
Task 7: INTEGRATION - Wire auth to router
```

**2. After component boundaries are clear:**
```
Group 1: Database client implementation
Group 2: Service layer implementation
Task N: INTEGRATION - Connect service to database
```

**3. Between dependent worktree groups:**
```
chain-1: Component A (Tasks 1-3)
chain-2: Component B (Tasks 4-6)
chain-3: INTEGRATION - Wire A to B (Task 7)
```

**4. Cross-cutting concerns:**
```
Tasks 1-5: Feature implementation
Task 6: INTEGRATION - Add logging to all components
Task 7: INTEGRATION - Add monitoring instrumentation
```

### No - Regular Component Task

**1. Single-component tasks:**
```
Task 1: Implement user model (single module)
```

**2. Wiring included in component:**
```
Task 1: Implement auth AND wire to router
(Not recommended - should split)
```

**3. Sequential feature-based tasks:**
```
Task 1: Setup
Task 2: Implement feature
Task 3: Add tests
(Greenfield, no integration needed)
```

**4. Explicit wiring in component description:**
```
Task 1: "Implement auth module and integrate with router"
(Already handles integration)
```

## Integration Task Structure

### Required Fields

```yaml
- task_number: N
  type: integration  # REQUIRED
  name: "Wire X to Y"
  agent: "fullstack-developer"  # Or appropriate integrator
  files:
    - "component/a/file.go"
    - "component/b/file.go"
    - "integration/point.go"
  depends_on: [1, 2, 3, ...]  # All components being wired

  # BOTH criteria types required
  success_criteria:
    - "Component-level checks"

  integration_criteria:
    - "Cross-component checks"

  description: |
    Integration task description
```

### Dual Criteria System

**success_criteria** - Component works standalone:
```yaml
success_criteria:
  - "Function returns correct value"
  - "Error handling implemented"
  - "Unit tests pass"
```

**integration_criteria** - Components work together:
```yaml
integration_criteria:
  - "Component A calls Component B"
  - "Data flows correctly end-to-end"
  - "Error propagation works"
  - "Resource lifecycle managed"
```

## Conductor's Automatic Prompt Enhancement

When conductor executes integration task, it AUTOMATICALLY enhances the prompt:

### Before Enhancement (Your Task Description)

```yaml
description: |
  Wire JWT auth module to API router.

  Integrate authentication middleware before route handling.
```

### After Enhancement (What Agent Receives)

```
# INTEGRATION TASK CONTEXT

Before implementing, you MUST read these dependency files:

## Dependency: Task 1 - JWT Auth Implementation
**Files to read**:
- internal/auth/jwt.go

**WHY YOU MUST READ THESE**:
You need to understand JWT Auth Implementation to properly integrate.
Read to see:
- Exported functions and signatures
- Data structures and types
- Error handling patterns
- Integration interfaces

## Dependency: Task 2 - API Router Setup
**Files to read**:
- internal/api/router.go

**WHY YOU MUST READ THESE**:
You need to understand API Router Setup to properly integrate.
Read to see:
- Exported functions and signatures
- Data structures and types
- Error handling patterns
- Integration interfaces

---

# YOUR INTEGRATION TASK

Wire JWT auth module to API router.

Integrate authentication middleware before route handling.
```

**You don't need to repeat dependency details** - conductor adds them automatically.

## Integration Task Patterns

### Pattern 1: Component Wiring

```yaml
- task_number: 8
  name: "Wire Database Layer to Repository"
  type: integration
  depends_on: [5, 6, 7]  # DB client, transactions, connection pool
  files:
    - internal/repository/user.go
    - internal/db/connection.go

  success_criteria:
    - "UserRepository interface implemented"
    - "CRUD operations defined"

  integration_criteria:
    - "Repository uses database client"
    - "Transactions properly scoped"
    - "Connection pooling utilized"
    - "Error handling propagates DB errors"

  description: |
    Connect database client to repository layer.
    Ensure transactions and connection pooling integrated.
```

### Pattern 2: Security Integration

```yaml
- task_number: 12
  name: "Integrate Auth Middleware into API Handlers"
  type: integration
  depends_on: [1, 3, 7, 9, 11]  # Auth, router, handlers
  files:
    - internal/api/middleware.go
    - internal/api/handlers.go

  success_criteria:
    - "Middleware registered in router"
    - "Protected routes identified"

  integration_criteria:
    - "Auth middleware executes before handlers"
    - "Token validation occurs pre-handler"
    - "Unauthenticated requests blocked"
    - "User context passed to handlers"

  description: |
    Add authentication middleware to protected endpoints.
    Ensure token validation and authorization checks.
```

### Pattern 3: Multi-Service Coordination

```yaml
- task_number: 20
  name: "Configure Service Mesh Communication"
  type: integration
  depends_on: [14, 15, 16, 17, 18, 19]  # All microservices
  files:
    - config/service-mesh.yaml
    - internal/discovery/resolver.go

  success_criteria:
    - "Service discovery configured"
    - "Communication endpoints defined"

  integration_criteria:
    - "All services discoverable"
    - "Inter-service calls routed correctly"
    - "Load balancing active"
    - "Circuit breakers configured"
    - "Distributed tracing enabled"

  description: |
    Set up service discovery and inter-service communication.
    Enable microservices to locate and communicate.
```

## Decision Tree

```
Is task wiring multiple components?
├─ NO (single module)
│  └─ Regular component task
└─ YES (connecting multiple components)
   ├─ Requires understanding completed implementations?
   │  └─ YES → Integration task (conductor adds context)
   └─ Truly independent (no dependencies)?
      └─ YES → Regular task
      └─ NO (has dependencies) → Integration task
```

## Quick Reference

| Aspect | Component Task | Integration Task |
|--------|----------------|------------------|
| **Scope** | Single feature/module | Wire multiple features |
| **Dependencies** | 0-2 | 3+ |
| **Type field** | (not set) | `"integration"` |
| **Criteria** | `success_criteria` only | Both criteria types |
| **Prompt** | None | Auto-enhanced |
| **Agent** | Feature-focused | Full-stack/architect |
| **Files** | Mostly new | Mix new + existing |
| **Example** | "Implement JWT auth" | "Wire auth to router" |

## Integration Criteria Examples

### Data Flow

```yaml
integration_criteria:
  - "Request data flows from handler to service"
  - "Service calls repository with correct params"
  - "Repository returns data to service"
  - "Service transforms and returns to handler"
```

### Sequencing

```yaml
integration_criteria:
  - "Auth middleware runs BEFORE route handler"
  - "Validation occurs BEFORE business logic"
  - "Transaction commits AFTER all updates"
  - "Response sent AFTER transaction commit"
```

### Error Propagation

```yaml
integration_criteria:
  - "Database errors propagate to service layer"
  - "Service layer wraps errors with context"
  - "Handler converts errors to HTTP status"
  - "Client receives appropriate error message"
```

### Resource Lifecycle

```yaml
integration_criteria:
  - "Database connection acquired at request start"
  - "Transaction opened before first query"
  - "Transaction committed before response"
  - "Connection released after response"
```

### State Management

```yaml
integration_criteria:
  - "User context created by auth middleware"
  - "Context passed to all downstream handlers"
  - "Handler accesses user ID from context"
  - "Audit log records user actions"
```

## Best Practices

### 1. Focus on Integration Points

**Good:**
```yaml
integration_criteria:
  - "Router imports auth package"
  - "auth.Middleware() registered in router.Use()"
  - "main.go initializes auth before router"
```

**Bad:**
```yaml
integration_criteria:
  - "Auth module works correctly"  # Component-level
  - "Router has correct endpoints"  # Component-level
```

### 2. Verify Visibility/Sequencing

**Good:**
```yaml
integration_criteria:
  - "Middleware executes BEFORE handler (verified via logs)"
  - "Transaction started BEFORE query (check context)"
```

**Bad:**
```yaml
integration_criteria:
  - "Middleware works"  # No sequencing check
```

### 3. Check Resource Lifecycle

**Good:**
```yaml
integration_criteria:
  - "Connection opened at handler start"
  - "Connection closed at handler exit"
  - "Panic recovery closes connection"
```

**Bad:**
```yaml
integration_criteria:
  - "Connection management implemented"  # Too vague
```

### 4. Validate Assumptions

**Good:**
```yaml
integration_criteria:
  - "Auth expects JWT in Authorization header (verified)"
  - "Router passes request context to handlers (checked)"
```

**Bad:**
```yaml
integration_criteria:
  - "Components integrate"  # No specific validation
```

## Common Mistakes

### Mistake 1: No Integration Task for Multi-Component Plan

```yaml
# WRONG - 10 component tasks, no integration
tasks: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

# RIGHT - Components + integration
tasks:
  [1-3]: Auth components
  [4-6]: API components
  [7]: INTEGRATION - wire auth to API
  [8-9]: Feature components
  [10]: INTEGRATION - wire features to API
```

### Mistake 2: Integration Criteria on Component Task

```yaml
# WRONG - component task with integration criteria
- task_number: 1
  name: "Implement JWT Module"
  type: component
  integration_criteria:  # Shouldn't be here
    - "Works with router"
```

### Mistake 3: Component Criteria on Integration Task

```yaml
# WRONG - integration task missing integration criteria
- task_number: 7
  name: "Wire Auth to Router"
  type: integration
  success_criteria:  # Only component checks
    - "Code compiles"
  # Missing integration_criteria!
```

### Mistake 4: Vague Integration Criteria

```yaml
# WRONG - too vague
integration_criteria:
  - "Everything works together"
  - "Integration successful"

# RIGHT - specific, verifiable
integration_criteria:
  - "Auth middleware executes before handlers (verify logs)"
  - "Unauthenticated requests return 401 (test endpoint)"
```

Integration tasks ensure components don't just work in isolation - they work together correctly.
