package commands

import (
	"testing"
)

func TestAdminDiff_RegisteredUnderAdmin(t *testing.T) {
	found := false
	for _, c := range adminCmd.Commands() {
		if c.Name() == "diff" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected adminDiffCmd registered under admin; got: %v", commandNames(adminCmd.Commands()))
	}
}

func TestAdminDiff_HasFlags(t *testing.T) {
	for _, name := range []string{"installed", "components"} {
		if adminDiffCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected adminDiffCmd to expose --%s flag", name)
		}
	}
}

func TestAdminDiff_NotHidden(t *testing.T) {
	if adminDiffCmd.Hidden {
		t.Error("expected adminDiffCmd to be visible")
	}
}

func TestUpdate_PreviewFlag_Registered(t *testing.T) {
	if updateCmd.Flags().Lookup("preview") == nil {
		t.Fatal("expected updateCmd to expose --preview flag")
	}
}

func TestUpdate_DiffFlag_HiddenButPresent(t *testing.T) {
	f := updateCmd.Flags().Lookup("diff")
	if f == nil {
		t.Fatal("expected updateCmd to keep --diff flag (legacy alias)")
	}
	if !f.Hidden {
		t.Error("expected --diff flag to be Hidden in v3.0.0 (use --preview)")
	}
}
