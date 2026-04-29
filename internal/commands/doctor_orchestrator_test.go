package commands

import (
	"testing"

	"github.com/ar4mirez/samuel/internal/orchestrator"
)

func TestRunOrchestratorDoctor_ReturnsThreeComponentsPlusBundleSummary(t *testing.T) {
	statuses := runOrchestratorDoctor(t.TempDir(), false)
	// 3 components + an optional bundle-summary line when everything
	// is healthy. Length must be ≥3, ≤4.
	if len(statuses) < 3 {
		t.Fatalf("expected at least 3 statuses; got %d (%+v)", len(statuses), statuses)
	}
	// Each status names its component.
	componentSeen := map[string]bool{}
	for _, s := range statuses {
		componentSeen[s.Component] = true
		if s.Message == "" {
			t.Errorf("HealthStatus has empty Message: %+v", s)
		}
	}
	for _, c := range []string{
		orchestrator.NameGstack,
		orchestrator.NameGbrain,
		orchestrator.NameSamuelSkills,
	} {
		if !componentSeen[c] {
			t.Errorf("expected component %q in doctor output; got %v", c, componentSeen)
		}
	}
}

func TestRunOrchestratorDoctor_VerifyDoesNotAppendBundleSummary(t *testing.T) {
	// In verify mode, the bundle-summary shortcut is omitted. Test
	// asserts the shape: exactly 3 statuses (one per component), no
	// orchestrator-named summary line.
	statuses := runOrchestratorDoctor(t.TempDir(), true)
	if len(statuses) != 3 {
		t.Errorf("verify=true should return exactly 3 statuses (one per component); got %d", len(statuses))
	}
	for _, s := range statuses {
		if s.Component == orchestrator.NameOrchestrator && s.OK {
			t.Errorf("verify mode must not append the orchestrator-summary line; got %+v", s)
		}
	}
}
