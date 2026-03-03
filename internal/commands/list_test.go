package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ar4mirez/samuel/internal/core"
)

// setupListTestDir creates a temp dir with an optional samuel.yaml and changes cwd.
// Returns the temp dir path and a cleanup function that restores the original cwd.
func setupListTestDir(t *testing.T, config *core.Config) (string, func()) {
	t.Helper()
	dir := t.TempDir()

	if config != nil {
		if err := config.Save(dir); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
	}

	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	return dir, func() { _ = os.Chdir(oldDir) }
}

// --- listInstalled tests ---

func TestListInstalled(t *testing.T) {
	t.Run("no_config_warns_and_returns_nil", func(t *testing.T) {
		_, cleanup := setupListTestDir(t, nil)
		defer cleanup()

		err := listInstalled("")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("corrupt_config_returns_error", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "samuel.yaml"), []byte("{{invalid yaml"), 0644); err != nil {
			t.Fatal(err)
		}

		oldDir, _ := os.Getwd()
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}
		defer func() { _ = os.Chdir(oldDir) }()

		err := listInstalled("")
		if err == nil {
			t.Error("expected error for corrupt config, got nil")
		}
	})

	t.Run("empty_config_shows_none_installed", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Languages:  []string{},
				Frameworks: []string{},
				Workflows:  []string{},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		err := listInstalled("")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("with_languages_only", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Languages:  []string{"go", "python"},
				Frameworks: []string{},
				Workflows:  []string{},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		err := listInstalled("")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("with_frameworks_only", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Languages:  []string{},
				Frameworks: []string{"react", "nextjs"},
				Workflows:  []string{},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		err := listInstalled("")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("with_workflows_all", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Languages:  []string{},
				Frameworks: []string{},
				Workflows:  []string{"all"},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		err := listInstalled("")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("with_specific_workflows", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Languages:  []string{},
				Frameworks: []string{},
				Workflows:  []string{"code-review", "testing-strategy"},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		err := listInstalled("")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("unknown_component_names", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Languages:  []string{"unknown-lang"},
				Frameworks: []string{"unknown-fw"},
				Workflows:  []string{"unknown-wf"},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		// Should not error — unknown names are displayed without descriptions
		err := listInstalled("")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("type_filter_languages", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Languages:  []string{"go"},
				Frameworks: []string{"react"},
				Workflows:  []string{"all"},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		// Should only display languages section
		err := listInstalled("languages")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("type_filter_frameworks", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Languages:  []string{"go"},
				Frameworks: []string{"react"},
				Workflows:  []string{"all"},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		err := listInstalled("frameworks")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("type_filter_workflows", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Languages:  []string{"go"},
				Frameworks: []string{"react"},
				Workflows:  []string{"all"},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		err := listInstalled("workflows")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("mixed_components", func(t *testing.T) {
		config := &core.Config{
			Version: "2.5.0",
			Installed: core.InstalledItems{
				Languages:  []string{"go", "typescript", "python"},
				Frameworks: []string{"react", "nextjs"},
				Workflows:  []string{"code-review", "testing-strategy"},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		err := listInstalled("")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})
}

// --- listAvailable tests ---

func TestListAvailable(t *testing.T) {
	t.Run("no_config_shows_all_available", func(t *testing.T) {
		_, cleanup := setupListTestDir(t, nil)
		defer cleanup()

		// Should not error — missing config is expected
		err := listAvailable("")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("with_config_marks_installed", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Languages:  []string{"go"},
				Frameworks: []string{"react"},
				Workflows:  []string{"code-review"},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		err := listAvailable("")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("with_all_workflows_installed", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Workflows: []string{"all"},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		err := listAvailable("")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("type_filter_languages", func(t *testing.T) {
		_, cleanup := setupListTestDir(t, nil)
		defer cleanup()

		err := listAvailable("languages")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("type_filter_frameworks", func(t *testing.T) {
		_, cleanup := setupListTestDir(t, nil)
		defer cleanup()

		err := listAvailable("frameworks")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("type_filter_workflows", func(t *testing.T) {
		_, cleanup := setupListTestDir(t, nil)
		defer cleanup()

		err := listAvailable("workflows")
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("corrupt_config_warns_and_continues", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "samuel.yaml"), []byte("{{invalid yaml"), 0644); err != nil {
			t.Fatal(err)
		}

		oldDir, _ := os.Getwd()
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}
		defer func() { _ = os.Chdir(oldDir) }()

		// Corrupt config should warn but not error — listAvailable is best-effort
		err := listAvailable("")
		if err != nil {
			t.Errorf("expected nil error for corrupt config, got: %v", err)
		}
	})
}

// --- runList tests ---

func TestRunList(t *testing.T) {
	t.Run("default_calls_listInstalled", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Languages: []string{"go"},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		cmd := listCmd
		cmd.SetArgs([]string{})
		// Reset flags to defaults
		if err := cmd.Flags().Set("available", "false"); err != nil {
			t.Fatalf("failed to set available flag: %v", err)
		}
		if err := cmd.Flags().Set("type", ""); err != nil {
			t.Fatalf("failed to set type flag: %v", err)
		}

		err := runList(cmd, []string{})
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("available_flag_calls_listAvailable", func(t *testing.T) {
		_, cleanup := setupListTestDir(t, nil)
		defer cleanup()

		cmd := listCmd
		if err := cmd.Flags().Set("available", "true"); err != nil {
			t.Fatalf("failed to set available flag: %v", err)
		}
		if err := cmd.Flags().Set("type", ""); err != nil {
			t.Fatalf("failed to set type flag: %v", err)
		}

		err := runList(cmd, []string{})
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
		// Reset flag for other tests
		if err := cmd.Flags().Set("available", "false"); err != nil {
			t.Fatalf("failed to reset available flag: %v", err)
		}
	})

	t.Run("type_filter_passed_through", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Languages:  []string{"go"},
				Frameworks: []string{"react"},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		cmd := listCmd
		if err := cmd.Flags().Set("available", "false"); err != nil {
			t.Fatalf("failed to set available flag: %v", err)
		}
		if err := cmd.Flags().Set("type", "languages"); err != nil {
			t.Fatalf("failed to set type flag: %v", err)
		}

		err := runList(cmd, []string{})
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
		// Reset
		if err := cmd.Flags().Set("type", ""); err != nil {
			t.Fatalf("failed to reset type flag: %v", err)
		}
	})
}
