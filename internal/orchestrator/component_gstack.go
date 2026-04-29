package orchestrator

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// gstackPinnedSHA is the git commit SHA that this Samuel release composes
// against. Bumping it is a deliberate Samuel release event, not a casual
// edit — gstack ships frequently and an unverified bump can break the
// install flow.
//
// Bump procedure:
//  1. `git ls-remote https://github.com/garrytan/gstack HEAD` for the new SHA
//  2. Run an integration test of `samuel init` against the new SHA in a
//     clean container
//  3. Update CHANGELOG with the gstack version delta
const gstackPinnedSHA = "e8893a18b18e32ebd63a21f6915337868249ebe1"

const (
	gstackRepoURL    = "https://github.com/garrytan/gstack"
	gstackInstallDir = ".claude/skills/gstack" // relative to home dir
)

// gstackExec is the exec function the gstack component shells out through.
// Tests override this to inject fake commands without invoking real git
// or running the gstack setup script.
var gstackExec = exec.CommandContext

// GstackComponent composes gstack into Samuel's curated bundle. gstack is
// not vendored — Samuel clones at gstackPinnedSHA and invokes gstack's
// official `./setup --team --quiet --host claude` so the upstream tool
// stays canonical.
type GstackComponent struct {
	homeDir string // overridable for tests; empty falls through to os.UserHomeDir
}

// NewGstackComponent constructs a gstack component. Pass an empty string
// for production use (resolves home from the environment); tests pass a
// temp dir.
func NewGstackComponent(homeDir string) *GstackComponent {
	return &GstackComponent{homeDir: homeDir}
}

// Name returns the constant component identifier.
func (g *GstackComponent) Name() string { return NameGstack }

func (g *GstackComponent) home() (string, error) {
	if g.homeDir != "" {
		return g.homeDir, nil
	}
	return os.UserHomeDir()
}

func (g *GstackComponent) installPath() (string, error) {
	home, err := g.home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, gstackInstallDir), nil
}

// Detect reads the gstack install path. If `.git` is present, returns the
// short SHA from `git rev-parse --short HEAD`. If absent or non-readable,
// returns Installed=false. Never mutates state.
func (g *GstackComponent) Detect(ctx context.Context) (DetectResult, error) {
	path, err := g.installPath()
	if err != nil {
		return DetectResult{}, err
	}
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil || !info.IsDir() {
		return DetectResult{Installed: false, Path: path}, nil
	}
	out, err := gstackExec(ctx, "git", "-C", path, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		// gstack tree exists but git failed (no git binary, corrupt
		// .git, etc.). Report installed but unknown version so Check
		// can flag it for repair rather than re-cloning blindly.
		return DetectResult{Installed: true, Path: path}, nil
	}
	return DetectResult{
		Installed: true,
		Version:   strings.TrimSpace(string(out)),
		Path:      path,
	}, nil
}

// Install clones gstack at gstackPinnedSHA and runs its setup script.
// Idempotent: if Detect reports the correct pinned SHA already, returns
// AlreadyInstalled=true without mutating. Force re-installs even when
// Detect says current. Never overwrites a different gstack install
// without Force — surface a clear error so the user can decide.
func (g *GstackComponent) Install(ctx context.Context, opts InstallOptions) (InstallResult, error) {
	res := InstallResult{Component: NameGstack}
	if opts.SkipGstack {
		res.Skipped = true
		return res, nil
	}

	path, err := g.installPath()
	if err != nil {
		return res, (&Error{
			Component:   NameGstack,
			Problem:     "cannot resolve gstack install path",
			Recoverable: true,
		}).Wrap(err)
	}

	detect, _ := g.Detect(ctx)
	if detect.Installed && !opts.Force {
		if matchesShortSHA(detect.Version, gstackPinnedSHA) {
			res.AlreadyInstalled = true
			return res, nil
		}
		return res, &Error{
			Component:   NameGstack,
			Problem:     fmt.Sprintf("gstack present at %q with SHA %q (expected %s)", path, detect.Version, gstackPinnedSHA[:7]),
			Fix:         "rerun with --force to replace the existing install",
			DocsURL:     "https://samuel.dev/docs/errors/SAM-GSTACK-001",
			Recoverable: true,
			Path:        path,
		}
	}

	if _, err := exec.LookPath("git"); err != nil {
		return res, &Error{
			Component:   NameGstack,
			Problem:     "git not found in PATH",
			Cause:       "gstack install requires git to clone the upstream repository",
			Fix:         "install git (macOS: xcode-select --install; Debian/Ubuntu: apt install git)",
			Recoverable: true,
		}
	}

	if opts.DryRun {
		return res, nil
	}

	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return res, (&Error{
			Component:   NameGstack,
			Problem:     "cannot create gstack parent directory",
			Path:        parent,
			Recoverable: true,
		}).Wrap(err)
	}

	// Force path: previous install at gstackPinnedSHA mismatch was
	// detected and the caller passed --force. Remove first.
	if detect.Installed && opts.Force {
		if err := os.RemoveAll(path); err != nil {
			return res, (&Error{
				Component:   NameGstack,
				Problem:     "cannot remove existing gstack install for forced reinstall",
				Path:        path,
				Recoverable: true,
			}).Wrap(err)
		}
	}

	cloneOut, err := gstackExec(ctx, "git", "clone", "--quiet", gstackRepoURL, path).CombinedOutput()
	if err != nil {
		return res, (&Error{
			Component:   NameGstack,
			Problem:     "gstack clone failed",
			Cause:       strings.TrimSpace(string(cloneOut)),
			Fix:         "check network connectivity and access to " + gstackRepoURL,
			Recoverable: true,
		}).Wrap(err)
	}
	res.Mutations = append(res.Mutations, Mutation{
		Kind:        MutationGitClone,
		Path:        path,
		Description: "cloned gstack from " + gstackRepoURL,
		Reverse: func(ctx context.Context) error {
			return os.RemoveAll(path)
		},
	})

	checkoutOut, err := gstackExec(ctx, "git", "-C", path, "checkout", "--quiet", gstackPinnedSHA).CombinedOutput()
	if err != nil {
		return res, (&Error{
			Component:   NameGstack,
			Problem:     "gstack checkout failed at pinned SHA",
			Cause:       strings.TrimSpace(string(checkoutOut)),
			Path:        path,
			Recoverable: true,
		}).Wrap(err)
	}

	setupCmd := gstackExec(ctx, filepath.Join(path, "setup"), "--team", "--quiet", "--host", "claude")
	setupCmd.Dir = path
	if opts.Verbose && opts.Stdout != nil {
		setupCmd.Stdout = opts.Stdout
		setupCmd.Stderr = opts.Stdout
	}
	setupOut, err := setupCmd.CombinedOutput()
	if err != nil {
		return res, (&Error{
			Component:   NameGstack,
			Problem:     "gstack setup failed",
			Cause:       strings.TrimSpace(string(setupOut)),
			Path:        filepath.Join(path, "setup"),
			Recoverable: true,
		}).Wrap(err)
	}
	res.Mutations = append(res.Mutations, Mutation{
		Kind:        MutationCommandRun,
		Path:        filepath.Join(path, "setup"),
		Description: "ran gstack setup --team --quiet --host claude",
		// Reverse is nil: the clone Reverse above removes the entire
		// tree, which subsumes whatever state setup wrote inside it.
		Reverse: nil,
	})

	return res, nil
}

// Check reports gstack health for `samuel doctor`. Read-only — never
// mutates. OK only when installed AND at gstackPinnedSHA.
func (g *GstackComponent) Check(ctx context.Context) HealthStatus {
	detect, err := g.Detect(ctx)
	if err != nil {
		return HealthStatus{
			Component: NameGstack,
			OK:        false,
			Message:   "cannot detect gstack: " + err.Error(),
			FixHint:   "samuel init",
		}
	}
	if !detect.Installed {
		return HealthStatus{
			Component: NameGstack,
			OK:        false,
			Message:   "gstack not installed at " + detect.Path,
			FixHint:   "samuel init",
		}
	}
	if detect.Version == "" {
		return HealthStatus{
			Component: NameGstack,
			OK:        false,
			Message:   "gstack present but version unreadable (git missing or .git corrupt)",
			FixHint:   "samuel init --force",
		}
	}
	if !matchesShortSHA(detect.Version, gstackPinnedSHA) {
		return HealthStatus{
			Component: NameGstack,
			OK:        false,
			Message:   fmt.Sprintf("gstack at %s, expected %s", detect.Version, gstackPinnedSHA[:7]),
			FixHint:   "samuel init --force",
		}
	}
	return HealthStatus{
		Component: NameGstack,
		OK:        true,
		Message:   "gstack at " + detect.Version,
	}
}

// Uninstall is intentionally a no-op (with diagnostic message). gstack
// is composed — the user owns the install and may share it with other
// tools (Codex skill adapters, Cursor integrations, etc.). Removing it
// from a Samuel uninstall would silently break unrelated workflows.
// Users who explicitly want gstack gone can `rm -rf ~/.claude/skills/gstack`.
func (g *GstackComponent) Uninstall(_ context.Context, opts UninstallOptions) (UninstallResult, error) {
	res := UninstallResult{Component: NameGstack, Skipped: true}
	if (opts.Global || opts.All) && opts.Stdout != nil {
		_, _ = io.WriteString(opts.Stdout, "gstack kept in place (composed, user-owned). Remove manually if desired:\n  rm -rf ~/.claude/skills/gstack\n")
	}
	return res, nil
}

// matchesShortSHA reports whether short is a case-insensitive prefix of
// full. gstack's `git rev-parse --short HEAD` defaults to 7 characters;
// the pinned constant is the full 40-char SHA. Compare prefix-only.
func matchesShortSHA(short, full string) bool {
	if short == "" || full == "" || len(short) > len(full) {
		return false
	}
	return strings.EqualFold(short, full[:len(short)])
}
