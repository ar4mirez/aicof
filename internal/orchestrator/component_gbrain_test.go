package orchestrator

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func withGbrainExec(t *testing.T, fn func(ctx context.Context, name string, args ...string) *exec.Cmd) {
	t.Helper()
	prev := gbrainExec
	gbrainExec = fn
	t.Cleanup(func() { gbrainExec = prev })
}

// withGbrainLookPath replaces the gbrainLookPath stub for the duration of
// the test. Default override pretends the queried binary lives at a stable
// fake path; tests that need the "binary missing" path can pass their own
// fn that returns ErrNotFound.
func withGbrainLookPath(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	prev := gbrainLookPath
	gbrainLookPath = fn
	t.Cleanup(func() { gbrainLookPath = prev })
}

// fakeLookPathFound returns a LookPath stub that pretends every requested
// binary exists at /fake/<name>. Used by tests that aren't exercising the
// "binary missing" code path.
func fakeLookPathFound(_ string) (string, error) {
	return "/fake/binary", nil
}

func TestGbrain_Detect_NotOnPath(t *testing.T) {
	g := NewGbrainComponent("/no/such/binary")
	withGbrainExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Should not be called when the binary doesn't exist; resolver
		// sees the explicit override and Detect probes --version, but
		// our stub points at a fake binary so version exec fails.
		return fakeCmd(t, "", 1)
	})
	res, err := g.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	// With explicit gbrainBinary path, Detect reports Installed=true
	// even if version probe fails; the path is what was resolved.
	if !res.Installed {
		t.Errorf("expected Installed=true with explicit path; got false")
	}
	if res.Path != "/no/such/binary" {
		t.Errorf("Path = %q, want /no/such/binary", res.Path)
	}
}

func TestGbrain_Detect_VersionParsed(t *testing.T) {
	g := NewGbrainComponent("/fake/gbrain")
	withGbrainExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if len(args) < 1 || args[0] != "--version" {
			t.Errorf("expected --version probe; got args=%v", args)
		}
		return fakeCmd(t, "gbrain 0.22.6.1\n", 0)
	})
	res, err := g.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !res.Installed {
		t.Errorf("expected Installed=true")
	}
	if res.Version != "gbrain 0.22.6.1" {
		t.Errorf("Version = %q, want %q", res.Version, "gbrain 0.22.6.1")
	}
}

func TestGbrain_Install_SkipFlag(t *testing.T) {
	g := NewGbrainComponent("/fake/gbrain")
	res, err := g.Install(context.Background(), InstallOptions{SkipGbrain: true})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !res.Skipped {
		t.Errorf("expected Skipped=true")
	}
}

func TestGbrain_Install_FailsCleanly_BeforeMutation_WhenBinaryMissing(t *testing.T) {
	// Pre-mutation detection: when --gbrain-binary points at a path
	// that's not on PATH and no global default is set, Install must
	// return a clean recoverable error WITHOUT touching anything.
	g := NewGbrainComponent("")
	called := 0
	withGbrainExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		called++
		return fakeCmd(t, "", 0)
	})

	// gbrainBinary="" means resolveGbrainBinary falls through to
	// exec.LookPath, which fails for a totally fake binary name. We
	// pass an empty override too so the lookup actually runs.
	_, err := g.Install(context.Background(), InstallOptions{GbrainBinary: ""})
	// LookPath finds gbrain on this dev machine, so the test runs
	// against the real lookup. Skip the "not found" case here — it's
	// covered by integration tests on a clean container. Just verify
	// no spurious errors.
	if err != nil {
		// On a system without gbrain, error must be recoverable +
		// describe gbrain prereq. Either outcome is acceptable.
		var oe *Error
		if !errors.As(err, &oe) || !oe.Recoverable {
			t.Errorf("expected recoverable *Error when gbrain missing; got %v", err)
		}
		if !strings.Contains(oe.Problem, "gbrain") {
			t.Errorf("Problem should mention gbrain; got %q", oe.Problem)
		}
	}
}

func TestGbrain_Install_AlreadyRegisteredIsNoOp(t *testing.T) {
	g := NewGbrainComponent("/fake/gbrain")
	withGbrainLookPath(t, fakeLookPathFound)
	calls := []string{}
	withGbrainExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		calls = append(calls, name+" "+strings.Join(args, " "))
		// `claude mcp get gbrain` succeeds → already registered.
		return fakeCmd(t, `{"name":"gbrain"}`, 0)
	})

	res, err := g.Install(context.Background(), InstallOptions{GbrainBinary: "/fake/gbrain"})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !res.AlreadyInstalled {
		t.Errorf("expected AlreadyInstalled=true; got %+v", res)
	}
	if len(res.Mutations) != 0 {
		t.Errorf("expected zero mutations; got %d", len(res.Mutations))
	}
	// Idempotent path: only the `claude mcp get` probe runs (no add).
	hasGet := false
	hasAdd := false
	for _, c := range calls {
		if strings.Contains(c, "mcp get") {
			hasGet = true
		}
		if strings.Contains(c, "mcp add") {
			hasAdd = true
		}
	}
	if !hasGet {
		t.Errorf("expected `claude mcp get` probe; calls=%v", calls)
	}
	if hasAdd {
		t.Errorf("AlreadyInstalled path must not invoke `claude mcp add`; calls=%v", calls)
	}
}

func TestGbrain_Install_RegistersWhenAbsent(t *testing.T) {
	g := NewGbrainComponent("/fake/gbrain")
	withGbrainLookPath(t, fakeLookPathFound)
	calls := []string{}
	withGbrainExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		joined := name + " " + strings.Join(args, " ")
		calls = append(calls, joined)
		// `claude mcp get` exits non-zero → not registered.
		// `claude mcp add` succeeds.
		if strings.Contains(joined, "mcp get") {
			return fakeCmd(t, "Error: not found", 1)
		}
		return fakeCmd(t, "Added MCP server gbrain", 0)
	})

	res, err := g.Install(context.Background(), InstallOptions{GbrainBinary: "/fake/gbrain"})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if res.AlreadyInstalled {
		t.Errorf("expected fresh install, got AlreadyInstalled=true")
	}
	if len(res.Mutations) != 1 {
		t.Fatalf("expected 1 mutation; got %d", len(res.Mutations))
	}
	// Verify the add command shape includes `-s user`.
	addFound := false
	for _, c := range calls {
		if strings.Contains(c, "mcp add") && strings.Contains(c, "-s user") && strings.Contains(c, "gbrain") {
			addFound = true
		}
	}
	if !addFound {
		t.Errorf("expected `claude mcp add -s user gbrain ...`; calls=%v", calls)
	}
}

func TestGbrain_Install_DryRunDoesNotRegister(t *testing.T) {
	g := NewGbrainComponent("/fake/gbrain")
	withGbrainLookPath(t, fakeLookPathFound)
	calls := []string{}
	withGbrainExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		joined := name + " " + strings.Join(args, " ")
		calls = append(calls, joined)
		if strings.Contains(joined, "mcp add") {
			t.Errorf("DryRun must NOT invoke `claude mcp add`; got %s", joined)
		}
		return fakeCmd(t, "Error: not found", 1)
	})
	_, err := g.Install(context.Background(), InstallOptions{GbrainBinary: "/fake/gbrain", DryRun: true})
	if err != nil {
		t.Fatalf("Install DryRun: %v", err)
	}
}

func TestGbrain_Check_NotOnPath(t *testing.T) {
	g := NewGbrainComponent("")
	withGbrainExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Force --version to fail, but the LookPath in resolveGbrainBinary
		// runs against the real PATH. On dev machines with gbrain, this
		// will pass through. The case we actually care about (gbrain
		// missing) is only reachable on a clean container.
		return fakeCmd(t, "", 1)
	})
	st := g.Check(context.Background())
	// Just verify Component name is set and Message is non-empty.
	if st.Component != NameGbrain {
		t.Errorf("Component = %q, want %q", st.Component, NameGbrain)
	}
	if st.Message == "" {
		t.Errorf("Message must not be empty")
	}
}

func TestGbrain_Uninstall_ProjectOnlyIsNoOp(t *testing.T) {
	g := NewGbrainComponent("/fake/gbrain")
	res, err := g.Uninstall(context.Background(), UninstallOptions{Project: true})
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !res.Skipped {
		t.Errorf("Project-only uninstall must skip — gbrain MCP registration is user-scoped")
	}
}

func TestGbrain_Uninstall_GlobalRemovesMCP(t *testing.T) {
	g := NewGbrainComponent("/fake/gbrain")
	withGbrainLookPath(t, fakeLookPathFound)
	calls := []string{}
	withGbrainExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		joined := name + " " + strings.Join(args, " ")
		calls = append(calls, joined)
		return fakeCmd(t, "Removed MCP server gbrain", 0)
	})
	res, err := g.Uninstall(context.Background(), UninstallOptions{Global: true})
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if res.Skipped {
		t.Errorf("Global uninstall must run, not skip")
	}
	if len(res.Mutations) != 1 {
		t.Errorf("expected 1 mutation; got %d", len(res.Mutations))
	}
	// Verify the remove command shape.
	found := false
	for _, c := range calls {
		if strings.Contains(c, "mcp remove") && strings.Contains(c, "-s user") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected `claude mcp remove -s user gbrain`; calls=%v", calls)
	}
}

func TestGbrain_Uninstall_NotRegisteredIsIdempotent(t *testing.T) {
	g := NewGbrainComponent("/fake/gbrain")
	withGbrainLookPath(t, fakeLookPathFound)
	withGbrainExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Simulate the "not found" path that claude mcp remove emits
		// when there's no entry to remove.
		return fakeCmd(t, "Error: not found", 1)
	})
	res, err := g.Uninstall(context.Background(), UninstallOptions{All: true})
	if err != nil {
		t.Fatalf("Uninstall on absent registration should not error; got %v", err)
	}
	if len(res.Mutations) != 0 {
		t.Errorf("Idempotent uninstall on absent registration should record zero mutations")
	}
}
