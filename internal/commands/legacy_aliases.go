package commands

import (
	"github.com/spf13/cobra"
)

// Legacy top-level aliases for commands that moved under `samuel admin` in v3.0.0.
// They forward to the new handlers via redirectAndRun, which prints a one-line
// deprecation notice (suppressed by SAMUEL_NO_DEPRECATION=1 or --no-deprecation).
// These wrappers remove in v3.1.0 — see plan: as-a-user-iterative-swan.md.

// --- samuel config (renamed to samuel admin config) ---

// Cobra's built-in Deprecated field auto-prints on every invocation and is not
// gated by SAMUEL_NO_DEPRECATION. We use Hidden + redirectAndRun so the
// suppression flag actually works for CI scripts.
var legacyConfigCmd = &cobra.Command{
	Use:    "config",
	Short:  "[DEPRECATED] Use 'samuel admin config'",
	Hidden: true,
	RunE: redirectAndRun("samuel admin config", func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	}),
}

var legacyConfigListCmd = &cobra.Command{
	Use:    "list",
	Short:  "[DEPRECATED] Use 'samuel admin config list'",
	Hidden: true,
	RunE:   redirectAndRun("samuel admin config list", runConfigList),
}

var legacyConfigGetCmd = &cobra.Command{
	Use:    "get <key>",
	Short:  "[DEPRECATED] Use 'samuel admin config get'",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE:   redirectAndRun("samuel admin config get", runConfigGet),
}

var legacyConfigSetCmd = &cobra.Command{
	Use:    "set <key> <value>",
	Short:  "[DEPRECATED] Use 'samuel admin config set'",
	Hidden: true,
	Args:   cobra.ExactArgs(2),
	RunE:   redirectAndRun("samuel admin config set", runConfigSet),
}

// --- samuel sync (renamed to samuel admin sync) ---

var legacySyncCmd = &cobra.Command{
	Use:    "sync",
	Short:  "[DEPRECATED] Use 'samuel admin sync'",
	Hidden: true,
	RunE:   redirectAndRun("samuel admin sync", runSync),
}

func init() {
	rootCmd.AddCommand(legacyConfigCmd)
	legacyConfigCmd.AddCommand(legacyConfigListCmd)
	legacyConfigCmd.AddCommand(legacyConfigGetCmd)
	legacyConfigCmd.AddCommand(legacyConfigSetCmd)

	rootCmd.AddCommand(legacySyncCmd)
	// Mirror sync's flags so the legacy invocation accepts the same options.
	legacySyncCmd.Flags().IntP("depth", "d", -1, "Max recursion depth (-1=unlimited)")
	legacySyncCmd.Flags().BoolP("force", "f", false, "Overwrite user-customized files")
	legacySyncCmd.Flags().Bool("dry-run", false, "Preview changes without writing files")
}
