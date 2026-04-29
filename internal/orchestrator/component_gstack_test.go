package orchestrator

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// withGstackExec replaces the package's exec function for the duration of
// the test. Restores on cleanup. Tests use this to inject fake commands
// instead of running real git or the gstack setup script.
func withGstackExec(t *testing.T, fn func(ctx context.Context, name string, args ...string) *exec.Cmd) {
	t.Helper()
	prev := gstackExec
	gstackExec = fn
	t.Cleanup(func() { gstackExec = prev })
}

// fakeCmd builds an exec.Cmd whose Run/Output/CombinedOutput resolves
// without actually invoking a subprocess. The trick: invoke the test
// binary itself with a sentinel arg that asks the helper to print the
// stub output and exit with the requested code.
func fakeCmd(t *testing.T, stdout string, exitCode int) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", stdout)
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"HELPER_EXIT_CODE="+strings.TrimSpace(itoa(exitCode)),
	)
	return cmd
}

// itoa wraps strconv.Itoa without importing strconv at the helper level.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// TestHelperProcess is the test-helper entry point. When run with
// GO_WANT_HELPER_PROCESS=1, it acts as the fake subprocess: prints the
// final argv element to stdout, then exits with HELPER_EXIT_CODE.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) > 0 {
		_, _ = os.Stdout.WriteString(args[0])
	}
	codeStr := os.Getenv("HELPER_EXIT_CODE")
	code := 0
	for _, c := range codeStr {
		if c >= '0' && c <= '9' {
			code = code*10 + int(c-'0')
		}
	}
	os.Exit(code)
}

func TestGstack_Detect_NotInstalled(t *testing.T) {
	dir := t.TempDir()
	g := NewGstackComponent(dir)
	res, err := g.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if res.Installed {
		t.Errorf("expected Installed=false, got true")
	}
	if !strings.HasSuffix(res.Path, gstackInstallDir) {
		t.Errorf("Path should end with %q, got %q", gstackInstallDir, res.Path)
	}
}

func TestGstack_Detect_InstalledReadsShortSHA(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, gstackInstallDir, ".git")
	if err := os.MkdirAll(gitDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	withGstackExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Verify call shape: git -C <path> rev-parse --short HEAD
		if name != "git" {
			t.Errorf("expected git, got %q", name)
		}
		return fakeCmd(t, "abc1234\n", 0)
	})

	g := NewGstackComponent(dir)
	res, err := g.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !res.Installed {
		t.Errorf("expected Installed=true")
	}
	if res.Version != "abc1234" {
		t.Errorf("Version = %q, want %q", res.Version, "abc1234")
	}
}

func TestGstack_Detect_GitFailureDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, gstackInstallDir, ".git")
	if err := os.MkdirAll(gitDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	withGstackExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return fakeCmd(t, "fatal: not a git repository", 128)
	})

	g := NewGstackComponent(dir)
	res, err := g.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect should not error on git failure; got %v", err)
	}
	if !res.Installed {
		t.Errorf("Installed should be true (tree exists), got false")
	}
	if res.Version != "" {
		t.Errorf("Version should be empty when git fails; got %q", res.Version)
	}
}

func TestGstack_Install_AlreadyInstalledIsNoOp(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, gstackInstallDir, ".git")
	if err := os.MkdirAll(gitDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	calls := 0
	withGstackExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		calls++
		// The Detect call returns the pinned SHA (truncated to 7 chars).
		return fakeCmd(t, gstackPinnedSHA[:7]+"\n", 0)
	})

	g := NewGstackComponent(dir)
	res, err := g.Install(context.Background(), InstallOptions{})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !res.AlreadyInstalled {
		t.Errorf("expected AlreadyInstalled=true; got %+v", res)
	}
	if len(res.Mutations) != 0 {
		t.Errorf("AlreadyInstalled should produce zero mutations; got %d", len(res.Mutations))
	}
	// Detect should run only once (no clone, no checkout, no setup).
	if calls != 1 {
		t.Errorf("expected 1 exec call (Detect), got %d", calls)
	}
}

func TestGstack_Install_DriftWithoutForceErrors(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, gstackInstallDir, ".git")
	if err := os.MkdirAll(gitDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	withGstackExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Detect returns an unrelated SHA — drift.
		return fakeCmd(t, "deadbee\n", 0)
	})

	g := NewGstackComponent(dir)
	_, err := g.Install(context.Background(), InstallOptions{})
	if err == nil {
		t.Fatalf("expected drift error, got nil")
	}
	var oe *Error
	if !errors.As(err, &oe) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if !oe.Recoverable {
		t.Errorf("drift error should be Recoverable")
	}
	if !strings.Contains(oe.Fix, "--force") {
		t.Errorf("Fix should mention --force; got %q", oe.Fix)
	}
}

func TestGstack_Install_DryRunDoesNotMutate(t *testing.T) {
	dir := t.TempDir()
	withGstackExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		t.Errorf("DryRun must not exec any subprocess; got %s %v", name, args)
		return fakeCmd(t, "", 0)
	})

	g := NewGstackComponent(dir)
	res, err := g.Install(context.Background(), InstallOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Install DryRun: %v", err)
	}
	if len(res.Mutations) != 0 {
		t.Errorf("DryRun must not record mutations; got %d", len(res.Mutations))
	}
	if _, statErr := os.Stat(filepath.Join(dir, gstackInstallDir)); !os.IsNotExist(statErr) {
		t.Errorf("DryRun must not create install dir; stat=%v", statErr)
	}
}

func TestGstack_Install_SkipFlag(t *testing.T) {
	g := NewGstackComponent(t.TempDir())
	res, err := g.Install(context.Background(), InstallOptions{SkipGstack: true})
	if err != nil {
		t.Fatalf("Install with SkipGstack: %v", err)
	}
	if !res.Skipped {
		t.Errorf("expected Skipped=true")
	}
}

func TestGstack_Check_NotInstalled(t *testing.T) {
	g := NewGstackComponent(t.TempDir())
	st := g.Check(context.Background())
	if st.OK {
		t.Errorf("Check should report unhealthy when not installed")
	}
	if st.FixHint != "samuel init" {
		t.Errorf("FixHint = %q, want %q", st.FixHint, "samuel init")
	}
}

func TestGstack_Check_VersionMismatch(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, gstackInstallDir, ".git")
	if err := os.MkdirAll(gitDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	withGstackExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return fakeCmd(t, "deadbee\n", 0)
	})
	g := NewGstackComponent(dir)
	st := g.Check(context.Background())
	if st.OK {
		t.Errorf("Check should report unhealthy on SHA drift")
	}
	if !strings.Contains(st.FixHint, "--force") {
		t.Errorf("FixHint should suggest --force; got %q", st.FixHint)
	}
}

func TestGstack_Check_HealthyAtPinnedSHA(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, gstackInstallDir, ".git")
	if err := os.MkdirAll(gitDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	withGstackExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return fakeCmd(t, gstackPinnedSHA[:7]+"\n", 0)
	})
	g := NewGstackComponent(dir)
	st := g.Check(context.Background())
	if !st.OK {
		t.Errorf("Check should be OK when at pinned SHA; got Message=%q", st.Message)
	}
}

func TestGstack_Uninstall_KeepsInstall(t *testing.T) {
	g := NewGstackComponent(t.TempDir())
	res, err := g.Uninstall(context.Background(), UninstallOptions{All: true})
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !res.Skipped {
		t.Errorf("Uninstall must Skip — gstack is composed and user-owned")
	}
	if len(res.Mutations) != 0 {
		t.Errorf("Uninstall must not record mutations; got %d", len(res.Mutations))
	}
}

func TestMatchesShortSHA(t *testing.T) {
	cases := []struct {
		short, full string
		want        bool
	}{
		{"abc1234", "abc1234567890def1234567890abcdef12345678", true},
		{"ABC1234", "abc1234567890def1234567890abcdef12345678", true},
		{"abc1234", "deadbeef567890def1234567890abcdef12345678", false},
		{"", "abc1234567890def1234567890abcdef12345678", false},
		{"abc1234", "", false},
		// short longer than full → no match
		{"abc1234567890def1234567890abcdef12345678abc", "abc1234567890def1234567890abcdef12345678", false},
	}
	for _, tc := range cases {
		if got := matchesShortSHA(tc.short, tc.full); got != tc.want {
			t.Errorf("matchesShortSHA(%q, %q) = %v, want %v", tc.short, tc.full, got, tc.want)
		}
	}
}

// Sanity: ensure the constants this test depends on are sensible.
func TestGstack_PinnedSHA_LookSane(t *testing.T) {
	if len(gstackPinnedSHA) != 40 {
		t.Errorf("gstackPinnedSHA must be a full 40-char SHA; got %d chars", len(gstackPinnedSHA))
	}
	for _, c := range gstackPinnedSHA {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("gstackPinnedSHA must be lowercase hex; got %q", gstackPinnedSHA)
			break
		}
	}
	if runtime.GOOS == "" {
		t.Skip("runtime.GOOS empty?")
	}
}
