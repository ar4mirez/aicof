# Project Initialization - Detailed Process Reference

Complete templates and examples for project initialization.

---

## Directory Structure Templates

### TypeScript/Node.js

```
project/
├── src/
│   ├── index.ts              # Entry point
│   ├── config/               # Configuration
│   │   ├── database.ts
│   │   ├── env.ts
│   │   └── logger.ts
│   ├── routes/               # API routes
│   │   ├── index.ts
│   │   ├── users.ts
│   │   └── auth.ts
│   ├── services/             # Business logic
│   │   ├── userService.ts
│   │   └── authService.ts
│   ├── models/               # Data models
│   │   ├── User.ts
│   │   └── Session.ts
│   ├── middleware/           # Express middleware
│   │   ├── auth.ts
│   │   ├── errorHandler.ts
│   │   └── validation.ts
│   ├── utils/                # Utilities
│   │   ├── crypto.ts
│   │   └── validation.ts
│   └── types/                # TypeScript types
│       ├── index.ts
│       └── api.ts
├── tests/
│   ├── unit/
│   │   ├── services/
│   │   └── utils/
│   ├── integration/
│   │   └── routes/
│   └── setup.ts
├── package.json
├── tsconfig.json
├── .eslintrc.json
├── .prettierrc
├── .gitignore
├── .env.example
└── README.md
```

### Python

```
project/
├── src/
│   ├── __init__.py
│   ├── main.py               # Entry point
│   ├── config/
│   │   ├── __init__.py
│   │   ├── settings.py
│   │   └── database.py
│   ├── api/                  # API routes
│   │   ├── __init__.py
│   │   ├── routes.py
│   │   └── dependencies.py
│   ├── services/             # Business logic
│   │   ├── __init__.py
│   │   ├── user_service.py
│   │   └── auth_service.py
│   ├── models/               # Data models
│   │   ├── __init__.py
│   │   ├── user.py
│   │   └── session.py
│   ├── schemas/              # Pydantic schemas
│   │   ├── __init__.py
│   │   ├── user.py
│   │   └── auth.py
│   └── utils/                # Utilities
│       ├── __init__.py
│       ├── crypto.py
│       └── validation.py
├── tests/
│   ├── __init__.py
│   ├── unit/
│   │   ├── test_services.py
│   │   └── test_utils.py
│   ├── integration/
│   │   └── test_api.py
│   └── conftest.py
├── requirements.txt
├── requirements-dev.txt
├── pyproject.toml
├── .python-version
├── .gitignore
├── .env.example
└── README.md
```

### Go

```
project/
├── cmd/
│   └── api/
│       └── main.go           # Entry point
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── database.go
│   ├── api/                  # HTTP handlers
│   │   ├── router.go
│   │   ├── users.go
│   │   └── auth.go
│   ├── service/              # Business logic
│   │   ├── user.go
│   │   └── auth.go
│   ├── repository/           # Data access
│   │   ├── user.go
│   │   └── session.go
│   ├── model/                # Data models
│   │   ├── user.go
│   │   └── session.go
│   ├── middleware/
│   │   ├── auth.go
│   │   └── logger.go
│   └── util/
│       ├── crypto.go
│       └── validation.go
├── pkg/                      # Public packages
│   └── types/
│       └── types.go
├── test/
│   ├── integration/
│   │   └── api_test.go
│   └── testutil/
│       └── setup.go
├── go.mod
├── go.sum
├── .golangci.yml
├── .gitignore
├── .env.example
└── README.md
```

### Rust

```
project/
├── src/
│   ├── main.rs               # Entry point
│   ├── lib.rs                # Library root
│   ├── config/
│   │   ├── mod.rs
│   │   ├── settings.rs
│   │   └── database.rs
│   ├── api/                  # API handlers
│   │   ├── mod.rs
│   │   ├── routes.rs
│   │   ├── users.rs
│   │   └── auth.rs
│   ├── service/              # Business logic
│   │   ├── mod.rs
│   │   ├── user.rs
│   │   └── auth.rs
│   ├── repository/           # Data access
│   │   ├── mod.rs
│   │   ├── user.rs
│   │   └── session.rs
│   ├── model/                # Data models
│   │   ├── mod.rs
│   │   ├── user.rs
│   │   └── session.rs
│   ├── middleware/
│   │   ├── mod.rs
│   │   └── auth.rs
│   └── util/
│       ├── mod.rs
│       ├── crypto.rs
│       └── validation.rs
├── tests/
│   ├── integration/
│   │   └── api_test.rs
│   └── common/
│       └── mod.rs
├── Cargo.toml
├── Cargo.lock
├── .clippy.toml
├── .gitignore
├── .env.example
└── README.md
```

---

## Configuration File Templates

### TypeScript: tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "commonjs",
    "lib": ["ES2022"],
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "moduleResolution": "node",
    "allowSyntheticDefaultImports": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true,
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist", "tests"]
}
```

### TypeScript: package.json

```json
{
  "name": "project-name",
  "version": "0.1.0",
  "description": "Project description",
  "main": "dist/index.js",
  "scripts": {
    "dev": "tsx watch src/index.ts",
    "build": "tsc",
    "start": "node dist/index.js",
    "test": "jest",
    "test:watch": "jest --watch",
    "test:cov": "jest --coverage",
    "lint": "eslint src tests --ext .ts",
    "lint:fix": "eslint src tests --ext .ts --fix",
    "format": "prettier --write \"src/**/*.ts\" \"tests/**/*.ts\"",
    "typecheck": "tsc --noEmit"
  },
  "keywords": [],
  "author": "",
  "license": "MIT",
  "devDependencies": {
    "@types/jest": "^29.5.0",
    "@types/node": "^20.0.0",
    "@typescript-eslint/eslint-plugin": "^6.0.0",
    "@typescript-eslint/parser": "^6.0.0",
    "eslint": "^8.50.0",
    "jest": "^29.7.0",
    "prettier": "^3.0.0",
    "ts-jest": "^29.1.0",
    "tsx": "^4.0.0",
    "typescript": "^5.2.0"
  },
  "dependencies": {}
}
```

### TypeScript: .eslintrc.json

```json
{
  "parser": "@typescript-eslint/parser",
  "parserOptions": {
    "ecmaVersion": 2022,
    "sourceType": "module",
    "project": "./tsconfig.json"
  },
  "plugins": ["@typescript-eslint"],
  "extends": [
    "eslint:recommended",
    "plugin:@typescript-eslint/recommended",
    "plugin:@typescript-eslint/recommended-requiring-type-checking"
  ],
  "rules": {
    "@typescript-eslint/explicit-function-return-type": "warn",
    "@typescript-eslint/no-unused-vars": ["error", { "argsIgnorePattern": "^_" }],
    "@typescript-eslint/no-explicit-any": "error",
    "no-console": ["warn", { "allow": ["warn", "error"] }],
    "max-lines-per-function": ["warn", { "max": 50 }],
    "complexity": ["warn", 10]
  }
}
```

### TypeScript: .prettierrc

```json
{
  "semi": true,
  "trailingComma": "es5",
  "singleQuote": true,
  "printWidth": 100,
  "tabWidth": 2,
  "useTabs": false,
  "arrowParens": "avoid"
}
```

### Python: pyproject.toml

```toml
[build-system]
requires = ["setuptools>=61.0"]
build-backend = "setuptools.build_meta"

[project]
name = "project-name"
version = "0.1.0"
description = "Project description"
requires-python = ">=3.11"
dependencies = []

[project.optional-dependencies]
dev = [
    "pytest>=7.4.0",
    "pytest-cov>=4.1.0",
    "black>=23.7.0",
    "isort>=5.12.0",
    "mypy>=1.5.0",
    "ruff>=0.0.285",
]

[tool.black]
line-length = 100
target-version = ['py311']
include = '\.pyi?$'

[tool.isort]
profile = "black"
line_length = 100

[tool.mypy]
python_version = "3.11"
strict = true
warn_return_any = true
warn_unused_configs = true
disallow_untyped_defs = true

[tool.ruff]
line-length = 100
target-version = "py311"
select = [
    "E",   # pycodestyle errors
    "W",   # pycodestyle warnings
    "F",   # pyflakes
    "I",   # isort
    "B",   # flake8-bugbear
    "C4",  # flake8-comprehensions
    "UP",  # pyupgrade
]
ignore = []

[tool.pytest.ini_options]
testpaths = ["tests"]
python_files = "test_*.py"
python_classes = "Test*"
python_functions = "test_*"
addopts = "-v --cov=src --cov-report=term-missing"
```

### Go: .golangci.yml

```yaml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - gosimple
    - ineffassign
    - unused
    - unconvert
    - goconst
    - gocyclo
    - misspell
    - unparam
    - prealloc
    - nakedret
    - goimports

linters-settings:
  gocyclo:
    min-complexity: 10
  goconst:
    min-len: 2
    min-occurrences: 3
  misspell:
    locale: US
  nakedret:
    max-func-lines: 30

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0

run:
  timeout: 5m
```

### Rust: Cargo.toml

```toml
[package]
name = "project-name"
version = "0.1.0"
edition = "2021"
rust-version = "1.70"

[dependencies]

[dev-dependencies]
tokio-test = "0.4"

[profile.dev]
opt-level = 0

[profile.release]
opt-level = 3
lto = true
codegen-units = 1
strip = true

[lints.rust]
unsafe_code = "forbid"
unused_must_use = "deny"
```

### Rust: .clippy.toml

```toml
cognitive-complexity-threshold = 10
too-many-arguments-threshold = 5
```

---

## .gitignore Templates

### TypeScript/Node.js

```
# Dependencies
node_modules/
npm-debug.log*
yarn-debug.log*
yarn-error.log*

# Build artifacts
dist/
build/
*.tsbuildinfo

# Environment
.env
.env.local
.env.*.local

# Testing
coverage/
*.lcov

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Logs
logs/
*.log
```

### Python

```
# Python
__pycache__/
*.py[cod]
*$py.class
*.so

# Virtual environments
venv/
env/
ENV/
.venv

# Build artifacts
dist/
build/
*.egg-info/

# Environment
.env
.env.local

# Testing
.coverage
htmlcov/
.pytest_cache/
.tox/

# IDE
.vscode/
.idea/
*.swp

# OS
.DS_Store
Thumbs.db

# Logs
*.log
logs/
```

### Go

```
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib

# Build artifacts
bin/
dist/
vendor/

# Test coverage
*.out
coverage.txt

# Environment
.env
.env.local

# IDE
.vscode/
.idea/
*.swp

# OS
.DS_Store
Thumbs.db

# Logs
*.log
logs/
```

### Rust

```
# Build artifacts
target/
Cargo.lock  # Commit for binaries, ignore for libraries

# Environment
.env
.env.local

# IDE
.vscode/
.idea/
*.swp

# OS
.DS_Store
Thumbs.db

# Logs
*.log
logs/
```

---

## .env.example Templates

### General

```bash
# Application
NODE_ENV=development
PORT=3000
LOG_LEVEL=info

# Database
DATABASE_URL=postgresql://user:password@localhost:5432/dbname
DB_POOL_SIZE=10

# Redis (if using)
REDIS_URL=redis://localhost:6379

# Authentication
JWT_SECRET=your-secret-key-here-generate-random
JWT_EXPIRES_IN=7d
SESSION_SECRET=your-session-secret-here

# External APIs
API_KEY=your-api-key-here
API_SECRET=your-api-secret-here

# Email (if using)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=user@example.com
SMTP_PASSWORD=your-smtp-password

# AWS (if using)
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-aws-key
AWS_SECRET_ACCESS_KEY=your-aws-secret
S3_BUCKET=your-bucket-name

# Monitoring (if using)
SENTRY_DSN=https://your-sentry-dsn
```

---

## README.md Template

```markdown
# Project Name

Brief description of what this project does.

## Features

- Feature 1
- Feature 2
- Feature 3

## Tech Stack

- **Language**: TypeScript/Python/Go/Rust
- **Framework**: Framework name and version
- **Database**: Database name
- **Testing**: Testing framework
- **Deployment**: Deployment platform

## Prerequisites

- Node.js 20+ / Python 3.11+ / Go 1.21+ / Rust 1.70+
- Database (PostgreSQL/MySQL/etc.)
- Other dependencies

## Setup

1. Clone the repository:
   \`\`\`bash
   git clone <repo-url>
   cd project-name
   \`\`\`

2. Install dependencies:
   \`\`\`bash
   npm install  # or: pip install -r requirements.txt, go mod download, cargo build
   \`\`\`

3. Copy environment variables:
   \`\`\`bash
   cp .env.example .env
   # Edit .env with your values
   \`\`\`

4. Set up database:
   \`\`\`bash
   npm run migrate  # or: python manage.py migrate, etc.
   \`\`\`

5. Start development server:
   \`\`\`bash
   npm run dev  # or: python manage.py runserver, go run ./cmd/api, cargo run
   \`\`\`

## Development

### Running Tests

\`\`\`bash
npm test              # Run all tests
npm run test:watch    # Watch mode
npm run test:cov      # With coverage
\`\`\`

### Linting

\`\`\`bash
npm run lint          # Check for issues
npm run lint:fix      # Auto-fix issues
\`\`\`

### Formatting

\`\`\`bash
npm run format        # Format all files
\`\`\`

### Type Checking (TypeScript)

\`\`\`bash
npm run typecheck     # Check types
\`\`\`

## Project Structure

\`\`\`
src/
├── config/          # Configuration
├── routes/          # API routes
├── services/        # Business logic
├── models/          # Data models
├── middleware/      # Middleware
├── utils/           # Utilities
└── types/           # Type definitions

tests/
├── unit/            # Unit tests
└── integration/     # Integration tests
\`\`\`

## API Documentation

API endpoints are documented at `/api/docs` (if using Swagger/OpenAPI).

### Example Endpoints

- `GET /api/users` - List users
- `POST /api/users` - Create user
- `GET /api/users/:id` - Get user by ID
- `PUT /api/users/:id` - Update user
- `DELETE /api/users/:id` - Delete user

## Deployment

### Production Build

\`\`\`bash
npm run build        # Build for production
npm start            # Start production server
\`\`\`

### Docker (if applicable)

\`\`\`bash
docker build -t project-name .
docker run -p 3000:3000 project-name
\`\`\`

### Environment Variables

See `.env.example` for all required environment variables.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feat/amazing-feature`)
5. Open a Pull Request

## License

[MIT](LICENSE)

## Contact

Your Name - your.email@example.com

Project Link: [https://github.com/username/project-name](https://github.com/username/project-name)
```

---

## Checklist: New Project Initialization

Use this checklist to ensure nothing is missed:

### Discovery Phase
- [ ] Ask tech stack questions
- [ ] Ask architecture questions
- [ ] Ask testing approach questions
- [ ] Ask deployment target questions
- [ ] Ask database questions
- [ ] Ask additional requirements questions

### Directory Setup
- [ ] Create language-appropriate directory structure
- [ ] Create src/ with proper subdirectories
- [ ] Create tests/ directory
- [ ] Create config/ directory (if needed)

### Configuration Files
- [ ] Generate language config (tsconfig.json, pyproject.toml, etc.)
- [ ] Generate linter config (.eslintrc.json, .golangci.yml, etc.)
- [ ] Generate formatter config (.prettierrc, .clippy.toml, etc.)
- [ ] Create package manifest (package.json, requirements.txt, etc.)

### Essential Files
- [ ] Create .gitignore with language-specific entries
- [ ] Create .env.example with secure defaults
- [ ] Create README.md with setup instructions
- [ ] Create LICENSE file (if specified)

### .claude/ Directory
- [ ] Create `CLAUDE.md` with tech stack documentation
- [ ] Copy CLAUDE.md if not present
- [ ] Create `.claude/README.md` if not present

### Verification
- [ ] Dependencies install successfully
- [ ] Tests run (even if empty)
- [ ] Linter runs without errors
- [ ] Dev server starts
- [ ] Build completes (if applicable)

---

## Checklist: Existing Project Adoption

### Analysis Phase
- [ ] Detect language and framework
- [ ] Examine directory structure
- [ ] Analyze code patterns
- [ ] Review git history
- [ ] Check existing tooling
- [ ] Identify gaps

### Documentation Phase
- [ ] Create `CLAUDE.md` with discovered tech stack
- [ ] Create `CLAUDE.md` with observed patterns
- [ ] Document gaps and recommendations
- [ ] List files that need refactoring

### Confirmation Phase
- [ ] User confirms tech stack is correct
- [ ] User confirms architecture pattern
- [ ] User confirms code conventions
- [ ] User identifies legacy code to avoid
- [ ] User identifies protected files

### Improvement Phase
- [ ] Add missing .env.example
- [ ] Add missing configuration files
- [ ] Propose test coverage improvements
- [ ] Propose refactoring plan (incremental)
- [ ] Propose tooling improvements

### .claude/ Directory
- [ ] `CLAUDE.md` created
- [ ] `CLAUDE.md` created
- [ ] Copy CLAUDE.md if not present
- [ ] Create `.claude/README.md` if not present

### Verification
- [ ] Existing tests still pass
- [ ] Existing build still works
- [ ] No breaking changes introduced
- [ ] User confirms readiness to proceed

---

**Remember**: These are reference templates. Adjust based on project requirements and user preferences.
