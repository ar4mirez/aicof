# Dependency Update - Extended Process Reference

Detailed per-ecosystem commands, extended checklists, and verbose examples for safe dependency updates.

---

## Table of Contents

1. [Ecosystem-Specific Commands](#ecosystem-specific-commands)
2. [Extended Update Workflows](#extended-update-workflows)
3. [Automation Configuration](#automation-configuration)
4. [Advanced Scenarios](#advanced-scenarios)
5. [Troubleshooting Guide](#troubleshooting-guide)

---

## Ecosystem-Specific Commands

### Node.js / JavaScript / TypeScript

#### Package Managers

**npm**:
```bash
# List outdated
npm outdated

# Audit security
npm audit
npm audit --json > audit-report.json
npm audit fix  # Auto-fix compatible issues
npm audit fix --force  # Force breaking updates (risky)

# Update specific package
npm install package@version
npm install package@latest

# Update all patches
npm update

# Update all (including minor/major)
npm update --depth 9999

# Check licenses
npx license-checker --summary
npx license-checker --json > licenses.json
```

**Yarn**:
```bash
# List outdated
yarn outdated

# Audit security
yarn audit
yarn audit --json > audit-report.json

# Update specific package
yarn upgrade package@version
yarn upgrade package --latest

# Update all patches
yarn upgrade

# Interactive upgrade
yarn upgrade-interactive
yarn upgrade-interactive --latest

# Check licenses
yarn licenses list
yarn licenses generate-disclaimer > licenses.txt
```

**pnpm**:
```bash
# List outdated
pnpm outdated

# Audit security
pnpm audit
pnpm audit --json > audit-report.json

# Update specific package
pnpm update package@version
pnpm update package --latest

# Update all
pnpm update

# Interactive upgrade
pnpm update --interactive
pnpm update --interactive --latest

# Check licenses
pnpm licenses list
```

#### Lock File Management

```bash
# Regenerate lock file (npm)
rm package-lock.json
npm install

# Regenerate lock file (yarn)
rm yarn.lock
yarn install

# Regenerate lock file (pnpm)
rm pnpm-lock.yaml
pnpm install

# Validate lock file integrity
npm ci  # Fails if lock file doesn't match package.json
```

---

### Python

#### Package Managers

**pip**:
```bash
# List outdated
pip list --outdated
pip list --outdated --format json

# Audit security
pip-audit
pip-audit --format json > audit-report.json
safety check
safety check --json > safety-report.json

# Update specific package
pip install --upgrade package
pip install package==version

# Update all packages
pip list --outdated --format json | jq -r '.[] | .name' | xargs pip install --upgrade

# Check licenses
pip-licenses
pip-licenses --format markdown > licenses.md
pip-licenses --allow-only="MIT;Apache-2.0;BSD"

# Generate requirements with versions
pip freeze > requirements.txt
```

**Poetry**:
```bash
# List outdated
poetry show --outdated

# Audit security
poetry export -f requirements.txt | safety check --stdin

# Update specific package
poetry update package
poetry add package@^version

# Update all
poetry update

# Update lock file only
poetry lock

# Check licenses
poetry show --no-dev | grep License
```

**Pipenv**:
```bash
# List outdated
pipenv update --outdated

# Audit security
pipenv check

# Update specific package
pipenv update package

# Update all
pipenv update

# Lock dependencies
pipenv lock

# Check licenses
pipenv run pip-licenses
```

---

### Go

#### Module Management

```bash
# List outdated modules
go list -u -m all
go list -u -m all | grep '\[' | awk '{print $1, $2, "->", $3}'

# Audit security
govulncheck ./...
govulncheck -json ./... > vulncheck.json

# Update specific module
go get package@version
go get package@latest

# Update direct dependencies
go get -u ./...

# Update all dependencies (including transitive)
go get -u all

# Tidy modules (remove unused)
go mod tidy

# Verify dependencies
go mod verify

# View dependency graph
go mod graph

# Check licenses
go-licenses csv ./... > licenses.csv
go-licenses check ./...

# Download dependencies
go mod download
```

#### Vendoring

```bash
# Vendor dependencies
go mod vendor

# Update vendored dependencies
go get -u all
go mod vendor

# Verify vendored dependencies
go mod verify
```

---

### Rust

#### Cargo Management

```bash
# List outdated crates
cargo outdated
cargo outdated --format json > outdated.json

# Audit security
cargo audit
cargo audit --json > audit.json

# Update specific crate
cargo update -p package
cargo update -p package --precise version

# Update all crates
cargo update

# Update with aggressive strategy
cargo update --aggressive

# Check dependency tree
cargo tree
cargo tree --duplicates

# Check licenses
cargo lichking check
cargo-license

# Build with updated dependencies
cargo build
cargo build --release

# Test with updated dependencies
cargo test
```

---

### Ruby

#### Bundler Management

```bash
# List outdated gems
bundle outdated
bundle outdated --only-explicit

# Audit security
bundle audit check
bundle audit update  # Update vulnerability database

# Update specific gem
bundle update gem-name

# Update all gems
bundle update

# Conservative update (respect version constraints)
bundle update --conservative

# Update to latest minor versions
bundle update --minor

# Update to latest patch versions
bundle update --patch

# Check licenses
bundle-audit
license_finder

# Install dependencies
bundle install

# Regenerate lock file
rm Gemfile.lock
bundle install
```

---

## Extended Update Workflows

### Workflow 1: Security-First Update

When security vulnerabilities are discovered:

```bash
# Step 1: Identify vulnerabilities
npm audit  # or equivalent for your ecosystem

# Step 2: Generate vulnerability report
npm audit --json > vulnerability-report.json

# Step 3: Prioritize by severity
cat vulnerability-report.json | jq '.vulnerabilities | to_entries | map({package: .key, severity: .value.severity})' | jq 'sort_by(.severity)'

# Step 4: Create update branch
git checkout -b security/fix-vulnerabilities-$(date +%Y%m%d)

# Step 5: Update vulnerable packages one at a time
npm install axios@latest
git add package.json package-lock.json
git commit -m "chore(deps): upgrade axios to fix CVE-XXXX-XXXX"

# Step 6: Run tests after each update
npm test

# Step 7: If tests fail, investigate or rollback
git reset --hard HEAD~1

# Step 8: Once all updates complete, run full test suite
npm test
npm run test:e2e
npm run build

# Step 9: Create PR with vulnerability details
```

### Workflow 2: Monthly Maintenance Update

Regular dependency maintenance:

```bash
# Step 1: Create update branch
git checkout -b chore/deps-monthly-$(date +%Y-%m)

# Step 2: Audit current state
npm outdated > outdated-before.txt
npm audit > audit-before.txt

# Step 3: Plan updates by risk level
# Low risk: patches
npm update

# Step 4: Test patches
npm test

# Step 5: Commit patches
git add package.json package-lock.json
git commit -m "chore(deps): update patch versions - $(date +%Y-%m)"

# Step 6: Update minor versions selectively
npm install react@latest react-dom@latest
npm test
git add package.json package-lock.json
git commit -m "chore(deps): update react to 18.3.0"

# Step 7: Document what was updated
npm outdated > outdated-after.txt
diff outdated-before.txt outdated-after.txt

# Step 8: Create PR with summary
```

### Workflow 3: Major Version Update

When updating major versions with breaking changes:

```bash
# Step 1: Research breaking changes
# Read changelog, migration guide, and release notes

# Step 2: Create dedicated branch
git checkout -b feat/upgrade-nextjs-15

# Step 3: Update package
npm install next@15

# Step 4: Identify breaking changes
npm run build 2>&1 | tee build-errors.txt

# Step 5: Fix breaking changes incrementally
# Update code to handle API changes

# Step 6: Test thoroughly
npm test
npm run test:e2e
npm run build

# Step 7: Update documentation
# Update README, API docs, etc.

# Step 8: Create detailed PR
# Include:
# - Breaking changes addressed
# - Migration notes
# - Before/after comparisons
# - Test results
```

---

## Automation Configuration

### Dependabot (GitHub)

**Complete Configuration**:

```yaml
# .github/dependabot.yml
version: 2
updates:
  # JavaScript/TypeScript
  - package-ecosystem: "npm"
    directory: "/"
    schedule:
      interval: "weekly"
      day: "monday"
      time: "06:00"
    open-pull-requests-limit: 10
    groups:
      # Group patch updates together
      patch-updates:
        patterns:
          - "*"
        update-types:
          - "patch"
      # Group minor updates together
      minor-updates:
        patterns:
          - "*"
        update-types:
          - "minor"
    labels:
      - "dependencies"
      - "automated"
    reviewers:
      - "team/reviewers"
    assignees:
      - "maintainer"
    commit-message:
      prefix: "chore(deps)"
    ignore:
      # Ignore major updates for these packages
      - dependency-name: "react"
        update-types: ["version-update:semver-major"]
      - dependency-name: "next"
        update-types: ["version-update:semver-major"]

  # Python
  - package-ecosystem: "pip"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5
    labels:
      - "dependencies"
      - "python"

  # Go
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5
    labels:
      - "dependencies"
      - "golang"

  # GitHub Actions
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "monthly"
    labels:
      - "dependencies"
      - "ci"
```

---

### Renovate

**Complete Configuration**:

```json
// renovate.json
{
  "extends": ["config:base"],
  "schedule": ["before 6am on Monday"],
  "timezone": "America/New_York",
  "labels": ["dependencies"],
  "assignees": ["@maintainer"],
  "reviewers": ["team:reviewers"],
  "packageRules": [
    {
      "description": "Auto-merge patch updates",
      "matchUpdateTypes": ["patch"],
      "automerge": true,
      "automergeType": "pr",
      "automergeStrategy": "squash"
    },
    {
      "description": "Group minor updates weekly",
      "matchUpdateTypes": ["minor"],
      "groupName": "minor dependencies",
      "schedule": ["before 6am on Monday"]
    },
    {
      "description": "Require manual review for major updates",
      "matchUpdateTypes": ["major"],
      "labels": ["breaking-change", "major-update"],
      "automerge": false,
      "schedule": ["before 6am on first day of month"]
    },
    {
      "description": "Separate security updates",
      "matchDatasources": ["npm"],
      "matchDepTypes": ["dependencies"],
      "vulnerabilityAlerts": {
        "labels": ["security"],
        "automerge": true
      }
    },
    {
      "description": "Group React ecosystem",
      "groupName": "React",
      "matchPackagePatterns": ["^react", "^@types/react"],
      "matchUpdateTypes": ["patch", "minor"]
    },
    {
      "description": "Pin development dependencies",
      "matchDepTypes": ["devDependencies"],
      "rangeStrategy": "pin"
    },
    {
      "description": "Widen production dependencies",
      "matchDepTypes": ["dependencies"],
      "rangeStrategy": "widen"
    },
    {
      "description": "Ignore experimental packages",
      "matchPackagePatterns": ["^@experimental/"],
      "enabled": false
    }
  ],
  "ignoreDeps": ["webpack"],
  "prConcurrentLimit": 10,
  "prHourlyLimit": 2,
  "commitMessagePrefix": "chore(deps):",
  "semanticCommits": "enabled",
  "vulnerabilityAlerts": {
    "enabled": true,
    "schedule": ["at any time"]
  }
}
```

---

### GitHub Actions Workflow

**Automated Dependency Update CI**:

```yaml
# .github/workflows/dependency-update.yml
name: Dependency Update CI

on:
  pull_request:
    branches: [main]
    paths:
      - 'package.json'
      - 'package-lock.json'
      - 'yarn.lock'
      - 'pnpm-lock.yaml'
      - 'requirements.txt'
      - 'poetry.lock'
      - 'go.mod'
      - 'go.sum'
      - 'Cargo.toml'
      - 'Cargo.lock'

jobs:
  validate-updates:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'

      - name: Install dependencies
        run: npm ci

      - name: Run tests
        run: npm test

      - name: Run type check
        run: npm run typecheck

      - name: Run linter
        run: npm run lint

      - name: Run build
        run: npm run build

      - name: Security audit
        run: npm audit --audit-level=moderate

      - name: License check
        run: |
          npx license-checker --onlyAllow "MIT;ISC;Apache-2.0;BSD-2-Clause;BSD-3-Clause"

      - name: Check bundle size
        run: npm run analyze-bundle

      - name: Comment PR with results
        uses: actions/github-script@v7
        if: always()
        with:
          script: |
            const output = `
            #### Dependency Update Validation
            - ✅ Tests passed
            - ✅ Type check passed
            - ✅ Lint passed
            - ✅ Build succeeded
            - ✅ No security vulnerabilities
            - ✅ All licenses compatible
            `;
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: output
            });

  security-audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run Snyk security scan
        uses: snyk/actions/node@master
        env:
          SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
        with:
          command: test

      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          format: 'sarif'
          output: 'trivy-results.sarif'

      - name: Upload Trivy results to GitHub Security
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: 'trivy-results.sarif'
```

---

## Advanced Scenarios

### Scenario 1: Resolving Dependency Conflicts

**Problem**: Package A requires library@^1.0.0, but Package B requires library@^2.0.0

**Solution**:

```bash
# Check dependency tree
npm ls library

# Option 1: Use resolutions (Yarn/npm)
# package.json
{
  "resolutions": {
    "library": "^2.0.0"
  }
}

# Option 2: Use overrides (npm 8+)
# package.json
{
  "overrides": {
    "library": "^2.0.0"
  }
}

# Option 3: Update conflicting package
npm install package-a@latest

# Verify resolution
npm ls library
```

### Scenario 2: Migrating Between Package Managers

**npm → pnpm**:

```bash
# Install pnpm
npm install -g pnpm

# Import from package-lock.json
pnpm import

# Verify
pnpm install
pnpm test

# Clean up
rm package-lock.json
git add pnpm-lock.yaml package.json
```

**npm → Yarn**:

```bash
# Install Yarn
npm install -g yarn

# Yarn will read package-lock.json automatically
yarn install

# Verify
yarn test

# Clean up
rm package-lock.json
git add yarn.lock package.json
```

### Scenario 3: Dealing with Breaking Changes

**Step-by-step migration**:

```bash
# 1. Read changelog
curl https://github.com/owner/package/releases/tag/v2.0.0

# 2. Check codemod availability
npx @package/codemod

# 3. Create migration branch
git checkout -b feat/migrate-package-v2

# 4. Update package
npm install package@2.0.0

# 5. Run codemod if available
npx @package/codemod@latest

# 6. Fix remaining issues manually
# Review build errors and fix

# 7. Update tests
npm test

# 8. Update documentation
```

### Scenario 4: Rollback Failed Update

**Quick rollback**:

```bash
# Option 1: Git revert
git revert HEAD
npm install

# Option 2: Git reset (if not pushed)
git reset --hard HEAD~1
npm install

# Option 3: Manual rollback
git checkout HEAD~1 -- package.json package-lock.json
npm install

# Verify rollback
npm test
npm run build
```

**Emergency production rollback**:

```bash
# 1. Revert merge commit
git revert -m 1 <merge-commit-hash>

# 2. Deploy revert immediately
npm install
npm run build
./deploy.sh production

# 3. Investigate root cause
git diff <merge-commit-hash> HEAD

# 4. Create hotfix if needed
git checkout -b hotfix/rollback-package-update
```

---

## Troubleshooting Guide

### Issue 1: Lock File Conflicts

**Symptoms**: Merge conflicts in package-lock.json

**Solution**:
```bash
# Accept their version
git checkout --theirs package-lock.json
npm install

# Or regenerate
git checkout --ours package.json
rm package-lock.json
npm install
```

### Issue 2: Dependency Resolution Errors

**Symptoms**: "Cannot find module" or "Peer dependency mismatch"

**Solution**:
```bash
# Clear cache
npm cache clean --force

# Remove node_modules
rm -rf node_modules

# Regenerate lock file
rm package-lock.json
npm install

# If still failing, check peer dependencies
npm ls <package>
```

### Issue 3: Security Audit False Positives

**Symptoms**: Vulnerability in dev dependency that doesn't affect production

**Solution**:
```bash
# Audit production only
npm audit --production

# Ignore specific vulnerability (temporary)
npm audit fix --force

# Or add to .npmrc (not recommended)
audit-level=moderate
```

### Issue 4: Tests Fail After Update

**Symptoms**: Previously passing tests now fail

**Solution**:
```bash
# 1. Identify which update caused failure
git bisect start
git bisect bad HEAD
git bisect good <last-working-commit>

# 2. For each bisect step
npm install
npm test
git bisect good  # or git bisect bad

# 3. Once identified
git bisect reset
git show <problematic-commit>

# 4. Fix or rollback that specific package
```

### Issue 5: Build Size Increased

**Symptoms**: Bundle size significantly increased after update

**Solution**:
```bash
# Analyze bundle
npm run analyze-bundle

# Check what changed
npm ls <package>

# Options:
# 1. Use tree-shaking
# 2. Import specific modules only
# 3. Consider alternative package
# 4. Use dynamic imports
```

---

## Best Practices Summary

### DO

- ✅ Read changelogs before updating
- ✅ Update one package at a time for major versions
- ✅ Run full test suite after each update
- ✅ Check license compatibility
- ✅ Document breaking changes
- ✅ Create atomic commits
- ✅ Have rollback plan ready
- ✅ Test in staging before production

### DON'T

- ❌ Update all packages blindly
- ❌ Skip security audits
- ❌ Ignore breaking changes
- ❌ Commit without testing
- ❌ Update production dependencies without review
- ❌ Ignore peer dependency warnings
- ❌ Update during critical periods
- ❌ Batch major updates together

---

## Quick Command Reference

| Task | Node.js | Python | Go | Rust | Ruby |
|------|---------|--------|----|------|------|
| **List outdated** | `npm outdated` | `pip list --outdated` | `go list -u -m all` | `cargo outdated` | `bundle outdated` |
| **Security audit** | `npm audit` | `pip-audit` | `govulncheck ./...` | `cargo audit` | `bundle audit` |
| **Update specific** | `npm install pkg@ver` | `pip install pkg==ver` | `go get pkg@ver` | `cargo update -p pkg` | `bundle update pkg` |
| **Update all** | `npm update` | `pip install -U -r requirements.txt` | `go get -u ./...` | `cargo update` | `bundle update` |
| **Check licenses** | `npx license-checker` | `pip-licenses` | `go-licenses` | `cargo-license` | `license_finder` |
| **Lock file** | `package-lock.json` | `requirements.txt` | `go.sum` | `Cargo.lock` | `Gemfile.lock` |

---

## Related Resources

- [Semantic Versioning](https://semver.org/)
- [Dependabot Documentation](https://docs.github.com/en/code-security/dependabot)
- [Renovate Documentation](https://docs.renovatebot.com/)
- [npm Security Best Practices](https://docs.npmjs.com/security-best-practices)
- [OWASP Dependency Check](https://owasp.org/www-project-dependency-check/)
