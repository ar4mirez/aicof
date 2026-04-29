package core

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// NextAvailableID returns the next free top-level task ID for auto-id task
// creation (used by `samuel run enqueue`). Top-level IDs are integers; the
// returned value is one greater than the largest existing top-level integer
// prefix. Tasks with non-integer or nested IDs ("1.1", "fix-x") do not move
// the counter.
//
// Examples:
//   prd has tasks "1", "2.0", "2.1", "3"          -> returns "4"
//   prd has tasks "1.0", "1.1"                    -> returns "2"
//   prd is empty                                   -> returns "1"
//   prd has tasks "fix-auth", "1"                  -> returns "2"
func (p *AutoPRD) NextAvailableID() string {
	max := 0
	for _, t := range p.Tasks {
		// Strip any nested suffix; we only care about the top-level integer.
		head := t.ID
		if idx := strings.Index(head, "."); idx >= 0 {
			head = head[:idx]
		}
		n, err := strconv.Atoi(head)
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return strconv.Itoa(max + 1)
}

// priorityRank returns a numeric rank for sorting (lower = higher priority)
func priorityRank(priority string) int {
	switch priority {
	case TaskPriorityCritical:
		return 0
	case TaskPriorityHigh:
		return 1
	case TaskPriorityMedium:
		return 2
	case TaskPriorityLow:
		return 3
	default:
		return 2
	}
}

// GetNextTask returns the highest-priority available pending task
func (p *AutoPRD) GetNextTask() *AutoTask {
	available := p.getAvailableTasks()
	if len(available) == 0 {
		return nil
	}

	sort.Slice(available, func(i, j int) bool {
		pi := priorityRank(available[i].Priority)
		pj := priorityRank(available[j].Priority)
		if pi != pj {
			return pi < pj
		}
		return available[i].ID < available[j].ID
	})

	return available[0]
}

// getAvailableTasks returns pending tasks whose dependencies are all completed
func (p *AutoPRD) getAvailableTasks() []*AutoTask {
	completed := make(map[string]bool)
	for i := range p.Tasks {
		if p.Tasks[i].Status == TaskStatusCompleted || p.Tasks[i].Status == TaskStatusSkipped {
			completed[p.Tasks[i].ID] = true
		}
	}

	var available []*AutoTask
	for i := range p.Tasks {
		if p.Tasks[i].Status != TaskStatusPending {
			continue
		}
		if allDependenciesMet(p.Tasks[i].DependsOn, completed) {
			available = append(available, &p.Tasks[i])
		}
	}
	return available
}

func allDependenciesMet(deps []string, completed map[string]bool) bool {
	for _, dep := range deps {
		if !completed[dep] {
			return false
		}
	}
	return true
}

// findTask returns a pointer to the task with the given ID
func (p *AutoPRD) findTask(id string) *AutoTask {
	for i := range p.Tasks {
		if p.Tasks[i].ID == id {
			return &p.Tasks[i]
		}
	}
	return nil
}

// CompleteTask marks a task as completed with commit info
func (p *AutoPRD) CompleteTask(id, commitSHA string, iteration int) error {
	task := p.findTask(id)
	if task == nil {
		return fmt.Errorf("task not found: %s", id)
	}

	task.Status = TaskStatusCompleted
	task.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	task.CommitSHA = commitSHA
	task.Iteration = iteration
	return nil
}

// SkipTask marks a task as skipped
func (p *AutoPRD) SkipTask(id string) error {
	task := p.findTask(id)
	if task == nil {
		return fmt.Errorf("task not found: %s", id)
	}
	task.Status = TaskStatusSkipped
	return nil
}

// ResetTask resets a task to pending
func (p *AutoPRD) ResetTask(id string) error {
	task := p.findTask(id)
	if task == nil {
		return fmt.Errorf("task not found: %s", id)
	}

	task.Status = TaskStatusPending
	task.CompletedAt = ""
	task.CommitSHA = ""
	task.Iteration = 0
	return nil
}

// AddTask appends a new task to the task list
func (p *AutoPRD) AddTask(task AutoTask) error {
	if task.ID == "" {
		return fmt.Errorf("task ID is required")
	}
	if existing := p.findTask(task.ID); existing != nil {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}
	if task.Status == "" {
		task.Status = TaskStatusPending
	}
	p.Tasks = append(p.Tasks, task)
	return nil
}

// RecalculateProgress updates the progress summary from task data
func (p *AutoPRD) RecalculateProgress() {
	total := 0
	completed := 0
	for _, t := range p.Tasks {
		total++
		if t.Status == TaskStatusCompleted {
			completed++
		}
	}
	p.Progress.TotalTasks = total
	p.Progress.CompletedTasks = completed

	if total > 0 && completed == total {
		p.Progress.Status = LoopStatusCompleted
	}
}

// ValidateAutoPRD checks prd.json for structural issues
func ValidateAutoPRD(prd *AutoPRD) []string {
	var errors []string

	if prd.Version == "" {
		errors = append(errors, "version is required")
	}
	if prd.Project.Name == "" {
		errors = append(errors, "project.name is required")
	}

	errors = append(errors, validateTasks(prd.Tasks)...)
	return errors
}

// validateTasks checks task-level constraints
func validateTasks(tasks []AutoTask) []string {
	var errors []string
	ids := make(map[string]bool)

	for _, t := range tasks {
		if t.ID == "" {
			errors = append(errors, "task missing ID")
			continue
		}
		if ids[t.ID] {
			errors = append(errors, fmt.Sprintf("duplicate task ID: %s", t.ID))
		}
		ids[t.ID] = true

		if t.Title == "" {
			errors = append(errors, fmt.Sprintf("task %s missing title", t.ID))
		}
		if !isValidStatus(t.Status) {
			errors = append(errors, fmt.Sprintf("task %s has invalid status: %s", t.ID, t.Status))
		}
	}

	// Validate dependency references
	for _, t := range tasks {
		for _, dep := range t.DependsOn {
			if !ids[dep] {
				errors = append(errors, fmt.Sprintf("task %s depends on unknown task: %s", t.ID, dep))
			}
		}
	}

	return errors
}

func isValidStatus(status string) bool {
	switch status {
	case TaskStatusPending, TaskStatusInProgress, TaskStatusCompleted,
		TaskStatusSkipped, TaskStatusBlocked:
		return true
	default:
		return false
	}
}
