package commands

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Samuel configuration",
	Long: `View and modify Samuel configuration settings.

Available subcommands:
  list   Show all configuration values
  get    Get a specific configuration value
  set    Set a configuration value

Valid configuration keys:
  version              Framework version
  registry             GitHub repository URL
  installed.languages  Comma-separated list of installed languages
  installed.frameworks Comma-separated list of installed frameworks
  installed.workflows  Comma-separated list of installed workflows

Examples:
  samuel config list                           # Show all config values
  samuel config get version                    # Get framework version
  samuel config set registry https://...       # Set custom registry
  samuel config set installed.languages go,rust  # Set installed languages`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all configuration values",
	Long: `Display all Samuel configuration values from samuel.yaml.

Example:
  samuel config list`,
	RunE: runConfigList,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long: `Get a specific configuration value by key.

Valid keys:
  version, registry, installed.languages, installed.frameworks, installed.workflows

Examples:
  samuel config get version
  samuel config get installed.languages`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value by key.

For list values (installed.*), use comma-separated format.

Examples:
  samuel config set registry https://github.com/myorg/myrepo
  samuel config set installed.languages typescript,python,go
  samuel config set installed.frameworks react,nextjs`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

func init() {
	adminCmd.AddCommand(configCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
}

func runConfigList(cmd *cobra.Command, args []string) error {
	config, err := core.LoadConfig()
	if err != nil {
		if os.IsNotExist(err) {
			if JSONMode(cmd) {
				ui.PrintJSONError("config list", fmt.Errorf("no Samuel installation found"))
			} else {
				ui.Warn("No Samuel installation found in current directory")
				ui.Info("Run 'samuel init' to initialize a project")
			}
			return nil
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	if JSONMode(cmd) {
		ui.PrintJSON("config list", config.GetAllValues())
		return nil
	}

	ui.Bold("Samuel Configuration")
	fmt.Println()

	values := config.GetAllValues()

	// Display in consistent order
	keys := []string{"version", "registry", "installed.languages", "installed.frameworks", "installed.workflows"}
	for _, key := range keys {
		value := values[key]
		displayValue := formatConfigValue(value)
		ui.Print("  %-24s %s", key+":", displayValue)
	}

	fmt.Println()
	ui.Dim("Config file: samuel.yaml")

	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	// Validate key
	if !isValidConfigKey(key) {
		return fmt.Errorf("invalid config key: %s", key)
	}

	config, err := core.LoadConfig()
	if err != nil {
		if os.IsNotExist(err) {
			if JSONMode(cmd) {
				ui.PrintJSONError("config get", fmt.Errorf("no Samuel installation found"))
			} else {
				ui.Warn("No Samuel installation found in current directory")
			}
			return nil
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	value, err := config.GetValue(key)
	if err != nil {
		return err
	}

	if JSONMode(cmd) {
		ui.PrintJSON("config get", map[string]interface{}{
			"key":   key,
			"value": value,
		})
		return nil
	}

	// Output raw value for scripting
	fmt.Println(formatConfigValue(value))
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Validate key
	if !isValidConfigKey(key) {
		ui.Error("Invalid config key: %s", key)
		ui.Info("Valid keys: %s", strings.Join(core.ValidConfigKeys, ", "))
		return fmt.Errorf("invalid config key: %s", key)
	}

	// Validate registry URL scheme before loading config
	if key == "registry" {
		if err := validateRegistryURL(value); err != nil {
			return fmt.Errorf("invalid registry value: %w", err)
		}
	}

	config, err := core.LoadConfig()
	if err != nil {
		if os.IsNotExist(err) {
			ui.Warn("No Samuel installation found in current directory")
			ui.Info("Run 'samuel init' to initialize a project")
			return nil
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get old value for display
	oldValue, _ := config.GetValue(key)

	// Set new value
	if err := config.SetValue(key, value); err != nil {
		return err
	}

	// Save config
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := config.Save(cwd); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	if JSONMode(cmd) {
		ui.PrintJSON("config set", map[string]interface{}{
			"key":      key,
			"oldValue": oldValue,
			"newValue": value,
		})
		return nil
	}

	ui.Success("Updated %s", key)
	ui.Dim("  Old: %s", formatConfigValue(oldValue))
	ui.Dim("  New: %s", formatConfigValue(value))

	return nil
}

func formatConfigValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		if v == "" {
			return "(empty)"
		}
		return v
	case []string:
		if len(v) == 0 {
			return "(none)"
		}
		return strings.Join(v, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

func isValidConfigKey(key string) bool {
	for _, k := range core.ValidConfigKeys {
		if k == key {
			return true
		}
	}
	return false
}

// validateRegistryURL checks that a registry value is a valid HTTPS URL.
func validateRegistryURL(value string) error {
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("registry must use HTTPS scheme, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("registry URL must have a host")
	}
	return nil
}
