package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Task context file constant
const AutoTaskContextFile = "task-context.md"

// GenerateTaskContext writes a compact task view to .claude/auto/task-context.md.
// For discovery iterations, it includes a one-line summary of each task plus
// files already covered. For implementation iterations, it includes only the
// next task's full detail plus recent completions.
func GenerateTaskContext(projectDir string, prd *AutoPRD, isDiscovery bool) error {
	contextPath := filepath.Join(projectDir, AutoDir, AutoTaskContextFile)

	if prd == nil {
		return writeEmptyTaskContext(contextPath)
	}

	if isDiscovery {
		return writeDiscoveryTaskContext(contextPath, prd)
	}
	return writeImplementationTaskContext(contextPath, prd)
}

func writeEmptyTaskContext(path string) error {
	content := "# Task Context (auto-generated — do not edit)\n\n" +
		"No tasks defined yet.\n"
	return os.WriteFile(path, []byte(content), 0644)
}

func writeDiscoveryTaskContext(path string, prd *AutoPRD) error {
	var sb strings.Builder
	sb.WriteString("# Task Context — Discovery Mode (auto-generated — do not edit)\n\n")

	writeTaskSummaryTable(&sb, prd)
	writeCoveredFiles(&sb, prd)

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func writeImplementationTaskContext(path string, prd *AutoPRD) error {
	var sb strings.Builder
	sb.WriteString("# Task Context — Implementation Mode (auto-generated — do not edit)\n\n")

	next := prd.GetNextTask()
	if next == nil {
		sb.WriteString("No pending tasks available.\n")
		return os.WriteFile(path, []byte(sb.String()), 0644)
	}

	writeNextTaskDetail(&sb, next)
	writeRecentCompletions(&sb, prd)

	sb.WriteString("\nNote: Update task status directly in prd.json using the task ID above.\n")
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func writeTaskSummaryTable(sb *strings.Builder, prd *AutoPRD) {
	pending, completed, other := 0, 0, 0
	for _, t := range prd.Tasks {
		switch t.Status {
		case TaskStatusPending:
			pending++
		case TaskStatusCompleted:
			completed++
		default:
			other++
		}
	}
	fmt.Fprintf(sb, "## Summary: %d total | %d pending | %d completed | %d other\n\n",
		len(prd.Tasks), pending, completed, other)

	if pending > 0 {
		sb.WriteString("### Pending Tasks\n\n")
		for _, t := range prd.Tasks {
			if t.Status == TaskStatusPending {
				fmt.Fprintf(sb, "- [%s] %s (priority: %s)\n", t.ID, t.Title, t.Priority)
			}
		}
		sb.WriteString("\n")
	}

	// Only show completed titles (no descriptions) to save tokens
	if completed > 0 {
		sb.WriteString("### Completed Tasks (titles only)\n\n")
		for _, t := range prd.Tasks {
			if t.Status == TaskStatusCompleted {
				fmt.Fprintf(sb, "- [%s] %s\n", t.ID, t.Title)
			}
		}
		sb.WriteString("\n")
	}
}

func writeCoveredFiles(sb *strings.Builder, prd *AutoPRD) {
	covered := make(map[string]bool)
	for _, t := range prd.Tasks {
		if t.Status == TaskStatusPending || t.Status == TaskStatusInProgress {
			for _, f := range t.FilesToModify {
				covered[f] = true
			}
			for _, f := range t.FilesToCreate {
				covered[f] = true
			}
		}
	}

	if len(covered) == 0 {
		return
	}

	sb.WriteString("### Files Already Covered by Pending Tasks\n\n")
	sb.WriteString("Do NOT create tasks for these files — they already have pending work:\n\n")
	for f := range covered {
		fmt.Fprintf(sb, "- %s\n", f)
	}
	sb.WriteString("\n")
}

func writeNextTaskDetail(sb *strings.Builder, task *AutoTask) {
	sb.WriteString("## Your Task\n\n")
	fmt.Fprintf(sb, "- **ID**: %s\n", task.ID)
	fmt.Fprintf(sb, "- **Title**: %s\n", task.Title)
	fmt.Fprintf(sb, "- **Priority**: %s\n", task.Priority)
	fmt.Fprintf(sb, "- **Complexity**: %s\n", task.Complexity)

	if task.Description != "" {
		fmt.Fprintf(sb, "\n### Description\n\n%s\n", task.Description)
	}

	if len(task.FilesToModify) > 0 {
		sb.WriteString("\n### Files to Modify\n\n")
		for _, f := range task.FilesToModify {
			fmt.Fprintf(sb, "- %s\n", f)
		}
	}

	if len(task.FilesToCreate) > 0 {
		sb.WriteString("\n### Files to Create\n\n")
		for _, f := range task.FilesToCreate {
			fmt.Fprintf(sb, "- %s\n", f)
		}
	}

	if len(task.DependsOn) > 0 {
		sb.WriteString("\n### Dependencies\n\n")
		for _, d := range task.DependsOn {
			fmt.Fprintf(sb, "- Task %s\n", d)
		}
	}
	sb.WriteString("\n")
}

func writeRecentCompletions(sb *strings.Builder, prd *AutoPRD) {
	const maxRecent = 5

	var recent []AutoTask
	for i := len(prd.Tasks) - 1; i >= 0 && len(recent) < maxRecent; i-- {
		if prd.Tasks[i].Status == TaskStatusCompleted {
			recent = append(recent, prd.Tasks[i])
		}
	}

	if len(recent) == 0 {
		return
	}

	sb.WriteString("## Recent Completions (for context)\n\n")
	for _, t := range recent {
		fmt.Fprintf(sb, "- [%s] %s", t.ID, t.Title)
		if t.CommitSHA != "" {
			fmt.Fprintf(sb, " (commit: %s)", t.CommitSHA)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}
