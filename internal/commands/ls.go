package commands

import (
	"fmt"
	"strings"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/spf13/cobra"
)

// lsCmd is the v3.0.0 unified discovery verb. It collapses the v2.x list,
// search, and info commands into one flag-driven command following fly/vercel
// ergonomics:
//
//	samuel ls                    # installed components (was: list)
//	samuel ls --all              # all available components (was: list --available)
//	samuel ls react              # search (was: search react)
//	samuel ls react --detail     # info (was: info <type> react)
//	samuel ls react --type framework
//	samuel ls react --limit 10
//	samuel ls react --detail --preview 20 --no-related
var lsCmd = &cobra.Command{
	Use:   "ls [query]",
	Short: "List, search, or get details on components",
	Long: `Discover Samuel components with a single verb.

Without arguments, lists installed components. With a query, searches across
languages, frameworks, workflows, and skills. Add --detail to get full info
on a specific component.

Examples:
  samuel ls                              # Installed components
  samuel ls --all                        # All available components
  samuel ls --type framework             # Filter installed by type
  samuel ls react                        # Search for "react"
  samuel ls react --detail               # Detail on react (auto-detects type)
  samuel ls react --detail --type framework --preview 20
  samuel ls react --limit 10             # Cap search results

Type values (singular):
  language, framework, workflow, skill

Type aliases (accepted for backward compatibility):
  lang/l, fw/f, wf/w, sk/s`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLs,
}

func init() {
	rootCmd.AddCommand(lsCmd)
	lsCmd.Flags().BoolP("all", "a", false, "Show all available components (replaces 'list --available')")
	lsCmd.Flags().StringP("type", "t", "", "Filter by type: language, framework, workflow, skill")
	lsCmd.Flags().Bool("detail", false, "Show detailed info on a single component (replaces 'info <type> <name>')")
	lsCmd.Flags().IntP("preview", "p", 0, "With --detail, preview N lines of the guide")
	lsCmd.Flags().Bool("no-related", false, "With --detail, omit related components")
	lsCmd.Flags().IntP("limit", "n", defaultSearchLimit, "Cap search results")
}

func runLs(cmd *cobra.Command, args []string) error {
	showAll, _ := cmd.Flags().GetBool("all")
	typeFilter, _ := cmd.Flags().GetString("type")
	detail, _ := cmd.Flags().GetBool("detail")

	// Normalize once. Aliases (lang/fw/wf/sk) → singular canonical form.
	canonicalType := normalizeTypeFilter(typeFilter)
	if typeFilter != "" && canonicalType == "" {
		return fmt.Errorf("invalid --type value: %q (valid: language, framework, workflow, skill)", typeFilter)
	}

	// --detail without a query is a user error.
	if detail && len(args) == 0 {
		return fmt.Errorf("--detail requires a component name (e.g., 'samuel ls react --detail')")
	}

	// No query: list mode.
	if len(args) == 0 {
		// listInstalled / listAvailable use the legacy plural form. Translate.
		listFilter := pluralizeTypeFilter(canonicalType)
		if JSONMode(cmd) {
			return listJSON(showAll, listFilter)
		}
		if showAll {
			return listAvailable(listFilter)
		}
		return listInstalled(listFilter)
	}

	query := args[0]

	// Detail mode: resolve to a single component.
	if detail {
		typ, name, err := resolveLsDetail(query, canonicalType)
		if err != nil {
			return err
		}
		// Hand off to runInfo with synthesized args. runInfo expects
		// (cmd, [type, name]) and reads --preview / --no-related from cmd.
		return runInfo(cmd, []string{typ, name})
	}

	// Search mode: hand off to runSearch with the query as args[0].
	// runSearch reads --type and --limit from the command's flags.
	if typeFilter != canonicalType {
		// runSearch normalizes again; ensure it sees the canonical value.
		_ = cmd.Flags().Set("type", canonicalType)
	}
	return runSearch(cmd, []string{query})
}

// resolveLsDetail returns the (type, name) tuple for `samuel ls <name> --detail`.
// If the user passed --type, that wins. Otherwise the type is inferred via
// core.InferComponentType (which scans Languages, Frameworks, and Workflows;
// Skills are excluded by design). The TestRegistry_NoCrossTypeNameCollisions
// invariant test in core ensures this can never return ambiguous matches today.
func resolveLsDetail(name, typeFilter string) (string, string, error) {
	lower := strings.ToLower(name)

	if typeFilter != "" {
		if typeFilter == "skill" {
			return "", "", fmt.Errorf("--detail does not support type 'skill' yet (use 'samuel skill info %s')", lower)
		}
		// Verify the component exists under the requested type.
		if findComponent(typeFilter, lower) == nil {
			return "", "", fmt.Errorf("component not found: %s %q", typeFilter, lower)
		}
		return typeFilter, lower, nil
	}

	typ, _, err := core.InferComponentType(lower)
	if err != nil {
		// core's "not found" message reads naturally; ambiguous adds a hint
		// pointing the user at --type since ls's flag is the disambiguator.
		return "", "", fmt.Errorf("%w. Try 'samuel ls %s' to search.", err, lower)
	}
	return typ, lower, nil
}

// pluralizeTypeFilter converts the canonical singular form used by ls/search
// into the plural form expected by listInstalled / listAvailable. Empty stays
// empty (meaning "all types").
func pluralizeTypeFilter(canonical string) string {
	switch canonical {
	case "language":
		return "languages"
	case "framework":
		return "frameworks"
	case "workflow":
		return "workflows"
	case "skill":
		// listInstalled / listAvailable do not surface skills today. Return a
		// sentinel that produces an empty filter; ls will short-circuit before
		// reaching them when --type=skill is requested without --detail.
		return "skills"
	}
	return ""
}

