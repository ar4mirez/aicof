package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockComponent is a configurable Component used to drive lifecycle tests.
// Each callback can be overridden per test. Counters are atomic so tests
// that exercise concurrent paths (Doctor, locking) are race-clean.
type mockComponent struct {
	name string

	detectFn    func(context.Context) (DetectResult, error)
	installFn   func(context.Context, InstallOptions) (InstallResult, error)
	checkFn     func(context.Context) HealthStatus
	uninstallFn func(context.Context, UninstallOptions) (UninstallResult, error)

	// Counters for assertions. atomic so concurrent test paths stay race-clean.
	installCalls   atomic.Int64
	checkCalls     atomic.Int64
	uninstallCalls atomic.Int64
}

func (m *mockComponent) Name() string { return m.name }

func (m *mockComponent) Detect(ctx context.Context) (DetectResult, error) {
	if m.detectFn != nil {
		return m.detectFn(ctx)
	}
	return DetectResult{Installed: false}, nil
}

func (m *mockComponent) Install(ctx context.Context, opts InstallOptions) (InstallResult, error) {
	m.installCalls.Add(1)
	if m.installFn != nil {
		return m.installFn(ctx, opts)
	}
	return InstallResult{Component: m.name}, nil
}

func (m *mockComponent) Check(ctx context.Context) HealthStatus {
	m.checkCalls.Add(1)
	if m.checkFn != nil {
		return m.checkFn(ctx)
	}
	return HealthStatus{Component: m.name, OK: true, Message: "healthy"}
}

func (m *mockComponent) Uninstall(ctx context.Context, opts UninstallOptions) (UninstallResult, error) {
	m.uninstallCalls.Add(1)
	if m.uninstallFn != nil {
		return m.uninstallFn(ctx, opts)
	}
	return UninstallResult{Component: m.name}, nil
}

// newOrchestratorWithTempHome returns an Orchestrator whose lock file
// lives under a fresh temp dir so concurrent tests do not collide.
func newOrchestratorWithTempHome(t *testing.T, components ...Component) *Orchestrator {
	t.Helper()
	dir := t.TempDir()
	o := New(components...).WithHomeDir(dir)
	return o
}

func TestNew_ZeroComponents(t *testing.T) {
	o := newOrchestratorWithTempHome(t)
	results, err := o.Install(context.Background(), InstallOptions{})
	if err != nil {
		t.Fatalf("Install with no components: unexpected err %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if got := o.Doctor(context.Background()); len(got) != 0 {
		t.Errorf("Doctor with no components should return empty, got %d", len(got))
	}
}

func TestInstall_HappyPath_AllComponentsCalled(t *testing.T) {
	a := &mockComponent{name: "gstack"}
	b := &mockComponent{name: "gbrain"}
	c := &mockComponent{name: "samuel-skills"}
	o := newOrchestratorWithTempHome(t, a, b, c)

	results, err := o.Install(context.Background(), InstallOptions{})
	if err != nil {
		t.Fatalf("Install: unexpected err %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, comp := range []*mockComponent{a, b, c} {
		if got := comp.installCalls.Load(); got != 1 {
			t.Errorf("component %d (%s) install calls = %d, want 1", i, comp.name, got)
		}
	}
}

func TestInstall_PopulatesComponentNameWhenMissing(t *testing.T) {
	c := &mockComponent{
		name: "samuel-skills",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{}, nil // Component intentionally empty
		},
	}
	o := newOrchestratorWithTempHome(t, c)
	results, err := o.Install(context.Background(), InstallOptions{})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if results[0].Component != "samuel-skills" {
		t.Errorf("expected Component to be set from Name(), got %q", results[0].Component)
	}
}

func TestInstall_PartialFailure_RollsBackInLIFOOrder(t *testing.T) {
	var reverseOrder []string
	mu := sync.Mutex{}
	record := func(name string) func(context.Context) error {
		return func(_ context.Context) error {
			mu.Lock()
			defer mu.Unlock()
			reverseOrder = append(reverseOrder, name)
			return nil
		}
	}

	a := &mockComponent{
		name: "gstack",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{
				Mutations: []Mutation{
					{Kind: MutationCommandRun, Path: "gstack-1", Reverse: record("a-1")},
					{Kind: MutationCommandRun, Path: "gstack-2", Reverse: record("a-2")},
				},
			}, nil
		},
	}
	b := &mockComponent{
		name: "gbrain",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{
				Mutations: []Mutation{
					{Kind: MutationCommandRun, Path: "gbrain-1", Reverse: record("b-1")},
				},
			}, nil
		},
	}
	c := &mockComponent{
		name: "samuel-skills",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{}, errors.New("samuel-skills failed")
		},
	}
	o := newOrchestratorWithTempHome(t, a, b, c)

	_, err := o.Install(context.Background(), InstallOptions{})
	if err == nil {
		t.Fatalf("expected install error, got nil")
	}

	mu.Lock()
	defer mu.Unlock()
	want := []string{"b-1", "a-2", "a-1"}
	if fmt.Sprint(reverseOrder) != fmt.Sprint(want) {
		t.Errorf("rollback order = %v, want %v (LIFO across all components)", reverseOrder, want)
	}
}

func TestInstall_RollbackJoinsErrors(t *testing.T) {
	a := &mockComponent{
		name: "gstack",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{
				Mutations: []Mutation{
					{Reverse: func(_ context.Context) error { return errors.New("reverse-failed") }},
				},
			}, nil
		},
	}
	b := &mockComponent{
		name: "gbrain",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{}, errors.New("install-failed")
		},
	}
	o := newOrchestratorWithTempHome(t, a, b)

	_, err := o.Install(context.Background(), InstallOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
	msg := err.Error()
	for _, want := range []string{"install-failed", "reverse-failed", "rollback"} {
		if !contains(msg, want) {
			t.Errorf("expected error to contain %q, got %q", want, msg)
		}
	}
}

func TestInstall_RollbackContinuesAfterReverseError(t *testing.T) {
	var reversed []string
	a := &mockComponent{
		name: "gstack",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{
				Mutations: []Mutation{
					{Path: "first", Reverse: func(_ context.Context) error {
						reversed = append(reversed, "first")
						return nil
					}},
					{Path: "second-broken", Reverse: func(_ context.Context) error {
						reversed = append(reversed, "second")
						return errors.New("boom")
					}},
				},
			}, nil
		},
	}
	b := &mockComponent{
		name: "gbrain",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{}, errors.New("trigger rollback")
		},
	}
	o := newOrchestratorWithTempHome(t, a, b)

	_, err := o.Install(context.Background(), InstallOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if len(reversed) != 2 {
		t.Errorf("rollback should continue past errors; reversed = %v", reversed)
	}
}

func TestInstall_NilReverseSkipped(t *testing.T) {
	a := &mockComponent{
		name: "gstack",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{
				Mutations: []Mutation{
					{Path: "no-reverse"}, // Reverse is nil
				},
			}, nil
		},
	}
	b := &mockComponent{
		name: "gbrain",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{}, errors.New("fail")
		},
	}
	o := newOrchestratorWithTempHome(t, a, b)

	_, err := o.Install(context.Background(), InstallOptions{})
	if err == nil || !contains(err.Error(), "fail") {
		t.Errorf("expected install error to surface; got %v", err)
	}
	// No panic on nil Reverse is the assertion.
}

func TestInstall_SkipGstackOmitsComponent(t *testing.T) {
	a := &mockComponent{name: "gstack"}
	b := &mockComponent{name: "gbrain"}
	o := newOrchestratorWithTempHome(t, a, b)

	results, err := o.Install(context.Background(), InstallOptions{SkipGstack: true})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if got := a.installCalls.Load(); got != 0 {
		t.Errorf("gstack should be skipped, got %d installs", got)
	}
	if got := b.installCalls.Load(); got != 1 {
		t.Errorf("gbrain should run, got %d installs", got)
	}
	if !results[0].Skipped || results[0].Component != "gstack" {
		t.Errorf("expected first result to be Skipped gstack, got %+v", results[0])
	}
}

func TestInstall_SkipGbrainOmitsComponent(t *testing.T) {
	a := &mockComponent{name: "gstack"}
	b := &mockComponent{name: "gbrain"}
	o := newOrchestratorWithTempHome(t, a, b)
	_, err := o.Install(context.Background(), InstallOptions{SkipGbrain: true})
	if err != nil {
		t.Fatalf("%v", err)
	}
	if got := b.installCalls.Load(); got != 0 {
		t.Errorf("gbrain should be skipped, got %d", got)
	}
}

func TestDoctor_CallsCheckOnEveryComponent(t *testing.T) {
	a := &mockComponent{name: "gstack"}
	b := &mockComponent{name: "gbrain"}
	c := &mockComponent{name: "samuel-skills"}
	o := New(a, b, c) // No lock needed for Doctor

	statuses := o.Doctor(context.Background())
	if len(statuses) != 3 {
		t.Fatalf("expected 3 statuses, got %d", len(statuses))
	}
	for _, c := range []*mockComponent{a, b, c} {
		if got := c.checkCalls.Load(); got != 1 {
			t.Errorf("Check on %s called %d times, want 1", c.name, got)
		}
	}
}

func TestDoctor_PopulatesComponentNameWhenMissing(t *testing.T) {
	c := &mockComponent{
		name: "samuel-skills",
		checkFn: func(_ context.Context) HealthStatus {
			return HealthStatus{OK: true, Message: "ok"} // Component intentionally empty
		},
	}
	o := New(c)
	statuses := o.Doctor(context.Background())
	if statuses[0].Component != "samuel-skills" {
		t.Errorf("Doctor should populate Component from Name(), got %q", statuses[0].Component)
	}
}

func TestDoctor_DoesNotAcquireLock(t *testing.T) {
	// If Doctor took the lock, two concurrent Doctor calls in quick
	// succession would serialize. Verify they don't.
	c := &mockComponent{name: "samuel-skills"}
	o := newOrchestratorWithTempHome(t, c)

	done := make(chan struct{}, 2)
	for i := 0; i < 2; i++ {
		go func() {
			o.Doctor(context.Background())
			done <- struct{}{}
		}()
	}
	timeout := time.After(2 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-timeout:
			t.Fatalf("Doctor calls did not complete; lock may have been acquired")
		}
	}
}

func TestUninstall_CallsComponentsInReverseOrder(t *testing.T) {
	var order []string
	makeComp := func(name string) *mockComponent {
		return &mockComponent{
			name: name,
			uninstallFn: func(_ context.Context, _ UninstallOptions) (UninstallResult, error) {
				order = append(order, name)
				return UninstallResult{Component: name}, nil
			},
		}
	}
	o := newOrchestratorWithTempHome(t, makeComp("gstack"), makeComp("gbrain"), makeComp("samuel-skills"))

	_, err := o.Uninstall(context.Background(), UninstallOptions{All: true})
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	want := []string{"samuel-skills", "gbrain", "gstack"}
	if fmt.Sprint(order) != fmt.Sprint(want) {
		t.Errorf("uninstall order = %v, want %v", order, want)
	}
}

func TestUninstall_BestEffort_ContinuesOnError(t *testing.T) {
	// Uninstall is best-effort: a failing component does NOT stop later
	// components from running. All component errors are joined.
	a := &mockComponent{name: "gstack"}
	b := &mockComponent{
		name: "gbrain",
		uninstallFn: func(_ context.Context, _ UninstallOptions) (UninstallResult, error) {
			return UninstallResult{}, errors.New("gbrain uninstall failed")
		},
	}
	c := &mockComponent{name: "samuel-skills"}
	o := newOrchestratorWithTempHome(t, a, b, c)

	results, err := o.Uninstall(context.Background(), UninstallOptions{All: true})
	if err == nil || !contains(err.Error(), "gbrain") {
		t.Errorf("expected joined error mentioning gbrain, got %v", err)
	}
	// All three components run despite gbrain's error.
	if got := c.uninstallCalls.Load(); got != 1 {
		t.Errorf("samuel-skills should have run; got %d", got)
	}
	if got := b.uninstallCalls.Load(); got != 1 {
		t.Errorf("gbrain should have been attempted; got %d", got)
	}
	if got := a.uninstallCalls.Load(); got != 1 {
		t.Errorf("gstack should have run after gbrain's error (best-effort); got %d", got)
	}
	// All three results present in reverse order (samuel-skills, gbrain, gstack).
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestUninstall_BestEffort_JoinsMultipleErrors(t *testing.T) {
	// When more than one component fails, the joined error must surface
	// every failure so the user can see the full picture.
	a := &mockComponent{
		name: "gstack",
		uninstallFn: func(_ context.Context, _ UninstallOptions) (UninstallResult, error) {
			return UninstallResult{}, errors.New("gstack-error")
		},
	}
	c := &mockComponent{
		name: "samuel-skills",
		uninstallFn: func(_ context.Context, _ UninstallOptions) (UninstallResult, error) {
			return UninstallResult{}, errors.New("samuel-skills-error")
		},
	}
	o := newOrchestratorWithTempHome(t, a, c)

	_, err := o.Uninstall(context.Background(), UninstallOptions{All: true})
	if err == nil {
		t.Fatalf("expected joined error")
	}
	for _, want := range []string{"gstack-error", "samuel-skills-error"} {
		if !contains(err.Error(), want) {
			t.Errorf("joined error should contain %q, got %q", want, err.Error())
		}
	}
}

func TestLock_SuccessiveInstallsSucceed(t *testing.T) {
	// flock auto-releases on close, so two sequential installs must
	// both succeed even though the lock file persists.
	dir := t.TempDir()
	c := &mockComponent{name: "samuel-skills"}
	o := New(c).WithHomeDir(dir)

	if _, err := o.Install(context.Background(), InstallOptions{}); err != nil {
		t.Fatalf("first install: %v", err)
	}
	// Lock file persists across runs (flock-based; file is just the
	// holder marker), but the kernel released the flock when the fd
	// closed.
	if _, err := os.Stat(filepath.Join(dir, LockPath)); err != nil {
		t.Errorf("lock file should still exist after release (flock model); stat err = %v", err)
	}
	if _, err := o.Install(context.Background(), InstallOptions{}); err != nil {
		t.Errorf("second install: %v", err)
	}
}

func TestLock_LockReleasedAfterInstallError(t *testing.T) {
	// Even when the install fails, the lock is released so the next
	// call can proceed.
	dir := t.TempDir()
	c := &mockComponent{
		name: "samuel-skills",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{}, errors.New("boom")
		},
	}
	o := New(c).WithHomeDir(dir)

	if _, err := o.Install(context.Background(), InstallOptions{}); err == nil {
		t.Fatalf("expected install error")
	}
	// A second Install must not be blocked.
	c2 := &mockComponent{name: "samuel-skills"}
	o2 := New(c2).WithHomeDir(dir)
	if _, err := o2.Install(context.Background(), InstallOptions{}); err != nil {
		t.Errorf("second install (after error) should succeed; got %v", err)
	}
}

func TestInstall_FailingComponentPartialMutationsRolledBack(t *testing.T) {
	// Regression test: the failing component itself may have written
	// partial mutations before erroring. Those must be rolled back
	// alongside the prior components' mutations.
	var reversed []string
	a := &mockComponent{
		name: "gstack",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{
				Mutations: []Mutation{
					{Path: "a-1", Reverse: func(_ context.Context) error {
						reversed = append(reversed, "a-1")
						return nil
					}},
				},
			}, nil
		},
	}
	b := &mockComponent{
		name: "gbrain",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			// Returns BOTH partial mutations AND an error.
			return InstallResult{
				Mutations: []Mutation{
					{Path: "b-partial-1", Reverse: func(_ context.Context) error {
						reversed = append(reversed, "b-partial-1")
						return nil
					}},
				},
			}, errors.New("gbrain failed mid-install")
		},
	}
	o := newOrchestratorWithTempHome(t, a, b)

	results, err := o.Install(context.Background(), InstallOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
	want := []string{"b-partial-1", "a-1"}
	if fmt.Sprint(reversed) != fmt.Sprint(want) {
		t.Errorf("rollback should include failing component's partial mutations; got %v, want %v", reversed, want)
	}
	// The failing component's result must be present in results so
	// callers can introspect what was attempted.
	if len(results) != 2 {
		t.Errorf("expected 2 results (success + failure), got %d", len(results))
	}
	if results[1].Component != "gbrain" {
		t.Errorf("expected failing component result to be present, got %q", results[1].Component)
	}
}

func TestInstall_RollbackUsesFreshContext(t *testing.T) {
	// Regression test: install ctx may be canceled at the moment of
	// failure. Rollback must run on a fresh context so cleanup isn't
	// also aborted.
	a := &mockComponent{
		name: "gstack",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{
				Mutations: []Mutation{
					{Path: "a-1", Reverse: func(rbCtx context.Context) error {
						// rbCtx must NOT be the canceled install ctx.
						if rbCtx.Err() != nil {
							return fmt.Errorf("rollback ctx already canceled: %w", rbCtx.Err())
						}
						return nil
					}},
				},
			}, nil
		},
	}
	b := &mockComponent{
		name: "gbrain",
		installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			return InstallResult{}, errors.New("trigger rollback")
		},
	}
	o := newOrchestratorWithTempHome(t, a, b)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled before Install runs
	_, err := o.Install(canceledCtx, InstallOptions{})
	if err == nil {
		t.Fatalf("expected install error")
	}
	// If rollback inherited the canceled ctx, the assertion in the
	// Reverse closure would have produced "rollback ctx already
	// canceled" and the joined error would mention it. Confirm the
	// rollback succeeded by checking the error contains the install
	// error but NOT the rollback-canceled marker.
	if contains(err.Error(), "rollback ctx already canceled") {
		t.Errorf("rollback ran with canceled context: %v", err)
	}
}

func TestShouldSkip_CaseInsensitive(t *testing.T) {
	// Regression test: a component whose Name() returns "GSTACK" or
	// "GStack" must still be skipped when SkipGstack is set.
	cases := []struct {
		name     string
		nameOf   string
		opts     InstallOptions
		expected bool
	}{
		{"exact", NameGstack, InstallOptions{SkipGstack: true}, true},
		{"upper", "GSTACK", InstallOptions{SkipGstack: true}, true},
		{"mixed", "GStack", InstallOptions{SkipGstack: true}, true},
		{"unrelated", "warp", InstallOptions{SkipGstack: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &mockComponent{name: tc.nameOf}
			if got := shouldSkip(c, tc.opts); got != tc.expected {
				t.Errorf("shouldSkip(%q, SkipGstack) = %v, want %v", tc.nameOf, got, tc.expected)
			}
		})
	}
}

// contains is a substring helper used by tests that only need a "yes/no
// substring exists" check on freeform error strings. Kept thin to avoid
// importing strings just for this one assertion shape.
func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
