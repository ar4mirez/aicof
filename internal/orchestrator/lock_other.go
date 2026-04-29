//go:build !unix

package orchestrator

// acquireFileLock on non-Unix platforms returns an unsupported-platform
// error. Samuel v4.0 explicitly targets macOS and Linux only — Windows
// support is on the v4.x roadmap. Goreleaser should drop the Windows
// build matrix; if a Windows binary somehow gets built and run, this
// stub fails fast with a clear message rather than silently breaking.
func acquireFileLock(home string) (release func(), err error) {
	_ = home
	return nil, &Error{
		Component:   "orchestrator",
		Problem:     "Samuel v4.0 does not support this platform",
		Cause:       "the orchestrator requires flock(2), which is unavailable on this OS",
		Fix:         "use Samuel v3.x on Windows, or run Samuel under WSL on Linux",
		DocsURL:     "https://samuel.dev/docs/v4-roadmap#windows",
		Recoverable: false,
	}
}
