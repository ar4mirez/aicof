package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
)

// captureStdout captures stdout output during fn execution.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// parseJSONOutput parses captured stdout as a JSON response.
func parseJSONOutput(t *testing.T, output string) ui.JSONResponse {
	t.Helper()
	var resp ui.JSONResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nraw: %s", err, output)
	}
	return resp
}

func TestJSONMode(t *testing.T) {
	t.Run("returns_false_when_flag_not_set", func(t *testing.T) {
		cmd := rootCmd
		if JSONMode(cmd) {
			t.Error("expected JSONMode to be false by default")
		}
	})

	t.Run("returns_true_when_flag_set", func(t *testing.T) {
		if err := rootCmd.PersistentFlags().Set("json", "true"); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = rootCmd.PersistentFlags().Set("json", "false") }()

		if !JSONMode(rootCmd) {
			t.Error("expected JSONMode to be true")
		}
	})
}

func TestVersionJSON(t *testing.T) {
	output := captureStdout(t, func() {
		_ = runVersionJSON(false)
	})

	resp := parseJSONOutput(t, output)
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Command != "version" {
		t.Errorf("expected command=version, got %s", resp.Command)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	cli, ok := data["cli"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data.cli to be a map")
	}
	if _, ok := cli["version"]; !ok {
		t.Error("expected data.cli.version to exist")
	}
}

func TestListJSON(t *testing.T) {
	t.Run("available_mode", func(t *testing.T) {
		_, cleanup := setupListTestDir(t, nil)
		defer cleanup()

		output := captureStdout(t, func() {
			_ = listJSON(listCmd, true, "")
		})

		resp := parseJSONOutput(t, output)
		if !resp.Success {
			t.Error("expected success=true")
		}
		if resp.SchemaVersion != 3 {
			t.Errorf("expected schemaVersion=3, got %d", resp.SchemaVersion)
		}
		if resp.Command != "list" {
			t.Errorf("expected command=list (legacy listCmd path), got %s", resp.Command)
		}

		data := resp.Data.(map[string]interface{})
		components, ok := data["components"].([]interface{})
		if !ok {
			t.Fatal("expected data.components to be an array")
		}
		if len(components) == 0 {
			t.Error("expected at least one component in available mode")
		}
	})

	t.Run("installed_mode_with_config", func(t *testing.T) {
		config := &core.Config{
			Version: "1.0.0",
			Installed: core.InstalledItems{
				Languages:  []string{"go"},
				Frameworks: []string{},
				Workflows:  []string{},
			},
		}
		_, cleanup := setupListTestDir(t, config)
		defer cleanup()

		output := captureStdout(t, func() {
			_ = listJSON(listCmd, false, "")
		})

		resp := parseJSONOutput(t, output)
		if !resp.Success {
			t.Error("expected success=true")
		}

		data := resp.Data.(map[string]interface{})
		if data["version"] != "1.0.0" {
			t.Errorf("expected version=1.0.0, got %v", data["version"])
		}
	})

	t.Run("type_filter", func(t *testing.T) {
		_, cleanup := setupListTestDir(t, nil)
		defer cleanup()

		output := captureStdout(t, func() {
			_ = listJSON(listCmd, true, "language")
		})

		resp := parseJSONOutput(t, output)
		data := resp.Data.(map[string]interface{})
		components := data["components"].([]interface{})
		for _, c := range components {
			comp := c.(map[string]interface{})
			if comp["type"] != "language" {
				t.Errorf("expected type=language, got %s", comp["type"])
			}
		}
	})
}

func TestSearchJSON(t *testing.T) {
	_, cleanup := setupListTestDir(t, nil)
	defer cleanup()

	cmd := searchCmd
	if err := cmd.Flags().Set("type", ""); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("limit", "5"); err != nil {
		t.Fatal(err)
	}

	// Set the json flag on the root command for inheritance
	if err := rootCmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rootCmd.PersistentFlags().Set("json", "false") }()

	output := captureStdout(t, func() {
		_ = runSearch(cmd, []string{"go"})
	})

	resp := parseJSONOutput(t, output)
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Command != "search" {
		t.Errorf("expected command=search, got %s", resp.Command)
	}

	data := resp.Data.(map[string]interface{})
	if _, ok := data["query"]; !ok {
		t.Error("expected data.query to exist")
	}
	if _, ok := data["results"]; !ok {
		t.Error("expected data.results to exist")
	}
}

func TestDoctorJSON(t *testing.T) {
	config := &core.Config{
		Version: "1.0.0",
		Installed: core.InstalledItems{
			Languages:  []string{},
			Frameworks: []string{},
			Workflows:  []string{},
		},
	}
	dir, cleanup := setupListTestDir(t, config)
	defer cleanup()

	// Create CLAUDE.md and AGENTS.md so checks pass
	_ = os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Test"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Test"), 0644)
	_ = os.MkdirAll(filepath.Join(dir, ".claude", "skills"), 0755)

	if err := rootCmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rootCmd.PersistentFlags().Set("json", "false") }()

	output := captureStdout(t, func() {
		_ = runDoctor(doctorCmd, []string{})
	})

	resp := parseJSONOutput(t, output)
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Command != "doctor" {
		t.Errorf("expected command=doctor, got %s", resp.Command)
	}

	data := resp.Data.(map[string]interface{})
	if _, ok := data["checks"]; !ok {
		t.Error("expected data.checks to exist")
	}
	if _, ok := data["passed"]; !ok {
		t.Error("expected data.passed to exist")
	}
}

func TestConfigListJSON(t *testing.T) {
	config := &core.Config{
		Version: "2.0.0",
		Installed: core.InstalledItems{
			Languages:  []string{"go", "python"},
			Frameworks: []string{"react"},
			Workflows:  []string{"all"},
		},
	}
	_, cleanup := setupListTestDir(t, config)
	defer cleanup()

	if err := rootCmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rootCmd.PersistentFlags().Set("json", "false") }()

	output := captureStdout(t, func() {
		_ = runConfigList(configListCmd, []string{})
	})

	resp := parseJSONOutput(t, output)
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.SchemaVersion != 3 {
		t.Errorf("expected schemaVersion=3, got %d", resp.SchemaVersion)
	}
	// configListCmd is mounted under 'samuel admin config' as of v3, so the
	// JSON envelope reports the invoked path. The legacy 'samuel config list'
	// path emits "config list" (see legacyConfigListCmd) and is exercised by
	// integration tests, not this unit test.
	if resp.Command != "admin config list" {
		t.Errorf("expected command='admin config list', got %s", resp.Command)
	}

	data := resp.Data.(map[string]interface{})
	if data["version"] != "2.0.0" {
		t.Errorf("expected version=2.0.0, got %v", data["version"])
	}
}

func TestConfigGetJSON(t *testing.T) {
	config := &core.Config{
		Version: "2.0.0",
		Installed: core.InstalledItems{
			Languages:  []string{"go"},
			Frameworks: []string{},
			Workflows:  []string{},
		},
	}
	_, cleanup := setupListTestDir(t, config)
	defer cleanup()

	if err := rootCmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rootCmd.PersistentFlags().Set("json", "false") }()

	output := captureStdout(t, func() {
		_ = runConfigGet(configGetCmd, []string{"version"})
	})

	resp := parseJSONOutput(t, output)
	if !resp.Success {
		t.Error("expected success=true")
	}

	data := resp.Data.(map[string]interface{})
	if data["key"] != "version" {
		t.Errorf("expected key=version, got %v", data["key"])
	}
	if data["value"] != "2.0.0" {
		t.Errorf("expected value=2.0.0, got %v", data["value"])
	}
}

func TestAutoTaskListJSON(t *testing.T) {
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	autoDir := core.GetAutoDir(dir)
	if err := os.MkdirAll(autoDir, 0755); err != nil {
		t.Fatal(err)
	}

	prd := core.NewAutoPRD("test-project", "Test description")
	task := core.AutoTask{
		ID:       "1.0",
		Title:    "Test task",
		Status:   core.TaskStatusPending,
		Priority: core.TaskPriorityMedium,
	}
	_ = prd.AddTask(task)
	prdPath := core.GetAutoPRDPath(dir)
	if err := prd.Save(prdPath); err != nil {
		t.Fatal(err)
	}

	if err := rootCmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rootCmd.PersistentFlags().Set("json", "false") }()

	output := captureStdout(t, func() {
		_ = runAutoTaskList(autoTaskListCmd, []string{})
	})

	resp := parseJSONOutput(t, output)
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.SchemaVersion != 3 {
		t.Errorf("expected schemaVersion=3, got %d", resp.SchemaVersion)
	}
	// autoTaskListCmd is the preserved legacy nested form mounted under the
	// renamed 'run task' parent. v3 reports the invoked path "run task list".
	// The new flat verb is autoTasksCmd ('samuel run tasks'), which would
	// emit "run tasks" instead.
	if resp.Command != "run task list" {
		t.Errorf("expected command='run task list', got %s", resp.Command)
	}

	data := resp.Data.(map[string]interface{})
	tasks := data["tasks"].([]interface{})
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
}

func TestPrintJSON(t *testing.T) {
	output := captureStdout(t, func() {
		ui.PrintJSON("test", map[string]string{"hello": "world"})
	})

	var resp ui.JSONResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Command != "test" {
		t.Errorf("expected command=test, got %s", resp.Command)
	}
}
