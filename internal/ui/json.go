package ui

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSONResponse is a standard envelope for JSON output.
type JSONResponse struct {
	Command string      `json:"command"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PrintJSON writes a successful JSON response to stdout.
func PrintJSON(command string, data interface{}) {
	resp := JSONResponse{
		Command: command,
		Success: true,
		Data:    data,
	}
	writeJSON(resp)
}

// PrintJSONError writes an error JSON response to stderr.
func PrintJSONError(command string, err error) {
	resp := JSONResponse{
		Command: command,
		Success: false,
		Error:   err.Error(),
	}
	out, _ := json.Marshal(resp)
	fmt.Fprintln(os.Stderr, string(out))
}

func writeJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
