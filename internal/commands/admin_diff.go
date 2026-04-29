package commands

import (
	"github.com/spf13/cobra"
)

// adminDiffCmd preserves the v2.x two-version comparison form under the admin
// namespace. The bare no-arg form is also accepted (compares installed vs
// latest), even though `samuel update --preview` is the recommended path for
// that case — admin diff stays internally consistent for users who want one
// command for both shapes.
var adminDiffCmd = &cobra.Command{
	Use:   "diff [version1] [version2]",
	Short: "Compare versions to see what changed",
	Long: `Compare Samuel versions to see what files have been added, removed, or modified.

Without arguments, compares installed files with the latest available version.
With two version arguments, compares those specific versions.

Examples:
  samuel admin diff                  # Compare installed vs latest
  samuel admin diff v1.6.0 v1.7.0    # Compare two specific versions

For the no-args case, prefer 'samuel update --preview' — same output, cleaner verb.
This command exists to support the two-arg comparison form.`,
	Args: cobra.MaximumNArgs(2),
	RunE: runDiff,
}

func init() {
	adminCmd.AddCommand(adminDiffCmd)
	adminDiffCmd.Flags().BoolP("installed", "i", false, "Compare installed files with latest version")
	adminDiffCmd.Flags().Bool("components", false, "Show component-level changes instead of files")
}
