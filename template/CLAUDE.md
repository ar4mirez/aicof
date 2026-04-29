# CLAUDE.md

AI-assisted development instructions. Opinionated guardrails for writing quality software.

> **AGENTS.md Compatible**: This file follows the [AGENTS.md](https://agents.md) standard.
> A copy exists as `AGENTS.md` for cross-tool compatibility (Cursor, Codex, Copilot, etc.)

---

## Operations

> **Purpose**: Immediate, executable instructions for AI agents. Commands first, context second.

### Setup Commands

```bash
# Clone and install (customize per project)
git clone <repo-url> && cd <project>

# Node.js/TypeScript
npm install          # or: yarn install | pnpm install
npm run dev          # Start development server

# Python
python -m venv venv && source venv/bin/activate
pip install -r requirements.txt
python manage.py runserver  # Django
uvicorn main:app --reload   # FastAPI

# Go
go mod download
go run ./cmd/api

# Rust
cargo build
cargo run
```

### Testing Commands

```bash
# Node.js/TypeScript
npm test             # Run all tests
npm run test:watch   # Watch mode
npm run test:cov     # With coverage (target: >80% business logic)

# Python
pytest               # Run all tests
pytest -v            # Verbose
pytest --cov=src     # With coverage

# Go
go test ./...        # All tests
go test -cover ./... # With coverage
go test -race ./...  # Race detection

# Rust
cargo test           # All tests
cargo test -- --nocapture  # With output
cargo tarpaulin      # Coverage
```

### Build & Deploy Commands

```bash
# Node.js/TypeScript
npm run build        # Production build
npm run lint         # Lint check
npm run lint:fix     # Auto-fix lint issues
npm run typecheck    # TypeScript validation

# Python
black .              # Format
isort .              # Sort imports
mypy .               # Type check
ruff check .         # Lint

# Go
go build ./...       # Build all
go vet ./...         # Static analysis
golangci-lint run    # Comprehensive lint

# Rust
cargo build --release  # Production build
cargo clippy           # Lint
cargo fmt              # Format
```

### Environment Variables

```bash
# Copy example env file (never commit .env)
cp .env.example .env

# Required variables (set in .env)
DATABASE_URL=        # Database connection string
API_KEY=             # External service API key
SECRET_KEY=          # Application secret (generate random)
NODE_ENV=            # development | production | test
```

---

## Boundaries (Do Not Touch)

> **Critical**: AI agents must NOT modify these without explicit permission.

### Protected Files
- `package-lock.json`, `yarn.lock`, `Cargo.lock`, `go.sum` (dependency locks)
- `.env`, `.env.local`, `.env.production` (environment configs)
- Database migration files (once applied to production)
- `tsconfig.json`, `Cargo.toml`, `go.mod` (without testing build)
- `.github/workflows/`, CI/CD configurations

### Never Commit
- Secrets, API keys, credentials, tokens
- `.env` files (commit `.env.example` instead)
- `node_modules/`, `venv/`, `target/`, build artifacts
- Personal IDE configs (`.vscode/settings.json`, `.idea/`)
- Large binaries, datasets (use Git LFS)

### Ask Before Modifying
- Authentication/authorization logic
- Database schemas (after production deployment)
- Public API contracts (breaks consumers)
- Build/deploy processes
- Major dependency versions

---

## Quick Reference

**Task Classification:**
- **ATOMIC** (<5 files, clear scope) → Implement directly
- **FEATURE** (5-10 files) → Break into subtasks
- **COMPLEX** (>10 files, new subsystem) → Use PRD workflow | .claude/skills/create-prd/SKILL.md

**Common Guardrails** (validate first):
- Function ≤50 lines | File ≤300 lines | Input validation | Parameterized queries
- Tests >80% (critical) | Conventional commits | No secrets in code

**Autonomous Mode (Ralph Wiggum methodology):**
- Initialize: `samuel run init --prd .claude/tasks/NNNN-prd-feature.md`
- Start loop: `samuel run start`
- Check status: `samuel run status` (or just `samuel run` when a loop exists)
- List tasks: `samuel run tasks`
- Mutate tasks: `samuel run done|skip|reset <id>`
- Add tasks: `samuel run enqueue "<title>"` (auto-id) or `samuel run task add <id> <title>` (explicit, for CI)
- Methodology: .claude/skills/auto/SKILL.md
- Backward compat: `samuel auto` is a permanent alias for `samuel run` and works for every form above

**Emergency Quick Links:**
- Security issue? → .claude/skills/security-audit/SKILL.md
- Tests failing? → .claude/skills/troubleshooting/SKILL.md
- Stuck >30 min? → .claude/skills/troubleshooting/SKILL.md
- Complex feature? → .claude/skills/create-prd/SKILL.md
- Code review? → .claude/skills/code-review/SKILL.md
- Language-specific? → .claude/skills/{lang}-guide/SKILL.md

**Skills** (capability modules - [Agent Skills](https://agentskills.io) standard):
- Create: `samuel skill create <name>` or `.claude/skills/create-skill/SKILL.md`
- Validate: `samuel skill validate`
- List: `samuel skill list`
- Load: `.claude/skills/<skill-name>/SKILL.md` when task matches description

<!-- SKILLS_START -->
## Available Skills

Skills extend AI capabilities. Load a skill when task matches its description.

| Skill | Description |
|-------|-------------|
| algorithmic-art | Generative art creation using p5.js with seeded randomness. |
| auto | Autonomous AI coding loop (Ralph Wiggum methodology). |
| cleanup-project | Project cleanup and pruning workflow. |
| code-review | Pre-commit code quality review workflow. |
| commit-message | Generate descriptive commit messages by analyzing git diffs. |
| create-prd | Product Requirements Document (PRD) creation workflow. |
| create-rfd | Request for Discussion (RFD) creation workflow. |
| create-skill | Agent Skill creation workflow. |
| dependency-update | Safe dependency update workflow. |
| doc-coauthoring | Collaborative document writing workflow. |
| document-work | Work documentation and pattern capture workflow. |
| frontend-design | Design-thinking workflow for frontend interfaces. |
| generate-agents-md | Cross-tool compatibility workflow (AGENTS.md). |
| generate-tasks | Task generation and breakdown workflow from PRDs. |
| initialize-project | Project initialization and setup workflow. |
| mcp-builder | MCP server creation and integration guide. |
| refactoring | Technical debt remediation and code restructuring workflow. |
| security-audit | Security assessment workflow (OWASP, auth, vulnerabilities). |
| sync-claude-md | Sync per-folder CLAUDE.md/AGENTS.md with context-aware content. |
| testing-strategy | Test planning and coverage strategy workflow. |
| theme-factory | Toolkit for styling artifacts with pre-set or custom themes. |
| troubleshooting | Debugging and problem-solving workflow. |
| update-framework | Samuel version update workflow. |
| web-artifacts-builder | React/TypeScript/shadcn toolchain for web applications. |
| webapp-testing | Playwright-based web application testing workflow. |

**To use a skill**: Read `.claude/skills/<skill-name>/SKILL.md`
<!-- SKILLS_END -->

**Load Language Guide** (automatic based on file extensions):
- TypeScript/JavaScript → .claude/skills/typescript-guide/SKILL.md
- Python → .claude/skills/python-guide/SKILL.md
- Go → .claude/skills/go-guide/SKILL.md
- Rust → .claude/skills/rust-guide/SKILL.md
- Kotlin → .claude/skills/kotlin-guide/SKILL.md
- Java → .claude/skills/java-guide/SKILL.md
- C# → .claude/skills/csharp-guide/SKILL.md
- PHP → .claude/skills/php-guide/SKILL.md
- Swift → .claude/skills/swift-guide/SKILL.md
- C/C++ → .claude/skills/cpp-guide/SKILL.md
- Ruby → .claude/skills/ruby-guide/SKILL.md
- SQL → .claude/skills/sql-guide/SKILL.md
- Shell/Bash → .claude/skills/shell-guide/SKILL.md
- R → .claude/skills/r-guide/SKILL.md
- Dart/Flutter → .claude/skills/dart-guide/SKILL.md
- HTML/CSS → .claude/skills/html-css-guide/SKILL.md
- Lua → .claude/skills/lua-guide/SKILL.md
- Assembly → .claude/skills/assembly-guide/SKILL.md
- CUDA → .claude/skills/cuda-guide/SKILL.md
- Solidity → .claude/skills/solidity-guide/SKILL.md
- Zig → .claude/skills/zig-guide/SKILL.md

**Load Framework Skill** (when using specific framework):
- TypeScript/JavaScript: react, nextjs, express
- Python: django, fastapi, flask
- Go: gin, echo, fiber
- Rust: axum, actix-web, rocket
- Kotlin: spring-boot-kotlin, ktor, android-compose
- Java: spring-boot-java, quarkus, micronaut
- C#: aspnet-core, blazor, unity
- PHP: laravel, symfony, wordpress
- Swift: swiftui, uikit, vapor
- Ruby: rails, sinatra, hanami
- Dart: flutter, shelf, dart-frog

All at `.claude/skills/<name>/SKILL.md`

---

## Core Guardrails (ALWAYS ENFORCE)

### Code Quality
- No function exceeds 50 lines (split with helper functions)
- No file exceeds 300 lines (components: 200, tests: 300, utils: 150)
- Cyclomatic complexity ≤ 10 per function
- All exported functions have type signatures and documentation
- No magic numbers (use named constants)
- No commented-out code in commits (use git history)
- No `TODO` without issue/ticket reference
- No dead code (unused imports, variables, functions)

### Security (CRITICAL)
- All user inputs validated before processing
- All API boundaries have input validation (prefer schema validators: Zod, Pydantic, etc.)
- All database queries use parameterized statements (no string concatenation)
- All environment variables have secure defaults (never hardcode secrets)
- All file operations validate paths (prevent directory traversal)
- All async operations have timeout/cancellation mechanisms
- Dependencies checked for known vulnerabilities before adding
- Dependencies checked for license compatibility before adding
- All database migrations include rollback (down) function

### Testing (CRITICAL)
- Coverage targets: >80% for business logic, >60% overall
- All public APIs have unit tests
- All bug fixes include regression tests
- All edge cases explicitly tested (null, empty, boundary values)
- Test names describe behavior: `test_user_login_fails_with_invalid_password`
- No test interdependencies (tests run in any order)
- Integration tests for external service interactions
- All deployments include smoke test validation

### Git & Commits
- Commit messages: `type(scope): description` (conventional commits)
- Types: feat, fix, docs, refactor, test, chore, perf, ci
- One logical change per commit (atomic commits)
- All commits must pass tests before pushing
- Branch naming: `type/short-description` (e.g., `feat/user-auth`)
- No commits directly to main/master (use PRs)
- Breaking API changes require major version bump (Semantic Versioning)

### Performance
- No N+1 queries (batch database operations)
- Large datasets use pagination/streaming (not full loads)
- Expensive computations memoized/cached when appropriate
- Frontend bundles < 200KB initial load (code-split when needed)
- API responses < 200ms for simple queries, < 1s for complex

---

## 4D Methodology

Apply appropriate mode based on task complexity:

### ATOMIC Mode (Default)
For single-file changes, bug fixes, small features:

1. **Deconstruct**: What's the minimal change needed?
2. **Diagnose**: Will this break anything? Check dependencies.
3. **Develop**: Make the change with tests.
4. **Deliver**: Validate (run tests, check guardrails) → Commit.

### FEATURE Mode
For multi-file features, new components, API endpoints:

1. **Deconstruct**: Break into 3-5 subtasks (each atomic).
2. **Diagnose**: Identify integration points and dependencies.
3. **Develop**: Implement subtasks sequentially with tests.
4. **Deliver**: Integration test → Documentation → Review → Commit.

### COMPLEX Mode
For architecture changes, major refactors, new systems:

1. **Deconstruct**: Full decomposition into phases/milestones.
2. **Diagnose**: Analyze risks, dependencies, migration paths.
3. **Develop**: Plan implementation → Execute incrementally.
4. **Deliver**: Staged rollout → Documentation → Retrospective.

**Workflow for COMPLEX tasks:**
1. Use `.claude/skills/create-prd/SKILL.md` to define requirements
2. Use `.claude/skills/generate-tasks/SKILL.md` to break down implementation
3. Implement tasks step-by-step with verification checkpoints

**Autonomous Execution (Optional):**
After task generation, convert to autonomous format for unattended execution:

1. `samuel run init --prd .claude/tasks/NNNN-prd-feature.md`
2. Review generated `prd.json` and `prompt.md`
3. `samuel run start`

See `.claude/skills/auto/SKILL.md` for the full methodology.

**Escalation Triggers:**
- Task affects >5 files → FEATURE mode
- Task affects >10 files → COMPLEX mode (consider PRD workflow)
- Task affects >15 files OR new subsystem → COMPLEX mode (PRD workflow MANDATORY)
- Task unclear/ambiguous → Ask user for clarification first

---

## Software Development Lifecycle

### Stage 1: Planning
**Atomic**: Read existing code → Identify change location
**Feature**: Review related code → Sketch interfaces/contracts
**Complex**: Use PRD workflow

### Stage 2: Implementation
- Write tests first (TDD) or alongside code
- Load language-specific guide: .claude/skills/{language}-guide/SKILL.md
- Follow language/framework idioms
- Validate against guardrails continuously

### Stage 3: Validation
- Run full test suite
- Check coverage thresholds
- Run linter/formatter
- Verify build succeeds

### Stage 4: Documentation
- Function/class docstrings (what, why, params, returns)
- Inline comments for complex logic (why, not what)
- API documentation updated

### Stage 5: Commit
```bash
git add <files>
git commit -m "type(scope): description

- Detail 1
- Detail 2

Refs: #issue-number"
```

---

## Per-Folder CLAUDE.md

This project uses **hierarchical CLAUDE.md files**. Each folder can have its own `CLAUDE.md` with folder-specific instructions that AI agents load on demand.

**When creating new directories**, create a `CLAUDE.md` with:
- Purpose of the folder
- Conventions specific to that folder
- Key patterns or constraints

AI agents automatically discover and load these files when working in subdirectories.

---

## Project Context

> **Instructions**: Fill in this section when the tech stack and architecture are decided.
> This replaces the need for a separate project.md file.

### Tech Stack
<!-- Fill in when tech decisions are made -->
- **Language**: [e.g., TypeScript 5.3, Python 3.11, Go 1.21]
- **Framework**: [e.g., React 18, Django 5, Gin]
- **Database**: [e.g., PostgreSQL 15, MongoDB 7]
- **Infrastructure**: [e.g., Vercel, AWS, Docker]

### Architecture
<!-- Fill in when architecture is decided -->
- **Type**: [e.g., Monolith, Microservices, Serverless]
- **Key Patterns**: [e.g., Repository pattern, Service layer, DI]

### Key Design Decisions
<!-- Add decisions as they're made -->
<!-- Format: **Decision**: [What] | **Date**: [When] | **Rationale**: [Why] -->

---

## Anti-Patterns (Avoid These)

### Code
- Premature optimization (measure first, optimize after)
- Over-engineering (YAGNI: You Aren't Gonna Need It)
- Copy-paste code (extract to shared function/component)
- Ignoring errors (every error needs handling)

### Testing
- Testing implementation details (test behavior, not internals)
- Flaky tests (non-deterministic results indicate bad design)
- Test interdependencies (each test should be isolated)

### Process
- Committing directly to main (use feature branches + PRs)
- Batch commits (commit after each logical change)
- Skipping tests because "it's a small change"

---

## When Stuck

**See:** `.claude/skills/troubleshooting/SKILL.md`

**Quick recovery:**
1. STOP trying random solutions (>30 min = stuck)
2. Document what you've tried
3. Simplify & isolate (minimal reproduction)
4. Check fundamentals (dependencies, config, versions)
5. Ask user with clear problem statement

---

## Version & Changelog

**Current Version**: 2.0.0
**Last Updated**: 2026-02-11

### Changelog

**v2.0.0 (2026-02-11) - Native Claude Code Integration**
- Migrated from `.agent/` to `.claude/` (native Claude Code directory)
- Merged AI_INSTRUCTIONS.md + CLAUDE.md + project.md into single CLAUDE.md
- Skills now live in `.claude/skills/` (native skill discovery)
- Added AGENTS.md as real copy for cross-tool compatibility
- Added per-folder CLAUDE.md support (hierarchical instructions)
- Dropped `.agent/memory/`, `.agent/tasks/`, `.agent/rfd/`, `.agent/state.md`
- Uses Claude Code's built-in memory system instead

**v1.8.0 (2025-02-04) - Agent Skills Integration**
- Added Agent Skills support (open standard for AI agent capabilities)
- New CLI commands: `samuel skill create`, `samuel skill validate`, `samuel skill list`, `samuel skill info`
- Skills compatible with 25+ agent products

---

**Remember**: This file is your guardrails. Small atomic changes. Validate continuously.

**Cross-Tool Compatibility**: AGENTS.md is a copy of this file for tools that read AGENTS.md.
