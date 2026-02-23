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

## Steps

1. **Read project context**:
   - Read ` + "`CLAUDE.md`" + ` or ` + "`AGENTS.md`" + ` for project guardrails and conventions
   - Read ` + "`.claude/auto/progress-context.md`" + ` for learnings from prior iterations (auto-generated summary)
   - Read ` + "`README.md`" + ` for project overview
   - Scan the project directory structure

2. **Analyze the codebase** for improvement opportunities:
   - Check test coverage gaps (files/packages with low or no tests)
   - Find TODOs, FIXMEs, and HACKs in the code
   - Look for code quality issues (long functions, high complexity, dead code)
   - Check documentation gaps (missing godocs, outdated README)
   - Identify security concerns (input validation, error handling)
   - Review recent git log for incomplete work or follow-up needs

3. **Read existing tasks**:
   - Read ` + "`.claude/auto/prd.json`" + ` to see current tasks
   - Do NOT create duplicate tasks — check titles and descriptions carefully
   - Skip areas that already have pending or in-progress tasks

4. **Generate new tasks**:
   - Add tasks to prd.json with status "pending"
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

5. **Document findings**:
   - Append a summary of what you discovered to ` + "`.claude/auto/progress.md`" + `
   - Format: ` + "`[timestamp] [discovery] FOUND: description`" + `

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
