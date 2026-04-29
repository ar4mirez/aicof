package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/ar4mirez/samuel/internal/orchestrator"
	"github.com/ar4mirez/samuel/internal/skills"
	"github.com/ar4mirez/samuel/internal/ui"
)

// orchestratorOptions captures the v4 init/doctor escape hatches that come
// from CLI flags. These let users opt out of components when their machine
// can't host them (gbrain not installed, gstack composed elsewhere, no
// symlink support on the target FS).
type orchestratorOptions struct {
	// SkipGstack omits the gstack component (e.g., user manages gstack
	// themselves outside Samuel).
	SkipGstack bool
	// SkipGbrain omits the gbrain MCP registration (e.g., gbrain not
	// installed and the user wants to defer setup).
	SkipGbrain bool
	// GbrainBinary overrides PATH lookup for gbrain (e.g., a custom
	// install location like /opt/local/bin/gbrain).
	GbrainBinary string
	// SkipSymlink omits the project-local symlink to the global skills
	// tree (e.g., dev container, network FS that doesn't support symlinks).
	SkipSymlink bool
	// DryRun reports what would change without mutating state.
	DryRun bool
	// Verbose enables progress streaming from the components.
	Verbose bool
}

// buildOrchestrator constructs the v4 orchestrator with the three standard
// components: gstack, gbrain, and samuel-skills. The order matches the
// install dependency order.
//
// projectDir is where the samuel-skills project symlink should be written.
// Pass "" to skip project-level symlink creation entirely (e.g., samuel
// doctor invoked outside a project).
//
// version is the Samuel binary version reported by Detect on the
// samuel-skills component (skills are embedded; binary version IS the
// content version).
func buildOrchestrator(projectDir, version string) (*orchestrator.Orchestrator, error) {
	skillFS, err := skills.FS()
	if err != nil {
		return nil, fmt.Errorf("samuel binary is missing embedded skills: %w", err)
	}
	return orchestrator.New(
		orchestrator.NewGstackComponent(""),
		orchestrator.NewGbrainComponent(""),
		orchestrator.NewSamuelComponent(skillFS, "", projectDir, version),
	), nil
}

// installOptionsFrom translates command-line escape hatches into
// orchestrator.InstallOptions. Centralizing the mapping keeps init.go
// from re-deriving fields and lets tests assert on the resolved options.
func installOptionsFrom(opts orchestratorOptions) orchestrator.InstallOptions {
	return orchestrator.InstallOptions{
		DryRun:       opts.DryRun,
		Verbose:      opts.Verbose,
		Stdout:       os.Stdout,
		SkipGstack:   opts.SkipGstack,
		SkipGbrain:   opts.SkipGbrain,
		GbrainBinary: opts.GbrainBinary,
		SkipSymlink:  opts.SkipSymlink,
	}
}

// renderOrchestratorError prints a structured *orchestrator.Error in the
// shape promised by the v4 error standard:
//
//	Error: <Problem>
//	  Cause: <Cause>
//	  Fix:   <Fix>
//	  Docs:  <DocsURL>
//
// Falls back to a plain Error: line when err is not an *orchestrator.Error.
// Returns the input err so callers can `return renderOrchestratorError(err)`
// — useful in command runners that propagate the error to Cobra.
func renderOrchestratorError(err error) error {
	if err == nil {
		return nil
	}
	var oe *orchestrator.Error
	if !errors.As(err, &oe) {
		ui.Error("%v", err)
		return err
	}
	ui.Error("%s", oe.Problem)
	if oe.Cause != "" {
		ui.ListItem(1, "Cause: %s", oe.Cause)
	}
	if oe.Fix != "" {
		ui.ListItem(1, "Fix:   %s", oe.Fix)
	}
	if oe.DocsURL != "" {
		ui.ListItem(1, "Docs:  %s", oe.DocsURL)
	}
	if oe.Path != "" {
		ui.ListItem(1, "Path:  %s", oe.Path)
	}
	return err
}

// renderInstallResults prints a one-line per-component summary of what
// the orchestrator did. Used by `samuel init` after Install returns.
func renderInstallResults(results []orchestrator.InstallResult) {
	for _, r := range results {
		switch {
		case r.Skipped:
			ui.Dim("  - %s: skipped", r.Component)
		case r.AlreadyInstalled:
			ui.Dim("  ✓ %s: already installed", r.Component)
		case len(r.Mutations) > 0:
			ui.SuccessItem(1, "%s: installed (%d changes)", r.Component, len(r.Mutations))
		default:
			ui.SuccessItem(1, "%s: ok", r.Component)
		}
	}
}

// renderHealthStatuses prints orchestrator HealthStatus values in the
// same shape as the v3 doctor checkResult lines, so users see one
// uniform health page mixing both layers.
func renderHealthStatuses(statuses []orchestrator.HealthStatus) {
	for _, s := range statuses {
		if s.OK {
			ui.SuccessItem(0, "%s: %s", s.Component, s.Message)
		} else {
			ui.ErrorItem(0, "%s: %s", s.Component, s.Message)
			if s.FixHint != "" {
				ui.ListItem(2, "Fix: %s", s.FixHint)
			}
		}
	}
}
