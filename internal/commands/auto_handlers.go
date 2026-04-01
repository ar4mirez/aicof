package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

func runAutoInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	if !core.ConfigExists(cwd) {
		return fmt.Errorf("no Samuel installation found. Run 'samuel init' first")
	}

	aiTool, _ := cmd.Flags().GetString("ai-tool")
	maxIter, _ := cmd.Flags().GetInt("max-iterations")
	prdPath, _ := cmd.Flags().GetString("prd")
	sandbox, _ := cmd.Flags().GetString("sandbox")
	sandboxImage, _ := cmd.Flags().GetString("sandbox-image")
	sandboxTemplate, _ := cmd.Flags().GetString("sandbox-template")

	if !core.IsValidAITool(aiTool) {
		return fmt.Errorf("unsupported AI tool: %s (supported: %v)", aiTool, core.GetSupportedAITools())
	}

	if !core.IsValidSandboxMode(sandbox) {
		return fmt.Errorf("unsupported sandbox mode: %s (supported: %v)", sandbox, core.GetSupportedSandboxModes())
	}

	return initAutoDir(cwd, prdPath, aiTool, maxIter, sandbox, sandboxImage, sandboxTemplate)
}

func initAutoDir(cwd, prdPath, aiTool string, maxIter int, sandbox, sandboxImage, sandboxTemplate string) error {
	autoDir := core.GetAutoDir(cwd)
	if err := os.MkdirAll(autoDir, 0755); err != nil {
		return fmt.Errorf("failed to create auto directory: %w", err)
	}

	config := core.AutoConfig{
		MaxIterations:   maxIter,
		QualityChecks:   detectQualityChecks(cwd),
		AITool:          aiTool,
		PromptFile:      filepath.Join(core.AutoDir, core.AutoPromptFile),
		Sandbox:         sandbox,
		SandboxImage:    sandboxImage,
		SandboxTemplate: sandboxTemplate,
	}

	if err := writeAutoFiles(autoDir, config); err != nil {
		return err
	}

	if prdPath != "" {
		if err := convertAndSavePRD(cwd, prdPath); err != nil {
			return err
		}
	} else {
		prd := core.NewAutoPRD("my-project", "Autonomous loop project")
		prd.Config = config
		prdFile := filepath.Join(autoDir, core.AutoPRDFile)
		if err := prd.Save(prdFile); err != nil {
			return fmt.Errorf("failed to save prd.json: %w", err)
		}
	}

	printInitSummary(autoDir, prdPath)
	return nil
}

func writeAutoFiles(autoDir string, config core.AutoConfig) error {
	promptContent := core.GeneratePromptFile(config)
	promptPath := filepath.Join(autoDir, core.AutoPromptFile)
	if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
		return fmt.Errorf("failed to write prompt.md: %w", err)
	}

	progressPath := filepath.Join(autoDir, core.AutoProgressFile)
	if _, err := os.Stat(progressPath); os.IsNotExist(err) {
		if err := os.WriteFile(progressPath, []byte(""), 0644); err != nil {
			return fmt.Errorf("failed to create progress.md: %w", err)
		}
	}

	return nil
}

func printInitSummary(autoDir, prdPath string) {
	ui.Success("Auto loop initialized at %s/", autoDir)
	ui.Print("")
	ui.Print("  Files created:")
	ui.Print("    %s", filepath.Join(core.AutoDir, core.AutoPRDFile))
	ui.Print("    %s", filepath.Join(core.AutoDir, core.AutoProgressFile))
	ui.Print("    %s", filepath.Join(core.AutoDir, core.AutoPromptFile))
	ui.Print("")

	if prdPath != "" {
		ui.Info("PRD converted from: %s", prdPath)
	} else {
		ui.Info("No PRD provided. Add tasks with 'samuel auto task add'")
		ui.Info("Or convert a PRD with 'samuel auto convert <prd-path>'")
	}

	ui.Print("")
	ui.Info("Next steps:")
	ui.Print("  1. Review %s", filepath.Join(core.AutoDir, core.AutoPRDFile))
	ui.Print("  2. Review %s", filepath.Join(core.AutoDir, core.AutoPromptFile))
	ui.Print("  3. Run 'samuel auto start' to begin the loop")
}

func detectQualityChecks(cwd string) []string {
	checks := []string{}
	if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
		return []string{"go test ./...", "go vet ./...", "go build ./..."}
	}
	if _, err := os.Stat(filepath.Join(cwd, "package.json")); err == nil {
		return []string{"npm test", "npm run lint", "npm run build"}
	}
	if _, err := os.Stat(filepath.Join(cwd, "Cargo.toml")); err == nil {
		return []string{"cargo test", "cargo clippy", "cargo build"}
	}
	if _, err := os.Stat(filepath.Join(cwd, "requirements.txt")); err == nil {
		return []string{"pytest", "ruff check ."}
	}
	return checks
}

func runAutoConvert(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	return convertAndSavePRD(cwd, args[0])
}

func convertAndSavePRD(cwd, prdPath string) error {
	tasksPath := core.FindTasksFile(prdPath)

	spinner := ui.NewSpinner("Converting PRD to prd.json")
	spinner.Start()

	prd, err := core.ConvertMarkdownToPRD(prdPath, tasksPath)
	if err != nil {
		spinner.Error("Conversion failed")
		return fmt.Errorf("failed to convert PRD: %w", err)
	}

	prdFile := core.GetAutoPRDPath(cwd)
	if err := prd.Save(prdFile); err != nil {
		spinner.Error("Save failed")
		return fmt.Errorf("failed to save prd.json: %w", err)
	}

	spinner.Success("Converted successfully")
	ui.Print("")
	ui.Print("  Project: %s", prd.Project.Name)
	ui.Print("  Tasks:   %d", prd.Progress.TotalTasks)
	if tasksPath != "" {
		ui.Print("  Source:  %s + %s", prdPath, tasksPath)
	} else {
		ui.Print("  Source:  %s (no task file found)", prdPath)
	}
	ui.Print("  Output:  %s", prdFile)
	return nil
}

func runAutoStatus(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	prdPath := core.GetAutoPRDPath(cwd)
	prd, err := core.LoadAutoPRD(prdPath)
	if err != nil {
		return fmt.Errorf("no auto loop found. Run 'samuel auto init' first")
	}

	prd.RecalculateProgress()

	if JSONMode(cmd) {
		counts := countTaskStatuses(prd)
		pct := 0
		if prd.Progress.TotalTasks > 0 {
			pct = (prd.Progress.CompletedTasks * 100) / prd.Progress.TotalTasks
		}
		var nextTask *map[string]string
		if t := prd.GetNextTask(); t != nil {
			nextTask = &map[string]string{"id": t.ID, "title": t.Title}
		}
		ui.PrintJSON("auto status", map[string]interface{}{
			"project":            prd.Project.Name,
			"status":             prd.Progress.Status,
			"pilotMode":          prd.Config.PilotMode,
			"aiTool":             prd.Config.AITool,
			"sandbox":            prd.Config.Sandbox,
			"maxIterations":      prd.Config.MaxIterations,
			"totalTasks":         prd.Progress.TotalTasks,
			"completedTasks":     prd.Progress.CompletedTasks,
			"progressPercent":    pct,
			"totalIterationsRun": prd.Progress.TotalIterationsRun,
			"lastIterationAt":    prd.Progress.LastIterationAt,
			"taskCounts":         counts,
			"nextTask":           nextTask,
		})
		return nil
	}

	printStatus(prd)
	return nil
}

func printStatus(prd *core.AutoPRD) {
	ui.Header("Auto Loop Status")

	ui.TableRow("Project", prd.Project.Name)
	if prd.Config.PilotMode {
		ui.TableRow("Mode", "pilot (autonomous discovery)")
	}
	ui.TableRow("Status", prd.Progress.Status)

	pct := 0
	if prd.Progress.TotalTasks > 0 {
		pct = (prd.Progress.CompletedTasks * 100) / prd.Progress.TotalTasks
	}
	ui.TableRow("Progress", fmt.Sprintf("%d/%d tasks (%d%%)",
		prd.Progress.CompletedTasks, prd.Progress.TotalTasks, pct))
	ui.TableRow("AI Tool", prd.Config.AITool)
	ui.TableRow("Sandbox", prd.Config.Sandbox)
	if prd.Config.Sandbox == core.SandboxDocker && prd.Config.SandboxImage != "" {
		ui.TableRow("Sandbox Image", prd.Config.SandboxImage)
	}
	if prd.Config.Sandbox == core.SandboxDockerSandbox && prd.Config.SandboxTemplate != "" {
		ui.TableRow("Sandbox Template", prd.Config.SandboxTemplate)
	}
	ui.TableRow("Max Iterations", fmt.Sprintf("%d", prd.Config.MaxIterations))

	if prd.Progress.TotalIterationsRun > 0 {
		ui.TableRow("Iterations Run", fmt.Sprintf("%d", prd.Progress.TotalIterationsRun))
	}
	if prd.Progress.LastIterationAt != "" {
		ui.TableRow("Last Iteration", prd.Progress.LastIterationAt)
	}

	printPilotStatus(prd)

	// Count by status
	counts := countTaskStatuses(prd)
	ui.Print("")
	ui.Print("  Pending: %d  Completed: %d  Blocked: %d  Skipped: %d",
		counts["pending"], counts["completed"], counts["blocked"], counts["skipped"])

	next := prd.GetNextTask()
	if next != nil {
		ui.Print("")
		ui.Info("Next task: %s %s", next.ID, next.Title)
	}
}

func printPilotStatus(prd *core.AutoPRD) {
	if !prd.Config.PilotMode || prd.Config.PilotConfig == nil {
		return
	}

	pilot := prd.Config.PilotConfig
	ui.TableRow("Discover Interval", fmt.Sprintf("every %d iterations", pilot.DiscoverInterval))
	ui.TableRow("Max Tasks/Discovery", fmt.Sprintf("%d", pilot.MaxDiscoveryTasks))
	if pilot.Focus != "" {
		ui.TableRow("Focus", pilot.Focus)
	}
	if prd.Progress.DiscoveryIterations > 0 {
		ui.TableRow("Discovery Iterations", fmt.Sprintf("%d", prd.Progress.DiscoveryIterations))
	}
	if prd.Progress.ImplIterations > 0 {
		ui.TableRow("Impl Iterations", fmt.Sprintf("%d", prd.Progress.ImplIterations))
	}
}

func countTaskStatuses(prd *core.AutoPRD) map[string]int {
	counts := map[string]int{
		"pending": 0, "in_progress": 0, "completed": 0, "skipped": 0, "blocked": 0,
	}
	for _, t := range prd.Tasks {
		counts[t.Status]++
	}
	return counts
}

func validateSandbox(sandbox string) error {
	if sandbox == core.SandboxDocker {
		if err := core.CheckDockerAvailable(); err != nil {
			return fmt.Errorf("docker sandbox unavailable: %w", err)
		}
	}
	if sandbox == core.SandboxDockerSandbox {
		if err := core.CheckDockerSandboxAvailable(); err != nil {
			return fmt.Errorf("docker sandbox unavailable: %w", err)
		}
	}
	return nil
}
