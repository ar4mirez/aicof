package commands

import (
	"fmt"

	"github.com/ar4mirez/samuel/internal/core"
)

// parseAddRemoveArgs accepts both v2 and v3 argument forms and returns the
// canonical (componentType, componentName) tuple.
//
// Forms:
//
//	samuel <verb> <name>                     v3 inference (1 arg)
//	samuel <verb> <name> <type>              v3 explicit  (2 args, name first)
//	samuel <verb> <type> <name>              v2 explicit  (2 args, type first)
//
// The 1-arg form uses core.InferComponentType which scans Languages,
// Frameworks, and Workflows (Skills are excluded — they mirror the others).
// The 2-arg form auto-detects which positional is the type by checking if
// either looks like a known type alias (language/lang/l, framework/fw/f,
// workflow/wf/w). Skill-related aliases are intentionally rejected because
// `samuel add` and `samuel rm` do not manage skills.
//
// `verb` is used purely for error message text ("add" vs "remove").
func parseAddRemoveArgs(verb string, args []string) (string, string, error) {
	switch len(args) {
	case 1:
		name := args[0]
		typ, candidates, err := core.InferComponentType(name)
		if err != nil {
			// "not found" or "ambiguous" — both useful messages from core.
			// Add a hint pointing the user at the explicit-type form.
			if len(candidates) > 1 {
				return "", "", fmt.Errorf("%w\n  Try: samuel %s %s <type>", err, verb, name)
			}
			return "", "", fmt.Errorf("%w\n  Run 'samuel ls --all' to list installable components", err)
		}
		return typ, name, nil

	case 2:
		// Detect arg order by which side parses as a known type alias.
		t0 := canonicalAddRemoveType(args[0])
		t1 := canonicalAddRemoveType(args[1])

		switch {
		case t0 != "" && t1 != "":
			// Both look like types. Highly unusual; treat first as type, second as name.
			return t0, args[1], nil
		case t0 != "":
			// v2 form: <type> <name>
			return t0, args[1], nil
		case t1 != "":
			// v3 form: <name> <type>
			return t1, args[0], nil
		default:
			return "", "", fmt.Errorf(
				"could not interpret 'samuel %s %s %s'. Expected 'samuel %s <name>' or 'samuel %s <name> <type>'.",
				verb, args[0], args[1], verb, verb,
			)
		}
	}
	// Cobra's RangeArgs(1,2) catches arg-count violations before this runs.
	return "", "", fmt.Errorf("expected 1 or 2 arguments, got %d", len(args))
}

// canonicalAddRemoveType returns the canonical type name ("language",
// "framework", "workflow") for a recognized alias, or "" otherwise. Skill
// aliases (skill, sk, s) are deliberately excluded — `samuel add` and
// `samuel rm` do not install/uninstall skills.
func canonicalAddRemoveType(s string) string {
	switch s {
	case "language", "lang", "l":
		return "language"
	case "framework", "fw", "f":
		return "framework"
	case "workflow", "wf", "w":
		return "workflow"
	}
	return ""
}
