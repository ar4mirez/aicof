package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// LoopConfig holds all parameters for running the autonomous loop.
type LoopConfig struct {
	ProjectDir     string
	PRDPath        string
	PromptPath     string
	AITool         string
	MaxIterations  int
	Sandbox        string
	SandboxImage   string
	SandboxTpl     string
	PauseSecs      int
	MaxConsecFails int
	OnIterStart    func(iter int, iterType string)
	OnIterEnd      func(iter int, err error)
}

// NewLoopConfig creates a LoopConfig with defaults from a PRD and project dir.
func NewLoopConfig(projectDir string, prd *AutoPRD) LoopConfig {
	pauseSecs := 2
	if val := os.Getenv("PAUSE_SECONDS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			pauseSecs = parsed
		}
	}

	maxConsecFails := 3
	if val := os.Getenv("MAX_CONSECUTIVE_FAILURES"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxConsecFails = parsed
		}
	}

	return LoopConfig{
		ProjectDir:     projectDir,
		PRDPath:        GetAutoPRDPath(projectDir),
		PromptPath:     filepath.Join(projectDir, prd.Config.PromptFile),
		AITool:         prd.Config.AITool,
		MaxIterations:  prd.Config.MaxIterations,
		Sandbox:        prd.Config.Sandbox,
		SandboxImage:   prd.Config.SandboxImage,
		SandboxTpl:     prd.Config.SandboxTemplate,
		PauseSecs:      pauseSecs,
		MaxConsecFails: maxConsecFails,
	}
}

// RunAutoLoop executes the autonomous loop using Go-native orchestration.
// It replaces the bash-based auto.sh script.
func RunAutoLoop(cfg LoopConfig) error {
	consecutiveFailures := 0

	for i := 1; i <= cfg.MaxIterations; i++ {
		prd, err := LoadAutoPRD(cfg.PRDPath)
		if err != nil {
			return fmt.Errorf("iteration %d: failed to reload prd.json: %w", i, err)
		}

		if prd.GetNextTask() == nil {
			notifyIterEnd(cfg.OnIterEnd, i, nil)
			return nil
		}

		notifyIterStart(cfg.OnIterStart, i, IterationTypeImplementation)

		PrepareProgressContext(cfg.ProjectDir)
		_ = GenerateProjectSnapshot(cfg.ProjectDir)
		_ = GenerateTaskContext(cfg.ProjectDir, prd, false)

		err = InvokeAgent(cfg)
		if err != nil {
			consecutiveFailures++
			notifyIterEnd(cfg.OnIterEnd, i, err)
			if consecutiveFailures >= cfg.MaxConsecFails {
				return fmt.Errorf(
					"%d consecutive failures reached — aborting. "+
						"Check AI tool auth/config", cfg.MaxConsecFails)
			}
		} else {
			consecutiveFailures = 0
			notifyIterEnd(cfg.OnIterEnd, i, nil)
		}

		if i < cfg.MaxIterations {
			time.Sleep(time.Duration(cfg.PauseSecs) * time.Second)
		}
	}

	return nil
}

// InvokeAgent calls the AI tool for one iteration of work.
// It validates cfg.AITool against the allow-list before execution
// to prevent arbitrary command injection via modified prd.json.
func InvokeAgent(cfg LoopConfig) error {
	if !IsValidAITool(cfg.AITool) {
		return fmt.Errorf(
			"refused to invoke invalid AI tool %q (allowed: %v)",
			cfg.AITool, GetSupportedAITools())
	}

	switch cfg.Sandbox {
	case SandboxDockerSandbox:
		return invokeAgentDockerSandbox(cfg)
	case SandboxDocker:
		return invokeAgentDocker(cfg)
	default:
		return invokeAgentLocal(cfg)
	}
}

func invokeAgentLocal(cfg LoopConfig) error {
	args, err := GetAgentArgs(cfg.AITool, cfg.PromptPath)
	if err != nil {
		return fmt.Errorf("failed to build agent args: %w", err)
	}

	cmd := exec.Command(cfg.AITool, args...)
	cmd.Dir = cfg.ProjectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func invokeAgentDocker(cfg LoopConfig) error {
	promptRel, err := filepath.Rel(cfg.ProjectDir, cfg.PromptPath)
	if err != nil {
		return fmt.Errorf("failed to compute relative prompt path: %w", err)
	}

	containerPromptPath := filepath.Join(DockerContainerMount, promptRel)
	agentPromptPath := containerPromptPath
	if requiresPromptContent(cfg.AITool) {
		agentPromptPath = cfg.PromptPath
	}

	agentArgs, err := GetAgentArgs(
		cfg.AITool,
		agentPromptPath,
	)
	if err != nil {
		return fmt.Errorf("failed to build agent args: %w", err)
	}

	image := cfg.SandboxImage
	if image == "" {
		image = DefaultSandboxImage
	}
	if !IsValidSandboxImage(image) {
		return fmt.Errorf(
			"refused to use invalid sandbox image %q: must match Docker image reference format",
			image)
	}

	dockerArgs := buildDockerRunArgs(cfg.ProjectDir, image, cfg.AITool, agentArgs)
	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// requiresPromptContent reports whether the agent expects prompt content in an
// argument rather than a prompt file path.
func requiresPromptContent(aiTool string) bool {
	switch aiTool {
	case "claude", "codex":
		return true
	default:
		return false
	}
}

func invokeAgentDockerSandbox(cfg LoopConfig) error {
	agentArgs, err := GetAgentArgs(cfg.AITool, cfg.PromptPath)
	if err != nil {
		return fmt.Errorf("failed to build agent args: %w", err)
	}

	sandboxCfg := DockerSandboxRunConfig{
		Agent:     cfg.AITool,
		WorkDir:   cfg.ProjectDir,
		Template:  cfg.SandboxTpl,
		AgentArgs: agentArgs,
	}

	args := BuildDockerSandboxArgs(sandboxCfg)
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// buildDockerRunArgs constructs docker run arguments for agent invocation.
func buildDockerRunArgs(workDir, image, aiTool string, agentArgs []string) []string {
	args := []string{"run", "--rm", "--init", "-i"}
	args = append(args, fmt.Sprintf("--user=%d:%d", os.Getuid(), os.Getgid()))
	args = append(args, "-v", fmt.Sprintf("%s:%s", workDir, DockerContainerMount))
	args = append(args, "-w", DockerContainerMount)
	args = append(args, getAIToolEnvVars()...)
	args = append(args, image)
	args = append(args, aiTool)
	args = append(args, agentArgs...)
	return args
}

func notifyIterStart(fn func(int, string), iter int, iterType string) {
	if fn != nil {
		fn(iter, iterType)
	}
}

func notifyIterEnd(fn func(int, error), iter int, err error) {
	if fn != nil {
		fn(iter, err)
	}
}
