package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
)

// LockPath is the path to the advisory lock file relative to the user's
// home directory. The orchestrator holds this file open while
// Install/Uninstall runs to serialize concurrent Samuel invocations.
const LockPath = ".claude/.samuel.lock"

// Orchestrator coordinates the lifecycle of a set of Components.
type Orchestrator struct {
	components []Component
	homeDir    string // root for the lock file; defaults to $HOME
	mu         sync.Mutex
}

// New constructs an Orchestrator with the given components, in the order
// they should be installed. Order matters: gstack first, gbrain second,
// samuel-skills third — matches the bundle dependency order from the
// design doc. Uninstall walks the same list in reverse.
func New(components ...Component) *Orchestrator {
	return &Orchestrator{components: components}
}

// WithHomeDir overrides the home directory used for the advisory lock.
// Tests use this to point at a temp dir.
func (o *Orchestrator) WithHomeDir(home string) *Orchestrator {
	o.homeDir = home
	return o
}

// Install runs each Component.Install in order, recording mutations as it
// goes. On any component failure, all already-applied mutations are rolled
// back in reverse LIFO order. The rollback error (if any) is joined to the
// install error so callers can see both.
func (o *Orchestrator) Install(ctx context.Context, opts InstallOptions) ([]InstallResult, error) {
	release, err := o.acquireLock(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	results := make([]InstallResult, 0, len(o.components))
	var applied []Mutation

	for _, c := range o.components {
		if shouldSkip(c, opts) {
			results = append(results, InstallResult{Component: c.Name(), Skipped: true})
			continue
		}
		res, ierr := c.Install(ctx, opts)
		if res.Component == "" {
			res.Component = c.Name()
		}
		if ierr != nil {
			rbErr := o.rollback(ctx, applied)
			if rbErr != nil {
				return results, fmt.Errorf("install %s: %w; rollback: %w", c.Name(), ierr, rbErr)
			}
			return results, fmt.Errorf("install %s: %w", c.Name(), ierr)
		}
		results = append(results, res)
		applied = append(applied, res.Mutations...)
	}
	return results, nil
}

// Doctor runs Check on every component and returns the per-component
// status. Doctor does NOT acquire the lock; concurrent doctor calls are
// safe (Check must not mutate state).
func (o *Orchestrator) Doctor(ctx context.Context) []HealthStatus {
	out := make([]HealthStatus, 0, len(o.components))
	for _, c := range o.components {
		s := c.Check(ctx)
		if s.Component == "" {
			s.Component = c.Name()
		}
		out = append(out, s)
	}
	return out
}

// Uninstall runs each Component.Uninstall in reverse order from Install.
// This unwinds dependencies in the correct order: samuel-skills first
// (project-level cleanup), gbrain second (MCP unregister), gstack last
// (the user owns gstack; we only uninstall it if they explicitly asked).
func (o *Orchestrator) Uninstall(ctx context.Context, opts UninstallOptions) ([]UninstallResult, error) {
	release, err := o.acquireLock(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	results := make([]UninstallResult, 0, len(o.components))
	for i := len(o.components) - 1; i >= 0; i-- {
		c := o.components[i]
		res, uerr := c.Uninstall(ctx, opts)
		if res.Component == "" {
			res.Component = c.Name()
		}
		if uerr != nil {
			return results, fmt.Errorf("uninstall %s: %w", c.Name(), uerr)
		}
		results = append(results, res)
	}
	return results, nil
}

// rollback runs Reverse on each mutation in reverse order. Errors are
// collected and joined; rollback continues even on partial failure
// (best-effort cleanup — leaving the system in a worse state than we
// started would be worse than recording all failures).
func (o *Orchestrator) rollback(ctx context.Context, muts []Mutation) error {
	var errs []error
	for i := len(muts) - 1; i >= 0; i-- {
		m := muts[i]
		if m.Reverse == nil {
			continue
		}
		if err := m.Reverse(ctx); err != nil {
			errs = append(errs, fmt.Errorf("rollback %s: %w", m.Path, err))
		}
	}
	return errors.Join(errs...)
}

// acquireLock writes the current PID into an advisory lock file using
// O_EXCL atomic creation. If the lock exists but the holding PID is dead
// (process crashed mid-install), the stale lock is evicted and the call
// retries once. Returns a release function that closes and removes the
// lock; callers MUST defer release.
func (o *Orchestrator) acquireLock(_ context.Context) (func(), error) {
	o.mu.Lock()
	home, err := o.resolveHome()
	if err != nil {
		o.mu.Unlock()
		return nil, err
	}
	lockFile := filepath.Join(home, LockPath)
	if mkErr := os.MkdirAll(filepath.Dir(lockFile), 0o700); mkErr != nil {
		o.mu.Unlock()
		return nil, fmt.Errorf("create lock dir: %w", mkErr)
	}

	f, openErr := o.tryOpenLock(lockFile)
	if openErr != nil {
		// Possible stale lock — read PID, check liveness, evict if dead.
		if data, readErr := os.ReadFile(lockFile); readErr == nil {
			if pid, perr := strconv.Atoi(string(data)); perr == nil && !pidAlive(pid) {
				_ = os.Remove(lockFile)
				f, openErr = o.tryOpenLock(lockFile)
			}
		}
	}
	if openErr != nil {
		o.mu.Unlock()
		return nil, &Error{
			Component:   "orchestrator",
			Problem:     "another samuel process is running",
			Cause:       openErr.Error(),
			Fix:         "wait for the other process to finish, or remove " + lockFile + " if no samuel process is actually running",
			DocsURL:     "https://samuel.dev/docs/errors/SAM-LOCK-001",
			Recoverable: true,
			Path:        lockFile,
		}
	}

	pid := strconv.Itoa(os.Getpid())
	if _, werr := io.WriteString(f, pid); werr != nil {
		_ = f.Close()
		_ = os.Remove(lockFile)
		o.mu.Unlock()
		return nil, fmt.Errorf("write lock pid: %w", werr)
	}

	release := func() {
		_ = f.Close()
		_ = os.Remove(lockFile)
		o.mu.Unlock()
	}
	return release, nil
}

func (o *Orchestrator) resolveHome() (string, error) {
	if o.homeDir != "" {
		return o.homeDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", &Error{
			Component:   "orchestrator",
			Problem:     "cannot determine home directory",
			Cause:       err.Error(),
			Fix:         "set HOME environment variable",
			Recoverable: true,
		}
	}
	return home, nil
}

func (o *Orchestrator) tryOpenLock(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
}

// pidAlive checks whether a process with the given PID exists. On Unix,
// signal 0 is the canonical "is this process alive" probe — it does not
// deliver a signal, just verifies the process can receive one.
func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

// shouldSkip reports whether the orchestrator should skip a component
// based on InstallOptions. The component name is the routing key.
func shouldSkip(c Component, opts InstallOptions) bool {
	switch c.Name() {
	case "gstack":
		return opts.SkipGstack
	case "gbrain":
		return opts.SkipGbrain
	}
	return false
}
