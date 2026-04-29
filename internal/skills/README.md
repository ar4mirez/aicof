# Agent Skills

Skills are capability modules that give AI agents new abilities. They follow the [Agent Skills](https://agentskills.io) open standard, supported by 25+ agent products including Claude Code, Cursor, GitHub Copilot, VS Code, and OpenAI Codex.

## What are Skills?

Skills are folders of instructions, scripts, and resources that agents can discover and use to perform tasks more accurately and efficiently. Unlike workflows (which guide *how* to approach tasks), skills define *what capabilities* an agent has.

## Directory Structure

Each skill is a directory containing:

```
skill-name/
├── SKILL.md          # Required: Metadata + instructions
├── scripts/          # Optional: Executable code
├── references/       # Optional: Additional documentation
└── assets/           # Optional: Templates, data files
```

### SKILL.md Format

Every skill must have a `SKILL.md` file with YAML frontmatter:

```yaml
---
name: skill-name
description: |
  What this skill does and when to use it.
  Include specific triggers and keywords.
license: MIT
metadata:
  author: your-name
  version: "1.0"
---

# Skill Name

## Purpose
What capability this skill provides.

## When to Use
- Scenario 1
- Scenario 2

## Instructions
1. Step one
2. Step two
3. Step three

## Examples
...
```

## Creating a Skill

Use the Samuel CLI:

```bash
samuel skill create my-skill-name
```

Or use the create-skill skill: `.claude/skills/create-skill/SKILL.md`

## Managing Skills

```bash
# Create a new skill
samuel skill create <name>

# Validate skills
samuel skill validate
samuel skill validate <name>

# List installed skills
samuel skill list

# Get skill details
samuel skill info <name>
```

## Best Practices

1. **Keep skills focused** - One capability per skill
2. **Write clear descriptions** - Help agents know when to use it
3. **Include examples** - Show input/output pairs
4. **Keep SKILL.md under 500 lines** - Split into reference files if needed
5. **Test with real tasks** - Verify the skill works as expected

## Specification

For the complete Agent Skills specification, see:
- https://agentskills.io/specification
- https://github.com/agentskills/agentskills

## Language Guide Skills

Language guides are implemented as skills with `metadata.category: language`. They follow the naming convention `{lang}-guide` (e.g., `go-guide`, `python-guide`).

Language guide skills use progressive disclosure:
- **SKILL.md** (~200-300 lines): Core guardrails, conventions, and essential patterns
- **references/**: Detailed patterns, common pitfalls, and security examples

These skills are automatically loaded when working with files matching the language's extensions.

## Skills vs Workflows

| Aspect | Skills | Workflows |
|--------|--------|-----------|
| Purpose | Add capabilities | Guide processes |
| Focus | "What AI can do" | "How to approach tasks" |
| Portability | Cross-tool (25+ products) | Samuel-specific |
| Structure | SKILL.md + resources | Markdown steps |
| Example | Language guides, commit messages | Code review, PRD creation |
