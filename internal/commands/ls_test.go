package commands

import (
	"strings"
	"testing"
)

func TestLs_RegisteredAtRoot(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "ls" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected lsCmd to be registered as a child of rootCmd")
	}
}

func TestLs_HasFlags(t *testing.T) {
	wantFlags := []string{"all", "type", "detail", "preview", "no-related", "limit"}
	for _, name := range wantFlags {
		if lsCmd.Flags().Lookup(name) == nil {
			t.Errorf("expected lsCmd to expose --%s flag", name)
		}
	}
}

func TestLs_NotHidden(t *testing.T) {
	if lsCmd.Hidden {
		t.Error("expected lsCmd to be visible (it's the new primary verb)")
	}
}

func TestLegacyList_HiddenAndForwards(t *testing.T) {
	if !listCmd.Hidden {
		t.Error("expected legacy listCmd.Hidden = true")
	}
	if listCmd.Short != "[DEPRECATED] Use 'samuel ls'" {
		t.Errorf("expected listCmd.Short to mark deprecation, got %q", listCmd.Short)
	}
	// listCmd retains its --available and --type flags so legacy invocations
	// keep working until v3.1.0.
	if listCmd.Flags().Lookup("available") == nil {
		t.Error("expected legacy listCmd to keep --available flag")
	}
}

func TestLegacySearch_HiddenAndForwards(t *testing.T) {
	if !searchCmd.Hidden {
		t.Error("expected legacy searchCmd.Hidden = true")
	}
	if searchCmd.Flags().Lookup("limit") == nil {
		t.Error("expected legacy searchCmd to keep --limit flag")
	}
}

func TestLegacyInfo_HiddenAndForwards(t *testing.T) {
	if !infoCmd.Hidden {
		t.Error("expected legacy infoCmd.Hidden = true")
	}
	if infoCmd.Flags().Lookup("preview") == nil {
		t.Error("expected legacy infoCmd to keep --preview flag")
	}
}

func TestLegacyDiff_HiddenAndForwards(t *testing.T) {
	if !diffCmd.Hidden {
		t.Error("expected legacy diffCmd.Hidden = true")
	}
}

func TestResolveLsDetail_UnambiguousFramework(t *testing.T) {
	typ, name, err := resolveLsDetail("react", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if typ != "framework" {
		t.Errorf("expected type=framework, got %q", typ)
	}
	if name != "react" {
		t.Errorf("expected name=react, got %q", name)
	}
}

func TestResolveLsDetail_UnambiguousLanguage(t *testing.T) {
	typ, _, err := resolveLsDetail("typescript", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if typ != "language" {
		t.Errorf("expected type=language, got %q", typ)
	}
}

func TestResolveLsDetail_UnambiguousWorkflow(t *testing.T) {
	typ, _, err := resolveLsDetail("create-prd", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if typ != "workflow" {
		t.Errorf("expected type=workflow, got %q", typ)
	}
}

func TestResolveLsDetail_NotFound(t *testing.T) {
	_, _, err := resolveLsDetail("definitely-not-a-real-component", "")
	if err == nil {
		t.Fatal("expected error for non-existent component")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to mention 'not found', got: %v", err)
	}
}

func TestResolveLsDetail_TypeFilterRespected(t *testing.T) {
	// Even though "react" is unambiguously a framework, an explicit --type=language
	// should fail (no language named "react").
	_, _, err := resolveLsDetail("react", "language")
	if err == nil {
		t.Fatal("expected error when --type contradicts component existence")
	}
}

func TestResolveLsDetail_TypeFilterMatches(t *testing.T) {
	typ, name, err := resolveLsDetail("react", "framework")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if typ != "framework" || name != "react" {
		t.Errorf("expected (framework, react), got (%s, %s)", typ, name)
	}
}

func TestResolveLsDetail_SkillRejected(t *testing.T) {
	_, _, err := resolveLsDetail("commit-message", "skill")
	if err == nil {
		t.Fatal("expected error when --type=skill (use 'samuel skill info' instead)")
	}
	if !strings.Contains(err.Error(), "skill") {
		t.Errorf("expected error to mention skill, got: %v", err)
	}
}

func TestPluralizeTypeFilter(t *testing.T) {
	tests := map[string]string{
		"":          "",
		"language":  "languages",
		"framework": "frameworks",
		"workflow":  "workflows",
		"skill":     "skills",
	}
	for in, want := range tests {
		if got := pluralizeTypeFilter(in); got != want {
			t.Errorf("pluralizeTypeFilter(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRunLs_DetailWithoutQueryErrors(t *testing.T) {
	t.Setenv("SAMUEL_NO_DEPRECATION", "1") // doesn't matter, but keep stderr quiet
	cmd := lsCmd
	if err := cmd.Flags().Set("detail", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("detail", "false") })

	err := runLs(cmd, []string{})
	if err == nil {
		t.Fatal("expected error when --detail is given without a query")
	}
	if !strings.Contains(err.Error(), "--detail") {
		t.Errorf("expected error to mention --detail, got: %v", err)
	}
}

func TestRunLs_InvalidTypeErrors(t *testing.T) {
	cmd := lsCmd
	if err := cmd.Flags().Set("type", "nonsense"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("type", "") })

	err := runLs(cmd, []string{})
	if err == nil {
		t.Fatal("expected error for invalid --type value")
	}
	if !strings.Contains(err.Error(), "invalid --type") {
		t.Errorf("expected error to mention invalid --type, got: %v", err)
	}
}
