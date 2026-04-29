//go:build unix

package orchestrator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// acquireFileLock opens the lock file (creating if needed), takes a
// non-blocking exclusive flock(2) lock, writes the current PID for
// diagnostics, and returns a release function that closes the file
// (which the kernel uses to release the lock).
//
// flock semantics on Linux/macOS:
//   - LOCK_EX: exclusive lock
//   - LOCK_NB: non-blocking; return EWOULDBLOCK if held
//   - The lock is associated with the open file description. Closing
//     the fd (or process death) releases it automatically. There is
//     no PID file to evict, no Read+Remove+OpenFile race.
//
// The lock file itself persists across runs — the PID body is purely
// informational ("which process held the lock"). Flock provides the
// exclusion guarantee.
func acquireFileLock(home string) (release func(), err error) {
	lockFile := filepath.Join(home, LockPath)
	if mkErr := os.MkdirAll(filepath.Dir(lockFile), 0o700); mkErr != nil {
		return nil, &Error{
			Component:   "orchestrator",
			Problem:     "cannot create lock directory",
			Cause:       mkErr.Error(),
			Fix:         "check write permissions on " + filepath.Dir(lockFile),
			Recoverable: true,
			Path:        filepath.Dir(lockFile),
		}
	}

	f, openErr := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0o600)
	if openErr != nil {
		return nil, &Error{
			Component:   "orchestrator",
			Problem:     "cannot open lock file",
			Cause:       openErr.Error(),
			Fix:         "check permissions on " + lockFile,
			Recoverable: true,
			Path:        lockFile,
		}
	}

	if flockErr := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); flockErr != nil {
		// Read holding PID for diagnostic message. Best-effort — if
		// the file is empty, we still report a clear error.
		holdingPid, _ := os.ReadFile(lockFile)
		_ = f.Close()
		holderHint := "unknown"
		if pidStr := string(holdingPid); pidStr != "" {
			holderHint = "pid=" + pidStr
		}
		return nil, &Error{
			Component:   "orchestrator",
			Problem:     "another samuel process is running",
			Cause:       fmt.Sprintf("flock: %v (holder: %s)", flockErr, holderHint),
			Fix:         "wait for the other samuel process to finish, then re-run",
			DocsURL:     "https://samuel.dev/docs/errors/SAM-LOCK-001",
			Recoverable: true,
			Path:        lockFile,
		}
	}

	// Write our PID for diagnostics. Truncate first so a stale PID
	// from a prior crashed run is replaced. Best-effort — flock is
	// what actually provides exclusion.
	if _, seekErr := f.Seek(0, 0); seekErr == nil {
		_ = f.Truncate(0)
		_, _ = io.WriteString(f, strconv.Itoa(os.Getpid()))
		_ = f.Sync()
	}

	release = func() {
		// LOCK_UN explicitly is a no-op on most systems (close releases
		// flock too) but is documented behavior. Keep it for clarity.
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
		// Do NOT remove the lock file. Removing it would race with
		// another acquirer that just took flock on the same path —
		// they'd hold flock on a now-deleted inode while a third
		// acquirer creates a fresh inode at the same path and takes
		// flock on it, defeating exclusion. Persistent lock file +
		// kernel-managed flock is the safe combination.
	}
	return release, nil
}
