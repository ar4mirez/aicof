# Update Framework - Detailed Process Reference

This document contains detailed migration steps, conflict resolution examples, and comprehensive troubleshooting procedures for updating Samuel.

---

## Table of Contents

1. [Detailed Migration Steps](#detailed-migration-steps)
2. [Conflict Resolution](#conflict-resolution)
3. [CLAUDE.md Merge Strategies](#claudemd-merge-strategies)
4. [Rollback Procedures](#rollback-procedures)
5. [Troubleshooting](#troubleshooting)
6. [Advanced Scenarios](#advanced-scenarios)

---

## Detailed Migration Steps

### Step-by-Step Full Replace Procedure

#### Step 1: Pre-Update Assessment

```bash
# 1. Check for uncommitted changes
git status

# 2. Create a pre-update branch (recommended)
git checkout -b pre-update-backup

# 3. Document current version
echo "Current version: $(grep 'Current Version' CLAUDE.md)"

# 4. List all customized files
git diff --name-only origin/main CLAUDE.md .claude/

# 5. Save list of installed skills
ls .claude/skills/ > .ai-skills-before.txt
```

#### Step 2: Backup Creation

```bash
# 1. Create timestamped backup directory
BACKUP_DIR=".ai-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

# 2. Backup CLAUDE.md with diff
cp CLAUDE.md "$BACKUP_DIR/"
git diff origin/main CLAUDE.md > "$BACKUP_DIR/CLAUDE.md.diff" 2>/dev/null || true

# 3. Backup entire .claude/ directory
cp -r .claude "$BACKUP_DIR/"

# 4. Create backup manifest
cat > "$BACKUP_DIR/manifest.txt" <<EOF
Backup created: $(date)
Current version: $(grep 'Current Version' CLAUDE.md)
Git commit: $(git rev-parse HEAD 2>/dev/null || echo "N/A")
Files backed up:
  - CLAUDE.md
  - .claude/
EOF

# 5. Verify backup
ls -lah "$BACKUP_DIR"
```

#### Step 3: Fetch Latest Version

```bash
# 1. Clone to temporary directory
TEMP_DIR=".ai-update-temp-$(date +%Y%m%d-%H%M%S)"
git clone --depth 1 https://github.com/ar4mirez/samuel.git "$TEMP_DIR"

# 2. Check what version we're getting
echo "Latest version: $(grep 'Current Version' $TEMP_DIR/template/CLAUDE.md)"

# 3. Review changelog
cat "$TEMP_DIR/CHANGELOG.md"

# 4. Copy template files to accessible location
cp -r "$TEMP_DIR/template/"* "$TEMP_DIR/"
```

#### Step 4: Compare Versions

```bash
# 1. Compare CLAUDE.md structure
diff -u CLAUDE.md "$TEMP_DIR/CLAUDE.md" > claude-diff.txt || true

# 2. Compare .claude/ directory structure
diff -qr .claude/ "$TEMP_DIR/.claude/" > agent-diff.txt || true

# 3. Identify new skills
comm -13 <(ls .claude/skills/ | sort) <(ls "$TEMP_DIR/.claude/skills/" | sort) > new-skills.txt

# 4. Identify new workflows
comm -13 <(ls .claude/skills/ | sort) <(ls "$TEMP_DIR/.claude/skills/" | sort) > new-workflows.txt

# 5. Review differences
echo "=== New Skills ==="
cat new-skills.txt
echo "=== New Workflows ==="
cat new-workflows.txt
```

#### Step 5: Execute Update

```bash
# 1. Preserve project-specific files
PROJECT_FILES="project.md patterns.md state.md"
mkdir -p .ai-preserve

for file in $PROJECT_FILES; do
  if [ -f ".claude/$file" ]; then
    cp ".claude/$file" .ai-preserve/
  fi
done

# Preserve memory and tasks
if [ -d ".claude/memory" ]; then
  cp -r .claude/memory .ai-preserve/
fi
if [ -d ".claude/tasks" ]; then
  cp -r .claude/tasks .ai-preserve/
fi

# 2. Replace template files
cp "$TEMP_DIR/CLAUDE.md" ./
rm -rf .claude/skills/ .claude/skills/
cp -r "$TEMP_DIR/.claude/skills" .claude/
cp -r "$TEMP_DIR/.claude/workflows" .claude/

# 3. Restore project-specific files
for file in $PROJECT_FILES; do
  if [ -f ".ai-preserve/$file" ]; then
    cp ".ai-preserve/$file" ".claude/"
  fi
done

if [ -d ".ai-preserve/memory" ]; then
  cp -r .ai-preserve/memory/* .claude/memory/
fi
if [ -d ".ai-preserve/tasks" ]; then
  cp -r .ai-preserve/tasks/* .claude/tasks/
fi

# 4. Create README files if they don't exist
if [ ! -f ".claude/README.md" ] && [ -f "$TEMP_DIR/.claude/README.md" ]; then
  cp "$TEMP_DIR/.claude/README.md" .claude/
fi
```

#### Step 6: Merge CLAUDE.md Customizations

If CLAUDE.md was customized:

```bash
# 1. Extract your custom sections
# Common customization areas:
# - Custom guardrails in "Core Guardrails" section
# - Company-specific in "Operations" section
# - Custom workflows in "Quick Reference"

# 2. Use AI to help merge
# Load both versions and ask AI to identify and merge customizations

# 3. Verify version number is updated
grep "Current Version" CLAUDE.md
```

#### Step 7: Post-Update Verification

```bash
# 1. Check new version is installed
echo "New version: $(grep 'Current Version' CLAUDE.md)"

# 2. Verify all skills are present
ls .claude/skills/ > .ai-skills-after.txt
diff .ai-skills-before.txt .ai-skills-after.txt

# 3. Verify project files are preserved
for file in project.md patterns.md state.md; do
  if [ -f ".claude/$file" ]; then
    echo "✓ .claude/$file preserved"
  fi
done

# 4. Check memory and tasks
echo "Memory files: $(ls .claude/memory/ 2>/dev/null | wc -l)"
echo "Task files: $(ls .claude/tasks/ 2>/dev/null | wc -l)"

# 5. Clean up temporary files
rm -rf "$TEMP_DIR"
rm -rf .ai-preserve
rm -f .ai-skills-before.txt .ai-skills-after.txt
rm -f claude-diff.txt agent-diff.txt new-skills.txt new-workflows.txt
```

---

## Conflict Resolution

### Common Conflict Scenarios

#### Scenario 1: Custom Guardrails in CLAUDE.md

**Problem**: You've added company-specific guardrails to CLAUDE.md

**Solution**:
1. Keep the new template version as base
2. Extract your custom guardrails from backup
3. Add them to appropriate section
4. Mark them clearly as custom

**Example**:
```markdown
### Code Quality
<!-- Standard guardrails from template -->
- ✓ No function exceeds 50 lines
- ✓ No file exceeds 300 lines
...

<!-- CUSTOM: Company-specific guardrails -->
- ✓ All API responses must include request ID for tracing
- ✓ All database queries must log execution time
```

#### Scenario 2: Modified Workflow Files

**Problem**: You've customized `.claude/skills/code-review.md` for your team's process

**Solution**:
1. Keep your customized version
2. Check what changed in the new template version
3. Manually merge new improvements
4. Document your customizations

**Example**:
```bash
# Compare versions
diff .ai-backup/.claude/skills/code-review.md \
     .ai-update-temp/.claude/skills/code-review.md

# If template has valuable updates, merge manually
# Keep your workflow, add new template checks
```

#### Scenario 3: Removed or Renamed Files

**Problem**: A file you customized was removed or renamed in the new version

**Solution**:
1. Check CHANGELOG.md for explanation
2. If file was renamed, update references
3. If file was removed, migrate content to new location
4. Document the change in `.claude/memory/`

**Example - File was renamed**:
```bash
# Old: .claude/language-guides/typescript.md
# New: .claude/skills/typescript-guide/SKILL.md

# If you had customizations in old file:
# 1. Copy your customizations from backup
# 2. Add them to new location's references/ directory
# 3. Update any references in your code
```

#### Scenario 4: Version Jump with Breaking Changes

**Problem**: Jumping multiple versions (e.g., 1.3.0 → 1.7.0) with breaking changes

**Solution**:
1. Read all changelogs between versions
2. Apply updates incrementally if possible
3. Test after each major change
4. Document migration path

**Example Migration Path**:
```bash
# Current: 1.3.0
# Target: 1.7.0
# Breaking changes in: 1.5.0 (framework-guides → skills), 1.8.0 (skills standard)

# Step 1: Update to 1.5.0
git clone --branch v1.5.0 https://github.com/ar4mirez/samuel.git
# Apply 1.5.0 changes
# Test

# Step 2: Update to 1.7.0
git clone --branch v1.7.0 https://github.com/ar4mirez/samuel.git
# Apply 1.7.0 changes
# Test

# Step 3: Update to 1.8.0
git clone --branch v1.8.0 https://github.com/ar4mirez/samuel.git
# Apply 1.8.0 changes (skills standard)
# Test
```

---

## CLAUDE.md Merge Strategies

### Strategy 1: Section-by-Section Merge

Best for: Heavy customizations throughout CLAUDE.md

```bash
# 1. Keep your CLAUDE.md as base
cp .ai-backup/CLAUDE.md CLAUDE.md

# 2. For each section, compare with template
# Operations section - usually safe to replace
# Boundaries section - usually safe to replace
# Core Guardrails - merge custom guardrails
# 4D Methodology - usually safe to replace
# Version & Changelog - take from template

# 3. Update version number and changelog
# Replace "Version & Changelog" section with template version
```

### Strategy 2: Template-First Merge

Best for: Light customizations in specific sections

```bash
# 1. Take new template as base
cp .ai-update-temp/CLAUDE.md CLAUDE.md

# 2. Extract your customizations from backup
grep -A 20 "CUSTOM:" .ai-backup/CLAUDE.md > custom-sections.txt

# 3. Add customizations to appropriate sections
# Mark them clearly with <!-- CUSTOM: --> comments

# 4. Verify version is correct
grep "Current Version" CLAUDE.md
```

### Strategy 3: Diff-Based Merge

Best for: Understanding exact changes made

```bash
# 1. Create three-way diff
diff -u .ai-backup/CLAUDE.md CLAUDE.md > your-changes.diff
diff -u .ai-backup/CLAUDE.md .ai-update-temp/CLAUDE.md > template-changes.diff

# 2. Review both diffs side by side
# Identify conflicts

# 3. Apply non-conflicting changes automatically
patch CLAUDE.md < your-changes.diff

# 4. Manually resolve conflicts
# Use AI to help identify and merge
```

### Strategy 4: Modular Approach

Best for: Large version jumps or complex customizations

```bash
# 1. Keep your CLAUDE.md untouched
# 2. Create custom sections in separate files

# Example structure:
.claude/
  custom/
    guardrails.md     # Your custom guardrails
    operations.md     # Company-specific commands
    workflows.md      # Custom workflow links

# 3. Reference custom files from CLAUDE.md
# Add at end of CLAUDE.md:
## Custom Extensions
See `.claude/custom/` for company-specific additions:
- Custom guardrails: .claude/custom/guardrails.md
- Custom operations: .claude/custom/operations.md
- Custom workflows: .claude/custom/workflows.md

# 4. Update template sections normally
# Your customizations remain in separate files
```

---

## Rollback Procedures

### Immediate Rollback (Update Just Completed)

```bash
# If backup still exists
BACKUP_DIR=".ai-backup-<timestamp>"

# Restore everything
cp "$BACKUP_DIR/CLAUDE.md" ./
rm -rf .claude/
cp -r "$BACKUP_DIR/.claude" ./

# Verify rollback
grep "Current Version" CLAUDE.md
```

### Git-Based Rollback

```bash
# If changes were committed
git log --oneline -5  # Find the commit before update

# Option 1: Revert the commit
git revert <commit-hash>

# Option 2: Reset to previous state (if not pushed)
git reset --hard <commit-hash-before-update>

# Option 3: Restore specific files
git checkout <commit-hash-before-update> -- CLAUDE.md .claude/
```

### Partial Rollback

```bash
# Rollback only CLAUDE.md
cp .ai-backup/CLAUDE.md ./

# Rollback specific skills
cp -r .ai-backup/.claude/skills/typescript-guide/ .claude/skills/

# Rollback specific workflow
cp .ai-backup/.claude/skills/code-review.md .claude/skills/
```

### Complete Reinstall

```bash
# 1. Remove everything
rm CLAUDE.md
rm -rf .claude/skills/ .claude/skills/

# 2. Clone specific version
git clone --branch v1.6.0 https://github.com/ar4mirez/samuel.git temp
cp temp/template/CLAUDE.md ./
cp -r temp/template/.claude/skills .claude/
cp -r temp/template/.claude/workflows .claude/
rm -rf temp

# 3. Restore project files from backup
cp .ai-backup/CLAUDE.md .claude/
cp .ai-backup/CLAUDE.md .claude/
cp -r .ai-backup/.claude/memory .claude/
cp -r .ai-backup/.claude/tasks .claude/
```

---

## Troubleshooting

### Problem: Update Failed Midway

**Symptoms**:
- Partial files copied
- Some skills missing
- Error messages during copy

**Diagnosis**:
```bash
# Check what's missing
ls .claude/skills/
ls .claude/skills/

# Check for partial files
find .claude/ -type f -size 0

# Check disk space
df -h .
```

**Solution**:
```bash
# 1. Stop the update process
# 2. Restore from backup
cp -r .ai-backup/* ./

# 3. Clear any partial temporary files
rm -rf .ai-update-temp*

# 4. Check system resources
df -h .           # Disk space
free -h           # Memory (Linux)
top               # Process usage

# 5. Try update again with verbose output
cp -v .ai-update-temp/CLAUDE.md ./
cp -rv .ai-update-temp/.claude/* .claude/
```

### Problem: Merge Conflicts in CLAUDE.md

**Symptoms**:
- New version missing your customizations
- Unclear which sections to keep
- Breaking changes not documented

**Diagnosis**:
```bash
# Create detailed diff
diff -u --color=always .ai-backup/CLAUDE.md CLAUDE.md | less -R

# Find your custom sections
grep -n "CUSTOM\|TODO\|FIXME\|COMPANY" .ai-backup/CLAUDE.md
```

**Solution**:
```bash
# 1. Extract your customizations
grep -B 5 -A 20 "CUSTOM" .ai-backup/CLAUDE.md > my-custom-sections.txt

# 2. Use AI to merge
# Load both CLAUDE.md versions
# Ask AI: "Merge these custom sections into the new template"

# 3. Verify critical customizations preserved
diff .ai-backup/CLAUDE.md CLAUDE.md | grep "^<.*✓"  # Your guardrails
diff .ai-backup/CLAUDE.md CLAUDE.md | grep "^<.*##"  # Your sections
```

### Problem: Missing Skills After Update

**Symptoms**:
- Skills directory empty or incomplete
- Language/framework skills missing
- AI can't load skills

**Diagnosis**:
```bash
# Check what should be there
ls .ai-update-temp/.claude/skills/

# Check what you have
ls .claude/skills/

# Find the difference
comm -13 <(ls .claude/skills/ | sort) \
         <(ls .ai-update-temp/.claude/skills/ | sort)
```

**Solution**:
```bash
# Re-copy missing skills
for skill in $(comm -13 <(ls .claude/skills/ | sort) \
                        <(ls .ai-update-temp/.claude/skills/ | sort)); do
  echo "Copying missing skill: $skill"
  cp -r ".ai-update-temp/.claude/skills/$skill" .claude/skills/
done

# Verify all skills present
ls .claude/skills/ | wc -l
```

### Problem: AI Not Recognizing New Version

**Symptoms**:
- AI still references old version
- New skills not accessible
- Workflows not found

**Diagnosis**:
```bash
# Verify file locations
test -f CLAUDE.md && echo "CLAUDE.md exists" || echo "CLAUDE.md missing"
test -d .claude/skills && echo ".claude/skills exists" || echo "Skills missing"

# Check file permissions
ls -la CLAUDE.md
ls -la .claude/

# Verify content
head -20 CLAUDE.md
grep "Current Version" CLAUDE.md
```

**Solution**:
```bash
# 1. Verify file is in correct location
pwd  # Should be project root
ls CLAUDE.md .claude/

# 2. Check file permissions
chmod 644 CLAUDE.md
chmod -R 755 .claude/

# 3. Restart AI session
# Exit and start new conversation

# 4. Test loading
# Ask AI: "What version of Samuel is installed?"
# Ask AI: "List available skills"
```

### Problem: Backup Directory Too Large

**Symptoms**:
- Backup taking too long
- Disk space warnings
- Slow copy operations

**Diagnosis**:
```bash
# Check backup size
du -sh .ai-backup*

# Find large files
find .ai-backup* -type f -size +10M

# Check what's consuming space
du -sh .ai-backup*/*
```

**Solution**:
```bash
# Exclude unnecessary files from backup
BACKUP_DIR=".ai-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Backup only essential files
cp CLAUDE.md "$BACKUP_DIR/"
cp CLAUDE.md "$BACKUP_DIR/" 2>/dev/null || true
cp CLAUDE.md "$BACKUP_DIR/" 2>/dev/null || true
cp CLAUDE.md "$BACKUP_DIR/" 2>/dev/null || true

# Backup memory and tasks with exclusions
rsync -av --exclude='*.tmp' --exclude='*.log' \
  .claude/memory/ "$BACKUP_DIR/memory/"
rsync -av --exclude='*.tmp' --exclude='*.log' \
  .claude/tasks/ "$BACKUP_DIR/tasks/"

# Don't backup skills and workflows (can be restored from template)
```

---

## Advanced Scenarios

### Scenario 1: Multi-Repository Sync

**Use Case**: Maintaining Samuel across multiple projects

**Solution**:
```bash
# 1. Update one project as reference
cd project-1
# Perform standard update

# 2. Create update script
cat > sync-samuel.sh <<'EOF'
#!/bin/bash
PROJECTS=(
  "/path/to/project-1"
  "/path/to/project-2"
  "/path/to/project-3"
)

for project in "${PROJECTS[@]}"; do
  echo "Updating $project"
  cd "$project"

  # Backup
  cp CLAUDE.md ".ai-backup-CLAUDE.md"

  # Update
  cp /path/to/project-1/CLAUDE.md ./
  rsync -av /path/to/project-1/.claude/skills/ .claude/skills/
  rsync -av /path/to/project-1/.claude/skills/ .claude/skills/

  # Preserve project-specific files
  git checkout CLAUDE.md CLAUDE.md

  echo "Updated $project to version $(grep 'Current Version' CLAUDE.md)"
done
EOF

chmod +x sync-samuel.sh
./sync-samuel.sh
```

### Scenario 2: Custom Skill Development

**Use Case**: You've created custom skills that need to coexist with template skills

**Solution**:
```bash
# 1. Namespace your custom skills
.claude/
  skills/
    # Template skills (updated from Samuel)
    typescript-guide/
    react/

    # Your custom skills (never overwritten)
    company-auth/
    company-api-standards/
    company-deployment/

# 2. During update, exclude your custom skills
rsync -av --exclude='company-*' \
  .ai-update-temp/.claude/skills/ .claude/skills/

# 3. Add your custom skills to .gitignore pattern
echo ".claude/skills/company-*" >> .samuel-custom
```

### Scenario 3: Forked Template

**Use Case**: You've forked Samuel for company-wide customization

**Solution**:
```bash
# 1. Set up remotes
git remote add upstream https://github.com/ar4mirez/samuel.git
git remote -v

# 2. Fetch upstream changes
git fetch upstream

# 3. Merge upstream changes into your fork
git checkout main
git merge upstream/main

# 4. Resolve conflicts (preserve your customizations)
# Use git mergetool or manual editing

# 5. Push to your company fork
git push origin main

# 6. Update projects from your fork
cd ~/project
git subtree pull --prefix=.ai-template \
  https://github.com/yourcompany/samuel-custom.git main --squash
```

### Scenario 4: Continuous Integration

**Use Case**: Automate Samuel version checks in CI/CD

**Solution**:
```yaml
# .github/workflows/check-samuel-version.yml
name: Check Samuel Version

on:
  schedule:
    - cron: '0 0 * * 1'  # Weekly on Monday
  workflow_dispatch:

jobs:
  check-version:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Get current version
        id: current
        run: |
          VERSION=$(grep 'Current Version' CLAUDE.md | cut -d: -f2 | tr -d ' ')
          echo "version=$VERSION" >> $GITHUB_OUTPUT

      - name: Get latest version
        id: latest
        run: |
          VERSION=$(curl -s https://raw.githubusercontent.com/ar4mirez/samuel/main/template/CLAUDE.md |
                    grep 'Current Version' | cut -d: -f2 | tr -d ' ')
          echo "version=$VERSION" >> $GITHUB_OUTPUT

      - name: Compare versions
        run: |
          if [ "${{ steps.current.outputs.version }}" != "${{ steps.latest.outputs.version }}" ]; then
            echo "⚠️ Samuel update available!"
            echo "Current: ${{ steps.current.outputs.version }}"
            echo "Latest: ${{ steps.latest.outputs.version }}"
            echo "Run: update-framework skill"
          else
            echo "✓ Samuel is up to date (${{ steps.current.outputs.version }})"
          fi
```

---

## Post-Update Maintenance

### Cleanup Tasks

```bash
# 1. Remove old backup directories (keep most recent)
ls -dt .ai-backup-* | tail -n +2 | xargs rm -rf

# 2. Clean up any residual temporary files
rm -f .ai-skills-*.txt
rm -f *-diff.txt

# 3. Verify git status
git status

# 4. Commit the update
git add CLAUDE.md .claude/
git commit -m "chore: update Samuel to v$(grep 'Current Version' CLAUDE.md | cut -d: -f2 | tr -d ' ')

Updated Samuel framework to latest version:
- New skills added
- Workflows updated
- Project customizations preserved

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Documentation Tasks

```bash
# 1. Document the update in memory
cat > .claude/memory/$(date +%Y-%m-%d)-samuel-update.md <<EOF
# Samuel Update: v$OLD_VERSION → v$NEW_VERSION

## Date
$(date)

## Changes Applied
- Updated CLAUDE.md core file
- Added N new skills
- Updated M workflows
- Preserved all project customizations

## Customizations Merged
- [ List any custom sections merged ]

## Breaking Changes
- [ List any breaking changes handled ]

## Testing
- [x] AI loads CLAUDE.md correctly
- [x] All skills accessible
- [x] Workflows function correctly
- [x] Project files preserved

## Notes
[ Any notes for future updates ]
EOF

# 2. Update project.md if structure changed
# 3. Notify team of update
# 4. Schedule cleanup workflow run
```

---

## Version-Specific Migration Guides

### Migrating to v1.8.0 (Agent Skills)

**Breaking Changes**:
- Language guides moved from `language-guides/` to `skills/*/SKILL.md`
- Framework guides moved from `framework-guides/` to `skills/*/SKILL.md`
- Skills now follow Agent Skills standard

**Migration Steps**:
```bash
# 1. Backup old guides if customized
if [ -d ".claude/language-guides" ]; then
  cp -r .claude/language-guides .ai-backup/
fi
if [ -d ".claude/framework-guides" ]; then
  cp -r .claude/framework-guides .ai-backup/
fi

# 2. Update to v1.8.0
# (follow standard update process)

# 3. Migrate customizations to new skill structure
# Old: .claude/language-guides/typescript.md (custom section)
# New: .claude/skills/typescript-guide/references/custom.md

# 4. Remove old directories
rm -rf .claude/language-guides
rm -rf .claude/framework-guides

# 5. Update any scripts that reference old paths
grep -r "language-guides\|framework-guides" .
```

### Migrating from Pre-1.6.0 Versions

**Major Changes**:
- Workflows system introduced
- .claude/ directory structure formalized
- Skills directory created

**Migration Steps**:
```bash
# 1. If you have old ad-hoc guides, migrate to skills
mkdir -p .claude/skills/

# 2. Follow standard update process
# 3. Integrate old content into new structure
```

---

## Validation and Testing

### Post-Update Test Suite

```bash
# 1. Version verification
grep "Current Version" CLAUDE.md || echo "ERROR: Version not found"

# 2. Structure verification
for dir in skills workflows; do
  test -d ".claude/$dir" || echo "ERROR: Missing .claude/$dir"
done

# 3. File count verification
SKILL_COUNT=$(ls -1 .claude/skills/ | wc -l)
echo "Skills installed: $SKILL_COUNT"
[ $SKILL_COUNT -gt 0 ] || echo "WARNING: No skills found"

# 4. Project file preservation
for file in project.md patterns.md state.md; do
  if [ -f ".claude/$file" ]; then
    echo "✓ Preserved: .claude/$file"
  fi
done

# 5. AI loading test
# Start AI session and verify:
# - AI acknowledges correct version
# - Skills are loadable
# - Workflows are accessible
```

### Regression Testing

```bash
# Test that existing functionality still works

# 1. Test skill loading
# Ask AI to load a language skill
# Verify it loads correctly

# 2. Test workflow execution
# Run a simple workflow (e.g., document-work)
# Verify it executes correctly

# 3. Test custom functionality
# If you have custom skills or workflows, test them

# 4. Test integration
# Verify AI can navigate between CLAUDE.md and skills
```

---

## Emergency Procedures

### Critical Update Failure

If update causes catastrophic failure:

```bash
# 1. STOP - Don't make it worse
# Document what happened

# 2. Restore from backup immediately
cp -r .ai-backup/* ./

# 3. Verify rollback successful
grep "Current Version" CLAUDE.md
ls .claude/

# 4. Test basic functionality
# Start AI session and verify it works

# 5. Report issue
# Create issue at: https://github.com/ar4mirez/samuel/issues
# Include: version numbers, error messages, what you were trying to do
```

### Data Loss Prevention

```bash
# Before ANY update, ensure you have:

# 1. Git commit of current state
git add -A
git commit -m "chore: pre-update checkpoint"

# 2. External backup of critical files
tar -czf ~/samuel-backup-$(date +%Y%m%d).tar.gz \
  CLAUDE.md CLAUDE.md CLAUDE.md \
  .claude/memory/ .claude/tasks/

# 3. Cloud backup (if using)
# Push to remote repository
git push origin main

# 4. Verify backups
tar -tzf ~/samuel-backup-$(date +%Y%m%d).tar.gz | head
```

---

## Best Practices Summary

1. **Always backup before updating**
2. **Read the changelog thoroughly**
3. **Test in a branch first**
4. **Document your customizations**
5. **Update incrementally for large version jumps**
6. **Verify after each major step**
7. **Keep team informed**
8. **Document the update in .claude/memory/**
9. **Clean up temporary files**
10. **Test AI functionality after update**
