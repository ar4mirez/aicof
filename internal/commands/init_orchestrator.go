package commands

import (
	"context"
	"fmt"

	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

// runOrchestratorBundle invokes the v4 orchestrator after the legacy v3
// init flow has finished. It installs the three-component bundle (gstack,
// gbrain, samuel-skills) into the user's global state and creates the
// project's symlink at <project>/.claude/skills/samuel/.
//
// When --no-orchestrator is set, this is a no-op so users on systems
// where gstack or gbrain can't run (offline machines, restricted CI) can
// still use Samuel's v3 template extraction. PR4b will redesign add/
// remove/update/list/etc. around the v4 sync model — until then, opting
// out keeps the legacy flow working.
//
// Errors are rendered through the structured renderer and propagated to
// Cobra so the command exits non-zero on bundle failure. Partial init
// (legacy template extracted but bundle install failed) is intentional:
// the user gets a clear error with a Fix hint and can re-run after
// installing the missing prereq (e.g., gbrain).
func runOrchestratorBundle(cmd *cobra.Command, flags *initFlags) error {
	if flags.noOrchestrator {
		ui.Dim("  - orchestrator skipped (--no-orchestrator)")
		return nil
	}

	orch, err := buildOrchestrator(flags.absTargetDir, samuelVersion())
	if err != nil {
		return renderOrchestratorError(fmt.Errorf("orchestrator setup: %w", err))
	}

	opts := installOptionsFrom(orchestratorOptions{
		SkipGstack:   flags.skipGstack,
		SkipGbrain:   flags.skipGbrain,
		GbrainBinary: flags.gbrainBinary,
		SkipSymlink:  flags.noSymlink,
		Verbose:      false,
	})

	if !flags.jsonMode {
		fmt.Println()
		ui.Bold("v4 orchestrator (gstack + gbrain + samuel-skills):")
	}

	results, err := orch.Install(context.Background(), opts)
	if err != nil {
		return renderOrchestratorError(err)
	}

	if !flags.jsonMode {
		renderInstallResults(results)
	}
	return nil
}

// samuelVersion returns the binary version reported to the samuel-skills
// component. Wrapping the package-level Version constant makes the
// dependency explicit and lets future code attach build metadata
// (commit, dirty marker) without touching every caller.
func samuelVersion() string {
	if Version != "" {
		return Version
	}
	return "0.0.0-dev"
}
