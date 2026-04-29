//go:build !unix

package orchestrator

import "runtime"

// acquireFileLock on non-Unix platforms returns an unsupported-platform
// error. Samuel v4.0 explicitly targets macOS and Linux only — Windows
// support is on the v4.x roadmap. Goreleaser should drop the Windows
// build matrix; if a Windows binary somehow gets built and run, this
// stub fails fast with a clear message rather than silently breaking.
//
// runtime.GOOS is included in the message so the user sees "GOOS=js"
// or "GOOS=windows" rather than a Windows-specific hint regardless of
// which !unix target they hit.
func acquireFileLock(home string) (release func(), err error) {
	_ = home
	return nil, &Error{
		Component:   NameOrchestrator,
		Problem:     "Samuel v4.0 does not support GOOS=" + runtime.GOOS,
		Cause:       "the orchestrator requires flock(2), unavailable on this platform",
		Fix:         "use Samuel v3.x on Windows, or run Samuel under WSL on Linux",
		DocsURL:     "https://samuel.dev/docs/v4-roadmap#windows",
		Recoverable: false,
	}
}
