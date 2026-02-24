package core

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Snapshot file constants
const (
	AutoSnapshotFile    = "project-snapshot.md"
	snapshotMaxFiles    = 200
	snapshotMaxGitLog   = 10
	snapshotMaxTODOList = 30
)

// FileEntry holds metadata for a single file in the project snapshot.
type FileEntry struct {
	Path string
	Size int64
}

// GenerateProjectSnapshot creates a compact project overview at
// .claude/auto/project-snapshot.md so the AI agent doesn't need to
// manually scan the directory tree, grep for TODOs, or check test gaps.
func GenerateProjectSnapshot(projectDir string) error {
	snapshotPath := filepath.Join(projectDir, AutoDir, AutoSnapshotFile)

	files, err := collectProjectFiles(projectDir)
	if err != nil {
		return fmt.Errorf("failed to collect project files: %w", err)
	}

	testGaps := findTestGaps(files)
	largeFiles := findLargeFiles(files)
	todoCounts := countTODOMarkers(projectDir, files)
	gitLog := recentGitLog(projectDir)

	return writeSnapshotFile(snapshotPath, files, testGaps, largeFiles, todoCounts, gitLog)
}

// collectProjectFiles walks the project and returns Go source files,
// skipping hidden dirs, vendor, bin, and other non-source directories.
func collectProjectFiles(projectDir string) ([]FileEntry, error) {
	var files []FileEntry
	err := filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		name := d.Name()
		if d.IsDir() {
			if shouldSkipDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSourceFile(name) {
			return nil
		}
		rel, relErr := filepath.Rel(projectDir, path)
		if relErr != nil {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		files = append(files, FileEntry{Path: rel, Size: info.Size()})
		if len(files) >= snapshotMaxFiles {
			return filepath.SkipAll
		}
		return nil
	})
	return files, err
}

func shouldSkipDir(name string) bool {
	skip := []string{".git", ".claude", "vendor", "node_modules", "bin", "site", "docs", "template"}
	for _, s := range skip {
		if name == s {
			return true
		}
	}
	return strings.HasPrefix(name, ".")
}

func isSourceFile(name string) bool {
	exts := []string{".go", ".py", ".js", ".ts", ".rs", ".java", ".rb"}
	for _, ext := range exts {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// findTestGaps returns Go source files that have no corresponding _test.go file.
func findTestGaps(files []FileEntry) []string {
	testFiles := make(map[string]bool)
	for _, f := range files {
		if strings.HasSuffix(f.Path, "_test.go") {
			testFiles[f.Path] = true
		}
	}

	var gaps []string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		if !strings.HasSuffix(f.Path, ".go") {
			continue
		}
		testPath := strings.TrimSuffix(f.Path, ".go") + "_test.go"
		if !testFiles[testPath] {
			gaps = append(gaps, f.Path)
		}
	}
	return gaps
}

// findLargeFiles returns files sorted by size descending, limited to top 15.
func findLargeFiles(files []FileEntry) []FileEntry {
	sorted := make([]FileEntry, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Size > sorted[j].Size
	})
	limit := 15
	if len(sorted) < limit {
		limit = len(sorted)
	}
	return sorted[:limit]
}

// countTODOMarkers counts TODO/FIXME/HACK occurrences per file
// using simple line scanning (no external tools needed).
func countTODOMarkers(projectDir string, files []FileEntry) map[string]int {
	counts := make(map[string]int)
	markers := []string{"TODO", "FIXME", "HACK"}

	for _, f := range files {
		if strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		count := countMarkersInFile(filepath.Join(projectDir, f.Path), markers)
		if count > 0 {
			counts[f.Path] = count
		}
	}
	return counts
}

func countMarkersInFile(path string, markers []string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		for _, m := range markers {
			if strings.Contains(line, m) {
				count++
				break // count each line at most once
			}
		}
	}
	return count
}

// recentGitLog returns the last N one-line git log entries.
func recentGitLog(projectDir string) []string {
	cmd := exec.Command("git", "log", "--oneline",
		fmt.Sprintf("-n%d", snapshotMaxGitLog))
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

func writeSnapshotFile(
	path string,
	files []FileEntry,
	testGaps []string,
	largeFiles []FileEntry,
	todoCounts map[string]int,
	gitLog []string,
) error {
	var sb strings.Builder
	sb.WriteString("# Project Snapshot (auto-generated — do not edit)\n\n")

	writeDirectorySection(&sb, files)
	writeTestGapSection(&sb, testGaps)
	writeLargeFilesSection(&sb, largeFiles)
	writeTODOSection(&sb, todoCounts)
	writeGitLogSection(&sb, gitLog)

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func writeDirectorySection(sb *strings.Builder, files []FileEntry) {
	sb.WriteString("## Source Files\n\n")
	fmt.Fprintf(sb, "Total: %d source files\n\n", len(files))
	for _, f := range files {
		fmt.Fprintf(sb, "- %s (%d bytes)\n", f.Path, f.Size)
	}
	sb.WriteString("\n")
}

func writeTestGapSection(sb *strings.Builder, gaps []string) {
	sb.WriteString("## Test Gaps (no corresponding _test.go)\n\n")
	if len(gaps) == 0 {
		sb.WriteString("All source files have test files.\n\n")
		return
	}
	for _, g := range gaps {
		fmt.Fprintf(sb, "- %s\n", g)
	}
	sb.WriteString("\n")
}

func writeLargeFilesSection(sb *strings.Builder, files []FileEntry) {
	sb.WriteString("## Largest Files (potential refactor candidates)\n\n")
	for _, f := range files {
		fmt.Fprintf(sb, "- %s (%d bytes)\n", f.Path, f.Size)
	}
	sb.WriteString("\n")
}

func writeTODOSection(sb *strings.Builder, counts map[string]int) {
	sb.WriteString("## TODO/FIXME/HACK Counts\n\n")
	if len(counts) == 0 {
		sb.WriteString("No TODO/FIXME/HACK markers found.\n\n")
		return
	}

	// Sort by count descending for priority
	type entry struct {
		path  string
		count int
	}
	var entries []entry
	for p, c := range counts {
		entries = append(entries, entry{p, c})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	limit := snapshotMaxTODOList
	if len(entries) < limit {
		limit = len(entries)
	}
	for _, e := range entries[:limit] {
		fmt.Fprintf(sb, "- %s: %d markers\n", e.path, e.count)
	}
	sb.WriteString("\n")
}

func writeGitLogSection(sb *strings.Builder, gitLog []string) {
	sb.WriteString("## Recent Git History\n\n")
	if len(gitLog) == 0 {
		sb.WriteString("No git history available.\n\n")
		return
	}
	for _, line := range gitLog {
		fmt.Fprintf(sb, "- %s\n", line)
	}
	sb.WriteString("\n")
}
