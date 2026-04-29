package commands

import (
	"os"
	"strings"
	"testing"

	"github.com/ar4mirez/samuel/internal/core"
)

func TestResolveComponent(t *testing.T) {
	config := core.NewConfig("1.0.0")
	config.Installed.Languages = []string{"go"}
	config.Installed.Frameworks = []string{"react"}
	config.Installed.Workflows = []string{"create-prd"}

	t.Run("language_types", func(t *testing.T) {
		aliases := []string{"language", "lang", "l"}
		for _, alias := range aliases {
			t.Run(alias, func(t *testing.T) {
				comp, alreadyInstalled, err := resolveComponent(alias, "rust", config)
				if err != nil {
					t.Fatalf("resolveComponent(%q, %q) error = %v", alias, "rust", err)
				}
				if comp == nil {
					t.Fatal("expected non-nil component")
				}
				if comp.Name != "rust" {
					t.Errorf("component.Name = %q, want %q", comp.Name, "rust")
				}
				if alreadyInstalled {
					t.Error("rust should not be already installed")
				}
			})
		}
	})

	t.Run("framework_types", func(t *testing.T) {
		aliases := []string{"framework", "fw", "f"}
		for _, alias := range aliases {
			t.Run(alias, func(t *testing.T) {
				comp, alreadyInstalled, err := resolveComponent(alias, "django", config)
				if err != nil {
					t.Fatalf("resolveComponent(%q, %q) error = %v", alias, "django", err)
				}
				if comp == nil {
					t.Fatal("expected non-nil component")
				}
				if comp.Name != "django" {
					t.Errorf("component.Name = %q, want %q", comp.Name, "django")
				}
				if alreadyInstalled {
					t.Error("django should not be already installed")
				}
			})
		}
	})

	t.Run("workflow_types", func(t *testing.T) {
		aliases := []string{"workflow", "wf", "w"}
		for _, alias := range aliases {
			t.Run(alias, func(t *testing.T) {
				comp, alreadyInstalled, err := resolveComponent(alias, "security-audit", config)
				if err != nil {
					t.Fatalf("resolveComponent(%q, %q) error = %v", alias, "security-audit", err)
				}
				if comp == nil {
					t.Fatal("expected non-nil component")
				}
				if comp.Name != "security-audit" {
					t.Errorf("component.Name = %q, want %q", comp.Name, "security-audit")
				}
				if alreadyInstalled {
					t.Error("security-audit should not be already installed")
				}
			})
		}
	})

	t.Run("already_installed", func(t *testing.T) {
		tests := []struct {
			name          string
			componentType string
			componentName string
		}{
			{"language", "language", "go"},
			{"framework", "framework", "react"},
			{"workflow", "workflow", "create-prd"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, alreadyInstalled, err := resolveComponent(tt.componentType, tt.componentName, config)
				if err != nil {
					t.Fatalf("resolveComponent() error = %v", err)
				}
				if !alreadyInstalled {
					t.Errorf("%s %q should be already installed", tt.componentType, tt.componentName)
				}
			})
		}
	})

	t.Run("unknown_component", func(t *testing.T) {
		tests := []struct {
			name          string
			componentType string
			componentName string
			wantErrMsg    string
		}{
			{"unknown language", "language", "nonexistent-lang", "unknown language: nonexistent-lang"},
			{"unknown framework", "framework", "nonexistent-fw", "unknown framework: nonexistent-fw"},
			{"unknown workflow", "workflow", "nonexistent-wf", "unknown workflow: nonexistent-wf"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				comp, _, err := resolveComponent(tt.componentType, tt.componentName, config)
				if err == nil {
					t.Fatal("expected error for unknown component")
				}
				if comp != nil {
					t.Error("expected nil component for unknown name")
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErrMsg)
				}
			})
		}
	})

	t.Run("unknown_type", func(t *testing.T) {
		comp, _, err := resolveComponent("invalid", "go", config)
		if err == nil {
			t.Fatal("expected error for unknown component type")
		}
		if comp != nil {
			t.Error("expected nil component for unknown type")
		}
		if !strings.Contains(err.Error(), "unknown component type: invalid") {
			t.Errorf("error = %q, want containing 'unknown component type'", err.Error())
		}
	})
}

func TestResolveComponent_ComponentPath(t *testing.T) {
	config := core.NewConfig("1.0.0")
	tests := []struct {
		name          string
		componentType string
		componentName string
		wantPath      string
	}{
		{"language path", "language", "go", ".claude/skills/go-guide"},
		{"framework path", "framework", "react", ".claude/skills/react"},
		{"workflow path", "workflow", "create-prd", ".claude/skills/create-prd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp, _, err := resolveComponent(tt.componentType, tt.componentName, config)
			if err != nil {
				t.Fatalf("resolveComponent() error = %v", err)
			}
			if comp.Path != tt.wantPath {
				t.Errorf("component.Path = %q, want %q", comp.Path, tt.wantPath)
			}
		})
	}
}

func TestUpdateAddConfig(t *testing.T) {
	t.Run("add_language", func(t *testing.T) {
		config := core.NewConfig("1.0.0")
		dir := setupConfigTestDir(t, config)

		err := updateAddConfig(config, "language", "rust", ".claude/skills/rust-guide")
		if err != nil {
			t.Fatalf("updateAddConfig() error = %v", err)
		}

		updated, err := core.LoadConfigFrom(dir)
		if err != nil {
			t.Fatalf("failed to reload config: %v", err)
		}
		if !updated.HasLanguage("rust") {
			t.Error("config should have language 'rust' after add")
		}
	})

	t.Run("add_framework", func(t *testing.T) {
		config := core.NewConfig("1.0.0")
		dir := setupConfigTestDir(t, config)

		err := updateAddConfig(config, "framework", "django", ".claude/skills/django")
		if err != nil {
			t.Fatalf("updateAddConfig() error = %v", err)
		}

		updated, err := core.LoadConfigFrom(dir)
		if err != nil {
			t.Fatalf("failed to reload config: %v", err)
		}
		if !updated.HasFramework("django") {
			t.Error("config should have framework 'django' after add")
		}
	})

	t.Run("add_workflow", func(t *testing.T) {
		config := core.NewConfig("1.0.0")
		dir := setupConfigTestDir(t, config)

		err := updateAddConfig(config, "workflow", "security-audit", ".claude/skills/security-audit")
		if err != nil {
			t.Fatalf("updateAddConfig() error = %v", err)
		}

		updated, err := core.LoadConfigFrom(dir)
		if err != nil {
			t.Fatalf("failed to reload config: %v", err)
		}
		if !updated.HasWorkflow("security-audit") {
			t.Error("config should have workflow 'security-audit' after add")
		}
	})

	t.Run("add_language_alias", func(t *testing.T) {
		config := core.NewConfig("1.0.0")
		dir := setupConfigTestDir(t, config)

		err := updateAddConfig(config, "lang", "python", ".claude/skills/python-guide")
		if err != nil {
			t.Fatalf("updateAddConfig() error = %v", err)
		}

		updated, err := core.LoadConfigFrom(dir)
		if err != nil {
			t.Fatalf("failed to reload config: %v", err)
		}
		if !updated.HasLanguage("python") {
			t.Error("config should have language 'python' after add with 'lang' alias")
		}
	})

	t.Run("add_framework_alias", func(t *testing.T) {
		config := core.NewConfig("1.0.0")
		dir := setupConfigTestDir(t, config)

		err := updateAddConfig(config, "fw", "react", ".claude/skills/react")
		if err != nil {
			t.Fatalf("updateAddConfig() error = %v", err)
		}

		updated, err := core.LoadConfigFrom(dir)
		if err != nil {
			t.Fatalf("failed to reload config: %v", err)
		}
		if !updated.HasFramework("react") {
			t.Error("config should have framework 'react' after add with 'fw' alias")
		}
	})

	t.Run("add_workflow_alias", func(t *testing.T) {
		config := core.NewConfig("1.0.0")
		dir := setupConfigTestDir(t, config)

		err := updateAddConfig(config, "wf", "code-review", ".claude/skills/code-review")
		if err != nil {
			t.Fatalf("updateAddConfig() error = %v", err)
		}

		updated, err := core.LoadConfigFrom(dir)
		if err != nil {
			t.Fatalf("failed to reload config: %v", err)
		}
		if !updated.HasWorkflow("code-review") {
			t.Error("config should have workflow 'code-review' after add with 'wf' alias")
		}
	})

	t.Run("preserves_existing_components", func(t *testing.T) {
		config := core.NewConfig("1.0.0")
		config.Installed.Languages = []string{"go"}
		config.Installed.Frameworks = []string{"react"}
		dir := setupConfigTestDir(t, config)

		err := updateAddConfig(config, "language", "rust", ".claude/skills/rust-guide")
		if err != nil {
			t.Fatalf("updateAddConfig() error = %v", err)
		}

		updated, err := core.LoadConfigFrom(dir)
		if err != nil {
			t.Fatalf("failed to reload config: %v", err)
		}
		if !updated.HasLanguage("go") {
			t.Error("existing language 'go' should be preserved")
		}
		if !updated.HasLanguage("rust") {
			t.Error("new language 'rust' should be added")
		}
		if !updated.HasFramework("react") {
			t.Error("existing framework 'react' should be preserved")
		}
	})
}

func TestRunAdd_NoConfig(t *testing.T) {
	setupConfigTestDir(t, nil)

	err := runAdd(nil, []string{"language", "go"})
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
	if !strings.Contains(err.Error(), "samuel init") {
		t.Errorf("error = %q, want containing 'samuel init'", err.Error())
	}
}

func TestRunAdd_CorruptConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/samuel.yaml", []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}
	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err := runAdd(nil, []string{"language", "go"})
	if err == nil {
		t.Fatal("expected error with corrupt config")
	}
	if !strings.Contains(err.Error(), "failed to load config") {
		t.Errorf("error = %q, want containing 'failed to load config'", err.Error())
	}
}

func TestRunAdd_InvalidType(t *testing.T) {
	// In v3, neither positional is a recognized type alias and "go" is a
	// language name (not a type), so the parser cannot identify which arg is
	// which. The error wording changed from "unknown component type" (v2) to
	// "could not interpret" — same outcome (rejection), clearer message.
	config := core.NewConfig("1.0.0")
	setupConfigTestDir(t, config)

	err := runAdd(nil, []string{"badtype", "go"})
	if err == nil {
		t.Fatal("expected error for unparseable arg pair")
	}
	if !strings.Contains(err.Error(), "could not interpret") {
		t.Errorf("error = %q, want containing 'could not interpret'", err.Error())
	}
}

func TestRunAdd_UnknownComponent(t *testing.T) {
	config := core.NewConfig("1.0.0")
	setupConfigTestDir(t, config)

	err := runAdd(nil, []string{"language", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown component")
	}
	if !strings.Contains(err.Error(), "unknown language") {
		t.Errorf("error = %q, want containing 'unknown language'", err.Error())
	}
}

func TestRunAdd_AlreadyInstalled(t *testing.T) {
	config := core.NewConfig("1.0.0")
	config.Installed.Languages = []string{"go"}
	setupConfigTestDir(t, config)

	// Already installed returns nil (warns but doesn't error)
	err := runAdd(nil, []string{"language", "go"})
	if err != nil {
		t.Errorf("runAdd() for already installed component should not error, got: %v", err)
	}
}
