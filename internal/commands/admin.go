package commands

import (
	"github.com/spf13/cobra"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Power-user commands (config, sync, diff)",
	Long: `Administrative commands grouped together to keep the top-level CLI lean.

Subcommands:
  config   Manage Samuel configuration
  sync     Sync per-folder CLAUDE.md and AGENTS.md files
  diff     Compare two specific versions of the framework

Examples:
  samuel admin config list
  samuel admin sync --dry-run
  samuel admin diff v2.5.0 v2.6.0`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(adminCmd)
	// configCmd and syncCmd register themselves under adminCmd in their own
	// init() functions. Package-level variable initialization runs before any
	// init(), so adminCmd is non-nil here regardless of init() ordering.
}
