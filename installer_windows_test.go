//go:build windows

package release_test

import (
	"bytes"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type windowsEnvironment struct {
	home, installDir, binary, npxLog, fakeBin, userPathFile string
	env                                                     []string
}

func newWindowsEnvironment(t *testing.T, f *releaseFixture) windowsEnvironment {
	t.Helper()
	home := t.TempDir()
	installDir := filepath.Join(home, "bin")
	fakeBin := filepath.Join(home, "fake-bin")
	if err := os.MkdirAll(filepath.Join(home, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	npxLog := filepath.Join(home, "npx.log")
	writeWindowsFakeNpx(t, fakeBin, false)
	userPathFile := filepath.Join(home, "user-path.txt")
	return windowsEnvironment{
		home: home, installDir: installDir, binary: filepath.Join(installDir, "ai-history.exe"),
		npxLog: npxLog, fakeBin: fakeBin, userPathFile: userPathFile,
		env: []string{
			"HOME=" + home, "USERPROFILE=" + home, "AI_HISTORY_INSTALL_DIR=" + installDir,
			"AI_HISTORY_RELEASE_BASE_URL=" + f.baseURL, "AI_HISTORY_LATEST_RELEASE_URL=" + f.latestURL,
			"AI_HISTORY_NPX_LOG=" + npxLog, "AI_HISTORY_TEST_USER_PATH_FILE=" + userPathFile,
			"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
		},
	}
}

func writeWindowsFakeNpx(t *testing.T, dir string, failCursor bool) {
	t.Helper()
	body := "@echo off\r\necho %*>>\"%AI_HISTORY_NPX_LOG%\"\r\n"
	if failCursor {
		body += "echo %*| findstr /C:\"--agent cursor\" >nul && exit /b 7\r\n"
	}
	body += "exit /b 0\r\n"
	if err := os.WriteFile(filepath.Join(dir, "npx.cmd"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func runWindowsInstaller(t *testing.T, e windowsEnvironment, args ...string) commandResult {
	t.Helper()
	allArgs := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", `scripts\install.ps1`}
	allArgs = append(allArgs, args...)
	return runCommand(t, e.env, "powershell", allArgs...)
}

func windowsInstall(t *testing.T, e windowsEnvironment, version string, extra ...string) commandResult {
	t.Helper()
	args := []string{"-Version", version, "-NoModifyPath"}
	args = append(args, extra...)
	return runWindowsInstaller(t, e, args...)
}

func TestPowerShellInstallerInstallsAndReruns(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	requireSuccess(t, windowsInstall(t, e, installerFixtureVersion))
	if got := binaryVersion(t, e.binary); got != "ai-history "+installerFixtureVersion {
		t.Fatalf("version = %q", got)
	}
	hits := f.hits(installerFixtureVersion)
	result := windowsInstall(t, e, installerFixtureVersion)
	requireSuccess(t, result)
	if !strings.Contains(strings.ToLower(result.output), "already installed") {
		t.Fatalf("missing already installed message:\n%s", result.output)
	}
	if got := f.hits(installerFixtureVersion); got != hits {
		t.Fatalf("rerun downloaded archive: %d -> %d", hits, got)
	}
}

func TestPowerShellInstallerUpgradesAndExplicitlyDowngrades(t *testing.T) {
	f := newReleaseFixture(t, []string{"v1.2.2", installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	for _, version := range []string{"v1.2.2", installerFixtureVersion, "v1.2.2"} {
		requireSuccess(t, windowsInstall(t, e, version))
		if got := binaryVersion(t, e.binary); got != "ai-history "+version {
			t.Fatalf("after %s: %q", version, got)
		}
	}
}

func TestPowerShellInstallerResolvesLatest(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	requireSuccess(t, runWindowsInstaller(t, e, "-NoModifyPath"))
	if got := binaryVersion(t, e.binary); got != "ai-history "+installerFixtureVersion {
		t.Fatalf("version = %q", got)
	}
}

func TestPowerShellInstallerReportsLatestReleaseFailure(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	f.setLatestStatus(http.StatusBadGateway)
	e := newWindowsEnvironment(t, f)
	requireFailure(t, runWindowsInstaller(t, e, "-NoModifyPath"), "latest")
	if _, err := os.Stat(e.binary); !os.IsNotExist(err) {
		t.Fatalf("binary should not exist: %v", err)
	}
}

func TestPowerShellInstallerRejectsMissingRelease(t *testing.T) {
	const missingVersion = "v9.9.9"
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	requireFailure(t, windowsInstall(t, e, missingVersion), missingVersion)
	if _, err := os.Stat(e.binary); !os.IsNotExist(err) {
		t.Fatalf("binary should not exist: %v", err)
	}
	if requests := f.requests(); requests == 0 {
		t.Fatal("missing release was rejected before making a network request")
	}
}

func TestPowerShellInstallerRejectsBadChecksum(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, false)
	e := newWindowsEnvironment(t, f)
	requireFailure(t, windowsInstall(t, e, installerFixtureVersion), "checksum")
	if _, err := os.Stat(e.binary); !os.IsNotExist(err) {
		t.Fatalf("binary should not exist: %v", err)
	}
}

func installOldWindowsBinary(t *testing.T, e windowsEnvironment) []byte {
	t.Helper()
	if err := os.MkdirAll(e.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	old := []byte("old windows installation bytes\r\n")
	if err := os.WriteFile(e.binary, old, 0o755); err != nil {
		t.Fatal(err)
	}
	return old
}

func TestPowerShellInstallerPreservesOldVersionAfterInterruptedDownload(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	f.setInterrupted(installerFixtureVersion, true)
	e := newWindowsEnvironment(t, f)
	old := installRecognizableOldBinary(t, e.installDir, e.binary)
	requireFailure(t, windowsInstall(t, e, installerFixtureVersion), "download")
	got, err := os.ReadFile(e.binary)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, old) {
		t.Fatal("old binary changed")
	}
}

func TestPowerShellInstallerPreservesUnknownTarget(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	old := installOldWindowsBinary(t, e)
	e.env = append(e.env, "AI_HISTORY_TEST_OS="+runtime.GOOS, "AI_HISTORY_TEST_ARCH="+runtime.GOARCH)
	requireFailure(t, windowsInstall(t, e, installerFixtureVersion), "existing")
	got, err := os.ReadFile(e.binary)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, old) {
		t.Fatal("old binary changed")
	}
	if requests := f.requests(); requests != 0 {
		t.Fatalf("unknown target made %d network requests", requests)
	}
}

func TestPowerShellInstallerRejectsUnsupportedPlatformBeforeDownload(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	e.env = append(e.env, "AI_HISTORY_TEST_OS=plan9", "AI_HISTORY_TEST_ARCH=amd64")
	requireFailure(t, windowsInstall(t, e, installerFixtureVersion), "unsupported")
	if requests := f.requests(); requests != 0 {
		t.Fatalf("made %d network requests", requests)
	}
}

func TestPowerShellInstallerArtifactMappingMatchesGoReleaser(t *testing.T) {
	config := readFile(t, ".goreleaser.yaml")
	for _, value := range []string{"windows", "amd64", "arm64", "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}", "goos: windows", "formats:\n          - zip"} {
		if !strings.Contains(config, value) {
			t.Fatalf(".goreleaser.yaml missing mapping token %q", value)
		}
	}
	for _, target := range []struct{ goarch, want string }{
		{"amd64", "ai-history_1.2.3_windows_amd64.zip"},
		{"arm64", "ai-history_1.2.3_windows_arm64.zip"},
	} {
		t.Run(target.goarch, func(t *testing.T) {
			f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
			e := newWindowsEnvironment(t, f)
			e.env = append(e.env, "AI_HISTORY_TEST_OS=windows", "AI_HISTORY_TEST_ARCH="+target.goarch)
			requireSuccess(t, windowsInstall(t, e, installerFixtureVersion))
			if hits := f.archiveNameHits(target.want); hits == 0 {
				t.Fatalf("installer did not request %s", target.want)
			}
		})
	}
}

func TestPowerShellInstallerUpdatesPathOnce(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	requireSuccess(t, runWindowsInstaller(t, e, "-Version", installerFixtureVersion))
	requireSuccess(t, runWindowsInstaller(t, e, "-Version", installerFixtureVersion))
	path := readOptional(t, e.userPathFile)
	if got := strings.Count(strings.ToLower(path), strings.ToLower(e.installDir)); got != 1 {
		t.Fatalf("install dir occurs %d times: %q", got, path)
	}
}

func TestPowerShellInstallerHonorsNoModifyPath(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	requireSuccess(t, windowsInstall(t, e, installerFixtureVersion))
	if data := readOptional(t, e.userPathFile); strings.Contains(strings.ToLower(data), strings.ToLower(e.installDir)) {
		t.Fatalf("NoModifyPath changed PATH: %q", data)
	}
}

func TestPowerShellInstallerWarnsAboutAnotherPathBinary(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	other := filepath.Join(e.home, "other-bin")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "ai-history.cmd"), []byte("@exit /b 0\r\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	e.env = append(e.env, "PATH="+other+string(os.PathListSeparator)+e.fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	result := windowsInstall(t, e, installerFixtureVersion)
	requireSuccess(t, result)
	if !strings.Contains(strings.ToLower(result.output), "warning") || !strings.Contains(strings.ToLower(result.output), strings.ToLower(other)) {
		t.Fatalf("missing conflict warning:\n%s", result.output)
	}
}

func TestPowerShellInstallerStopsBeforeSkillWhenDoctorIsInvalid(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	f.replaceVersionBinary(installerFixtureVersion, false)
	e := newWindowsEnvironment(t, f)
	requireFailure(t, windowsInstall(t, e, installerFixtureVersion, "-WithSkill", "-Agent", "cursor"), "doctor")
	if log := readOptional(t, e.npxLog); log != "" {
		t.Fatalf("npx ran: %s", log)
	}
}

func TestPowerShellInstallerInstallsTaggedSkillForExplicitAgents(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	for _, dir := range []string{".codex", ".claude", ".cursor"} {
		if err := os.RemoveAll(filepath.Join(e.home, dir)); err != nil {
			t.Fatal(err)
		}
	}
	windowsRoot := os.Getenv("SystemRoot")
	e.env = append(e.env, "PATH="+strings.Join([]string{
		e.fakeBin,
		filepath.Join(windowsRoot, "System32"),
		filepath.Join(windowsRoot, "System32", "WindowsPowerShell", "v1.0"),
	}, string(os.PathListSeparator)))
	args := []string{"-WithSkill", "-Agent", "codex,claude-code,cursor"}
	requireSuccess(t, windowsInstall(t, e, installerFixtureVersion, args...))
	archiveHits := f.hits(installerFixtureVersion)
	requireSuccess(t, windowsInstall(t, e, installerFixtureVersion, args...))
	if got := f.hits(installerFixtureVersion); got != archiveHits {
		t.Fatalf("same-version skill refresh downloaded archive: hits %d -> %d", archiveHits, got)
	}
	log := readOptional(t, e.npxLog)
	source := "https://github.com/yangkushu/ai-session-history/tree/v1.2.3/skills/ai-history"
	assertSkillCommandLog(t, log, source, "codex", "claude-code", "cursor", "codex", "claude-code", "cursor")
}

func TestPowerShellInstallerDetectsInstalledAgents(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	for _, dir := range []string{".codex", ".claude", ".cursor"} {
		if err := os.MkdirAll(filepath.Join(e.home, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	windowsRoot := os.Getenv("SystemRoot")
	e.env = append(e.env, "PATH="+strings.Join([]string{
		e.fakeBin,
		filepath.Join(windowsRoot, "System32"),
		filepath.Join(windowsRoot, "System32", "WindowsPowerShell", "v1.0"),
	}, string(os.PathListSeparator)))
	requireSuccess(t, windowsInstall(t, e, installerFixtureVersion, "-WithSkill"))
	log := readOptional(t, e.npxLog)
	source := "https://github.com/yangkushu/ai-session-history/tree/v1.2.3/skills/ai-history"
	assertSkillCommandLog(t, log, source, "codex", "claude-code", "cursor")
}

func TestPowerShellInstallerRequiresAgentWhenNoneDetected(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	for _, dir := range []string{".codex", ".claude", ".cursor"} {
		if err := os.RemoveAll(filepath.Join(e.home, dir)); err != nil {
			t.Fatal(err)
		}
	}
	windowsRoot := os.Getenv("SystemRoot")
	e.env = append(e.env, "PATH="+strings.Join([]string{
		e.fakeBin,
		filepath.Join(windowsRoot, "System32"),
		filepath.Join(windowsRoot, "System32", "WindowsPowerShell", "v1.0"),
	}, string(os.PathListSeparator)))
	requireFailure(t, windowsInstall(t, e, installerFixtureVersion, "-WithSkill"), "agent")
	if _, err := os.Stat(e.binary); err != nil {
		t.Fatalf("binary should remain installed: %v", err)
	}
	if log := readOptional(t, e.npxLog); log != "" {
		t.Fatalf("npx ran without a detected agent: %s", log)
	}
}

func TestPowerShellInstallerKeepsBinaryWhenNpxIsMissing(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	if err := os.Remove(filepath.Join(e.fakeBin, "npx.cmd")); err != nil {
		t.Fatal(err)
	}
	e.env = append(e.env, "PATH="+filepath.Dir(mustLookPath(t, "powershell.exe")))
	requireFailure(t, windowsInstall(t, e, installerFixtureVersion, "-WithSkill", "-Agent", "cursor"), "npx")
	if got := binaryVersion(t, e.binary); got != "ai-history "+installerFixtureVersion {
		t.Fatalf("binary not preserved: %q", got)
	}
}

func TestPowerShellInstallerReportsPartialAgentFailure(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	writeWindowsFakeNpx(t, e.fakeBin, true)
	result := windowsInstall(t, e, installerFixtureVersion, "-WithSkill", "-Agent", "codex,cursor,claude-code")
	requireFailure(t, result, "cursor")
	if !strings.Contains(strings.ToLower(result.output), "partial") {
		t.Fatalf("missing partial report:\n%s", result.output)
	}
	if got := binaryVersion(t, e.binary); got != "ai-history "+installerFixtureVersion {
		t.Fatalf("binary not preserved: %q", got)
	}
	log := readOptional(t, e.npxLog)
	source := "https://github.com/yangkushu/ai-session-history/tree/v1.2.3/skills/ai-history"
	assertSkillCommandLog(t, log, source, "codex", "cursor", "claude-code")
}

func mustLookPath(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Fatal(err)
	}
	return path
}
