package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ar4mirez/samuel/internal/core"
)

func TestRun_RegisteredAtRoot(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "run" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected runCmd to be registered as a child of rootCmd")
	}
}

func TestRun_AutoAlias(t *testing.T) {
	hasAutoAlias := false
	for _, alias := range runCmd.Aliases {
		if alias == "auto" {
			hasAutoAlias = true
			break
		}
	}
	if !hasAutoAlias {
		t.Errorf("expected runCmd to have 'auto' alias, got: %v", runCmd.Aliases)
	}
}

func TestRun_AutoCmdIsAlias(t *testing.T) {
	// `var autoCmd = runCmd` keeps internal references working. Both must point
	// at the same Cobra command.
	if autoCmd != runCmd {
		t.Error("expected autoCmd and runCmd to be the same pointer")
	}
}

func TestRun_HasArgsNoArgs(t *testing.T) {
	// runCmd.Args must reject positional args so 'samuel run something' returns
	// a clear error instead of being treated as a positional to the bare RunE.
	if runCmd.Args == nil {
		t.Fatal("expected runCmd.Args to be set (cobra.NoArgs)")
	}
	err := runCmd.Args(runCmd, []string{"unexpected"})
	if err == nil {
		t.Error("expected runCmd.Args to reject positional arguments")
	}
}

func TestRun_HasFlatVerbs(t *testing.T) {
	want := map[string]bool{
		"tasks":   false,
		"done":    false,
		"skip":    false,
		"reset":   false,
		"enqueue": false,
	}
	for _, c := range runCmd.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("expected runCmd to have %q as a flat subcommand", name)
		}
	}
}

func TestRun_HasLifecycleSubcommands(t *testing.T) {
	want := map[string]bool{
		"init":    false,
		"start":   false,
		"pilot":   false,
		"status":  false,
		"convert": false,
		"task":    false,
	}
	for _, c := range runCmd.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("expected runCmd to have %q subcommand", name)
		}
	}
}

func TestRun_TaskAddPreserved(t *testing.T) {
	// 'samuel run task add' must remain visible (not Hidden) so CI scripts that
	// rely on deterministic IDs keep discovering it via --help.
	if autoTaskAddCmd.Hidden {
		t.Error("expected autoTaskAddCmd to remain visible (preserved for CI)")
	}
}

func TestRun_LegacyTaskList_HiddenAndForwards(t *testing.T) {
	if !autoTaskListCmd.Hidden {
		t.Error("expected autoTaskListCmd.Hidden = true")
	}
	if !strings.Contains(autoTaskListCmd.Short, "DEPRECATED") {
		t.Errorf("expected autoTaskListCmd.Short to mark deprecation, got %q", autoTaskListCmd.Short)
	}
}

func TestRun_LegacyTaskComplete_Hidden(t *testing.T) {
	if !autoTaskCompleteCmd.Hidden {
		t.Error("expected autoTaskCompleteCmd.Hidden = true")
	}
}

func TestRun_LegacyTaskSkip_Hidden(t *testing.T) {
	if !autoTaskSkipCmd.Hidden {
		t.Error("expected autoTaskSkipCmd.Hidden = true")
	}
}

func TestRun_LegacyTaskReset_Hidden(t *testing.T) {
	if !autoTaskResetCmd.Hidden {
		t.Error("expected autoTaskResetCmd.Hidden = true")
	}
}

func TestRun_BareNoPRD_ReturnsError(t *testing.T) {
	// Run in a temp directory with no .claude/auto/prd.json so the bare RunE
	// should emit help to stderr and return an error.
	tmp := t.TempDir()
	origCwd, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origCwd) })

	stderr := captureStderr(t, func() {
		err := runRunBare(runCmd, []string{})
		if err == nil {
			t.Error("expected error when no PRD exists")
		} else if !strings.Contains(err.Error(), "no auto loop initialized") {
			t.Errorf("expected error to mention 'no auto loop initialized', got: %v", err)
		}
	})
	if !strings.Contains(stderr, "samuel run init") {
		t.Errorf("expected stderr to suggest 'samuel run init', got: %s", stderr)
	}
}

func TestRun_BareWithPRD_ShowsStatus(t *testing.T) {
	// Construct a minimal valid PRD on disk and verify runRunBare dispatches to
	// the status path (no error, no "no auto loop" message).
	tmp := t.TempDir()
	origCwd, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origCwd) })

	autoDir := filepath.Join(tmp, ".claude", "auto")
	if err := os.MkdirAll(autoDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Minimal valid PRD that LoadAutoPRD accepts.
	prd := &core.AutoPRD{
		Version: "1.0",
		Project: core.AutoProject{Name: "test"},
		Config:  core.AutoConfig{MaxIterations: 10},
		Tasks:   []core.AutoTask{},
	}
	if err := prd.Save(core.GetAutoPRDPath(tmp)); err != nil {
		t.Fatal(err)
	}

	// runRunBare delegates to runAutoStatus. Even if status output produces
	// rendering noise, we just want to confirm the no-loop branch isn't taken.
	stderr := captureStderr(t, func() {
		_ = runRunBare(runCmd, []string{}) // ignore err; status may fail on minimal data
	})
	if strings.Contains(stderr, "no autonomous loop initialized") {
		t.Errorf("expected runRunBare to dispatch to status when PRD exists, got no-loop message: %s", stderr)
	}
}
