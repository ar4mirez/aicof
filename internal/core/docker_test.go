package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetSupportedSandboxModes(t *testing.T) {
	modes := GetSupportedSandboxModes()
	if len(modes) != 3 {
		t.Fatalf("expected 3 modes, got %d", len(modes))
	}
	if modes[0] != SandboxNone {
		t.Errorf("expected first mode %q, got %q", SandboxNone, modes[0])
	}
	if modes[1] != SandboxDocker {
		t.Errorf("expected second mode %q, got %q", SandboxDocker, modes[1])
	}
	if modes[2] != SandboxDockerSandbox {
		t.Errorf("expected third mode %q, got %q", SandboxDockerSandbox, modes[2])
	}
}

func TestIsValidSandboxMode(t *testing.T) {
	tests := []struct {
		mode string
		want bool
	}{
		{"none", true},
		{"docker", true},
		{"docker-sandbox", true},
		{"podman", false},
		{"", false},
		{"DOCKER", true},
		{"Docker", true},
		{"DOCKER-SANDBOX", true},
		{"Docker-Sandbox", true},
		{"kubernetes", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("mode=%q", tt.mode), func(t *testing.T) {
			got := IsValidSandboxMode(tt.mode)
			if got != tt.want {
				t.Errorf("IsValidSandboxMode(%q) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}

func TestBuildDockerSandboxArgs(t *testing.T) {
	tests := []struct {
		name     string
		config   DockerSandboxRunConfig
		wantArgs []string
	}{
		{
			name: "basic claude sandbox",
			config: DockerSandboxRunConfig{
				Agent:   "claude",
				WorkDir: "/home/user/project",
			},
			wantArgs: []string{
				"sandbox", "run", "claude", "/home/user/project",
			},
		},
		{
			name: "with template",
			config: DockerSandboxRunConfig{
				Agent:    "claude",
				WorkDir:  "/home/user/project",
				Template: "python:3-alpine",
			},
			wantArgs: []string{
				"sandbox", "run", "--template", "python:3-alpine",
				"claude", "/home/user/project",
			},
		},
		{
			name: "with name",
			config: DockerSandboxRunConfig{
				Agent:   "claude",
				WorkDir: "/home/user/project",
				Name:    "my-sandbox",
			},
			wantArgs: []string{
				"sandbox", "run", "--name", "my-sandbox",
				"claude", "/home/user/project",
			},
		},
		{
			name: "with agent args",
			config: DockerSandboxRunConfig{
				Agent:   "claude",
				WorkDir: "/home/user/project",
				AgentArgs: []string{
					"--print", "--dangerously-skip-permissions",
					"/home/user/project/.claude/auto/prompt.md",
				},
			},
			wantArgs: []string{
				"sandbox", "run", "claude", "/home/user/project",
				"--",
				"--print", "--dangerously-skip-permissions",
				"/home/user/project/.claude/auto/prompt.md",
			},
		},
		{
			name: "default agent when empty",
			config: DockerSandboxRunConfig{
				Agent:   "",
				WorkDir: "/tmp/proj",
			},
			wantArgs: []string{
				"sandbox", "run", DefaultDockerSandboxAgent, "/tmp/proj",
			},
		},
		{
			name: "default workdir when empty",
			config: DockerSandboxRunConfig{
				Agent:   "claude",
				WorkDir: "",
			},
			wantArgs: []string{
				"sandbox", "run", "claude", ".",
			},
		},
		{
			name: "with name and template and args",
			config: DockerSandboxRunConfig{
				Agent:     "codex",
				WorkDir:   "/proj",
				Name:      "test-run",
				Template:  "node:20",
				AgentArgs: []string{"exec", "--full-auto", "do the work"},
			},
			wantArgs: []string{
				"sandbox", "run", "--name", "test-run",
				"--template", "node:20",
				"codex", "/proj",
				"--", "exec", "--full-auto", "do the work",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildDockerSandboxArgs(tt.config)
			if len(args) != len(tt.wantArgs) {
				t.Fatalf("got %d args %v, want %d args %v",
					len(args), args, len(tt.wantArgs), tt.wantArgs)
			}
			for i, got := range args {
				if got != tt.wantArgs[i] {
					t.Errorf("arg[%d] = %q, want %q", i, got, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestGetAgentArgs_Claude(t *testing.T) {
	// Claude reads the prompt file and passes content as -p arg
	promptFile := filepath.Join(t.TempDir(), "prompt.md")
	if err := os.WriteFile(promptFile, []byte("do the work"), 0644); err != nil {
		t.Fatal(err)
	}

	args, err := GetAgentArgs("claude", promptFile)
	if err != nil {
		t.Fatalf("GetAgentArgs claude: %v", err)
	}

	wantArgs := []string{"-p", "do the work", "--dangerously-skip-permissions"}
	if len(args) != len(wantArgs) {
		t.Fatalf("got %d args %v, want %d args %v",
			len(args), args, len(wantArgs), wantArgs)
	}
	for i, got := range args {
		if got != wantArgs[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got, wantArgs[i])
		}
	}
}

func TestGetAgentArgs_Claude_MissingFile(t *testing.T) {
	_, err := GetAgentArgs("claude", "/nonexistent/prompt.md")
	if err == nil {
		t.Error("expected error for missing prompt file, got nil")
	}
}

func TestGetAgentArgs_OtherTools(t *testing.T) {
	tests := []struct {
		aiTool     string
		promptPath string
		wantArgs   []string
	}{
		{
			"codex",
			"/path/prompt.md",
			[]string{"exec", "--full-auto", "write code"},
		},
		{
			"amp",
			"/path/prompt.md",
			[]string{"--prompt-file", "/path/prompt.md"},
		},
		{
			"unknown-tool",
			"/path/prompt.md",
			[]string{"/path/prompt.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.aiTool, func(t *testing.T) {
			promptPath := tt.promptPath
			if tt.aiTool == "codex" {
				promptPath = filepath.Join(t.TempDir(), "prompt.md")
				if err := os.WriteFile(promptPath, []byte("write code"), 0644); err != nil {
					t.Fatalf("failed to write codex prompt file: %v", err)
				}
			}
			args, err := GetAgentArgs(tt.aiTool, promptPath)
			if err != nil {
				t.Fatalf("GetAgentArgs %s: %v", tt.aiTool, err)
			}
			if len(args) != len(tt.wantArgs) {
				t.Fatalf("got %d args %v, want %d args %v",
					len(args), args, len(tt.wantArgs), tt.wantArgs)
			}
			for i, got := range args {
				if got != tt.wantArgs[i] {
					t.Errorf("arg[%d] = %q, want %q", i, got, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestGetAIToolEnvVars(t *testing.T) {
	// Clear all tracked env vars first
	for _, name := range aiToolEnvVarNames {
		os.Unsetenv(name)
	}

	t.Setenv("ANTHROPIC_API_KEY", "sk-test-123")
	t.Setenv("AI_TOOL", "claude")

	envArgs := getAIToolEnvVars()

	// Should contain -e pairs for the two vars we set
	joined := strings.Join(envArgs, " ")
	if !strings.Contains(joined, "-e ANTHROPIC_API_KEY=sk-test-123") {
		t.Errorf("expected ANTHROPIC_API_KEY in env args: %v", envArgs)
	}
	if !strings.Contains(joined, "-e AI_TOOL=claude") {
		t.Errorf("expected AI_TOOL in env args: %v", envArgs)
	}

	// Should NOT contain vars we didn't set
	if strings.Contains(joined, "OPENAI_API_KEY") {
		t.Errorf("did not expect OPENAI_API_KEY in env args: %v", envArgs)
	}
}

func TestGetAIToolEnvVars_NoHomeOrPath(t *testing.T) {
	// HOME and PATH must NOT be forwarded — they leak host paths into
	// the container where they don't exist.
	for _, name := range aiToolEnvVarNames {
		os.Unsetenv(name)
	}

	// Set HOME and PATH to verify they are NOT in the forwarded list
	t.Setenv("HOME", "/Users/testuser")
	t.Setenv("PATH", "/usr/local/bin:/usr/bin")
	t.Setenv("ANTHROPIC_API_KEY", "sk-test")

	envArgs := getAIToolEnvVars()
	joined := strings.Join(envArgs, " ")

	if strings.Contains(joined, "HOME=") {
		t.Errorf("HOME should NOT be forwarded to container: %v", envArgs)
	}
	if strings.Contains(joined, "PATH=") {
		t.Errorf("PATH should NOT be forwarded to container: %v", envArgs)
	}
}

func TestCheckDockerAvailable(t *testing.T) {
	// Only run if docker is available on the test machine
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available, skipping")
	}

	err := CheckDockerAvailable()
	if err != nil {
		t.Logf("CheckDockerAvailable returned: %v (daemon may not be running)", err)
	}
}

func TestCheckDockerSandboxAvailable(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available, skipping")
	}

	err := CheckDockerSandboxAvailable()
	if err != nil {
		t.Logf("CheckDockerSandboxAvailable returned: %v (sandbox plugin may not be installed)", err)
	}
}

func TestIsValidSandboxImage(t *testing.T) {
	tests := []struct {
		image string
		want  bool
	}{
		// Valid images
		{"node:lts", true},
		{"ubuntu:22.04", true},
		{"python:3.12-slim", true},
		{"alpine", true},
		{"ghcr.io/owner/image:latest", true},
		{"registry.example.com/org/image:v1.2.3", true},
		{"my-image_name:tag-1.0", true},
		{"node:lts@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789", true},
		{"docker.io/library/node:20", true},

		// Invalid images
		{"", false},
		{"/bin/sh", false},
		{"./malicious", false},
		{"../escape", false},
		{"image;rm -rf /", false},
		{"image$(whoami)", false},
		{"image`id`", false},
		{"image|cat /etc/passwd", false},
		{"image&malicious", false},
		{"image name with spaces", false},
		{".hidden-image", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("image=%q", tt.image), func(t *testing.T) {
			got := IsValidSandboxImage(tt.image)
			if got != tt.want {
				t.Errorf("IsValidSandboxImage(%q) = %v, want %v",
					tt.image, got, tt.want)
			}
		})
	}
}
