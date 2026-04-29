package ui

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSONSchemaVersion identifies the v3 JSON envelope shape. Bumped from
// implicit-v2 (no field) to 3 in v3.0.0. Consumers that pin to a specific
// schema version can branch on this field.
//
// Schema 3 changes vs v2:
//   - SchemaVersion field added (always 3 in v3.x)
//   - The Command field reflects the *invoked* command path (e.g. "ls" or
//     "run done") rather than the handler's hardcoded label. Handlers with
//     multiple invocation paths (legacy aliases) emit the path the user
//     actually typed.
const JSONSchemaVersion = 3

// JSONResponse is a standard envelope for JSON output.
type JSONResponse struct {
	SchemaVersion int         `json:"schemaVersion"`
	Command       string      `json:"command"`
	Success       bool        `json:"success"`
	Data          interface{} `json:"data,omitempty"`
	Error         string      `json:"error,omitempty"`
}

// PrintJSON writes a successful JSON response to stdout.
func PrintJSON(command string, data interface{}) {
	resp := JSONResponse{
		SchemaVersion: JSONSchemaVersion,
		Command:       command,
		Success:       true,
		Data:          data,
	}
	writeJSON(resp)
}

// PrintJSONError writes an error JSON response to stderr.
func PrintJSONError(command string, err error) {
	resp := JSONResponse{
		SchemaVersion: JSONSchemaVersion,
		Command:       command,
		Success:       false,
		Error:         err.Error(),
	}
	out, _ := json.Marshal(resp)
	fmt.Fprintln(os.Stderr, string(out))
}

func writeJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
