# Generate Tasks: Detailed Process Reference

This file contains detailed examples, verbose templates, and extended guidance for the generate-tasks skill.

## Complete Output Format Template

The generated task list MUST follow this structure:

```markdown
# Task List: [Feature Name]

> **Source PRD**: `.claude/tasks/NNNN-prd-feature-name.md`
> **Generated**: YYYY-MM-DD
> **Status**: Not Started / In Progress / Completed

---

## Current State Assessment

[Summary from Step 3]

---

## Relevant Files

### Create (New Files)
- `path/to/file.ts` - Description [Complexity: Simple/Medium/Complex]
- `path/to/file.test.ts` - Unit tests for file.ts [Complexity: Medium]

### Modify (Existing Files)
- `path/to/existing.ts` - Changes needed [Complexity: Simple]

### Reference (Read Only)
- `path/to/reference.ts` - Existing pattern to follow

---

## Implementation Notes

### Testing Requirements
[From Step 10]

### Security Checklist
[From Step 10]

### Performance Targets
[From Step 10]

### Common Pitfalls
[From Step 10]

---

## Tasks

- [ ] 1.0 Parent Task Title
  - [ ] 1.1 Sub-task description [~2,000 tokens - Simple]
    <!-- Guardrails: ✓ Specific guardrails for this task -->
  - [ ] 1.2 Sub-task description [~4,000 tokens - Medium]
    <!-- Guardrails: ✓ Specific guardrails for this task -->

- [ ] 2.0 Parent Task Title
  - [ ] 2.1 Sub-task description [~3,000 tokens - Medium]
  - [ ] 2.2 Sub-task description [~5,000 tokens - Complex]

- [ ] 3.0 Testing & Validation
  - [ ] 3.1 Verify all unit tests passing
  - [ ] 3.2 Verify coverage >95% for auth module
  - [ ] 3.3 Verify all guardrails validated
  - [ ] 3.4 Run security audit (npm audit)
  - [ ] 3.5 Performance testing (load testing if applicable)

- [ ] 4.0 Documentation & Deployment
  - [ ] 4.1 Update API documentation
  - [ ] 4.2 Update CLAUDE.md with auth architecture
  - [ ] 4.3 Add to CLAUDE.md if new patterns emerged
  - [ ] 4.4 Create deployment checklist
  - [ ] 4.5 Update README with setup instructions

---

## Progress Tracking

**Total Tasks**: X parent, Y sub-tasks
**Completed**: 0/Y (0%)
**In Progress**: None
**Blocked**: None

**Last Updated**: YYYY-MM-DD HH:MM

---

## Success Criteria

Before marking this task list complete, verify:
- [ ] All functional requirements from PRD implemented
- [ ] All guardrails validated (especially security & testing)
- [ ] Test coverage meets targets (>95% for auth)
- [ ] Performance targets met (<200ms auth responses)
- [ ] Documentation updated (.claude/ files, API docs, README)
- [ ] No security vulnerabilities (npm audit clean)
- [ ] User acceptance testing passed (if applicable)

---

## Next Steps

1. Review this task list with user
2. User confirms or requests changes
3. Start with task 1.1 (implement atomically, one commit per task)
4. After each task: Run tests, validate guardrails, commit
5. Update progress tracking after each task
6. Mark parent task complete only when all sub-tasks done
```

---

## Detailed File Listing Format

### Example: Authentication Feature

```markdown
## Relevant Files

### Create (New Files)
- `src/db/schemas/user.schema.ts` - User table schema with Prisma [Complexity: Simple]
- `src/db/schemas/user.schema.test.ts` - Schema validation tests [Complexity: Simple]
- `src/auth/auth.service.ts` - Core authentication business logic [Complexity: Complex]
- `src/auth/auth.service.test.ts` - Service unit tests (>95% coverage target) [Complexity: Complex]
- `src/auth/auth.middleware.ts` - JWT validation middleware [Complexity: Medium]
- `src/auth/auth.middleware.test.ts` - Middleware tests [Complexity: Medium]
- `src/auth/auth.types.ts` - TypeScript types and interfaces [Complexity: Simple]
- `src/api/auth/auth.controller.ts` - Route handlers (login, register, logout) [Complexity: Medium]
- `src/api/auth/auth.controller.test.ts` - Controller integration tests [Complexity: Medium]
- `src/api/auth/auth.routes.ts` - Express route definitions [Complexity: Simple]

### Modify (Existing Files)
- `src/app.ts` - Add auth routes to Express app [Complexity: Simple]
- `src/middleware/index.ts` - Export auth middleware [Complexity: Simple]
- `.env.example` - Add JWT_SECRET, BCRYPT_ROUNDS placeholders [Complexity: Simple]
- `package.json` - Add dependencies (bcrypt, jsonwebtoken, passport) [Complexity: Simple]

### Reference (Read but don't modify)
- `lib/validation.ts` - Use existing Zod validators
- `lib/errors.ts` - Use existing error classes
- `CLAUDE.md` - Check for architecture patterns
```

---

## Extended Implementation Notes Examples

### Example 1: Authentication Feature

```markdown
## Implementation Notes

### Testing Requirements
- All auth logic must have >95% coverage (business-critical)
- Test files alongside code (e.g., `auth.service.ts` + `auth.service.test.ts`)
- Use `npm test src/auth` to run auth module tests only
- Integration tests must test actual JWT validation, not mocks
- Test both happy paths and error scenarios
- Include tests for token expiration, invalid tokens, missing tokens
- Mock external dependencies (database, email service) appropriately

### Security Checklist (verify before each commit)
- [ ] All passwords hashed (never store plain text)
- [ ] All JWT secrets from environment variables
- [ ] All inputs validated with Zod schemas
- [ ] All SQL queries parameterized (use Prisma)
- [ ] Rate limiting on login/register endpoints
- [ ] Authentication events logged
- [ ] No sensitive data in error messages
- [ ] HTTPS enforced in production
- [ ] Tokens use httpOnly cookies (not localStorage)
- [ ] CSRF protection enabled
- [ ] Password complexity requirements enforced

### Performance Targets
- Login/register: <200ms (p95)
- JWT validation: <50ms (p95)
- Password hashing: 100-300ms (bcrypt cost 12)
- Database queries: <100ms per query
- Session lookup: <50ms (use caching if needed)

### Common Pitfalls
- ❌ Don't validate JWT in database query (use middleware)
- ❌ Don't store tokens in localStorage (use httpOnly cookies)
- ❌ Don't return sensitive data in error messages
- ❌ Don't skip rate limiting (prevents brute force)
- ❌ Don't use synchronous bcrypt (blocks event loop)
- ❌ Don't forget to hash passwords before storing
- ❌ Don't use weak JWT secrets (min 256 bits)
- ❌ Don't implement your own crypto (use battle-tested libraries)

### Integration Points
- Email service for verification emails (use existing service)
- Logging service for audit trail
- Rate limiting middleware (install express-rate-limit)
- Session store (Redis recommended for production)

### Dependencies to Add
```json
{
  "bcrypt": "^5.1.1",
  "jsonwebtoken": "^9.0.2",
  "passport": "^0.7.0",
  "passport-jwt": "^4.0.1",
  "express-rate-limit": "^7.1.5",
  "zod": "^3.22.4"
}
```

### Environment Variables
```bash
# .env.example
JWT_SECRET=your-secret-key-here-minimum-256-bits
JWT_EXPIRES_IN=7d
BCRYPT_ROUNDS=12
RATE_LIMIT_WINDOW_MS=900000
RATE_LIMIT_MAX_REQUESTS=5
```
```

### Example 2: Database Migration Feature

```markdown
## Implementation Notes

### Testing Requirements
- Test migrations in development environment first
- Verify rollback (down migration) works correctly
- Test with sample data (not empty database)
- Verify indexes created correctly
- Check foreign key constraints enforced
- Test data type validations
- Verify default values applied

### Migration Checklist (verify before applying)
- [ ] Migration includes both up and down functions
- [ ] Foreign keys reference correct tables/columns
- [ ] Indexes added for frequently queried columns
- [ ] Default values set for new columns
- [ ] NOT NULL constraints appropriate
- [ ] Migration tested locally first
- [ ] Backup created before production migration
- [ ] Rollback plan documented

### Performance Considerations
- Large tables: Use batched updates (not single transaction)
- Add indexes concurrently if database supports it
- Consider maintenance window for large migrations
- Monitor query performance after migration
- Plan for zero-downtime deployment if needed

### Common Pitfalls
- ❌ Don't forget rollback (down) migration
- ❌ Don't modify existing migrations (create new one)
- ❌ Don't add NOT NULL without default on existing table
- ❌ Don't forget to update TypeScript types
- ❌ Don't skip testing rollback
- ❌ Don't run migrations manually (use migration tool)

### Prisma-Specific Notes
- Run `npx prisma migrate dev` in development
- Run `npx prisma migrate deploy` in production
- Use `npx prisma db push` only for prototyping
- Update schema.prisma first, then generate migration
- Verify generated SQL before applying
- Keep migrations in version control
```

---

## Extended Task Breakdown Examples

### Example 1: Complete Authentication Feature

```markdown
# Task List: User Authentication System

> **Source PRD**: `.claude/tasks/0001-prd-user-authentication.md`
> **Generated**: 2025-01-15
> **Status**: Not Started

---

## Current State Assessment

**Existing Infrastructure:**
- Express.js backend with TypeScript
- Prisma ORM with PostgreSQL
- Jest for testing (78% coverage currently)
- React frontend with Tailwind CSS

**Reusable Components:**
- `lib/validation.ts` - Input validation utilities (use Zod)
- `components/Form.tsx` - Form component (reuse for auth forms)
- `lib/errors.ts` - Centralized error handling (extend for auth errors)

**Patterns to Follow:**
- API routes in `src/api/<feature>/` structure
- Tests alongside code (`*.test.ts`)
- Repository pattern for data access
- Environment variables via `.env` with fallbacks

**Guardrails Note:**
- `src/api/user/user.controller.ts` is 285 lines (near 300 limit - don't modify)
- Create new auth module instead of extending user module

---

## Relevant Files

### Create (New Files)
- `src/db/schemas/user.schema.ts` - User table schema [Simple]
- `src/db/schemas/user.schema.test.ts` - Schema validation tests [Simple]
- `src/db/schemas/session.schema.ts` - Session table schema [Simple]
- `src/auth/auth.service.ts` - Core authentication logic [Complex]
- `src/auth/auth.service.test.ts` - Service unit tests [Complex]
- `src/auth/auth.middleware.ts` - JWT validation middleware [Medium]
- `src/auth/auth.middleware.test.ts` - Middleware tests [Medium]
- `src/auth/auth.types.ts` - TypeScript interfaces [Simple]
- `src/api/auth/auth.controller.ts` - Route handlers [Medium]
- `src/api/auth/auth.controller.test.ts` - Controller tests [Medium]
- `src/api/auth/auth.routes.ts` - Express routes [Simple]

### Modify (Existing Files)
- `src/app.ts` - Add auth routes [Simple]
- `src/middleware/index.ts` - Export auth middleware [Simple]
- `.env.example` - Add auth environment variables [Simple]
- `package.json` - Add dependencies [Simple]

### Reference (Read Only)
- `lib/validation.ts` - Use existing Zod validators
- `lib/errors.ts` - Use existing error classes

---

## Implementation Notes

[See Extended Implementation Notes Examples above]

---

## Tasks

- [ ] 1.0 Database Schema & Migrations
  - [ ] 1.1 Create user schema with Prisma [~1,500 tokens - Simple]
    <!-- Guardrails:
      ✓ All database migrations include rollback function
      ✓ Parameterized queries (Prisma enforces)
      ✓ All environment variables have secure defaults
    -->
  - [ ] 1.2 Create session schema with Prisma [~1,500 tokens - Simple]
    <!-- Guardrails:
      ✓ Foreign keys reference correct tables
      ✓ Indexes added for user_id and token columns
      ✓ Expiration timestamp for cleanup
    -->
  - [ ] 1.3 Generate and run Prisma migration [~1,000 tokens - Simple]
    <!-- Guardrails:
      ✓ Test rollback before applying
      ✓ Verify indexes created
    -->
  - [ ] 1.4 Verify schema in PostgreSQL [~800 tokens - Simple]
    <!-- Guardrails:
      ✓ Check constraints enforced
      ✓ Indexes exist
    -->
  - [ ] 1.5 Update .env.example with DATABASE_URL [~500 tokens - Simple]

- [ ] 2.0 Backend Authentication Service
  - [ ] 2.1 Create auth.types.ts with TypeScript interfaces [~1,200 tokens - Simple]
    <!-- Guardrails:
      ✓ All exported types documented
      ✓ No any types (strict mode)
    -->
  - [ ] 2.2 Implement password hashing utilities [~2,500 tokens - Medium]
    <!-- Guardrails:
      ✓ Function ≤50 lines (split hash, verify, validate)
      ✓ All exported functions have type signatures
      ✓ Environment variables for bcrypt rounds
      ✓ Edge cases tested
    -->
  - [ ] 2.3 Implement JWT token generation [~2,500 tokens - Medium]
    <!-- Guardrails:
      ✓ JWT secret from environment
      ✓ Token expiration configured
      ✓ No sensitive data in payload
    -->
  - [ ] 2.4 Implement user registration logic [~4,000 tokens - Medium]
    <!-- Guardrails:
      ✓ All inputs validated with Zod
      ✓ Password hashed before storage
      ✓ Duplicate email check
      ✓ Edge cases tested
    -->
  - [ ] 2.5 Implement login logic [~4,000 tokens - Medium]
    <!-- Guardrails:
      ✓ All inputs validated
      ✓ Password verified with bcrypt
      ✓ Session created on success
      ✓ Failed attempts logged
    -->
  - [ ] 2.6 Implement logout logic [~2,000 tokens - Simple]
    <!-- Guardrails:
      ✓ Session invalidated
      ✓ Token blacklisted if needed
    -->

- [ ] 3.0 API Routes & Middleware
  - [ ] 3.1 Create JWT validation middleware [~3,500 tokens - Medium]
    <!-- Guardrails:
      ✓ Function ≤50 lines
      ✓ All async operations have timeout
      ✓ Proper error handling
      ✓ Token verified against JWT secret
    -->
  - [ ] 3.2 Create auth.controller.ts with route handlers [~5,000 tokens - Complex]
    <!-- Guardrails:
      ✓ All API boundaries have input validation
      ✓ API responses <200ms for simple queries
      ✓ Proper HTTP status codes
      ✓ No sensitive data in responses
    -->
  - [ ] 3.3 Create auth.routes.ts with Express routes [~2,000 tokens - Simple]
    <!-- Guardrails:
      ✓ Rate limiting applied
      ✓ Middleware order correct
      ✓ Routes documented
    -->
  - [ ] 3.4 Integrate auth routes into Express app [~1,500 tokens - Simple]
    <!-- Guardrails:
      ✓ Routes mounted at /api/auth
      ✓ Middleware applied globally where needed
    -->

- [ ] 4.0 Testing & Validation
  - [ ] 4.1 Write unit tests for auth.service.ts [~6,000 tokens - Complex]
    <!-- Guardrails:
      ✓ Coverage >95% for auth module
      ✓ All edge cases tested
      ✓ No test interdependencies
      ✓ Mock database appropriately
    -->
  - [ ] 4.2 Write middleware tests [~3,500 tokens - Medium]
    <!-- Guardrails:
      ✓ Test valid, invalid, expired tokens
      ✓ Test missing Authorization header
      ✓ Test malformed tokens
    -->
  - [ ] 4.3 Write controller integration tests [~5,000 tokens - Complex]
    <!-- Guardrails:
      ✓ Test all endpoints
      ✓ Test authentication flow end-to-end
      ✓ Verify rate limiting
    -->
  - [ ] 4.4 Run security audit [~1,000 tokens - Simple]
    <!-- Guardrails:
      ✓ npm audit clean
      ✓ No secrets in code
      ✓ Dependencies up to date
    -->
  - [ ] 4.5 Verify coverage >95% [~500 tokens - Simple]

- [ ] 5.0 Documentation & Deployment
  - [ ] 5.1 Update API documentation [~2,000 tokens - Medium]
    <!-- Include: endpoints, request/response formats, authentication headers -->
  - [ ] 5.2 Update CLAUDE.md with auth architecture [~1,500 tokens - Simple]
  - [ ] 5.3 Add auth patterns to CLAUDE.md [~1,000 tokens - Simple]
  - [ ] 5.4 Update README with setup instructions [~1,500 tokens - Simple]
  - [ ] 5.5 Create deployment checklist [~1,000 tokens - Simple]

---

## Progress Tracking

**Total Tasks**: 5 parent, 26 sub-tasks
**Completed**: 0/26 (0%)
**In Progress**: None
**Blocked**: None

**Last Updated**: 2025-01-15 10:30

---

## Success Criteria

Before marking this task list complete, verify:
- [ ] All functional requirements from PRD implemented
- [ ] All guardrails validated (especially security & testing)
- [ ] Test coverage >95% for auth module
- [ ] Performance targets met (<200ms auth responses)
- [ ] Documentation updated (.claude/ files, API docs, README)
- [ ] No security vulnerabilities (npm audit clean)
- [ ] User acceptance testing passed

---

## Next Steps

1. Review this task list with user
2. User confirms or requests changes
3. Start with task 1.1 (implement atomically, one commit per task)
4. After each task: Run tests, validate guardrails, commit
5. Update progress tracking after each task
6. Mark parent task complete only when all sub-tasks done
```

---

## Guardrails Quick Reference

### Code Quality Guardrails
- ✓ No function exceeds 50 lines
- ✓ No file exceeds 300 lines
- ✓ Cyclomatic complexity ≤10 per function
- ✓ All exported functions have type signatures and JSDoc
- ✓ No magic numbers (use named constants)
- ✓ No commented-out code
- ✓ No TODO without issue reference
- ✓ No dead code

### Security Guardrails
- ✓ All user inputs validated
- ✓ All API boundaries have input validation
- ✓ All database queries parameterized
- ✓ All environment variables have secure defaults
- ✓ All file operations validate paths
- ✓ All async operations have timeout/cancellation
- ✓ Dependencies checked for vulnerabilities
- ✓ Database migrations include rollback

### Testing Guardrails
- ✓ Coverage >80% business logic, >60% overall
- ✓ All public APIs have unit tests
- ✓ Bug fixes include regression tests
- ✓ Edge cases tested
- ✓ Test names describe behavior
- ✓ No test interdependencies
- ✓ Integration tests for external services

### Git Guardrails
- ✓ Conventional commit messages
- ✓ One logical change per commit
- ✓ All commits pass tests
- ✓ Branch naming convention
- ✓ No commits directly to main
- ✓ Breaking changes bump major version

### Performance Guardrails
- ✓ No N+1 queries
- ✓ Large datasets use pagination
- ✓ Expensive computations cached
- ✓ Frontend bundles <200KB
- ✓ API responses <200ms simple, <1s complex

---

## Task Complexity Guidelines

### Simple Tasks (<2,000 tokens)
- Single function with straightforward logic
- Type definitions
- Configuration updates
- Simple tests for straightforward code
- Environment variable additions

**Examples:**
- Create TypeScript interface
- Add environment variable to .env.example
- Write test for simple utility function
- Update import statements

### Medium Tasks (2,000-5,000 tokens)
- Multiple related functions
- Moderate complexity logic
- Integration with existing systems
- Tests with some edge cases

**Examples:**
- Implement password hashing service
- Create JWT middleware
- Write integration tests for API endpoint
- Implement form validation

### Complex Tasks (5,000-10,000 tokens)
- Multiple files affected
- Integration across systems
- Comprehensive test suite
- Complex business logic

**Examples:**
- Complete authentication service
- Full controller with all CRUD operations
- End-to-end feature implementation
- Comprehensive test suite for module

---

## Common Task Patterns

### Pattern 1: Create New Service

```markdown
- [ ] X.0 Create [Service Name] Service
  - [ ] X.1 Create service types/interfaces [~1,200 tokens - Simple]
  - [ ] X.2 Implement core service logic [~5,000 tokens - Complex]
  - [ ] X.3 Write unit tests [~4,000 tokens - Medium]
  - [ ] X.4 Add service to dependency injection [~1,500 tokens - Simple]
```

### Pattern 2: Create New API Endpoint

```markdown
- [ ] X.0 Create [Endpoint Name] API
  - [ ] X.1 Create request/response types [~1,000 tokens - Simple]
  - [ ] X.2 Implement controller handler [~3,000 tokens - Medium]
  - [ ] X.3 Add input validation [~2,000 tokens - Medium]
  - [ ] X.4 Create route definition [~1,500 tokens - Simple]
  - [ ] X.5 Write integration tests [~4,000 tokens - Complex]
  - [ ] X.6 Update API documentation [~1,500 tokens - Simple]
```

### Pattern 3: Database Migration

```markdown
- [ ] X.0 Database Migration: [Name]
  - [ ] X.1 Create schema definition [~2,000 tokens - Simple]
  - [ ] X.2 Generate migration [~1,000 tokens - Simple]
  - [ ] X.3 Test migration locally [~1,500 tokens - Simple]
  - [ ] X.4 Test rollback [~1,500 tokens - Simple]
  - [ ] X.5 Update TypeScript types [~1,000 tokens - Simple]
  - [ ] X.6 Apply migration to dev environment [~1,000 tokens - Simple]
```

### Pattern 4: Frontend Component

```markdown
- [ ] X.0 Create [Component Name] Component
  - [ ] X.1 Create component types [~1,000 tokens - Simple]
  - [ ] X.2 Implement component UI [~4,000 tokens - Medium]
  - [ ] X.3 Add component logic/hooks [~3,000 tokens - Medium]
  - [ ] X.4 Style with Tailwind/CSS [~2,000 tokens - Simple]
  - [ ] X.5 Write component tests [~3,500 tokens - Medium]
  - [ ] X.6 Add to Storybook (if applicable) [~2,000 tokens - Simple]
```

---

**Remember**: This reference file provides detailed templates and examples. The main SKILL.md should remain concise and actionable.
