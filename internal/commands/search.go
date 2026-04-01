package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

// defaultSearchLimit is the maximum number of search results returned by default.
const defaultSearchLimit = 20

// SearchResult represents a search match
type SearchResult struct {
	Name        string
	Type        string
	Description string
	Score       int
	Installed   bool
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for components by keyword",
	Long: `Search for Samuel components across languages, frameworks, workflows, and skills.

Supports fuzzy matching for typos and partial matches. Results are sorted by relevance.

Examples:
  samuel search react              # Search all component types
  samuel search --type fw django   # Search only frameworks
  samuel search py                 # Fuzzy match finds "python"
  samuel search "spring boot"      # Multi-word search
  samuel search commit             # Finds commit-message skill

Types (with aliases):
  language   (lang, l)   Language guides
  framework  (fw, f)     Framework guides
  workflow   (wf, w)     Workflow templates
  skill      (sk, s)     Agent Skills`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringP("type", "t", "", "Filter by type: language/lang/l, framework/fw/f, workflow/wf/w, skill/sk/s")
	searchCmd.Flags().IntP("limit", "n", defaultSearchLimit, "Limit number of results")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.ToLower(args[0])
	typeFilter, _ := cmd.Flags().GetString("type")
	limit, _ := cmd.Flags().GetInt("limit")

	typeFilter = normalizeTypeFilter(typeFilter)
	config, configErr := core.LoadConfig()
	if configErr != nil && !os.IsNotExist(configErr) {
		ui.Warn("Could not load config: %v", configErr)
	}
	results := searchComponents(query, typeFilter, config)
	results = sortAndLimitResults(results, limit)

	if JSONMode(cmd) {
		type resultJSON struct {
			Name        string `json:"name"`
			Type        string `json:"type"`
			Description string `json:"description"`
			Score       int    `json:"score"`
			Installed   bool   `json:"installed"`
		}
		items := make([]resultJSON, 0, len(results))
		for _, r := range results {
			items = append(items, resultJSON(r))
		}
		ui.PrintJSON("search", map[string]interface{}{
			"query":   query,
			"results": items,
			"total":   len(items),
		})
		return nil
	}

	if len(results) == 0 {
		ui.Warn("No components found matching '%s'", query)
		ui.Info("Try a different search term or use 'samuel list --available' to see all components")
		return nil
	}

	displaySearchResults(query, results)
	return nil
}

func sortAndLimitResults(results []SearchResult, limit int) []SearchResult {
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		return results[:limit]
	}
	return results
}

func displaySearchResults(query string, results []SearchResult) {
	ui.Bold("Search results for '%s'", query)
	fmt.Println()

	languages := filterByType(results, "language")
	frameworks := filterByType(results, "framework")
	workflows := filterByType(results, "workflow")
	skills := filterByType(results, "skill")

	if len(languages) > 0 {
		ui.Section(fmt.Sprintf("Languages (%d)", len(languages)))
		displayResults(languages)
	}
	if len(frameworks) > 0 {
		ui.Section(fmt.Sprintf("Frameworks (%d)", len(frameworks)))
		displayResults(frameworks)
	}
	if len(workflows) > 0 {
		ui.Section(fmt.Sprintf("Workflows (%d)", len(workflows)))
		displayResults(workflows)
	}
	if len(skills) > 0 {
		ui.Section(fmt.Sprintf("Skills (%d)", len(skills)))
		displayResults(skills)
	}

	fmt.Println()
	ui.Dim("%d result(s) found", len(results))
}

func searchComponents(query, typeFilter string, config *core.Config) []SearchResult {
	var results []SearchResult

	if typeFilter == "" || typeFilter == "language" {
		results = append(results, searchLanguages(query, config)...)
	}
	if typeFilter == "" || typeFilter == "framework" {
		results = append(results, searchFrameworks(query, config)...)
	}
	if typeFilter == "" || typeFilter == "workflow" {
		results = append(results, searchWorkflows(query, config)...)
	}
	if typeFilter == "" || typeFilter == "skill" {
		results = append(results, searchSkills(query, config)...)
	}

	return results
}

func searchLanguages(query string, config *core.Config) []SearchResult {
	var results []SearchResult
	for _, lang := range core.Languages {
		score := matchScore(query, lang.Name, lang.Description)
		// Also check tags for matches
		if score == 0 {
			for _, tag := range lang.Tags {
				if tagScore := matchScore(query, tag, ""); tagScore > 0 {
					score = tagScore
					break
				}
			}
		}
		if score > 0 {
			results = append(results, SearchResult{
				Name:        lang.Name,
				Type:        "language",
				Description: lang.Description,
				Score:       score,
				Installed:   config != nil && config.HasLanguage(lang.Name),
			})
		}
	}
	return results
}

func searchFrameworks(query string, config *core.Config) []SearchResult {
	var results []SearchResult
	for _, fw := range core.Frameworks {
		score := matchScore(query, fw.Name, fw.Description)
		// Also check tags for matches
		if score == 0 {
			for _, tag := range fw.Tags {
				if tagScore := matchScore(query, tag, ""); tagScore > 0 {
					score = tagScore
					break
				}
			}
		}
		if score > 0 {
			results = append(results, SearchResult{
				Name:        fw.Name,
				Type:        "framework",
				Description: fw.Description,
				Score:       score,
				Installed:   config != nil && config.HasFramework(fw.Name),
			})
		}
	}
	return results
}

func searchWorkflows(query string, config *core.Config) []SearchResult {
	var results []SearchResult
	for _, wf := range core.Workflows {
		score := matchScore(query, wf.Name, wf.Description)
		// Also check tags for matches
		if score == 0 {
			for _, tag := range wf.Tags {
				if tagScore := matchScore(query, tag, ""); tagScore > 0 {
					score = tagScore
					break
				}
			}
		}
		if score > 0 {
			results = append(results, SearchResult{
				Name:        wf.Name,
				Type:        "workflow",
				Description: wf.Description,
				Score:       score,
				Installed:   config != nil && config.HasWorkflow(wf.Name),
			})
		}
	}
	return results
}

func searchSkills(query string, config *core.Config) []SearchResult {
	var results []SearchResult
	for _, skill := range core.Skills {
		if score := matchScore(query, skill.Name, skill.Description); score > 0 {
			results = append(results, SearchResult{
				Name:        skill.Name,
				Type:        "skill",
				Description: skill.Description,
				Score:       score,
				Installed:   config != nil && config.HasSkill(skill.Name),
			})
		}
	}
	return results
}

// matchScore returns a score indicating match quality (0 = no match)
func matchScore(query, name, description string) int {
	queryLower := strings.ToLower(query)
	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(description)

	score := 0

	// Exact match on name (highest score)
	if nameLower == queryLower {
		return 100
	}

	// Prefix match on name
	if strings.HasPrefix(nameLower, queryLower) {
		score = 80
	}

	// Contains match on name
	if score == 0 && strings.Contains(nameLower, queryLower) {
		score = 60
	}

	// Contains match on description
	if score == 0 && strings.Contains(descLower, queryLower) {
		score = 40
	}

	// Fuzzy match on name (Levenshtein distance)
	if score == 0 {
		dist := levenshteinDistance(queryLower, nameLower)
		if dist <= 2 && dist < len(nameLower)/2 {
			score = 30 - dist*5
		}
	}

	return score
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func normalizeTypeFilter(filter string) string {
	switch strings.ToLower(filter) {
	case "language", "lang", "l":
		return "language"
	case "framework", "fw", "f":
		return "framework"
	case "workflow", "wf", "w":
		return "workflow"
	case "skill", "sk", "s":
		return "skill"
	default:
		return ""
	}
}

func filterByType(results []SearchResult, componentType string) []SearchResult {
	var filtered []SearchResult
	for _, r := range results {
		if r.Type == componentType {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func displayResults(results []SearchResult) {
	for _, r := range results {
		if r.Installed {
			ui.SuccessItem(1, "%s - %s (installed)", r.Name, r.Description)
		} else {
			ui.ListItem(1, "%s - %s", r.Name, r.Description)
		}
	}
}
