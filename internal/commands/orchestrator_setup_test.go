package commands

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ar4mirez/samuel/internal/orchestrator"
)

func TestInstallOptionsFrom_PassesEverythingThrough(t *testing.T) {
	got := installOptionsFrom(orchestratorOptions{
		SkipGstack:   true,
		SkipGbrain:   true,
		GbrainBinary: "/opt/gbrain/bin/gbrain",
		SkipSymlink:  true,
		DryRun:       true,
		Verbose:      true,
	})
	if !got.SkipGstack || !got.SkipGbrain || !got.SkipSymlink || !got.DryRun || !got.Verbose {
		t.Errorf("escape-hatch fields lost in translation: %+v", got)
	}
	if got.GbrainBinary != "/opt/gbrain/bin/gbrain" {
		t.Errorf("GbrainBinary = %q, want /opt/gbrain/bin/gbrain", got.GbrainBinary)
	}
	if got.Stdout == nil {
		t.Errorf("Stdout should default to os.Stdout (non-nil)")
	}
}

func TestBuildOrchestrator_ReturnsThreeComponents(t *testing.T) {
	orch, err := buildOrchestrator(t.TempDir(), "v4.0.0-test")
	if err != nil {
		t.Fatalf("buildOrchestrator: %v", err)
	}
	if orch == nil {
		t.Fatal("expected non-nil orchestrator")
	}
	// Verify components are present by exercising Doctor — read-only,
	// safe to run in tests. Three statuses = three components.
	statuses := orch.Doctor(context.Background())
	if len(statuses) != 3 {
		t.Errorf("expected 3 components in bundle; got %d (%+v)", len(statuses), statuses)
	}
	// Each status must have a Component name set (defensive against
	// future component changes that forget to populate it).
	for _, s := range statuses {
		if s.Component == "" {
			t.Errorf("HealthStatus missing Component name: %+v", s)
		}
	}
}

func TestRenderOrchestratorError_NilIsNoop(t *testing.T) {
	if got := renderOrchestratorError(nil); got != nil {
		t.Errorf("renderOrchestratorError(nil) = %v, want nil", got)
	}
}

func TestRenderOrchestratorError_PreservesError(t *testing.T) {
	err := &orchestrator.Error{
		Component:   orchestrator.NameGbrain,
		Problem:     "test problem",
		Cause:       "test cause",
		Fix:         "test fix",
		DocsURL:     "https://example.test/docs",
		Recoverable: true,
	}
	got := renderOrchestratorError(err)
	if got == nil {
		t.Fatal("expected non-nil error returned")
	}
	var oe *orchestrator.Error
	if !errors.As(got, &oe) {
		t.Fatalf("returned error should still be *orchestrator.Error; got %T", got)
	}
	if oe.Problem != "test problem" {
		t.Errorf("Problem = %q, want %q", oe.Problem, "test problem")
	}
}

func TestRenderOrchestratorError_PlainErrorPath(t *testing.T) {
	err := errors.New("plain error")
	got := renderOrchestratorError(err)
	if got == nil {
		t.Fatal("expected non-nil error returned")
	}
	if !strings.Contains(got.Error(), "plain error") {
		t.Errorf("returned error string = %q, want to contain 'plain error'", got.Error())
	}
}

func TestHealthStatusToCheckResult_OK(t *testing.T) {
	s := orchestrator.HealthStatus{
		Component: orchestrator.NameGstack,
		OK:        true,
		Message:   "gstack at e8893a1",
	}
	got := healthStatusToCheckResult(s)
	if got.name != "v4:gstack" {
		t.Errorf("name = %q, want v4:gstack", got.name)
	}
	if !got.passed {
		t.Errorf("passed should be true")
	}
	if got.message != "gstack at e8893a1" {
		t.Errorf("message = %q, want unchanged", got.message)
	}
}

func TestHealthStatusToCheckResult_FailWithFixHint(t *testing.T) {
	s := orchestrator.HealthStatus{
		Component: orchestrator.NameGbrain,
		OK:        false,
		Message:   "gbrain not on PATH",
		FixHint:   "bun add -g gbrain",
	}
	got := healthStatusToCheckResult(s)
	if got.passed {
		t.Errorf("passed should be false")
	}
	if !strings.Contains(got.message, "fix: bun add -g gbrain") {
		t.Errorf("message should embed FixHint; got %q", got.message)
	}
}

func TestAllHealthy(t *testing.T) {
	tests := []struct {
		name     string
		statuses []orchestrator.HealthStatus
		want     bool
	}{
		{"empty", nil, true},
		{"single ok", []orchestrator.HealthStatus{{OK: true}}, true},
		{"single bad", []orchestrator.HealthStatus{{OK: false}}, false},
		{"mixed", []orchestrator.HealthStatus{{OK: true}, {OK: false}}, false},
		{"all ok", []orchestrator.HealthStatus{{OK: true}, {OK: true}, {OK: true}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := allHealthy(tc.statuses); got != tc.want {
				t.Errorf("allHealthy = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSamuelVersion_FallsBackToDevWhenUnset(t *testing.T) {
	prev := Version
	defer func() { Version = prev }()
	Version = ""
	got := samuelVersion()
	if got == "" {
		t.Errorf("samuelVersion should never return empty string")
	}
	if got != "0.0.0-dev" {
		t.Errorf("empty Version should fall back to 0.0.0-dev; got %q", got)
	}
}

func TestSamuelVersion_ReturnsBuildVersionWhenSet(t *testing.T) {
	prev := Version
	defer func() { Version = prev }()
	Version = "v4.0.0-rc1"
	if got := samuelVersion(); got != "v4.0.0-rc1" {
		t.Errorf("samuelVersion = %q, want v4.0.0-rc1", got)
	}
}
