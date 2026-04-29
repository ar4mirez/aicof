// Package orchestrator coordinates installation, detection, health checking,
// and uninstallation of the components that make up Samuel's curated bundle.
//
// A Component is a discrete unit Samuel manages: gstack (composed via clone
// + setup --team --quiet), gbrain (registered via "claude mcp add"), or
// samuel-skills (synced from embedded content to ~/.claude/skills/samuel/).
//
// Components MUST be safe to call repeatedly. Both Install and Uninstall
// are required to be idempotent: installing an already-installed component
// at the correct version is a no-op (tracked via InstallResult.AlreadyInstalled).
package orchestrator

import (
	"context"
	"io"
)

// Component name constants. Use these instead of string literals when
// constructing components or routing options — typos in the literal
// cause silent skip-flag misroutes (verified by reviewer feedback).
const (
	NameGstack       = "gstack"
	NameGbrain       = "gbrain"
	NameSamuelSkills = "samuel-skills"
)

// Component is the contract every orchestrated piece implements.
type Component interface {
	// Name returns the stable identifier used in logs, errors, and the
	// component manifest. Examples: "gstack", "gbrain", "samuel-skills".
	Name() string

	// Detect inspects the system and reports whether the component is
	// installed, what version is present, and where it lives. It MUST NOT
	// mutate state.
	Detect(ctx context.Context) (DetectResult, error)

	// Install brings the component to the desired state. It MUST be
	// idempotent. On failure, the component is responsible for staging
	// changes atomically so callers can roll back via the returned
	// InstallResult.Mutations.
	Install(ctx context.Context, opts InstallOptions) (InstallResult, error)

	// Check reports the current health for samuel doctor. It MUST NOT
	// mutate state. A component installed at the wrong version, with a
	// broken symlink, or otherwise drifted should return OK=false with a
	// clear FixHint.
	Check(ctx context.Context) HealthStatus

	// Uninstall reverses Install. Like Install, it is idempotent —
	// uninstalling an absent component is a no-op. Components MAY refuse
	// to uninstall shared resources owned by the user (e.g., gstack's
	// component is a no-op or warning since the user owns the gstack
	// install).
	Uninstall(ctx context.Context, opts UninstallOptions) (UninstallResult, error)
}

// DetectResult captures what the system currently looks like for a component.
type DetectResult struct {
	// Installed is true when the component is present and reachable.
	Installed bool
	// Version is the component-specific version string. Per-component
	// semantics: gstack returns the short git SHA of the clone's HEAD,
	// gbrain returns the result of "gbrain --version", samuel-skills
	// returns the Samuel binary's own version since skills are embedded.
	// Empty string when not installed.
	Version string
	// Path is where the component lives on disk. Empty when not installed.
	Path string
}

// InstallOptions configures how Install runs.
type InstallOptions struct {
	// DryRun reports what would change without mutating state.
	DryRun bool
	// Force reinstalls even when Detect says the component is current.
	Force bool
	// Verbose enables progress output to Stdout.
	Verbose bool
	// Stdout receives progress messages. Components SHOULD respect Verbose.
	Stdout io.Writer
	// SkipGstack tells the orchestrator to omit the gstack component.
	// Component implementations ignore options that don't apply to them.
	SkipGstack bool
	// SkipGbrain tells the orchestrator to omit the gbrain component.
	SkipGbrain bool
	// GbrainBinary overrides the default gbrain path discovery.
	GbrainBinary string
	// SkipSymlink tells the samuel-skills component to omit the project
	// symlink (the user wants to manage .claude/skills/samuel/ themselves).
	SkipSymlink bool
}

// UninstallOptions configures how Uninstall runs.
type UninstallOptions struct {
	DryRun  bool
	Verbose bool
	Stdout  io.Writer
	// Project removes only project-local artifacts (symlinks, CLAUDE.md
	// routing block).
	Project bool
	// Global removes only ~/.claude/skills/samuel + MCP registration.
	Global bool
	// All removes both. Mutually exclusive with Project/Global; the
	// orchestrator validates this at the CLI boundary.
	All bool
}

// InstallResult records what Install actually changed. The orchestrator
// uses Mutations to roll back partial-failure scenarios in reverse order.
type InstallResult struct {
	// Component is the name of the component that produced this result.
	// Convenience field copied from Component.Name().
	Component string
	// Mutations describes every state change the component made, in the
	// order they occurred. The orchestrator runs Reverse on each in LIFO
	// order on rollback.
	Mutations []Mutation
	// AlreadyInstalled is true if Detect found the component current and
	// Install was a no-op (no Mutations).
	AlreadyInstalled bool
	// Skipped is true when InstallOptions told the orchestrator to omit
	// this component (e.g., SkipGbrain).
	Skipped bool
}

// UninstallResult mirrors InstallResult for the reverse direction.
type UninstallResult struct {
	Component string
	Mutations []Mutation
	// Skipped is true when the component declined to uninstall (e.g., the
	// gstack component refusing to remove a shared resource the user owns).
	Skipped bool
}

// Mutation describes one state change. Components emit these in
// chronological order; the orchestrator rolls back in reverse.
type Mutation struct {
	// Kind classifies the change for logging and rollback dispatch.
	Kind MutationKind
	// Path is the resource affected (file, symlink, directory).
	Path string
	// Description is a human-readable note used in samuel doctor output
	// and dry-run rendering.
	Description string
	// Reverse is the function that undoes this mutation. Required for
	// rollback. MUST be safe to call multiple times — components reuse
	// the same Reverse closure across nominally-idempotent paths (e.g.,
	// removing a file that was already removed is fine).
	Reverse func(context.Context) error
}

// MutationKind classifies state changes for telemetry and rollback handling.
type MutationKind string

const (
	// MutationFileWritten covers atomic writes (write tmp + rename).
	MutationFileWritten MutationKind = "file_written"
	// MutationSymlinkCreated covers symlink creation.
	MutationSymlinkCreated MutationKind = "symlink_created"
	// MutationDirCreated covers directory creation.
	MutationDirCreated MutationKind = "dir_created"
	// MutationCommandRun covers shell-out side effects (claude mcp add,
	// gstack setup --team --quiet).
	MutationCommandRun MutationKind = "command_run"
	// MutationGitClone covers cloning a remote repository.
	MutationGitClone MutationKind = "git_clone"
)

// HealthStatus is what Check returns. The orchestrator rolls all
// HealthStatuses up into samuel doctor output.
type HealthStatus struct {
	// Component is copied from Component.Name() for rendering convenience.
	Component string
	// OK is true when the component is healthy.
	OK bool
	// Message is human-readable status text shown in doctor output.
	// Required for both OK=true and OK=false.
	Message string
	// FixHint is an optional command the user can run to repair an
	// unhealthy component. Only populated when OK=false.
	FixHint string
}
