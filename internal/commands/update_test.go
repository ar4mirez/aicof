package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// --- fileExists tests ---

func TestFileExists(t *testing.T) {
	t.Run("existing_file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}
		if !fileExists(path) {
			t.Error("fileExists() returned false for existing file")
		}
	})

	t.Run("nonexistent_file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "missing.txt")
		if fileExists(path) {
			t.Error("fileExists() returned true for nonexistent file")
		}
	})

	t.Run("existing_directory", func(t *testing.T) {
		dir := t.TempDir()
		if !fileExists(dir) {
			t.Error("fileExists() returned false for existing directory")
		}
	})

	t.Run("empty_path", func(t *testing.T) {
		if fileExists("") {
			t.Error("fileExists() returned true for empty path")
		}
	})
}

// --- categorizeFileChanges tests ---

func TestCategorizeFileChanges(t *testing.T) {
	t.Run("empty_paths", func(t *testing.T) {
		changes := categorizeFileChanges(nil, "/local", "/cache")
		if len(changes.newFiles) != 0 || len(changes.modifiedFiles) != 0 || len(changes.unchangedFiles) != 0 {
			t.Errorf("expected all empty, got new=%d, modified=%d, unchanged=%d",
				len(changes.newFiles), len(changes.modifiedFiles), len(changes.unchangedFiles))
		}
	})

	t.Run("new_file_in_cache_only", func(t *testing.T) {
		cwd := t.TempDir()
		cache := t.TempDir()

		// File exists in cache but not locally
		cachePath := filepath.Join(cache, "CLAUDE.md")
		if err := os.WriteFile(cachePath, []byte("new content"), 0644); err != nil {
			t.Fatal(err)
		}

		changes := categorizeFileChanges([]string{"CLAUDE.md"}, cwd, cache)
		if len(changes.newFiles) != 1 || changes.newFiles[0] != "CLAUDE.md" {
			t.Errorf("expected 1 new file 'CLAUDE.md', got %v", changes.newFiles)
		}
		if len(changes.modifiedFiles) != 0 {
			t.Errorf("expected 0 modified, got %d", len(changes.modifiedFiles))
		}
		if len(changes.unchangedFiles) != 0 {
			t.Errorf("expected 0 unchanged, got %d", len(changes.unchangedFiles))
		}
	})

	t.Run("unchanged_file", func(t *testing.T) {
		cwd := t.TempDir()
		cache := t.TempDir()

		content := []byte("same content")
		if err := os.WriteFile(filepath.Join(cwd, "file.md"), content, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cache, "file.md"), content, 0644); err != nil {
			t.Fatal(err)
		}

		changes := categorizeFileChanges([]string{"file.md"}, cwd, cache)
		if len(changes.unchangedFiles) != 1 || changes.unchangedFiles[0] != "file.md" {
			t.Errorf("expected 1 unchanged file, got %v", changes.unchangedFiles)
		}
		if len(changes.newFiles) != 0 {
			t.Errorf("expected 0 new, got %d", len(changes.newFiles))
		}
		if len(changes.modifiedFiles) != 0 {
			t.Errorf("expected 0 modified, got %d", len(changes.modifiedFiles))
		}
	})

	t.Run("modified_file", func(t *testing.T) {
		cwd := t.TempDir()
		cache := t.TempDir()

		if err := os.WriteFile(filepath.Join(cwd, "config.yaml"), []byte("local version"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cache, "config.yaml"), []byte("cache version"), 0644); err != nil {
			t.Fatal(err)
		}

		changes := categorizeFileChanges([]string{"config.yaml"}, cwd, cache)
		if len(changes.modifiedFiles) != 1 || changes.modifiedFiles[0] != "config.yaml" {
			t.Errorf("expected 1 modified file, got %v", changes.modifiedFiles)
		}
		if len(changes.newFiles) != 0 {
			t.Errorf("expected 0 new, got %d", len(changes.newFiles))
		}
		if len(changes.unchangedFiles) != 0 {
			t.Errorf("expected 0 unchanged, got %d", len(changes.unchangedFiles))
		}
	})

	t.Run("file_removed_in_new_version", func(t *testing.T) {
		cwd := t.TempDir()
		cache := t.TempDir()

		// File exists locally but not in cache (removed in new version)
		if err := os.WriteFile(filepath.Join(cwd, "old.md"), []byte("old"), 0644); err != nil {
			t.Fatal(err)
		}

		changes := categorizeFileChanges([]string{"old.md"}, cwd, cache)
		if len(changes.newFiles) != 0 || len(changes.modifiedFiles) != 0 || len(changes.unchangedFiles) != 0 {
			t.Errorf("file removed in new version should be skipped, got new=%d, modified=%d, unchanged=%d",
				len(changes.newFiles), len(changes.modifiedFiles), len(changes.unchangedFiles))
		}
	})

	t.Run("mixed_categories", func(t *testing.T) {
		cwd := t.TempDir()
		cache := t.TempDir()

		// new file (only in cache)
		if err := os.WriteFile(filepath.Join(cache, "new.md"), []byte("new"), 0644); err != nil {
			t.Fatal(err)
		}

		// unchanged file (same in both)
		sameContent := []byte("same")
		if err := os.WriteFile(filepath.Join(cwd, "same.md"), sameContent, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cache, "same.md"), sameContent, 0644); err != nil {
			t.Fatal(err)
		}

		// modified file (different content)
		if err := os.WriteFile(filepath.Join(cwd, "changed.md"), []byte("old"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cache, "changed.md"), []byte("new"), 0644); err != nil {
			t.Fatal(err)
		}

		// removed file (only local, not in cache) — should be skipped
		if err := os.WriteFile(filepath.Join(cwd, "removed.md"), []byte("gone"), 0644); err != nil {
			t.Fatal(err)
		}

		paths := []string{"new.md", "same.md", "changed.md", "removed.md"}
		changes := categorizeFileChanges(paths, cwd, cache)

		if len(changes.newFiles) != 1 || changes.newFiles[0] != "new.md" {
			t.Errorf("expected newFiles=[new.md], got %v", changes.newFiles)
		}
		if len(changes.unchangedFiles) != 1 || changes.unchangedFiles[0] != "same.md" {
			t.Errorf("expected unchangedFiles=[same.md], got %v", changes.unchangedFiles)
		}
		if len(changes.modifiedFiles) != 1 || changes.modifiedFiles[0] != "changed.md" {
			t.Errorf("expected modifiedFiles=[changed.md], got %v", changes.modifiedFiles)
		}
	})

	t.Run("nested_paths", func(t *testing.T) {
		cwd := t.TempDir()
		cache := t.TempDir()

		nestedPath := filepath.Join(".claude", "skills", "go-guide", "SKILL.md")

		// Create nested dirs in both locations
		if err := os.MkdirAll(filepath.Join(cwd, ".claude", "skills", "go-guide"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(cache, ".claude", "skills", "go-guide"), 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(cwd, nestedPath), []byte("local"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cache, nestedPath), []byte("cached"), 0644); err != nil {
			t.Fatal(err)
		}

		changes := categorizeFileChanges([]string{nestedPath}, cwd, cache)
		if len(changes.modifiedFiles) != 1 {
			t.Errorf("expected 1 modified nested file, got %d", len(changes.modifiedFiles))
		}
	})

	t.Run("unreadable_local_file_skipped", func(t *testing.T) {
		cwd := t.TempDir()
		cache := t.TempDir()

		localPath := filepath.Join(cwd, "secret.md")
		if err := os.WriteFile(localPath, []byte("local"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cache, "secret.md"), []byte("cache"), 0644); err != nil {
			t.Fatal(err)
		}

		// Make local file unreadable
		if err := os.Chmod(localPath, 0000); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Chmod(localPath, 0644); err != nil {
				t.Errorf("failed to restore file mode: %v", err)
			}
		}()

		changes := categorizeFileChanges([]string{"secret.md"}, cwd, cache)
		// Unreadable local file should be silently skipped
		if len(changes.newFiles)+len(changes.modifiedFiles)+len(changes.unchangedFiles) != 0 {
			t.Errorf("unreadable file should be skipped, got new=%d, modified=%d, unchanged=%d",
				len(changes.newFiles), len(changes.modifiedFiles), len(changes.unchangedFiles))
		}
	})

	t.Run("unreadable_cache_file_skipped", func(t *testing.T) {
		cwd := t.TempDir()
		cache := t.TempDir()

		if err := os.WriteFile(filepath.Join(cwd, "secret.md"), []byte("local"), 0644); err != nil {
			t.Fatal(err)
		}
		cachePath := filepath.Join(cache, "secret.md")
		if err := os.WriteFile(cachePath, []byte("cache"), 0644); err != nil {
			t.Fatal(err)
		}

		// Make cache file unreadable
		if err := os.Chmod(cachePath, 0000); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Chmod(cachePath, 0644); err != nil {
				t.Errorf("failed to restore file mode: %v", err)
			}
		}()

		changes := categorizeFileChanges([]string{"secret.md"}, cwd, cache)
		// Unreadable cache file should be silently skipped
		if len(changes.newFiles)+len(changes.modifiedFiles)+len(changes.unchangedFiles) != 0 {
			t.Errorf("unreadable cache file should be skipped, got new=%d, modified=%d, unchanged=%d",
				len(changes.newFiles), len(changes.modifiedFiles), len(changes.unchangedFiles))
		}
	})

	t.Run("empty_files_are_unchanged", func(t *testing.T) {
		cwd := t.TempDir()
		cache := t.TempDir()

		// Both files empty — should be unchanged
		if err := os.WriteFile(filepath.Join(cwd, "empty.md"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cache, "empty.md"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}

		changes := categorizeFileChanges([]string{"empty.md"}, cwd, cache)
		if len(changes.unchangedFiles) != 1 {
			t.Errorf("expected empty files to be unchanged, got %v", changes)
		}
	})
}

// --- runUpdate tests ---

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "update",
		RunE: runUpdate,
	}
	cmd.Flags().Bool("check", false, "Check for updates without applying")
	cmd.Flags().Bool("diff", false, "Show what files will change")
	cmd.Flags().BoolP("force", "f", false, "Overwrite local modifications")
	cmd.Flags().String("version", "", "Update to specific version")
	return cmd
}

func TestRunUpdate(t *testing.T) {
	t.Run("no_config_returns_error", func(t *testing.T) {
		dir := t.TempDir()
		oldDir, _ := os.Getwd()
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Chdir(oldDir); err != nil {
				t.Errorf("failed to restore working directory: %v", err)
			}
		}()

		cmd := newUpdateCmd()
		err := cmd.RunE(cmd, nil)
		if err == nil {
			t.Fatal("expected error when no config exists")
		}
		if !strings.Contains(err.Error(), "no Samuel installation found") {
			t.Errorf("expected 'no Samuel installation found', got: %v", err)
		}
	})

	t.Run("corrupt_config_returns_error", func(t *testing.T) {
		dir := t.TempDir()
		oldDir, _ := os.Getwd()
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Chdir(oldDir); err != nil {
				t.Errorf("failed to restore working directory: %v", err)
			}
		}()

		// Write invalid YAML
		if err := os.WriteFile(filepath.Join(dir, "samuel.yaml"), []byte("{{invalid yaml}}"), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := newUpdateCmd()
		err := cmd.RunE(cmd, nil)
		if err == nil {
			t.Fatal("expected error for corrupt config")
		}
		if !strings.Contains(err.Error(), "failed to load config") {
			t.Errorf("expected 'failed to load config', got: %v", err)
		}
	})
}
