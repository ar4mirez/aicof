package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/orchestrator"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check Samuel installation health",
	Long: `Verify the Samuel framework installation is complete and healthy.

Checks performed:
- Config file exists and is valid
- CLAUDE.md is present
- All installed components exist
- No broken file references
- Directory structure is correct

Examples:
  samuel doctor           # Run health check
  samuel doctor --fix     # Auto-fix issues where possible`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().Bool("fix", false, "Auto-fix issues where possible")
	doctorCmd.Flags().Bool("verify", false, "Run deeper bundle verification (gstack SHA, gbrain MCP registration, skill resolution)")
	doctorCmd.Flags().Bool("no-orchestrator", false, "Skip the v4 orchestrator health checks; show only legacy v3 doctor checks")
}

type checkResult struct {
	name    string
	passed  bool
	message string
	fixable bool
}

func runDoctor(cmd *cobra.Command, args []string) error {
	autoFix, _ := cmd.Flags().GetBool("fix")
	noOrchestrator, _ := cmd.Flags().GetBool("no-orchestrator")
	verify, _ := cmd.Flags().GetBool("verify")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	var results []checkResult

	configResult, config := checkConfigFile()
	results = append(results, configResult)
	results = append(results, checkCLAUDEMD(cwd))
	results = append(results, checkAGENTSMD(cwd))

	dirResult, missingDirs := checkDirectoryStructure(cwd)
	results = append(results, dirResult)

	if config != nil {
		results = append(results, checkInstalledComponents(cwd, config)...)
	}

	results = append(results, checkSkillsIntegrity(cwd)...)

	autoDir := core.GetAutoDir(cwd)
	if _, err := os.Stat(autoDir); err == nil {
		results = append(results, checkAutoHealth(cwd)...)
	}

	if config != nil {
		results = append(results, checkLocalModifications(cwd, config)...)
	}

	// v4 orchestrator health: walk gstack, gbrain, samuel-skills components
	// and merge their HealthStatus into the existing checkResult stream so
	// users see one unified health page.
	var orchStatuses []orchestrator.HealthStatus
	if !noOrchestrator {
		orchStatuses = runOrchestratorDoctor(cwd, verify)
		for _, s := range orchStatuses {
			results = append(results, healthStatusToCheckResult(s))
		}
	}

	if JSONMode(cmd) {
		type checkJSON struct {
			Name    string `json:"name"`
			Passed  bool   `json:"passed"`
			Message string `json:"message"`
			Fixable bool   `json:"fixable,omitempty"`
		}
		checks := make([]checkJSON, 0, len(results))
		passed, failed := 0, 0
		for _, r := range results {
			checks = append(checks, checkJSON{
				Name:    r.name,
				Passed:  r.passed,
				Message: r.message,
				Fixable: r.fixable,
			})
			if r.passed {
				passed++
			} else {
				failed++
			}
		}
		healthy := failed == 0
		ui.PrintJSON("doctor", map[string]interface{}{
			"healthy": healthy,
			"passed":  passed,
			"failed":  failed,
			"checks":  checks,
		})
		return nil
	}

	ui.Header("Samuel Health Check")

	passedCount, failedCount, fixableCount := printCheckResults(results)
	printCheckSummary(passedCount, failedCount, fixableCount, autoFix)

	if autoFix && fixableCount > 0 {
		performAutoFix(cwd, config, missingDirs)
	}

	return nil
}

// printCheckResults displays each check result and returns pass/fail/fixable counts.
func printCheckResults(results []checkResult) (int, int, int) {
	passedCount := 0
	failedCount := 0
	fixableCount := 0

	for _, r := range results {
		if r.passed {
			ui.SuccessItem(0, "%s: %s", r.name, r.message)
			passedCount++
		} else {
			ui.ErrorItem(0, "%s: %s", r.name, r.message)
			failedCount++
			if r.fixable {
				fixableCount++
			}
		}
	}

	return passedCount, failedCount, fixableCount
}

// printCheckSummary displays the overall health status summary.
func printCheckSummary(passedCount, failedCount, fixableCount int, autoFix bool) {
	fmt.Println()
	if failedCount == 0 {
		ui.Bold("Status: Healthy")
		ui.Success("All %d checks passed", passedCount)
	} else {
		ui.Bold("Status: Issues Found")
		ui.Error("%d checks failed, %d passed", failedCount, passedCount)

		if fixableCount > 0 && !autoFix {
			ui.Info("\n%d issues can be auto-fixed. Run 'samuel doctor --fix' to repair.", fixableCount)
		}
	}
}

// performAutoFix attempts to repair fixable issues by re-downloading missing files.
func performAutoFix(cwd string, config *core.Config, missingDirs []string) {
	fmt.Println()
	ui.Info("Attempting to fix issues...")

	if config == nil {
		return
	}

	for _, dir := range missingDirs {
		dirPath := filepath.Join(cwd, dir)
		if err := os.MkdirAll(dirPath, 0755); err == nil {
			ui.Success("Created %s", dir)
		}
	}

	downloader, err := core.NewDownloader()
	if err != nil {
		ui.Error("Failed to initialize downloader: %v", err)
		return
	}

	cachePath, err := downloader.DownloadVersion(config.Version)
	if err != nil {
		ui.Error("Failed to download version: %v", err)
		return
	}

	restoreMissingComponents(cwd, cachePath, config)
	ui.Success("Fix complete. Run 'samuel doctor' again to verify.")
}

// restoreMissingComponents copies missing component files from cache.
func restoreMissingComponents(cwd, cachePath string, config *core.Config) {
	paths := core.GetComponentPaths(
		config.Installed.Languages,
		config.Installed.Frameworks,
		config.Installed.Workflows,
	)

	for _, path := range paths {
		localPath := filepath.Join(cwd, path)
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			if err := core.CopyFromCache(cachePath, cwd, path); err == nil {
				ui.Success("Restored %s", path)
			} else {
				ui.Error("Failed to restore %s: %v", path, err)
			}
		}
	}
}
