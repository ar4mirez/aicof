package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ar4mirez/samuel/internal/core"
)

func TestTaskStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{core.TaskStatusCompleted, "[x]"},
		{core.TaskStatusSkipped, "[-]"},
		{core.TaskStatusBlocked, "[!]"},
		{core.TaskStatusInProgress, "[>]"},
		{core.TaskStatusPending, "[ ]"},
		{"unknown", "[ ]"},
		{"", "[ ]"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := taskStatusIcon(tt.status)
			if got != tt.want {
				t.Errorf("taskStatusIcon(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

// setupTestPRD creates a temporary directory with a valid prd.json and returns
// the directory path and prd.json path. The caller can customize the PRD by
// modifying the returned AutoPRD before calling Save.
func setupTestPRD(t *testing.T, tasks []core.AutoTask) (string, string) {
	t.Helper()
	dir := t.TempDir()
	autoDir := filepath.Join(dir, ".claude", "auto")
	if err := os.MkdirAll(autoDir, 0755); err != nil {
		t.Fatalf("failed to create auto dir: %v", err)
	}

	prd := core.NewAutoPRD("test", "test project")
	prd.Tasks = tasks

	prdPath := filepath.Join(autoDir, "prd.json")
	if err := prd.Save(prdPath); err != nil {
		t.Fatalf("failed to save test prd.json: %v", err)
	}

	return dir, prdPath
}

func TestUpdateTaskStatus_Complete(t *testing.T) {
	dir, prdPath := setupTestPRD(t, []core.AutoTask{
		{ID: "1", Title: "First task", Status: core.TaskStatusPending},
		{ID: "2", Title: "Second task", Status: core.TaskStatusPending},
	})

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = updateTaskStatus("1", func(prd *core.AutoPRD, id string) error {
		return prd.CompleteTask(id, "abc123", 1)
	}, "completed")
	if err != nil {
		t.Fatalf("updateTaskStatus returned error: %v", err)
	}

	// Verify the task was saved
	prd, err := core.LoadAutoPRD(prdPath)
	if err != nil {
		t.Fatalf("failed to reload prd.json: %v", err)
	}

	for _, task := range prd.Tasks {
		if task.ID == "1" {
			if task.Status != core.TaskStatusCompleted {
				t.Errorf("task 1 status = %q, want %q", task.Status, core.TaskStatusCompleted)
			}
			if task.CommitSHA != "abc123" {
				t.Errorf("task 1 commit_sha = %q, want %q", task.CommitSHA, "abc123")
			}
			return
		}
	}
	t.Error("task 1 not found after save")
}

func TestUpdateTaskStatus_Skip(t *testing.T) {
	dir, prdPath := setupTestPRD(t, []core.AutoTask{
		{ID: "1", Title: "Task to skip", Status: core.TaskStatusPending},
	})

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = updateTaskStatus("1", func(prd *core.AutoPRD, id string) error {
		return prd.SkipTask(id)
	}, "skipped")
	if err != nil {
		t.Fatalf("updateTaskStatus returned error: %v", err)
	}

	prd, err := core.LoadAutoPRD(prdPath)
	if err != nil {
		t.Fatalf("failed to reload prd.json: %v", err)
	}

	if prd.Tasks[0].Status != core.TaskStatusSkipped {
		t.Errorf("task status = %q, want %q", prd.Tasks[0].Status, core.TaskStatusSkipped)
	}
}

func TestUpdateTaskStatus_Reset(t *testing.T) {
	dir, prdPath := setupTestPRD(t, []core.AutoTask{
		{ID: "1", Title: "Completed task", Status: core.TaskStatusCompleted, CommitSHA: "old123"},
	})

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = updateTaskStatus("1", func(prd *core.AutoPRD, id string) error {
		return prd.ResetTask(id)
	}, "reset to pending")
	if err != nil {
		t.Fatalf("updateTaskStatus returned error: %v", err)
	}

	prd, err := core.LoadAutoPRD(prdPath)
	if err != nil {
		t.Fatalf("failed to reload prd.json: %v", err)
	}

	task := prd.Tasks[0]
	if task.Status != core.TaskStatusPending {
		t.Errorf("task status = %q, want %q", task.Status, core.TaskStatusPending)
	}
	if task.CommitSHA != "" {
		t.Errorf("task commit_sha = %q, want empty after reset", task.CommitSHA)
	}
}

func TestUpdateTaskStatus_TaskNotFound(t *testing.T) {
	dir, _ := setupTestPRD(t, []core.AutoTask{
		{ID: "1", Title: "Only task", Status: core.TaskStatusPending},
	})

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = updateTaskStatus("nonexistent", func(prd *core.AutoPRD, id string) error {
		return prd.CompleteTask(id, "", 0)
	}, "completed")
	if err == nil {
		t.Fatal("expected error for nonexistent task, got nil")
	}
}

func TestUpdateTaskStatus_NoPRD(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = updateTaskStatus("1", func(prd *core.AutoPRD, id string) error {
		return prd.CompleteTask(id, "", 0)
	}, "completed")
	if err == nil {
		t.Fatal("expected error when prd.json missing, got nil")
	}
}

func TestRunAutoTaskAdd(t *testing.T) {
	dir, prdPath := setupTestPRD(t, []core.AutoTask{
		{ID: "1", Title: "Existing task", Status: core.TaskStatusPending},
	})

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = runAutoTaskAdd(nil, []string{"2", "New task title"})
	if err != nil {
		t.Fatalf("runAutoTaskAdd returned error: %v", err)
	}

	prd, err := core.LoadAutoPRD(prdPath)
	if err != nil {
		t.Fatalf("failed to reload prd.json: %v", err)
	}

	if len(prd.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(prd.Tasks))
	}

	newTask := prd.Tasks[1]
	if newTask.ID != "2" {
		t.Errorf("new task ID = %q, want %q", newTask.ID, "2")
	}
	if newTask.Title != "New task title" {
		t.Errorf("new task title = %q, want %q", newTask.Title, "New task title")
	}
	if newTask.Status != core.TaskStatusPending {
		t.Errorf("new task status = %q, want %q", newTask.Status, core.TaskStatusPending)
	}
	if newTask.Priority != core.TaskPriorityMedium {
		t.Errorf("new task priority = %q, want %q", newTask.Priority, core.TaskPriorityMedium)
	}
}

func TestRunAutoTaskAdd_DuplicateID(t *testing.T) {
	dir, _ := setupTestPRD(t, []core.AutoTask{
		{ID: "1", Title: "Existing task", Status: core.TaskStatusPending},
	})

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = runAutoTaskAdd(nil, []string{"1", "Duplicate ID task"})
	if err == nil {
		t.Fatal("expected error for duplicate task ID, got nil")
	}
}

func TestRunAutoTaskAdd_NoPRD(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = runAutoTaskAdd(nil, []string{"1", "Task without PRD"})
	if err == nil {
		t.Fatal("expected error when prd.json missing, got nil")
	}
}

func TestRunAutoTaskList(t *testing.T) {
	tasks := []core.AutoTask{
		{ID: "1", Title: "Completed task", Status: core.TaskStatusCompleted},
		{ID: "2", Title: "Pending task", Status: core.TaskStatusPending},
		{ID: "2.1", Title: "Subtask", Status: core.TaskStatusPending, ParentID: "2"},
		{ID: "3", Title: "In progress", Status: core.TaskStatusInProgress},
	}
	dir, _ := setupTestPRD(t, tasks)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	// runAutoTaskList prints to stdout — verify it doesn't error
	err = runAutoTaskList(nil, nil)
	if err != nil {
		t.Fatalf("runAutoTaskList returned error: %v", err)
	}
}

func TestRunAutoTaskList_NoPRD(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = runAutoTaskList(nil, nil)
	if err == nil {
		t.Fatal("expected error when prd.json missing, got nil")
	}
}

func TestRunAutoTaskComplete(t *testing.T) {
	dir, prdPath := setupTestPRD(t, []core.AutoTask{
		{ID: "1", Title: "Pending task", Status: core.TaskStatusPending},
	})

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = runAutoTaskComplete(nil, []string{"1"})
	if err != nil {
		t.Fatalf("runAutoTaskComplete returned error: %v", err)
	}

	prd, err := core.LoadAutoPRD(prdPath)
	if err != nil {
		t.Fatalf("failed to reload prd.json: %v", err)
	}

	if prd.Tasks[0].Status != core.TaskStatusCompleted {
		t.Errorf("task status = %q, want %q", prd.Tasks[0].Status, core.TaskStatusCompleted)
	}
}

func TestRunAutoTaskSkip(t *testing.T) {
	dir, prdPath := setupTestPRD(t, []core.AutoTask{
		{ID: "1", Title: "Task to skip", Status: core.TaskStatusPending},
	})

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = runAutoTaskSkip(nil, []string{"1"})
	if err != nil {
		t.Fatalf("runAutoTaskSkip returned error: %v", err)
	}

	prd, err := core.LoadAutoPRD(prdPath)
	if err != nil {
		t.Fatalf("failed to reload prd.json: %v", err)
	}

	if prd.Tasks[0].Status != core.TaskStatusSkipped {
		t.Errorf("task status = %q, want %q", prd.Tasks[0].Status, core.TaskStatusSkipped)
	}
}

func TestRunAutoTaskReset(t *testing.T) {
	dir, prdPath := setupTestPRD(t, []core.AutoTask{
		{ID: "1", Title: "Done task", Status: core.TaskStatusCompleted, CommitSHA: "sha1"},
	})

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = runAutoTaskReset(nil, []string{"1"})
	if err != nil {
		t.Fatalf("runAutoTaskReset returned error: %v", err)
	}

	prd, err := core.LoadAutoPRD(prdPath)
	if err != nil {
		t.Fatalf("failed to reload prd.json: %v", err)
	}

	task := prd.Tasks[0]
	if task.Status != core.TaskStatusPending {
		t.Errorf("task status = %q, want %q", task.Status, core.TaskStatusPending)
	}
	if task.CommitSHA != "" {
		t.Errorf("task commit_sha = %q, want empty after reset", task.CommitSHA)
	}
}

func TestUpdateTaskStatus_CorruptPRD(t *testing.T) {
	dir := t.TempDir()
	autoDir := filepath.Join(dir, ".claude", "auto")
	if err := os.MkdirAll(autoDir, 0755); err != nil {
		t.Fatalf("failed to create auto dir: %v", err)
	}

	// Write invalid JSON
	prdPath := filepath.Join(autoDir, "prd.json")
	if err := os.WriteFile(prdPath, []byte("{invalid json"), 0644); err != nil {
		t.Fatalf("failed to write corrupt prd.json: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = updateTaskStatus("1", func(prd *core.AutoPRD, id string) error {
		return prd.CompleteTask(id, "", 0)
	}, "completed")
	if err == nil {
		t.Fatal("expected error for corrupt prd.json, got nil")
	}
}

func TestUpdateTaskStatus_ProgressRecalculated(t *testing.T) {
	dir, prdPath := setupTestPRD(t, []core.AutoTask{
		{ID: "1", Title: "Task one", Status: core.TaskStatusPending},
		{ID: "2", Title: "Task two", Status: core.TaskStatusPending},
		{ID: "3", Title: "Task three", Status: core.TaskStatusCompleted},
	})

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	// Complete task 1
	err = updateTaskStatus("1", func(prd *core.AutoPRD, id string) error {
		return prd.CompleteTask(id, "sha456", 2)
	}, "completed")
	if err != nil {
		t.Fatalf("updateTaskStatus returned error: %v", err)
	}

	// Save calls RecalculateProgress — verify counts
	prd, err := core.LoadAutoPRD(prdPath)
	if err != nil {
		t.Fatalf("failed to reload prd.json: %v", err)
	}

	if prd.Progress.TotalTasks != 3 {
		t.Errorf("total_tasks = %d, want 3", prd.Progress.TotalTasks)
	}
	if prd.Progress.CompletedTasks != 2 {
		t.Errorf("completed_tasks = %d, want 2", prd.Progress.CompletedTasks)
	}
}

func TestRunAutoTaskAdd_SavedCorrectly(t *testing.T) {
	dir, prdPath := setupTestPRD(t, []core.AutoTask{})

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = runAutoTaskAdd(nil, []string{"100", "Brand new task"})
	if err != nil {
		t.Fatalf("runAutoTaskAdd returned error: %v", err)
	}

	// Verify by reading raw JSON to ensure proper serialization
	data, err := os.ReadFile(prdPath)
	if err != nil {
		t.Fatalf("failed to read prd.json: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to parse prd.json: %v", err)
	}

	var tasks []core.AutoTask
	if err := json.Unmarshal(raw["tasks"], &tasks); err != nil {
		t.Fatalf("failed to parse tasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "100" || tasks[0].Title != "Brand new task" {
		t.Errorf("task = {ID: %q, Title: %q}, want {ID: %q, Title: %q}",
			tasks[0].ID, tasks[0].Title, "100", "Brand new task")
	}
}
