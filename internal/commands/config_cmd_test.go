package commands

import (
	"os"
	"strings"
	"testing"

	"github.com/ar4mirez/samuel/internal/core"
)

func TestIsValidConfigKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		// Standard keys
		{"version", true},
		{"registry", true},
		{"installed.languages", true},
		{"installed.frameworks", true},
		{"installed.workflows", true},
		{"installed.skills", true},
		// Auto keys
		{"auto.enabled", true},
		{"auto.ai_tool", true},
		{"auto.max_iterations", true},
		{"auto.quality_checks", true},
		// Invalid keys
		{"invalid", false},
		{"", false},
		{"VERSION", false},
		{"installed", false},
		{"auto", false},
		{"installed.", false},
		{"auto.", false},
		{".version", false},
		{"version ", false},
		{" version", false},
	}

	for _, tt := range tests {
		name := tt.key
		if name == "" {
			name = "(empty)"
		}
		t.Run(name, func(t *testing.T) {
			got := isValidConfigKey(tt.key)
			if got != tt.want {
				t.Errorf("isValidConfigKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestIsValidConfigKey_MatchesValidConfigKeys(t *testing.T) {
	// Every key in ValidConfigKeys should be accepted
	for _, key := range core.ValidConfigKeys {
		if !isValidConfigKey(key) {
			t.Errorf("isValidConfigKey(%q) = false, but key is in ValidConfigKeys", key)
		}
	}
}

func TestFormatConfigValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{"non-empty string", "hello", "hello"},
		{"empty string", "", "(empty)"},
		{"string with spaces", "hello world", "hello world"},
		{"url string", "https://github.com/ar4mirez/samuel", "https://github.com/ar4mirez/samuel"},
		{"string slice multiple", []string{"a", "b", "c"}, "a, b, c"},
		{"string slice single", []string{"single"}, "single"},
		{"string slice empty", []string{}, "(none)"},
		{"nil value", nil, "<nil>"},
		{"integer", 42, "42"},
		{"boolean true", true, "true"},
		{"boolean false", false, "false"},
		{"float", 3.14, "3.14"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatConfigValue(tt.value)
			if got != tt.want {
				t.Errorf("formatConfigValue(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestValidateRegistryURL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
		errMsg  string
	}{
		{"valid https", "https://github.com/myorg/myrepo", false, ""},
		{"valid https with path", "https://github.com/ar4mirez/samuel", false, ""},
		{"valid https with port", "https://github.com:443/myorg/myrepo", false, ""},
		{"valid https bare host", "https://example.com", false, ""},
		{"http rejected", "http://github.com/myorg/myrepo", true, "HTTPS scheme"},
		{"empty scheme", "github.com/myorg/myrepo", true, "HTTPS scheme"},
		{"ftp rejected", "ftp://example.com/repo", true, "HTTPS scheme"},
		{"empty string", "", true, "HTTPS scheme"},
		{"plain text", "not-a-url", true, "HTTPS scheme"},
		{"scheme only", "https://", true, "must have a host"},
		{"ssh rejected", "ssh://git@github.com/repo", true, "HTTPS scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegistryURL(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRegistryURL(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateRegistryURL(%q) error = %v, want error containing %q", tt.value, err, tt.errMsg)
				}
			}
		})
	}
}

func setupConfigTestDir(t *testing.T, config *core.Config) string {
	t.Helper()
	dir := t.TempDir()
	if config != nil {
		if err := config.Save(dir); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
	}
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})
	return dir
}

func TestRunConfigList(t *testing.T) {
	t.Run("no_config_file", func(t *testing.T) {
		setupConfigTestDir(t, nil)
		err := runConfigList(nil, nil)
		if err != nil {
			t.Errorf("runConfigList() with no config should not error, got: %v", err)
		}
	})

	t.Run("valid_config", func(t *testing.T) {
		config := core.NewConfig("1.0.0")
		config.Registry = "https://github.com/ar4mirez/samuel"
		config.Installed.Languages = []string{"go", "rust"}
		setupConfigTestDir(t, config)

		err := runConfigList(nil, nil)
		if err != nil {
			t.Errorf("runConfigList() error = %v", err)
		}
	})

	t.Run("corrupt_config", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(dir+"/samuel.yaml", []byte("{{invalid yaml"), 0644); err != nil {
			t.Fatal(err)
		}
		oldDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get cwd: %v", err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}
		defer func() {
			if err := os.Chdir(oldDir); err != nil {
				t.Errorf("failed to restore working directory: %v", err)
			}
		}()

		runErr := runConfigList(nil, nil)
		if runErr == nil {
			t.Error("runConfigList() with corrupt config should error")
		}
		if !strings.Contains(runErr.Error(), "failed to load config") {
			t.Errorf("unexpected error: %v", runErr)
		}
	})
}

func TestRunConfigGet(t *testing.T) {
	t.Run("invalid_key", func(t *testing.T) {
		setupConfigTestDir(t, nil)
		err := runConfigGet(nil, []string{"invalid_key"})
		if err == nil {
			t.Error("runConfigGet() with invalid key should error")
		}
		if !strings.Contains(err.Error(), "invalid config key") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("no_config_file", func(t *testing.T) {
		setupConfigTestDir(t, nil)
		err := runConfigGet(nil, []string{"version"})
		if err != nil {
			t.Errorf("runConfigGet() with no config should not error, got: %v", err)
		}
	})

	t.Run("valid_key", func(t *testing.T) {
		config := core.NewConfig("2.5.0")
		setupConfigTestDir(t, config)

		err := runConfigGet(nil, []string{"version"})
		if err != nil {
			t.Errorf("runConfigGet(version) error = %v", err)
		}
	})

	t.Run("installed_languages", func(t *testing.T) {
		config := core.NewConfig("1.0.0")
		config.Installed.Languages = []string{"go", "python"}
		setupConfigTestDir(t, config)

		err := runConfigGet(nil, []string{"installed.languages"})
		if err != nil {
			t.Errorf("runConfigGet(installed.languages) error = %v", err)
		}
	})
}

func TestRunConfigSet(t *testing.T) {
	t.Run("invalid_key", func(t *testing.T) {
		setupConfigTestDir(t, nil)
		err := runConfigSet(nil, []string{"bad_key", "value"})
		if err == nil {
			t.Error("runConfigSet() with invalid key should error")
		}
		if !strings.Contains(err.Error(), "invalid config key") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid_registry_url", func(t *testing.T) {
		config := core.NewConfig("1.0.0")
		setupConfigTestDir(t, config)

		err := runConfigSet(nil, []string{"registry", "http://insecure.com"})
		if err == nil {
			t.Error("runConfigSet() with HTTP registry should error")
		}
		if !strings.Contains(err.Error(), "invalid registry value") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("no_config_file", func(t *testing.T) {
		setupConfigTestDir(t, nil)
		err := runConfigSet(nil, []string{"version", "2.0.0"})
		if err != nil {
			t.Errorf("runConfigSet() with no config should not error, got: %v", err)
		}
	})

	t.Run("set_version", func(t *testing.T) {
		config := core.NewConfig("1.0.0")
		dir := setupConfigTestDir(t, config)

		err := runConfigSet(nil, []string{"version", "2.0.0"})
		if err != nil {
			t.Fatalf("runConfigSet() error = %v", err)
		}

		// Verify config was updated
		updated, err := core.LoadConfigFrom(dir)
		if err != nil {
			t.Fatalf("failed to reload config: %v", err)
		}
		if updated.Version != "2.0.0" {
			t.Errorf("config.Version = %q, want %q", updated.Version, "2.0.0")
		}
	})

	t.Run("set_registry_valid", func(t *testing.T) {
		config := core.NewConfig("1.0.0")
		dir := setupConfigTestDir(t, config)

		err := runConfigSet(nil, []string{"registry", "https://github.com/custom/repo"})
		if err != nil {
			t.Fatalf("runConfigSet() error = %v", err)
		}

		updated, err := core.LoadConfigFrom(dir)
		if err != nil {
			t.Fatalf("failed to reload config: %v", err)
		}
		if updated.Registry != "https://github.com/custom/repo" {
			t.Errorf("config.Registry = %q, want %q", updated.Registry, "https://github.com/custom/repo")
		}
	})

	t.Run("set_installed_languages", func(t *testing.T) {
		config := core.NewConfig("1.0.0")
		dir := setupConfigTestDir(t, config)

		err := runConfigSet(nil, []string{"installed.languages", "go,rust,python"})
		if err != nil {
			t.Fatalf("runConfigSet() error = %v", err)
		}

		updated, err := core.LoadConfigFrom(dir)
		if err != nil {
			t.Fatalf("failed to reload config: %v", err)
		}
		if len(updated.Installed.Languages) != 3 {
			t.Errorf("installed.languages has %d items, want 3", len(updated.Installed.Languages))
		}
	})
}
