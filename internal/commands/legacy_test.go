package commands

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// captureStderr captures stderr output during fn execution.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	fn()

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestDeprecationSuppressed_EnvVar(t *testing.T) {
	t.Setenv("SAMUEL_NO_DEPRECATION", "1")
	if !deprecationSuppressed(rootCmd) {
		t.Error("expected deprecationSuppressed to be true when SAMUEL_NO_DEPRECATION=1")
	}
}

func TestDeprecationSuppressed_EnvVarOtherValues(t *testing.T) {
	// Only "1" suppresses. Other truthy strings should not.
	for _, val := range []string{"true", "yes", "0", ""} {
		t.Run("val="+val, func(t *testing.T) {
			t.Setenv("SAMUEL_NO_DEPRECATION", val)
			if val == "1" {
				return
			}
			if deprecationSuppressed(rootCmd) {
				t.Errorf("expected deprecationSuppressed to be false for SAMUEL_NO_DEPRECATION=%q", val)
			}
		})
	}
}

func TestDeprecationSuppressed_Flag(t *testing.T) {
	t.Setenv("SAMUEL_NO_DEPRECATION", "")

	// Set the flag to true, then restore it after the test.
	flag := rootCmd.PersistentFlags().Lookup("no-deprecation")
	if flag == nil {
		t.Fatal("--no-deprecation flag not registered on rootCmd")
	}
	original := flag.Value.String()
	t.Cleanup(func() {
		_ = flag.Value.Set(original)
	})

	_ = flag.Value.Set("true")
	if !deprecationSuppressed(rootCmd) {
		t.Error("expected deprecationSuppressed to be true when --no-deprecation=true")
	}

	_ = flag.Value.Set("false")
	if deprecationSuppressed(rootCmd) {
		t.Error("expected deprecationSuppressed to be false when --no-deprecation=false")
	}
}

func TestDeprecationSuppressed_NilCmd(t *testing.T) {
	t.Setenv("SAMUEL_NO_DEPRECATION", "")
	if deprecationSuppressed(nil) {
		t.Error("expected deprecationSuppressed to be false for nil cmd")
	}
}

func TestRedirectAndRun_CallsHandler(t *testing.T) {
	t.Setenv("SAMUEL_NO_DEPRECATION", "1") // silence stderr during the test
	called := false
	handler := func(cmd *cobra.Command, args []string) error {
		called = true
		return nil
	}
	wrapped := redirectAndRun("samuel admin config", handler)

	// Build a minimal cobra command to satisfy the helper's signature.
	cmd := &cobra.Command{Use: "config"}
	rootCmd.AddCommand(cmd)
	t.Cleanup(func() { rootCmd.RemoveCommand(cmd) })

	if err := wrapped(cmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected wrapped handler to be called")
	}
}

func TestRedirectAndRun_PrintsDeprecation(t *testing.T) {
	t.Setenv("SAMUEL_NO_DEPRECATION", "")

	handler := func(cmd *cobra.Command, args []string) error { return nil }
	wrapped := redirectAndRun("samuel admin config list", handler)

	// Build a parent + child to give the cmd a realistic CommandPath().
	parent := &cobra.Command{Use: "config"}
	child := &cobra.Command{Use: "list"}
	parent.AddCommand(child)
	rootCmd.AddCommand(parent)
	t.Cleanup(func() { rootCmd.RemoveCommand(parent) })

	stderr := captureStderr(t, func() {
		_ = wrapped(child, []string{})
	})

	if !strings.Contains(stderr, "samuel config list") {
		t.Errorf("expected stderr to mention legacy command path, got: %s", stderr)
	}
	if !strings.Contains(stderr, "samuel admin config list") {
		t.Errorf("expected stderr to mention new command path, got: %s", stderr)
	}
	if !strings.Contains(stderr, "SAMUEL_NO_DEPRECATION") {
		t.Errorf("expected stderr to mention opt-out env var, got: %s", stderr)
	}
}

func TestRedirectAndRun_SuppressedByEnv(t *testing.T) {
	t.Setenv("SAMUEL_NO_DEPRECATION", "1")

	called := false
	handler := func(cmd *cobra.Command, args []string) error {
		called = true
		return nil
	}
	wrapped := redirectAndRun("samuel admin config", handler)

	cmd := &cobra.Command{Use: "config"}
	rootCmd.AddCommand(cmd)
	t.Cleanup(func() { rootCmd.RemoveCommand(cmd) })

	stderr := captureStderr(t, func() {
		_ = wrapped(cmd, []string{})
	})

	if stderr != "" {
		t.Errorf("expected empty stderr when SAMUEL_NO_DEPRECATION=1, got: %s", stderr)
	}
	if !called {
		t.Error("expected handler to still be called when deprecation is suppressed")
	}
}
