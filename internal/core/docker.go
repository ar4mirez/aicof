package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Sandbox mode constants
const (
	SandboxNone          = "none"
	SandboxDocker        = "docker"
	SandboxDockerSandbox = "docker-sandbox"
	DefaultSandboxImage  = "node:lts"
	DockerContainerMount = "/workspace"
	// DefaultDockerSandboxAgent is the agent name for docker sandbox run.
	DefaultDockerSandboxAgent = "claude"
)

// aiToolEnvVarNames is the allowlist of environment variables passed into the
// Docker container. Only variables that are actually set on the host are
// forwarded, preventing accidental secret leakage.
var aiToolEnvVarNames = []string{
	"ANTHROPIC_API_KEY",
	"OPENAI_API_KEY",
	"AMP_API_KEY",
	"AI_TOOL",
	"PAUSE_SECONDS",
	"MAX_CONSECUTIVE_FAILURES",
	"TERM",
}

// DockerSandboxRunConfig holds the parameters for docker sandbox run.
type DockerSandboxRunConfig struct {
	Agent     string   // agent name: claude, codex, copilot, gemini, kiro
	WorkDir   string   // host path (mounted at same absolute path inside VM)
	Template  string   // custom template override (optional)
	Name      string   // sandbox name for persistence (optional)
	AgentArgs []string // extra arguments passed after -- to the agent
}

// GetSupportedSandboxModes returns the list of supported sandbox modes.
func GetSupportedSandboxModes() []string {
	return []string{SandboxNone, SandboxDocker, SandboxDockerSandbox}
}

// IsValidSandboxMode checks if the given mode is supported.
func IsValidSandboxMode(mode string) bool {
	return slices.Contains(GetSupportedSandboxModes(), strings.ToLower(mode))
}

// validSandboxImagePattern matches Docker image references:
// [registry/]name[:tag][@digest] with alphanumeric, dots, dashes, underscores,
// colons, and slashes. Rejects shell metacharacters and absolute paths.
var validSandboxImagePattern = regexp.MustCompile(
	`^[a-zA-Z0-9][a-zA-Z0-9._\-/]*(:[a-zA-Z0-9._\-]+)?(@sha256:[a-f0-9]{64})?$`,
)

// IsValidSandboxImage validates a Docker image name for safe use with docker run.
// It rejects empty strings, shell metacharacters, absolute paths, and names
// that don't match the Docker image reference format.
func IsValidSandboxImage(image string) bool {
	if image == "" {
		return false
	}
	if strings.HasPrefix(image, "/") || strings.HasPrefix(image, ".") {
		return false
	}
	return validSandboxImagePattern.MatchString(image)
}

// CheckDockerAvailable verifies Docker is installed and the daemon is running.
func CheckDockerAvailable() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker not found in PATH; install Docker or use --sandbox=none")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := exec.CommandContext(ctx, "docker", "info").Run(); err != nil {
		return fmt.Errorf("docker daemon is not running; start Docker Desktop or the docker service")
	}
	return nil
}

// getAIToolEnvVars returns -e KEY=VALUE pairs for set environment variables.
func getAIToolEnvVars() []string {
	var envArgs []string
	for _, name := range aiToolEnvVarNames {
		if val, ok := os.LookupEnv(name); ok {
			envArgs = append(envArgs, "-e", fmt.Sprintf("%s=%s", name, val))
		}
	}
	return envArgs
}

// CheckDockerSandboxAvailable verifies the docker sandbox plugin is installed.
func CheckDockerSandboxAvailable() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker not found in PATH; install Docker Desktop")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := exec.CommandContext(ctx, "docker", "sandbox", "version").Run(); err != nil {
		return fmt.Errorf(
			"docker sandbox plugin not available; " +
				"install Docker Desktop with Sandbox support",
		)
	}
	return nil
}

// BuildDockerSandboxArgs constructs the argument slice for docker sandbox run.
func BuildDockerSandboxArgs(config DockerSandboxRunConfig) []string {
	agent := config.Agent
	if agent == "" {
		agent = DefaultDockerSandboxAgent
	}

	args := []string{"sandbox", "run"}

	if config.Name != "" {
		args = append(args, "--name", config.Name)
	}
	if config.Template != "" {
		args = append(args, "--template", config.Template)
	}

	args = append(args, agent)

	workDir := config.WorkDir
	if workDir == "" {
		workDir = "."
	}
	args = append(args, workDir)

	if len(config.AgentArgs) > 0 {
		args = append(args, "--")
		args = append(args, config.AgentArgs...)
	}

	return args
}

// GetAgentArgs returns the CLI arguments for an AI agent in docker sandbox.
// For tools without prompt-file support, the prompt file content is read and
// passed directly as a CLI argument.
func GetAgentArgs(aiTool, promptPath string) ([]string, error) {
	switch aiTool {
	case "claude":
		content, err := os.ReadFile(promptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read prompt file: %w", err)
		}
		return []string{
			"-p", string(content), "--dangerously-skip-permissions",
		}, nil
	case "codex":
		content, err := os.ReadFile(promptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read prompt file: %w", err)
		}
		return []string{
			"--sandbox", "danger-full-access",
			"--ask-for-approval", "never",
			"exec", string(content),
		}, nil
	case "amp":
		return []string{"--prompt-file", promptPath}, nil
	default:
		return []string{promptPath}, nil
	}
}
