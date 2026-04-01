package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <type> <name>",
	Short: "Remove a component from your project",
	Long: `Remove a language guide, framework guide, or workflow from your project.

This removes the file and updates the config. Core files (CLAUDE.md, workflows)
cannot be removed.

Types:
  language   Remove a language guide
  framework  Remove a framework guide
  workflow   Remove a workflow (only individual workflows, not 'all')

Examples:
  samuel remove language rust
  samuel remove framework django`,
	Args: cobra.ExactArgs(2),
	RunE: runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")
}

func runRemove(cmd *cobra.Command, args []string) error {
	componentType := args[0]
	componentName := args[1]
	force, _ := cmd.Flags().GetBool("force")

	// Load config
	config, err := core.LoadConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no Samuel installation found. Run 'samuel init' first")
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate component type and find component
	var component *core.Component
	var isInstalled bool

	switch componentType {
	case "language", "lang", "l":
		component = core.FindLanguage(componentName)
		if component == nil {
			return fmt.Errorf("unknown language: %s", componentName)
		}
		isInstalled = config.HasLanguage(componentName)

	case "framework", "fw", "f":
		component = core.FindFramework(componentName)
		if component == nil {
			return fmt.Errorf("unknown framework: %s", componentName)
		}
		isInstalled = config.HasFramework(componentName)

	case "workflow", "wf", "w":
		// Don't allow removing all workflows
		if componentName == "all" {
			return fmt.Errorf("cannot remove 'all' workflows. Remove individual workflows instead")
		}
		component = core.FindWorkflow(componentName)
		if component == nil {
			return fmt.Errorf("unknown workflow: %s", componentName)
		}
		isInstalled = config.HasWorkflow(componentName)

	default:
		return fmt.Errorf("unknown component type: %s\nValid types: language, framework, workflow", componentType)
	}

	// Check if installed
	if !isInstalled {
		if JSONMode(cmd) {
			ui.PrintJSON("remove", map[string]interface{}{
				"type":         componentType,
				"name":         componentName,
				"wasInstalled": false,
			})
			return nil
		}
		ui.Warn("%s '%s' is not installed", componentType, componentName)
		return nil
	}

	// Confirm removal
	if !force {
		confirmed, err := ui.Confirm(fmt.Sprintf("Remove %s '%s'?", componentType, componentName), false)
		if err != nil || !confirmed {
			ui.Info("Removal cancelled")
			return nil
		}
	}

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Remove the file (validate path stays within project directory)
	filePath, err := validateRemovePath(cwd, component.Path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filePath); err == nil {
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove file: %w", err)
		}
		ui.Success("Removed %s", component.Path)
	} else {
		ui.Warn("File not found: %s (updating config anyway)", component.Path)
	}

	// Update config
	switch componentType {
	case "language", "lang", "l":
		config.RemoveLanguage(componentName)
	case "framework", "fw", "f":
		config.RemoveFramework(componentName)
	case "workflow", "wf", "w":
		config.RemoveWorkflow(componentName)
	}

	if err := config.Save(cwd); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	if JSONMode(cmd) {
		ui.PrintJSON("remove", map[string]interface{}{
			"type":    componentType,
			"name":    componentName,
			"path":    component.Path,
			"removed": true,
		})
		return nil
	}

	ui.Success("Updated samuel.yaml")

	return nil
}

// validateRemovePath checks that a component path, when joined with the
// project directory, stays within that directory (prevents path traversal).
func validateRemovePath(projectDir, componentPath string) (string, error) {
	cleanDir := filepath.Clean(projectDir)
	fullPath := filepath.Clean(filepath.Join(cleanDir, componentPath))
	if fullPath != cleanDir && !strings.HasPrefix(fullPath, cleanDir+string(os.PathSeparator)) {
		return "", fmt.Errorf("path traversal detected: component path %q escapes project directory", componentPath)
	}
	return fullPath, nil
}
