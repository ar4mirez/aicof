package orchestrator

import (
	"errors"
	"strings"
	"testing"
)

func TestError_FormatsWithCause(t *testing.T) {
	e := &Error{
		Component: "gbrain",
		Problem:   "cannot register MCP server",
		Cause:     "claude not on PATH",
	}
	got := e.Error()
	want := "[gbrain] cannot register MCP server: claude not on PATH"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestError_FormatsWithoutCause(t *testing.T) {
	e := &Error{
		Component: "gstack",
		Problem:   "clone failed",
	}
	got := e.Error()
	want := "[gstack] clone failed"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestError_NilSafe(t *testing.T) {
	var e *Error
	if got := e.Error(); got != "" {
		t.Errorf("nil Error() = %q, want empty", got)
	}
	if got := e.Unwrap(); got != nil {
		t.Errorf("nil Unwrap() = %v, want nil", got)
	}
}

func TestError_Wrap_PopulatesCauseFromErr(t *testing.T) {
	base := &Error{Component: "samuel-skills", Problem: "symlink failed"}
	inner := errors.New("permission denied")
	wrapped := base.Wrap(inner)

	if wrapped.Cause != "permission denied" {
		t.Errorf("Wrap should populate Cause from err; got %q", wrapped.Cause)
	}
	if !errors.Is(wrapped, inner) {
		t.Errorf("errors.Is should traverse Wrap chain")
	}
}

func TestError_Wrap_PreservesExplicitCause(t *testing.T) {
	base := &Error{
		Component: "gbrain",
		Problem:   "register failed",
		Cause:     "explicit cause",
	}
	inner := errors.New("inner cause")
	wrapped := base.Wrap(inner)

	if wrapped.Cause != "explicit cause" {
		t.Errorf("Wrap should not overwrite explicit Cause; got %q", wrapped.Cause)
	}
	if !errors.Is(wrapped, inner) {
		t.Errorf("Wrap should still preserve underlying err for errors.Is")
	}
}

func TestError_Wrap_NilSafe(t *testing.T) {
	var e *Error
	if got := e.Wrap(errors.New("x")); got != nil {
		t.Errorf("nil.Wrap() = %v, want nil", got)
	}
}

func TestError_Wrap_DoesNotMutateReceiver(t *testing.T) {
	base := &Error{Component: "gstack", Problem: "x"}
	_ = base.Wrap(errors.New("inner"))
	if base.wrapped != nil || base.Cause != "" {
		t.Errorf("Wrap mutated receiver: wrapped=%v cause=%q", base.wrapped, base.Cause)
	}
}

func TestIsRecoverable_TrueForRecoverableErrors(t *testing.T) {
	e := &Error{Component: "x", Problem: "y", Recoverable: true}
	if !IsRecoverable(e) {
		t.Errorf("IsRecoverable should return true for Recoverable=true")
	}
}

func TestIsRecoverable_FalseForBareError(t *testing.T) {
	if IsRecoverable(errors.New("plain")) {
		t.Errorf("IsRecoverable should return false for non-orchestrator.Error")
	}
}

func TestIsRecoverable_TraversesWrapped(t *testing.T) {
	inner := &Error{Component: "x", Problem: "y", Recoverable: true}
	wrapper := errors.New("wrapper")
	combined := errors.Join(wrapper, inner)
	if !IsRecoverable(combined) {
		t.Errorf("IsRecoverable should traverse joined errors")
	}
}

func TestError_AsAcrossWrap(t *testing.T) {
	base := &Error{Component: "samuel-skills", Problem: "sync"}
	wrapped := base.Wrap(errors.New("inner"))
	var target *Error
	if !errors.As(wrapped, &target) {
		t.Fatalf("errors.As failed to extract *Error")
	}
	if !strings.Contains(target.Error(), "samuel-skills") {
		t.Errorf("extracted Error did not preserve Component")
	}
}
