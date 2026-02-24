package core

import (
	"fmt"
	"strings"
)

// GetDiscoveryPromptTemplate returns the raw discovery prompt template.
// This prompt instructs the AI to analyze the project and generate tasks
// into prd.json without making any code changes.
func GetDiscoveryPromptTemplate() string {
	return `# Discovery Iteration Prompt

You are running in DISCOVERY mode as part of the autonomous pilot loop.
Your job is to analyze the project and generate high-value tasks.

**CRITICAL: Do NOT write any code or make any commits in this iteration.**
**Only update prd.json and progress.md.**

## Context Efficiency

Your context window is limited and expensive. Follow these rules strictly:

1. **Read ` + "`.claude/auto/project-snapshot.md`" + ` FIRST** — it contains pre-computed
   directory structure, test gap analysis, file sizes, TODO counts, and recent git log.
   Do NOT manually scan the directory tree, run find/ls, or grep for TODOs.
2. **Read ` + "`.claude/auto/task-context.md`" + `** for existing task summary — do NOT read
   the full prd.json for analysis. Only open prd.json when writing new tasks.
3. **Read ` + "`.claude/auto/progress-context.md`" + `** for learnings and explored areas
   from prior iterations.
4. **Use grep/glob over file reads** — search for patterns across files instead of
   reading entire files. Only read a file when you need specific line-level context.
5. **Read at most 10 source files** per discovery iteration. Use the snapshot and
   grep results to decide which files are highest-value.
6. **Skip files already covered by pending tasks** — task-context.md lists them.
7. **Skip files in the "Areas Already Analyzed" section** of progress-context.md
   unless the snapshot shows they have changed recently (check git log).

## Steps

1. **Read pre-computed context** (these are small, read all three):
   - ` + "`.claude/auto/project-snapshot.md`" + ` — project structure, test gaps, TODOs, large files
   - ` + "`.claude/auto/task-context.md`" + ` — existing task summary, covered files
   - ` + "`.claude/auto/progress-context.md`" + ` — learnings, explored areas

2. **Analyze using the snapshot** (do NOT read all source files):
   - Review test gaps from project-snapshot.md → generate test tasks for uncovered files
   - Review TODO/FIXME counts from snapshot → read only files with high counts
   - Review large files from snapshot → check if they exceed guardrail limits (50-line functions, 300-line files)
   - Use ` + "`grep`" + ` for security patterns (e.g., unchecked errors, path traversal, missing validation)
   - Check recent git log in snapshot for follow-up needs
   - Read ` + "`CLAUDE.md`" + ` or ` + "`AGENTS.md`" + ` only if you need to verify project-specific guardrails

3. **Read existing tasks** (from task-context.md, NOT full prd.json):
   - Do NOT create duplicate tasks — check titles in task-context.md
   - Skip files listed under "Files Already Covered by Pending Tasks"

4. **Generate new tasks**:
   - Read prd.json, add new tasks with status "pending", write prd.json back
   - Each task must be atomic (affects <=5 files)
   - Use clear, actionable titles
   - Set appropriate priority and complexity
   - Set the "source" field to "pilot-discovery"
   - Each task in the ` + "`tasks`" + ` array MUST follow this exact structure (all IDs are strings):

` + "```" + `json
{
  "id": "1",
  "title": "Clear actionable title",
  "description": "What needs to be done and why",
  "status": "pending",
  "priority": "high",
  "complexity": "medium",
  "files_to_modify": ["path/to/file.go"],
  "source": "pilot-discovery"
}
` + "```" + `

   **IMPORTANT**: The ` + "`id`" + ` field MUST be a string (e.g., ` + "`\"1\"`" + `, ` + "`\"2\"`" + `, ` + "`\"1.1\"`" + `), never a number.
   Use sequential string IDs starting after the highest existing task ID.

5. **Document findings and exploration**:
   - Append discovered issues to ` + "`.claude/auto/progress.md`" + `
   - Format: ` + "`[timestamp] [discovery] FOUND: description`" + `
   - For each source file you read, log: ` + "`[timestamp] [discovery] EXPLORED: path/to/file.go`" + `
     (this helps future discoveries skip already-analyzed files)

## Priority Order

When generating tasks, prioritize in this order:
1. **Security issues** (critical priority)
2. **Failing or missing tests** (high priority)
3. **Code quality violations** (medium-high priority)
4. **Documentation gaps** (medium priority)
5. **Performance improvements** (medium-low priority)
6. **Refactoring opportunities** (low priority)

## Rules

- Generate ONLY atomic tasks (each task affects <=5 files)
- Do NOT make any code changes — only update prd.json and progress.md
- Do NOT create duplicate tasks
- Do NOT commit any changes
- Keep task descriptions specific and actionable
- Include files_to_modify in each task when possible
- **Read at most 10 source files** — use grep and the snapshot for everything else
`
}

// GenerateDiscoveryPrompt creates a customized discovery prompt.
func GenerateDiscoveryPrompt(config AutoConfig, pilot *PilotConfig) string {
	var sb strings.Builder
	sb.WriteString(GetDiscoveryPromptTemplate())

	if pilot != nil {
		sb.WriteString("\n## Discovery Configuration\n\n")
		fmt.Fprintf(&sb, "- **Max new tasks to generate**: %d\n", pilot.MaxDiscoveryTasks)

		if pilot.Focus != "" {
			sb.WriteString(generateFocusSection(pilot.Focus))
		}
	}

	if len(config.QualityChecks) > 0 {
		sb.WriteString("\n## Quality Checks Reference\n\n")
		sb.WriteString("These are the project's quality check commands:\n\n")
		sb.WriteString("```bash\n")
		for _, check := range config.QualityChecks {
			sb.WriteString(check + "\n")
		}
		sb.WriteString("```\n")
	}

	return sb.String()
}

func generateFocusSection(focus string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n### Focus Area: %s\n\n", focus)
	sb.WriteString("Prioritize tasks related to this focus area. ")

	switch strings.ToLower(focus) {
	case "testing":
		sb.WriteString("Focus on test coverage gaps, missing edge case tests, " +
			"flaky tests, and test infrastructure improvements.\n")
	case "docs", "documentation":
		sb.WriteString("Focus on missing documentation, outdated README, " +
			"missing godocs, and API documentation.\n")
	case "security":
		sb.WriteString("Focus on input validation, authentication, authorization, " +
			"dependency vulnerabilities, and OWASP top 10.\n")
	case "performance":
		sb.WriteString("Focus on hot paths, unnecessary allocations, " +
			"N+1 queries, caching opportunities, and benchmarks.\n")
	case "refactoring":
		sb.WriteString("Focus on code duplication, long functions, high complexity, " +
			"dead code, and architectural improvements.\n")
	default:
		fmt.Fprintf(&sb, "Look for improvements related to: %s\n", focus)
	}

	return sb.String()
}
