package commands

import (
	"fmt"
	"os"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <name> [type]",
	Short: "Add a component to your project",
	Long: `Add a language guide, framework guide, or workflow to your project.

In v3, type is optional — Samuel infers it from the name when unambiguous.
Both arg orders are accepted: 'samuel add react' (inferred), 'samuel add react framework'
(name first, v3 style), and 'samuel add framework react' (type first, v2 style)
all work identically.

Types (with aliases):
  language   (lang, l)   Language guides
  framework  (fw, f)     Framework guides
  workflow   (wf, w)     Workflow templates

Examples:
  samuel add react                     # v3: type inferred (framework)
  samuel add typescript                # inferred (language)
  samuel add react framework           # v3: name first
  samuel add framework react           # v2 form, still works
  samuel add lang rust                 # short type alias`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	componentType, componentName, err := parseAddRemoveArgs("add", args)
	if err != nil {
		return err
	}

	config, err := core.LoadConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no Samuel installation found. Run 'samuel init' first")
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	component, alreadyInstalled, err := resolveComponent(componentType, componentName, config)
	if err != nil {
		return err
	}
	if alreadyInstalled {
		if JSONMode(cmd) {
			ui.PrintJSON("add", map[string]interface{}{
				"type":             componentType,
				"name":             componentName,
				"alreadyInstalled": true,
			})
			return nil
		}
		ui.Warn("%s '%s' is already installed", componentType, componentName)
		return nil
	}

	if err := downloadAndInstall(config.Version, component); err != nil {
		return err
	}

	if err := updateAddConfig(config, componentType, componentName, component.Path); err != nil {
		return err
	}

	if JSONMode(cmd) {
		ui.PrintJSON("add", map[string]interface{}{
			"type":      componentType,
			"name":      componentName,
			"path":      component.Path,
			"installed": true,
		})
	}

	return nil
}

// resolveComponent validates the component type, finds it in the registry,
// and checks whether it's already installed in the given config.
func resolveComponent(componentType, componentName string, config *core.Config) (*core.Component, bool, error) {
	switch componentType {
	case "language", "lang", "l":
		component := core.FindLanguage(componentName)
		if component == nil {
			return nil, false, fmt.Errorf("unknown language: %s\nRun 'samuel list --available --type languages' to see available languages", componentName)
		}
		return component, config.HasLanguage(componentName), nil
	case "framework", "fw", "f":
		component := core.FindFramework(componentName)
		if component == nil {
			return nil, false, fmt.Errorf("unknown framework: %s\nRun 'samuel list --available --type frameworks' to see available frameworks", componentName)
		}
		return component, config.HasFramework(componentName), nil
	case "workflow", "wf", "w":
		component := core.FindWorkflow(componentName)
		if component == nil {
			return nil, false, fmt.Errorf("unknown workflow: %s\nRun 'samuel list --available --type workflows' to see available workflows", componentName)
		}
		return component, config.HasWorkflow(componentName), nil
	default:
		return nil, false, fmt.Errorf("unknown component type: %s\nValid types: language, framework, workflow", componentType)
	}
}

// downloadAndInstall downloads the framework version and copies the component to the current directory.
func downloadAndInstall(version string, component *core.Component) error {
	spinner := ui.NewSpinner(fmt.Sprintf("Downloading %s...", component.Name))
	spinner.Start()

	downloader, err := core.NewDownloader()
	if err != nil {
		spinner.Error("Failed to initialize")
		return fmt.Errorf("failed to initialize: %w", err)
	}

	cachePath, err := downloader.DownloadVersion(version)
	if err != nil {
		spinner.Error("Download failed")
		return fmt.Errorf("failed to download: %w", err)
	}
	spinner.Stop()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	if err := core.CopyFromCache(cachePath, cwd, component.Path); err != nil {
		return fmt.Errorf("failed to install %s: %w", component.Name, err)
	}

	return nil
}

// updateAddConfig adds the component to the project config and saves it.
func updateAddConfig(config *core.Config, componentType, componentName, componentPath string) error {
	switch componentType {
	case "language", "lang", "l":
		config.AddLanguage(componentName)
	case "framework", "fw", "f":
		config.AddFramework(componentName)
	case "workflow", "wf", "w":
		config.AddWorkflow(componentName)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	if err := config.Save(cwd); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	ui.Success("Installed %s", componentPath)
	ui.Success("Updated samuel.yaml")
	return nil
}
