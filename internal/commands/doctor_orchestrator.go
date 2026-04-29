package commands

import (
	"context"
	"fmt"

	"github.com/ar4mirez/samuel/internal/orchestrator"
	"github.com/ar4mirez/samuel/internal/skills"
)

// runOrchestratorDoctor invokes Check on each v4 component and returns
// the per-component HealthStatus. Read-only — never mutates state.
//
// On internal setup failure (missing embedded skills, etc.), returns a
// single failed HealthStatus so the user sees the problem in `samuel
// doctor` output rather than a stack trace.
//
// verify is reserved for deeper probes (e.g., actually invoking the MCP
// server, resolving a known skill end-to-end). The current Check methods
// already cover the static health surface; verify is wired so v4.x can
// add live probes without touching every call site.
func runOrchestratorDoctor(projectDir string, verify bool) []orchestrator.HealthStatus {
	skillFS, err := skills.FS()
	if err != nil {
		return []orchestrator.HealthStatus{{
			Component: orchestrator.NameOrchestrator,
			OK:        false,
			Message:   "samuel binary missing embedded skills: " + err.Error(),
			FixHint:   "rebuild Samuel from a clean checkout",
		}}
	}

	orch := orchestrator.New(
		orchestrator.NewGstackComponent(""),
		orchestrator.NewGbrainComponent(""),
		orchestrator.NewSamuelComponent(skillFS, "", projectDir, samuelVersion()),
	)

	statuses := orch.Doctor(context.Background())

	// Bundle-level summary: if every component reports healthy, append a
	// "bundle: all components green" line so users get a single positive
	// signal at the top of the orchestrator section. Stays out of the way
	// when something is broken.
	if !verify && allHealthy(statuses) {
		statuses = append(statuses, orchestrator.HealthStatus{
			Component: orchestrator.NameOrchestrator,
			OK:        true,
			Message:   fmt.Sprintf("bundle healthy (%d components)", len(statuses)),
		})
	}

	return statuses
}

// allHealthy reports whether every status has OK=true.
func allHealthy(statuses []orchestrator.HealthStatus) bool {
	for _, s := range statuses {
		if !s.OK {
			return false
		}
	}
	return true
}

// healthStatusToCheckResult adapts an orchestrator HealthStatus to the
// legacy doctor checkResult shape so both layers render through the same
// code path. The component name becomes the check name, prefixed with
// "v4:" so users can tell at a glance which check came from where.
func healthStatusToCheckResult(s orchestrator.HealthStatus) checkResult {
	msg := s.Message
	if !s.OK && s.FixHint != "" {
		msg = fmt.Sprintf("%s — fix: %s", s.Message, s.FixHint)
	}
	return checkResult{
		name:    "v4:" + s.Component,
		passed:  s.OK,
		message: msg,
		// fixable=false: orchestrator components self-report fix hints in
		// their FixHint; samuel doctor --fix doesn't yet know how to act
		// on them. Wiring orchestrator-aware --fix lands in PR4b/PR5.
		fixable: false,
	}
}
