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

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Samuel framework to the latest version",
	Long: `Update the installed Samuel framework to the latest version.

This command will:
1. Check for available updates
2. Download the new version
3. Apply updates while preserving local modifications
4. Create backups of modified files

Examples:
  samuel update              # Update to latest version
  samuel update --check      # Check for updates without applying
  samuel update --diff       # Show what will change
  samuel update --force      # Overwrite local modifications`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().Bool("check", false, "Check for updates without applying")
	updateCmd.Flags().Bool("diff", false, "[DEPRECATED] Use --preview instead")
	updateCmd.Flags().Bool("preview", false, "Show what files will change without applying (replaces 'samuel diff')")
	updateCmd.Flags().BoolP("force", "f", false, "Overwrite local modifications")
	updateCmd.Flags().String("version", "", "Update to specific version")
	// Hide the legacy --diff flag from help output. It still works for v3.0.x
	// scripts, but the user-facing flag is --preview.
	if f := updateCmd.Flags().Lookup("diff"); f != nil {
		f.Hidden = true
	}
}

func runUpdate(cmd *cobra.Command, args []string) error {
	checkOnly, _ := cmd.Flags().GetBool("check")
	// --preview is the v3 spelling; --diff is the v2 alias kept Hidden for one
	// minor release. Either flag triggers the same dry-run output.
	showPreview, _ := cmd.Flags().GetBool("preview")
	showDiff, _ := cmd.Flags().GetBool("diff")
	showDiff = showDiff || showPreview
	force, _ := cmd.Flags().GetBool("force")
	targetVersion, _ := cmd.Flags().GetString("version")

	config, err := core.LoadConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no Samuel installation found. Run 'samuel init' first")
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	cachePath, targetVersion, err := downloadTargetVersion(
		config.Version, targetVersion, checkOnly, force,
	)
	if err != nil {
		return err
	}
	if cachePath == "" {
		return nil // up-to-date or check-only
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	paths := core.GetComponentPaths(
		config.Installed.Languages,
		config.Installed.Frameworks,
		config.Installed.Workflows,
	)
	extractor := core.NewExtractor(cachePath, cwd)
	changes := categorizeFileChanges(paths, cwd, cachePath)

	if showDiff {
		if JSONMode(cmd) {
			ui.PrintJSON("update", map[string]interface{}{
				"currentVersion": config.Version,
				"targetVersion":  targetVersion,
				"newFiles":       changes.newFiles,
				"modifiedFiles":  changes.modifiedFiles,
				"unchangedFiles": len(changes.unchangedFiles),
				"dryRun":         true,
			})
			return nil
		}
		displayChangeDiff(changes, force)
		return nil
	}

	previousVersion := config.Version

	err = applyUpdate(extractor, changes, force, cwd, targetVersion, config)
	if err != nil {
		return err
	}

	if JSONMode(cmd) {
		ui.PrintJSON("update", map[string]interface{}{
			"previousVersion": previousVersion,
			"newVersion":      targetVersion,
			"newFiles":        len(changes.newFiles),
			"modifiedFiles":   len(changes.modifiedFiles),
			"unchangedFiles":  len(changes.unchangedFiles),
		})
	}

	return nil
}

// downloadTargetVersion resolves the target version, checks if an update is needed,
// and downloads it. Returns empty cachePath if no update is needed.
func downloadTargetVersion(currentVersion, targetVersion string, checkOnly, force bool) (string, string, error) {
	downloader, err := core.NewDownloader()
	if err != nil {
		return "", "", fmt.Errorf("failed to initialize: %w", err)
	}

	if targetVersion == "" {
		spinner := ui.NewSpinner("Checking for updates...")
		spinner.Start()
		latest, err := downloader.GetLatestVersion()
		if err != nil {
			spinner.Error("Failed to check for updates")
			return "", "", fmt.Errorf("failed to get latest version: %w", err)
		}
		spinner.Stop()
		targetVersion = latest
	}

	ui.Bold("Samuel Update")
	ui.TableRow("Current version", currentVersion)
	ui.TableRow("Target version", targetVersion)

	if currentVersion == targetVersion && !force {
		fmt.Println()
		ui.Success("Already up to date!")
		return "", targetVersion, nil
	}

	if checkOnly {
		if currentVersion != targetVersion {
			fmt.Println()
			ui.Success("Update available: %s → %s", currentVersion, targetVersion)
			ui.Info("Run 'samuel update' to apply")
		}
		return "", targetVersion, nil
	}

	spinner := ui.NewSpinner("Downloading...")
	spinner.Start()
	cachePath, err := downloader.DownloadVersion(targetVersion)
	if err != nil {
		spinner.Error("Download failed")
		return "", "", fmt.Errorf("failed to download: %w", err)
	}
	spinner.Success(fmt.Sprintf("Downloaded v%s", targetVersion))

	return cachePath, targetVersion, nil
}

// displayChangeDiff prints the file change summary without applying updates.
func displayChangeDiff(changes fileChanges, force bool) {
	fmt.Println()
	ui.Section("Changes")

	if len(changes.newFiles) > 0 {
		ui.ListItem(1, "%d new files:", len(changes.newFiles))
		for _, f := range changes.newFiles {
			ui.SuccessItem(2, "%s", f)
		}
	}

	if len(changes.modifiedFiles) > 0 {
		ui.ListItem(1, "%d files with local modifications:", len(changes.modifiedFiles))
		for _, f := range changes.modifiedFiles {
			ui.WarnItem(2, "%s", f)
		}
	}

	if len(changes.unchangedFiles) > 0 {
		ui.ListItem(1, "%d files to update:", len(changes.unchangedFiles))
	}

	fmt.Println()
	if !force {
		ui.Info("Modified files will be preserved. Use --force to overwrite.")
	}
}

// applyUpdate backs up modified files, extracts updates, and saves the config.
func applyUpdate(
	extractor *core.Extractor, changes fileChanges,
	force bool, cwd, targetVersion string, config *core.Config,
) error {
	var backupDir string
	if len(changes.modifiedFiles) > 0 && !force {
		var err error
		backupDir, err = backupModifiedFiles(extractor, changes.modifiedFiles, cwd)
		if err != nil {
			return err
		}
	}

	var filesToUpdate []string
	filesToUpdate = append(filesToUpdate, changes.newFiles...)
	filesToUpdate = append(filesToUpdate, changes.unchangedFiles...)
	if force {
		filesToUpdate = append(filesToUpdate, changes.modifiedFiles...)
	}

	result, err := extractor.Extract(filesToUpdate, true)
	if err != nil {
		return fmt.Errorf("failed to apply updates: %w", err)
	}

	ui.Success("Updated %d files", len(result.FilesCreated))
	reportUpdateResults(changes, force, backupDir)

	config.Version = targetVersion
	if err := config.Save(cwd); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	ui.Success("Updated samuel.yaml to v%s", targetVersion)

	return nil
}

// backupModifiedFiles creates a timestamped backup directory and copies files into it.
func backupModifiedFiles(
	extractor *core.Extractor, modifiedFiles []string, cwd string,
) (string, error) {
	backupDir := filepath.Join(
		cwd,
		fmt.Sprintf(".samuel-backup-%s", time.Now().Format("20060102-150405")),
	)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	for _, f := range modifiedFiles {
		if err := extractor.BackupFile(f, backupDir); err != nil {
			ui.Warn("Failed to backup %s: %v", f, err)
		}
	}
	ui.Success("Backed up %d modified files to %s", len(modifiedFiles), backupDir)

	return backupDir, nil
}

// reportUpdateResults displays the update summary and preserved file instructions.
func reportUpdateResults(changes fileChanges, force bool, backupDir string) {
	if len(changes.newFiles) > 0 {
		ui.Success("Added %d new files", len(changes.newFiles))
	}

	if len(changes.modifiedFiles) > 0 && !force {
		ui.Warn("Preserved %d locally modified files", len(changes.modifiedFiles))
		if backupDir != "" {
			ui.Info("Backups saved to: %s", backupDir)
		}
	}

	if len(changes.modifiedFiles) > 0 && !force {
		fmt.Println()
		ui.Bold("Modified files preserved:")
		for _, f := range changes.modifiedFiles {
			ui.WarnItem(1, "%s", f)
		}
		ui.Info("\nTo see changes: diff -u %s/<file> <file>", backupDir)
		ui.Info("To accept new version: cp %s/<file> <file>", backupDir)
	}
}

// fileChanges holds the categorized file lists from comparing local vs cached files.
type fileChanges struct {
	newFiles       []string
	modifiedFiles  []string
	unchangedFiles []string
}

// categorizeFileChanges compares component paths between the local project and
// the cache, categorizing each file as new, modified, or unchanged.
func categorizeFileChanges(paths []string, cwd, cachePath string) fileChanges {
	var changes fileChanges

	for _, path := range paths {
		localPath := filepath.Join(cwd, path)
		cacheSrcPath := filepath.Join(cachePath, path)

		if !fileExists(cacheSrcPath) {
			continue
		}

		if !fileExists(localPath) {
			changes.newFiles = append(changes.newFiles, path)
			continue
		}

		localContent, err := os.ReadFile(localPath)
		if err != nil {
			ui.Warn("Skipping %s: failed to read local file: %v", path, err)
			continue
		}

		cacheContent, err := os.ReadFile(cacheSrcPath)
		if err != nil {
			ui.Warn("Skipping %s: failed to read cached file: %v", path, err)
			continue
		}

		if string(localContent) != string(cacheContent) {
			changes.modifiedFiles = append(changes.modifiedFiles, path)
		} else {
			changes.unchangedFiles = append(changes.unchangedFiles, path)
		}
	}

	return changes
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
