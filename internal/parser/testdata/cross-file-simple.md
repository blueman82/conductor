# Simple Cross-File Example

## Task 1: Setup Database

**File(s)**: `internal/db/schema.sql`, `internal/db/connection.go`
**Depends on**: None
**Estimated time**: 45m
**Agent**: golang-pro

Initialize database schema and connection pool.

## Task 2: Implement API Layer

**File(s)**: `internal/api/handlers.go`, `internal/api/routes.go`
**Depends on**: file:cross-file-mixed.md:task:1
**Estimated time**: 1h30m
**Agent**: golang-pro

Create API endpoints that depend on database from other plan.
