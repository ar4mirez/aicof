package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateTaskContext_NilPRD(t *testing.T) {
	dir := t.TempDir()
	autoDir := filepath.Join(dir, AutoDir)
	if err := os.MkdirAll(autoDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	err := GenerateTaskContext(dir, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, AutoDir, AutoTaskContextFile))
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	if !strings.Contains(string(data), "No tasks defined yet") {
		t.Error("expected empty context message")
	}
}

func TestGenerateTaskContext_DiscoveryMode(t *testing.T) {
	dir := t.TempDir()
	autoDir := filepath.Join(dir, AutoDir)
	if err := os.MkdirAll(autoDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	prd := &AutoPRD{
		Tasks: []AutoTask{
			{ID: "1", Title: "Fix bug", Status: TaskStatusCompleted, Priority: TaskPriorityHigh},
			{ID: "2", Title: "Add tests", Status: TaskStatusPending, Priority: TaskPriorityMedium,
				FilesToModify: []string{"internal/core/foo.go"}},
			{ID: "3", Title: "Refactor bar", Status: TaskStatusPending, Priority: TaskPriorityLow,
				FilesToModify: []string{"internal/core/bar.go"}},
		},
	}

	err := GenerateTaskContext(dir, prd, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, AutoDir, AutoTaskContextFile))
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "Discovery Mode") {
		t.Error("expected discovery mode header")
	}
	if !strings.Contains(content, "2 pending") {
		t.Error("expected pending count")
	}
	if !strings.Contains(content, "1 completed") {
		t.Error("expected completed count")
	}
	if !strings.Contains(content, "[1] Fix bug") {
		t.Error("expected completed task title")
	}
	if !strings.Contains(content, "[2] Add tests") {
		t.Error("expected pending task title")
	}
	if !strings.Contains(content, "internal/core/foo.go") {
		t.Error("expected covered file")
	}
	if !strings.Contains(content, "internal/core/bar.go") {
		t.Error("expected covered file")
	}
}

func TestGenerateTaskContext_ImplementationMode(t *testing.T) {
	dir := t.TempDir()
	autoDir := filepath.Join(dir, AutoDir)
	if err := os.MkdirAll(autoDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	prd := &AutoPRD{
		Tasks: []AutoTask{
			{ID: "1", Title: "Done task", Status: TaskStatusCompleted,
				Priority: TaskPriorityHigh, CommitSHA: "abc123"},
			{ID: "2", Title: "Next task", Status: TaskStatusPending,
				Priority: TaskPriorityHigh, Complexity: "medium",
				Description:   "Implement this feature",
				FilesToModify: []string{"internal/core/foo.go", "internal/core/bar.go"}},
			{ID: "3", Title: "Later task", Status: TaskStatusPending,
				Priority: TaskPriorityLow},
		},
	}

	err := GenerateTaskContext(dir, prd, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, AutoDir, AutoTaskContextFile))
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "Implementation Mode") {
		t.Error("expected implementation mode header")
	}
	if !strings.Contains(content, "**ID**: 2") {
		t.Error("expected next task ID")
	}
	if !strings.Contains(content, "Next task") {
		t.Error("expected next task title")
	}
	if !strings.Contains(content, "Implement this feature") {
		t.Error("expected task description")
	}
	if !strings.Contains(content, "internal/core/foo.go") {
		t.Error("expected files to modify")
	}
	// Should include recent completions
	if !strings.Contains(content, "Done task") {
		t.Error("expected recent completion")
	}
	if !strings.Contains(content, "abc123") {
		t.Error("expected commit SHA in completions")
	}
	// Should NOT include the later task details
	if strings.Contains(content, "Later task") {
		t.Error("should not include non-next pending tasks in impl mode")
	}
}

func TestGenerateTaskContext_ImplementationMode_NoTasks(t *testing.T) {
	dir := t.TempDir()
	autoDir := filepath.Join(dir, AutoDir)
	if err := os.MkdirAll(autoDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	prd := &AutoPRD{
		Tasks: []AutoTask{
			{ID: "1", Title: "Done", Status: TaskStatusCompleted},
		},
	}

	err := GenerateTaskContext(dir, prd, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, AutoDir, AutoTaskContextFile))
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	if !strings.Contains(string(data), "No pending tasks available") {
		t.Error("expected no tasks message")
	}
}

func TestWriteTaskSummaryTable_EmptyPRD(t *testing.T) {
	prd := &AutoPRD{Tasks: []AutoTask{}}

	var sb strings.Builder
	writeTaskSummaryTable(&sb, prd)

	content := sb.String()
	if !strings.Contains(content, "0 total") {
		t.Error("expected zero total")
	}
}

func TestWriteCoveredFiles_NoPending(t *testing.T) {
	prd := &AutoPRD{
		Tasks: []AutoTask{
			{ID: "1", Title: "Done", Status: TaskStatusCompleted,
				FilesToModify: []string{"foo.go"}},
		},
	}

	var sb strings.Builder
	writeCoveredFiles(&sb, prd)

	// Completed tasks should not appear in covered files
	if strings.Contains(sb.String(), "foo.go") {
		t.Error("completed task files should not be in covered list")
	}
}

func TestWriteRecentCompletions_LimitedTo5(t *testing.T) {
	tasks := make([]AutoTask, 10)
	for i := range tasks {
		tasks[i] = AutoTask{
			ID:     fmt.Sprintf("%d", i+1),
			Title:  fmt.Sprintf("Task %d", i+1),
			Status: TaskStatusCompleted,
		}
	}

	prd := &AutoPRD{Tasks: tasks}

	var sb strings.Builder
	writeRecentCompletions(&sb, prd)

	content := sb.String()
	// Should contain last 5 (tasks 6-10)
	if !strings.Contains(content, "Task 10") {
		t.Error("expected Task 10")
	}
	if !strings.Contains(content, "Task 6") {
		t.Error("expected Task 6")
	}
	// Should not contain tasks 1-5
	if strings.Contains(content, "[1]") {
		t.Error("should not contain old task 1")
	}
}
