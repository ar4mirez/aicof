package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// rollbackTimeout is the deadline applied to a fresh rollback context.
// Rollback runs on a separate context so install failures triggered by
// ctx cancellation do not also abort cleanup (multi-voice review finding).
const rollbackTimeout = 30 * time.Second

// LockPath is the path to the advisory lock file relative to the user's
// home directory. The file is opened and held under flock(2) for the
// duration of Install/Uninstall — the kernel auto-releases the lock on
// process death, so there is no PID-eviction race. The file persists
// across runs; its body is informational (PID of the holder).
const LockPath = ".claude/.samuel.lock"

// Orchestrator coordinates the lifecycle of a set of Components.
// Concurrent invocations (in-process or cross-process) are serialized
// by flock(2) on LockPath — no in-process mutex is needed.
type Orchestrator struct {
	components []Component
	homeDir    string // root for the lock file; defaults to $HOME
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
// goes. On any component failure, all already-applied mutations PLUS any
// partial mutations the failing component reported are rolled back in
// reverse LIFO order. Rollback runs on a fresh context with rollbackTimeout
// so install failures triggered by ctx cancellation do not also abort
// cleanup.
func (o *Orchestrator) Install(ctx context.Context, opts InstallOptions) ([]InstallResult, error) {
	release, err := o.acquireLock()
	if err != nil {
		return nil, err
	}
	defer release()

	results := make([]InstallResult, 0, len(o.components))
	applied := make([]Mutation, 0, len(o.components)*4)

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
			// Include the failing component's partial mutations in the
			// rollback queue. Components are required by contract to
			// stage atomically, but defense-in-depth: don't trust them.
			applied = append(applied, res.Mutations...)
			results = append(results, res)
			rbCtx, cancel := context.WithTimeout(context.Background(), rollbackTimeout)
			defer cancel()
			rbErr := o.rollback(rbCtx, applied)
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
//
// Uninstall is BEST-EFFORT: a failure in one component does not stop
// later components from running. All component errors are collected and
// returned as a joined error, mirroring rollback semantics. The user's
// worst case becomes "most things uninstalled, here are the failures"
// rather than "stuck halfway with no recovery."
func (o *Orchestrator) Uninstall(ctx context.Context, opts UninstallOptions) ([]UninstallResult, error) {
	release, err := o.acquireLock()
	if err != nil {
		return nil, err
	}
	defer release()

	results := make([]UninstallResult, 0, len(o.components))
	var errs []error
	for i := len(o.components) - 1; i >= 0; i-- {
		c := o.components[i]
		res, uerr := c.Uninstall(ctx, opts)
		if res.Component == "" {
			res.Component = c.Name()
		}
		results = append(results, res)
		if uerr != nil {
			errs = append(errs, fmt.Errorf("uninstall %s: %w", c.Name(), uerr))
		}
	}
	if len(errs) > 0 {
		return results, errors.Join(errs...)
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

// acquireLock takes the orchestrator's advisory lock. Cross-process and
// in-process exclusion is provided by flock(2) on a persistent lock file
// (see lock_unix.go). The returned release function closes the file
// descriptor, which the kernel uses to release the flock.
func (o *Orchestrator) acquireLock() (func(), error) {
	home, err := o.resolveHome()
	if err != nil {
		return nil, err
	}
	return acquireFileLock(home)
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

// shouldSkip reports whether the orchestrator should skip a component
// based on InstallOptions. The component name (case-insensitive) is the
// routing key — case-folding hardens against typos in component
// implementations (multi-voice review finding).
func shouldSkip(c Component, opts InstallOptions) bool {
	switch strings.ToLower(c.Name()) {
	case NameGstack:
		return opts.SkipGstack
	case NameGbrain:
		return opts.SkipGbrain
	}
	return false
}
