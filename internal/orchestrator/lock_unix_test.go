//go:build unix

package orchestrator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

// holdFlock takes an exclusive flock on the lock file at the given home
// dir from the test process. Returns a release function the caller MUST
// call. Used to simulate a live cross-process lock holder.
func holdFlock(t *testing.T, home string) func() {
	t.Helper()
	lockFile := filepath.Join(home, LockPath)
	if err := os.MkdirAll(filepath.Dir(lockFile), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("open lock: %v", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		t.Fatalf("flock: %v", err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}
}

func TestFlock_LiveLockRejectsInstall(t *testing.T) {
	dir := t.TempDir()
	releaseTestLock := holdFlock(t, dir)
	defer releaseTestLock()

	c := &mockComponent{name: "samuel-skills"}
	o := New(c).WithHomeDir(dir)

	_, err := o.Install(context.Background(), InstallOptions{})
	if err == nil {
		t.Fatalf("expected lock-busy error while flock is held, got nil")
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
	// Component must NOT have been called.
	if got := c.installCalls.Load(); got != 0 {
		t.Errorf("Install should not call component when lock fails; got %d calls", got)
	}
}

func TestFlock_LiveLockRejectsUninstall(t *testing.T) {
	dir := t.TempDir()
	releaseTestLock := holdFlock(t, dir)
	defer releaseTestLock()

	c := &mockComponent{name: "samuel-skills"}
	o := New(c).WithHomeDir(dir)

	_, err := o.Uninstall(context.Background(), UninstallOptions{All: true})
	if err == nil {
		t.Fatalf("expected lock-busy error from Uninstall")
	}
	var oe *Error
	if !errors.As(err, &oe) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if !oe.Recoverable {
		t.Errorf("Uninstall lock-busy error should be Recoverable")
	}
	if got := c.uninstallCalls.Load(); got != 0 {
		t.Errorf("Uninstall should not call component when lock fails; got %d calls", got)
	}
}

func TestFlock_ReleasedThenAcquired(t *testing.T) {
	// Hold flock, release it, then verify Install succeeds. Proves the
	// kernel-managed release path works end-to-end.
	dir := t.TempDir()
	release := holdFlock(t, dir)
	release()

	c := &mockComponent{name: "samuel-skills"}
	o := New(c).WithHomeDir(dir)
	if _, err := o.Install(context.Background(), InstallOptions{}); err != nil {
		t.Errorf("Install should succeed after test lock released; got %v", err)
	}
}

func TestFlock_ConcurrentInstallsSerialize(t *testing.T) {
	// Two goroutines calling Install on different Orchestrator instances
	// pointing at the same home dir must serialize via flock. Unlike the
	// old O_EXCL+PID gate-based test, this is deterministic: the second
	// caller fails fast with "another samuel process is running" if the
	// first is mid-install.
	dir := t.TempDir()

	firstStarted := make(chan struct{})
	gate := make(chan struct{})
	makeOrchestrator := func(name string, blocking bool) (*Orchestrator, *mockComponent) {
		c := &mockComponent{name: name}
		c.installFn = func(_ context.Context, _ InstallOptions) (InstallResult, error) {
			if blocking {
				close(firstStarted)
				<-gate
			}
			return InstallResult{}, nil
		}
		return New(c).WithHomeDir(dir), c
	}

	o1, _ := makeOrchestrator("samuel-skills", true)
	o2, _ := makeOrchestrator("samuel-skills", false)

	var wg sync.WaitGroup
	var (
		err1, err2 error
		mu         sync.Mutex
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, e := o1.Install(context.Background(), InstallOptions{})
		mu.Lock()
		err1 = e
		mu.Unlock()
	}()

	// Wait until o1 is mid-install (holding the lock), then attempt o2.
	select {
	case <-firstStarted:
	case <-time.After(2 * time.Second):
		close(gate)
		wg.Wait()
		t.Fatalf("first install never started")
	}

	_, err2 = o2.Install(context.Background(), InstallOptions{})
	close(gate)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if err1 != nil {
		t.Errorf("first install (lock holder) should succeed; got %v", err1)
	}
	if err2 == nil {
		t.Fatalf("second install must fail with lock-busy while first holds the lock")
	}
	var oe *Error
	if !errors.As(err2, &oe) {
		t.Fatalf("expected *Error from second install, got %T: %v", err2, err2)
	}
	if !contains(oe.Problem, "samuel process") {
		t.Errorf("expected lock-busy Problem, got %q", oe.Problem)
	}
}
