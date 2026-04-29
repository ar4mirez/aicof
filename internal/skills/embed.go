// Package skills owns Samuel's curated skill content. The content tree
// (one directory per skill) is embedded into the Samuel binary at build
// time via go:embed, so a single binary download ships every skill.
//
// This is the v4 replacement for v3's "download tarball + extract template/"
// flow. The orchestrator's samuel-skills component reads from this fs.FS
// when populating the global ~/.claude/skills/samuel/ tree.
//
// See the v4 plan in docs/v4-roadmap.md for the migration story.
package skills

import (
	"embed"
	"io/fs"
)

// content embeds every file under content/ at build time. The directive
// uses `all:content` (no trailing glob) so directories nested at any depth
// are walked — `content/*` would only match direct children.
//
//go:embed all:content
var content embed.FS

// FS returns a read-only filesystem rooted at the skill content tree.
// Top-level entries are skill directories (go-guide/, nextjs/, ...).
// The returned fs.FS is safe to use concurrently — embed.FS is immutable.
func FS() (fs.FS, error) {
	// fs.Sub strips the "content" prefix so callers see "go-guide/..."
	// instead of "content/go-guide/..." — matches the expected layout of
	// ~/.claude/skills/samuel/.
	return fs.Sub(content, "content")
}

// MustFS is like FS but panics on error. The error path is unreachable in
// practice because the embedded directory is fixed at build time.
func MustFS() fs.FS {
	sub, err := FS()
	if err != nil {
		panic("skills: embedded content tree is malformed: " + err.Error())
	}
	return sub
}
