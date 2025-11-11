# Backend Implementation Plan - Part 2: Features

**Created**: 2025-11-11
**Target**: API endpoints and business logic
**Estimated Tasks**: 4
**Files**: Part 2 of 3
**Depends on**: Part 1 (split-plan-backend-1-setup.md)

Part 2 implements the core API endpoints and business logic.

---

## Task 4: Authentication Service

**File(s)**: `internal/auth/auth.go`, `internal/auth/auth_test.go`
**Depends on**: Task 3
**Estimated time**: 45m
**Agent**: backend-engineer
**WorktreeGroup**: backend-features

Implement JWT-based authentication with user login/signup endpoints. Hash passwords using bcrypt.

**Requirements:**
- JWT token generation and validation
- Password hashing with bcrypt
- Token refresh mechanism
- User session management
- Comprehensive test coverage (>90%)

---

## Task 5: User API Endpoints

**File(s)**: `internal/api/users.go`, `internal/api/handlers/users_test.go`
**Depends on**: Task 4
**Estimated time**: 40m
**Agent**: backend-engineer
**WorktreeGroup**: backend-features

Create REST endpoints for user management: GET /users, POST /users (create), GET /users/:id, PUT /users/:id, DELETE /users/:id.

**Requirements:**
- CRUD operations for users
- Proper HTTP status codes
- Input validation
- Error responses in standard format
- Authentication required for protected endpoints

---

## Task 6: Product Catalog API

**File(s)**: `internal/api/products.go`, `internal/api/handlers/products_test.go`
**Depends on**: Task 4
**Estimated time**: 35m
**Agent**: backend-engineer
**WorktreeGroup**: backend-features

Implement product catalog endpoints: GET /products, GET /products/:id, POST /products (admin only), search/filter.

**Requirements:**
- List products with pagination
- Product detail endpoint
- Admin-only create/update
- Search by name/category
- Inventory tracking

---

## Task 7: Order Processing

**File(s)**: `internal/api/orders.go`, `internal/models/order.go`
**Depends on**: Task 5, Task 6
**Estimated time**: 50m
**Agent**: backend-engineer
**WorktreeGroup**: backend-features

Implement order creation, tracking, and history. Calculate totals, apply discounts, handle inventory.

**Requirements:**
- Create orders from cart items
- Calculate order total with tax/shipping
- Inventory deduction
- Order history and status tracking
- Cancellation support

---

## Execution Notes

Part 2 implements the business logic layer:
- Users can register and authenticate
- Browse and search products
- Create and track orders
- Protected endpoints with authorization

The API is now functional but lacks:
- Testing (covered in Part 3)
- Deployment configuration
- Performance optimization

Proceed to Part 3 (split-plan-backend-3-testing.md) for testing and optimization.

---

## Notes for Review

**File Organization:**
- `split-plan-backend-1-setup.md` - Database and server (3 tasks)
- `split-plan-backend-2-features.md` - API and business logic (4 tasks)
- `split-plan-backend-3-testing.md` - Testing and deployment (3 tasks)

**Dependencies:**
- Part 2 tasks depend only on Part 1 (except Task 7 which depends on Tasks 5-6)
- Part 3 tasks depend on Part 2 (and indirectly Part 1)
- Clean dependency boundaries between parts

**Execution:**
```bash
# Validate all parts together
conductor validate split-plan-backend-*.md

# Run all parts
conductor run split-plan-backend-*.md --verbose

# Or run incrementally with skip-completed
conductor run split-plan-backend-1-setup.md
conductor run split-plan-backend-*.md --skip-completed
```
