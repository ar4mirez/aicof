package orchestrator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// samuelGlobalDir is where Samuel publishes its skill content. Resolved
// relative to the user's home dir.
const samuelGlobalDir = ".claude/skills/samuel"

// samuelProjectSymlink is the symlink path inside a project that points
// to the global install. Resolved relative to the project root.
const samuelProjectSymlink = ".claude/skills/samuel"

// SamuelComponent syncs Samuel's curated skills (framework guides + workflow
// helpers) from an embedded fs.FS to ~/.claude/skills/samuel/ and creates a
// symlink at <project>/.claude/skills/samuel/ pointing at the global tree.
//
// In production the Source is internal/skills.FS() — an embed.FS built into
// the Samuel binary at compile time. Tests inject fstest.MapFS or
// os.DirFS for hermetic, fast assertions.
type SamuelComponent struct {
	// Source is the read-only filesystem of skill content. Each top-level
	// entry is a skill directory (go-guide/, nextjs/, ...).
	Source fs.FS
	// homeDir overrides the user's home for testing. Empty falls through
	// to os.UserHomeDir().
	homeDir string
	// projectDir is the project root where the symlink lives. Empty means
	// "skip project work entirely" (e.g., samuel doctor at user level).
	projectDir string
	// version is what Detect reports as the installed version. In practice
	// this is the Samuel binary version — skills are embedded, so the
	// binary IS the source of truth.
	version string
}

// NewSamuelComponent constructs a samuel-skills component.
//
// source: fs.FS rooted at the skills tree (top-level dirs = skill names)
// homeDir: empty for production; tests pass a temp dir
// projectDir: project root for the symlink, empty to skip project work
// version: stable identifier reported by Detect (Samuel binary version)
func NewSamuelComponent(source fs.FS, homeDir, projectDir, version string) *SamuelComponent {
	return &SamuelComponent{
		Source:     source,
		homeDir:    homeDir,
		projectDir: projectDir,
		version:    version,
	}
}

// Name returns the constant component identifier.
func (s *SamuelComponent) Name() string { return NameSamuelSkills }

func (s *SamuelComponent) home() (string, error) {
	if s.homeDir != "" {
		return s.homeDir, nil
	}
	return os.UserHomeDir()
}

// globalPath returns the absolute path to ~/.claude/skills/samuel.
func (s *SamuelComponent) globalPath() (string, error) {
	home, err := s.home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, samuelGlobalDir), nil
}

// projectSymlinkPath returns the absolute path to the in-project symlink.
// Empty string when projectDir is unset.
func (s *SamuelComponent) projectSymlinkPath() string {
	if s.projectDir == "" {
		return ""
	}
	return filepath.Join(s.projectDir, samuelProjectSymlink)
}

// Detect reports whether the global skill tree exists. We treat "directory
// present and non-empty" as installed; the version is what the binary
// reports (skills are embedded, so binary version IS skills version).
func (s *SamuelComponent) Detect(ctx context.Context) (DetectResult, error) {
	path, err := s.globalPath()
	if err != nil {
		return DetectResult{}, err
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return DetectResult{Installed: false, Path: path}, nil
	}
	// Empty directory shouldn't count as installed (would happen if a
	// previous run failed mid-sync and left an empty dir behind).
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) == 0 {
		return DetectResult{Installed: false, Path: path}, nil
	}
	return DetectResult{
		Installed: true,
		Version:   s.version,
		Path:      path,
	}, nil
}

// Install syncs Source → ~/.claude/skills/samuel/ and creates the project
// symlink. Idempotent: if the global tree's content hash matches Source's
// content hash and the symlink already points where we'd point it, no-op.
//
// Atomicity model: write to a sibling tmp dir (samuel.tmp-<rand>), then
// rename onto the target after sync completes. On failure, the tmp dir is
// removed without disturbing the live tree.
func (s *SamuelComponent) Install(ctx context.Context, opts InstallOptions) (InstallResult, error) {
	res := InstallResult{Component: NameSamuelSkills}
	if s.Source == nil {
		return res, &Error{
			Component: NameSamuelSkills,
			Problem:   "samuel-skills component has no Source fs.FS configured",
			Cause:     "constructor was called with nil source",
		}
	}

	target, err := s.globalPath()
	if err != nil {
		return res, (&Error{
			Component:   NameSamuelSkills,
			Problem:     "cannot resolve samuel-skills install path",
			Recoverable: true,
		}).Wrap(err)
	}

	// Compute desired content hash up front. If it matches what's on disk
	// AND the symlink (if requested) already points at target, no-op.
	desiredHash, err := hashFS(s.Source)
	if err != nil {
		return res, (&Error{
			Component:   NameSamuelSkills,
			Problem:     "cannot hash skill source",
			Recoverable: true,
		}).Wrap(err)
	}
	currentHash, _ := hashTree(target) // best-effort; missing tree → empty
	symlinkOK := s.symlinkPointsAtTarget(target)
	skipSymlink := opts.SkipSymlink || s.projectSymlinkPath() == ""

	if desiredHash == currentHash && (skipSymlink || symlinkOK) && !opts.Force {
		res.AlreadyInstalled = true
		return res, nil
	}

	if opts.DryRun {
		return res, nil
	}

	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return res, (&Error{
			Component:   NameSamuelSkills,
			Problem:     "cannot create parent directory for samuel skills",
			Path:        parent,
			Recoverable: true,
		}).Wrap(err)
	}

	// Stage in a sibling temp dir so the swap is atomic.
	tmp, err := os.MkdirTemp(parent, "samuel.tmp-")
	if err != nil {
		return res, (&Error{
			Component:   NameSamuelSkills,
			Problem:     "cannot create staging dir for samuel skills",
			Path:        parent,
			Recoverable: true,
		}).Wrap(err)
	}
	// On any error below, drop the staging dir.
	cleanupTmp := func() { _ = os.RemoveAll(tmp) }

	if err := syncFS(s.Source, tmp); err != nil {
		cleanupTmp()
		// syncFS already wraps in *Error when path traversal triggers.
		return res, err
	}

	// Swap: if a previous tree exists, rename it aside, then move tmp into
	// place. On failure, we restore.
	var backup string
	if _, statErr := os.Stat(target); statErr == nil {
		backup = target + ".bak-" + shortHash(desiredHash)
		if err := os.Rename(target, backup); err != nil {
			cleanupTmp()
			return res, (&Error{
				Component:   NameSamuelSkills,
				Problem:     "cannot move existing samuel skills out of the way",
				Path:        target,
				Recoverable: true,
			}).Wrap(err)
		}
	}
	if err := os.Rename(tmp, target); err != nil {
		// Restore the backup if we made one.
		if backup != "" {
			_ = os.Rename(backup, target)
		}
		cleanupTmp()
		return res, (&Error{
			Component:   NameSamuelSkills,
			Problem:     "cannot rename staged samuel skills into place",
			Path:        target,
			Recoverable: true,
		}).Wrap(err)
	}
	// Successful swap. Drop the backup.
	if backup != "" {
		_ = os.RemoveAll(backup)
	}
	res.Mutations = append(res.Mutations, Mutation{
		Kind:        MutationDirCreated,
		Path:        target,
		Description: "synced samuel skills to " + target,
		Reverse: func(ctx context.Context) error {
			return os.RemoveAll(target)
		},
	})

	if skipSymlink {
		return res, nil
	}

	if err := s.ensureSymlink(target); err != nil {
		return res, err
	}
	res.Mutations = append(res.Mutations, Mutation{
		Kind:        MutationSymlinkCreated,
		Path:        s.projectSymlinkPath(),
		Description: "linked project skills/samuel → " + target,
		Reverse: func(ctx context.Context) error {
			path := s.projectSymlinkPath()
			if path == "" {
				return nil
			}
			// Only remove if it's still a symlink pointing where we put it.
			info, err := os.Lstat(path)
			if err != nil {
				return nil
			}
			if info.Mode()&os.ModeSymlink == 0 {
				return nil
			}
			return os.Remove(path)
		},
	})

	return res, nil
}

// Check reports samuel-skills health: global tree present and current,
// project symlink present and pointing at the global tree (when projectDir
// is set).
func (s *SamuelComponent) Check(ctx context.Context) HealthStatus {
	detect, err := s.Detect(ctx)
	if err != nil {
		return HealthStatus{
			Component: NameSamuelSkills,
			OK:        false,
			Message:   "cannot detect samuel skills: " + err.Error(),
			FixHint:   "samuel init",
		}
	}
	if !detect.Installed {
		return HealthStatus{
			Component: NameSamuelSkills,
			OK:        false,
			Message:   "samuel skills not synced to " + detect.Path,
			FixHint:   "samuel init",
		}
	}

	if s.projectSymlinkPath() != "" {
		target, _ := s.globalPath()
		if !s.symlinkPointsAtTarget(target) {
			return HealthStatus{
				Component: NameSamuelSkills,
				OK:        false,
				Message:   "project symlink missing or pointing elsewhere: " + s.projectSymlinkPath(),
				FixHint:   "samuel init --force",
			}
		}
	}

	msg := "samuel skills synced"
	if detect.Version != "" {
		msg = "samuel skills " + detect.Version + " synced"
	}
	return HealthStatus{
		Component: NameSamuelSkills,
		OK:        true,
		Message:   msg,
	}
}

// Uninstall removes project symlinks (Project), the global tree (Global),
// or both (All).
func (s *SamuelComponent) Uninstall(ctx context.Context, opts UninstallOptions) (UninstallResult, error) {
	res := UninstallResult{Component: NameSamuelSkills}

	doProject := opts.Project || opts.All
	doGlobal := opts.Global || opts.All
	if !doProject && !doGlobal {
		res.Skipped = true
		return res, nil
	}

	if opts.DryRun {
		return res, nil
	}

	if doProject {
		path := s.projectSymlinkPath()
		if path != "" {
			if info, err := os.Lstat(path); err == nil {
				// Only remove if it's a symlink — never a real dir users
				// might have populated.
				if info.Mode()&os.ModeSymlink != 0 {
					if err := os.Remove(path); err != nil {
						return res, (&Error{
							Component:   NameSamuelSkills,
							Problem:     "cannot remove project samuel symlink",
							Path:        path,
							Recoverable: true,
						}).Wrap(err)
					}
					res.Mutations = append(res.Mutations, Mutation{
						Kind:        MutationSymlinkCreated,
						Path:        path,
						Description: "removed project samuel symlink",
					})
				}
			}
		}
	}

	if doGlobal {
		target, err := s.globalPath()
		if err == nil {
			if _, statErr := os.Stat(target); statErr == nil {
				if err := os.RemoveAll(target); err != nil {
					return res, (&Error{
						Component:   NameSamuelSkills,
						Problem:     "cannot remove global samuel skills",
						Path:        target,
						Recoverable: true,
					}).Wrap(err)
				}
				res.Mutations = append(res.Mutations, Mutation{
					Kind:        MutationDirCreated,
					Path:        target,
					Description: "removed global samuel skills",
				})
			}
		}
	}

	return res, nil
}

// ensureSymlink creates (or repairs) the project symlink to point at target.
// Conflict matrix:
//   - missing → create
//   - symlink to target → no-op
//   - symlink to elsewhere → remove + recreate
//   - regular file/dir at the path → return Recoverable error (don't clobber
//     user data without explicit guidance)
func (s *SamuelComponent) ensureSymlink(target string) error {
	path := s.projectSymlinkPath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return (&Error{
			Component:   NameSamuelSkills,
			Problem:     "cannot create parent directory for project symlink",
			Path:        filepath.Dir(path),
			Recoverable: true,
		}).Wrap(err)
	}
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			cur, _ := os.Readlink(path)
			if cur == target {
				return nil
			}
			// Wrong target — replace.
			if err := os.Remove(path); err != nil {
				return (&Error{
					Component:   NameSamuelSkills,
					Problem:     "cannot replace existing project symlink",
					Path:        path,
					Recoverable: true,
				}).Wrap(err)
			}
		} else {
			// Real file or directory at the symlink path. Don't clobber.
			return &Error{
				Component:   NameSamuelSkills,
				Problem:     "project skills/samuel exists and is not a symlink",
				Cause:       "Samuel won't overwrite a real file or directory",
				Fix:         "remove " + path + " manually, then re-run samuel init",
				Path:        path,
				Recoverable: true,
			}
		}
	}
	if err := os.Symlink(target, path); err != nil {
		return (&Error{
			Component:   NameSamuelSkills,
			Problem:     "cannot create project symlink",
			Path:        path,
			Recoverable: true,
		}).Wrap(err)
	}
	return nil
}

// symlinkPointsAtTarget reports whether the project symlink (if configured)
// already resolves to the expected target. Returns true when projectDir is
// unset (no symlink expected), so callers can short-circuit.
func (s *SamuelComponent) symlinkPointsAtTarget(target string) bool {
	path := s.projectSymlinkPath()
	if path == "" {
		return true
	}
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	cur, err := os.Readlink(path)
	if err != nil {
		return false
	}
	return cur == target
}

// syncFS copies every file from src into dst, preserving directory structure.
// Path traversal defense: every entry is validated with filepath.IsLocal
// before any filesystem write. A malicious or buggy fs.FS that yields paths
// like ../../etc/passwd is rejected with a *Error.
func syncFS(src fs.FS, dst string) error {
	return fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return os.MkdirAll(dst, 0o700)
		}
		// Defense-in-depth: refuse any non-local path. fs.FS produces
		// forward-slash paths regardless of OS, so IsLocal is the right
		// check here.
		if !filepath.IsLocal(p) {
			return &Error{
				Component: NameSamuelSkills,
				Problem:   "path traversal attempt in skill source",
				Cause:     "fs.FS yielded a non-local path: " + p,
				Path:      p,
			}
		}
		out := filepath.Join(dst, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(out, 0o700)
		}
		f, err := src.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		w, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
		if err != nil {
			return err
		}
		defer w.Close()
		if _, err := io.Copy(w, f); err != nil {
			return err
		}
		return nil
	})
}

// hashFS returns a deterministic content hash of src's tree. Used by
// Install's idempotency check to skip work when the live tree already
// matches the desired tree.
func hashFS(src fs.FS) (string, error) {
	h := sha256.New()
	err := fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return nil
		}
		if !filepath.IsLocal(p) {
			return fmt.Errorf("non-local path in skill source: %s", p)
		}
		fmt.Fprintf(h, "P:%s\n", p)
		if d.IsDir() {
			return nil
		}
		f, err := src.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// hashTree returns the same deterministic hash as hashFS but operates on a
// real directory tree on disk.
func hashTree(root string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		// Normalize to forward slashes so on-disk hashes match fs.FS hashes.
		fmt.Fprintf(h, "P:%s\n", filepath.ToSlash(rel))
		if d.IsDir() {
			return nil
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// shortHash returns the first 8 chars of a hex hash, suitable for naming
// transient backup directories.
func shortHash(full string) string {
	if len(full) < 8 {
		return full
	}
	return strings.ToLower(full[:8])
}
