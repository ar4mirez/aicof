# Samuel — the autonomous AI coding loop your CLI needs

> **`samuel run` and walk away.**
> Ralph Wiggum methodology • Cross-tool (Claude Code, Cursor, Codex, Copilot) • Opinionated guardrails baked in

[![Version](https://img.shields.io/badge/version-3.0.0--pre-blue.svg)](CHANGELOG.md)
[![AGENTS.md](https://img.shields.io/badge/AGENTS.md-compatible-brightgreen.svg)](https://agents.md)
[![Status](https://img.shields.io/badge/status-production%20ready-brightgreen.svg)](CLAUDE.md)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

---

## What is Samuel?

Samuel is a Go CLI that ships an autonomous AI coding loop plus a library of language, framework, and workflow guides that any AI tool can read. The differentiator is `samuel run` — point it at a structured task list, walk away, and let the loop pick tasks, implement them, run tests, and commit. Across multiple fresh context windows. With your guardrails enforced.

Underneath, Samuel is a versioned framework manager: it installs an opinionated CLAUDE.md + skill set into your project, keeps it migrated as the upstream evolves, and ships the same content as AGENTS.md so non-Claude tools work too.

---

## What's New in v3.0.0 (in flight)

v3.0.0 is the lean reshape. Five of seven PRs have landed on `main`; tag drops when PRs 6 and 7 land.

- **`samuel run`** — autonomous loop renamed from `samuel auto` (which stays as a permanent alias). Smart bare invocation: status if a loop exists, actionable help if not — never silently starts.
- **`samuel ls`** — single discovery verb collapsing v2's `list` + `search` + `info`. `samuel ls react --detail` infers the type.
- **`samuel add react`** — type inference. The 1-arg form works for any unambiguous component name.
- **`samuel rm`** — permanent alias for `remove`.
- **`samuel admin <verb>`** — power-user commands (`config`, `sync`, `diff`) grouped together. `doctor` and `skill` stay top-level.
- **JSON `schemaVersion: 3`** — every `--json` envelope carries it. The `command` field reflects the invoked path.
- **"Did you mean?"** — Cobra suggestions for typos (`samuel adminn` → `samuel admin`).
- **`SAMUEL_NO_DEPRECATION=1`** — env var (and `--no-deprecation` flag) silence the one-line redirect that legacy aliases print.

**v2 scripts keep working** through v3.0.x. See [Migration Guide](docs/getting-started/migration-v3.md) for the v2→v3 mapping.

---

## Quick Start (60 Seconds)

### Option 1: Using the CLI (Recommended)

```bash
# Install CLI
brew tap ar4mirez/tap
brew install samuel

# Or via curl
curl -sSL https://raw.githubusercontent.com/ar4mirez/samuel/main/install.sh | sh

# Initialize your project
samuel init my-project
cd my-project

# Explore available components
samuel ls --all              # List everything available
samuel ls react              # Search "react"
samuel ls react --detail     # Full info on a component (type inferred)

# Add components (type inferred)
samuel add typescript        # Adds the TypeScript language guide
samuel add react             # Adds the React framework guide
samuel add code-review       # Adds the code-review workflow

# Start an autonomous coding loop
samuel run init              # Bootstrap the loop (or pass a PRD: 'samuel run init <path>')
samuel run start             # Begin
samuel run                   # Bare run = status when a loop exists

# Or start coding with AI yourself — guardrails apply automatically.
```

### Option 2: Manual Copy

```bash
# 1. Copy template files to your project
cp -r /path/to/samuel/template/{CLAUDE.md,AGENTS.md,.claude} ./

# 2. Start coding with AI - guardrails apply automatically!
```

**The system works immediately:**

- AI loads [CLAUDE.md](template/CLAUDE.md) automatically (500 lines of guardrails + operations)
- Language guides auto-load based on file extensions
- Workflows available when you need them
- Progressive - starts minimal, grows with your project

**[Read the full Quick Start Guide](CLAUDE.md)**

---

## Cross-Tool Compatibility (AGENTS.md)

This system follows the [AGENTS.md](https://agents.md) standard - the universal format for AI coding assistants adopted by 20,000+ repositories.

### How It Works

| Tool | Primary File | Fallback |
|------|--------------|----------|
| **Claude Code** | CLAUDE.md | AGENTS.md |
| **Cursor** | AGENTS.md | CLAUDE.md |
| **OpenAI Codex** | AGENTS.md | CLAUDE.md |
| **GitHub Copilot** | AGENTS.md | CLAUDE.md |
| **Google Jules** | AGENTS.md | CLAUDE.md |

### Setup for Cross-Tool Teams

**Option 1: Real copy (recommended)**
`samuel init` creates both `CLAUDE.md` and `AGENTS.md` as real files automatically.

**Option 2: Symlink**
```bash
ln -s CLAUDE.md AGENTS.md
```

**Option 3: Generate standalone AGENTS.md**
```
@.claude/skills/generate-agents-md/SKILL.md
```

**Why both files?**
- `CLAUDE.md` = Full methodology (guardrails + 4D + workflows)
- `AGENTS.md` = Operations only (commands, boundaries, style)

---

## What Is Samuel?

Samuel (Artificial Intelligence Coding Framework) is an **opinionated AI development framework** designed for professional software teams.

### Key Features

| Feature | Description |
|---------|-------------|
| **35+ Guardrails** | Testable rules, not vague suggestions |
| **21 Language Guides** | All major languages with auto-loading support |
| **33 Framework Skills** | Framework-specific patterns and best practices |
| **24 Workflows** | PRD, tasks, init, troubleshooting, code-review, sync, and more |
| **3 Modes** | ATOMIC/FEATURE/COMPLEX (scales from bugs to architecture) |
| **4D Methodology** | Deconstruct → Diagnose → Develop → Deliver |
| **Cross-Tool** | Works with any AI coding assistant |

### Philosophy

> Small, validated changes. Quality enforced. Documentation grows organically.

---

## CLI Commands (v3)

The `samuel` CLI manages framework installation, updates, component discovery, and the autonomous loop.

### Core Commands

| Command | Description | Example |
|---------|-------------|---------|
| `init [project]` | Initialize Samuel in a project | `samuel init my-app` |
| `update [--preview]` | Update framework (or preview changes) | `samuel update --preview` |
| `doctor` | Check installation health | `samuel doctor` |
| `version` | Show CLI and framework versions | `samuel version` |

### Components

| Command | Description | Example |
|---------|-------------|---------|
| `add <name> [type]` | Add a component (type inferred when unambiguous) | `samuel add react` |
| `rm <name> [type]` | Remove a component (alias of `remove`) | `samuel rm rust` |
| `ls [query]` | List, search, or detail (collapses v2 list/search/info) | `samuel ls react --detail` |

`samuel ls` supports `--all` (everything available), `--type framework`, `--detail` (info-style), `--preview N`, `--no-related`, and `--limit N`. Type aliases (`lang`/`l`, `fw`/`f`, `wf`/`w`) are still accepted.

### Autonomous loop (Ralph Wiggum)

| Command | Description | Example |
|---------|-------------|---------|
| `run` (bare) | Status if loop exists, actionable help otherwise | `samuel run` |
| `run init [prd]` | Bootstrap the loop | `samuel run init .claude/tasks/0001-prd.md` |
| `run start` | Begin or resume | `samuel run start` |
| `run pilot` | Zero-setup discover-and-implement | `samuel run pilot` |
| `run status` | Show progress | `samuel run status` |
| `run tasks` | List tasks | `samuel run tasks` |
| `run done <id>` | Mark complete | `samuel run done 1.1` |
| `run skip <id>` | Mark skipped | `samuel run skip 1.1` |
| `run reset <id>` | Reset to pending | `samuel run reset 1.1` |
| `run enqueue <title>` | Add task with auto-id | `samuel run enqueue "Add metrics"` |
| `run task add <id> <title>` | Add task with explicit id (CI) | `samuel run task add 5.0 "Title"` |

`samuel auto` is a permanent alias for `samuel run` — every form above works with either name forever.

### Power-user (admin)

| Command | Description | Example |
|---------|-------------|---------|
| `admin config <key>` | Get a config value | `samuel admin config get version` |
| `admin config set <key> <value>` | Set a config value | `samuel admin config set registry https://...` |
| `admin sync` | Sync per-folder CLAUDE.md/AGENTS.md | `samuel admin sync --dry-run` |
| `admin diff [v1] [v2]` | Compare two specific versions | `samuel admin diff v2.5.0 v2.6.0` |

**Valid config keys**: `version`, `registry`, `installed.languages`, `installed.frameworks`, `installed.workflows`

### Skills (authoring)

| Command | Description | Example |
|---------|-------------|---------|
| `skill list` | List installed skills | `samuel skill list` |
| `skill info <name>` | Show skill detail | `samuel skill info commit-message` |
| `skill validate [name]` | Validate skill schema | `samuel skill validate` |
| `skill create <name>` | Scaffold a new skill | `samuel skill create my-skill` |

### Examples

```bash
# Initialize and customize
samuel init my-project
samuel add typescript           # Type inferred (language)
samuel add react                # Type inferred (framework)
samuel add code-review          # Type inferred (workflow)

# Discover components
samuel ls react                 # Search "react"
samuel ls react --detail        # Full info, type auto-detected
samuel ls --type framework      # All installed frameworks
samuel ls --all                 # Everything available

# Manage updates
samuel update --preview         # Compare installed vs latest
samuel admin diff v2.5.0 v2.6.0 # Compare two specific versions
samuel update                   # Apply updates

# Run the autonomous loop
samuel run init                 # Bootstrap with the convention defaults
samuel run start                # Begin
samuel run                      # Bare = status check
samuel run done 1.1             # Mark task complete
samuel run enqueue "Add tests"  # Add a task (auto-id)

# Troubleshoot
samuel doctor                   # Check installation health
samuel admin config get version # View configuration
```

### v2 commands

Every v2 command keeps working through the v3.0.x window. Aliases print a one-line redirect on stderr; suppress with `SAMUEL_NO_DEPRECATION=1` or `--no-deprecation`. Aliases remove in v3.1.0. See the [Migration Guide](docs/getting-started/migration-v3.md) for the full v2→v3 mapping.

### JSON output

All commands accept `--json` for programmatic consumption. The envelope reports `schemaVersion: 3` and the `command` field reflects the invoked path:

```json
{
  "schemaVersion": 3,
  "command": "ls",
  "success": true,
  "data": { ... }
}
```

Pin consumers to `schemaVersion` for forward compatibility.

---

## Professional Prompt Examples

### Bug Fixes (ATOMIC Mode)

```
Fix the null pointer exception in UserService.getProfile() when user.email is undefined
```

```
The checkout button is disabled after form validation passes - debug and fix
```

```
Memory leak in useWebSocket hook - component doesn't cleanup on unmount
```

### Feature Development (FEATURE Mode)

```
Add pagination to the /api/users endpoint with cursor-based navigation
```

```
Implement email verification flow: send verification link, validate token, update user status
```

```
Create a reusable DataTable component with sorting, filtering, and column resizing
```

### Complex Features (COMPLEX Mode)

```
@.claude/skills/create-prd/SKILL.md
Build a real-time notification system with WebSocket support, push notifications,
and user preference management
```

```
@.claude/skills/create-prd/SKILL.md
Implement multi-tenant architecture: tenant isolation, data partitioning,
tenant-specific configurations
```

```
@.claude/skills/create-prd/SKILL.md
Add comprehensive audit logging: user actions, data changes, security events,
with retention policies and export functionality
```

### Code Review & Analysis

```
Review the authentication module for security vulnerabilities and suggest improvements
```

```
Analyze the database query patterns in OrderService and identify N+1 query issues
```

```
Evaluate the error handling strategy in the API layer and propose a consistent approach
```

### Refactoring

```
Refactor UserController to use dependency injection and separate business logic into a service layer
```

```
Extract common validation logic from all form components into a reusable validation hook
```

```
Migrate the codebase from callbacks to async/await while maintaining backward compatibility
```

### Architecture & Planning

```
@.claude/skills/create-prd/SKILL.md
Design a caching strategy for the product catalog: cache invalidation,
distributed caching, cache warming
```

```
Analyze the current monolith and propose a microservices decomposition plan
with clear service boundaries
```

```
Create a database migration strategy for adding multi-region support
without downtime
```

### Debugging & Troubleshooting

```
@.claude/skills/troubleshooting/SKILL.md
Production error: "Connection pool exhausted" appearing intermittently under load
```

```
Performance degradation: API response times increased from 50ms to 500ms
after last deployment
```

```
Investigate why the CI pipeline is flaky - tests pass locally but fail
randomly in GitHub Actions
```

### Testing

```
Generate comprehensive unit tests for the PaymentService with edge cases
for failed transactions, refunds, and currency conversion
```

```
Create integration tests for the OAuth flow covering Google, GitHub,
and email/password authentication
```

```
Add E2E tests for the checkout flow using Playwright, including
error scenarios and payment failures
```

### Documentation

```
Generate API documentation for the /api/v2/orders endpoint including
request/response examples and error codes
```

```
Create a technical design document for the upcoming search feature
with architecture diagrams
```

### DevOps & Infrastructure

```
Create a Dockerfile for the Node.js API with multi-stage build,
non-root user, and health checks
```

```
Write GitHub Actions workflow for: lint, test, build, deploy to staging
on PR, deploy to production on merge
```

---

## How It Works

### The 3 Modes

**ATOMIC** (<5 files, clear scope)
```
"Fix the login button alignment"
```
- Direct implementation
- Quick validation
- One commit
- Examples: Bug fixes, styling, simple features

**FEATURE** (5-10 files)
```
"Add user profile editing with avatar upload"
```
- Break into 3-5 subtasks
- Implement sequentially
- Integration testing
- Examples: New component, API endpoint, refactoring

**COMPLEX** (>10 files, new subsystem)
```
@.claude/skills/create-prd/SKILL.md
"Build user authentication with OAuth"
```
- Create PRD (Product Requirements Document)
- Generate task breakdown
- Step-by-step implementation
- Examples: Authentication, payments, analytics

**AI auto-detects which mode to use.**

### The Guardrails (35+ Rules)

**Code Quality:**
- Functions ≤50 lines
- Files ≤300 lines
- Complexity ≤10 per function
- All exports have types/docs

**Security (CRITICAL):**
- All inputs validated
- Parameterized queries only
- No secrets in code
- Dependencies checked for vulnerabilities + licenses

**Testing (CRITICAL):**
- >80% coverage for business logic
- >60% overall coverage
- Tests for all public APIs
- Regression tests for bugs

**Git:**
- Conventional commits (`feat:`, `fix:`, etc.)
- One logical change per commit
- All tests pass before push
- PRs required (no direct commits to main)

**[See all guardrails in CLAUDE.md](CLAUDE.md)**

### The 4D Methodology

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ DECONSTRUCT │ ──▶ │  DIAGNOSE   │ ──▶ │   DEVELOP   │ ──▶ │   DELIVER   │
│             │     │             │     │             │     │             │
│ Break down  │     │ Identify    │     │ Implement   │     │ Validate    │
│ the task    │     │ risks &     │     │ with tests  │     │ & commit    │
│             │     │ dependencies│     │             │     │             │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

---

## Use Cases

### Starting a New Project

```
@.claude/skills/initialize-project/SKILL.md
"Initialize a new TypeScript API with Express, PostgreSQL, and Jest"
```

AI will:
1. Ask clarifying questions (architecture, deployment target)
2. Create directory structure
3. Generate config files (tsconfig, package.json, etc.)
4. Set up testing framework

### Onboarding to Existing Project

```
@.claude/skills/initialize-project/SKILL.md
"This is an existing project - analyze the codebase and document patterns"
```

AI will:
1. Scan tech stack (package.json, requirements.txt, etc.)
2. Analyze code patterns and conventions
3. Review recent commits
4. Document findings and patterns

### Building Complex Features

```
@.claude/skills/create-prd/SKILL.md
"Build a subscription billing system with Stripe integration"
```

AI will:
1. Ask clarifying questions (plans, trial periods, webhooks)
2. Create comprehensive PRD
3. Generate task breakdown (20-50 subtasks)
4. Implement step-by-step with verification
5. Update documentation

### Cross-Tool Team Workflow

```
@.claude/skills/generate-agents-md/SKILL.md
"Generate AGENTS.md for our team using Cursor and Claude Code"
```

AI will:
1. Extract Operations section from CLAUDE.md
2. Generate universal AGENTS.md
3. Ensure both tools work consistently

---

## Documentation

| Document | Purpose | When to Read |
|----------|---------|--------------|
| [CLAUDE.md](template/CLAUDE.md) | Core guardrails & methodology | Read this first / AI loads automatically |

### Language Guides (Auto-Load)

| Language | Files | Guide |
|----------|-------|-------|
| TypeScript/JavaScript | `.ts`, `.tsx`, `.js`, `.jsx` | [SKILL.md](template/.claude/skills/typescript-guide/SKILL.md) |
| Python | `.py` | [SKILL.md](template/.claude/skills/python-guide/SKILL.md) |
| Go | `.go` | [SKILL.md](template/.claude/skills/go-guide/SKILL.md) |
| Rust | `.rs` | [SKILL.md](template/.claude/skills/rust-guide/SKILL.md) |
| Kotlin | `.kt`, `.kts` | [SKILL.md](template/.claude/skills/kotlin-guide/SKILL.md) |
| Java | `.java` | [SKILL.md](template/.claude/skills/java-guide/SKILL.md) |
| C# | `.cs` | [SKILL.md](template/.claude/skills/csharp-guide/SKILL.md) |
| PHP | `.php` | [SKILL.md](template/.claude/skills/php-guide/SKILL.md) |
| Swift | `.swift` | [SKILL.md](template/.claude/skills/swift-guide/SKILL.md) |
| C/C++ | `.c`, `.cpp`, `.h`, `.hpp` | [SKILL.md](template/.claude/skills/cpp-guide/SKILL.md) |
| Ruby | `.rb` | [SKILL.md](template/.claude/skills/ruby-guide/SKILL.md) |
| SQL | `.sql` | [SKILL.md](template/.claude/skills/sql-guide/SKILL.md) |
| Shell/Bash | `.sh`, `.bash` | [SKILL.md](template/.claude/skills/shell-guide/SKILL.md) |
| R | `.r`, `.R` | [SKILL.md](template/.claude/skills/r-guide/SKILL.md) |
| Dart/Flutter | `.dart` | [SKILL.md](template/.claude/skills/dart-guide/SKILL.md) |
| HTML/CSS | `.html`, `.css`, `.scss` | [SKILL.md](template/.claude/skills/html-css-guide/SKILL.md) |
| Lua | `.lua` | [SKILL.md](template/.claude/skills/lua-guide/SKILL.md) |
| Assembly | `.asm`, `.s` | [SKILL.md](template/.claude/skills/assembly-guide/SKILL.md) |
| CUDA | `.cu`, `.cuh` | [SKILL.md](template/.claude/skills/cuda-guide/SKILL.md) |
| Solidity | `.sol` | [SKILL.md](template/.claude/skills/solidity-guide/SKILL.md) |
| Zig | `.zig` | [SKILL.md](template/.claude/skills/zig-guide/SKILL.md) |

### Framework Skills (On-Demand)

| Language | Frameworks |
|----------|------------|
| TypeScript/JS | [React](template/.claude/skills/react/SKILL.md), [Next.js](template/.claude/skills/nextjs/SKILL.md), [Express](template/.claude/skills/express/SKILL.md) |
| Python | [Django](template/.claude/skills/django/SKILL.md), [FastAPI](template/.claude/skills/fastapi/SKILL.md), [Flask](template/.claude/skills/flask/SKILL.md) |
| Go | [Gin](template/.claude/skills/gin/SKILL.md), [Echo](template/.claude/skills/echo/SKILL.md), [Fiber](template/.claude/skills/fiber/SKILL.md) |
| Rust | [Axum](template/.claude/skills/axum/SKILL.md), [Actix-web](template/.claude/skills/actix-web/SKILL.md), [Rocket](template/.claude/skills/rocket/SKILL.md) |
| Kotlin | [Spring Boot](template/.claude/skills/spring-boot-kotlin/SKILL.md), [Ktor](template/.claude/skills/ktor/SKILL.md), [Android Compose](template/.claude/skills/android-compose/SKILL.md) |
| Java | [Spring Boot](template/.claude/skills/spring-boot-java/SKILL.md), [Quarkus](template/.claude/skills/quarkus/SKILL.md), [Micronaut](template/.claude/skills/micronaut/SKILL.md) |
| C# | [ASP.NET Core](template/.claude/skills/aspnet-core/SKILL.md), [Blazor](template/.claude/skills/blazor/SKILL.md), [Unity](template/.claude/skills/unity/SKILL.md) |
| PHP | [Laravel](template/.claude/skills/laravel/SKILL.md), [Symfony](template/.claude/skills/symfony/SKILL.md), [WordPress](template/.claude/skills/wordpress/SKILL.md) |
| Swift | [SwiftUI](template/.claude/skills/swiftui/SKILL.md), [UIKit](template/.claude/skills/uikit/SKILL.md), [Vapor](template/.claude/skills/vapor/SKILL.md) |
| Ruby | [Rails](template/.claude/skills/rails/SKILL.md), [Sinatra](template/.claude/skills/sinatra/SKILL.md), [Hanami](template/.claude/skills/hanami/SKILL.md) |
| Dart | [Flutter](template/.claude/skills/flutter/SKILL.md), [Shelf](template/.claude/skills/shelf/SKILL.md), [Dart Frog](template/.claude/skills/dart-frog/SKILL.md) |

### Workflows (On-Demand)

| Workflow | Purpose |
|----------|---------|
| [Initialize Project](template/.claude/skills/initialize-project/SKILL.md) | Setup new/existing projects |
| [Create PRD](template/.claude/skills/create-prd/SKILL.md) | Plan complex features |
| [Generate Tasks](template/.claude/skills/generate-tasks/SKILL.md) | Break PRDs into tasks |
| [Code Review](template/.claude/skills/code-review/SKILL.md) | Pre-commit quality review |
| [Security Audit](template/.claude/skills/security-audit/SKILL.md) | Security assessment |
| [Testing Strategy](template/.claude/skills/testing-strategy/SKILL.md) | Test planning & coverage |
| [Refactoring](template/.claude/skills/refactoring/SKILL.md) | Technical debt remediation |
| [Dependency Update](template/.claude/skills/dependency-update/SKILL.md) | Safe dependency updates |
| [Troubleshooting](template/.claude/skills/troubleshooting/SKILL.md) | Debug systematically |
| [Cleanup Project](template/.claude/skills/cleanup-project/SKILL.md) | Prune unused guides |
| [Document Work](template/.claude/skills/document-work/SKILL.md) | Capture patterns & decisions |
| [Update Framework](template/.claude/skills/update-framework/SKILL.md) | Update Samuel safely |
| [Generate AGENTS.md](template/.claude/skills/generate-agents-md/SKILL.md) | Cross-tool compatibility |

---

## Repository Structure

```text
samuel/
├── template/                    # Distributable template files
│   ├── CLAUDE.md               # Main AI instructions (copy to your project)
│   ├── AGENTS.md               # Cross-tool compatible version
│   └── .claude/                # AI context directory
│       └── skills/             # 21 language guides + 33 framework skills + 15 workflows
├── cmd/samuel/                  # CLI entry point
├── internal/                   # CLI implementation (commands, core, ui)
└── docs/                       # Documentation website source
```

**Why this structure?**

- `template/` contains files distributed to users via the CLI
- `internal/` contains the CLI tool that manages installations

---

## System Stats

| Metric | Value |
|--------|-------|
| **Version** | 2.6.4 |
| **Status** | Production Ready |
| **AGENTS.md** | Compatible |
| **Total Files** | 67 markdown files |
| **CLAUDE.md** | ~500 lines |
| **Language Guides** | 21 (all major programming languages) |
| **Framework Skills** | 33 (across 11 language families) |
| **Workflows** | 24 (PRD, tasks, init, troubleshoot, code-review, sync, etc.) |
| **Guardrails** | 35+ testable rules |

---

## Learning Path

### Week 1: Learn the Basics

- [ ] Install Samuel via CLI or copy template files to your project
- [ ] Write 5 features using ATOMIC mode
- [ ] Review guardrails in [CLAUDE.md](template/CLAUDE.md)
- [ ] Check which language guide applies to you

### Week 2: Try Complex Features

- [ ] Use PRD workflow for a medium feature
- [ ] Generate task breakdown
- [ ] Implement step-by-step

### Week 3: Customize & Extend

- [ ] (Multi-tool) Set up AGENTS.md for your team
- [ ] Experiment with different modes

---

## FAQ

### Q: Do I need to use Claude Code?
**A:** No! While designed for Claude Code, the system works with any AI coding assistant. Use the AGENTS.md symlink for other tools like Cursor or Codex.

### Q: Which file should I edit - CLAUDE.md or AGENTS.md?
**A:** Edit CLAUDE.md (the source of truth). If using symlink, AGENTS.md updates automatically. If using standalone AGENTS.md, regenerate it after CLAUDE.md changes.

### Q: My language isn't covered - what do I do?
**A:** The core guardrails in CLAUDE.md are language-agnostic (90% still applies). Consider contributing a new language guide!

### Q: Is the PRD workflow required?
**A:** No! Only for complex features (>10 files, new subsystems). Most work is ATOMIC or FEATURE mode.

### Q: Can I customize the guardrails?
**A:** Yes! Edit CLAUDE.md for your team. Common customizations:
- File length limits (300 → 500)
- Coverage targets (80% → 90%)
- Commit format (add your types)

---

## Contributing

Contributions welcome! Ideas:
- Additional language guides (Scala, Elixir, Haskell, OCaml, Julia)
- Additional framework skills for existing languages
- Integration examples with other AI tools
- Real-world case studies

**To contribute:**
1. Fork this repository
2. Create a feature branch
3. Make your changes following the guardrails
4. Submit a pull request

---

## License

MIT License - See [LICENSE](LICENSE) file for details.

---

## Acknowledgments

Built with:
- **Claude Code** - Anthropic's AI coding assistant
- **AGENTS.md Standard** - Universal AI agent instructions
- **4D Methodology** - Systematic problem-solving approach
- **Community feedback** - Continuous improvement

---

## Support

- **Documentation**: [CLAUDE.md](CLAUDE.md)
- **Issues**: [GitHub Issues](https://github.com/ar4mirez/samuel/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ar4mirez/samuel/discussions)

---

**Happy coding with AI!**

*Works with Claude Code, Cursor, Codex, Copilot, and any AI assistant that reads AGENTS.md*
