package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize Samuel framework in a project",
	Long: `Initialize the Samuel (Artificial Intelligence Coding Framework) in a new or existing project.

This command downloads and installs framework files including:
- CLAUDE.md and AGENTS.md (core guardrails and methodology)
- Skills in .claude/skills/ (language guides, frameworks, workflows)
- Per-folder CLAUDE.md stubs for existing directories

Examples:
  samuel init my-project              # Create new project
  samuel init .                       # Initialize in current directory
  samuel init --template minimal      # Use minimal template
  samuel init --languages ts,py,go    # Select specific languages`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("template", "t", "", "Template: full, starter, minimal")
	initCmd.Flags().StringSlice("languages", nil, "Languages to install (comma-separated)")
	initCmd.Flags().StringSlice("frameworks", nil, "Frameworks to install (comma-separated)")
	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing files")
	initCmd.Flags().Bool("non-interactive", false, "Skip prompts, use defaults")
}

func runInit(cmd *cobra.Command, args []string) error {
	flags, err := parseInitFlags(cmd, args)
	if err != nil {
		return err
	}

	if err := validateInitTarget(flags); err != nil {
		return err
	}

	sel, err := selectComponents(flags)
	if err != nil {
		return err
	}

	if !displayAndConfirm(flags, sel) {
		return nil
	}

	version, cachePath, err := downloadFramework()
	if err != nil {
		return err
	}

	if err := installAndSetup(flags, sel, version, cachePath); err != nil {
		return err
	}

	return saveInitConfig(flags, sel, version)
}

// expandLanguages expands short language names.
func expandLanguages(flags []string) []string {
	aliases := map[string]string{
		"ts":         "typescript",
		"js":         "typescript",
		"javascript": "typescript",
		"py":         "python",
		"cs":         "csharp",
		"c++":        "cpp",
		"c":          "cpp",
		"rb":         "ruby",
		"sh":         "shell",
		"bash":       "shell",
	}

	var result []string
	for _, f := range flags {
		for _, name := range strings.Split(f, ",") {
			name = strings.TrimSpace(strings.ToLower(name))
			if alias, ok := aliases[name]; ok {
				name = alias
			}
			if core.FindLanguage(name) != nil {
				result = append(result, name)
			}
		}
	}
	return result
}

// expandFrameworks expands short framework names.
func expandFrameworks(flags []string) []string {
	aliases := map[string]string{
		"next":   "nextjs",
		"spring": "spring-boot-java",
	}

	var result []string
	for _, f := range flags {
		for _, name := range strings.Split(f, ",") {
			name = strings.TrimSpace(strings.ToLower(name))
			if alias, ok := aliases[name]; ok {
				name = alias
			}
			if core.FindFramework(name) != nil {
				result = append(result, name)
			}
		}
	}
	return result
}

// isSamuelRepository checks if the target directory is the Samuel repository itself.
// This prevents users from accidentally initializing inside the framework source.
func isSamuelRepository(targetDir string) bool {
	templateDir := filepath.Join(targetDir, "template")
	if info, err := os.Stat(templateDir); err == nil && info.IsDir() {
		claudeMD := filepath.Join(templateDir, "CLAUDE.md")
		if _, err := os.Stat(claudeMD); err == nil {
			return true
		}
	}

	cliDir := filepath.Join(targetDir, "packages", "cli")
	if info, err := os.Stat(cliDir); err == nil && info.IsDir() {
		goMod := filepath.Join(cliDir, "go.mod")
		if _, err := os.Stat(goMod); err == nil {
			return true
		}
	}

	return false
}

// getRelevantFrameworks returns frameworks related to selected languages.
func getRelevantFrameworks(languages []string) []core.Component {
	languageFrameworks := map[string][]string{
		"typescript": {"react", "nextjs", "express"},
		"python":     {"django", "fastapi", "flask"},
		"go":         {"gin", "echo", "fiber"},
		"rust":       {"axum", "actix-web", "rocket"},
		"kotlin":     {"spring-boot-kotlin", "ktor", "android-compose"},
		"java":       {"spring-boot-java", "quarkus", "micronaut"},
		"csharp":     {"aspnet-core", "blazor", "unity"},
		"php":        {"laravel", "symfony", "wordpress"},
		"swift":      {"swiftui", "uikit", "vapor"},
		"ruby":       {"rails", "sinatra", "hanami"},
		"dart":       {"flutter", "shelf", "dart-frog"},
	}

	var result []core.Component
	seen := make(map[string]bool)

	for _, lang := range languages {
		if fws, ok := languageFrameworks[lang]; ok {
			for _, fwName := range fws {
				if !seen[fwName] {
					if fw := core.FindFramework(fwName); fw != nil {
						result = append(result, *fw)
						seen[fwName] = true
					}
				}
			}
		}
	}

	return result
}

// reportInitResults displays the installation summary to the user.
func reportInitResults(result *core.ExtractResult, version string, sel *initSelections, installedSkills []*core.SkillInfo) {
	ui.Success("Installed CLAUDE.md (v%s)", version)
	ui.Success("Installed AGENTS.md (cross-tool compatibility)")
	ui.Success("Installed %d language guides", len(sel.languages))
	ui.Success("Installed %d framework guides", len(sel.frameworks))
	ui.Success("Installed %d workflows", len(core.Workflows))
	if len(installedSkills) > 0 {
		ui.Success("Installed %d skills", len(installedSkills))
	}
	if len(result.FilesSkipped) > 0 {
		ui.Warn("Skipped %d existing files (use --force to overwrite)", len(result.FilesSkipped))
	}
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			ui.Error("%v", e)
		}
	}
}

// saveInitConfig creates and saves the samuel.yaml config file and shows next steps.
func saveInitConfig(flags *initFlags, sel *initSelections, version string) error {
	config := core.NewConfig(version)
	config.Installed.Languages = sel.languages
	config.Installed.Frameworks = sel.frameworks
	config.Installed.Workflows = []string{"all"}

	if err := config.Save(flags.absTargetDir); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	if flags.jsonMode {
		ui.PrintJSON("init", map[string]interface{}{
			"version":    version,
			"path":       flags.absTargetDir,
			"languages":  sel.languages,
			"frameworks": sel.frameworks,
			"workflows":  []string{"all"},
		})
		return nil
	}

	ui.Success("Created samuel.yaml")

	fmt.Println()
	ui.Bold("Next steps:")
	if flags.createDir {
		ui.ListItem(1, "cd %s", filepath.Base(flags.absTargetDir))
	}
	ui.ListItem(1, "Start coding with AI assistance!")
	ui.ListItem(1, "Run 'samuel doctor' to verify installation")

	return nil
}
