# Changelog

All notable changes to Samuel (AI Coding Framework) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

<!-- Add unreleased changes here -->

## [3.0.0] - 2026-04-29

### The lean reshape

v3.0.0 condenses the Samuel CLI from 14 top-level commands (~31 verbs once subcommands are counted) to 11 visible commands organized around the hot path: `init` → `run` → `update`. The autonomous Ralph Wiggum loop, which was buried under the opaque `auto` verb, is now `samuel run` (with `samuel auto` preserved as a permanent alias). Discovery collapses into one `ls` verb. Power-user surface moves under `admin`. Type inference replaces explicit `<type>` arguments. Every renamed command keeps working through v3.0.x via Hidden+Deprecated wrappers so v2 scripts have time to adapt.

The full v2 → v3 mapping is in [docs/getting-started/migration-v3.md](docs/getting-started/migration-v3.md).

### Added

- **`samuel run`** — primary verb for the autonomous Ralph Wiggum loop. New flat task verbs (`run tasks`, `run done <id>`, `run skip <id>`, `run reset <id>`, `run enqueue <title>`). The explicit-id form `run task add <id> <title>` is preserved for CI scripts that need deterministic IDs. Smart bare invocation: `samuel run` shows status when a loop is initialized in the current directory; in an empty directory it prints actionable help and exits non-zero. Never silently starts a loop.
- **`samuel ls`** — single discovery verb collapsing v2's `list`, `search`, and `info`. Full flag matrix: `--all`, `--type framework`, `--detail`, `--preview N`, `--no-related`, `--limit N`. Type aliases (`lang`/`l`, `fw`/`f`, `wf`/`w`) accepted.
- **`samuel admin`** — power-user umbrella. Children: `admin config`, `admin sync`, `admin diff [v1] [v2]`.
- **`samuel rm`** — permanent alias for `samuel remove`. Unix-y shorthand.
- **Type inference for `add` / `rm`** — `samuel add react` (no type) infers `framework`. `samuel add typescript` infers `language`. Both v2 (`add framework react`) and v3 (`add react framework`) argument orders work. `core.InferComponentType` scans `{Languages, Frameworks, Workflows}` (Skills excluded — the registry mirrors them).
- **`samuel run enqueue <title>`** — auto-id task creation. `core.AutoPRD.NextAvailableID()` finds the largest top-level integer prefix and returns max+1.
- **`samuel update --preview`** — replaces v2's bare `samuel diff`. Same dry-run output, cleaner verb. The `--diff` flag is kept as a Hidden alias.
- **`SAMUEL_NO_DEPRECATION=1` env var and `--no-deprecation` flag** — silence the one-line redirect notice that legacy aliases print. CI scripts can opt out without modification.
- **JSON `schemaVersion: 3`** — every `--json` envelope now carries the field. Consumers can branch on it.
- **JSON `command` field reflects invoked path** — `samuel ls --json` emits `"command": "ls"`. Legacy paths preserve their v2 strings (e.g., `samuel list --json` still emits `"command": "list"`) for backward compatibility through v3.0.x.
- **Cobra "did you mean?"** — `samuel adminn` now suggests `admin`. `SuggestionsMinimumDistance: 2` matches cargo and gh.
- **`TestRegistry_NoCrossTypeNameCollisions`** — invariant test enforces that no component name appears in two of `{Languages, Frameworks, Workflows}`. Future contributors who add a duplicate fail the build instead of silently breaking type inference.
- **Shell completion auto-installs via brew** — `brew install samuel` now generates bash/zsh/fish/powershell completions during install. Tab completion works immediately, no extra steps.
- **Migration guide** at [docs/getting-started/migration-v3.md](docs/getting-started/migration-v3.md) — full v2 → v3 command map, JSON schema migration, CI script guidance.

### Changed

- **`samuel auto` → `samuel run` (with `samuel auto` as permanent alias).** Both names work for every subcommand forever. `samuel auto` is not deprecated.
- **`samuel add <type> <name>` → `samuel add <name> [type]`.** Type optional; both arg orders accepted. v2 form stays valid forever.
- **JSON envelope `command` field semantics.** Now reflects the invoked path on v3 commands. v2 paths preserve their strings.
- **`samuel update`'s `--diff` flag is now Hidden.** Use `--preview`. The `--diff` flag still works for v2 scripts.
- **Registry component type inference.** Inference scope is `{Languages, Frameworks, Workflows}` only — Skills excluded by design (the registry triple-counts them). Documented in `core.InferComponentType` godoc.
- **README hero rewrite.** v2's generic "Build smarter, faster, more scalable software" tagline replaced with the autonomous-loop wedge: *"the autonomous AI coding loop your CLI needs — `samuel run` and walk away."* Direct response to /autoplan CEO + DX phase findings.

### Deprecated (Hidden in v3.0.x, removed in v3.1.0)

- `samuel list` → `samuel ls`
- `samuel search <query>` → `samuel ls <query>`
- `samuel info <type> <name>` → `samuel ls <name> --detail [--type <type>]`
- `samuel diff` (bare) → `samuel update --preview`
- `samuel diff <v1> <v2>` → `samuel admin diff <v1> <v2>`
- `samuel config <list|get|set>` → `samuel admin config <list|get|set>`
- `samuel sync` → `samuel admin sync`
- `samuel auto task list` → `samuel run tasks`
- `samuel auto task complete|skip|reset <id>` → `samuel run done|skip|reset <id>`

`samuel auto` itself is **NOT** deprecated. It's a permanent top-level alias for `samuel run`. Both names will work forever.

### Migration

- **Quick fix for CI**: set `SAMUEL_NO_DEPRECATION=1` to silence redirect notices. Legacy commands keep working through v3.0.x.
- **Proper fix**: update commands per [the migration guide](docs/getting-started/migration-v3.md). New forms produce identical output.
- **JSON consumers**: pin to `schemaVersion: 3` rather than command-name strings.
- **v3.1.0 (~3 months)**: legacy aliases removed. v2 commands return "unknown command" with cargo-style "did you mean?" suggestions pointing at the v3 names.

### Internal

- New `internal/commands/legacy.go` — `redirectAndRun` helper + `deprecationSuppressed` for the v3.0.x shim window.
- New `internal/commands/inference.go` — `parseAddRemoveArgs` shared between `add` and `remove`.
- New `internal/commands/json_helpers.go` — `commandPath`, `PrintJSONForCmd`, `PrintJSONErrorForCmd` derive the JSON `command` field from `cmd.CommandPath()`.
- `ui.JSONResponse` gains `SchemaVersion int` field. `ui.JSONSchemaVersion` constant exposed.
- Test coverage: 60+ new tests across the v3 surface (admin parent, legacy wrappers, ls inference, run rename, NextAvailableID edge cases, type inference, JSON envelope, did-you-mean wiring).

## [2.0.0] - 2026-02-12

### Renamed to Samuel

Breaking change: project renamed from AICoF to **Samuel**. Binary, config files, cache paths, and Go module path all changed.

### Added

- **Autonomous AI Coding Loop** (Ralph Wiggum methodology):
  - `samuel auto init` - Initialize autonomous loop from PRD
  - `samuel auto convert` - Convert markdown PRD/tasks to prd.json
  - `samuel auto status` - Show loop progress
  - `samuel auto start` - Begin/resume autonomous execution
  - `samuel auto task` - Manage tasks (list, complete, skip, reset, add)
- **Auto workflow skill** at `.claude/skills/auto/SKILL.md`
- **Per-folder CLAUDE.md** support for hierarchical instructions
- **Homebrew formula** (was cask) — `brew install samuel`

### Changed

- **Project renamed**: `aicof` → `samuel` (binary, config, module path)
- Config file: `aicof.yaml` → `samuel.yaml`
- Cache/config dirs: `~/.config/aicof/` → `~/.config/samuel/`
- Go module: `github.com/ar4mirez/aicof` → `github.com/ar4mirez/samuel`
- Migrated from `.agent/` to `.claude/` (native Claude Code directory)
- Skills now live in `.claude/skills/` (native skill discovery)
- Homebrew distribution changed from cask to formula

### Breaking

- Binary renamed from `aicof` to `samuel`
- Config file renamed from `aicof.yaml` to `samuel.yaml`
- `.agent/` directory no longer used — migrate to `.claude/`

## [1.8.0] - 2026-02-04

### Added

- **Agent Skills Management CLI**:
  - `samuel skill create <name>` - Scaffold new skills
  - `samuel skill validate [name]` - Validate against Agent Skills specification
  - `samuel skill list` - List installed skills
  - `samuel skill info <name>` - Show skill details
- **Create Skill workflow** at `.claude/skills/create-skill/SKILL.md`
- Skills compatible with 25+ agent products
- `installed.skills` config key

### Changed

- All components now follow Agent Skills standard (SKILL.md with YAML frontmatter)
- Language guides renamed to `<name>-guide/SKILL.md` format
- Framework guides moved to `<name>/SKILL.md` format
- Workflows moved to `<name>/SKILL.md` format

## [1.7.0] - 2025-01-14

### Changed

#### Rebrand to AICoF
- **Repository renamed**: `ai-code-template` → `aicof`
- **Brand name**: "AI Code Template" → "AICoF (Artificial Intelligence Coding Framework)"
- **New tagline**: "Build smarter, faster, and more scalable software"

#### Documentation
- Updated all documentation to reflect new brand name
- Updated all GitHub URLs and repository references
- Updated installation/clone commands throughout
- Updated MkDocs configuration for new site URL

## [1.6.0] - 2025-01-14

### Added

#### Workflows
- **document-work.md** - Capture patterns, decisions, and learnings from recent development work
  - Analyzes git commits to identify documentation needs
  - Creates/updates patterns.md, memory files, state.md, project.md
  - Supports end-of-session, feature completion, and handoff scenarios
- **update-framework.md** - Update AICoF while preserving customizations
  - Detects current version and compares with latest
  - Identifies customizations to preserve
  - Supports full replace, selective merge, and new files only strategies
  - Includes rollback procedures

#### Documentation
- **Comprehensive workflow examples** in .agent/workflows/README.md
  - Real-world usage scenarios for all 13 workflows
  - Example conversations showing AI responses
  - Expected outputs for each workflow

### Fixed
- **Repository name references** - Changed `ai-claude-code` to `ai-code-template` across:
  - README.md (3 occurrences)
  - AI_INSTRUCTIONS.md (1 occurrence)
  - CHANGELOG.md (7 version comparison links)

### Changed
- **Workflow count**: Now 13 workflows (was 11)
- **Workflow categories**:
  - Maintenance: Added update-framework
  - Utility: Added document-work
- **Workflow map**: Updated ASCII diagram to include new workflows

## [1.5.0] - 2025-12-15

### Added

#### Framework Guides
- **33 framework-specific guides** across 11 language families:
  - **TypeScript/JavaScript**: React, Next.js, Express
  - **Python**: Django, FastAPI, Flask
  - **Go**: Gin, Echo, Fiber
  - **Rust**: Axum, Actix-web, Rocket
  - **Kotlin**: Spring Boot (Kotlin), Ktor, Android Compose
  - **Java**: Spring Boot, Quarkus, Micronaut
  - **C#**: ASP.NET Core, Blazor, Unity
  - **PHP**: Laravel, Symfony, WordPress
  - **Swift**: SwiftUI, UIKit, Vapor
  - **Ruby**: Rails, Sinatra, Hanami
  - **Dart**: Flutter, Shelf, Dart Frog

### Changed

- **CLAUDE.md**: Added "Load Framework Guide" section to Quick Reference
- **Total guides**: Now 54 (21 language guides + 33 framework guides)
- **README.md**: Added Framework Guides section with full table
- **Documentation site**: Added frameworks section with all 33 guides

## [1.4.0] - 2025-12-13

### Added

#### Language Guides - Specialized Domains
- **10 new language guides** for specialized programming domains:
  - **[sql.md](.agent/language-guides/sql.md)** - SQL (PostgreSQL, MySQL, query optimization, migrations)
  - **[shell.md](.agent/language-guides/shell.md)** - Shell/Bash (scripting, automation, POSIX compliance)
  - **[r.md](.agent/language-guides/r.md)** - R (statistical computing, tidyverse, Shiny)
  - **[dart.md](.agent/language-guides/dart.md)** - Dart (Flutter, mobile development)
  - **[html-css.md](.agent/language-guides/html-css.md)** - HTML/CSS (web standards, accessibility, BEM)
  - **[lua.md](.agent/language-guides/lua.md)** - Lua (scripting, Love2D, Neovim)
  - **[assembly.md](.agent/language-guides/assembly.md)** - Assembly (x86-64, ARM64, RISC-V)
  - **[cuda.md](.agent/language-guides/cuda.md)** - CUDA (GPU computing, parallel processing)
  - **[solidity.md](.agent/language-guides/solidity.md)** - Solidity (Ethereum, smart contracts, DeFi)
  - **[zig.md](.agent/language-guides/zig.md)** - Zig (systems programming, C interop)

### Changed

- **Total language guides**: Now 21 (up from 11 in v1.3.0)
- **CLAUDE.md**: Updated Quick Reference with all 21 language guides
- **README.md**: Updated language guides table with all supported languages
- **.agent/language-guides/README.md**: Reorganized with categorized sections

## [1.3.0] - 2025-12-13

### Added

#### Language Guides - Enterprise & Systems
- **6 new language guides** for enterprise and systems programming:
  - **[java.md](.agent/language-guides/java.md)** - Java (Spring Boot, Jakarta EE, JUnit 5)
  - **[csharp.md](.agent/language-guides/csharp.md)** - C# (.NET 8, ASP.NET Core, xUnit)
  - **[php.md](.agent/language-guides/php.md)** - PHP (Laravel, Symfony, PHPUnit)
  - **[swift.md](.agent/language-guides/swift.md)** - Swift (iOS, macOS, SwiftUI)
  - **[cpp.md](.agent/language-guides/cpp.md)** - C/C++ (C++20, CMake, GoogleTest)
  - **[ruby.md](.agent/language-guides/ruby.md)** - Ruby (Rails 7, RSpec, Sidekiq)

### Changed

- **Total language guides**: Now 11 (up from 5 in v1.2.0)
- **CLAUDE.md**: Updated context system documentation with new language guides
- **README.md**: Added new languages to documentation table

## [1.2.0] - 2025-12-13

### Added

#### AGENTS.md Compatibility
- **Operations section** in CLAUDE.md (AGENTS.md standard compatible)
  - Setup Commands for Node.js, Python, Go, Rust
  - Testing Commands with coverage targets
  - Build & Deploy Commands
  - Code Style Commands (format, lint)
  - Environment Variables template
- **Boundaries section** - prominent "do not touch" list for AI agents
- **Cross-tool symlink instructions** - `ln -s CLAUDE.md AGENTS.md`
- **generate-agents-md.md workflow** - generates standalone AGENTS.md for cross-tool teams

#### Documentation Site
- **MkDocs Material documentation site** with GitHub Pages
  - Light/dark mode toggle with system preference detection
  - Full-text search with highlighting
  - Responsive design for mobile and desktop
  - Code syntax highlighting with copy button
  - Mermaid diagram support
- **Comprehensive documentation structure**
  - Getting Started: quick-start, installation, first-task guides
  - Core System: overview, CLAUDE.md, methodology, guardrails, .agent directory
  - Language Guides: TypeScript, Python, Go, Rust, Kotlin
  - Workflows: initialize, create-prd, generate-tasks, troubleshooting, generate-agents-md
  - Reference: cross-tool compatibility, FAQ, changelog, contributing
- **GitHub Actions workflow** for automatic deployment to GitHub Pages
- **requirements-docs.txt** for documentation dependencies

#### Documentation
- **Professional prompt examples** in README.md (25+ real-world prompts)
  - Bug fixes (ATOMIC mode)
  - Feature development (FEATURE mode)
  - Complex features (COMPLEX mode)
  - Code review & analysis
  - Refactoring
  - Architecture & planning
  - Debugging & troubleshooting
  - Testing
  - Documentation
  - DevOps & infrastructure
- **4D Methodology visual diagram** in README.md
- **Cross-tool compatibility section** with tool priority table
- **Use cases section** with detailed examples

#### Language Guides
- **Kotlin language guide** (.agent/language-guides/kotlin.md)
  - Null safety patterns
  - Coroutines and structured concurrency
  - Data classes and sealed classes
  - Android and Spring Boot patterns
  - Kotest and MockK testing

### Changed

- **CLAUDE.md structure** - Operations section now at top (commands first, context second)
- **README.md** - Complete rewrite with professional focus
- **AI_INSTRUCTIONS.md** - Updated for v1.2.0 with cross-tool info
- **.agent/README.md** - Added AGENTS.md compatibility section
- **.agent/workflows/README.md** - Added generate-agents-md workflow

### Removed

- **Duplicate "Protected Boundaries" section** - consolidated into "Boundaries" at top of CLAUDE.md

## [1.1.0] - 2025-01-15

### Added

#### Guardrails
- Dependency license checking before adding packages
- Database migration rollback (down) function requirement
- Semantic versioning for breaking API changes
- Smoke test validation for deployments

#### Language Guides
- **TypeScript/JavaScript guide** (.agent/language-guides/typescript.md)
  - Strict mode configuration
  - Zod validation patterns
  - React hooks and functional components
  - Async/await best practices
- **Python guide** (.agent/language-guides/python.md)
  - Type hints (PEP 484)
  - Pydantic validation
  - Django and FastAPI patterns
  - pytest testing
- **Go guide** (.agent/language-guides/go.md)
  - Error handling patterns
  - Concurrency with goroutines
  - Table-driven tests
  - Interface design
- **Rust guide** (.agent/language-guides/rust.md)
  - Ownership and borrowing
  - Result<T,E> error handling
  - Async patterns with Tokio
  - Zero-cost abstractions

#### Workflows
- **initialize-project.md** - New/existing project setup workflow
- **troubleshooting.md** - Systematic debugging guide

#### Documentation
- **Quick Reference section** in CLAUDE.md
- **Line number references** for quick navigation

### Changed

- **CLAUDE.md size** - Reduced from 490 → 400 lines (18% reduction)
- **Workflow requirements** - Clarified MANDATORY vs RECOMMENDED vs OPTIONAL
- **Language-specific content** - Extracted to .agent/language-guides/
- **Initialization content** - Extracted to .agent/workflows/initialize-project.md
- **Troubleshooting content** - Extracted to .agent/workflows/troubleshooting.md

### Improved

- Token efficiency - ~1,600 tokens base context (vs. 4,000 before)
- Auto-loading - Language guides load based on file extensions
- Progressive loading - .agent/ files loaded on-demand

## [1.0.0] - 2025-01-14

### Added

#### Core System
- **CLAUDE.md** - Core guardrails file with 30+ rules
- **4D Methodology** - Deconstruct → Diagnose → Develop → Deliver
- **3 Modes** - ATOMIC, FEATURE, COMPLEX task classification
- **5 SDLC Stages** - Planning, Implementation, Validation, Documentation, Commit

#### Guardrails
- **Code Quality** (8 rules)
  - Function length ≤50 lines
  - File length ≤300 lines
  - Cyclomatic complexity ≤10
  - Type signatures and documentation
  - No magic numbers
  - No commented-out code
  - No TODO without issue reference
  - No dead code
- **Security** (6 rules)
  - Input validation
  - Parameterized queries
  - No hardcoded secrets
  - Path validation
  - Timeout/cancellation for async
  - Dependency vulnerability checking
- **Testing** (8 rules)
  - Coverage targets (80% business logic, 60% overall)
  - Public API tests
  - Regression tests for bugs
  - Edge case testing
  - Descriptive test names
  - No test interdependencies
  - Integration tests for external services
  - Smoke test validation
- **Git** (7 rules)
  - Conventional commits
  - Atomic commits
  - Tests pass before push
  - Branch naming conventions
  - No direct commits to main
  - PR required
  - Semantic versioning
- **Performance** (5 rules)
  - No N+1 queries
  - Pagination for large datasets
  - Memoization/caching
  - Bundle size limits
  - API response time limits

#### Workflows
- **create-prd.md** - Product Requirements Document workflow
- **generate-tasks.md** - Task breakdown from PRDs

#### Context System
- **.agent/ directory structure**
  - project.md - Tech stack and architecture
  - patterns.md - Coding patterns and conventions
  - state.md - Current work tracking
  - memory/ - Decision logs
  - tasks/ - PRDs and task lists
- **Progressive growth philosophy** - Start minimal, grow organically
- **On-demand loading** - Load context only when needed

#### Documentation
- **README.md** - Project overview and quick start
- **AI_INSTRUCTIONS.md** - Quick start guide
- **.agent/README.md** - Context system documentation
- **.agent/workflows/README.md** - Workflow documentation
- **Example PRD** - .agent/tasks/EXAMPLE-0001-prd-api-rate-limiting.md

### Technical Details

- **Token optimized** - Designed for efficient context usage
- **Tech-stack agnostic** - Works with any language/framework
- **AI-first design** - Structured for AI coding assistants

---

## Version History Summary

| Version | Date | Highlights |
|---------|------|------------|
| 1.7.0 | 2025-01-14 | Rebrand to AICoF (Artificial Intelligence Coding Framework) |
| 1.6.0 | 2025-01-14 | document-work and update-framework workflows, comprehensive examples |
| 1.5.0 | 2025-12-15 | 33 framework guides across 11 language families |
| 1.4.0 | 2025-12-13 | 10 specialized language guides (SQL, Shell, R, Dart, HTML/CSS, Lua, Assembly, CUDA, Solidity, Zig) |
| 1.3.0 | 2025-12-13 | 6 enterprise language guides (Java, C#, PHP, Swift, C++, Ruby) |
| 1.2.0 | 2025-12-13 | AGENTS.md compatibility, MkDocs documentation site, Kotlin guide |
| 1.1.0 | 2025-01-15 | 4 language guides, 2 new workflows, 18% size reduction |
| 1.0.0 | 2025-01-14 | Initial release with 30+ guardrails, 4D methodology |

---

## Migration Guide

### From 1.3.0 to 1.4.0

No breaking changes. New specialized language guides are additive:
- SQL, Shell/Bash, R, Dart/Flutter, HTML/CSS, Lua, Assembly, CUDA, Solidity, Zig

### From 1.2.0 to 1.3.0

No breaking changes. New enterprise language guides are additive:
- Java, C#, PHP, Swift, C/C++, Ruby

### From 1.1.0 to 1.2.0

No breaking changes. To enable cross-tool compatibility:

```bash
# Option 1: Symlink (recommended)
ln -s CLAUDE.md AGENTS.md

# Option 2: Generate standalone AGENTS.md
# Use @.agent/workflows/generate-agents-md.md
```

### From 1.0.0 to 1.1.0

No breaking changes. New language guides and workflows are additive.

---

## Links

- [CLAUDE.md](CLAUDE.md) - Core guardrails
- [README.md](README.md) - Project overview
- [AI_INSTRUCTIONS.md](AI_INSTRUCTIONS.md) - Quick start guide
- [AGENTS.md Standard](https://agents.md) - Cross-tool compatibility standard

[Unreleased]: https://github.com/ar4mirez/samuel/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/ar4mirez/samuel/compare/v1.8.0...v2.0.0
[1.8.0]: https://github.com/ar4mirez/samuel/compare/v1.7.0...v1.8.0
[1.7.0]: https://github.com/ar4mirez/samuel/compare/v1.6.0...v1.7.0
[1.6.0]: https://github.com/ar4mirez/samuel/compare/v1.5.0...v1.6.0
[1.5.0]: https://github.com/ar4mirez/samuel/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/ar4mirez/samuel/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/ar4mirez/samuel/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/ar4mirez/samuel/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/ar4mirez/samuel/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/ar4mirez/samuel/releases/tag/v1.0.0
