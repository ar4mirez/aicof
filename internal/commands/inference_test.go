package commands

import (
	"strings"
	"testing"
)

func TestParseAddRemoveArgs_OneArg_Inferred(t *testing.T) {
	tests := []struct {
		name    string
		wantTyp string
	}{
		{"react", "framework"},
		{"typescript", "language"},
		{"create-prd", "workflow"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ, name, err := parseAddRemoveArgs("add", []string{tt.name})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if typ != tt.wantTyp {
				t.Errorf("typ = %q, want %q", typ, tt.wantTyp)
			}
			if name != tt.name {
				t.Errorf("name = %q, want %q", name, tt.name)
			}
		})
	}
}

func TestParseAddRemoveArgs_OneArg_NotFound(t *testing.T) {
	_, _, err := parseAddRemoveArgs("add", []string{"definitely-not-a-real-component"})
	if err == nil {
		t.Fatal("expected error for nonexistent component")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to mention 'not found', got: %v", err)
	}
	if !strings.Contains(err.Error(), "samuel ls --all") {
		t.Errorf("expected error to suggest 'samuel ls --all', got: %v", err)
	}
}

func TestParseAddRemoveArgs_TwoArgs_V3Order(t *testing.T) {
	// v3 form: name first, type second.
	tests := []struct {
		args    []string
		wantTyp string
	}{
		{[]string{"react", "framework"}, "framework"},
		{[]string{"react", "fw"}, "framework"},
		{[]string{"react", "f"}, "framework"},
		{[]string{"typescript", "language"}, "language"},
		{[]string{"typescript", "lang"}, "language"},
		{[]string{"typescript", "l"}, "language"},
		{[]string{"create-prd", "workflow"}, "workflow"},
		{[]string{"create-prd", "wf"}, "workflow"},
		{[]string{"create-prd", "w"}, "workflow"},
	}
	for _, tt := range tests {
		t.Run(tt.args[0]+"_"+tt.args[1], func(t *testing.T) {
			typ, name, err := parseAddRemoveArgs("add", tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if typ != tt.wantTyp {
				t.Errorf("typ = %q, want %q", typ, tt.wantTyp)
			}
			if name != tt.args[0] {
				t.Errorf("name = %q, want %q", name, tt.args[0])
			}
		})
	}
}

func TestParseAddRemoveArgs_TwoArgs_V2Order(t *testing.T) {
	// v2 form: type first, name second. Must continue to work indefinitely.
	tests := []struct {
		args    []string
		wantTyp string
	}{
		{[]string{"framework", "react"}, "framework"},
		{[]string{"fw", "react"}, "framework"},
		{[]string{"language", "typescript"}, "language"},
		{[]string{"lang", "typescript"}, "language"},
		{[]string{"workflow", "create-prd"}, "workflow"},
		{[]string{"wf", "create-prd"}, "workflow"},
	}
	for _, tt := range tests {
		t.Run(tt.args[0]+"_"+tt.args[1], func(t *testing.T) {
			typ, name, err := parseAddRemoveArgs("add", tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if typ != tt.wantTyp {
				t.Errorf("typ = %q, want %q", typ, tt.wantTyp)
			}
			if name != tt.args[1] {
				t.Errorf("name = %q, want %q", name, tt.args[1])
			}
		})
	}
}

func TestParseAddRemoveArgs_TwoArgs_BothUnknown(t *testing.T) {
	// Neither arg is a known type alias — parser cannot resolve.
	_, _, err := parseAddRemoveArgs("add", []string{"foo", "bar"})
	if err == nil {
		t.Fatal("expected error when neither arg is a recognized type")
	}
	if !strings.Contains(err.Error(), "could not interpret") {
		t.Errorf("expected error to mention 'could not interpret', got: %v", err)
	}
}

func TestParseAddRemoveArgs_SkillAliasRejected(t *testing.T) {
	// 'samuel add' does not manage skills; the skill alias must NOT be recognized
	// as a valid type in this context.
	_, _, err := parseAddRemoveArgs("add", []string{"react", "skill"})
	if err == nil {
		t.Fatal("expected error when 'skill' is given as a type to 'add'")
	}
	// "react" is a known framework, "skill" is rejected as a type → parser
	// can't decide; falls into "could not interpret".
	if !strings.Contains(err.Error(), "could not interpret") {
		t.Errorf("expected 'could not interpret' error, got: %v", err)
	}
}

func TestCanonicalAddRemoveType(t *testing.T) {
	tests := map[string]string{
		"language":  "language",
		"lang":      "language",
		"l":         "language",
		"framework": "framework",
		"fw":        "framework",
		"f":         "framework",
		"workflow":  "workflow",
		"wf":        "workflow",
		"w":         "workflow",
		"skill":     "", // explicitly excluded
		"sk":        "",
		"s":         "",
		"":          "",
		"random":    "",
	}
	for in, want := range tests {
		if got := canonicalAddRemoveType(in); got != want {
			t.Errorf("canonicalAddRemoveType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRemoveCmd_RmAlias(t *testing.T) {
	hasRm := false
	for _, alias := range removeCmd.Aliases {
		if alias == "rm" {
			hasRm = true
			break
		}
	}
	if !hasRm {
		t.Errorf("expected removeCmd to have 'rm' alias, got: %v", removeCmd.Aliases)
	}
}

func TestAddCmd_AcceptsOneOrTwoArgs(t *testing.T) {
	// cobra.RangeArgs(1,2): 0 args → error, 1 or 2 args → OK, 3+ → error.
	if err := addCmd.Args(addCmd, []string{}); err == nil {
		t.Error("expected addCmd.Args to reject zero args")
	}
	if err := addCmd.Args(addCmd, []string{"react"}); err != nil {
		t.Errorf("addCmd.Args rejected 1 arg: %v", err)
	}
	if err := addCmd.Args(addCmd, []string{"react", "framework"}); err != nil {
		t.Errorf("addCmd.Args rejected 2 args: %v", err)
	}
	if err := addCmd.Args(addCmd, []string{"a", "b", "c"}); err == nil {
		t.Error("expected addCmd.Args to reject 3 args")
	}
}
