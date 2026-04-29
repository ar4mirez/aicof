package commands

import (
	"strings"

	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

// commandPath returns the invoked command path as it should appear in JSON
// output's `command` field. cobra.Command.CommandPath() includes the binary
// name ("samuel"); we strip that so JSON consumers see "ls", "run done",
// "admin config", etc.
//
// Handlers with multiple invocation paths (legacy aliases that forward to
// the same handler) call PrintJSONForCmd instead of ui.PrintJSON so the
// envelope reflects what the user actually typed:
//
//	samuel ls --json                  -> command="ls"
//	samuel list --json (legacy)       -> command="list"
//	samuel run done X --json          -> command="run done"
//	samuel auto task complete X --json -> command="auto task complete"
//
// When cmd is nil, returns "" — consistent with the legacy PrintJSON path
// which expects the caller to provide a string.
func commandPath(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	return strings.TrimPrefix(cmd.CommandPath(), "samuel ")
}

// PrintJSONForCmd writes a successful JSON response with the command field
// derived from cmd.CommandPath(). v3-preferred replacement for ui.PrintJSON
// in handlers that have more than one invocation path.
func PrintJSONForCmd(cmd *cobra.Command, data interface{}) {
	ui.PrintJSON(commandPath(cmd), data)
}

// PrintJSONErrorForCmd is the error-path companion to PrintJSONForCmd.
func PrintJSONErrorForCmd(cmd *cobra.Command, err error) {
	ui.PrintJSONError(commandPath(cmd), err)
}
