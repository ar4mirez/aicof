package commands

import (
	"github.com/spf13/cobra"
)

var (
	// Version information (set at build time)
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "samuel",
	Short: "Samuel - Artificial Intelligence Coding Framework CLI",
	Long: `Samuel CLI manages the Artificial Intelligence Coding Framework.

It helps you initialize projects with AI coding guardrails, update framework
versions, and manage language/framework guides without cloning the repository.

Examples:
  samuel init my-project          # Initialize a new project
  samuel init .                   # Initialize in current directory
  samuel update                   # Update to latest framework version
  samuel add rust                 # Add a component (type inferred)
  samuel ls --all                 # List all available components
  samuel doctor                   # Check installation health
  samuel run                      # Status of autonomous loop (or 'samuel auto')`,
	SilenceUsage:  true,
	SilenceErrors: true,
	// SuggestionsMinimumDistance enables Cobra's "did you mean?" hint when a
	// user types an unknown command. Distance 2 is the documented sweet spot —
	// catches typos like 'samuel buld' -> 'samuel build' without firing on
	// genuinely different inputs. Cargo and gh use the same default.
	SuggestionsMinimumDistance: 2,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// JSONMode returns true when the --json flag is set on the command.
func JSONMode(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	if f := cmd.Flags().Lookup("json"); f != nil {
		return f.Value.String() == "true"
	}
	if f := cmd.Root().PersistentFlags().Lookup("json"); f != nil {
		return f.Value.String() == "true"
	}
	return false
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().Bool("json", false, "Output in JSON format for programmatic consumption")
	rootCmd.PersistentFlags().Bool("no-deprecation", false, "Suppress legacy-command deprecation warnings (CI scripts)")
}
