package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

func TestUninstall_StopsOnFirstError(t *testing.T) {
	a := &mockComponent{name: "gstack"}
	b := &mockComponent{
		name: "gbrain",
		uninstallFn: func(_ context.Context, _ UninstallOptions) (UninstallResult, error) {
			return UninstallResult{}, errors.New("gbrain uninstall failed")
		},
	}
	c := &mockComponent{name: "samuel-skills"}
	o := newOrchestratorWithTempHome(t, a, b, c)

	_, err := o.Uninstall(context.Background(), UninstallOptions{All: true})
	if err == nil || !contains(err.Error(), "gbrain") {
		t.Errorf("expected error mentioning gbrain, got %v", err)
	}
	// samuel-skills runs first (LIFO), then gbrain fails, then gstack does NOT run.
	if got := c.uninstallCalls.Load(); got != 1 {
		t.Errorf("samuel-skills (called first) should have run; got %d", got)
	}
	if got := a.uninstallCalls.Load(); got != 0 {
		t.Errorf("gstack (called last) should not have run after gbrain error; got %d", got)
	}
}

func TestLock_ConcurrentInstallsSerialize(t *testing.T) {
	dir := t.TempDir()

	// Use blocking install fns so we can assert the second caller waits.
	gate := make(chan struct{})
	var mu sync.Mutex
	var inFlight int
	var maxInFlight int

	makeComp := func(name string) *mockComponent {
		return &mockComponent{
			name: name,
			installFn: func(_ context.Context, _ InstallOptions) (InstallResult, error) {
				mu.Lock()
				inFlight++
				if inFlight > maxInFlight {
					maxInFlight = inFlight
				}
				mu.Unlock()
				<-gate // hold the lock until released
				mu.Lock()
				inFlight--
				mu.Unlock()
				return InstallResult{}, nil
			},
		}
	}
	o := New(makeComp("samuel-skills")).WithHomeDir(dir)

	done := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			done <- mustNotPanic(func() error {
				_, err := o.Install(context.Background(), InstallOptions{})
				return err
			})
		}()
	}

	// First caller is now holding the lock and blocked on gate.
	// The orchestrator's sync.Mutex serializes acquireLock, but the second
	// goroutine should also fail O_EXCL since the first wrote the file.
	// Release the gate so the first call returns and frees the lock.
	time.Sleep(50 * time.Millisecond)
	close(gate)

	for i := 0; i < 2; i++ {
		select {
		case err := <-done:
			if err != nil {
				// One of the two callers may legitimately fail with
				// "another samuel process is running" if the second
				// reaches acquireLock before the first releases it,
				// which is racy under the gate scheme. Either result
				// (success or LOCK error) is acceptable as long as
				// inFlight never exceeded 1.
				var oe *Error
				if !errors.As(err, &oe) || !contains(oe.Problem, "samuel process") {
					t.Errorf("unexpected error: %v", err)
				}
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("install did not complete")
		}
	}
	if maxInFlight > 1 {
		t.Errorf("max concurrent installs = %d, want 1 (lock failed to serialize)", maxInFlight)
	}
}

func TestLock_StaleLockEvicted(t *testing.T) {
	dir := t.TempDir()
	// Pre-create a stale lock file with a PID that definitely doesn't exist.
	lockPath := filepath.Join(dir, LockPath)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte("999999999"), 0o600); err != nil {
		t.Fatalf("write stale lock: %v", err)
	}

	c := &mockComponent{name: "samuel-skills"}
	o := New(c).WithHomeDir(dir)

	if _, err := o.Install(context.Background(), InstallOptions{}); err != nil {
		t.Errorf("Install with stale lock should succeed (stale lock evicted), got %v", err)
	}
}

func TestLock_LiveLockRejects(t *testing.T) {
	dir := t.TempDir()
	// Pre-create a lock with our own PID — clearly alive.
	lockPath := filepath.Join(dir, LockPath)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	pid := fmt.Sprintf("%d", os.Getpid())
	if err := os.WriteFile(lockPath, []byte(pid), 0o600); err != nil {
		t.Fatalf("write live lock: %v", err)
	}

	c := &mockComponent{name: "samuel-skills"}
	o := New(c).WithHomeDir(dir)

	_, err := o.Install(context.Background(), InstallOptions{})
	if err == nil {
		t.Fatalf("expected lock-busy error, got nil")
	}
	var oe *Error
	if !errors.As(err, &oe) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if !oe.Recoverable {
		t.Errorf("lock-busy error should be Recoverable")
	}
	if oe.DocsURL == "" {
		t.Errorf("lock-busy error should have DocsURL")
	}
}

func TestLock_ReleasedAfterInstall(t *testing.T) {
	dir := t.TempDir()
	c := &mockComponent{name: "samuel-skills"}
	o := New(c).WithHomeDir(dir)

	if _, err := o.Install(context.Background(), InstallOptions{}); err != nil {
		t.Fatalf("first install: %v", err)
	}
	// Lock file should be gone.
	if _, err := os.Stat(filepath.Join(dir, LockPath)); !os.IsNotExist(err) {
		t.Errorf("lock file should be removed after install; stat err = %v", err)
	}
	// Second install should succeed without LOCK error.
	if _, err := o.Install(context.Background(), InstallOptions{}); err != nil {
		t.Errorf("second install: %v", err)
	}
}

func TestLock_ReleasedAfterInstallError(t *testing.T) {
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
	if _, err := os.Stat(filepath.Join(dir, LockPath)); !os.IsNotExist(err) {
		t.Errorf("lock file should be removed after install error; stat err = %v", err)
	}
}

func TestPidAlive_NegativeAndZero(t *testing.T) {
	if pidAlive(0) {
		t.Errorf("pidAlive(0) should be false")
	}
	if pidAlive(-1) {
		t.Errorf("pidAlive(-1) should be false")
	}
}

func TestPidAlive_SelfIsAlive(t *testing.T) {
	if !pidAlive(os.Getpid()) {
		t.Errorf("pidAlive(self) should be true")
	}
}

// contains is a substring helper for cases where strings.Contains is
// awkward (avoids importing "strings" in every test).
func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// mustNotPanic wraps a function that returns an error so panics surface
// as test failures rather than crashing the whole run.
func mustNotPanic(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn()
}
