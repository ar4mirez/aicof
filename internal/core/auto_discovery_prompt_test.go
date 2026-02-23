package core

import (
	"strings"
	"testing"
)

func TestGetDiscoveryPromptTemplate_RequiredSections(t *testing.T) {
	tmpl := GetDiscoveryPromptTemplate()

	required := []string{
		"Discovery Iteration Prompt",
		"DISCOVERY mode",
		"Do NOT write any code",
		"prd.json",
		"progress.md",
		"progress-context.md",
		"Priority Order",
		"pilot-discovery",
		"atomic",
	}

	for _, section := range required {
		if !strings.Contains(tmpl, section) {
			t.Errorf("template missing required content: %q", section)
		}
	}
}

func TestGenerateDiscoveryPrompt_IncludesFocus(t *testing.T) {
	pilot := &PilotConfig{
		MaxDiscoveryTasks: 8,
		Focus:             "testing",
	}
	config := AutoConfig{
		QualityChecks: []string{"go test ./..."},
	}

	prompt := GenerateDiscoveryPrompt(config, pilot)

	if !strings.Contains(prompt, "Focus Area: testing") {
		t.Error("expected focus area section for testing")
	}
	if !strings.Contains(prompt, "test coverage") {
		t.Error("expected testing-specific guidance")
	}
	if !strings.Contains(prompt, "8") {
		t.Error("expected max tasks constraint")
	}
}

func TestGenerateDiscoveryPrompt_NoFocus(t *testing.T) {
	pilot := &PilotConfig{
		MaxDiscoveryTasks: 10,
	}
	config := AutoConfig{}

	prompt := GenerateDiscoveryPrompt(config, pilot)

	if strings.Contains(prompt, "Focus Area:") {
		t.Error("should not contain focus section when no focus set")
	}
}

func TestGenerateDiscoveryPrompt_QualityChecks(t *testing.T) {
	pilot := &PilotConfig{MaxDiscoveryTasks: 10}
	config := AutoConfig{
		QualityChecks: []string{"go test ./...", "go vet ./..."},
	}

	prompt := GenerateDiscoveryPrompt(config, pilot)

	if !strings.Contains(prompt, "go test ./...") {
		t.Error("expected quality check command in prompt")
	}
	if !strings.Contains(prompt, "go vet ./...") {
		t.Error("expected quality check command in prompt")
	}
}

func TestGenerateDiscoveryPrompt_NilPilot(t *testing.T) {
	config := AutoConfig{}

	prompt := GenerateDiscoveryPrompt(config, nil)

	if !strings.Contains(prompt, "Discovery Iteration Prompt") {
		t.Error("expected base template even with nil pilot")
	}
	if strings.Contains(prompt, "Discovery Configuration") {
		t.Error("should not have configuration section with nil pilot")
	}
}

func TestGenerateFocusSection_AllTypes(t *testing.T) {
	focuses := []struct {
		focus    string
		contains string
	}{
		{"testing", "test coverage"},
		{"docs", "documentation"},
		{"documentation", "documentation"},
		{"security", "input validation"},
		{"performance", "hot paths"},
		{"refactoring", "code duplication"},
		{"custom-focus", "custom-focus"},
	}

	for _, tt := range focuses {
		t.Run(tt.focus, func(t *testing.T) {
			section := generateFocusSection(tt.focus)
			if !strings.Contains(section, tt.contains) {
				t.Errorf("focus=%s: expected %q in output", tt.focus, tt.contains)
			}
		})
	}
}
