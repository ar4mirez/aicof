package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAdmin_RegisteredAtRoot(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "admin" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected adminCmd to be registered as a child of rootCmd")
	}
}

func TestAdmin_HasConfigChild(t *testing.T) {
	found := false
	for _, c := range adminCmd.Commands() {
		if c.Name() == "config" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected adminCmd to have 'config' as a child; got: %v", commandNames(adminCmd.Commands()))
	}
}

func TestAdmin_HasSyncChild(t *testing.T) {
	found := false
	for _, c := range adminCmd.Commands() {
		if c.Name() == "sync" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected adminCmd to have 'sync' as a child; got: %v", commandNames(adminCmd.Commands()))
	}
}

func TestAdmin_ConfigKeepsSubcommands(t *testing.T) {
	// configCmd moved under adminCmd; its list/get/set children must still attach.
	want := map[string]bool{"list": false, "get": false, "set": false}
	for _, c := range configCmd.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("expected configCmd to have %q child after move under admin", name)
		}
	}
}

func TestLegacyConfig_RegisteredHidden(t *testing.T) {
	if !legacyConfigCmd.Hidden {
		t.Error("expected legacyConfigCmd.Hidden = true")
	}
	// Cobra's Deprecated field auto-prints and ignores SAMUEL_NO_DEPRECATION,
	// so we deliberately leave it empty; redirectAndRun owns the deprecation message.
	if legacyConfigCmd.Deprecated != "" {
		t.Errorf("expected legacyConfigCmd.Deprecated to be empty (managed by redirectAndRun), got %q", legacyConfigCmd.Deprecated)
	}
}

func TestLegacySync_RegisteredHidden(t *testing.T) {
	if !legacySyncCmd.Hidden {
		t.Error("expected legacySyncCmd.Hidden = true")
	}
	if legacySyncCmd.Deprecated != "" {
		t.Errorf("expected legacySyncCmd.Deprecated to be empty (managed by redirectAndRun), got %q", legacySyncCmd.Deprecated)
	}
}

func TestLegacyConfig_HasFamiliarSubcommands(t *testing.T) {
	want := map[string]bool{"list": false, "get": false, "set": false}
	for _, c := range legacyConfigCmd.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("expected legacyConfigCmd to have %q legacy child", name)
		}
	}
}

func TestLegacySync_HasFlags(t *testing.T) {
	// Mirrors the real syncCmd flags so legacy invocations still parse.
	for _, name := range []string{"depth", "force", "dry-run"} {
		if legacySyncCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected legacySyncCmd to expose --%s flag", name)
		}
	}
}

func TestRoot_NoDeprecationFlag(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("no-deprecation")
	if f == nil {
		t.Fatal("expected --no-deprecation persistent flag on rootCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("expected --no-deprecation default to be false, got %q", f.DefValue)
	}
}

func commandNames(cmds []*cobra.Command) []string {
	out := make([]string, 0, len(cmds))
	for _, c := range cmds {
		out = append(out, c.Name())
	}
	return out
}
