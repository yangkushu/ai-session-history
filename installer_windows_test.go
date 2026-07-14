//go:build windows

package release_test

import (
	"bytes"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

func TestWindowsInstallerInstallsAndReruns(t *testing.T) {
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

func TestWindowsInstallerUpgradesAndExplicitlyDowngrades(t *testing.T) {
	f := newReleaseFixture(t, []string{"v1.2.2", installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	for _, version := range []string{"v1.2.2", installerFixtureVersion, "v1.2.2"} {
		requireSuccess(t, windowsInstall(t, e, version))
		if got := binaryVersion(t, e.binary); got != "ai-history "+version {
			t.Fatalf("after %s: %q", version, got)
		}
	}
}

func TestWindowsInstallerResolvesLatest(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	requireSuccess(t, runWindowsInstaller(t, e, "-NoModifyPath"))
	if got := binaryVersion(t, e.binary); got != "ai-history "+installerFixtureVersion {
		t.Fatalf("version = %q", got)
	}
}

func TestWindowsInstallerReportsLatestReleaseFailure(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	f.latestStatus = http.StatusBadGateway
	e := newWindowsEnvironment(t, f)
	requireFailure(t, runWindowsInstaller(t, e, "-NoModifyPath"), "latest")
}

func TestWindowsInstallerRejectsBadChecksum(t *testing.T) {
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

func TestWindowsInstallerPreservesOldVersionAfterInterruptedDownload(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	f.interrupt[installerFixtureVersion] = true
	e := newWindowsEnvironment(t, f)
	old := installOldWindowsBinary(t, e)
	requireFailure(t, windowsInstall(t, e, installerFixtureVersion), "download")
	got, err := os.ReadFile(e.binary)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, old) {
		t.Fatal("old binary changed")
	}
}

func TestWindowsInstallerPreservesUnknownTarget(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	old := installOldWindowsBinary(t, e)
	requireFailure(t, windowsInstall(t, e, "v9.9.9"), "404")
	got, err := os.ReadFile(e.binary)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, old) {
		t.Fatal("old binary changed")
	}
}

func TestWindowsInstallerRejectsUnsupportedPlatformBeforeDownload(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	e.env = append(e.env, "AI_HISTORY_TEST_OS=plan9", "AI_HISTORY_TEST_ARCH=amd64")
	requireFailure(t, windowsInstall(t, e, installerFixtureVersion), "unsupported")
	if hits := f.hits(installerFixtureVersion); hits != 0 {
		t.Fatalf("made %d archive requests", hits)
	}
}

func TestWindowsInstallerUpdatesPathOnce(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	requireSuccess(t, runWindowsInstaller(t, e, "-Version", installerFixtureVersion))
	requireSuccess(t, runWindowsInstaller(t, e, "-Version", installerFixtureVersion))
	path := readOptional(t, e.userPathFile)
	if got := strings.Count(strings.ToLower(path), strings.ToLower(e.installDir)); got != 1 {
		t.Fatalf("install dir occurs %d times: %q", got, path)
	}
}

func TestWindowsInstallerHonorsNoModifyPath(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	requireSuccess(t, windowsInstall(t, e, installerFixtureVersion))
	if data := readOptional(t, e.userPathFile); strings.Contains(strings.ToLower(data), strings.ToLower(e.installDir)) {
		t.Fatalf("NoModifyPath changed PATH: %q", data)
	}
}

func TestWindowsInstallerWarnsAboutAnotherPathBinary(t *testing.T) {
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

func TestWindowsInstallerStopsBeforeSkillWhenDoctorIsInvalid(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	f.replaceVersionBinary(installerFixtureVersion, false)
	e := newWindowsEnvironment(t, f)
	requireFailure(t, windowsInstall(t, e, installerFixtureVersion, "-Agent", "cursor"), "doctor")
	if log := readOptional(t, e.npxLog); log != "" {
		t.Fatalf("npx ran: %s", log)
	}
}

func TestWindowsInstallerInstallsTaggedSkillForExplicitAgents(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	requireSuccess(t, windowsInstall(t, e, installerFixtureVersion, "-Agent", "codex,claude-code,cursor"))
	log := readOptional(t, e.npxLog)
	source := "https://github.com/yangkushu/ai-session-history/tree/v1.2.3/skills/ai-history"
	for _, agent := range []string{"codex", "claude-code", "cursor"} {
		if strings.Count(log, "--agent "+agent) != 1 {
			t.Errorf("agent %s count != 1:\n%s", agent, log)
		}
	}
	if strings.Count(log, source) != 3 {
		t.Fatalf("tagged source count != 3:\n%s", log)
	}
}

func TestWindowsInstallerDetectsInstalledAgents(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	for _, dir := range []string{".codex", ".claude", ".cursor"} {
		if err := os.MkdirAll(filepath.Join(e.home, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	requireSuccess(t, windowsInstall(t, e, installerFixtureVersion))
	log := readOptional(t, e.npxLog)
	for _, agent := range []string{"codex", "claude-code", "cursor"} {
		if strings.Count(log, "--agent "+agent) != 1 {
			t.Errorf("agent %s count != 1:\n%s", agent, log)
		}
	}
}

func TestWindowsInstallerRequiresAgentWhenNoneDetected(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	if err := os.RemoveAll(filepath.Join(e.home, ".cursor")); err != nil {
		t.Fatal(err)
	}
	requireFailure(t, windowsInstall(t, e, installerFixtureVersion), "agent")
	if _, err := os.Stat(e.binary); err != nil {
		t.Fatalf("binary should remain installed: %v", err)
	}
}

func TestWindowsInstallerKeepsBinaryWhenNpxIsMissing(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	if err := os.Remove(filepath.Join(e.fakeBin, "npx.cmd")); err != nil {
		t.Fatal(err)
	}
	e.env = append(e.env, "PATH="+filepath.Dir(mustLookPath(t, "powershell.exe")))
	requireFailure(t, windowsInstall(t, e, installerFixtureVersion, "-Agent", "cursor"), "npx")
	if got := binaryVersion(t, e.binary); got != "ai-history "+installerFixtureVersion {
		t.Fatalf("binary not preserved: %q", got)
	}
}

func TestWindowsInstallerReportsPartialAgentFailure(t *testing.T) {
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newWindowsEnvironment(t, f)
	writeWindowsFakeNpx(t, e.fakeBin, true)
	result := windowsInstall(t, e, installerFixtureVersion, "-Agent", "codex,cursor")
	requireFailure(t, result, "cursor")
	if !strings.Contains(strings.ToLower(result.output), "partial") {
		t.Fatalf("missing partial report:\n%s", result.output)
	}
	if got := binaryVersion(t, e.binary); got != "ai-history "+installerFixtureVersion {
		t.Fatalf("binary not preserved: %q", got)
	}
}

func mustLookPath(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Fatal(err)
	}
	return path
}
