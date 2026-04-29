package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

// infoCmd is preserved as a Hidden+Deprecated alias for v3.0.0.
// Use 'samuel ls <name> --detail [--type <type>]'.
var infoCmd = &cobra.Command{
	Use:    "info <type> <name>",
	Short:  "[DEPRECATED] Use 'samuel ls <name> --detail'",
	Hidden: true,
	Args:   cobra.ExactArgs(2),
	RunE:   redirectAndRun("samuel ls <name> --detail [--type <type>]", runInfo),
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().IntP("preview", "p", 0, "Number of lines to preview from the guide")
	infoCmd.Flags().Bool("no-related", false, "Skip showing related components")
}

func runInfo(cmd *cobra.Command, args []string) error {
	componentType := normalizeTypeFilter(args[0])
	componentName := strings.ToLower(args[1])
	previewLines, _ := cmd.Flags().GetInt("preview")
	noRelated, _ := cmd.Flags().GetBool("no-related")

	if componentType == "" {
		if JSONMode(cmd) {
			PrintJSONErrorForCmd(cmd, fmt.Errorf("invalid component type: %s", args[0]))
		} else {
			ui.Error("Invalid component type: %s", args[0])
			ui.Info("Valid types: language (lang, l), framework (fw, f), workflow (wf, w)")
		}
		return fmt.Errorf("invalid component type")
	}

	component := findComponent(componentType, componentName)
	if component == nil {
		if JSONMode(cmd) {
			PrintJSONErrorForCmd(cmd, fmt.Errorf("component not found: %s %s", componentType, componentName))
		} else {
			ui.Error("Component not found: %s %s", componentType, componentName)
			ui.Info("Use 'samuel search %s' to find available components", componentName)
		}
		return fmt.Errorf("component not found")
	}

	config, configErr := core.LoadConfig()
	if configErr != nil && !os.IsNotExist(configErr) && !JSONMode(cmd) {
		ui.Warn("Could not load config: %v", configErr)
	}
	installed := checkInstallStatus(config, componentType, componentName)

	if JSONMode(cmd) {
		type relatedJSON struct {
			Name        string `json:"name"`
			Type        string `json:"type"`
			Description string `json:"description"`
			Installed   bool   `json:"installed"`
		}
		data := map[string]interface{}{
			"name":        component.Name,
			"type":        componentType,
			"description": component.Description,
			"path":        component.Path,
			"installed":   installed,
		}
		if installed {
			if info, err := os.Stat(component.Path); err == nil {
				data["fileSize"] = info.Size()
				data["modified"] = info.ModTime().Format(time.RFC3339)
			}
		}
		if !noRelated {
			related := getRelatedComponents(component, componentType)
			relItems := make([]relatedJSON, 0, len(related))
			for _, r := range related {
				relItems = append(relItems, relatedJSON{
					Name:        r.Name,
					Type:        r.Type,
					Description: r.Description,
					Installed:   config != nil && isInstalled(config, r.Type, r.Name),
				})
			}
			data["related"] = relItems
		}
		PrintJSONForCmd(cmd, data)
		return nil
	}

	displayComponentInfo(component, componentType, installed)
	displayRelatedComponents(config, component, componentType, noRelated)
	displayPreview(component.Path, previewLines, installed)

	return nil
}

func findComponent(componentType, name string) *core.Component {
	switch componentType {
	case "language":
		return core.FindLanguage(name)
	case "framework":
		return core.FindFramework(name)
	case "workflow":
		return core.FindWorkflow(name)
	}
	return nil
}

func checkInstallStatus(config *core.Config, componentType, name string) bool {
	if config == nil {
		return false
	}
	switch componentType {
	case "language":
		return config.HasLanguage(name)
	case "framework":
		return config.HasFramework(name)
	case "workflow":
		return config.HasWorkflow(name)
	}
	return false
}

func displayComponentInfo(component *core.Component, componentType string, installed bool) {
	ui.Bold("Component: %s", component.Name)
	fmt.Println()

	ui.Print("  %-16s %s", "Type:", componentType)
	ui.Print("  %-16s %s", "Description:", component.Description)
	fmt.Println()

	if installed {
		ui.Print("  %-16s %s", "Status:", ui.SuccessSymbol+" Installed")
		if info, err := os.Stat(component.Path); err == nil {
			ui.Print("  %-16s %s", "File Path:", component.Path)
			ui.Print("  %-16s %s", "File Size:", formatFileSize(info.Size()))
			ui.Print("  %-16s %s", "Modified:", info.ModTime().Format(time.RFC3339))
		}
	} else {
		ui.Print("  %-16s %s", "Status:", "Not installed")
		ui.Print("  %-16s %s", "Install Path:", component.Path)
	}
}

func displayRelatedComponents(config *core.Config, component *core.Component, componentType string, skip bool) {
	if skip {
		return
	}
	related := getRelatedComponents(component, componentType)
	if len(related) == 0 {
		return
	}
	fmt.Println()
	ui.Section("Related Components")
	for _, r := range related {
		if config != nil && isInstalled(config, r.Type, r.Name) {
			ui.SuccessItem(1, "%s - %s (%s, installed)", r.Name, r.Description, r.Type)
		} else {
			ui.ListItem(1, "%s - %s (%s)", r.Name, r.Description, r.Type)
		}
	}
}

func displayPreview(filePath string, lines int, installed bool) {
	if lines <= 0 || !installed {
		return
	}
	preview, err := getFilePreview(filePath, lines)
	if err == nil && preview != "" {
		fmt.Println()
		ui.Section("Preview")
		fmt.Println(preview)
	}
}

// RelatedComponent represents a related component
type RelatedComponent struct {
	Name        string
	Type        string
	Description string
}

// getRelatedComponents finds related components
func getRelatedComponents(component *core.Component, componentType string) []RelatedComponent {
	var related []RelatedComponent

	switch componentType {
	case "language":
		// Find frameworks related to this language
		related = getFrameworksForLanguage(component.Name)
	case "framework":
		// Find the language this framework is built on
		related = getLanguageForFramework(component.Name)
	case "workflow":
		// Workflows can relate to multiple things - skip for now
	}

	return related
}

// getFrameworksForLanguage returns frameworks related to a language
func getFrameworksForLanguage(langName string) []RelatedComponent {
	var related []RelatedComponent

	// Map languages to their frameworks
	langFrameworks := map[string][]string{
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

	if fws, ok := langFrameworks[langName]; ok {
		for _, fwName := range fws {
			if fw := core.FindFramework(fwName); fw != nil {
				related = append(related, RelatedComponent{
					Name:        fw.Name,
					Type:        "framework",
					Description: fw.Description,
				})
			}
		}
	}

	return related
}

// getLanguageForFramework returns the language a framework is built on
func getLanguageForFramework(fwName string) []RelatedComponent {
	var related []RelatedComponent

	// Map frameworks to their languages
	frameworkLang := map[string]string{
		"react": "typescript", "nextjs": "typescript", "express": "typescript",
		"django": "python", "fastapi": "python", "flask": "python",
		"gin": "go", "echo": "go", "fiber": "go",
		"axum": "rust", "actix-web": "rust", "rocket": "rust",
		"spring-boot-kotlin": "kotlin", "ktor": "kotlin", "android-compose": "kotlin",
		"spring-boot-java": "java", "quarkus": "java", "micronaut": "java",
		"aspnet-core": "csharp", "blazor": "csharp", "unity": "csharp",
		"laravel": "php", "symfony": "php", "wordpress": "php",
		"swiftui": "swift", "uikit": "swift", "vapor": "swift",
		"rails": "ruby", "sinatra": "ruby", "hanami": "ruby",
		"flutter": "dart", "shelf": "dart", "dart-frog": "dart",
	}

	if langName, ok := frameworkLang[fwName]; ok {
		if lang := core.FindLanguage(langName); lang != nil {
			related = append(related, RelatedComponent{
				Name:        lang.Name,
				Type:        "language",
				Description: lang.Description,
			})
		}
	}

	return related
}

func isInstalled(config *core.Config, componentType, name string) bool {
	switch componentType {
	case "language":
		return config.HasLanguage(name)
	case "framework":
		return config.HasFramework(name)
	case "workflow":
		return config.HasWorkflow(name)
	}
	return false
}

func getFilePreview(filePath string, lines int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var result strings.Builder
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() && lineNum < lines {
		lineNum++
		result.WriteString(fmt.Sprintf("  %3d | %s\n", lineNum, scanner.Text()))
	}

	return result.String(), scanner.Err()
}

func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
	)

	switch {
	case size >= MB:
		return fmt.Sprintf("%.1f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.1f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}
