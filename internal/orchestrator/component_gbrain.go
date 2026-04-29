package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// gbrainExec is the exec function the gbrain component shells out
// through. Tests inject fakes by overriding this.
var gbrainExec = exec.CommandContext

// GbrainComponent registers gbrain as a Claude Code MCP server. Samuel
// does NOT install gbrain itself — the user is responsible for getting
// it on PATH (e.g., `bun add -g gbrain` or `npm install -g gbrain`).
// What Samuel owns is the MCP registration that wires Claude Code to
// the installed gbrain binary; that goes through `claude mcp add`
// rather than direct settings.json manipulation.
type GbrainComponent struct {
	// gbrainBinary, when non-empty, overrides PATH lookup for diagnostic
	// commands like `gbrain --version`. The MCP registration always
	// uses the same path so the registered command matches what Detect
	// actually probed.
	gbrainBinary string
}

// NewGbrainComponent constructs a gbrain component. Pass an empty string
// to use PATH lookup; tests pass a known stub path.
func NewGbrainComponent(gbrainBinary string) *GbrainComponent {
	return &GbrainComponent{gbrainBinary: gbrainBinary}
}

// Name returns the constant component identifier.
func (g *GbrainComponent) Name() string { return NameGbrain }

// resolveGbrainBinary returns the path Samuel will register and probe.
// Honors opts.GbrainBinary > component default > PATH lookup.
func (g *GbrainComponent) resolveGbrainBinary(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if g.gbrainBinary != "" {
		return g.gbrainBinary, nil
	}
	path, err := exec.LookPath("gbrain")
	if err != nil {
		return "", err
	}
	return path, nil
}

// Detect probes the system for gbrain. Reports Installed=true when the
// binary is reachable AND prints a parseable version. Never mutates.
func (g *GbrainComponent) Detect(ctx context.Context) (DetectResult, error) {
	bin, err := g.resolveGbrainBinary("")
	if err != nil {
		return DetectResult{Installed: false}, nil
	}
	out, err := gbrainExec(ctx, bin, "--version").Output()
	if err != nil {
		return DetectResult{Installed: true, Path: bin}, nil
	}
	return DetectResult{
		Installed: true,
		Version:   strings.TrimSpace(string(out)),
		Path:      bin,
	}, nil
}

// Install registers gbrain with Claude Code at user scope.
//
// PRE-MUTATION DETECTION: if gbrain is not on PATH and no override was
// passed, return a clean error BEFORE invoking `claude mcp add`. This
// matters because the orchestrator runs gstack first; we don't want a
// half-installed bundle when the user's gbrain prereq isn't met.
//
// Idempotent: if `claude mcp get gbrain` already shows the server
// registered, returns AlreadyInstalled=true without re-registering.
func (g *GbrainComponent) Install(ctx context.Context, opts InstallOptions) (InstallResult, error) {
	res := InstallResult{Component: NameGbrain}
	if opts.SkipGbrain {
		res.Skipped = true
		return res, nil
	}

	bin, err := g.resolveGbrainBinary(opts.GbrainBinary)
	if err != nil {
		return res, &Error{
			Component:   NameGbrain,
			Problem:     "gbrain not found on PATH",
			Cause:       "Samuel requires gbrain to be installed separately",
			Fix:         "install gbrain (Bun: `bun add -g gbrain`; npm: `npm install -g gbrain`), then re-run; or pass --gbrain-binary to point at an existing install",
			DocsURL:     "https://samuel.dev/docs/errors/SAM-GBRAIN-001",
			Recoverable: true,
		}
	}

	if _, err := exec.LookPath("claude"); err != nil {
		return res, &Error{
			Component:   NameGbrain,
			Problem:     "claude CLI not found on PATH",
			Cause:       "MCP registration uses `claude mcp add`, which requires Claude Code installed",
			Fix:         "install Claude Code (https://claude.com/claude-code) and ensure `claude` is on PATH",
			DocsURL:     "https://samuel.dev/docs/errors/SAM-GBRAIN-002",
			Recoverable: true,
		}
	}

	// Idempotency: if already registered, no-op.
	registered, err := g.isRegistered(ctx)
	if err == nil && registered && !opts.Force {
		res.AlreadyInstalled = true
		return res, nil
	}

	if opts.DryRun {
		return res, nil
	}

	// `claude mcp add -s user gbrain <bin> serve` — name=gbrain,
	// commandOrUrl=<bin>, args=[serve]. Scope is "user" so the
	// registration is per-user, not per-project.
	addOut, err := gbrainExec(ctx,
		"claude", "mcp", "add", "-s", "user", "gbrain", bin, "serve",
	).CombinedOutput()
	if err != nil {
		return res, (&Error{
			Component:   NameGbrain,
			Problem:     "failed to register gbrain MCP server",
			Cause:       strings.TrimSpace(string(addOut)),
			Fix:         "verify Claude Code MCP support is healthy: `claude mcp list`",
			DocsURL:     "https://samuel.dev/docs/errors/SAM-GBRAIN-003",
			Recoverable: true,
		}).Wrap(err)
	}
	res.Mutations = append(res.Mutations, Mutation{
		Kind:        MutationCommandRun,
		Path:        "~/.claude/settings.json::mcpServers.gbrain",
		Description: fmt.Sprintf("registered gbrain MCP server (command=%s, scope=user)", bin),
		Reverse: func(ctx context.Context) error {
			// Best-effort removal on rollback.
			_ = gbrainExec(ctx, "claude", "mcp", "remove", "-s", "user", "gbrain").Run()
			return nil
		},
	})

	return res, nil
}

// Check reports gbrain health. OK when the binary is on PATH AND the
// MCP server is registered. Read-only — never mutates.
func (g *GbrainComponent) Check(ctx context.Context) HealthStatus {
	detect, err := g.Detect(ctx)
	if err != nil {
		return HealthStatus{
			Component: NameGbrain,
			OK:        false,
			Message:   "cannot detect gbrain: " + err.Error(),
			FixHint:   "samuel init",
		}
	}
	if !detect.Installed {
		return HealthStatus{
			Component: NameGbrain,
			OK:        false,
			Message:   "gbrain not on PATH",
			FixHint:   "bun add -g gbrain (or npm install -g gbrain), then samuel init",
		}
	}

	if _, err := exec.LookPath("claude"); err != nil {
		return HealthStatus{
			Component: NameGbrain,
			OK:        false,
			Message:   "claude CLI not on PATH; cannot verify MCP registration",
			FixHint:   "install Claude Code, then samuel init",
		}
	}

	registered, err := g.isRegistered(ctx)
	if err != nil {
		return HealthStatus{
			Component: NameGbrain,
			OK:        false,
			Message:   "cannot query MCP registration: " + err.Error(),
			FixHint:   "claude mcp list",
		}
	}
	if !registered {
		return HealthStatus{
			Component: NameGbrain,
			OK:        false,
			Message:   "gbrain installed but not registered as MCP server",
			FixHint:   "samuel init",
		}
	}
	msg := "gbrain registered"
	if detect.Version != "" {
		msg = "gbrain " + detect.Version + " registered"
	}
	return HealthStatus{
		Component: NameGbrain,
		OK:        true,
		Message:   msg,
	}
}

// Uninstall removes the gbrain MCP registration via `claude mcp remove`.
// Does NOT uninstall the gbrain binary itself — Samuel didn't install it.
func (g *GbrainComponent) Uninstall(ctx context.Context, opts UninstallOptions) (UninstallResult, error) {
	res := UninstallResult{Component: NameGbrain}

	// Project-only uninstall has nothing to remove for gbrain (the MCP
	// registration is user-scoped and shared across all projects).
	if !opts.Global && !opts.All {
		res.Skipped = true
		return res, nil
	}

	if _, err := exec.LookPath("claude"); err != nil {
		// Without the claude CLI we can't deregister cleanly; surface
		// a manual fallback rather than silently leaving stale config.
		return res, &Error{
			Component:   NameGbrain,
			Problem:     "claude CLI not found; cannot remove gbrain MCP registration",
			Fix:         "remove the `gbrain` entry from ~/.claude/settings.json's mcpServers manually, or reinstall Claude Code",
			Recoverable: true,
		}
	}

	if opts.DryRun {
		return res, nil
	}

	out, err := gbrainExec(ctx, "claude", "mcp", "remove", "-s", "user", "gbrain").CombinedOutput()
	if err != nil {
		// "not registered" is expected on second uninstall — treat
		// idempotently.
		txt := strings.ToLower(string(out))
		if strings.Contains(txt, "not found") || strings.Contains(txt, "no such") {
			return res, nil
		}
		return res, (&Error{
			Component:   NameGbrain,
			Problem:     "failed to remove gbrain MCP registration",
			Cause:       strings.TrimSpace(string(out)),
			Fix:         "manually run: claude mcp remove -s user gbrain",
			Recoverable: true,
		}).Wrap(err)
	}
	res.Mutations = append(res.Mutations, Mutation{
		Kind:        MutationCommandRun,
		Path:        "~/.claude/settings.json::mcpServers.gbrain",
		Description: "removed gbrain MCP registration",
		Reverse:     nil,
	})
	return res, nil
}

// isRegistered queries Claude Code for the gbrain MCP entry. Returns
// (true, nil) when registered, (false, nil) when not, (false, err) on
// genuine failures (claude CLI errors, etc.).
func (g *GbrainComponent) isRegistered(ctx context.Context) (bool, error) {
	_, err := gbrainExec(ctx, "claude", "mcp", "get", "gbrain").CombinedOutput()
	if err == nil {
		return true, nil
	}
	// `claude mcp get` exits non-zero when the entry is absent. Treat
	// that as "not registered" rather than a genuine error.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}
	return false, err
}
