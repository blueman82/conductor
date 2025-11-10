

# Feature Design: User Authentication System

**Created**: 2025-01-09
**Status**: Approved
**Design Method**: Multi-Model Deliberation (cook-auto)

## Overview & Objectives

### What We're Building

A complete user authentication system for the web application, including:
- User registration with email/password
- Login with JWT token generation
- Password hashing with bcrypt
- Token-based authentication for API endpoints

### Why We're Building This

Currently, the application has no user authentication, which means:
- No user-specific features or personalization
- No access control or authorization
- No way to identify who is making requests

This system will enable user accounts and lay the foundation for user-specific features.

### Success Criteria

- Users can register with email and password
- Users can login and receive JWT token
- Tokens can be used to access protected API endpoints
- Passwords are securely hashed (never stored plain text)
- All authentication endpoints have comprehensive tests

## Architecture & Technical Approach

### System Architecture

```
┌─────────────┐
│   Client    │
│ (Browser)   │
└──────┬──────┘
       │ HTTP/JSON
       ↓
┌─────────────────────────────────┐
│      HTTP API Layer             │
│  POST /api/register             │
│  POST /api/login                │
└──────┬──────────────────────────┘
       │
       ↓
┌─────────────────────────────────┐
│   Authentication Service        │
│  - Register()                   │
│  - Login()                      │
│  - GenerateToken()              │
└──────┬──────────────────────────┘
       │
       ↓
┌─────────────────────────────────┐
│      User Repository            │
│  - Create()                     │
│  - FindByEmail()                │
└──────┬──────────────────────────┘
       │
       ↓
┌─────────────────────────────────┐
│    PostgreSQL Database          │
│    users table                  │
└─────────────────────────────────┘
```

### Technology Stack

**Language**: Go 1.21+
**Framework**: Standard library (net/http)
**Database**: PostgreSQL
**Password Hashing**: bcrypt (cost factor 12)
**JWT**: golang-jwt/jwt/v5 with RS256 algorithm
**Testing**: Go testing package + testify for assertions

### Integration with Existing Code

The authentication system will follow the existing patterns in the codebase:

1. **Models**: Place in `internal/models/` (like existing Post model)
2. **Services**: Place in `internal/services/` (dependency injection pattern)
3. **API Handlers**: Place in `internal/api/handlers/` (existing handler structure)
4. **Database Migrations**: Place in `migrations/` (sequential numbering)
5. **Tests**: Co-located with source files (`*_test.go`)

### Key Technical Decisions

#### Decision 1: JWT vs Session-Based Auth

**Choice**: JWT (JSON Web Tokens)

**Rationale** (from multi-model consensus):
- **Claude Opus**: JWT enables stateless authentication, reducing server memory requirements and enabling horizontal scaling
- **Claude Sonnet**: JWTs work better for API-first architecture and mobile clients
- **GPT-5**: Session-based auth requires server-side storage; JWT is self-contained

**Trade-offs**:
- ✓ Stateless (no server-side session storage)
- ✓ Works across multiple servers
- ✓ Easy to use with mobile/SPA clients
- ✗ Can't easily revoke tokens (must wait for expiration)
- ✗ Tokens can be larger than session IDs

**Mitigation**: Use short expiration times (1 hour) and implement refresh tokens in future iteration

#### Decision 2: Password Hashing Algorithm

**Choice**: bcrypt with cost factor 12

**Rationale**:
- **Claude Opus**: bcrypt is specifically designed for password hashing, intentionally slow
- **Claude Sonnet**: Industry standard for password security, proven track record
- **GPT-5**: Adaptive cost factor allows increasing security as hardware improves

**Alternative considered**: Argon2 (newer, more secure)
**Why not chosen**: bcrypt is more widely adopted in Go ecosystem, sufficient for current needs

#### Decision 3: JWT Signing Algorithm

**Choice**: RS256 (RSA with SHA-256)

**Rationale**:
- **Claude Opus**: Asymmetric keys allow verification without exposing signing key
- **Claude Sonnet**: Public key can be distributed to services for token verification
- **Gemini**: Better for microservices architecture (future-proofing)

**Alternative considered**: HS256 (HMAC with SHA-256)
**Why not chosen**: Symmetric key must be shared with all verifying services

## User Experience & Interface

### Registration Flow

```
1. User visits /register page
2. Enters email and password
3. Frontend sends POST to /api/register
4. Backend validates input:
   - Email format correct
   - Password meets strength requirements (min 8 chars)
   - Email not already registered
5. Backend creates user with hashed password
6. Returns 201 Created with user info (no password)
7. User is redirected to login page
```

**API Contract:**

Request:
```json
POST /api/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepass123"
}
```

Success Response:
```json
HTTP/1.1 201 Created
Content-Type: application/json

{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "created_at": "2025-01-09T10:30:00Z"
}
```

Error Response (Duplicate Email):
```json
HTTP/1.1 409 Conflict
Content-Type: application/json

{
  "error": "email already registered"
}
```

Error Response (Validation):
```json
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "invalid email format"
}
```

### Login Flow

```
1. User visits /login page
2. Enters email and password
3. Frontend sends POST to /api/login
4. Backend finds user by email
5. Backend verifies password hash
6. Backend generates JWT token (1 hour expiration)
7. Returns 200 OK with token
8. Frontend stores token in localStorage
9. Frontend includes token in Authorization header for subsequent requests
```

**API Contract:**

Request:
```json
POST /api/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepass123"
}
```

Success Response:
```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-01-09T11:30:00Z"
}
```

Error Response:
```json
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "invalid credentials"
}
```

### Authenticated Request Flow

```
1. User makes request to protected endpoint
2. Frontend includes JWT in Authorization header:
   Authorization: Bearer <token>
3. Backend middleware extracts token
4. Backend validates token:
   - Signature valid
   - Not expired
   - Claims valid
5. Backend extracts user ID from token
6. Backend attaches user to request context
7. Handler processes request with user context
8. Returns response
```

## Data Model & State Management

### User Model

```go
type User struct {
    ID           string    `json:"id" db:"id"`
    Email        string    `json:"email" db:"email"`
    PasswordHash string    `json:"-" db:"password_hash"` // Never expose in JSON
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
```

**Field Specifications:**
- `ID`: UUID v4 (generated on creation)
- `Email`: String, max 255 chars, unique, validated format
- `PasswordHash`: String, bcrypt hash (60 chars), never exposed in API
- `CreatedAt`: Timestamp, set on creation
- `UpdatedAt`: Timestamp, updated on modification

### Database Schema

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(60) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

**Indexes:**
- Primary key on `id` (automatic)
- Unique index on `email` (enforce uniqueness, speed up lookups)

### JWT Token Structure

```json
{
  "header": {
    "alg": "RS256",
    "typ": "JWT"
  },
  "payload": {
    "sub": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "iat": 1704795000,
    "exp": 1704798600
  }
}
```

**Claims:**
- `sub` (subject): User ID
- `email`: User email (for convenience)
- `iat` (issued at): Token creation timestamp
- `exp` (expiration): Token expiration timestamp (1 hour from issue)

## API Design & Interfaces

### Authentication Service Interface

```go
type AuthService interface {
    // Register creates a new user account
    // Returns error if email already exists or validation fails
    Register(ctx context.Context, email, password string) (*User, error)

    // Login authenticates user and returns JWT token
    // Returns error if credentials are invalid
    Login(ctx context.Context, email, password string) (token string, err error)

    // ValidateToken verifies JWT token and returns user ID
    // Returns error if token is invalid or expired
    ValidateToken(ctx context.Context, token string) (userID string, err error)
}
```

### User Repository Interface

```go
type UserRepository interface {
    // Create stores a new user in the database
    // Returns error if email already exists
    Create(ctx context.Context, user *User) error

    // FindByEmail retrieves user by email address
    // Returns ErrNotFound if user doesn't exist
    FindByEmail(ctx context.Context, email string) (*User, error)

    // FindByID retrieves user by ID
    // Returns ErrNotFound if user doesn't exist
    FindByID(ctx context.Context, id string) (*User, error)
}
```

### HTTP API Endpoints

| Method | Path | Description | Auth Required |
|--------|------|-------------|---------------|
| POST | /api/register | Create new user account | No |
| POST | /api/login | Authenticate and get token | No |
| GET | /api/me | Get current user info | Yes |

### Error Codes

| HTTP Code | Error | Meaning |
|-----------|-------|---------|
| 200 | - | Success |
| 201 | - | Created (registration) |
| 400 | invalid_input | Bad request format or validation error |
| 401 | invalid_credentials | Wrong email/password or invalid token |
| 409 | duplicate_email | Email already registered |
| 500 | internal_error | Server error |

## Implementation Considerations

### Security Considerations

**Password Security:**
- Never log passwords
- Never return password hash in API responses
- Use bcrypt cost factor 12 (balance security/performance)
- Enforce minimum password length (8 characters)

**Token Security:**
- Use RS256 for asymmetric signing
- Short expiration time (1 hour)
- Include expiration check in validation
- Verify signature on every request

**Input Validation:**
- Validate email format (regex)
- Sanitize inputs to prevent SQL injection
- Rate limit login attempts (future enhancement)

### Performance Considerations

**Database Performance:**
- Index on email for fast lookups during login
- Connection pooling for database connections
- Consider caching user data (future optimization)

**bcrypt Performance:**
- Cost factor 12 is ~250ms per hash
- Acceptable for login/registration (infrequent operations)
- Run hashing in background goroutine if needed

### Error Handling

**Authentication Errors:**
- Generic "invalid credentials" message (don't reveal if email exists)
- Log failed login attempts for security monitoring
- Return consistent error format

**Database Errors:**
- Handle constraint violations (duplicate email)
- Handle connection errors
- Retry transient failures

### Logging & Observability

**Events to Log:**
- User registration (info level)
- Successful login (info level)
- Failed login attempts (warn level)
- Token validation failures (warn level)

**Never Log:**
- Passwords (plain text or hashed)
- JWT tokens (contain sensitive info)
- Full user objects (may contain PII)

## Testing Strategy

### Unit Tests

**User Model Tests:**
- Email validation (valid/invalid formats)
- Password hashing (bcrypt)
- Password comparison (correct/incorrect)
- JSON serialization (ensure password never exposed)

**Auth Service Tests:**
- Registration with valid data
- Registration with duplicate email
- Registration with invalid data
- Login with valid credentials
- Login with invalid credentials
- Token generation (valid JWT)
- Token validation (valid/expired/malformed)

**Repository Tests:**
- User creation
- Find by email (exists/not exists)
- Find by ID (exists/not exists)
- Duplicate email constraint

### Integration Tests

**Full Authentication Flow:**
1. Register new user → 201 Created
2. Attempt duplicate registration → 409 Conflict
3. Login with correct credentials → 200 OK with token
4. Login with wrong password → 401 Unauthorized
5. Access protected endpoint with token → 200 OK
6. Access protected endpoint without token → 401 Unauthorized
7. Access protected endpoint with expired token → 401 Unauthorized

### Test Coverage Goals

- Unit tests: >85% code coverage
- Integration tests: All critical paths covered
- Edge cases: All error conditions tested

## Success Metrics

### Functional Metrics

- ✓ Users can register with email/password
- ✓ Users can login and receive JWT token
- ✓ Tokens grant access to protected endpoints
- ✓ Invalid credentials are rejected
- ✓ Duplicate registrations are prevented

### Quality Metrics

- ✓ Test coverage >85%
- ✓ All tests passing
- ✓ No security vulnerabilities
- ✓ Code follows existing patterns
- ✓ Documentation complete

### Performance Metrics

- Registration: <500ms (including bcrypt hashing)
- Login: <300ms (including bcrypt verification and token generation)
- Token validation: <10ms (signature verification only)

## Future Enhancements

**Not in scope for this iteration, but consider for future:**

1. **Password Reset Flow**
   - Email-based password reset
   - Time-limited reset tokens

2. **Refresh Tokens**
   - Long-lived refresh tokens for mobile apps
   - Token rotation on refresh

3. **Rate Limiting**
   - Limit login attempts per IP
   - Exponential backoff for failed attempts

4. **OAuth Integration**
   - Google OAuth
   - GitHub OAuth

5. **Multi-Factor Authentication**
   - TOTP (Time-based One-Time Password)
   - SMS verification

6. **Email Verification**
   - Verify email address before allowing login
   - Confirmation link sent to email

## Design Approval

**Reviewed by**: AI Model Consensus (Claude Opus, Claude Sonnet, GPT-5, Gemini 2.5 Pro)

**Key Consensus Points:**
- JWT is appropriate for this use case
- bcrypt is sufficient for password hashing
- RS256 provides good security for token signing
- Simple email/password auth is good starting point

**Areas of Debate:**
- Argon2 vs bcrypt: Consensus chose bcrypt for ecosystem support
- HS256 vs RS256: Consensus chose RS256 for future microservices

**Final Status**: ✓ Approved - Ready for implementation planning

---

**Next Step**: Generate implementation plan using `/doc-yaml` command
