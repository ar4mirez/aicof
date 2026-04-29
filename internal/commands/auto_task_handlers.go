package commands

import (
	"fmt"
	"os"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

func runAutoTaskList(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	prd, err := core.LoadAutoPRD(core.GetAutoPRDPath(cwd))
	if err != nil {
		return fmt.Errorf("no auto loop found. Run 'samuel auto init' first")
	}

	if JSONMode(cmd) {
		type taskJSON struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Status   string `json:"status"`
			ParentID string `json:"parentId,omitempty"`
			Priority string `json:"priority,omitempty"`
		}
		prd.RecalculateProgress()
		tasks := make([]taskJSON, 0, len(prd.Tasks))
		for _, t := range prd.Tasks {
			tasks = append(tasks, taskJSON{
				ID:       t.ID,
				Title:    t.Title,
				Status:   t.Status,
				ParentID: t.ParentID,
				Priority: t.Priority,
			})
		}
		PrintJSONForCmd(cmd, map[string]interface{}{
			"total":     prd.Progress.TotalTasks,
			"completed": prd.Progress.CompletedTasks,
			"tasks":     tasks,
		})
		return nil
	}

	ui.Header("Tasks")
	for _, t := range prd.Tasks {
		icon := taskStatusIcon(t.Status)
		indent := 0
		if t.ParentID != "" {
			indent = 1
		}
		ui.ListItem(indent, "%s %s %s", icon, t.ID, t.Title)
	}

	ui.Print("")
	prd.RecalculateProgress()
	ui.Print("Total: %d  Completed: %d  Pending: %d",
		prd.Progress.TotalTasks, prd.Progress.CompletedTasks,
		prd.Progress.TotalTasks-prd.Progress.CompletedTasks)
	return nil
}

func taskStatusIcon(status string) string {
	switch status {
	case core.TaskStatusCompleted:
		return "[x]"
	case core.TaskStatusSkipped:
		return "[-]"
	case core.TaskStatusBlocked:
		return "[!]"
	case core.TaskStatusInProgress:
		return "[>]"
	default:
		return "[ ]"
	}
}

func runAutoTaskComplete(cmd *cobra.Command, args []string) error {
	err := updateTaskStatus(args[0], func(prd *core.AutoPRD, id string) error {
		return prd.CompleteTask(id, "", 0)
	}, "completed")
	if err == nil && JSONMode(cmd) {
		runAutoTaskJSON(cmd, args[0], "complete")
	}
	return err
}

func runAutoTaskSkip(cmd *cobra.Command, args []string) error {
	err := updateTaskStatus(args[0], func(prd *core.AutoPRD, id string) error {
		return prd.SkipTask(id)
	}, "skipped")
	if err == nil && JSONMode(cmd) {
		runAutoTaskJSON(cmd, args[0], "skip")
	}
	return err
}

func runAutoTaskReset(cmd *cobra.Command, args []string) error {
	err := updateTaskStatus(args[0], func(prd *core.AutoPRD, id string) error {
		return prd.ResetTask(id)
	}, "reset to pending")
	if err == nil && JSONMode(cmd) {
		runAutoTaskJSON(cmd, args[0], "reset")
	}
	return err
}

func updateTaskStatus(id string, fn func(*core.AutoPRD, string) error, label string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	prdPath := core.GetAutoPRDPath(cwd)
	prd, err := core.LoadAutoPRD(prdPath)
	if err != nil {
		return fmt.Errorf("no auto loop found. Run 'samuel auto init' first")
	}

	if err := fn(prd, id); err != nil {
		return err
	}

	if err := prd.Save(prdPath); err != nil {
		return fmt.Errorf("failed to save prd.json: %w", err)
	}

	ui.Success("Task %s %s", id, label)
	return nil
}

// runAutoTaskJSON outputs a JSON result for task mutation commands.
// It is called from the individual task command handlers when --json is set.
// The command field reflects the invoked path (run done / run skip / run reset
// / auto task complete / auto task skip / auto task reset) — schema version 3.
func runAutoTaskJSON(cmd *cobra.Command, id, action string) {
	if JSONMode(cmd) {
		PrintJSONForCmd(cmd, map[string]interface{}{
			"taskId": id,
			"action": action,
		})
	}
}

func runAutoTaskAdd(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	prdPath := core.GetAutoPRDPath(cwd)
	prd, err := core.LoadAutoPRD(prdPath)
	if err != nil {
		return fmt.Errorf("no auto loop found. Run 'samuel auto init' first")
	}

	task := core.AutoTask{
		ID:       args[0],
		Title:    args[1],
		Status:   core.TaskStatusPending,
		Priority: core.TaskPriorityMedium,
	}

	if err := prd.AddTask(task); err != nil {
		return err
	}

	if err := prd.Save(prdPath); err != nil {
		return fmt.Errorf("failed to save prd.json: %w", err)
	}

	if JSONMode(cmd) {
		PrintJSONForCmd(cmd, map[string]interface{}{
			"taskId": task.ID,
			"title":  task.Title,
			"action": "add",
		})
		return nil
	}

	ui.Success("Task %s added: %s", task.ID, task.Title)
	return nil
}
