package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// deprecationSuppressed returns true when the user has opted out of legacy-command
// deprecation warnings. CI scripts can set SAMUEL_NO_DEPRECATION=1 or pass
// --no-deprecation to silence the redirect noise during the v3.0.x shim window.
func deprecationSuppressed(cmd *cobra.Command) bool {
	if os.Getenv("SAMUEL_NO_DEPRECATION") == "1" {
		return true
	}
	if cmd != nil {
		if f := cmd.Root().PersistentFlags().Lookup("no-deprecation"); f != nil {
			return f.Value.String() == "true"
		}
	}
	return false
}

// redirectAndRun wraps a RunE handler so that the legacy command path prints a
// one-line deprecation notice and then dispatches to the new handler. The notice
// goes to stderr so JSON consumers reading stdout are unaffected.
func redirectAndRun(newPath string, handler func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if !deprecationSuppressed(cmd) {
			oldPath := cmd.CommandPath()
			fmt.Fprintf(os.Stderr, "samuel: %q was renamed in v3.0.0. Use: %s\n", oldPath, newPath)
			fmt.Fprintln(os.Stderr, "        Set SAMUEL_NO_DEPRECATION=1 (or --no-deprecation) to silence. Removes in v3.1.0.")
		}
		return handler(cmd, args)
	}
}
