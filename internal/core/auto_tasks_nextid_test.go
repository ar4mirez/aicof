package core

import "testing"

func TestNextAvailableID_Empty(t *testing.T) {
	prd := &AutoPRD{}
	if got := prd.NextAvailableID(); got != "1" {
		t.Errorf("empty prd: NextAvailableID() = %q, want %q", got, "1")
	}
}

func TestNextAvailableID_TopLevelOnly(t *testing.T) {
	prd := &AutoPRD{Tasks: []AutoTask{
		{ID: "1"}, {ID: "2"}, {ID: "3"},
	}}
	if got := prd.NextAvailableID(); got != "4" {
		t.Errorf("top-level: NextAvailableID() = %q, want %q", got, "4")
	}
}

func TestNextAvailableID_NestedIDs(t *testing.T) {
	// Nested IDs ("2.0", "2.1") shouldn't move the counter past their parent.
	prd := &AutoPRD{Tasks: []AutoTask{
		{ID: "1"}, {ID: "2.0"}, {ID: "2.1"}, {ID: "2.2"},
	}}
	if got := prd.NextAvailableID(); got != "3" {
		t.Errorf("nested: NextAvailableID() = %q, want %q (max top-level = 2)", got, "3")
	}
}

func TestNextAvailableID_MixedFormats(t *testing.T) {
	// Mixed integer and non-integer IDs. Non-integers don't move the counter.
	prd := &AutoPRD{Tasks: []AutoTask{
		{ID: "fix-auth"}, {ID: "1"}, {ID: "5"}, {ID: "investigate-flake"},
	}}
	if got := prd.NextAvailableID(); got != "6" {
		t.Errorf("mixed: NextAvailableID() = %q, want %q", got, "6")
	}
}

func TestNextAvailableID_AllNonInteger(t *testing.T) {
	// Edge case: all task IDs are strings. NextAvailableID returns 1 (max=0 stays).
	prd := &AutoPRD{Tasks: []AutoTask{
		{ID: "alpha"}, {ID: "beta"},
	}}
	if got := prd.NextAvailableID(); got != "1" {
		t.Errorf("non-integer: NextAvailableID() = %q, want %q", got, "1")
	}
}

func TestNextAvailableID_Sparse(t *testing.T) {
	// Sparse IDs (skipped numbers): NextAvailableID continues past the largest.
	prd := &AutoPRD{Tasks: []AutoTask{
		{ID: "1"}, {ID: "3"}, {ID: "7"},
	}}
	if got := prd.NextAvailableID(); got != "8" {
		t.Errorf("sparse: NextAvailableID() = %q, want %q (largest+1, no gap-filling)", got, "8")
	}
}
