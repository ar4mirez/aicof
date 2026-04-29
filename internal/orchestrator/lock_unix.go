//go:build unix

package orchestrator

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// acquireFileLock opens the lock file (creating if needed), takes a
// non-blocking exclusive flock(2) lock, writes the current PID for
// diagnostics, and returns a release function that closes the file
// (which the kernel uses to release the lock).
//
// flock semantics on Linux/macOS:
//   - LOCK_EX: exclusive lock
//   - LOCK_NB: non-blocking; return EWOULDBLOCK/EAGAIN if held
//   - The lock is associated with the open file description. Closing
//     the fd (or process death) releases it automatically. There is
//     no PID file to evict, no Read+Remove+OpenFile race.
//
// The lock file persists across runs; the PID body is purely
// informational. Flock provides the exclusion guarantee. The fd is
// opened with O_CLOEXEC so child processes (gstack setup, claude mcp,
// etc.) do not inherit the lock and accidentally hold it past Samuel's
// own lifetime.
func acquireFileLock(home string) (release func(), err error) {
	lockFile := filepath.Join(home, LockPath)
	if mkErr := os.MkdirAll(filepath.Dir(lockFile), 0o700); mkErr != nil {
		return nil, (&Error{
			Component:   NameOrchestrator,
			Problem:     "cannot create lock directory",
			Fix:         "check write permissions on " + filepath.Dir(lockFile),
			Recoverable: true,
			Path:        filepath.Dir(lockFile),
		}).Wrap(mkErr)
	}

	// O_CLOEXEC: don't leak the locked fd into child processes. If we
	// shell out to gstack/claude during Install, the child inherits the
	// flock through fd inheritance and would keep holding it past
	// Samuel's exit if it execs into a daemon.
	f, openErr := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR|syscall.O_CLOEXEC, 0o600)
	if openErr != nil {
		return nil, (&Error{
			Component:   NameOrchestrator,
			Problem:     "cannot open lock file",
			Fix:         "check permissions on " + lockFile,
			Recoverable: true,
			Path:        lockFile,
		}).Wrap(openErr)
	}

	if flockErr := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); flockErr != nil {
		_ = f.Close()
		// EWOULDBLOCK/EAGAIN means another process holds the lock.
		// Anything else (ENOLCK, ENOSYS, EBADF, EOPNOTSUPP from NFS)
		// is a different failure class and gets its own message so
		// the user can diagnose correctly.
		if errors.Is(flockErr, syscall.EWOULDBLOCK) || errors.Is(flockErr, syscall.EAGAIN) {
			holderHint := readHolderHint(lockFile)
			return nil, (&Error{
				Component:   NameOrchestrator,
				Problem:     "another samuel process is running",
				Cause:       fmt.Sprintf("flock busy (holder: %s)", holderHint),
				Fix:         "wait for the other samuel process to finish, then re-run",
				DocsURL:     "https://samuel.dev/docs/errors/SAM-LOCK-001",
				Recoverable: true,
				Path:        lockFile,
			}).Wrap(flockErr)
		}
		return nil, (&Error{
			Component:   NameOrchestrator,
			Problem:     "cannot acquire lock (flock failed)",
			Fix:         "verify the filesystem at " + lockFile + " supports flock(2); NFS may require lockd",
			DocsURL:     "https://samuel.dev/docs/errors/SAM-LOCK-002",
			Recoverable: true,
			Path:        lockFile,
		}).Wrap(flockErr)
	}

	// Write our PID for diagnostics. Truncate first so a stale PID
	// from a prior crashed run is replaced. Best-effort — flock is
	// what actually provides exclusion.
	if _, seekErr := f.Seek(0, 0); seekErr == nil {
		_ = f.Truncate(0)
		_, _ = io.WriteString(f, strconv.Itoa(os.Getpid()))
		_ = f.Sync()
	}

	// Wrap release in sync.Once so a defensive double-release does not
	// LOCK_UN on a closed fd whose number may have been reused for an
	// unrelated file in the same process.
	var once sync.Once
	release = func() {
		once.Do(func() {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			_ = f.Close()
			// Do NOT remove the lock file. Removing it would race with
			// another acquirer that just took flock on the same path —
			// they'd hold flock on a now-deleted inode while a third
			// acquirer creates a fresh inode at the same path and takes
			// flock on it, defeating exclusion. Persistent lock file
			// plus kernel-managed flock is the safe combination.
		})
	}
	return release, nil
}

// readHolderHint reads the lock file's PID body for diagnostic display.
// The bytes on disk are user-controlled (a same-uid attacker, or a
// crashed prior process, could plant arbitrary content), so we validate
// the body parses as a PID before rendering it. Anything else surfaces
// as "unknown" rather than letting raw bytes (control chars, ANSI
// escapes, multi-MB blobs) flow into error output.
func readHolderHint(lockFile string) string {
	const maxPidBytes = 32
	data, err := os.ReadFile(lockFile)
	if err != nil || len(data) == 0 {
		return "unknown"
	}
	if len(data) > maxPidBytes {
		data = data[:maxPidBytes]
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return "unknown"
	}
	return fmt.Sprintf("pid=%d", pid)
}
