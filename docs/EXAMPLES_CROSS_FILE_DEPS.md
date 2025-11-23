# Examples: Cross-File Dependencies

Real-world examples showing how to use Conductor's cross-file dependency features.

## Table of Contents

- [Simple Two-File Setup](#simple-two-file-setup)
- [Three-File Microservice Architecture](#three-file-microservice-architecture)
- [Complex Multi-Service System](#complex-multi-service-system)
- [Frontend + Backend Integration](#frontend--backend-integration)
- [Multi-Environment Deployment](#multi-environment-deployment)
- [Shared Infrastructure Pattern](#shared-infrastructure-pattern)
- [Progressive Feature Rollout](#progressive-feature-rollout)

---

## Simple Two-File Setup

Perfect for getting started with cross-file dependencies.

### Scenario

Building a simple web application with database setup and API implementation.

### Files

**1. foundation.yaml** - Database initialization
```yaml
plan:
  name: Foundation Layer

  tasks:
    - id: 1
      name: Initialize PostgreSQL Database
      files: [infrastructure/postgres.tf, migrations/001_init.sql]
      depends_on: []
      estimated_time: 5 minutes
      agent: devops-expert
      description: |
        Set up PostgreSQL database with initial schema.
        Create required tables and indexes.

    - id: 2
      name: Setup Database Migrations
      files: [migrations/runner.go, migrations/migrations_test.go]
      depends_on: [1]
      estimated_time: 3 minutes
      agent: golang-pro
      description: |
        Implement database migration system.
        Ensure safe schema versioning.

    - id: 3
      name: Create Test Database
      files: [infrastructure/test_postgres.tf]
      depends_on: [1]
      estimated_time: 2 minutes
      agent: devops-expert
      description: Create isolated test database for CI/CD.
```

**2. services.yaml** - API implementation
```yaml
plan:
  name: API Services

  tasks:
    - id: 4
      name: Implement API Server
      files: [cmd/api/main.go, cmd/api/server.go]
      # Explicit cross-file dependency
      depends_on:
        - file: foundation.yaml
          task: 2    # Needs migration system from foundation
      estimated_time: 10 minutes
      agent: golang-pro
      description: |
        Build REST API server with middleware.
        Implement request routing and response handling.
      success_criteria:
        - "API server starts on port 8080"
        - "Health check endpoint responds"
        - "Graceful shutdown implemented"

    - id: 5
      name: Implement User Endpoints
      files: [internal/api/users.go, internal/api/users_test.go]
      depends_on:
        - 4  # Local: needs API server from same file
        - file: foundation.yaml
          task: 2  # Cross-file: needs migration system
      estimated_time: 8 minutes
      agent: golang-pro
      description: |
        Create CRUD endpoints for user management.
        Include proper error handling and validation.
      success_criteria:
        - "GET /users returns user list"
        - "POST /users creates new user"
        - "PUT /users/:id updates user"
        - "DELETE /users/:id deletes user"
      test_commands:
        - "go test ./internal/api/ -v"

    - id: 6
      name: API Integration Tests
      files: [test/api_integration_test.go]
      depends_on:
        - 5  # Needs user endpoints implemented
        - file: foundation.yaml
          task: 3  # Uses test database
      estimated_time: 5 minutes
      agent: golang-pro
      description: Test API endpoints with real database.
```

### Execution

```bash
# Validate both files
conductor validate foundation.yaml services.yaml

# Execute with progression
conductor run foundation.yaml services.yaml --verbose

# Expected execution order:
# Wave 1: foundation#1
# Wave 2: foundation#2, foundation#3
# Wave 3: services#4
# Wave 4: services#5
# Wave 5: services#6
```

### What's Happening

1. **Wave 1**: Initialize PostgreSQL
2. **Wave 2**: Setup migrations and test database (both depend on Wave 1)
3. **Wave 3**: Build API server (depends on migrations from Wave 2)
4. **Wave 4**: Implement user endpoints (depends on API server)
5. **Wave 5**: Run integration tests (depends on endpoints and test DB)

---

## Three-File Microservice Architecture

Typical microservices setup with shared infrastructure.

### Scenario

Backend system with shared infrastructure, multiple services, and integration layer.

### Files

**1. infrastructure.yaml** - Shared infrastructure
```yaml
plan:
  name: Shared Infrastructure

  tasks:
    - id: 1
      name: Setup Docker & Kubernetes
      files: [k8s/docker-compose.yml, k8s/deployment.yaml]
      depends_on: []
      agent: devops-expert

    - id: 2
      name: Setup Service Registry
      files: [infrastructure/consul/config.hcl]
      depends_on: [1]
      agent: devops-expert

    - id: 3
      name: Setup Distributed Logging
      files: [infrastructure/logging/elk-stack.yaml]
      depends_on: [1]
      agent: devops-expert

    - id: 4
      name: Setup Metrics Collection
      files: [infrastructure/monitoring/prometheus.yaml]
      depends_on: [1]
      agent: devops-expert
```

**2. services.yaml** - Microservices
```yaml
plan:
  name: Core Services

  tasks:
    - id: 5
      name: Build User Service
      files: [services/user/main.go, services/user/user_test.go]
      depends_on:
        - file: infrastructure.yaml
          task: 2  # Needs service registry
        - file: infrastructure.yaml
          task: 3  # Needs logging
      agent: golang-pro

    - id: 6
      name: Build Order Service
      files: [services/order/main.go, services/order/order_test.go]
      depends_on:
        - file: infrastructure.yaml
          task: 2  # Needs service registry
        - file: infrastructure.yaml
          task: 3  # Needs logging
      agent: golang-pro

    - id: 7
      name: Build Payment Service
      files: [services/payment/main.go, services/payment/payment_test.go]
      depends_on:
        - file: infrastructure.yaml
          task: 2  # Needs service registry
        - file: infrastructure.yaml
          task: 3  # Needs logging
      agent: golang-pro
```

**3. integration.yaml** - Service integration
```yaml
plan:
  name: Service Integration

  tasks:
    - id: 8
      name: Configure Service Discovery
      files: [infrastructure/service-mesh/config.yaml]
      depends_on:
        - file: services.yaml
          task: 5  # User service
        - file: services.yaml
          task: 6  # Order service
        - file: services.yaml
          task: 7  # Payment service
      agent: devops-expert

    - id: 9
      name: Setup Inter-Service Communication
      files: [services/api-gateway/main.go]
      depends_on:
        - 8  # Configuration from same file
        - file: services.yaml
          task: 5
        - file: services.yaml
          task: 6
        - file: services.yaml
          task: 7
      agent: golang-pro

    - id: 10
      name: E2E Integration Tests
      files: [test/e2e/services_test.go]
      depends_on:
        - 9  # API gateway must be running
        - file: infrastructure.yaml
          task: 4  # Metrics collection
      agent: golang-pro
```

### Execution

```bash
# Validate all files
conductor validate infrastructure.yaml services.yaml integration.yaml

# Execute with max concurrency
conductor run infrastructure.yaml services.yaml integration.yaml --max-concurrency 4

# Expected order:
# Wave 1: infrastructure#1
# Wave 2: infrastructure#2, infrastructure#3, infrastructure#4
# Wave 3: services#5, services#6, services#7 (parallel - all same deps)
# Wave 4: integration#8
# Wave 5: integration#9
# Wave 6: integration#10
```

### Key Features

- **Star pattern**: Multiple services depend on same infrastructure
- **Parallel execution**: Services can run in parallel (same dependencies)
- **Progressive integration**: Integration tasks only run after services
- **Clean separation**: Infrastructure, services, and integration in separate files

---

## Complex Multi-Service System

Large enterprise system with multiple dependency chains.

### Scenario

Full-stack system: shared utilities, multiple service families, deployment orchestration.

### Files

**1. foundation.yaml** - Core utilities
```yaml
plan:
  name: Foundation & Utilities
  tasks:
    - id: 1
      name: Build Common Libraries
      files: [pkg/common/*.go]
      depends_on: []
      agent: golang-pro

    - id: 2
      name: Setup Shared Database Layer
      files: [pkg/db/*.go, pkg/db/*_test.go]
      depends_on: [1]
      agent: golang-pro

    - id: 3
      name: Setup Configuration Management
      files: [pkg/config/*.go]
      depends_on: [1]
      agent: golang-pro

    - id: 4
      name: Setup Logging Framework
      files: [pkg/logging/*.go]
      depends_on: [1]
      agent: golang-pro
```

**2. data-services.yaml** - Data layer services
```yaml
plan:
  name: Data Services
  tasks:
    - id: 5
      name: Build User Data Service
      files: [services/data/user/*.go]
      depends_on:
        - file: foundation.yaml
          task: 2  # Database
        - file: foundation.yaml
          task: 4  # Logging
      agent: golang-pro

    - id: 6
      name: Build Product Data Service
      files: [services/data/product/*.go]
      depends_on:
        - file: foundation.yaml
          task: 2  # Database
        - file: foundation.yaml
          task: 4  # Logging
      agent: golang-pro
```

**3. business-services.yaml** - Business logic
```yaml
plan:
  name: Business Services
  tasks:
    - id: 7
      name: Build Order Service
      files: [services/business/order/*.go]
      depends_on:
        - file: data-services.yaml
          task: 5  # User data
        - file: data-services.yaml
          task: 6  # Product data
      agent: golang-pro

    - id: 8
      name: Build Recommendation Service
      files: [services/business/recommend/*.go]
      depends_on:
        - file: data-services.yaml
          task: 6  # Product data
      agent: golang-pro
```

**4. api-gateway.yaml** - API layer
```yaml
plan:
  name: API Gateway & Controllers
  tasks:
    - id: 9
      name: Build API Gateway
      files: [services/api/gateway/*.go]
      depends_on:
        - file: foundation.yaml
          task: 3  # Configuration
        - file: foundation.yaml
          task: 4  # Logging
      agent: golang-pro

    - id: 10
      name: Build Order API
      files: [services/api/order/*.go]
      depends_on:
        - 9  # API gateway
        - file: business-services.yaml
          task: 7  # Order service
      agent: golang-pro

    - id: 11
      name: Build Recommendation API
      files: [services/api/recommend/*.go]
      depends_on:
        - 9  # API gateway
        - file: business-services.yaml
          task: 8  # Recommendation service
      agent: golang-pro
```

**5. deployment.yaml** - Deployment
```yaml
plan:
  name: Deployment & Testing
  tasks:
    - id: 12
      name: Build Docker Images
      files: [docker/Dockerfile]
      depends_on:
        - file: api-gateway.yaml
          task: 10
        - file: api-gateway.yaml
          task: 11
      agent: devops-expert

    - id: 13
      name: Deploy to Kubernetes
      files: [k8s/deployment.yaml]
      depends_on: [12]
      agent: devops-expert

    - id: 14
      name: Run Integration Tests
      files: [test/integration/*_test.go]
      depends_on: [13]
      agent: golang-pro
```

### Dependency Visualization

```
foundation.yaml#1
├─ foundation.yaml#2
│  └─ data-services.yaml#5 (User)
│     └─ business-services.yaml#7 (Order)
│        └─ api-gateway.yaml#10
│           └─ deployment.yaml#12
├─ foundation.yaml#3
│  └─ api-gateway.yaml#9
├─ foundation.yaml#4
│  ├─ data-services.yaml#5, #6
│  └─ api-gateway.yaml#9

data-services.yaml#6 (Product)
├─ business-services.yaml#7 (Order)
├─ business-services.yaml#8 (Recommendation)
   └─ api-gateway.yaml#11

deployment.yaml#12 (Build)
└─ deployment.yaml#13 (Deploy)
   └─ deployment.yaml#14 (Test)
```

---

## Frontend + Backend Integration

Full-stack web application.

### Files

**1. backend.yaml** - Backend implementation
```yaml
plan:
  name: Backend
  tasks:
    - id: 1
      name: Database Schema
      files: [db/schema.sql]
      depends_on: []
      agent: devops-expert

    - id: 2
      name: REST API
      files: [api/server.go]
      depends_on: [1]
      agent: golang-pro
```

**2. frontend.yaml** - Frontend implementation
```yaml
plan:
  name: Frontend
  tasks:
    - id: 3
      name: API Client
      files: [client/api.ts]
      depends_on:
        - file: backend.yaml
          task: 2  # Must have API available
      agent: typescript-pro

    - id: 4
      name: UI Components
      files: [ui/components.tsx]
      depends_on: [3]
      agent: typescript-pro
```

**3. integration.yaml** - E2E testing
```yaml
plan:
  name: Integration
  tasks:
    - id: 5
      name: E2E Tests
      files: [test/e2e.test.ts]
      depends_on:
        - file: backend.yaml
          task: 2
        - file: frontend.yaml
          task: 4
      agent: typescript-pro
```

---

## Multi-Environment Deployment

Same application across multiple environments.

### Files Structure

```
deploy/
├── foundation.yaml      (shared)
├── dev.yaml            (dev environment)
├── staging.yaml        (staging environment)
└── production.yaml     (production environment)
```

### Example (dev.yaml)

```yaml
plan:
  name: Development Environment
  tasks:
    - id: 1
      name: Deploy to Dev
      depends_on:
        - file: foundation.yaml
          task: 1  # Base setup
      agent: devops-expert
```

---

## Shared Infrastructure Pattern

Common pattern: shared layer + multiple consumers.

### Files

**shared.yaml:**
```yaml
plan:
  name: Shared
  tasks:
    - id: 1
      name: Infrastructure
      depends_on: []
```

**service-a.yaml:**
```yaml
tasks:
  - id: 2
    depends_on:
      - file: shared.yaml
        task: 1
```

**service-b.yaml:**
```yaml
tasks:
  - id: 3
    depends_on:
      - file: shared.yaml
        task: 1
```

**integration.yaml:**
```yaml
tasks:
  - id: 4
    depends_on:
      - file: service-a.yaml
        task: 2
      - file: service-b.yaml
        task: 3
```

---

## Progressive Feature Rollout

Gradual feature implementation across files.

### Phase 1: Core Features (core.yaml)
```yaml
tasks:
  - id: 1
    name: Core API
    depends_on: []
```

### Phase 2: Premium Features (premium.yaml)
```yaml
tasks:
  - id: 2
    name: Premium Features
    depends_on:
      - file: core.yaml
        task: 1
```

### Phase 3: Enterprise Features (enterprise.yaml)
```yaml
tasks:
  - id: 3
    name: Enterprise Features
    depends_on:
      - file: premium.yaml
        task: 2
```

### Execution

```bash
# Phase 1 only
conductor run core.yaml

# Phase 1 + 2
conductor run core.yaml premium.yaml

# All phases
conductor run core.yaml premium.yaml enterprise.yaml
```

---

## Running These Examples

### Validate
```bash
conductor validate foundation.yaml services.yaml
```

### Dry Run
```bash
conductor run *.yaml --dry-run --verbose
```

### Execute
```bash
conductor run *.yaml --max-concurrency 4 --verbose
```

### Resume
```bash
conductor run *.yaml --skip-completed --retry-failed
```

### See Logs
```bash
tail -f .conductor/logs/*.log
```
