package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

// runCmd is the v3.0.0 primary verb for the autonomous Ralph Wiggum loop.
// `samuel auto` is preserved as a permanent top-level alias (not Hidden,
// not deprecated) so muscle memory and existing community references keep
// working forever.
var runCmd = &cobra.Command{
	Use:     "run",
	Aliases: []string{"auto"},
	Short:   "Autonomous AI coding loop (Ralph Wiggum methodology)",
	Long: `Manage autonomous AI coding loops using the Ralph Wiggum methodology.

The run command enables unattended AI-driven development: an AI agent
independently selects, implements, and commits tasks from a structured
task list (prd.json), running across multiple fresh context windows.

Bare 'samuel run' shows status when an autonomous loop is initialized in
the current directory; otherwise it prints actionable help and exits non-zero.
It never silently starts a loop.

Subcommands:
  init       Initialize the autonomous loop for a project
  start      Begin or resume the loop (requires init)
  pilot      Zero-setup discover-and-implement loop
  status     Show progress and current state
  tasks      List all tasks (recommended; replaces 'task list')
  done       Mark a task complete (replaces 'task complete')
  skip       Mark a task skipped
  reset      Reset a task to pending
  enqueue    Add a task with auto-assigned id
  task       Preserved nested namespace (task add for CI; list/complete/skip/reset Hidden)
  convert    Convert markdown PRD/tasks to prd.json

Workflow:
  1. samuel run init --prd .claude/tasks/0001-prd-feature.md
  2. Review .claude/auto/prd.json and prompt.md
  3. samuel run start
  4. samuel run status   (or just 'samuel run' when in the project)

Examples:
  samuel run init --prd .claude/tasks/0001-prd-auth.md
  samuel run start --iterations 20
  samuel run                          # Smart: status if loop exists
  samuel run tasks
  samuel run done 1.1
  samuel run enqueue "Add metrics dashboard"
  samuel run task add 5.0 "Explicit-id task for CI"

Backward compatibility: 'samuel auto' is a permanent alias for 'samuel run'.
The legacy 'samuel auto task complete/skip/reset/list' subcommands also work
through v3.x but print a one-line deprecation pointing at the flat verbs.`,
	Args: cobra.NoArgs,
	RunE: runRunBare,
}

// runRunBare implements smart bare-invocation behavior:
//   - PRD exists  → show status (read-only)
//   - PRD missing → print actionable help and exit non-zero
//
// This intentionally never starts a loop without explicit user action,
// preventing the v2-era footgun where a stray `samuel auto` could kick off
// pilot mode in a directory the user wasn't expecting.
func runRunBare(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	prdPath := core.GetAutoPRDPath(cwd)
	if _, statErr := os.Stat(prdPath); statErr == nil {
		// Loop is initialized — show status.
		return runAutoStatus(cmd, args)
	}

	// No PRD — actionable help, non-zero exit.
	fmt.Fprintln(os.Stderr, "samuel: no autonomous loop initialized in this directory.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  Initialize one:    samuel run init")
	fmt.Fprintln(os.Stderr, "  From a PRD:        samuel run init --prd .claude/tasks/0001-prd-feature.md")
	fmt.Fprintln(os.Stderr, "  Zero-setup mode:   samuel run pilot")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "See 'samuel run --help' for the full subcommand list.")
	return errors.New("no auto loop initialized")
}

var autoInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize autonomous loop for a project",
	Long: `Initialize the autonomous loop directory structure and configuration.

Creates .claude/auto/ with:
  - prd.json      Machine-readable task state
  - progress.md   Append-only learnings journal
  - prompt.md     Iteration prompt template

If --prd is provided, converts the PRD and associated task file to prd.json.

Examples:
  samuel run init
  samuel run init --prd .claude/tasks/0001-prd-auth.md
  samuel run init --ai-tool amp --max-iterations 100`,
	RunE: runAutoInit,
}

var autoConvertCmd = &cobra.Command{
	Use:   "convert <prd-path>",
	Short: "Convert markdown PRD/tasks to prd.json",
	Long: `Convert a markdown PRD and optional task list into prd.json format.

Automatically looks for a matching tasks file using the convention:
  PRD: .claude/tasks/0001-prd-feature.md
  Tasks: .claude/tasks/tasks-0001-prd-feature.md

Examples:
  samuel run convert .claude/tasks/0001-prd-auth.md`,
	Args: cobra.ExactArgs(1),
	RunE: runAutoConvert,
}

var autoStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show autonomous loop status",
	Long: `Display the current state of the autonomous loop including
task progress, iteration count, and recent activity.

Equivalent to bare 'samuel run' when a loop is initialized.

Examples:
  samuel run status`,
	RunE: runAutoStatus,
}

var autoStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Begin or resume the autonomous loop",
	Long: `Start the autonomous AI coding loop.

The loop runs natively in Go, invoking the configured AI tool on each
iteration until all tasks are completed or the max iteration count is reached.

Examples:
  samuel run start
  samuel run start --iterations 20
  samuel run start --dry-run
  samuel run start --yes`,
	RunE: runAutoStart,
}

// --- Flat task verbs (v3 primary forms) ---

var autoTasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "List all tasks with status",
	Long: `List every task in the autonomous loop with current status.

Replaces 'samuel auto task list' (still works as a Hidden alias).

Examples:
  samuel run tasks
  samuel run tasks --json`,
	RunE: runAutoTaskList,
}

var autoDoneCmd = &cobra.Command{
	Use:   "done <task-id>",
	Short: "Mark a task complete (replaces 'task complete')",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutoTaskComplete,
}

var autoSkipCmd = &cobra.Command{
	Use:   "skip <task-id>",
	Short: "Mark a task skipped (replaces 'task skip')",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutoTaskSkip,
}

var autoResetCmd = &cobra.Command{
	Use:   "reset <task-id>",
	Short: "Reset a task to pending (replaces 'task reset')",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutoTaskReset,
}

var autoEnqueueCmd = &cobra.Command{
	Use:   "enqueue <title>",
	Short: "Add a task with auto-assigned id",
	Long: `Add a new top-level task to the autonomous loop. The task ID is
automatically assigned as the next available top-level integer (e.g., the
next "5" in a list of "1", "2.0", "3").

For explicit IDs (CI scripts, structured task creation), use the preserved
'samuel run task add <id> <title>' form.

Examples:
  samuel run enqueue "Add input validation"
  samuel run enqueue "Investigate flaky test"`,
	Args: cobra.ExactArgs(1),
	RunE: runAutoEnqueue,
}

// --- Preserved nested 'task' namespace ---

// autoTaskCmd preserves the v2 'samuel auto task' nesting. v3 hides 'list',
// 'complete', 'skip', 'reset' under it (forwards to flat verbs) but keeps
// 'add <id> <title>' as the explicit-id path for CI scripts that depend on
// deterministic IDs.
var autoTaskCmd = &cobra.Command{
	Use:   "task",
	Short: "Preserved nested task namespace (use flat verbs instead)",
	Long: `Manual task management. The 'task' namespace is preserved for backward
compatibility; prefer the flat verbs in v3:

  samuel run tasks                # was: samuel auto task list
  samuel run done <id>            # was: samuel auto task complete <id>
  samuel run skip <id>            # was: samuel auto task skip <id>
  samuel run reset <id>           # was: samuel auto task reset <id>
  samuel run enqueue <title>      # auto-assigned id (recommended)
  samuel run task add <id> <title># explicit id (CI/scripts, preserved)`,
}

// autoTaskAddCmd remains visible — preserved for CI scripts that need
// deterministic task IDs. The flat 'enqueue' verb is the recommended path
// for human users.
var autoTaskAddCmd = &cobra.Command{
	Use:   "add <task-id> <title>",
	Short: "Add a task with explicit id (CI/scripts; humans use 'enqueue')",
	Args:  cobra.ExactArgs(2),
	RunE:  runAutoTaskAdd,
}

// Hidden+Deprecated wrappers under 'task' for the four verbs that have flat
// equivalents. They retain the original handlers and just add a one-line
// stderr redirect that points at the flat verb.

var autoTaskListCmd = &cobra.Command{
	Use:    "list",
	Short:  "[DEPRECATED] Use 'samuel run tasks'",
	Hidden: true,
	RunE:   redirectAndRun("samuel run tasks", runAutoTaskList),
}

var autoTaskCompleteCmd = &cobra.Command{
	Use:    "complete <task-id>",
	Short:  "[DEPRECATED] Use 'samuel run done'",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE:   redirectAndRun("samuel run done", runAutoTaskComplete),
}

var autoTaskSkipCmd = &cobra.Command{
	Use:    "skip <task-id>",
	Short:  "[DEPRECATED] Use 'samuel run skip'",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE:   redirectAndRun("samuel run skip", runAutoTaskSkip),
}

var autoTaskResetCmd = &cobra.Command{
	Use:    "reset <task-id>",
	Short:  "[DEPRECATED] Use 'samuel run reset'",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE:   redirectAndRun("samuel run reset", runAutoTaskReset),
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Lifecycle subcommands
	runCmd.AddCommand(autoInitCmd)
	runCmd.AddCommand(autoConvertCmd)
	runCmd.AddCommand(autoStatusCmd)
	runCmd.AddCommand(autoStartCmd)
	registerPilotCmd()

	// Flat task verbs (v3 primary)
	runCmd.AddCommand(autoTasksCmd)
	runCmd.AddCommand(autoDoneCmd)
	runCmd.AddCommand(autoSkipCmd)
	runCmd.AddCommand(autoResetCmd)
	runCmd.AddCommand(autoEnqueueCmd)

	// Nested task namespace (preserved for explicit-id CI use)
	runCmd.AddCommand(autoTaskCmd)
	autoTaskCmd.AddCommand(autoTaskAddCmd) // visible
	autoTaskCmd.AddCommand(autoTaskListCmd)
	autoTaskCmd.AddCommand(autoTaskCompleteCmd)
	autoTaskCmd.AddCommand(autoTaskSkipCmd)
	autoTaskCmd.AddCommand(autoTaskResetCmd)

	// init flags
	autoInitCmd.Flags().String("prd", "", "Path to PRD markdown file to convert")
	autoInitCmd.Flags().String("ai-tool", "claude", "AI tool to use (claude, amp, cursor, codex)")
	autoInitCmd.Flags().Int("max-iterations", 50, "Maximum loop iterations")
	autoInitCmd.Flags().String("sandbox", "none", "Sandbox mode (none, docker, docker-sandbox)")
	autoInitCmd.Flags().String("sandbox-image", "", "Docker image for docker mode (default: node:lts)")
	autoInitCmd.Flags().String("sandbox-template", "", "Docker sandbox template (e.g., python:3-alpine)")

	// start flags
	autoStartCmd.Flags().Int("iterations", 0, "Override max iterations for this run")
	autoStartCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	autoStartCmd.Flags().Bool("dry-run", false, "Show what would happen without executing")
	autoStartCmd.Flags().String("sandbox", "", "Override sandbox mode for this run (none, docker, docker-sandbox)")
	autoStartCmd.Flags().String("sandbox-image", "", "Override Docker image for docker mode")
	autoStartCmd.Flags().String("sandbox-template", "", "Override Docker sandbox template for this run")
}

// runAutoEnqueue adds a task with an auto-assigned ID. It's the human-friendly
// counterpart to 'samuel run task add <id> <title>': you don't think about IDs,
// the loop just gets a new top-level task at the next available integer.
func runAutoEnqueue(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	prdPath := core.GetAutoPRDPath(cwd)
	prd, err := core.LoadAutoPRD(prdPath)
	if err != nil {
		return fmt.Errorf("no auto loop found. Run 'samuel run init' first")
	}

	id := prd.NextAvailableID()
	title := args[0]

	task := core.AutoTask{
		ID:       id,
		Title:    title,
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
			"taskId": id,
			"title":  title,
			"action": "enqueue",
		})
		return nil
	}

	ui.Success("Task %s enqueued: %s", id, title)
	return nil
}

// autoCmd is exported as an alias of runCmd for backward compatibility with
// any internal code that still references the old name. Functionally identical.
var autoCmd = runCmd
