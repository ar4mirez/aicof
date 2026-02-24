package core

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGetDefaultPromptTemplate(t *testing.T) {
	template := GetDefaultPromptTemplate()

	if template == "" {
		t.Fatal("expected non-empty template")
	}

	requiredSections := []string{
		"# Autonomous Iteration Prompt",
		"## Context Efficiency",
		"## Your Task",
		"## Rules",
		"## Error Recovery",
		"task-context.md",
		"progress-context.md",
		"Implement the task",
		"Run quality checks",
		"Commit changes",
		"Update state",
		"Document learnings",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("template missing required section: %q", section)
		}
	}
}

func TestGetDefaultPromptTemplate_Idempotent(t *testing.T) {
	first := GetDefaultPromptTemplate()
	second := GetDefaultPromptTemplate()
	if first != second {
		t.Error("expected GetDefaultPromptTemplate to return identical results on successive calls")
	}
}

func TestGeneratePromptFile(t *testing.T) {
	tests := []struct {
		name           string
		config         AutoConfig
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "basic config",
			config: AutoConfig{
				AITool:        "claude",
				MaxIterations: 10,
			},
			wantContains: []string{
				"# Autonomous Iteration Prompt",
				"## Project-Specific Configuration",
				"- **AI Tool**: claude",
				"- **Max Iterations**: 10",
				"- **PRD File**: " + filepath.Join(AutoDir, AutoPRDFile),
				"- **Progress File**: " + filepath.Join(AutoDir, AutoProgressFile),
			},
			wantNotContain: []string{
				"### Quality Checks",
				"## Pilot Mode Note",
			},
		},
		{
			name: "with quality checks",
			config: AutoConfig{
				AITool:        "codex",
				MaxIterations: 5,
				QualityChecks: []string{"go test ./...", "go vet ./..."},
			},
			wantContains: []string{
				"- **AI Tool**: codex",
				"- **Max Iterations**: 5",
				"### Quality Checks",
				"```bash",
				"go test ./...",
				"go vet ./...",
			},
			wantNotContain: []string{
				"## Pilot Mode Note",
			},
		},
		{
			name: "with pilot mode",
			config: AutoConfig{
				AITool:        "claude",
				MaxIterations: 30,
				PilotMode:     true,
			},
			wantContains: []string{
				"## Pilot Mode Note",
				"pilot mode",
				"auto-discovered",
			},
			wantNotContain: []string{
				"### Quality Checks",
			},
		},
		{
			name: "with quality checks and pilot mode",
			config: AutoConfig{
				AITool:        "claude",
				MaxIterations: 20,
				QualityChecks: []string{"npm test", "npm run lint"},
				PilotMode:     true,
			},
			wantContains: []string{
				"- **AI Tool**: claude",
				"- **Max Iterations**: 20",
				"### Quality Checks",
				"npm test",
				"npm run lint",
				"## Pilot Mode Note",
			},
		},
		{
			name: "empty quality checks slice",
			config: AutoConfig{
				AITool:        "claude",
				MaxIterations: 1,
				QualityChecks: []string{},
			},
			wantNotContain: []string{
				"### Quality Checks",
				"```bash",
			},
		},
		{
			name: "zero max iterations",
			config: AutoConfig{
				AITool:        "claude",
				MaxIterations: 0,
			},
			wantContains: []string{
				"- **Max Iterations**: 0",
			},
		},
		{
			name: "empty AI tool",
			config: AutoConfig{
				AITool:        "",
				MaxIterations: 10,
			},
			wantContains: []string{
				"- **AI Tool**: \n",
			},
		},
		{
			name: "single quality check",
			config: AutoConfig{
				AITool:        "claude",
				MaxIterations: 10,
				QualityChecks: []string{"make test"},
			},
			wantContains: []string{
				"### Quality Checks",
				"make test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneratePromptFile(tt.config)

			if result == "" {
				t.Fatal("expected non-empty result")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("result missing expected content: %q", want)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(result, notWant) {
					t.Errorf("result should not contain: %q", notWant)
				}
			}
		})
	}
}

func TestGeneratePromptFile_StartsWithTemplate(t *testing.T) {
	template := GetDefaultPromptTemplate()
	result := GeneratePromptFile(AutoConfig{
		AITool:        "claude",
		MaxIterations: 10,
	})

	if !strings.HasPrefix(result, template) {
		t.Error("GeneratePromptFile should start with the default template")
	}
}

func TestGeneratePromptFile_QualityChecksOrder(t *testing.T) {
	checks := []string{"first", "second", "third"}
	result := GeneratePromptFile(AutoConfig{
		AITool:        "claude",
		MaxIterations: 10,
		QualityChecks: checks,
	})

	firstIdx := strings.Index(result, "first")
	secondIdx := strings.Index(result, "second")
	thirdIdx := strings.Index(result, "third")

	if firstIdx >= secondIdx || secondIdx >= thirdIdx {
		t.Errorf("quality checks should appear in order: first=%d, second=%d, third=%d",
			firstIdx, secondIdx, thirdIdx)
	}
}
