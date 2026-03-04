package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLoopConfig_Defaults(t *testing.T) {
	dir := t.TempDir()
	prd := NewAutoPRD("test", "test project")
	prd.Config.AITool = "claude"
	prd.Config.Sandbox = SandboxNone

	prdPath := filepath.Join(dir, AutoDir, AutoPRDFile)
	if err := prd.Save(prdPath); err != nil {
		t.Fatalf("failed to save prd: %v", err)
	}

	cfg := NewLoopConfig(dir, prd)

	if cfg.ProjectDir != dir {
		t.Errorf("expected ProjectDir=%s, got=%s", dir, cfg.ProjectDir)
	}
	if cfg.AITool != "claude" {
		t.Errorf("expected AITool=claude, got=%s", cfg.AITool)
	}
	if cfg.MaxIterations != 50 {
		t.Errorf("expected MaxIterations=50, got=%d", cfg.MaxIterations)
	}
	if cfg.PauseSecs != 2 {
		t.Errorf("expected PauseSecs=2, got=%d", cfg.PauseSecs)
	}
	if cfg.MaxConsecFails != 3 {
		t.Errorf("expected MaxConsecFails=3, got=%d", cfg.MaxConsecFails)
	}
}

func TestNewLoopConfig_EnvOverrides(t *testing.T) {
	os.Setenv("PAUSE_SECONDS", "5")
	os.Setenv("MAX_CONSECUTIVE_FAILURES", "7")
	defer os.Unsetenv("PAUSE_SECONDS")
	defer os.Unsetenv("MAX_CONSECUTIVE_FAILURES")

	dir := t.TempDir()
	prd := NewAutoPRD("test", "test project")

	cfg := NewLoopConfig(dir, prd)

	if cfg.PauseSecs != 5 {
		t.Errorf("expected PauseSecs=5, got=%d", cfg.PauseSecs)
	}
	if cfg.MaxConsecFails != 7 {
		t.Errorf("expected MaxConsecFails=7, got=%d", cfg.MaxConsecFails)
	}
}

func TestRunAutoLoop_AllTasksCompleted(t *testing.T) {
	dir := t.TempDir()
	prd := NewAutoPRD("test", "test project")
	// No tasks means GetNextTask returns nil -> immediate exit
	prdPath := filepath.Join(dir, AutoDir, AutoPRDFile)
	if err := prd.Save(prdPath); err != nil {
		t.Fatalf("failed to save prd: %v", err)
	}

	cfg := LoopConfig{
		ProjectDir:     dir,
		PRDPath:        prdPath,
		MaxIterations:  10,
		MaxConsecFails: 3,
	}

	err := RunAutoLoop(cfg)
	if err != nil {
		t.Errorf("expected no error when all tasks done, got: %v", err)
	}
}

func TestRunAutoLoop_BadPRDPath(t *testing.T) {
	cfg := LoopConfig{
		ProjectDir:     t.TempDir(),
		PRDPath:        "/nonexistent/prd.json",
		MaxIterations:  1,
		MaxConsecFails: 3,
	}

	err := RunAutoLoop(cfg)
	if err == nil {
		t.Error("expected error for non-existent PRD path")
	}
}

func TestNotifyCallbacks(t *testing.T) {
	startCalled := false
	endCalled := false

	notifyIterStart(func(iter int, iterType string) {
		startCalled = true
		if iter != 1 {
			t.Errorf("expected iter=1, got=%d", iter)
		}
		if iterType != IterationTypeImplementation {
			t.Errorf("expected type=implementation, got=%s", iterType)
		}
	}, 1, IterationTypeImplementation)

	notifyIterEnd(func(iter int, err error) {
		endCalled = true
	}, 1, nil)

	if !startCalled {
		t.Error("OnIterStart callback was not called")
	}
	if !endCalled {
		t.Error("OnIterEnd callback was not called")
	}
}

func TestNotifyCallbacks_NilSafe(t *testing.T) {
	// Should not panic with nil callbacks
	notifyIterStart(nil, 1, IterationTypeImplementation)
	notifyIterEnd(nil, 1, nil)
}

func TestInvokeAgent_RejectsInvalidAITool(t *testing.T) {
	tests := []struct {
		name   string
		aiTool string
	}{
		{"arbitrary binary", "/bin/sh"},
		{"path traversal", "../../../malicious"},
		{"empty string", ""},
		{"shell injection", "claude; rm -rf /"},
		{"unknown tool", "unsupported-tool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := LoopConfig{
				ProjectDir: t.TempDir(),
				AITool:     tt.aiTool,
			}
			err := InvokeAgent(cfg)
			if err == nil {
				t.Errorf("expected error for invalid AI tool %q", tt.aiTool)
			}
		})
	}
}

func TestInvokeAgent_AcceptsValidTools(t *testing.T) {
	for _, tool := range GetSupportedAITools() {
		t.Run(tool, func(t *testing.T) {
			cfg := LoopConfig{
				ProjectDir: t.TempDir(),
				AITool:     tool,
				PromptPath: "/nonexistent/prompt.md",
			}
			err := InvokeAgent(cfg)
			// Should fail with prompt/exec error, NOT with invalid tool error
			if err != nil && err.Error() == fmt.Sprintf(
				"refused to invoke invalid AI tool %q (allowed: %v)",
				tool, GetSupportedAITools()) {
				t.Errorf("valid tool %q was rejected", tool)
			}
		})
	}
}

func TestInvokeAgent_DispatchesByMode(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		sandbox string
		aiTool  string
		prompt  string
		wantErr string
	}{
		{
			name:    "default dispatches to local",
			sandbox: "",
			aiTool:  "claude",
			prompt:  "/nonexistent/prompt.md",
			// invokeAgentLocal → GetAgentArgs("claude", ...) fails reading prompt
			wantErr: "failed to build agent args",
		},
		{
			name:    "none dispatches to local",
			sandbox: SandboxNone,
			aiTool:  "claude",
			prompt:  "/nonexistent/prompt.md",
			wantErr: "failed to build agent args",
		},
		{
			name:    "docker dispatches to invokeAgentDocker",
			sandbox: SandboxDocker,
			aiTool:  "claude",
			prompt:  filepath.Join(dir, "nonexistent", "prompt.md"),
			// invokeAgentDocker → filepath.Rel OK → GetAgentArgs("claude",...) fails
			wantErr: "failed to build agent args",
		},
		{
			name:    "docker-sandbox dispatches to invokeAgentDockerSandbox",
			sandbox: SandboxDockerSandbox,
			aiTool:  "claude",
			prompt:  "/nonexistent/prompt.md",
			// invokeAgentDockerSandbox → GetAgentArgs("claude",...) fails
			wantErr: "failed to build agent args",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := LoopConfig{
				ProjectDir: dir,
				AITool:     tt.aiTool,
				PromptPath: tt.prompt,
				Sandbox:    tt.sandbox,
			}
			err := InvokeAgent(cfg)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestInvokeAgentLocal_PromptReadError(t *testing.T) {
	cfg := LoopConfig{
		ProjectDir: t.TempDir(),
		AITool:     "claude", // claude reads prompt file
		PromptPath: "/nonexistent/prompt.md",
	}

	err := invokeAgentLocal(cfg)
	if err == nil {
		t.Fatal("expected error for missing prompt file")
	}
	if !strings.Contains(err.Error(), "failed to build agent args") {
		t.Errorf("expected 'failed to build agent args' error, got: %v", err)
	}
}

func TestInvokeAgentLocal_RunsCommand(t *testing.T) {
	dir := t.TempDir()
	promptFile := filepath.Join(dir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := LoopConfig{
		ProjectDir: dir,
		AITool:     "codex", // doesn't read prompt file
		PromptPath: promptFile,
	}

	err := invokeAgentLocal(cfg)
	// codex binary not in PATH, so cmd.Run should fail
	if err == nil {
		t.Skip("codex unexpectedly available in PATH")
	}
	// Error should be from exec, not from arg building
	if strings.Contains(err.Error(), "failed to build agent args") {
		t.Errorf("expected exec error, got arg building error: %v", err)
	}
}

func TestInvokeAgentDocker_RejectsInvalidImage(t *testing.T) {
	dir := t.TempDir()
	promptFile := filepath.Join(dir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		image string
	}{
		{"shell injection", "node:lts;rm -rf /"},
		{"absolute path", "/bin/malicious"},
		{"command substitution", "$(whoami)"},
		{"relative path escape", "../escape"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := LoopConfig{
				ProjectDir:   dir,
				AITool:       "codex",
				PromptPath:   promptFile,
				Sandbox:      SandboxDocker,
				SandboxImage: tt.image,
			}
			err := invokeAgentDocker(cfg)
			if err == nil {
				t.Fatalf("expected error for invalid image %q", tt.image)
			}
			if !strings.Contains(err.Error(), "invalid sandbox image") {
				t.Errorf("expected 'invalid sandbox image' error, got: %v", err)
			}
		})
	}
}

func TestInvokeAgentDocker_PromptReadError(t *testing.T) {
	dir := t.TempDir()
	// Prompt path must be inside project dir for filepath.Rel to succeed
	promptPath := filepath.Join(dir, "nonexistent", "prompt.md")

	cfg := LoopConfig{
		ProjectDir: dir,
		AITool:     "claude", // claude reads prompt file
		PromptPath: promptPath,
	}

	err := invokeAgentDocker(cfg)
	if err == nil {
		t.Fatal("expected error for missing prompt file")
	}
	if !strings.Contains(err.Error(), "failed to build agent args") {
		t.Errorf("expected 'failed to build agent args' error, got: %v", err)
	}
}

func TestInvokeAgentDocker_RunsDockerCommand(t *testing.T) {
	dir := t.TempDir()
	promptFile := filepath.Join(dir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := LoopConfig{
		ProjectDir:   dir,
		AITool:       "codex",
		PromptPath:   promptFile,
		SandboxImage: "nonexistent-image-test:0.0.0",
	}

	err := invokeAgentDocker(cfg)
	// Error expected: docker run will fail (no container, or docker not installed)
	if err == nil {
		t.Skip("docker unexpectedly succeeded")
	}
	// Should NOT be an arg building error — it should get past arg construction
	if strings.Contains(err.Error(), "failed to build agent args") {
		t.Errorf("expected docker exec error, got arg building error: %v", err)
	}
}

func TestInvokeAgentDockerSandbox_PromptReadError(t *testing.T) {
	cfg := LoopConfig{
		ProjectDir: t.TempDir(),
		AITool:     "claude",
		PromptPath: "/nonexistent/prompt.md",
	}

	err := invokeAgentDockerSandbox(cfg)
	if err == nil {
		t.Fatal("expected error for missing prompt file")
	}
	if !strings.Contains(err.Error(), "failed to build agent args") {
		t.Errorf("expected 'failed to build agent args' error, got: %v", err)
	}
}

func TestInvokeAgentDockerSandbox_RunsDockerCommand(t *testing.T) {
	dir := t.TempDir()
	promptFile := filepath.Join(dir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := LoopConfig{
		ProjectDir: dir,
		AITool:     "codex",
		PromptPath: promptFile,
		SandboxTpl: "nonexistent-test-template",
	}

	err := invokeAgentDockerSandbox(cfg)
	if err == nil {
		t.Skip("docker sandbox unexpectedly succeeded")
	}
	// Should NOT be an arg building error
	if strings.Contains(err.Error(), "failed to build agent args") {
		t.Errorf("expected docker exec error, got arg building error: %v", err)
	}
}

func TestBuildDockerRunArgs(t *testing.T) {
	// Clear all AI tool env vars so they don't pollute the output
	for _, name := range aiToolEnvVarNames {
		t.Setenv(name, "")
		os.Unsetenv(name)
	}

	tests := []struct {
		name      string
		workDir   string
		image     string
		aiTool    string
		agentArgs []string
		wantParts []string // Substrings that must appear in the args
	}{
		{
			name:      "basic invocation",
			workDir:   "/home/user/project",
			image:     "node:lts",
			aiTool:    "codex",
			agentArgs: []string{
				"--sandbox", "danger-full-access",
				"--ask-for-approval", "never",
				"exec", "do the work",
			},
			wantParts: []string{
				"run", "--rm", "--init", "-i",
				"-v", "/home/user/project:/workspace",
				"-w", "/workspace",
				"node:lts",
				"codex",
				"--sandbox", "danger-full-access",
				"--ask-for-approval", "never",
				"exec", "do the work",
			},
		},
		{
			name:      "custom image",
			workDir:   "/tmp/proj",
			image:     "python:3-slim",
			aiTool:    "amp",
			agentArgs: []string{"--prompt-file", "/workspace/prompt.md"},
			wantParts: []string{
				"python:3-slim",
				"amp", "--prompt-file",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildDockerRunArgs(tt.workDir, tt.image, tt.aiTool, tt.agentArgs)

			joined := strings.Join(args, " ")
			for _, part := range tt.wantParts {
				if !strings.Contains(joined, part) {
					t.Errorf("args %v missing expected part %q", args, part)
				}
			}

			// Verify structural invariants
			if args[0] != "run" {
				t.Errorf("first arg should be 'run', got %q", args[0])
			}
			if args[1] != "--rm" {
				t.Errorf("second arg should be '--rm', got %q", args[1])
			}
		})
	}
}

func TestBuildDockerRunArgs_Structure(t *testing.T) {
	// Clear env vars for predictable output
	for _, name := range aiToolEnvVarNames {
		t.Setenv(name, "")
		os.Unsetenv(name)
	}

	args := buildDockerRunArgs("/proj", "img:1", "claude", []string{"-p", "hello"})

	// Verify fixed structure: run --rm --init -i --user=UID:GID -v MOUNT -w /workspace IMAGE TOOL ARGS...
	if len(args) < 10 {
		t.Fatalf("expected at least 10 args, got %d: %v", len(args), args)
	}

	if args[0] != "run" || args[1] != "--rm" || args[2] != "--init" || args[3] != "-i" {
		t.Errorf("expected [run --rm --init -i], got %v", args[:4])
	}

	// --user=UID:GID
	if !strings.HasPrefix(args[4], "--user=") {
		t.Errorf("expected --user=UID:GID at arg[4], got %q", args[4])
	}

	// -v workDir:mount
	if args[5] != "-v" || args[6] != "/proj:"+DockerContainerMount {
		t.Errorf("expected volume mount, got args[5:7]=%v", args[5:7])
	}

	// -w mount
	if args[7] != "-w" || args[8] != DockerContainerMount {
		t.Errorf("expected workdir, got args[7:9]=%v", args[7:9])
	}

	// Image then tool then args
	if args[9] != "img:1" {
		t.Errorf("expected image at arg[9], got %q", args[9])
	}
	if args[10] != "claude" {
		t.Errorf("expected tool at arg[10], got %q", args[10])
	}
	if args[11] != "-p" || args[12] != "hello" {
		t.Errorf("expected agent args at [11:13], got %v", args[11:])
	}
}

func TestBuildDockerRunArgs_WithEnvVars(t *testing.T) {
	// Clear all env vars first
	for _, name := range aiToolEnvVarNames {
		os.Unsetenv(name)
	}

	t.Setenv("ANTHROPIC_API_KEY", "sk-test")
	t.Setenv("AI_TOOL", "claude")

	args := buildDockerRunArgs("/proj", "img:1", "claude", []string{"-p", "hi"})
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "-e ANTHROPIC_API_KEY=sk-test") {
		t.Errorf("expected ANTHROPIC_API_KEY env var in args: %v", args)
	}
	if !strings.Contains(joined, "-e AI_TOOL=claude") {
		t.Errorf("expected AI_TOOL env var in args: %v", args)
	}
}

func TestRunAutoLoop_ConsecutiveFailures(t *testing.T) {
	dir := t.TempDir()
	prd := NewAutoPRD("test", "test project")
	prd.Tasks = []AutoTask{
		{ID: "1", Title: "task 1", Status: TaskStatusPending},
		{ID: "2", Title: "task 2", Status: TaskStatusPending},
		{ID: "3", Title: "task 3", Status: TaskStatusPending},
	}
	prdPath := filepath.Join(dir, AutoDir, AutoPRDFile)
	if err := prd.Save(prdPath); err != nil {
		t.Fatalf("failed to save prd: %v", err)
	}

	// Use "codex" — valid tool but not installed, so InvokeAgent fails each time
	cfg := LoopConfig{
		ProjectDir:     dir,
		PRDPath:        prdPath,
		AITool:         "codex",
		PromptPath:     filepath.Join(dir, "prompt.md"),
		MaxIterations:  10,
		MaxConsecFails: 2,
		PauseSecs:      0,
	}

	err := RunAutoLoop(cfg)
	if err == nil {
		t.Fatal("expected consecutive failure error")
	}
	if !strings.Contains(err.Error(), "consecutive failures") {
		t.Errorf("expected 'consecutive failures' error, got: %v", err)
	}
}

func TestRunAutoLoop_CallbacksInvoked(t *testing.T) {
	dir := t.TempDir()
	prd := NewAutoPRD("test", "test project")
	prd.Tasks = []AutoTask{
		{ID: "1", Title: "task 1", Status: TaskStatusPending},
	}
	prdPath := filepath.Join(dir, AutoDir, AutoPRDFile)
	if err := prd.Save(prdPath); err != nil {
		t.Fatalf("failed to save prd: %v", err)
	}

	var startIters []int
	var endIters []int
	var endErrors []error

	cfg := LoopConfig{
		ProjectDir:     dir,
		PRDPath:        prdPath,
		AITool:         "codex",
		PromptPath:     filepath.Join(dir, "prompt.md"),
		MaxIterations:  1,
		MaxConsecFails: 5, // Won't trigger
		PauseSecs:      0,
		OnIterStart: func(iter int, iterType string) {
			startIters = append(startIters, iter)
			if iterType != IterationTypeImplementation {
				t.Errorf("expected type=%s, got=%s", IterationTypeImplementation, iterType)
			}
		},
		OnIterEnd: func(iter int, err error) {
			endIters = append(endIters, iter)
			endErrors = append(endErrors, err)
		},
	}

	_ = RunAutoLoop(cfg)

	if len(startIters) != 1 || startIters[0] != 1 {
		t.Errorf("expected OnIterStart called once with iter=1, got %v", startIters)
	}
	if len(endIters) != 1 || endIters[0] != 1 {
		t.Errorf("expected OnIterEnd called once with iter=1, got %v", endIters)
	}
	// codex not installed, so there should be an error
	if len(endErrors) == 1 && endErrors[0] == nil {
		t.Log("codex unexpectedly succeeded")
	}
}
