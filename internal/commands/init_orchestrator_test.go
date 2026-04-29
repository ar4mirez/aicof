package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

// flagSetForOrchestrator constructs a cobra command with just the
// orchestrator-related flags wired up. Used to exercise the parsing
// and runOrchestratorBundle code without spinning up the whole init
// command tree.
func flagSetForOrchestrator() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("skip-gstack", false, "")
	cmd.Flags().Bool("skip-gbrain", false, "")
	cmd.Flags().String("gbrain-binary", "", "")
	cmd.Flags().Bool("no-symlink", false, "")
	cmd.Flags().Bool("no-orchestrator", false, "")
	return cmd
}

func TestRunOrchestratorBundle_NoOrchestratorIsCleanSkip(t *testing.T) {
	flags := &initFlags{
		absTargetDir:   t.TempDir(),
		noOrchestrator: true,
	}
	if err := runOrchestratorBundle(flagSetForOrchestrator(), flags); err != nil {
		t.Errorf("--no-orchestrator should be a clean no-op; got %v", err)
	}
}

func TestRunOrchestratorBundle_RespectsSkipFlags(t *testing.T) {
	// When all three components are skipped, the orchestrator should
	// run cleanly with no mutations and no errors.
	flags := &initFlags{
		absTargetDir: t.TempDir(),
		skipGstack:   true,
		skipGbrain:   true,
		noSymlink:    true,
		jsonMode:     true,
	}
	// samuel-skills isn't covered by SkipGstack/SkipGbrain — but with
	// noSymlink=true and a tmpdir target, it'll do a fresh sync to a
	// throwaway HOME-rooted path. To make this test fully hermetic
	// without creating files in the real ~/.claude, we'd need component
	// substitution. For now: assert no error AND no panic.
	//
	// In a clean container with no claude/gbrain/gstack on PATH, this
	// runs the samuel-skills component against the real $HOME. That's
	// acceptable for now; PR4b can add proper test seams.
	t.Skip("samuel-skills component writes to real $HOME — requires test seam to be hermetic")
	_ = runOrchestratorBundle(flagSetForOrchestrator(), flags)
}
