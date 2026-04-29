# CLAUDE.md

AI-assisted development instructions for the Samuel CLI project.

> **AGENTS.md Compatible**: A copy exists as `AGENTS.md` for cross-tool compatibility (Cursor, Codex, Copilot, etc.)

---

## Operations

### Setup Commands

```bash
git clone https://github.com/ar4mirez/samuel.git && cd samuel
go mod download
make build            # → ./bin/samuel
./bin/samuel version   # Verify installation
```

### Testing Commands

```bash
make test             # Run all tests (verbose, race detection, coverage)
go test ./...         # Quick run
go test -cover ./...  # With coverage
go test -race ./...   # Race detection only
```

### Build & Lint Commands

```bash
make build            # Build binary to ./bin/samuel
go build ./...        # Build all packages
go vet ./...          # Static analysis
golangci-lint run     # Comprehensive lint
go fmt ./...          # Format code
```

### Documentation

```bash
pip install -r requirements-docs.txt
mkdocs serve          # Local preview at http://127.0.0.1:8000
mkdocs build          # Build static site to ./site/
```

---

## Boundaries (Do Not Touch)

### Protected Files
- `go.sum` (dependency lock)
- `.github/workflows/` (CI/CD configurations)
- `.goreleaser.yaml` (release automation)
- `go.mod` (without testing build)

### Never Commit
- Secrets, API keys, credentials, tokens
- `.env` files
- `bin/`, build artifacts, the root `samuel` binary
- `.claude/settings.local.json` (personal settings)

### Ask Before Modifying
- Template files in `template/` (affects all users)
- Public CLI command interfaces (breaks user scripts)
- Registry data in `internal/core/registry.go` (component definitions)
- Build/deploy processes (Makefile, .goreleaser.yaml)

---

## Project Context

### Tech Stack
- **Language**: Go 1.21+
- **CLI Framework**: Cobra
- **UI**: fatih/color, manifoldco/promptui, schollz/progressbar
- **Config**: gopkg.in/yaml.v3
- **HTTP**: net/http (standard library)
- **Archive**: archive/tar, compress/gzip (standard library)
- **Docs**: MkDocs with Material theme (Python)

### Architecture

```
samuel/
├── cmd/samuel/              # Entry point (main.go) — minimal
├── internal/
│   ├── commands/            # 14 CLI commands (init, update, add, remove, list, doctor, version, search, info, config, diff, skill, auto, sync)
│   ├── core/                # Business logic (config, registry, extractor, skill, auto)
│   ├── github/              # GitHub API client (releases, archives)
│   ├── orchestrator/        # v4 component lifecycle (Component interface, structured Error, flock concurrent init lock) — internal foundation, no CLI surface yet
│   └── ui/                  # User interface helpers (prompts, spinner, colors)
├── template/                # Distributable template files
│   ├── CLAUDE.md            # Template AI instructions
│   ├── AGENTS.md            # Cross-tool copy
│   └── .claude/skills/      # 21 language + 33 framework + 25 workflow/utility skills
├── docs/                    # MkDocs documentation source
├── Makefile                 # Build targets (build, test, lint, install)
└── .goreleaser.yaml         # Release automation
```

**Core Flow**: User command → GitHub download → Tar extraction → Local files
**Caching**: Downloaded versions cached in `~/.cache/samuel/`
**Config**: Project config in `samuel.yaml`

### Key Design Decisions

- **template/ directory**: Separates distributable files from project source. CLI paths use `template/` prefix for extraction.
- **GitHub archive downloads**: Uses GitHub's tar.gz API — no git required on user machines.
- **Standard Go layout**: `cmd/` + `internal/` at root. `internal/commands/` (not `cmd/`) to avoid confusion.
- **Go for CLI**: Single binary, cross-platform, no runtime dependencies. Cobra for CLI framework.
- **Auto (Ralph Wiggum)**: Autonomous AI coding loop. Tasks stored in `.claude/auto/prd.json`, progress in `progress.md`, orchestrated by Go-native loop. Supports PRD-based and zero-setup pilot modes.

---

## Quick Reference

**Task Classification:**
- **ATOMIC** (<5 files, clear scope) → Implement directly
- **FEATURE** (5-10 files) → Break into subtasks
- **COMPLEX** (>10 files, new subsystem) → Use PRD workflow | .claude/skills/create-prd/SKILL.md

**Common Guardrails** (validate first):
- Function ≤50 lines | File ≤300 lines | Input validation | Parameterized queries
- Tests >80% (critical) | Conventional commits | No secrets in code

**Emergency Quick Links:**
- Security issue? → .claude/skills/security-audit/SKILL.md
- Tests failing? → .claude/skills/troubleshooting/SKILL.md
- Stuck >30 min? → .claude/skills/troubleshooting/SKILL.md
- Complex feature? → .claude/skills/create-prd/SKILL.md
- Code review? → .claude/skills/code-review/SKILL.md
- Go-specific? → .claude/skills/go-guide/SKILL.md

**Skills** (capability modules - [Agent Skills](https://agentskills.io) standard):
- Create: `samuel skill create <name>` or `.claude/skills/create-skill/SKILL.md`
- Validate: `samuel skill validate`
- List: `samuel skill list`
- Load: `.claude/skills/<skill-name>/SKILL.md` when task matches description

**Autonomous Mode (Ralph Wiggum methodology):**

- Initialize: `samuel auto init --prd .claude/tasks/NNNN-prd-feature.md`
- Start loop: `samuel auto start`
- Check status: `samuel auto status`
- Manage tasks: `samuel auto task list|complete|skip|reset|add`
- Methodology: `.claude/skills/auto/SKILL.md`

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
| go-guide | Go language guardrails, patterns, and best practices. |
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
- All API boundaries have input validation
- All file operations validate paths (prevent directory traversal)
- All async operations have timeout/cancellation mechanisms
- Dependencies checked for known vulnerabilities before adding
- Dependencies checked for license compatibility before adding

### Testing (CRITICAL)
- Coverage targets: >80% for business logic, >60% overall
- All public APIs have unit tests
- All bug fixes include regression tests
- All edge cases explicitly tested (null, empty, boundary values)
- Test names describe behavior: `TestComputeDiff_Sorting`
- No test interdependencies (tests run in any order)
- Table-driven tests for functions with multiple scenarios

### Git & Commits
- Commit messages: `type(scope): description` (conventional commits)
- Types: feat, fix, docs, refactor, test, chore, perf, ci
- One logical change per commit (atomic commits)
- All commits must pass tests before pushing
- Branch naming: `type/short-description` (e.g., `feat/user-auth`)
- Breaking API changes require major version bump (Semantic Versioning)

---

## 4D Methodology

### ATOMIC Mode (Default)
1. **Deconstruct**: What's the minimal change needed?
2. **Diagnose**: Will this break anything? Check dependencies.
3. **Develop**: Make the change with tests.
4. **Deliver**: Validate (run tests, check guardrails) → Commit.

### FEATURE Mode
1. **Deconstruct**: Break into 3-5 subtasks (each atomic).
2. **Diagnose**: Identify integration points and dependencies.
3. **Develop**: Implement subtasks sequentially with tests.
4. **Deliver**: Integration test → Documentation → Review → Commit.

### COMPLEX Mode
1. **Deconstruct**: Full decomposition into phases/milestones.
2. **Diagnose**: Analyze risks, dependencies, migration paths.
3. **Develop**: Plan implementation → Execute incrementally.
4. **Deliver**: Staged rollout → Documentation → Retrospective.

**Workflow for COMPLEX tasks:**
1. Use `.claude/skills/create-prd/SKILL.md` to define requirements
2. Use `.claude/skills/generate-tasks/SKILL.md` to break down implementation
3. Implement tasks step-by-step with verification checkpoints

**Escalation Triggers:**
- Task affects >5 files → FEATURE mode
- Task affects >10 files → COMPLEX mode (consider PRD workflow)
- Task affects >15 files OR new subsystem → COMPLEX mode (PRD workflow MANDATORY)

---

## Software Development Lifecycle

### Stage 1: Planning
**Atomic**: Read existing code → Identify change location
**Feature**: Review related code → Sketch interfaces/contracts
**Complex**: Use PRD workflow

### Stage 2: Implementation
- Write tests first (TDD) or alongside code
- Load Go guide: .claude/skills/go-guide/SKILL.md
- Follow Go idioms (error handling, naming, packages)
- Validate against guardrails continuously

### Stage 3: Validation
- `make test` — full test suite with race detection
- `go vet ./...` — static analysis
- `go build ./...` — verify build

### Stage 4: Documentation
- Exported function godocs (what, why, params, returns)
- Inline comments for complex logic (why, not what)

### Stage 5: Commit
```bash
git add <files>
git commit -m "type(scope): description"
```

---

## Anti-Patterns (Avoid These)

- Premature optimization (measure first, optimize after)
- Over-engineering (YAGNI)
- Copy-paste code (extract to shared function)
- Ignoring errors (every error needs handling)
- Testing implementation details (test behavior, not internals)
- Flaky tests (non-deterministic = bad design)
- Batch commits (commit after each logical change)
- Skipping tests because "it's a small change"

---

## When Stuck

**See:** `.claude/skills/troubleshooting/SKILL.md`

1. STOP trying random solutions (>30 min = stuck)
2. Document what you've tried
3. Simplify & isolate (minimal reproduction)
4. Check fundamentals (dependencies, config, versions)
5. Ask user with clear problem statement

---

**Remember**: This file is your guardrails. Small atomic changes. Validate continuously.

**Cross-Tool Compatibility**: AGENTS.md is a copy of this file.

## gstack (REQUIRED — global install)

**Before doing ANY work, verify gstack is installed:**

```bash
test -d ~/.claude/skills/gstack/bin && echo "GSTACK_OK" || echo "GSTACK_MISSING"
```

If GSTACK_MISSING: STOP. Do not proceed. Tell the user:

> gstack is required for all AI-assisted work in this repo.
> Install it:
> ```bash
> git clone --depth 1 https://github.com/garrytan/gstack.git ~/.claude/skills/gstack
> cd ~/.claude/skills/gstack && ./setup --team
> ```
> Then restart your AI coding tool.

Do not skip skills, ignore gstack errors, or work around missing gstack.

Using gstack skills: After install, skills like /qa, /ship, /review, /investigate,
and /browse are available. Use /browse for all web browsing.
Use ~/.claude/skills/gstack/... for gstack file paths (the global path).
