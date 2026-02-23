package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProgressContextConfigFromAuto(t *testing.T) {
	tests := []struct {
		name string
		cfg  AutoConfig
		want ProgressContextConfig
	}{
		{
			name: "zero values use defaults",
			cfg:  AutoConfig{},
			want: ProgressContextConfig{
				MaxLearnings:     DefaultMaxContextLearnings,
				MaxCompleted:     DefaultMaxContextCompleted,
				MaxProgressLines: DefaultMaxProgressLines,
			},
		},
		{
			name: "custom values override",
			cfg: AutoConfig{
				ProgressMaxLearnings: 20,
				ProgressMaxCompleted: 5,
				ProgressMaxLines:     300,
			},
			want: ProgressContextConfig{MaxLearnings: 20, MaxCompleted: 5, MaxProgressLines: 300},
		},
		{
			name: "partial override",
			cfg:  AutoConfig{ProgressMaxLearnings: 30},
			want: ProgressContextConfig{
				MaxLearnings:     30,
				MaxCompleted:     DefaultMaxContextCompleted,
				MaxProgressLines: DefaultMaxProgressLines,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProgressContextConfigFromAuto(tt.cfg)
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGenerateProgressContext_EmptyAndMissing(t *testing.T) {
	dir := t.TempDir()
	contextPath := filepath.Join(dir, "progress-context.md")

	// Missing file produces empty context
	if err := GenerateProgressContext(filepath.Join(dir, "nonexistent.md"), contextPath, DefaultProgressContextConfig()); err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	data, _ := os.ReadFile(contextPath)
	if !strings.Contains(string(data), "No prior progress") {
		t.Error("expected 'No prior progress' message")
	}

	// Empty file produces zero-count context
	progressPath := filepath.Join(dir, "progress.md")
	if err := os.WriteFile(progressPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := GenerateProgressContext(progressPath, contextPath, DefaultProgressContextConfig()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ = os.ReadFile(contextPath)
	if !strings.Contains(string(data), "0 iterations completed") {
		t.Error("expected zero iterations in empty context")
	}
}

func TestGenerateProgressContext_ExtractsLearnings(t *testing.T) {
	dir := t.TempDir()
	progressPath := filepath.Join(dir, "progress.md")
	contextPath := filepath.Join(dir, "progress-context.md")

	content := "[ts] [iteration:1] [task:1] COMPLETED: Did thing\n" +
		"[ts] [iteration:1] LEARNING: Important insight\n" +
		"[ts] [iteration:2] [task:2] STARTED: Another thing\n" +
		"[ts] [iteration:2] LEARNING: Second insight\n"
	if err := os.WriteFile(progressPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := GenerateProgressContext(progressPath, contextPath, DefaultProgressContextConfig()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(contextPath)
	ctx := string(data)
	for _, want := range []string{"Important insight", "Second insight", "Key Learnings"} {
		if !strings.Contains(ctx, want) {
			t.Errorf("expected %q in context", want)
		}
	}
}

func TestGenerateProgressContext_ExtractsCompleted(t *testing.T) {
	dir := t.TempDir()
	progressPath := filepath.Join(dir, "progress.md")
	contextPath := filepath.Join(dir, "progress-context.md")
	content := "[ts] [iteration:1] [task:1] COMPLETED: First task\n" +
		"[ts] [iteration:2] [task:2] COMPLETED: Second task\n" +
		"[ts] [iteration:3] [task:3] COMPLETED: Third task\n"
	if err := os.WriteFile(progressPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := GenerateProgressContext(progressPath, contextPath, DefaultProgressContextConfig()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(contextPath)
	ctx := string(data)
	if !strings.Contains(ctx, "3 tasks done") {
		t.Error("expected 3 tasks done in summary")
	}
	if !strings.Contains(ctx, "Recent Completions") {
		t.Error("expected Recent Completions section")
	}
}

func TestGenerateProgressContext_RespectsMaxLimits(t *testing.T) {
	dir := t.TempDir()
	progressPath := filepath.Join(dir, "progress.md")
	contextPath := filepath.Join(dir, "progress-context.md")
	var sb strings.Builder
	for i := 1; i <= 20; i++ {
		sb.WriteString("[ts] [iteration:1] LEARNING: Insight number ")
		sb.WriteString(strings.Repeat("X", i))
		sb.WriteString("\n")
	}
	if err := os.WriteFile(progressPath, []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := ProgressContextConfig{MaxLearnings: 5, MaxCompleted: 2, MaxProgressLines: 500}
	if err := GenerateProgressContext(progressPath, contextPath, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(contextPath)
	ctx := string(data)
	if count := strings.Count(ctx, "LEARNING:"); count != 5 {
		t.Errorf("expected 5 learnings, got %d", count)
	}
	if !strings.Contains(ctx, strings.Repeat("X", 20)) {
		t.Error("expected last learnings to be kept, not first")
	}
}

func TestExtractEntries(t *testing.T) {
	lines := []string{
		"[ts] [iteration:1] LEARNING: First",
		"[ts] [iteration:1] COMPLETED: Done one",
		"[ts] [iteration:2] LEARNING: Second",
		"[ts] [iteration:2] STARTED: Begin",
		"[ts] [iteration:3] LEARNING: Third",
	}
	tests := []struct {
		name   string
		marker string
		max    int
		want   int
	}{
		{"all learnings", "LEARNING:", 10, 3},
		{"limited learnings", "LEARNING:", 2, 2},
		{"completed", "COMPLETED:", 10, 1},
		{"no match", "ERROR:", 10, 0},
		{"zero max returns all", "LEARNING:", 0, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractEntries(lines, tt.marker, tt.max); len(got) != tt.want {
				t.Errorf("expected %d entries, got %d", tt.want, len(got))
			}
		})
	}
}

func TestCountProgressStats(t *testing.T) {
	tests := []struct {
		name       string
		lines      []string
		wantIters  int
		wantCompls int
	}{
		{"empty", nil, 0, 0},
		{
			"mixed entries",
			[]string{
				"[ts] [iteration:1] [task:1] COMPLETED: One",
				"[ts] [iteration:1] [task:2] COMPLETED: Two",
				"[ts] [iteration:2] [task:3] COMPLETED: Three",
				"[ts] [iteration:3] LEARNING: Something",
			},
			3, 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iters, completions := countProgressStats(tt.lines)
			if iters != tt.wantIters {
				t.Errorf("iterations: got %d, want %d", iters, tt.wantIters)
			}
			if completions != tt.wantCompls {
				t.Errorf("completions: got %d, want %d", completions, tt.wantCompls)
			}
		})
	}
}

func TestRotateProgressIfNeeded_BelowThreshold(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.md")
	content := "line1\nline2\nline3\n"
	os.WriteFile(path, []byte(content), 0644)
	if err := RotateProgressIfNeeded(path, 100); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != content {
		t.Error("file should not be modified when below threshold")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "progress-archive-*.md"))
	if len(matches) != 0 {
		t.Error("no archive should be created")
	}
}

func TestRotateProgressIfNeeded_AboveThreshold(t *testing.T) {
	dir := t.TempDir()
	progressPath := filepath.Join(dir, "progress.md")
	var sb strings.Builder
	for i := 1; i <= 100; i++ {
		sb.WriteString("line\n")
	}
	if err := os.WriteFile(progressPath, []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RotateProgressIfNeeded(progressPath, 20); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(progressPath)
	remaining := strings.Count(strings.TrimSpace(string(data)), "\n") + 1
	if remaining != 10 {
		t.Errorf("expected 10 remaining lines, got %d", remaining)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "progress-archive-*.md"))
	if len(matches) != 1 {
		t.Fatalf("expected 1 archive file, got %d", len(matches))
	}
	archiveData, _ := os.ReadFile(matches[0])
	archiveLines := strings.Count(strings.TrimSpace(string(archiveData)), "\n") + 1
	if archiveLines != 90 {
		t.Errorf("expected 90 archived lines, got %d", archiveLines)
	}
}

func TestRotateProgressIfNeeded_MissingFile(t *testing.T) {
	if err := RotateProgressIfNeeded("/nonexistent/progress.md", 100); err != nil {
		t.Errorf("expected nil error for missing file, got: %v", err)
	}
}

func TestPrepareProgressContext(t *testing.T) {
	dir := t.TempDir()
	autoDir := filepath.Join(dir, AutoDir)
	if err := os.MkdirAll(autoDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := "[ts] [iteration:1] LEARNING: Test insight\n" +
		"[ts] [iteration:1] [task:1] COMPLETED: Test task\n"
	if err := os.WriteFile(filepath.Join(autoDir, AutoProgressFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	PrepareProgressContext(dir)

	data, err := os.ReadFile(filepath.Join(autoDir, AutoProgressContextFile))
	if err != nil {
		t.Fatalf("context file should be created: %v", err)
	}
	ctx := string(data)
	if !strings.Contains(ctx, "Test insight") {
		t.Error("expected learning in context")
	}
	if !strings.Contains(ctx, "Test task") {
		t.Error("expected completion in context")
	}
}
