# Mixed Dependency Example

## Task 1: Implement Authentication

**File(s)**: `internal/auth/jwt.go`, `internal/auth/middleware.go`
**Depends on**: None
**Estimated time**: 1h
**Agent**: golang-pro

Implement JWT-based authentication.

## Task 2: Setup Cache Layer

**File(s)**: `internal/cache/redis.go`
**Depends on**: Task 1
**Estimated time**: 45m
**Agent**: golang-pro

Configure Redis cache with authentication.

## Task 3: Wire Everything Together

**File(s)**: `cmd/api/main.go`
**Depends on**: Task 2, file:cross-file-simple.md/task:2
**Estimated time**: 30m
**Agent**: golang-pro

Integration task that depends on tasks from current and another plan.
