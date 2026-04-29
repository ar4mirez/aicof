package orchestrator

import (
	"errors"
	"fmt"
)

// Error is the structured error type every orchestrator component returns
// instead of bare errors.New / fmt.Errorf strings. It carries enough
// context for samuel doctor and CLI rendering to give the user actionable
// feedback.
//
// The Samuel CLI renders these across multiple lines in interactive mode:
//
//	Error: Cannot register gbrain MCP server
//	  Cause: gbrain not found on PATH
//	  Fix:   bun install -g gbrain
//	  Docs:  https://samuel.dev/docs/errors/SAM-MCP-001
//
// The Error method itself returns a single-line string suitable for logs.
type Error struct {
	// Component identifies which Component produced the error.
	Component string
	// Problem describes what failed in one short sentence.
	Problem string
	// Cause is the underlying root cause, often a wrapped error string.
	Cause string
	// Fix is the recommended remediation. Should be copy-paste-able when
	// possible.
	Fix string
	// DocsURL points to a documentation page covering this error class.
	// Optional but encouraged.
	DocsURL string
	// Recoverable signals whether the user can fix this themselves
	// (true) vs. needing to file a bug (false).
	Recoverable bool
	// Path is the filesystem path involved, when relevant.
	Path string
	// wrapped preserves the original error chain for errors.Is /
	// errors.As traversal.
	wrapped error
}

// Error formats the structured fields into a single line suitable for
// logging.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Component, e.Problem, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Component, e.Problem)
}

// Unwrap returns the wrapped error so errors.Is and errors.As work across
// the orchestrator.Error boundary. Safe to call on a nil receiver.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.wrapped
}

// Wrap returns a copy of e with err preserved as the underlying cause.
// If Cause is empty, it is populated from err.Error().
func (e *Error) Wrap(err error) *Error {
	if e == nil {
		return nil
	}
	cp := *e
	cp.wrapped = err
	if cp.Cause == "" && err != nil {
		cp.Cause = err.Error()
	}
	return &cp
}

// IsRecoverable reports whether err carries Recoverable=true, treating
// non-orchestrator.Error values as non-recoverable.
func IsRecoverable(err error) bool {
	var oe *Error
	if errors.As(err, &oe) {
		return oe.Recoverable
	}
	return false
}
