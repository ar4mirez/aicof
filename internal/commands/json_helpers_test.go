package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

func TestCommandPath_NilCmd(t *testing.T) {
	if got := commandPath(nil); got != "" {
		t.Errorf("commandPath(nil) = %q, want empty", got)
	}
}

func TestCommandPath_StripsBinaryPrefix(t *testing.T) {
	// commandPath should strip the leading "samuel " from cobra's CommandPath().
	// Build a small isolated tree so this test doesn't depend on rootCmd state.
	root := &cobra.Command{Use: "samuel"}
	parent := &cobra.Command{Use: "admin"}
	child := &cobra.Command{Use: "config"}
	leaf := &cobra.Command{Use: "list"}
	root.AddCommand(parent)
	parent.AddCommand(child)
	child.AddCommand(leaf)

	tests := []struct {
		cmd  *cobra.Command
		want string
	}{
		// Root keeps its full name — TrimPrefix only fires for paths that
		// continue past "samuel ". A handler emitting JSON from root context
		// would record "samuel"; in practice no handler does, but the helper
		// should not panic or strip the binary name into "".
		{root, "samuel"},
		{parent, "admin"},
		{child, "admin config"},
		{leaf, "admin config list"},
	}
	for _, tt := range tests {
		if got := commandPath(tt.cmd); got != tt.want {
			t.Errorf("commandPath(%s) = %q, want %q", tt.cmd.CommandPath(), got, tt.want)
		}
	}
}

func TestPrintJSONForCmd_EmitsInvokedPath(t *testing.T) {
	root := &cobra.Command{Use: "samuel"}
	leaf := &cobra.Command{Use: "ls"}
	root.AddCommand(leaf)

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	PrintJSONForCmd(leaf, map[string]string{"hello": "world"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	var resp ui.JSONResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}

	if resp.SchemaVersion != 3 {
		t.Errorf("expected schemaVersion=3, got %d", resp.SchemaVersion)
	}
	if resp.Command != "ls" {
		t.Errorf("expected command='ls', got %q", resp.Command)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestPrintJSONErrorForCmd_EmitsInvokedPath(t *testing.T) {
	root := &cobra.Command{Use: "samuel"}
	parent := &cobra.Command{Use: "run"}
	leaf := &cobra.Command{Use: "done"}
	root.AddCommand(parent)
	parent.AddCommand(leaf)

	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	PrintJSONErrorForCmd(leaf, errors.New("boom"))

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	var resp ui.JSONResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}

	if resp.SchemaVersion != 3 {
		t.Errorf("expected schemaVersion=3, got %d", resp.SchemaVersion)
	}
	if resp.Command != "run done" {
		t.Errorf("expected command='run done', got %q", resp.Command)
	}
	if resp.Success {
		t.Error("expected success=false")
	}
	if resp.Error != "boom" {
		t.Errorf("expected error='boom', got %q", resp.Error)
	}
}

func TestRoot_SuggestionsMinimumDistance(t *testing.T) {
	if rootCmd.SuggestionsMinimumDistance != 2 {
		t.Errorf("expected SuggestionsMinimumDistance=2, got %d", rootCmd.SuggestionsMinimumDistance)
	}
}

func TestUI_JSONSchemaVersion(t *testing.T) {
	if ui.JSONSchemaVersion != 3 {
		t.Errorf("expected JSONSchemaVersion=3, got %d", ui.JSONSchemaVersion)
	}
}

// TestPrintJSON_EnvelopeIncludesSchemaVersion is a regression guard: every
// JSON envelope written by ui.PrintJSON must carry schemaVersion=3 so that
// consumers can branch on it. This test catches accidental removals of the
// SchemaVersion field from the JSONResponse struct.
func TestPrintJSON_EnvelopeIncludesSchemaVersion(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	ui.PrintJSON("test", map[string]string{"k": "v"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if !strings.Contains(buf.String(), `"schemaVersion": 3`) {
		t.Errorf("expected envelope to include schemaVersion=3, got: %s", buf.String())
	}
}
