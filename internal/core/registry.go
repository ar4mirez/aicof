package core

import "fmt"

// DefaultRegistry is the default GitHub repository for Samuel
const DefaultRegistry = "https://github.com/ar4mirez/samuel"

// DefaultOwner is the GitHub owner
const DefaultOwner = "ar4mirez"

// DefaultRepo is the GitHub repository name
const DefaultRepo = "samuel"

// TemplatePrefix is the path prefix in the repository where template files are stored
// This allows the CLI to find source files in the downloaded archive
const TemplatePrefix = "template/"

// Component represents an installable component
type Component struct {
	Name        string
	Path        string
	Description string
	Category    string   // Optional: "language", "framework", "skill", ""
	Tags        []string // Optional: additional search terms e.g. ["golang", "backend"]
}

// ComponentType represents the type of component
type ComponentType string

const (
	ComponentTypeLanguage  ComponentType = "language"
	ComponentTypeFramework ComponentType = "framework"
	ComponentTypeWorkflow  ComponentType = "workflow"
	ComponentTypeSkill     ComponentType = "skill"
)

// Languages contains all available language guide skills.
// Language guides are now Agent Skills stored at .claude/skills/<name>-guide/.
var Languages = []Component{
	{Name: "typescript", Path: ".claude/skills/typescript-guide", Description: "TypeScript/JavaScript", Category: "language", Tags: []string{"ts", "js", "javascript", "node"}},
	{Name: "python", Path: ".claude/skills/python-guide", Description: "Python", Category: "language", Tags: []string{"py", "pip", "django", "fastapi"}},
	{Name: "go", Path: ".claude/skills/go-guide", Description: "Go", Category: "language", Tags: []string{"golang", "goroutine"}},
	{Name: "rust", Path: ".claude/skills/rust-guide", Description: "Rust", Category: "language", Tags: []string{"cargo", "crate"}},
	{Name: "kotlin", Path: ".claude/skills/kotlin-guide", Description: "Kotlin", Category: "language", Tags: []string{"kt", "android", "jvm"}},
	{Name: "java", Path: ".claude/skills/java-guide", Description: "Java", Category: "language", Tags: []string{"jvm", "maven", "gradle"}},
	{Name: "csharp", Path: ".claude/skills/csharp-guide", Description: "C#/.NET", Category: "language", Tags: []string{"dotnet", "net", "cs"}},
	{Name: "php", Path: ".claude/skills/php-guide", Description: "PHP", Category: "language", Tags: []string{"composer", "laravel"}},
	{Name: "swift", Path: ".claude/skills/swift-guide", Description: "Swift", Category: "language", Tags: []string{"ios", "macos", "xcode"}},
	{Name: "cpp", Path: ".claude/skills/cpp-guide", Description: "C/C++", Category: "language", Tags: []string{"c", "cplusplus", "cmake"}},
	{Name: "ruby", Path: ".claude/skills/ruby-guide", Description: "Ruby", Category: "language", Tags: []string{"rb", "gem", "rails"}},
	{Name: "sql", Path: ".claude/skills/sql-guide", Description: "SQL", Category: "language", Tags: []string{"postgres", "mysql", "sqlite"}},
	{Name: "shell", Path: ".claude/skills/shell-guide", Description: "Shell/Bash", Category: "language", Tags: []string{"bash", "sh", "zsh", "scripting"}},
	{Name: "r", Path: ".claude/skills/r-guide", Description: "R", Category: "language", Tags: []string{"rstats", "tidyverse", "cran"}},
	{Name: "dart", Path: ".claude/skills/dart-guide", Description: "Dart", Category: "language", Tags: []string{"flutter", "pub"}},
	{Name: "html-css", Path: ".claude/skills/html-css-guide", Description: "HTML/CSS", Category: "language", Tags: []string{"html", "css", "web", "accessibility"}},
	{Name: "lua", Path: ".claude/skills/lua-guide", Description: "Lua", Category: "language", Tags: []string{"luajit", "love2d", "neovim"}},
	{Name: "assembly", Path: ".claude/skills/assembly-guide", Description: "Assembly", Category: "language", Tags: []string{"asm", "x86", "arm"}},
	{Name: "cuda", Path: ".claude/skills/cuda-guide", Description: "CUDA", Category: "language", Tags: []string{"gpu", "nvidia", "parallel"}},
	{Name: "solidity", Path: ".claude/skills/solidity-guide", Description: "Solidity", Category: "language", Tags: []string{"ethereum", "evm", "smart-contract"}},
	{Name: "zig", Path: ".claude/skills/zig-guide", Description: "Zig", Category: "language", Tags: []string{"systems", "comptime"}},
}

// Frameworks contains all available framework guide skills.
// Framework guides are now Agent Skills stored at .claude/skills/<name>/.
var Frameworks = []Component{
	// TypeScript/JavaScript
	{Name: "react", Path: ".claude/skills/react", Description: "React", Category: "framework", Tags: []string{"reactjs", "jsx", "tsx", "frontend"}},
	{Name: "nextjs", Path: ".claude/skills/nextjs", Description: "Next.js", Category: "framework", Tags: []string{"next", "ssr", "react", "vercel"}},
	{Name: "express", Path: ".claude/skills/express", Description: "Express.js", Category: "framework", Tags: []string{"node", "rest", "api", "middleware"}},
	// Python
	{Name: "django", Path: ".claude/skills/django", Description: "Django", Category: "framework", Tags: []string{"python", "orm", "drf", "admin"}},
	{Name: "fastapi", Path: ".claude/skills/fastapi", Description: "FastAPI", Category: "framework", Tags: []string{"python", "async", "pydantic", "openapi"}},
	{Name: "flask", Path: ".claude/skills/flask", Description: "Flask", Category: "framework", Tags: []string{"python", "wsgi", "jinja", "lightweight"}},
	// Go
	{Name: "gin", Path: ".claude/skills/gin", Description: "Gin", Category: "framework", Tags: []string{"go", "golang", "rest", "api"}},
	{Name: "echo", Path: ".claude/skills/echo", Description: "Echo", Category: "framework", Tags: []string{"go", "golang", "rest", "labstack"}},
	{Name: "fiber", Path: ".claude/skills/fiber", Description: "Fiber", Category: "framework", Tags: []string{"go", "golang", "express-like", "fasthttp"}},
	// Rust
	{Name: "axum", Path: ".claude/skills/axum", Description: "Axum", Category: "framework", Tags: []string{"rust", "tower", "async", "tokio"}},
	{Name: "actix-web", Path: ".claude/skills/actix-web", Description: "Actix-web", Category: "framework", Tags: []string{"rust", "actor", "async", "http"}},
	{Name: "rocket", Path: ".claude/skills/rocket", Description: "Rocket", Category: "framework", Tags: []string{"rust", "type-safe", "macros"}},
	// Kotlin
	{Name: "spring-boot-kotlin", Path: ".claude/skills/spring-boot-kotlin", Description: "Spring Boot (Kotlin)", Category: "framework", Tags: []string{"kotlin", "spring", "jvm", "coroutines"}},
	{Name: "ktor", Path: ".claude/skills/ktor", Description: "Ktor", Category: "framework", Tags: []string{"kotlin", "coroutines", "dsl", "jvm"}},
	{Name: "android-compose", Path: ".claude/skills/android-compose", Description: "Android Compose", Category: "framework", Tags: []string{"kotlin", "android", "material3", "jetpack"}},
	// Java
	{Name: "spring-boot-java", Path: ".claude/skills/spring-boot-java", Description: "Spring Boot (Java)", Category: "framework", Tags: []string{"java", "spring", "jvm", "jpa"}},
	{Name: "quarkus", Path: ".claude/skills/quarkus", Description: "Quarkus", Category: "framework", Tags: []string{"java", "graalvm", "reactive", "cloud-native"}},
	{Name: "micronaut", Path: ".claude/skills/micronaut", Description: "Micronaut", Category: "framework", Tags: []string{"java", "compile-time", "di", "cloud-native"}},
	// C#
	{Name: "aspnet-core", Path: ".claude/skills/aspnet-core", Description: "ASP.NET Core", Category: "framework", Tags: []string{"csharp", "dotnet", "minimal-api", "efcore"}},
	{Name: "blazor", Path: ".claude/skills/blazor", Description: "Blazor", Category: "framework", Tags: []string{"csharp", "dotnet", "wasm", "signalr"}},
	{Name: "unity", Path: ".claude/skills/unity", Description: "Unity", Category: "framework", Tags: []string{"csharp", "game", "3d", "scripting"}},
	// PHP
	{Name: "laravel", Path: ".claude/skills/laravel", Description: "Laravel", Category: "framework", Tags: []string{"php", "eloquent", "blade", "artisan"}},
	{Name: "symfony", Path: ".claude/skills/symfony", Description: "Symfony", Category: "framework", Tags: []string{"php", "doctrine", "twig", "components"}},
	{Name: "wordpress", Path: ".claude/skills/wordpress", Description: "WordPress", Category: "framework", Tags: []string{"php", "cms", "themes", "plugins"}},
	// Swift
	{Name: "swiftui", Path: ".claude/skills/swiftui", Description: "SwiftUI", Category: "framework", Tags: []string{"swift", "ios", "macos", "declarative"}},
	{Name: "uikit", Path: ".claude/skills/uikit", Description: "UIKit", Category: "framework", Tags: []string{"swift", "ios", "storyboard", "programmatic"}},
	{Name: "vapor", Path: ".claude/skills/vapor", Description: "Vapor", Category: "framework", Tags: []string{"swift", "server", "fluent", "async"}},
	// Ruby
	{Name: "rails", Path: ".claude/skills/rails", Description: "Rails", Category: "framework", Tags: []string{"ruby", "activerecord", "mvc", "hotwire"}},
	{Name: "sinatra", Path: ".claude/skills/sinatra", Description: "Sinatra", Category: "framework", Tags: []string{"ruby", "lightweight", "dsl", "rack"}},
	{Name: "hanami", Path: ".claude/skills/hanami", Description: "Hanami", Category: "framework", Tags: []string{"ruby", "clean-architecture", "dry-rb"}},
	// Dart
	{Name: "flutter", Path: ".claude/skills/flutter", Description: "Flutter", Category: "framework", Tags: []string{"dart", "mobile", "material", "riverpod"}},
	{Name: "shelf", Path: ".claude/skills/shelf", Description: "Shelf", Category: "framework", Tags: []string{"dart", "http", "server", "middleware"}},
	{Name: "dart-frog", Path: ".claude/skills/dart-frog", Description: "Dart Frog", Category: "framework", Tags: []string{"dart", "server", "file-based", "routing"}},
}

// Workflows contains all available workflow skills.
// Workflows are now Agent Skills stored at .claude/skills/<name>/.
var Workflows = []Component{
	// Planning
	{Name: "initialize-project", Path: ".claude/skills/initialize-project", Description: "Project setup", Category: "workflow", Tags: []string{"init", "setup", "scaffold", "new-project"}},
	{Name: "create-rfd", Path: ".claude/skills/create-rfd", Description: "Technical decision documents", Category: "workflow", Tags: []string{"rfd", "decision", "architecture", "discussion"}},
	{Name: "create-prd", Path: ".claude/skills/create-prd", Description: "Requirements documents", Category: "workflow", Tags: []string{"prd", "requirements", "planning", "spec"}},
	{Name: "generate-tasks", Path: ".claude/skills/generate-tasks", Description: "Task breakdown", Category: "workflow", Tags: []string{"tasks", "breakdown", "subtasks", "planning"}},
	// Quality
	{Name: "code-review", Path: ".claude/skills/code-review", Description: "Pre-commit quality review", Category: "workflow", Tags: []string{"review", "quality", "pr", "lint"}},
	{Name: "security-audit", Path: ".claude/skills/security-audit", Description: "Security assessment", Category: "workflow", Tags: []string{"security", "audit", "owasp", "vulnerability"}},
	{Name: "testing-strategy", Path: ".claude/skills/testing-strategy", Description: "Test planning", Category: "workflow", Tags: []string{"testing", "coverage", "tdd", "strategy"}},
	// Maintenance
	{Name: "cleanup-project", Path: ".claude/skills/cleanup-project", Description: "Prune unused guides", Category: "workflow", Tags: []string{"cleanup", "prune", "unused", "maintenance"}},
	{Name: "refactoring", Path: ".claude/skills/refactoring", Description: "Technical debt remediation", Category: "workflow", Tags: []string{"refactor", "debt", "improve", "restructure"}},
	{Name: "dependency-update", Path: ".claude/skills/dependency-update", Description: "Safe dependency updates", Category: "workflow", Tags: []string{"dependencies", "update", "upgrade", "packages"}},
	{Name: "update-framework", Path: ".claude/skills/update-framework", Description: "Samuel version updates", Category: "workflow", Tags: []string{"samuel", "update", "version", "migrate"}},
	// Utility
	{Name: "troubleshooting", Path: ".claude/skills/troubleshooting", Description: "Debugging workflow", Category: "workflow", Tags: []string{"debug", "stuck", "error", "fix"}},
	{Name: "generate-agents-md", Path: ".claude/skills/generate-agents-md", Description: "Cross-tool compatibility", Category: "workflow", Tags: []string{"agents-md", "compatibility", "cross-tool"}},
	{Name: "document-work", Path: ".claude/skills/document-work", Description: "Capture patterns", Category: "workflow", Tags: []string{"document", "patterns", "decisions", "learnings"}},
	{Name: "create-skill", Path: ".claude/skills/create-skill", Description: "Create Agent Skills", Category: "workflow", Tags: []string{"skill", "create", "scaffold", "agent-skills"}},
	// Autonomous
	{Name: "auto", Path: ".claude/skills/auto", Description: "Autonomous AI coding loop", Category: "workflow", Tags: []string{"auto", "autonomous", "loop", "ralph", "afk"}},
	// Anthropic community (source: github.com/anthropics/skills)
	{Name: "algorithmic-art", Path: ".claude/skills/algorithmic-art", Description: "Generative art with p5.js", Category: "workflow", Tags: []string{"art", "generative", "p5js", "creative"}},
	{Name: "doc-coauthoring", Path: ".claude/skills/doc-coauthoring", Description: "Collaborative document writing", Category: "workflow", Tags: []string{"writing", "docs", "collaboration"}},
	{Name: "frontend-design", Path: ".claude/skills/frontend-design", Description: "Design-thinking for frontend", Category: "workflow", Tags: []string{"design", "frontend", "ui", "ux"}},
	{Name: "mcp-builder", Path: ".claude/skills/mcp-builder", Description: "MCP server development", Category: "workflow", Tags: []string{"mcp", "server", "api", "integration"}},
	{Name: "theme-factory", Path: ".claude/skills/theme-factory", Description: "Artifact theming toolkit", Category: "workflow", Tags: []string{"theme", "styling", "design", "colors"}},
	{Name: "web-artifacts-builder", Path: ".claude/skills/web-artifacts-builder", Description: "React/shadcn web apps", Category: "workflow", Tags: []string{"react", "typescript", "shadcn", "vite"}},
	{Name: "webapp-testing", Path: ".claude/skills/webapp-testing", Description: "Playwright web testing", Category: "workflow", Tags: []string{"playwright", "testing", "browser", "e2e"}},
	{Name: "sync-claude-md", Path: ".claude/skills/sync-claude-md", Description: "Sync per-folder CLAUDE.md files", Category: "workflow", Tags: []string{"sync", "claude-md", "folder", "context"}},
}

// Skills contains bundled skills (community skills loaded dynamically).
// Language guide skills are also registered in the Languages slice for backward compatibility.
var Skills = []Component{
	{Name: "commit-message", Path: ".claude/skills/commit-message", Description: "Generate commit messages"},
	// Language guide skills (mirrored from Languages with skill naming)
	{Name: "go-guide", Path: ".claude/skills/go-guide", Description: "Go language guardrails and patterns", Category: "language", Tags: []string{"go", "golang"}},
	{Name: "typescript-guide", Path: ".claude/skills/typescript-guide", Description: "TypeScript/JavaScript guardrails and patterns", Category: "language", Tags: []string{"typescript", "ts", "javascript", "js"}},
	{Name: "python-guide", Path: ".claude/skills/python-guide", Description: "Python guardrails and patterns", Category: "language", Tags: []string{"python", "py"}},
	{Name: "rust-guide", Path: ".claude/skills/rust-guide", Description: "Rust guardrails and patterns", Category: "language", Tags: []string{"rust", "cargo"}},
	{Name: "kotlin-guide", Path: ".claude/skills/kotlin-guide", Description: "Kotlin guardrails and patterns", Category: "language", Tags: []string{"kotlin", "kt"}},
	{Name: "java-guide", Path: ".claude/skills/java-guide", Description: "Java guardrails and patterns", Category: "language", Tags: []string{"java", "jvm"}},
	{Name: "csharp-guide", Path: ".claude/skills/csharp-guide", Description: "C#/.NET guardrails and patterns", Category: "language", Tags: []string{"csharp", "dotnet"}},
	{Name: "php-guide", Path: ".claude/skills/php-guide", Description: "PHP guardrails and patterns", Category: "language", Tags: []string{"php", "composer"}},
	{Name: "swift-guide", Path: ".claude/skills/swift-guide", Description: "Swift guardrails and patterns", Category: "language", Tags: []string{"swift", "ios"}},
	{Name: "cpp-guide", Path: ".claude/skills/cpp-guide", Description: "C/C++ guardrails and patterns", Category: "language", Tags: []string{"cpp", "c", "cmake"}},
	{Name: "ruby-guide", Path: ".claude/skills/ruby-guide", Description: "Ruby guardrails and patterns", Category: "language", Tags: []string{"ruby", "rb"}},
	{Name: "sql-guide", Path: ".claude/skills/sql-guide", Description: "SQL guardrails and patterns", Category: "language", Tags: []string{"sql", "postgres", "mysql"}},
	{Name: "shell-guide", Path: ".claude/skills/shell-guide", Description: "Shell/Bash guardrails and patterns", Category: "language", Tags: []string{"shell", "bash", "sh"}},
	{Name: "r-guide", Path: ".claude/skills/r-guide", Description: "R guardrails and patterns", Category: "language", Tags: []string{"r", "rstats"}},
	{Name: "dart-guide", Path: ".claude/skills/dart-guide", Description: "Dart guardrails and patterns", Category: "language", Tags: []string{"dart", "flutter"}},
	{Name: "html-css-guide", Path: ".claude/skills/html-css-guide", Description: "HTML/CSS guardrails and patterns", Category: "language", Tags: []string{"html", "css"}},
	{Name: "lua-guide", Path: ".claude/skills/lua-guide", Description: "Lua guardrails and patterns", Category: "language", Tags: []string{"lua", "luajit"}},
	{Name: "assembly-guide", Path: ".claude/skills/assembly-guide", Description: "Assembly guardrails and patterns", Category: "language", Tags: []string{"assembly", "asm"}},
	{Name: "cuda-guide", Path: ".claude/skills/cuda-guide", Description: "CUDA guardrails and patterns", Category: "language", Tags: []string{"cuda", "gpu"}},
	{Name: "solidity-guide", Path: ".claude/skills/solidity-guide", Description: "Solidity guardrails and patterns", Category: "language", Tags: []string{"solidity", "ethereum"}},
	{Name: "zig-guide", Path: ".claude/skills/zig-guide", Description: "Zig guardrails and patterns", Category: "language", Tags: []string{"zig", "comptime"}},
	// Framework skills (mirrored from Frameworks)
	{Name: "react", Path: ".claude/skills/react", Description: "React framework guardrails and patterns", Category: "framework", Tags: []string{"react", "reactjs", "jsx", "tsx"}},
	{Name: "nextjs", Path: ".claude/skills/nextjs", Description: "Next.js framework guardrails and patterns", Category: "framework", Tags: []string{"nextjs", "next", "ssr"}},
	{Name: "express", Path: ".claude/skills/express", Description: "Express.js framework guardrails and patterns", Category: "framework", Tags: []string{"express", "node", "rest"}},
	{Name: "django", Path: ".claude/skills/django", Description: "Django framework guardrails and patterns", Category: "framework", Tags: []string{"django", "python", "orm"}},
	{Name: "fastapi", Path: ".claude/skills/fastapi", Description: "FastAPI framework guardrails and patterns", Category: "framework", Tags: []string{"fastapi", "python", "async"}},
	{Name: "flask", Path: ".claude/skills/flask", Description: "Flask framework guardrails and patterns", Category: "framework", Tags: []string{"flask", "python", "wsgi"}},
	{Name: "gin", Path: ".claude/skills/gin", Description: "Gin framework guardrails and patterns", Category: "framework", Tags: []string{"gin", "go", "golang"}},
	{Name: "echo", Path: ".claude/skills/echo", Description: "Echo framework guardrails and patterns", Category: "framework", Tags: []string{"echo", "go", "golang"}},
	{Name: "fiber", Path: ".claude/skills/fiber", Description: "Fiber framework guardrails and patterns", Category: "framework", Tags: []string{"fiber", "go", "golang"}},
	{Name: "axum", Path: ".claude/skills/axum", Description: "Axum framework guardrails and patterns", Category: "framework", Tags: []string{"axum", "rust", "tower"}},
	{Name: "actix-web", Path: ".claude/skills/actix-web", Description: "Actix-web framework guardrails and patterns", Category: "framework", Tags: []string{"actix-web", "rust", "actor"}},
	{Name: "rocket", Path: ".claude/skills/rocket", Description: "Rocket framework guardrails and patterns", Category: "framework", Tags: []string{"rocket", "rust", "macros"}},
	{Name: "spring-boot-kotlin", Path: ".claude/skills/spring-boot-kotlin", Description: "Spring Boot (Kotlin) framework guardrails and patterns", Category: "framework", Tags: []string{"spring", "kotlin", "jvm"}},
	{Name: "ktor", Path: ".claude/skills/ktor", Description: "Ktor framework guardrails and patterns", Category: "framework", Tags: []string{"ktor", "kotlin", "coroutines"}},
	{Name: "android-compose", Path: ".claude/skills/android-compose", Description: "Android Compose framework guardrails and patterns", Category: "framework", Tags: []string{"compose", "android", "kotlin"}},
	{Name: "spring-boot-java", Path: ".claude/skills/spring-boot-java", Description: "Spring Boot (Java) framework guardrails and patterns", Category: "framework", Tags: []string{"spring", "java", "jvm"}},
	{Name: "quarkus", Path: ".claude/skills/quarkus", Description: "Quarkus framework guardrails and patterns", Category: "framework", Tags: []string{"quarkus", "java", "graalvm"}},
	{Name: "micronaut", Path: ".claude/skills/micronaut", Description: "Micronaut framework guardrails and patterns", Category: "framework", Tags: []string{"micronaut", "java", "di"}},
	{Name: "aspnet-core", Path: ".claude/skills/aspnet-core", Description: "ASP.NET Core framework guardrails and patterns", Category: "framework", Tags: []string{"aspnet", "csharp", "dotnet"}},
	{Name: "blazor", Path: ".claude/skills/blazor", Description: "Blazor framework guardrails and patterns", Category: "framework", Tags: []string{"blazor", "csharp", "wasm"}},
	{Name: "unity", Path: ".claude/skills/unity", Description: "Unity framework guardrails and patterns", Category: "framework", Tags: []string{"unity", "csharp", "game"}},
	{Name: "laravel", Path: ".claude/skills/laravel", Description: "Laravel framework guardrails and patterns", Category: "framework", Tags: []string{"laravel", "php", "eloquent"}},
	{Name: "symfony", Path: ".claude/skills/symfony", Description: "Symfony framework guardrails and patterns", Category: "framework", Tags: []string{"symfony", "php", "doctrine"}},
	{Name: "wordpress", Path: ".claude/skills/wordpress", Description: "WordPress framework guardrails and patterns", Category: "framework", Tags: []string{"wordpress", "php", "cms"}},
	{Name: "swiftui", Path: ".claude/skills/swiftui", Description: "SwiftUI framework guardrails and patterns", Category: "framework", Tags: []string{"swiftui", "swift", "ios"}},
	{Name: "uikit", Path: ".claude/skills/uikit", Description: "UIKit framework guardrails and patterns", Category: "framework", Tags: []string{"uikit", "swift", "ios"}},
	{Name: "vapor", Path: ".claude/skills/vapor", Description: "Vapor framework guardrails and patterns", Category: "framework", Tags: []string{"vapor", "swift", "server"}},
	{Name: "rails", Path: ".claude/skills/rails", Description: "Rails framework guardrails and patterns", Category: "framework", Tags: []string{"rails", "ruby", "activerecord"}},
	{Name: "sinatra", Path: ".claude/skills/sinatra", Description: "Sinatra framework guardrails and patterns", Category: "framework", Tags: []string{"sinatra", "ruby", "dsl"}},
	{Name: "hanami", Path: ".claude/skills/hanami", Description: "Hanami framework guardrails and patterns", Category: "framework", Tags: []string{"hanami", "ruby", "dry-rb"}},
	{Name: "flutter", Path: ".claude/skills/flutter", Description: "Flutter framework guardrails and patterns", Category: "framework", Tags: []string{"flutter", "dart", "mobile"}},
	{Name: "shelf", Path: ".claude/skills/shelf", Description: "Shelf framework guardrails and patterns", Category: "framework", Tags: []string{"shelf", "dart", "http"}},
	{Name: "dart-frog", Path: ".claude/skills/dart-frog", Description: "Dart Frog framework guardrails and patterns", Category: "framework", Tags: []string{"dart-frog", "dart", "server"}},
	// Workflow skills (mirrored from Workflows)
	{Name: "initialize-project", Path: ".claude/skills/initialize-project", Description: "Project setup workflow", Category: "workflow", Tags: []string{"init", "setup", "scaffold"}},
	{Name: "create-rfd", Path: ".claude/skills/create-rfd", Description: "Technical decision documents workflow", Category: "workflow", Tags: []string{"rfd", "decision", "architecture"}},
	{Name: "create-prd", Path: ".claude/skills/create-prd", Description: "Requirements documents workflow", Category: "workflow", Tags: []string{"prd", "requirements", "planning"}},
	{Name: "generate-tasks", Path: ".claude/skills/generate-tasks", Description: "Task breakdown workflow", Category: "workflow", Tags: []string{"tasks", "breakdown", "subtasks"}},
	{Name: "code-review", Path: ".claude/skills/code-review", Description: "Pre-commit quality review workflow", Category: "workflow", Tags: []string{"review", "quality", "pr"}},
	{Name: "security-audit", Path: ".claude/skills/security-audit", Description: "Security assessment workflow", Category: "workflow", Tags: []string{"security", "audit", "owasp"}},
	{Name: "testing-strategy", Path: ".claude/skills/testing-strategy", Description: "Test planning workflow", Category: "workflow", Tags: []string{"testing", "coverage", "tdd"}},
	{Name: "cleanup-project", Path: ".claude/skills/cleanup-project", Description: "Prune unused guides workflow", Category: "workflow", Tags: []string{"cleanup", "prune", "maintenance"}},
	{Name: "refactoring", Path: ".claude/skills/refactoring", Description: "Technical debt remediation workflow", Category: "workflow", Tags: []string{"refactor", "debt", "improve"}},
	{Name: "dependency-update", Path: ".claude/skills/dependency-update", Description: "Safe dependency updates workflow", Category: "workflow", Tags: []string{"dependencies", "update", "upgrade"}},
	{Name: "update-framework", Path: ".claude/skills/update-framework", Description: "Samuel version updates workflow", Category: "workflow", Tags: []string{"samuel", "update", "version"}},
	{Name: "troubleshooting", Path: ".claude/skills/troubleshooting", Description: "Debugging workflow", Category: "workflow", Tags: []string{"debug", "stuck", "error"}},
	{Name: "generate-agents-md", Path: ".claude/skills/generate-agents-md", Description: "Cross-tool compatibility workflow", Category: "workflow", Tags: []string{"agents-md", "compatibility"}},
	{Name: "document-work", Path: ".claude/skills/document-work", Description: "Capture patterns workflow", Category: "workflow", Tags: []string{"document", "patterns", "decisions"}},
	{Name: "create-skill", Path: ".claude/skills/create-skill", Description: "Create Agent Skills workflow", Category: "workflow", Tags: []string{"skill", "create", "scaffold"}},
	{Name: "auto", Path: ".claude/skills/auto", Description: "Autonomous AI coding loop workflow", Category: "workflow", Tags: []string{"auto", "autonomous", "loop", "ralph"}},
	// Anthropic community workflow skills (source: github.com/anthropics/skills)
	{Name: "algorithmic-art", Path: ".claude/skills/algorithmic-art", Description: "Generative art with p5.js workflow", Category: "workflow", Tags: []string{"art", "generative", "p5js"}},
	{Name: "doc-coauthoring", Path: ".claude/skills/doc-coauthoring", Description: "Collaborative document writing workflow", Category: "workflow", Tags: []string{"writing", "docs", "collaboration"}},
	{Name: "frontend-design", Path: ".claude/skills/frontend-design", Description: "Design-thinking for frontend workflow", Category: "workflow", Tags: []string{"design", "frontend", "ui"}},
	{Name: "mcp-builder", Path: ".claude/skills/mcp-builder", Description: "MCP server development workflow", Category: "workflow", Tags: []string{"mcp", "server", "api"}},
	{Name: "theme-factory", Path: ".claude/skills/theme-factory", Description: "Artifact theming toolkit workflow", Category: "workflow", Tags: []string{"theme", "styling", "design"}},
	{Name: "web-artifacts-builder", Path: ".claude/skills/web-artifacts-builder", Description: "React/shadcn web apps workflow", Category: "workflow", Tags: []string{"react", "typescript", "shadcn"}},
	{Name: "webapp-testing", Path: ".claude/skills/webapp-testing", Description: "Playwright web testing workflow", Category: "workflow", Tags: []string{"playwright", "testing", "browser"}},
	{Name: "sync-claude-md", Path: ".claude/skills/sync-claude-md", Description: "Sync per-folder CLAUDE.md files workflow", Category: "workflow", Tags: []string{"sync", "claude-md", "folder"}},
}

// CoreFiles contains essential files always installed
var CoreFiles = []string{
	"CLAUDE.md",
	"AGENTS.md",
	".claude/skills/README.md",
}

// Template represents a predefined set of components
type Template struct {
	Name        string
	Description string
	Languages   []string
	Frameworks  []string
	Workflows   []string
}

// Templates contains predefined installation templates
var Templates = []Template{
	{
		Name:        "full",
		Description: "All guides and workflows (~160 files)",
		Languages:   getAllNames(Languages),
		Frameworks:  getAllNames(Frameworks),
		Workflows:   []string{"all"},
	},
	{
		Name:        "starter",
		Description: "Core files + common languages (TypeScript, Python, Go)",
		Languages:   []string{"typescript", "python", "go"},
		Frameworks:  []string{},
		Workflows:   []string{"all"},
	},
	{
		Name:        "minimal",
		Description: "Just CLAUDE.md and workflows",
		Languages:   []string{},
		Frameworks:  []string{},
		Workflows:   []string{"all"},
	},
}

// FindLanguage finds a language by name
func FindLanguage(name string) *Component {
	for _, lang := range Languages {
		if lang.Name == name {
			return &lang
		}
	}
	return nil
}

// FindFramework finds a framework by name
func FindFramework(name string) *Component {
	for _, fw := range Frameworks {
		if fw.Name == name {
			return &fw
		}
	}
	return nil
}

// FindWorkflow finds a workflow by name
func FindWorkflow(name string) *Component {
	for _, wf := range Workflows {
		if wf.Name == name {
			return &wf
		}
	}
	return nil
}

// FindSkill finds a skill by name
func FindSkill(name string) *Component {
	for _, s := range Skills {
		if s.Name == name {
			return &s
		}
	}
	return nil
}

// InferComponentType resolves a component name to its type by searching the
// three installable categories (Languages, Frameworks, Workflows). Skills are
// deliberately excluded — the registry mirrors every Language/Framework/
// Workflow into Skills as `<name>-guide` entries, so including Skills here
// would cause every component name to "collide" with itself.
//
// Returns:
//   - typ:        canonical singular type ("language", "framework", "workflow")
//                 when exactly one of the three categories contains the name
//   - candidates: type names that match when the name appears in multiple
//                 categories (used to render disambiguation errors)
//   - err:        nil on unique match; non-nil when not found or ambiguous
//
// Used by `samuel add <name>`, `samuel rm <name>`, and `samuel ls <name> --detail`
// to skip the explicit-type argument when the name is unambiguous.
//
// The TestRegistry_NoCrossTypeNameCollisions invariant test in
// internal/core/registry_invariant_test.go enforces that no name appears in
// two of the three slices simultaneously, so the ambiguous branch should
// never fire today. The branch exists so future registry additions that
// accidentally collide fail loudly with a useful message.
func InferComponentType(name string) (typ string, candidates []string, err error) {
	if FindLanguage(name) != nil {
		candidates = append(candidates, "language")
	}
	if FindFramework(name) != nil {
		candidates = append(candidates, "framework")
	}
	if FindWorkflow(name) != nil {
		candidates = append(candidates, "workflow")
	}

	switch len(candidates) {
	case 0:
		return "", nil, fmt.Errorf("component not found: %q", name)
	case 1:
		return candidates[0], candidates, nil
	default:
		return "", candidates, fmt.Errorf(
			"ambiguous: %q exists as %s. Specify the type explicitly.",
			name, joinTypesForError(candidates),
		)
	}
}

// joinTypesForError formats a candidate list for human-readable error messages:
//   ["language", "framework"]            -> "language and framework"
//   ["language", "framework", "workflow"] -> "language, framework, and workflow"
func joinTypesForError(candidates []string) string {
	switch len(candidates) {
	case 0:
		return ""
	case 1:
		return candidates[0]
	case 2:
		return candidates[0] + " and " + candidates[1]
	default:
		// Oxford comma form: "a, b, and c"
		out := ""
		for i, c := range candidates {
			switch {
			case i == 0:
				out = c
			case i == len(candidates)-1:
				out += ", and " + c
			default:
				out += ", " + c
			}
		}
		return out
	}
}

// FindTemplate finds a template by name
func FindTemplate(name string) *Template {
	for _, t := range Templates {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

// GetComponentPaths returns all paths for a given set of components
func GetComponentPaths(languages, frameworks, workflows []string) []string {
	var paths []string

	// Add core files
	paths = append(paths, CoreFiles...)

	// Add languages
	for _, name := range languages {
		if lang := FindLanguage(name); lang != nil {
			paths = append(paths, lang.Path)
		}
	}

	// Add frameworks
	for _, name := range frameworks {
		if fw := FindFramework(name); fw != nil {
			paths = append(paths, fw.Path)
		}
	}

	// Add workflows
	if len(workflows) == 1 && workflows[0] == "all" {
		for _, wf := range Workflows {
			paths = append(paths, wf.Path)
		}
	} else {
		for _, name := range workflows {
			if wf := FindWorkflow(name); wf != nil {
				paths = append(paths, wf.Path)
			}
		}
	}

	return paths
}

func getAllNames(components []Component) []string {
	names := make([]string, len(components))
	for i, c := range components {
		names[i] = c.Name
	}
	return names
}

// GetAllLanguageNames returns all language names
func GetAllLanguageNames() []string {
	return getAllNames(Languages)
}

// GetAllFrameworkNames returns all framework names
func GetAllFrameworkNames() []string {
	return getAllNames(Frameworks)
}

// GetAllWorkflowNames returns all workflow names
func GetAllWorkflowNames() []string {
	return getAllNames(Workflows)
}

// GetAllSkillNames returns all skill names
func GetAllSkillNames() []string {
	return getAllNames(Skills)
}

// GetLanguageSkills returns skills with category "language"
func GetLanguageSkills() []Component {
	var result []Component
	for _, s := range Skills {
		if s.Category == "language" {
			result = append(result, s)
		}
	}
	return result
}

// LanguageToSkillName converts a language name to its skill name (e.g., "go" -> "go-guide")
func LanguageToSkillName(langName string) string {
	return langName + "-guide"
}

// SkillToLanguageName converts a skill name to its language name (e.g., "go-guide" -> "go")
func SkillToLanguageName(skillName string) string {
	if len(skillName) > 6 && skillName[len(skillName)-6:] == "-guide" {
		return skillName[:len(skillName)-6]
	}
	return skillName
}

// FrameworkToSkillName converts a framework name to its skill name.
// For frameworks, the skill name IS the framework name (no suffix).
func FrameworkToSkillName(fwName string) string {
	return fwName
}

// GetFrameworkSkills returns skills with category "framework"
func GetFrameworkSkills() []Component {
	var result []Component
	for _, s := range Skills {
		if s.Category == "framework" {
			result = append(result, s)
		}
	}
	return result
}

// WorkflowToSkillName converts a workflow name to its skill name.
// For workflows, the skill name IS the workflow name (no suffix).
func WorkflowToSkillName(wfName string) string {
	return wfName
}

// GetWorkflowSkills returns skills with category "workflow"
func GetWorkflowSkills() []Component {
	var result []Component
	for _, s := range Skills {
		if s.Category == "workflow" {
			result = append(result, s)
		}
	}
	return result
}

// GetSourcePath returns the source path in the repository for a given destination path
// This prepends the TemplatePrefix to locate files in the downloaded archive
func GetSourcePath(destPath string) string {
	return TemplatePrefix + destPath
}

// GetSourcePaths returns source paths for a slice of destination paths
func GetSourcePaths(destPaths []string) []string {
	srcPaths := make([]string, len(destPaths))
	for i, p := range destPaths {
		srcPaths[i] = GetSourcePath(p)
	}
	return srcPaths
}
