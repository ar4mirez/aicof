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

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage Agent Skills",
	Long: `Manage Agent Skills - capability modules that give AI agents new abilities.

Skills follow the Agent Skills open standard (https://agentskills.io) and
are supported by 25+ agent products including Claude Code, Cursor, and VS Code.

Subcommands:
  create    Create a new skill scaffold
  validate  Validate skill(s) against the specification
  list      List installed skills
  info      Show detailed information about a skill

Examples:
  samuel skill create database-ops     # Create a new skill
  samuel skill validate                # Validate all skills
  samuel skill list                    # List installed skills`,
}

var skillCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new skill scaffold",
	Long: `Create a new skill scaffold with SKILL.md template and optional directories.

The skill name must:
  - Be lowercase alphanumeric with hyphens only
  - Not start or end with a hyphen
  - Not contain consecutive hyphens
  - Be max 64 characters

Examples:
  samuel skill create database-ops
  samuel skill create my-custom-skill`,
	Args: cobra.ExactArgs(1),
	RunE: runSkillCreate,
}

var skillValidateCmd = &cobra.Command{
	Use:   "validate [name]",
	Short: "Validate skill(s) against the Agent Skills specification",
	Long: `Validate skill(s) against the Agent Skills specification.

If no name is provided, validates all skills in .claude/skills/

Checks:
  - SKILL.md exists with valid YAML frontmatter
  - Name matches directory name
  - Name format (lowercase, hyphens, max 64 chars)
  - Description present (max 1024 chars)
  - Compatibility field (max 500 chars if present)

Examples:
  samuel skill validate                # Validate all skills
  samuel skill validate database-ops   # Validate specific skill`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSkillValidate,
}

var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills",
	Long: `List all skills installed in .claude/skills/

Shows skill name, description, and validation status.

Examples:
  samuel skill list`,
	RunE: runSkillList,
}

var skillInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show detailed information about a skill",
	Long: `Show detailed information about an installed skill.

Displays:
  - Name and description
  - License and compatibility
  - Optional directories (scripts, references, assets)
  - Validation status
  - Line count and estimated tokens

Examples:
  samuel skill info database-ops`,
	Args: cobra.ExactArgs(1),
	RunE: runSkillInfo,
}

func init() {
	rootCmd.AddCommand(skillCmd)
	skillCmd.AddCommand(skillCreateCmd)
	skillCmd.AddCommand(skillValidateCmd)
	skillCmd.AddCommand(skillListCmd)
	skillCmd.AddCommand(skillInfoCmd)
}

func runSkillCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate name first
	if errors := core.ValidateSkillName(name); len(errors) > 0 {
		for _, e := range errors {
			ui.Error("Invalid name: %s", e)
		}
		return fmt.Errorf("skill name validation failed")
	}

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check if Samuel is initialized
	if !core.ConfigExists(cwd) {
		return fmt.Errorf("no Samuel installation found. Run 'samuel init' first")
	}

	// Skills directory
	skillsDir := filepath.Join(cwd, ".claude", "skills")

	// Create skills directory if it doesn't exist
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	// Create skill scaffold
	if !JSONMode(cmd) {
		ui.Info("Creating skill '%s'...", name)
	}

	if err := core.CreateSkillScaffold(skillsDir, name); err != nil {
		return fmt.Errorf("failed to create skill: %w", err)
	}

	skillPath := filepath.Join(skillsDir, name)

	if JSONMode(cmd) {
		ui.PrintJSON("skill create", map[string]interface{}{
			"name": name,
			"path": skillPath,
			"files": []string{
				name + "/SKILL.md",
				name + "/scripts/.gitkeep",
				name + "/references/.gitkeep",
				name + "/assets/.gitkeep",
			},
		})
		return nil
	}

	ui.Success("Created skill scaffold at %s/", skillPath)
	ui.Print("")
	ui.Print("  Files created:")
	ui.Print("    %s/SKILL.md", name)
	ui.Print("    %s/scripts/.gitkeep", name)
	ui.Print("    %s/references/.gitkeep", name)
	ui.Print("    %s/assets/.gitkeep", name)
	ui.Print("")
	ui.Info("Edit .claude/skills/%s/SKILL.md to define your skill", name)

	return nil
}

func runSkillValidate(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	skillsDir := filepath.Join(cwd, ".claude", "skills")

	// Check if skills directory exists
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		ui.Info("No skills directory found at .claude/skills/")
		return nil
	}

	var skills []*core.SkillInfo

	if len(args) == 1 {
		// Validate specific skill
		skillPath := filepath.Join(skillsDir, args[0])
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			return fmt.Errorf("skill '%s' not found", args[0])
		}

		info, err := core.LoadSkillInfo(skillPath)
		if err != nil {
			return fmt.Errorf("failed to load skill: %w", err)
		}
		skills = append(skills, info)
	} else {
		// Validate all skills
		var err error
		skills, err = core.ScanSkillsDirectory(skillsDir)
		if err != nil {
			return fmt.Errorf("failed to scan skills: %w", err)
		}
	}

	if len(skills) == 0 {
		ui.Info("No skills found in .claude/skills/")
		return nil
	}

	validCount := 0
	invalidCount := 0

	if JSONMode(cmd) {
		type validateJSON struct {
			Name   string   `json:"name"`
			Valid  bool     `json:"valid"`
			Errors []string `json:"errors,omitempty"`
		}
		items := make([]validateJSON, 0, len(skills))
		for _, skill := range skills {
			valid := len(skill.Errors) == 0
			if valid {
				validCount++
			} else {
				invalidCount++
			}
			items = append(items, validateJSON{
				Name:   skill.DirName,
				Valid:  valid,
				Errors: skill.Errors,
			})
		}
		ui.PrintJSON("skill validate", map[string]interface{}{
			"valid":   validCount,
			"invalid": invalidCount,
			"skills":  items,
		})
		if invalidCount > 0 {
			return fmt.Errorf("%d skill(s) failed validation", invalidCount)
		}
		return nil
	}

	for _, skill := range skills {
		if len(skill.Errors) == 0 {
			validCount++
			ui.SuccessItem(0, "%s: valid", skill.DirName)
		} else {
			invalidCount++
			ui.ErrorItem(0, "%s: invalid", skill.DirName)
			for _, e := range skill.Errors {
				ui.ErrorItem(1, "%s", e)
			}
		}
	}

	ui.Print("")
	if invalidCount > 0 {
		ui.Warn("Validated %d skills: %d valid, %d invalid", len(skills), validCount, invalidCount)
		return fmt.Errorf("%d skill(s) failed validation", invalidCount)
	}

	ui.Success("All %d skills are valid", validCount)
	return nil
}

func runSkillList(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	skillsDir := filepath.Join(cwd, ".claude", "skills")

	// Check if skills directory exists
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		ui.Info("No skills directory found at .claude/skills/")
		ui.Print("Run 'samuel skill create <name>' to create your first skill")
		return nil
	}

	skills, err := core.ScanSkillsDirectory(skillsDir)
	if err != nil {
		return fmt.Errorf("failed to scan skills: %w", err)
	}

	if JSONMode(cmd) {
		type skillJSON struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Valid       bool     `json:"valid"`
			Errors      []string `json:"errors,omitempty"`
		}
		items := make([]skillJSON, 0, len(skills))
		for _, skill := range skills {
			desc := strings.ReplaceAll(skill.Metadata.Description, "\n", " ")
			desc = strings.TrimSpace(desc)
			items = append(items, skillJSON{
				Name:        skill.Metadata.Name,
				Description: desc,
				Valid:       len(skill.Errors) == 0,
				Errors:      skill.Errors,
			})
		}
		ui.PrintJSON("skill list", map[string]interface{}{
			"total":  len(items),
			"skills": items,
		})
		return nil
	}

	if len(skills) == 0 {
		ui.Info("No skills found in .claude/skills/")
		ui.Print("Run 'samuel skill create <name>' to create your first skill")
		return nil
	}

	ui.Header("Installed Skills")

	for _, skill := range skills {
		// Truncate description for display
		desc := skill.Metadata.Description
		desc = strings.ReplaceAll(desc, "\n", " ")
		desc = strings.TrimSpace(desc)
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}

		if len(skill.Errors) > 0 {
			ui.ErrorItem(0, "%s (invalid)", skill.DirName)
			ui.Dim("     %s", desc)
		} else {
			ui.SuccessItem(0, "%s", skill.Metadata.Name)
			ui.Dim("     %s", desc)
		}
	}

	ui.Print("")
	ui.Print("Total: %d skill(s)", len(skills))
	ui.Print("")
	ui.Info("Run 'samuel skill info <name>' for details")

	return nil
}

func runSkillInfo(cmd *cobra.Command, args []string) error {
	name := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	skillPath := filepath.Join(cwd, ".claude", "skills", name)

	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		return fmt.Errorf("skill '%s' not found", name)
	}

	info, err := core.LoadSkillInfo(skillPath)
	if err != nil {
		return fmt.Errorf("failed to load skill: %w", err)
	}

	if JSONMode(cmd) {
		data := map[string]interface{}{
			"name":          info.Metadata.Name,
			"dirName":       info.DirName,
			"description":   strings.TrimSpace(info.Metadata.Description),
			"path":          info.Path,
			"license":       info.Metadata.License,
			"compatibility": info.Metadata.Compatibility,
			"hasScripts":    info.HasScripts,
			"hasReferences": info.HasRefs,
			"hasAssets":     info.HasAssets,
			"valid":         len(info.Errors) == 0,
			"errors":        info.Errors,
		}
		if info.Body != "" {
			data["bodyLines"] = core.CountLines(info.Body)
		}
		if len(info.Metadata.Metadata) > 0 {
			data["metadata"] = info.Metadata.Metadata
		}
		ui.PrintJSON("skill info", data)
		return nil
	}

	ui.Header(fmt.Sprintf("Skill: %s", info.DirName))
	displaySkillMetadata(info)
	displaySkillStructure(info)

	// Validation
	ui.Section("Validation")
	if len(info.Errors) == 0 {
		ui.SuccessItem(0, "Valid (passes Agent Skills specification)")
	} else {
		ui.ErrorItem(0, "Invalid")
		for _, e := range info.Errors {
			ui.ErrorItem(1, "%s", e)
		}
	}

	ui.Print("")
	return nil
}
