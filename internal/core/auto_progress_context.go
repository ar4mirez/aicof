package core

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Progress context constants
const (
	AutoProgressContextFile    = "progress-context.md"
	DefaultMaxContextLearnings = 50
	DefaultMaxContextCompleted = 10
	DefaultMaxProgressLines    = 500
)

// ProgressContextConfig controls how progress-context.md is generated.
type ProgressContextConfig struct {
	MaxLearnings     int
	MaxCompleted     int
	MaxProgressLines int
}

// DefaultProgressContextConfig returns sensible defaults.
func DefaultProgressContextConfig() ProgressContextConfig {
	return ProgressContextConfig{
		MaxLearnings:     DefaultMaxContextLearnings,
		MaxCompleted:     DefaultMaxContextCompleted,
		MaxProgressLines: DefaultMaxProgressLines,
	}
}

// ProgressContextConfigFromAuto creates a config from AutoConfig fields,
// falling back to defaults for zero values.
func ProgressContextConfigFromAuto(cfg AutoConfig) ProgressContextConfig {
	pcc := DefaultProgressContextConfig()
	if cfg.ProgressMaxLearnings > 0 {
		pcc.MaxLearnings = cfg.ProgressMaxLearnings
	}
	if cfg.ProgressMaxCompleted > 0 {
		pcc.MaxCompleted = cfg.ProgressMaxCompleted
	}
	if cfg.ProgressMaxLines > 0 {
		pcc.MaxProgressLines = cfg.ProgressMaxLines
	}
	return pcc
}

// GenerateProgressContext reads progress.md and writes a compact
// progress-context.md containing only learnings and recent completions.
func GenerateProgressContext(progressPath, contextPath string, cfg ProgressContextConfig) error {
	lines, err := readAllLines(progressPath)
	if err != nil {
		if os.IsNotExist(err) {
			return writeEmptyContext(contextPath)
		}
		return fmt.Errorf("failed to read progress file: %w", err)
	}
	if len(lines) == 0 {
		return writeEmptyContext(contextPath)
	}

	learnings := extractEntries(lines, "LEARNING:", cfg.MaxLearnings)
	completed := extractEntries(lines, "COMPLETED:", cfg.MaxCompleted)
	iterations, completions := countProgressStats(lines)

	return writeContextFile(contextPath, iterations, completions, learnings, completed)
}

func writeContextFile(path string, iters, completions int, learnings, completed []string) error {
	var sb strings.Builder
	sb.WriteString("# Progress Context (auto-generated — do not edit)\n")
	fmt.Fprintf(&sb, "Summary: %d iterations completed, %d tasks done\n", iters, completions)

	if len(learnings) > 0 {
		sb.WriteString("\n## Key Learnings\n")
		for _, l := range learnings {
			sb.WriteString(l + "\n")
		}
	}

	if len(completed) > 0 {
		sb.WriteString("\n## Recent Completions\n")
		for _, c := range completed {
			sb.WriteString(c + "\n")
		}
	}

	sb.WriteString("\nNote: Full history in progress.md. Append new learnings/status to progress.md.\n")
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func writeEmptyContext(path string) error {
	content := "# Progress Context (auto-generated — do not edit)\n" +
		"Summary: 0 iterations completed, 0 tasks done\n\n" +
		"No prior progress recorded.\n\n" +
		"Note: Full history in progress.md. Append new learnings/status to progress.md.\n"
	return os.WriteFile(path, []byte(content), 0644)
}

// extractEntries filters lines containing the given marker and returns the last max.
func extractEntries(lines []string, marker string, max int) []string {
	var matches []string
	for _, line := range lines {
		if strings.Contains(line, marker) {
			matches = append(matches, line)
		}
	}
	if max > 0 && len(matches) > max {
		matches = matches[len(matches)-max:]
	}
	return matches
}

// countProgressStats counts unique iterations and completions from progress lines.
func countProgressStats(lines []string) (iterations int, completions int) {
	seen := make(map[string]bool)
	for _, line := range lines {
		if strings.Contains(line, "COMPLETED:") {
			completions++
		}
		if idx := strings.Index(line, "[iteration:"); idx >= 0 {
			end := strings.Index(line[idx:], "]")
			if end > 0 {
				tag := line[idx : idx+end+1]
				seen[tag] = true
			}
		}
	}
	iterations = len(seen)
	return
}

func readAllLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// RotateProgressIfNeeded archives old progress entries when the file exceeds maxLines.
func RotateProgressIfNeeded(progressPath string, maxLines int) error {
	lines, err := readAllLines(progressPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read progress for rotation: %w", err)
	}
	if len(lines) <= maxLines {
		return nil
	}

	keepFrom := len(lines) - maxLines/2
	archiveLines := lines[:keepFrom]
	keepLines := lines[keepFrom:]

	archiveName := fmt.Sprintf("progress-archive-%s.md",
		time.Now().UTC().Format("20060102-150405"))
	archivePath := filepath.Join(filepath.Dir(progressPath), archiveName)

	archiveContent := strings.Join(archiveLines, "\n") + "\n"
	if err := os.WriteFile(archivePath, []byte(archiveContent), 0644); err != nil {
		return fmt.Errorf("failed to write archive: %w", err)
	}

	keepContent := strings.Join(keepLines, "\n") + "\n"
	if err := os.WriteFile(progressPath, []byte(keepContent), 0644); err != nil {
		return fmt.Errorf("failed to rewrite progress: %w", err)
	}

	return nil
}

// PrepareProgressContext generates progress-context.md and optionally
// rotates progress.md. Exported for use from the commands package.
func PrepareProgressContext(projectDir string) {
	progressPath := filepath.Join(projectDir, AutoDir, AutoProgressFile)
	contextPath := filepath.Join(projectDir, AutoDir, AutoProgressContextFile)
	_ = GenerateProgressContext(progressPath, contextPath, DefaultProgressContextConfig())
	_ = RotateProgressIfNeeded(progressPath, DefaultMaxProgressLines)
}
