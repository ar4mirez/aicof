package skills

import (
	"io/fs"
	"strings"
	"testing"
)

// TestFS_TopLevelEntriesAreSkills walks the embedded fs.FS and confirms
// every top-level entry looks like a skill directory (one with at least
// one file inside). This guards against the embed directive silently
// dropping content — the failing case is a misconfigured directive that
// embeds nothing or only top-level metadata files.
func TestFS_TopLevelEntriesAreSkills(t *testing.T) {
	skillFS, err := FS()
	if err != nil {
		t.Fatalf("FS: %v", err)
	}
	entries, err := fs.ReadDir(skillFS, ".")
	if err != nil {
		t.Fatalf("ReadDir root: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("embed produced 0 entries — //go:embed directive is wrong")
	}
	// Every top-level entry must be a directory (no stray README.md at
	// the skills root).
	dirs := 0
	for _, e := range entries {
		if !e.IsDir() {
			t.Errorf("unexpected non-directory at root: %s", e.Name())
			continue
		}
		dirs++
	}
	if dirs < 50 {
		// Spike findings showed 81 skill directories. A drop below 50
		// means content was lost in the move.
		t.Errorf("expected ~80 skill directories; got %d — content likely lost", dirs)
	}
}

// TestFS_KnownSkillsPresent asserts a representative subset of skills is
// reachable. If any of these go missing, the parity test fails loudly.
func TestFS_KnownSkillsPresent(t *testing.T) {
	skillFS := MustFS()
	known := []string{
		"go-guide",
		"create-prd",
		"code-review",
		"webapp-testing",
		"security-audit",
		"troubleshooting",
	}
	for _, name := range known {
		info, err := fs.Stat(skillFS, name)
		if err != nil {
			t.Errorf("expected %s/ in embedded skills; got %v", name, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory; got mode %v", name, info.Mode())
		}
		// Skill directory should contain at least SKILL.md.
		skillMd := name + "/SKILL.md"
		if _, err := fs.Stat(skillFS, skillMd); err != nil {
			t.Errorf("expected %s; got %v", skillMd, err)
		}
	}
}

// TestFS_NestedAssetsEmbedded checks that non-markdown files (binary
// templates, examples) at deeper paths survived the embed. Spike findings
// recorded 12 non-md asset files across algorithmic-art, webapp-testing,
// mcp-builder, web-artifacts-builder.
func TestFS_NestedAssetsEmbedded(t *testing.T) {
	skillFS := MustFS()
	nonMD := 0
	files := 0
	err := fs.WalkDir(skillFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files++
		if !strings.HasSuffix(strings.ToLower(p), ".md") {
			nonMD++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
	if files < 100 {
		t.Errorf("expected >= 100 files in embedded skills; got %d", files)
	}
	if nonMD == 0 {
		// `//go:embed all:content` is required to pull non-md assets in;
		// without `all:`, embed skips files starting with _ or . and may
		// skip non-text. Zero non-md files signals a directive regression.
		t.Errorf("expected non-markdown files (templates, examples); got 0 — embed directive may be wrong")
	}
}

// TestMustFS_DoesNotPanic confirms the package-level helper works in
// the steady-state.
func TestMustFS_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustFS panicked unexpectedly: %v", r)
		}
	}()
	_ = MustFS()
}
