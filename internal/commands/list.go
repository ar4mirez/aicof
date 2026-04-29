package commands

import (
	"fmt"
	"os"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

// listCmd is preserved as a Hidden+Deprecated alias for v3.0.0. Use 'samuel ls'.
// The wrapper prints a one-line redirect (suppressed by SAMUEL_NO_DEPRECATION=1
// or --no-deprecation) and forwards to the same handler.
var listCmd = &cobra.Command{
	Use:    "list",
	Short:  "[DEPRECATED] Use 'samuel ls'",
	Hidden: true,
	RunE:   redirectAndRun("samuel ls", runList),
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolP("available", "a", false, "Show all available components")
	listCmd.Flags().StringP("type", "t", "", "Filter by type: languages, frameworks, workflows")
}

func runList(cmd *cobra.Command, args []string) error {
	showAvailable, _ := cmd.Flags().GetBool("available")
	typeFilter, _ := cmd.Flags().GetString("type")

	if JSONMode(cmd) {
		return listJSON(cmd, showAvailable, typeFilter)
	}

	if showAvailable {
		return listAvailable(typeFilter)
	}

	return listInstalled(typeFilter)
}

func listJSON(cmd *cobra.Command, showAvailable bool, typeFilter string) error {
	type componentJSON struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description"`
		Installed   bool   `json:"installed"`
	}

	config, _ := core.LoadConfig()
	var components []componentJSON

	addComponents := func(compType string, items []core.Component, isInstalled func(string) bool) {
		if typeFilter != "" && typeFilter != compType {
			return
		}
		for _, c := range items {
			installed := isInstalled(c.Name)
			if !showAvailable && !installed {
				continue
			}
			components = append(components, componentJSON{
				Name:        c.Name,
				Type:        compType,
				Description: c.Description,
				Installed:   installed,
			})
		}
	}

	hasLang := func(n string) bool { return config != nil && config.HasLanguage(n) }
	hasFw := func(n string) bool { return config != nil && config.HasFramework(n) }
	hasWf := func(n string) bool { return config != nil && config.HasWorkflow(n) }

	addComponents("language", core.Languages, hasLang)
	addComponents("framework", core.Frameworks, hasFw)
	addComponents("workflow", core.Workflows, hasWf)

	version := ""
	if config != nil {
		version = config.Version
	}

	PrintJSONForCmd(cmd, map[string]interface{}{
		"version":    version,
		"available":  showAvailable,
		"components": components,
	})
	return nil
}

func listInstalled(typeFilter string) error {
	config, err := core.LoadConfig()
	if err != nil {
		if os.IsNotExist(err) {
			ui.Warn("No Samuel installation found in current directory")
			ui.Info("Run 'samuel init' to initialize or 'samuel list --available' to see available components")
			return nil
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	ui.Bold("Samuel Framework v%s", config.Version)
	fmt.Println()

	// Languages
	if typeFilter == "" || typeFilter == "languages" {
		ui.Section(fmt.Sprintf("Languages (%d/%d)", len(config.Installed.Languages), len(core.Languages)))
		if len(config.Installed.Languages) == 0 {
			ui.Dim("  None installed")
		} else {
			for _, name := range config.Installed.Languages {
				if lang := core.FindLanguage(name); lang != nil {
					ui.SuccessItem(1, "%s - %s", lang.Name, lang.Description)
				} else {
					ui.SuccessItem(1, "%s", name)
				}
			}
		}
	}

	// Frameworks
	if typeFilter == "" || typeFilter == "frameworks" {
		ui.Section(fmt.Sprintf("Frameworks (%d/%d)", len(config.Installed.Frameworks), len(core.Frameworks)))
		if len(config.Installed.Frameworks) == 0 {
			ui.Dim("  None installed")
		} else {
			for _, name := range config.Installed.Frameworks {
				if fw := core.FindFramework(name); fw != nil {
					ui.SuccessItem(1, "%s - %s", fw.Name, fw.Description)
				} else {
					ui.SuccessItem(1, "%s", name)
				}
			}
		}
	}

	// Workflows
	if typeFilter == "" || typeFilter == "workflows" {
		workflowCount := len(config.Installed.Workflows)
		if len(config.Installed.Workflows) == 1 && config.Installed.Workflows[0] == "all" {
			workflowCount = len(core.Workflows)
		}
		ui.Section(fmt.Sprintf("Workflows (%d/%d)", workflowCount, len(core.Workflows)))

		if len(config.Installed.Workflows) == 1 && config.Installed.Workflows[0] == "all" {
			for _, wf := range core.Workflows {
				ui.SuccessItem(1, "%s - %s", wf.Name, wf.Description)
			}
		} else if len(config.Installed.Workflows) == 0 {
			ui.Dim("  None installed")
		} else {
			for _, name := range config.Installed.Workflows {
				if wf := core.FindWorkflow(name); wf != nil {
					ui.SuccessItem(1, "%s - %s", wf.Name, wf.Description)
				} else {
					ui.SuccessItem(1, "%s", name)
				}
			}
		}
	}

	return nil
}

func listAvailable(typeFilter string) error {
	ui.Bold("Available Samuel Components")
	fmt.Println()

	// Check if installed to mark installed items
	config, configErr := core.LoadConfig()
	if configErr != nil && !os.IsNotExist(configErr) {
		ui.Warn("Could not load config: %v", configErr)
	}

	// Languages
	if typeFilter == "" || typeFilter == "languages" {
		ui.Section(fmt.Sprintf("Languages (%d)", len(core.Languages)))
		for _, lang := range core.Languages {
			installed := config != nil && config.HasLanguage(lang.Name)
			if installed {
				ui.SuccessItem(1, "%s - %s (installed)", lang.Name, lang.Description)
			} else {
				ui.ListItem(1, "%s - %s", lang.Name, lang.Description)
			}
		}
	}

	// Frameworks
	if typeFilter == "" || typeFilter == "frameworks" {
		ui.Section(fmt.Sprintf("Frameworks (%d)", len(core.Frameworks)))
		for _, fw := range core.Frameworks {
			installed := config != nil && config.HasFramework(fw.Name)
			if installed {
				ui.SuccessItem(1, "%s - %s (installed)", fw.Name, fw.Description)
			} else {
				ui.ListItem(1, "%s - %s", fw.Name, fw.Description)
			}
		}
	}

	// Workflows
	if typeFilter == "" || typeFilter == "workflows" {
		ui.Section(fmt.Sprintf("Workflows (%d)", len(core.Workflows)))
		for _, wf := range core.Workflows {
			installed := config != nil && config.HasWorkflow(wf.Name)
			if installed {
				ui.SuccessItem(1, "%s - %s (installed)", wf.Name, wf.Description)
			} else {
				ui.ListItem(1, "%s - %s", wf.Name, wf.Description)
			}
		}
	}

	return nil
}
