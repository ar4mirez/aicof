package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

// initFlags holds parsed command-line flags for the init command.
type initFlags struct {
	force          bool
	nonInteractive bool
	templateName   string
	languageFlags  []string
	frameworkFlags []string
	cliProvided    bool
	absTargetDir   string
	createDir      bool
	jsonMode       bool
}

// initSelections holds the user's component selections.
type initSelections struct {
	template   *core.Template
	languages  []string
	frameworks []string
}

// parseInitFlags extracts CLI flags and resolves the target directory.
func parseInitFlags(cmd *cobra.Command, args []string) (*initFlags, error) {
	flags := &initFlags{}
	flags.force, _ = cmd.Flags().GetBool("force")
	flags.nonInteractive, _ = cmd.Flags().GetBool("non-interactive")
	flags.templateName, _ = cmd.Flags().GetString("template")
	flags.languageFlags, _ = cmd.Flags().GetStringSlice("languages")
	flags.frameworkFlags, _ = cmd.Flags().GetStringSlice("frameworks")
	flags.jsonMode = JSONMode(cmd)
	flags.cliProvided = flags.templateName != "" || len(flags.languageFlags) > 0 || len(flags.frameworkFlags) > 0

	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	flags.absTargetDir = absTargetDir

	if targetDir != "." {
		if _, err := os.Stat(absTargetDir); os.IsNotExist(err) {
			flags.createDir = true
		}
	}

	return flags, nil
}

// validateInitTarget checks that the target directory is valid for initialization.
func validateInitTarget(flags *initFlags) error {
	if isSamuelRepository(flags.absTargetDir) {
		return fmt.Errorf("cannot initialize inside the Samuel repository itself.\nUse 'samuel init <project-name>' to create a new project directory")
	}
	if core.ConfigExists(flags.absTargetDir) && !flags.force {
		return fmt.Errorf("Samuel already initialized in %s. Use --force to reinitialize", flags.absTargetDir)
	}
	return nil
}

// selectTemplateInteractive prompts the user to choose a template.
func selectTemplateInteractive() (string, error) {
	templateOptions := make([]ui.SelectOption, len(core.Templates))
	for i, t := range core.Templates {
		templateOptions[i] = ui.SelectOption{
			Name: t.Name, Description: t.Description, Value: t.Name,
		}
	}
	selected, err := ui.Select("Select template", templateOptions)
	if err != nil {
		return "", fmt.Errorf("template selection cancelled: %w", err)
	}
	return selected.Value, nil
}

// selectComponents orchestrates template, language, and framework selection.
func selectComponents(flags *initFlags) (*initSelections, error) {
	sel := &initSelections{}
	templateName := flags.templateName
	if !flags.nonInteractive && templateName == "" && len(flags.languageFlags) == 0 {
		name, err := selectTemplateInteractive()
		if err != nil {
			return nil, err
		}
		templateName = name
	}
	if templateName != "" {
		sel.template = core.FindTemplate(templateName)
		if sel.template == nil {
			return nil, fmt.Errorf("unknown template: %s", templateName)
		}
		sel.languages = sel.template.Languages
		sel.frameworks = sel.template.Frameworks
	}
	// Override with CLI flags
	if len(flags.languageFlags) > 0 {
		sel.languages = expandLanguages(flags.languageFlags)
	}
	if len(flags.frameworkFlags) > 0 {
		sel.frameworks = expandFrameworks(flags.frameworkFlags)
	}
	// Interactive language selection
	if !flags.nonInteractive && !flags.cliProvided && sel.template != nil && sel.template.Name != "full" {
		langs, err := selectLanguagesInteractive(sel.languages)
		if err != nil {
			return nil, err
		}
		sel.languages = langs
	}
	// Interactive framework selection
	if !flags.nonInteractive && !flags.cliProvided && sel.template != nil && sel.template.Name != "full" && len(sel.languages) > 0 {
		sel.frameworks = selectFrameworksInteractive(sel.languages)
	}
	// Default to starter template if nothing selected
	if sel.template == nil && len(sel.languages) == 0 {
		sel.template = core.FindTemplate("starter")
		sel.languages = sel.template.Languages
		sel.frameworks = sel.template.Frameworks
	}
	return sel, nil
}

// selectLanguagesInteractive presents a multi-select prompt for languages.
func selectLanguagesInteractive(defaults []string) ([]string, error) {
	langOptions := make([]ui.SelectOption, len(core.Languages))
	for i, l := range core.Languages {
		langOptions[i] = ui.SelectOption{
			Name: l.Name, Description: l.Description, Value: l.Name,
		}
	}

	selected, err := ui.MultiSelect("Select languages", langOptions, defaults)
	if err != nil {
		return nil, fmt.Errorf("language selection cancelled: %w", err)
	}

	result := make([]string, len(selected))
	for i, s := range selected {
		result[i] = s.Value
	}
	return result, nil
}

// selectFrameworksInteractive presents a multi-select prompt for frameworks.
func selectFrameworksInteractive(selectedLangs []string) []string {
	relevantFrameworks := getRelevantFrameworks(selectedLangs)
	if len(relevantFrameworks) == 0 {
		return []string{}
	}

	fwOptions := make([]ui.SelectOption, len(relevantFrameworks))
	for i, fw := range relevantFrameworks {
		fwOptions[i] = ui.SelectOption{
			Name: fw.Name, Description: fw.Description, Value: fw.Name,
		}
	}

	selected, err := ui.MultiSelect("Select frameworks (optional)", fwOptions, nil)
	if err != nil {
		return []string{}
	}

	result := make([]string, len(selected))
	for i, s := range selected {
		result[i] = s.Value
	}
	return result
}

// displayAndConfirm shows the installation summary and asks for confirmation.
func displayAndConfirm(flags *initFlags, sel *initSelections) bool {
	ui.Header("Samuel Initialization")
	ui.TableRow("Target", flags.absTargetDir)
	ui.TableRow("Languages", fmt.Sprintf("%d selected", len(sel.languages)))
	ui.TableRow("Frameworks", fmt.Sprintf("%d selected", len(sel.frameworks)))
	ui.TableRow("Workflows", "all (13)")

	if !flags.nonInteractive && !flags.cliProvided {
		confirmed, err := ui.Confirm("\nProceed with installation?", true)
		if err != nil || !confirmed {
			ui.Info("Installation cancelled")
			return false
		}
	}
	return true
}

// downloadFramework downloads the latest framework version from GitHub.
func downloadFramework() (version string, cachePath string, err error) {
	spinner := ui.NewSpinner("Downloading framework...")
	spinner.Start()

	downloader, err := core.NewDownloader()
	if err != nil {
		spinner.Error("Failed to initialize")
		return "", "", fmt.Errorf("failed to initialize downloader: %w", err)
	}

	version, err = downloader.GetLatestVersion()
	if err != nil {
		spinner.Error("Failed to get latest version")
		return "", "", fmt.Errorf("failed to get latest version: %w", err)
	}

	cachePath, err = downloader.DownloadVersion(version)
	if err != nil {
		spinner.Error("Download failed")
		return "", "", fmt.Errorf("failed to download framework: %w", err)
	}
	spinner.Success(fmt.Sprintf("Downloaded Samuel v%s", version))

	return version, cachePath, nil
}

// installAndSetup extracts framework files and performs post-install setup.
func installAndSetup(flags *initFlags, sel *initSelections, version, cachePath string) error {
	if flags.createDir {
		if err := os.MkdirAll(flags.absTargetDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		ui.Success("Created %s/", filepath.Base(flags.absTargetDir))
	}

	workflows := []string{"all"}
	paths := core.GetComponentPaths(sel.languages, sel.frameworks, workflows)
	extractor := core.NewExtractor(cachePath, flags.absTargetDir)
	result, err := extractor.Extract(paths, flags.force)
	if err != nil {
		return fmt.Errorf("failed to extract files: %w", err)
	}

	installedSkills := updateSkillsAndAgentsMD(flags.absTargetDir)

	syncResult, syncErr := core.SyncFolderCLAUDEMDs(core.SyncOptions{
		RootDir:  flags.absTargetDir,
		MaxDepth: 1,
	})
	if syncErr != nil {
		ui.Warn("Could not create per-folder CLAUDE.md files: %v", syncErr)
	} else if len(syncResult.Created) > 0 {
		ui.Success("Created %d per-folder CLAUDE.md/AGENTS.md files", len(syncResult.Created))
	}

	reportInitResults(result, version, sel, installedSkills)
	return nil
}

// updateSkillsAndAgentsMD updates the skills section in CLAUDE.md and copies it to AGENTS.md.
func updateSkillsAndAgentsMD(absTargetDir string) []*core.SkillInfo {
	skillsDir := filepath.Join(absTargetDir, ".claude", "skills")
	claudeMDPath := filepath.Join(absTargetDir, "CLAUDE.md")

	installedSkills, scanErr := core.ScanSkillsDirectory(skillsDir)
	if scanErr != nil {
		ui.Warn("Could not scan skills directory: %v", scanErr)
	}
	if len(installedSkills) > 0 {
		if err := core.UpdateCLAUDEMDSkillsSection(claudeMDPath, installedSkills); err != nil {
			ui.Warn("Could not update skills section in CLAUDE.md: %v", err)
		}
	}

	agentsMDPath := filepath.Join(absTargetDir, "AGENTS.md")
	if claudeContent, err := os.ReadFile(claudeMDPath); err == nil {
		if err := os.WriteFile(agentsMDPath, claudeContent, 0644); err != nil {
			ui.Warn("Could not create AGENTS.md: %v", err)
		}
	}

	return installedSkills
}

