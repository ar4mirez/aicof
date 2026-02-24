package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

var autoPilotCmd = &cobra.Command{
	Use:   "pilot",
	Short: "Fully autonomous discover-and-implement loop",
	Long: `Run a fully autonomous loop that requires zero setup.

Pilot mode automatically analyzes the project, discovers the highest-value
work, generates tasks, and implements them continuously until reaching the
iteration limit.

The loop alternates between discovery iterations (analyzing the project
for improvement opportunities) and implementation iterations (executing
tasks from prd.json).

Examples:
  samuel auto pilot
  samuel auto pilot --iterations 20 --focus testing
  samuel auto pilot --dry-run
  samuel auto pilot --discover-interval 3 --max-tasks 5
  samuel auto pilot --sandbox docker --yes`,
	RunE: runAutoPilot,
}

func registerPilotCmd() {
	autoCmd.AddCommand(autoPilotCmd)

	autoPilotCmd.Flags().Int("iterations", core.DefaultPilotIterations,
		"Max total iterations (discovery + implementation)")
	autoPilotCmd.Flags().Int("discover-interval", core.DefaultDiscoverInterval,
		"Re-discover every N iterations")
	autoPilotCmd.Flags().Int("max-tasks", core.DefaultMaxDiscoveryTasks,
		"Max tasks to generate per discovery")
	autoPilotCmd.Flags().String("focus", "",
		"Focus area: testing, docs, security, performance, refactoring")
	autoPilotCmd.Flags().String("ai-tool", "claude",
		"AI tool (claude, amp, codex)")
	autoPilotCmd.Flags().String("sandbox", "none",
		"Sandbox mode: none, docker, docker-sandbox")
	autoPilotCmd.Flags().String("sandbox-image", "",
		"Docker image for docker mode")
	autoPilotCmd.Flags().String("sandbox-template", "",
		"Docker sandbox template")
	autoPilotCmd.Flags().Bool("dry-run", false,
		"Preview without executing")
	autoPilotCmd.Flags().BoolP("yes", "y", false,
		"Skip confirmation prompt")
}

func runAutoPilot(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	if !core.ConfigExists(cwd) {
		return fmt.Errorf("no Samuel installation found. Run 'samuel init' first")
	}

	pilotCfg, err := parsePilotFlags(cmd)
	if err != nil {
		return err
	}

	autoCfg, err := parseAutoFlags(cmd, cwd)
	if err != nil {
		return err
	}

	if err := validateSandbox(autoCfg.Sandbox); err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		return printPilotDryRun(autoCfg, pilotCfg, cwd)
	}

	skipConfirm, _ := cmd.Flags().GetBool("yes")
	if !skipConfirm {
		confirmed, confirmErr := ui.Confirm("Start pilot mode? This will analyze and modify your project.", false)
		if confirmErr != nil || !confirmed {
			ui.Info("Cancelled")
			return nil
		}
	}

	return executePilotLoop(cwd, autoCfg, pilotCfg)
}

func parsePilotFlags(cmd *cobra.Command) (*core.PilotConfig, error) {
	cfg := core.NewPilotConfig()

	if v, _ := cmd.Flags().GetInt("discover-interval"); v > 0 {
		cfg.DiscoverInterval = v
	}
	if v, _ := cmd.Flags().GetInt("max-tasks"); v > 0 {
		cfg.MaxDiscoveryTasks = v
	}
	if v, _ := cmd.Flags().GetString("focus"); v != "" {
		cfg.Focus = v
	}

	return cfg, nil
}

func parseAutoFlags(cmd *cobra.Command, cwd string) (core.AutoConfig, error) {
	aiTool, _ := cmd.Flags().GetString("ai-tool")
	if !core.IsValidAITool(aiTool) {
		return core.AutoConfig{}, fmt.Errorf(
			"unsupported AI tool: %s (supported: %v)", aiTool, core.GetSupportedAITools())
	}

	sandbox, _ := cmd.Flags().GetString("sandbox")
	if !core.IsValidSandboxMode(sandbox) {
		return core.AutoConfig{}, fmt.Errorf(
			"unsupported sandbox mode: %s (supported: %v)", sandbox, core.GetSupportedSandboxModes())
	}

	maxIter, _ := cmd.Flags().GetInt("iterations")
	sandboxImage, _ := cmd.Flags().GetString("sandbox-image")
	sandboxTpl, _ := cmd.Flags().GetString("sandbox-template")

	return core.AutoConfig{
		MaxIterations:   maxIter,
		QualityChecks:   detectQualityChecks(cwd),
		AITool:          aiTool,
		Sandbox:         sandbox,
		SandboxImage:    sandboxImage,
		SandboxTemplate: sandboxTpl,
		PilotMode:       true,
	}, nil
}

func executePilotLoop(cwd string, autoCfg core.AutoConfig, pilotCfg *core.PilotConfig) error {
	prd, err := initPilotMode(cwd, autoCfg, pilotCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize pilot mode: %w", err)
	}

	prdPath := core.GetAutoPRDPath(cwd)
	autoDir := core.GetAutoDir(cwd)

	implPromptPath := filepath.Join(autoDir, core.AutoPromptFile)
	discoveryPromptPath := filepath.Join(autoDir, core.AutoDiscoveryPromptFile)

	loopCfg := core.NewLoopConfig(cwd, prd)
	loopCfg.MaxIterations = autoCfg.MaxIterations

	lastDiscoveryIter := 0
	emptyDiscoveries := 0
	consecutiveFailures := 0

	stats := pilotStats{}

	ui.Info("Pilot mode starting...")
	ui.Print("  AI Tool:     %s", autoCfg.AITool)
	ui.Print("  Iterations:  %d", autoCfg.MaxIterations)
	ui.Print("  Discover:    every %d iterations", pilotCfg.DiscoverInterval)
	if pilotCfg.Focus != "" {
		ui.Print("  Focus:       %s", pilotCfg.Focus)
	}
	ui.Print("")

	for i := 1; i <= autoCfg.MaxIterations; i++ {
		currentPRD, loadErr := core.LoadAutoPRD(prdPath)
		if loadErr != nil {
			return fmt.Errorf("iteration %d: failed to reload prd.json: %w", i, loadErr)
		}

		isDiscovery := core.ShouldRunDiscovery(
			currentPRD, i, lastDiscoveryIter, pilotCfg.DiscoverInterval)

		core.PrepareProgressContext(cwd)
		_ = core.GenerateProjectSnapshot(cwd)
		_ = core.GenerateTaskContext(cwd, currentPRD, isDiscovery)

		if isDiscovery {
			ui.Info("[iteration:%d] DISCOVERY - analyzing project for tasks...", i)
			loopCfg.PromptPath = discoveryPromptPath
			lastDiscoveryIter = i
			stats.discoveryCount++

			tasksBefore := len(currentPRD.Tasks)
			if err := runSingleIteration(loopCfg, &consecutiveFailures); err != nil {
				return err
			}

			reloaded, reloadErr := core.LoadAutoPRD(prdPath)
			if reloadErr != nil {
				ui.Warn("[iteration:%d] Failed to reload prd.json after discovery: %v", i, reloadErr)
			}
			if reloaded != nil && len(reloaded.Tasks) <= tasksBefore {
				emptyDiscoveries++
				ui.Warn("[iteration:%d] Discovery found no new tasks (%d/%d empty)",
					i, emptyDiscoveries, core.MaxEmptyDiscoveries)
			} else {
				emptyDiscoveries = 0
				if reloaded != nil {
					newTasks := len(reloaded.Tasks) - tasksBefore
					ui.Success("[iteration:%d] Discovery added %d new tasks", i, newTasks)
				}
			}
		} else {
			if currentPRD.GetNextTask() == nil {
				ui.Success("All tasks completed and no more to discover!")
				break
			}

			next := currentPRD.GetNextTask()
			ui.Info("[iteration:%d] IMPLEMENTING - %s: %s", i, next.ID, next.Title)
			loopCfg.PromptPath = implPromptPath
			stats.implCount++

			if err := runSingleIteration(loopCfg, &consecutiveFailures); err != nil {
				return err
			}
		}

		if emptyDiscoveries >= core.MaxEmptyDiscoveries {
			reloaded, reloadErr := core.LoadAutoPRD(prdPath)
			if reloadErr != nil {
				ui.Warn("Failed to reload prd.json for empty discovery check: %v", reloadErr)
			}
			if reloaded == nil || core.CountPendingTasks(reloaded) == 0 {
				ui.Info("No new tasks after %d discoveries. Stopping.", emptyDiscoveries)
				break
			}
		}

		if i < autoCfg.MaxIterations {
			time.Sleep(time.Duration(loopCfg.PauseSecs) * time.Second)
		}
	}

	printPilotSummary(prdPath, stats)
	return nil
}

func initPilotMode(cwd string, autoCfg core.AutoConfig, pilotCfg *core.PilotConfig) (*core.AutoPRD, error) {
	autoDir := core.GetAutoDir(cwd)
	if err := os.MkdirAll(autoDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create auto directory: %w", err)
	}

	prd := core.InitPilotPRD(cwd, autoCfg, pilotCfg)

	prdPath := core.GetAutoPRDPath(cwd)
	if err := prd.Save(prdPath); err != nil {
		return nil, fmt.Errorf("failed to save prd.json: %w", err)
	}

	implPrompt := core.GeneratePromptFile(prd.Config)
	implPath := filepath.Join(autoDir, core.AutoPromptFile)
	if err := os.WriteFile(implPath, []byte(implPrompt), 0644); err != nil {
		return nil, fmt.Errorf("failed to write prompt.md: %w", err)
	}

	discoveryPrompt := core.GenerateDiscoveryPrompt(prd.Config, pilotCfg)
	discoveryPath := filepath.Join(autoDir, core.AutoDiscoveryPromptFile)
	if err := os.WriteFile(discoveryPath, []byte(discoveryPrompt), 0644); err != nil {
		return nil, fmt.Errorf("failed to write discovery-prompt.md: %w", err)
	}

	progressPath := filepath.Join(autoDir, core.AutoProgressFile)
	if _, err := os.Stat(progressPath); os.IsNotExist(err) {
		if err := os.WriteFile(progressPath, []byte(""), 0644); err != nil {
			return nil, fmt.Errorf("failed to create progress.md: %w", err)
		}
	}

	return prd, nil
}

func runSingleIteration(cfg core.LoopConfig, consecutiveFailures *int) error {
	if err := core.InvokeAgent(cfg); err != nil {
		*consecutiveFailures++
		ui.Warn("Agent error (%d consecutive): %v", *consecutiveFailures, err)
		if *consecutiveFailures >= cfg.MaxConsecFails {
			return fmt.Errorf(
				"%d consecutive failures — aborting. Check AI tool auth/config",
				cfg.MaxConsecFails)
		}
		return nil
	}
	*consecutiveFailures = 0
	return nil
}

