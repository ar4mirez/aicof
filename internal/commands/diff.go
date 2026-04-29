package commands

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ar4mirez/samuel/internal/core"
	"github.com/ar4mirez/samuel/internal/ui"
	"github.com/spf13/cobra"
)

// VersionDiff represents differences between two versions
type VersionDiff struct {
	FromVersion string
	ToVersion   string
	Added       []string
	Removed     []string
	Modified    []string
	Unchanged   int
}

// diffCmd is preserved as a Hidden+Deprecated alias for v3.0.0.
// Bare 'samuel diff' moved to 'samuel update --preview'.
// 'samuel diff v1 v2' moved to 'samuel admin diff v1 v2'.
var diffCmd = &cobra.Command{
	Use:    "diff [version1] [version2]",
	Short:  "[DEPRECATED] Use 'samuel update --preview' or 'samuel admin diff v1 v2'",
	Hidden: true,
	Args:   cobra.MaximumNArgs(2),
	RunE:   redirectAndRun("samuel update --preview (bare) or samuel admin diff v1 v2 (two-arg)", runDiff),
}

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().BoolP("installed", "i", false, "Compare installed files with latest version")
	diffCmd.Flags().Bool("components", false, "Show component-level changes instead of files")
}

func runDiff(cmd *cobra.Command, args []string) error {
	showComponents, _ := cmd.Flags().GetBool("components")

	var diff *VersionDiff
	var err error

	if len(args) == 2 {
		// Compare two specific versions
		diff, err = compareVersions(args[0], args[1])
	} else {
		// Compare installed with latest
		diff, err = compareInstalledWithLatest()
	}

	if err != nil {
		return err
	}

	if JSONMode(cmd) {
		PrintJSONForCmd(cmd, map[string]interface{}{
			"fromVersion": diff.FromVersion,
			"toVersion":   diff.ToVersion,
			"added":       diff.Added,
			"removed":     diff.Removed,
			"modified":    diff.Modified,
			"unchanged":   diff.Unchanged,
		})
		return nil
	}

	// Display diff
	if showComponents {
		displayComponentDiff(diff)
	} else {
		displayFileDiff(diff)
	}

	return nil
}

func compareInstalledWithLatest() (*VersionDiff, error) {
	// Load config to get installed version
	config, err := core.LoadConfig()
	if err != nil {
		if os.IsNotExist(err) {
			ui.Warn("No Samuel installation found in current directory")
			return nil, fmt.Errorf("no installation found")
		}
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	installedVersion := config.Version

	// Get latest version
	downloader, err := core.NewDownloader()
	if err != nil {
		return nil, fmt.Errorf("failed to create downloader: %w", err)
	}

	latestVersion, err := downloader.GetLatestVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	if installedVersion == latestVersion {
		ui.Success("You are on the latest version (%s)", installedVersion)
		return &VersionDiff{
			FromVersion: installedVersion,
			ToVersion:   latestVersion,
		}, nil
	}

	// Compare installed files with latest version
	return compareLocalWithVersion(installedVersion, latestVersion, downloader)
}

func compareLocalWithVersion(installedVersion, latestVersion string, downloader *core.Downloader) (*VersionDiff, error) {
	ui.Info("Comparing installed (%s) with latest (%s)...", installedVersion, latestVersion)
	fmt.Println()

	// Download latest version to cache
	spinner := ui.NewSpinner("Downloading latest version...")
	spinner.Start()

	latestPath, err := downloader.DownloadVersion(latestVersion)
	if err != nil {
		spinner.Error("Failed to download")
		return nil, fmt.Errorf("failed to download latest version: %w", err)
	}
	spinner.Success("Downloaded latest version")

	// Get file hashes for local installation
	localFiles := getLocalFileHashes(".")

	// Get file hashes for latest version
	latestFiles := getVersionFileHashes(latestPath)

	// Compute diff
	diff := computeDiff(installedVersion, latestVersion, localFiles, latestFiles)

	return diff, nil
}

func compareVersions(v1, v2 string) (*VersionDiff, error) {
	ui.Info("Comparing %s with %s...", v1, v2)
	fmt.Println()

	downloader, err := core.NewDownloader()
	if err != nil {
		return nil, fmt.Errorf("failed to create downloader: %w", err)
	}

	// Download both versions
	spinner := ui.NewSpinner("Downloading versions...")
	spinner.Start()

	path1, err := downloader.DownloadVersion(v1)
	if err != nil {
		spinner.Error("Failed to download " + v1)
		return nil, fmt.Errorf("failed to download %s: %w", v1, err)
	}

	path2, err := downloader.DownloadVersion(v2)
	if err != nil {
		spinner.Error("Failed to download " + v2)
		return nil, fmt.Errorf("failed to download %s: %w", v2, err)
	}
	spinner.Success("Downloaded both versions")

	// Get file hashes
	files1 := getVersionFileHashes(path1)
	files2 := getVersionFileHashes(path2)

	// Compute diff
	diff := computeDiff(v1, v2, files1, files2)

	return diff, nil
}

func getLocalFileHashes(basePath string) map[string]string {
	hashes := make(map[string]string)

	// Only check Samuel-related files
	patterns := []string{
		"CLAUDE.md",
		"AGENTS.md",
		".claude/**/*.md",
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(basePath, pattern))
		if err != nil {
			ui.Warn("Failed to glob pattern %q: %v", pattern, err)
			continue
		}
		for _, match := range matches {
			relPath, err := filepath.Rel(basePath, match)
			if err != nil {
				ui.Warn("Failed to compute relative path for %q: %v", match, err)
				continue
			}
			if hash, err := hashFile(match); err == nil {
				hashes[relPath] = hash
			}
		}
	}

	// Also walk .agent directory explicitly
	agentDir := filepath.Join(basePath, ".agent")
	if err := filepath.Walk(agentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".md") {
			relPath, relErr := filepath.Rel(basePath, path)
			if relErr != nil {
				ui.Warn("Failed to compute relative path for %q: %v", path, relErr)
				return nil
			}
			if hash, err := hashFile(path); err == nil {
				hashes[relPath] = hash
			}
		}
		return nil
	}); err != nil && !os.IsNotExist(err) {
		ui.Warn("Failed to walk .agent directory: %v", err)
	}

	return hashes
}

func getVersionFileHashes(cachePath string) map[string]string {
	hashes := make(map[string]string)

	// The template files are in cachePath/template/
	templatePath := filepath.Join(cachePath, "template")

	if err := filepath.Walk(templatePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Only track markdown files and key files
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		relPath, relErr := filepath.Rel(templatePath, path)
		if relErr != nil {
			ui.Warn("Failed to compute relative path for %q: %v", path, relErr)
			return nil
		}
		if hash, err := hashFile(path); err == nil {
			hashes[relPath] = hash
		}
		return nil
	}); err != nil && !os.IsNotExist(err) {
		ui.Warn("Failed to walk template directory: %v", err)
	}

	return hashes
}

func computeDiff(v1, v2 string, files1, files2 map[string]string) *VersionDiff {
	diff := &VersionDiff{
		FromVersion: v1,
		ToVersion:   v2,
	}

	// Find added and modified files
	for path, hash2 := range files2 {
		hash1, exists := files1[path]
		if !exists {
			diff.Added = append(diff.Added, path)
		} else if hash1 != hash2 {
			diff.Modified = append(diff.Modified, path)
		} else {
			diff.Unchanged++
		}
	}

	// Find removed files
	for path := range files1 {
		if _, exists := files2[path]; !exists {
			diff.Removed = append(diff.Removed, path)
		}
	}

	// Sort for consistent output
	sort.Strings(diff.Added)
	sort.Strings(diff.Removed)
	sort.Strings(diff.Modified)

	return diff
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Display functions are in diff_display.go
