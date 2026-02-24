package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectProjectFiles(t *testing.T) {
	dir := t.TempDir()

	// Create source files
	createTestFile(t, dir, "main.go", "package main\n")
	createTestFile(t, dir, "internal/core/foo.go", "package core\n")
	createTestFile(t, dir, "internal/core/foo_test.go", "package core\n")

	// Create non-source files (should be skipped)
	createTestFile(t, dir, "README.md", "# Readme\n")
	createTestFile(t, dir, "go.mod", "module test\n")

	// Create hidden dir (should be skipped)
	createTestFile(t, dir, ".git/config", "config\n")

	files, err := collectProjectFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("expected 3 source files, got %d", len(files))
		for _, f := range files {
			t.Logf("  file: %s", f.Path)
		}
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}

	if !paths["main.go"] {
		t.Error("missing main.go")
	}
	if !paths[filepath.Join("internal", "core", "foo.go")] {
		t.Error("missing internal/core/foo.go")
	}
}

func TestCollectProjectFiles_SkipsDirs(t *testing.T) {
	dir := t.TempDir()

	createTestFile(t, dir, "vendor/lib.go", "package lib\n")
	createTestFile(t, dir, "node_modules/index.js", "// js\n")
	createTestFile(t, dir, "bin/samuel.go", "package main\n")
	createTestFile(t, dir, "src/app.go", "package app\n")

	files, err := collectProjectFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only src/app.go should be collected
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
		for _, f := range files {
			t.Logf("  file: %s", f.Path)
		}
	}
}

func TestFindTestGaps(t *testing.T) {
	files := []FileEntry{
		{Path: "internal/core/foo.go", Size: 100},
		{Path: "internal/core/foo_test.go", Size: 50},
		{Path: "internal/core/bar.go", Size: 200},
		{Path: "cmd/main.go", Size: 30},
	}

	gaps := findTestGaps(files)

	if len(gaps) != 2 {
		t.Fatalf("expected 2 gaps, got %d: %v", len(gaps), gaps)
	}

	gapSet := make(map[string]bool)
	for _, g := range gaps {
		gapSet[g] = true
	}

	if !gapSet["internal/core/bar.go"] {
		t.Error("missing gap: internal/core/bar.go")
	}
	if !gapSet["cmd/main.go"] {
		t.Error("missing gap: cmd/main.go")
	}
}

func TestFindTestGaps_AllCovered(t *testing.T) {
	files := []FileEntry{
		{Path: "foo.go", Size: 100},
		{Path: "foo_test.go", Size: 50},
	}

	gaps := findTestGaps(files)
	if len(gaps) != 0 {
		t.Errorf("expected 0 gaps, got %d", len(gaps))
	}
}

func TestFindLargeFiles(t *testing.T) {
	files := []FileEntry{
		{Path: "a.go", Size: 100},
		{Path: "b.go", Size: 500},
		{Path: "c.go", Size: 200},
	}

	large := findLargeFiles(files)
	if len(large) != 3 {
		t.Fatalf("expected 3 files, got %d", len(large))
	}
	if large[0].Path != "b.go" {
		t.Errorf("expected b.go first, got %s", large[0].Path)
	}
}

func TestCountTODOMarkers(t *testing.T) {
	dir := t.TempDir()

	content := "package main\n// TODO: fix this\n// FIXME: also this\nfunc main() {}\n"
	createTestFile(t, dir, "main.go", content)

	// Test files should be skipped
	createTestFile(t, dir, "main_test.go", "// TODO: add test\n")

	files := []FileEntry{
		{Path: "main.go", Size: int64(len(content))},
		{Path: "main_test.go", Size: 20},
	}

	counts := countTODOMarkers(dir, files)

	if counts["main.go"] != 2 {
		t.Errorf("expected 2 markers in main.go, got %d", counts["main.go"])
	}
	if _, exists := counts["main_test.go"]; exists {
		t.Error("test files should be skipped")
	}
}

func TestCountMarkersInFile_MultipleOnSameLine(t *testing.T) {
	dir := t.TempDir()
	// Two markers on the same line should count as 1
	createTestFile(t, dir, "multi.go", "// TODO FIXME both here\n")

	count := countMarkersInFile(filepath.Join(dir, "multi.go"), []string{"TODO", "FIXME"})
	if count != 1 {
		t.Errorf("expected 1 (per-line), got %d", count)
	}
}

func TestGenerateProjectSnapshot(t *testing.T) {
	dir := t.TempDir()

	// Initialize a git repo for git log
	createTestFile(t, dir, "main.go", "package main\n// TODO: implement\nfunc main() {}\n")
	createTestFile(t, dir, "lib.go", "package main\nfunc helper() {}\n")

	autoDir := filepath.Join(dir, AutoDir)
	if err := os.MkdirAll(autoDir, 0755); err != nil {
		t.Fatalf("failed to create auto dir: %v", err)
	}

	err := GenerateProjectSnapshot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshotPath := filepath.Join(dir, AutoDir, AutoSnapshotFile)
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("failed to read snapshot: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "# Project Snapshot") {
		t.Error("missing header")
	}
	if !strings.Contains(content, "main.go") {
		t.Error("missing main.go in snapshot")
	}
	if !strings.Contains(content, "Test Gaps") {
		t.Error("missing test gaps section")
	}
	if !strings.Contains(content, "TODO/FIXME/HACK") {
		t.Error("missing TODO section")
	}
}

func TestShouldSkipDir_Snapshot(t *testing.T) {
	tests := []struct {
		name string
		skip bool
	}{
		{".git", true},
		{".claude", true},
		{"vendor", true},
		{"node_modules", true},
		{"bin", true},
		{"internal", false},
		{"cmd", false},
		{"src", false},
	}

	for _, tt := range tests {
		if got := shouldSkipDir(tt.name); got != tt.skip {
			t.Errorf("shouldSkipDir(%q) = %v, want %v", tt.name, got, tt.skip)
		}
	}
}

func TestIsSourceFile(t *testing.T) {
	tests := []struct {
		name   string
		source bool
	}{
		{"main.go", true},
		{"app.py", true},
		{"index.js", true},
		{"README.md", false},
		{"go.mod", false},
		{"Makefile", false},
	}

	for _, tt := range tests {
		if got := isSourceFile(tt.name); got != tt.source {
			t.Errorf("isSourceFile(%q) = %v, want %v", tt.name, got, tt.source)
		}
	}
}

// createTestFile creates a file with directories as needed.
func createTestFile(t *testing.T, base, rel, content string) {
	t.Helper()
	path := filepath.Join(base, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", rel, err)
	}
}
