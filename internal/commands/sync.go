package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync per-folder CLAUDE.md and AGENTS.md files",
	Long: `Recursively scan the project and create or update per-folder CLAUDE.md
and AGENTS.md files with context-aware content based on folder analysis.

Files are auto-generated with language detection, purpose inference,
and key file identification. User-customized files are preserved unless
--force is specified.

Examples:
  samuel sync                # Sync all directories
  samuel sync --depth 1      # Only top-level directories
  samuel sync --dry-run      # Preview without writing
  samuel sync --force        # Overwrite user-customized files`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().IntP("depth", "d", -1, "Max recursion depth (-1=unlimited)")
	syncCmd.Flags().BoolP("force", "f", false, "Overwrite user-customized files")
	syncCmd.Flags().Bool("dry-run", false, "Preview changes without writing files")
}

func runSync(cmd *cobra.Command, args []string) error {
	depth, _ := cmd.Flags().GetInt("depth")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	rootDir := "."
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if dryRun {
		ui.Info("Dry run — no files will be written")
	}
	ui.Header("Syncing per-folder CLAUDE.md files...")

	result, err := core.SyncFolderCLAUDEMDs(core.SyncOptions{
		RootDir:  absRoot,
		MaxDepth: depth,
		Force:    force,
		DryRun:   dryRun,
	})
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	if JSONMode(cmd) {
		toRel := func(paths []string) []string {
			out := make([]string, 0, len(paths))
			for _, p := range paths {
				out = append(out, relPath(absRoot, p))
			}
			return out
		}
		errStrs := make([]string, 0, len(result.Errors))
		for _, e := range result.Errors {
			errStrs = append(errStrs, e.Error())
		}
		ui.PrintJSON("sync", map[string]interface{}{
			"dryRun":  dryRun,
			"created": toRel(result.Created),
			"updated": toRel(result.Updated),
			"skipped": toRel(result.Skipped),
			"errors":  errStrs,
		})
		return nil
	}

	// Display results
	for _, f := range result.Created {
		rel := relPath(absRoot, f)
		ui.SuccessItem(1, "Created %s", rel)
	}
	for _, f := range result.Updated {
		rel := relPath(absRoot, f)
		ui.Info("  ~ Updated %s", rel)
	}
	for _, f := range result.Skipped {
		rel := relPath(absRoot, f)
		ui.Dim("  - Skipped %s (user-customized)", rel)
	}
	for _, e := range result.Errors {
		ui.Error("  ! %v", e)
	}

	// Summary
	fmt.Println()
	parts := []string{}
	if len(result.Created) > 0 {
		parts = append(parts, fmt.Sprintf("%d created", len(result.Created)))
	}
	if len(result.Updated) > 0 {
		parts = append(parts, fmt.Sprintf("%d updated", len(result.Updated)))
	}
	if len(result.Skipped) > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", len(result.Skipped)))
	}
	if len(result.Errors) > 0 {
		parts = append(parts, fmt.Sprintf("%d errors", len(result.Errors)))
	}

	if len(parts) == 0 {
		ui.Info("No directories found to sync")
	} else {
		ui.Bold("Summary: %s", strings.Join(parts, ", "))
	}

	return nil
}

// relPath returns a relative path from base, falling back to the full path.
func relPath(base, full string) string {
	rel, err := filepath.Rel(base, full)
	if err != nil {
		return full
	}
	return rel
}
