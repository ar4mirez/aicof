package core

import (
	"sort"
	"strings"
	"testing"
)

// TestRegistry_NoCrossTypeNameCollisions enforces that no component name
// appears in more than one of {Languages, Frameworks, Workflows}. This is
// the invariant that lets `samuel add <name>`, `samuel rm <name>`, and
// `samuel ls <name> --detail` infer the type without ambiguity.
//
// Skills are deliberately NOT included in this check — the registry mirrors
// every Language/Framework/Workflow into Skills as <name>-guide entries,
// so cross-checking against Skills would always trip.
//
// If this test fails: a new component was added with a name that already
// exists in another category. Either rename it or leave the registry alone.
// Adding a colliding name silently breaks type inference for everyone.
func TestRegistry_NoCrossTypeNameCollisions(t *testing.T) {
	// Build a map: name -> categories where it appears.
	seen := make(map[string][]string)

	for _, c := range Languages {
		seen[c.Name] = append(seen[c.Name], "language")
	}
	for _, c := range Frameworks {
		seen[c.Name] = append(seen[c.Name], "framework")
	}
	for _, c := range Workflows {
		seen[c.Name] = append(seen[c.Name], "workflow")
	}

	var collisions []string
	for name, cats := range seen {
		if len(cats) > 1 {
			sort.Strings(cats)
			collisions = append(collisions, name+" (appears in: "+strings.Join(cats, ", ")+")")
		}
	}

	if len(collisions) > 0 {
		sort.Strings(collisions)
		t.Errorf("cross-type name collisions detected — type inference will break:\n  %s",
			strings.Join(collisions, "\n  "))
	}
}

func TestInferComponentType_Language(t *testing.T) {
	typ, candidates, err := InferComponentType("typescript")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if typ != "language" {
		t.Errorf("typescript: typ = %q, want %q", typ, "language")
	}
	if len(candidates) != 1 || candidates[0] != "language" {
		t.Errorf("typescript: candidates = %v, want [language]", candidates)
	}
}

func TestInferComponentType_Framework(t *testing.T) {
	typ, _, err := InferComponentType("react")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if typ != "framework" {
		t.Errorf("react: typ = %q, want %q", typ, "framework")
	}
}

func TestInferComponentType_Workflow(t *testing.T) {
	typ, _, err := InferComponentType("create-prd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if typ != "workflow" {
		t.Errorf("create-prd: typ = %q, want %q", typ, "workflow")
	}
}

func TestInferComponentType_NotFound(t *testing.T) {
	_, _, err := InferComponentType("this-component-does-not-exist")
	if err == nil {
		t.Fatal("expected error for nonexistent component")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to mention 'not found', got: %v", err)
	}
}

func TestInferComponentType_SkillsExcluded(t *testing.T) {
	// Every Framework gets mirrored into Skills (e.g., "react" exists in both
	// Frameworks and Skills). InferComponentType should NOT see Skills, so it
	// returns exactly one candidate (framework) for "react".
	_, candidates, err := InferComponentType("react")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 1 {
		t.Errorf("react: candidates = %v, want exactly 1 (Skills must be excluded)", candidates)
	}
}

func TestJoinTypesForError(t *testing.T) {
	tests := []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{"language"}, "language"},
		{[]string{"language", "framework"}, "language and framework"},
		{[]string{"language", "framework", "workflow"}, "language, framework, and workflow"},
	}
	for _, tt := range tests {
		got := joinTypesForError(tt.in)
		if got != tt.want {
			t.Errorf("joinTypesForError(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
