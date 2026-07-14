package release_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

const installerFixtureVersion = "v1.2.3"

type commandResult struct {
	output string
	err    error
}

func runCommand(t *testing.T, env []string, name string, args ...string) commandResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), env...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("command timed out: %s %s\n%s", name, strings.Join(args, " "), output.String())
	}
	return commandResult{output: output.String(), err: err}
}

type releaseFixture struct {
	t             *testing.T
	server        *httptest.Server
	baseURL       string
	latestURL     string
	mu            sync.Mutex
	archives      map[string][]byte
	checksums     map[string]string
	archiveHits   map[string]int
	totalRequests int
	latestVersion string
	latestStatus  int
	interrupt     map[string]bool
	checksumValid bool
}

func newReleaseFixture(t *testing.T, versions []string, checksumValid bool) *releaseFixture {
	t.Helper()
	installer := filepath.Join("scripts", "install.sh")
	if runtime.GOOS == "windows" {
		installer = filepath.Join("scripts", "install.ps1")
	}
	if _, err := os.Stat(installer); err != nil {
		t.Fatalf("installer contract is red because %s is unavailable: %v", installer, err)
	}
	f := &releaseFixture{
		t: t, archives: map[string][]byte{}, checksums: map[string]string{},
		archiveHits: map[string]int{}, interrupt: map[string]bool{}, checksumValid: checksumValid,
	}
	for _, version := range versions {
		f.addVersion(version, true)
	}
	if len(versions) != 0 {
		f.latestVersion = versions[len(versions)-1]
	}
	f.server = httptest.NewServer(http.HandlerFunc(f.serveHTTP))
	f.baseURL = f.server.URL
	f.latestURL = f.server.URL + "/latest"
	t.Cleanup(f.server.Close)
	return f
}

func (f *releaseFixture) addVersion(version string, doctorValid bool) {
	f.t.Helper()
	dir := f.t.TempDir()
	binary := buildFixtureBinaryWithDoctor(f.t, dir, version, doctorValid)
	binaryBytes, err := os.ReadFile(binary)
	if err != nil {
		f.t.Fatalf("read fixture binary: %v", err)
	}
	for _, target := range []struct{ goos, goarch string }{
		{"linux", "amd64"}, {"linux", "arm64"}, {"darwin", "amd64"},
		{"darwin", "arm64"}, {"windows", "amd64"}, {"windows", "arm64"},
	} {
		filename := fixtureArchiveName(version, target.goos, target.goarch)
		var archive bytes.Buffer
		if target.goos == "windows" {
			writeZip(f.t, &archive, map[string][]byte{"ai-history.exe": binaryBytes, "README.md": []byte("fixture\n")})
		} else {
			writeTarGz(f.t, &archive, map[string][]byte{"ai-history": binaryBytes, "README.md": []byte("fixture\n")})
		}
		key := version + "/" + filename
		f.archives[key] = archive.Bytes()
	}
	f.refreshChecksum(version)
}

func (f *releaseFixture) replaceVersionBinary(version string, doctorValid bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.addVersion(version, doctorValid)
}

func (f *releaseFixture) refreshChecksum(version string) {
	var lines []string
	for key, data := range f.archives {
		if !strings.HasPrefix(key, version+"/") {
			continue
		}
		sum := fmt.Sprintf("%x", sha256.Sum256(data))
		if !f.checksumValid {
			sum = strings.Repeat("0", 64)
		}
		lines = append(lines, sum+"  "+filepath.Base(key))
	}
	sort.Strings(lines)
	f.checksums[version] = strings.Join(lines, "\n") + "\n"
}

func (f *releaseFixture) serveHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	f.mu.Lock()
	defer f.mu.Unlock()
	f.totalRequests++
	if path == "latest" {
		if f.latestStatus != 0 {
			http.Error(w, "latest unavailable", f.latestStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"tag_name":%q}`, f.latestVersion)
		return
	}
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	version, name := parts[0], parts[1]
	if name == "checksums.txt" {
		checksum, ok := f.checksums[version]
		if !ok {
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, checksum)
		return
	}
	data, ok := f.archives[path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	f.archiveHits[version]++
	if f.interrupt[version] {
		w.Header().Set("Content-Length", fmt.Sprint(len(data)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data[:len(data)/2])
		if h, ok := w.(http.Hijacker); ok {
			conn, _, err := h.Hijack()
			if err == nil {
				_ = conn.Close()
			}
		}
		return
	}
	w.Write(data)
}

func (f *releaseFixture) hits(version string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.archiveHits[version]
}

func (f *releaseFixture) requests() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.totalRequests
}

func buildFixtureBinary(t *testing.T, dir, version string) string {
	t.Helper()
	return buildFixtureBinaryWithDoctor(t, dir, version, true)
}

func buildFixtureBinaryWithDoctor(t *testing.T, dir, version string, doctorValid bool) string {
	t.Helper()
	doctor := "[]"
	if !doctorValid {
		doctor = "not-json"
	}
	source := fmt.Sprintf(`package main
import ("fmt"; "os")
func main() {
 if len(os.Args) == 2 && (os.Args[1] == "version" || os.Args[1] == "--version") { fmt.Println("ai-history %s"); return }
 if len(os.Args) == 3 && os.Args[1] == "doctor" && os.Args[2] == "--json" { fmt.Println(%q); return }
 os.Exit(2)
}`, version, doctor)
	sourcePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		t.Fatalf("write fixture source: %v", err)
	}
	binaryName := "ai-history"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(dir, binaryName)
	result := runCommand(t, []string{"GOCACHE=/tmp/go-build"}, "go", "build", "-o", binaryPath, sourcePath)
	if result.err != nil {
		t.Fatalf("build fixture binary: %v\n%s", result.err, result.output)
	}
	return binaryPath
}

func writeTarGz(t *testing.T, dst io.Writer, files map[string][]byte) {
	t.Helper()
	gz := gzip.NewWriter(dst)
	tw := tar.NewWriter(gz)
	for name, data := range files {
		mode := int64(0o644)
		if name == "ai-history" {
			mode = 0o755
		}
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(data))}); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatalf("write tar data: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
}

func writeZip(t *testing.T, dst io.Writer, files map[string][]byte) {
	t.Helper()
	zw := zip.NewWriter(dst)
	for name, data := range files {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0o755)
		writer, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
}

func fixtureArchiveName(version, goos, goarch string) string {
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("ai-history_%s_%s_%s%s", strings.TrimPrefix(version, "v"), goos, goarch, ext)
}

type unixEnvironment struct {
	home, installDir, binary, npxLog, fakeBin string
	env                                       []string
}

func newUnixEnvironment(t *testing.T, f *releaseFixture) unixEnvironment {
	t.Helper()
	home := t.TempDir()
	installDir := filepath.Join(home, ".local", "bin")
	fakeBin := filepath.Join(home, "fake-bin")
	if err := os.MkdirAll(filepath.Join(home, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	npxLog := filepath.Join(home, "npx.log")
	writeFakeNpx(t, fakeBin, false)
	return unixEnvironment{
		home: home, installDir: installDir, binary: filepath.Join(installDir, "ai-history"), npxLog: npxLog, fakeBin: fakeBin,
		env: []string{
			"HOME=" + home, "USERPROFILE=" + home, "AI_HISTORY_INSTALL_DIR=" + installDir,
			"AI_HISTORY_RELEASE_BASE_URL=" + f.baseURL, "AI_HISTORY_LATEST_RELEASE_URL=" + f.latestURL,
			"AI_HISTORY_NPX_LOG=" + npxLog, "PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
		},
	}
}

func writeFakeNpx(t *testing.T, dir string, failCursor bool) {
	t.Helper()
	body := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$AI_HISTORY_NPX_LOG\"\n"
	if failCursor {
		body += "case \" $* \" in *' --agent cursor '*) exit 7;; esac\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "npx"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func runUnixInstaller(t *testing.T, e unixEnvironment, args ...string) commandResult {
	t.Helper()
	allArgs := append([]string{"scripts/install.sh"}, args...)
	return runCommand(t, e.env, "sh", allArgs...)
}

func requireSuccess(t *testing.T, result commandResult) {
	t.Helper()
	if result.err != nil {
		t.Fatalf("installer failed: %v\n%s", result.err, result.output)
	}
}

func requireFailure(t *testing.T, result commandResult, contains string) {
	t.Helper()
	if result.err == nil {
		t.Fatalf("installer unexpectedly succeeded:\n%s", result.output)
	}
	if contains != "" && !strings.Contains(strings.ToLower(result.output), strings.ToLower(contains)) {
		t.Fatalf("failure output missing %q:\n%s", contains, result.output)
	}
}

func binaryVersion(t *testing.T, binary string) string {
	t.Helper()
	result := runCommand(t, nil, binary, "version")
	if result.err != nil {
		t.Fatalf("run installed binary: %v\n%s", result.err, result.output)
	}
	return strings.TrimSpace(result.output)
}

func TestUnixInstallerInstallsAndReruns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	first := runUnixInstaller(t, e, "--version", installerFixtureVersion, "--no-modify-path")
	requireSuccess(t, first)
	if got := binaryVersion(t, e.binary); got != "ai-history "+installerFixtureVersion {
		t.Fatalf("version = %q", got)
	}
	hits := f.hits(installerFixtureVersion)
	second := runUnixInstaller(t, e, "--version", installerFixtureVersion, "--no-modify-path")
	requireSuccess(t, second)
	if !strings.Contains(strings.ToLower(second.output), "already installed") {
		t.Fatalf("missing already installed message:\n%s", second.output)
	}
	if got := f.hits(installerFixtureVersion); got != hits {
		t.Fatalf("rerun downloaded archive: hits %d -> %d", hits, got)
	}
}

func TestUnixInstallerUpgradesAndExplicitlyDowngrades(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{"v1.2.2", installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	for _, version := range []string{"v1.2.2", installerFixtureVersion, "v1.2.2"} {
		requireSuccess(t, runUnixInstaller(t, e, "--version", version, "--no-modify-path"))
		if got := binaryVersion(t, e.binary); got != "ai-history "+version {
			t.Fatalf("after %s, version = %q", version, got)
		}
	}
}

func TestUnixInstallerResolvesLatest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	requireSuccess(t, runUnixInstaller(t, e, "--no-modify-path"))
	if got := binaryVersion(t, e.binary); got != "ai-history "+installerFixtureVersion {
		t.Fatalf("version = %q", got)
	}
}

func TestUnixInstallerReportsLatestReleaseFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	f.latestStatus = http.StatusServiceUnavailable
	e := newUnixEnvironment(t, f)
	requireFailure(t, runUnixInstaller(t, e, "--no-modify-path"), "latest")
	if _, err := os.Stat(e.binary); !os.IsNotExist(err) {
		t.Fatalf("binary should not be installed: %v", err)
	}
}

func TestUnixInstallerRejectsMissingRelease(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	const missingVersion = "v9.9.9"
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	requireFailure(t, runUnixInstaller(t, e, "--version", missingVersion, "--no-modify-path"), missingVersion)
	if _, err := os.Stat(e.binary); !os.IsNotExist(err) {
		t.Fatalf("binary should not be installed: %v", err)
	}
	if requests := f.requests(); requests == 0 {
		t.Fatal("missing release was rejected before making a network request")
	}
}

func TestUnixInstallerRejectsBadChecksum(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, false)
	e := newUnixEnvironment(t, f)
	requireFailure(t, runUnixInstaller(t, e, "--version", installerFixtureVersion, "--no-modify-path"), "checksum")
	if _, err := os.Stat(e.binary); !os.IsNotExist(err) {
		t.Fatalf("binary should not be installed: %v", err)
	}
}

func installOldBinary(t *testing.T, e unixEnvironment) []byte {
	t.Helper()
	if err := os.MkdirAll(e.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	old := []byte("old installation bytes\n")
	if err := os.WriteFile(e.binary, old, 0o755); err != nil {
		t.Fatal(err)
	}
	return old
}

func installRecognizableOldBinary(t *testing.T, installDir, binary string) []byte {
	t.Helper()
	fixtureBinary := buildFixtureBinary(t, t.TempDir(), "v1.2.2")
	old, err := os.ReadFile(fixtureBinary)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, old, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := binaryVersion(t, binary); got != "ai-history v1.2.2" {
		t.Fatalf("old fixture version = %q", got)
	}
	return old
}

func TestUnixInstallerPreservesOldVersionAfterInterruptedDownload(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	f.interrupt[installerFixtureVersion] = true
	e := newUnixEnvironment(t, f)
	old := installRecognizableOldBinary(t, e.installDir, e.binary)
	requireFailure(t, runUnixInstaller(t, e, "--version", installerFixtureVersion, "--no-modify-path"), "download")
	got, err := os.ReadFile(e.binary)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, old) {
		t.Fatal("interrupted download replaced existing binary")
	}
}

func TestUnixInstallerPreservesUnknownTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	old := installOldBinary(t, e)
	requireFailure(t, runUnixInstaller(t, e, "--version", "unknown", "--no-modify-path"), "version")
	got, err := os.ReadFile(e.binary)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, old) {
		t.Fatal("unknown target replaced existing binary")
	}
	if requests := f.requests(); requests != 0 {
		t.Fatalf("unknown target made %d network requests", requests)
	}
}

func TestUnixInstallerRejectsUnsupportedPlatformBeforeDownload(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	e.env = append(e.env, "AI_HISTORY_TEST_OS=plan9", "AI_HISTORY_TEST_ARCH=amd64")
	requireFailure(t, runUnixInstaller(t, e, "--version", installerFixtureVersion, "--no-modify-path"), "unsupported")
	if got := f.requests(); got != 0 {
		t.Fatalf("unsupported platform made %d network requests", got)
	}
}

func TestInstallerArtifactMappingMatchesGoReleaser(t *testing.T) {
	config := readFile(t, ".goreleaser.yaml")
	for _, value := range []string{"linux", "darwin", "windows", "amd64", "arm64", "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}", "goos: windows", "formats:\n          - zip"} {
		if !strings.Contains(config, value) {
			t.Fatalf(".goreleaser.yaml missing mapping token %q", value)
		}
	}
	for _, target := range []struct{ goos, goarch, want string }{
		{"linux", "amd64", "ai-history_1.2.3_linux_amd64.tar.gz"}, {"linux", "arm64", "ai-history_1.2.3_linux_arm64.tar.gz"},
		{"darwin", "amd64", "ai-history_1.2.3_darwin_amd64.tar.gz"}, {"darwin", "arm64", "ai-history_1.2.3_darwin_arm64.tar.gz"},
		{"windows", "amd64", "ai-history_1.2.3_windows_amd64.zip"}, {"windows", "arm64", "ai-history_1.2.3_windows_arm64.zip"},
	} {
		if got := fixtureArchiveName(installerFixtureVersion, target.goos, target.goarch); got != target.want {
			t.Errorf("mapping %s/%s = %q, want %q", target.goos, target.goarch, got, target.want)
		}
	}
}

func TestUnixInstallerUpdatesPathOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	requireSuccess(t, runUnixInstaller(t, e, "--version", installerFixtureVersion))
	requireSuccess(t, runUnixInstaller(t, e, "--version", installerFixtureVersion))
	profile, err := os.ReadFile(filepath.Join(e.home, ".profile"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(profile), e.installDir); got != 1 {
		t.Fatalf("install dir occurs %d times in profile:\n%s", got, profile)
	}
	if got := strings.Count(string(profile), "# ai-history installer"); got != 1 {
		t.Fatalf("PATH marker occurs %d times in profile:\n%s", got, profile)
	}
}

func TestUnixInstallerHonorsNoModifyPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	requireSuccess(t, runUnixInstaller(t, e, "--version", installerFixtureVersion, "--no-modify-path"))
	if data, err := os.ReadFile(filepath.Join(e.home, ".profile")); err == nil && strings.Contains(string(data), e.installDir) {
		t.Fatal("--no-modify-path wrote install dir to profile")
	}
}

func TestUnixInstallerWarnsAboutAnotherPathBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	other := filepath.Join(e.home, "other-bin")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "ai-history"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	e.env = append(e.env, "PATH="+other+string(os.PathListSeparator)+e.fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	result := runUnixInstaller(t, e, "--version", installerFixtureVersion, "--no-modify-path")
	requireSuccess(t, result)
	if !strings.Contains(strings.ToLower(result.output), "warning") || !strings.Contains(result.output, other) {
		t.Fatalf("missing PATH conflict warning:\n%s", result.output)
	}
}

func TestUnixInstallerStopsBeforeSkillWhenDoctorIsInvalid(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	f.replaceVersionBinary(installerFixtureVersion, false)
	e := newUnixEnvironment(t, f)
	requireFailure(t, runUnixInstaller(t, e, "--version", installerFixtureVersion, "--no-modify-path", "--with-skill", "--agent", "cursor"), "doctor")
	if data, _ := os.ReadFile(e.npxLog); len(data) != 0 {
		t.Fatalf("npx ran after invalid doctor: %s", data)
	}
}

func TestUnixInstallerInstallsTaggedSkillForExplicitAgents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	result := runUnixInstaller(t, e, "--version", installerFixtureVersion, "--no-modify-path", "--with-skill", "--agent", "codex", "--agent", "claude-code", "--agent", "cursor")
	requireSuccess(t, result)
	log := readOptional(t, e.npxLog)
	wantSource := "https://github.com/yangkushu/ai-session-history/tree/v1.2.3/skills/ai-history"
	for _, agent := range []string{"codex", "claude-code", "cursor"} {
		if strings.Count(log, "--agent "+agent) != 1 {
			t.Errorf("agent %s invocation count != 1:\n%s", agent, log)
		}
	}
	if strings.Count(log, wantSource) != 3 {
		t.Fatalf("tagged source count != 3:\n%s", log)
	}
}

func TestUnixInstallerDetectsInstalledAgents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	for _, dir := range []string{".codex", ".claude", ".cursor"} {
		if err := os.MkdirAll(filepath.Join(e.home, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	requireSuccess(t, runUnixInstaller(t, e, "--version", installerFixtureVersion, "--no-modify-path", "--with-skill"))
	log := readOptional(t, e.npxLog)
	for _, agent := range []string{"codex", "claude-code", "cursor"} {
		if strings.Count(log, "--agent "+agent) != 1 {
			t.Errorf("auto-detected %s count != 1:\n%s", agent, log)
		}
	}
}

func TestUnixInstallerRequiresAgentWhenNoneDetected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	if err := os.RemoveAll(filepath.Join(e.home, ".cursor")); err != nil {
		t.Fatal(err)
	}
	requireFailure(t, runUnixInstaller(t, e, "--version", installerFixtureVersion, "--no-modify-path", "--with-skill"), "agent")
	if _, err := os.Stat(e.binary); err != nil {
		t.Fatalf("binary should remain installed: %v", err)
	}
}

func TestUnixInstallerKeepsBinaryWhenNpxIsMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	if err := os.Remove(filepath.Join(e.fakeBin, "npx")); err != nil {
		t.Fatal(err)
	}
	toolBin := filepath.Join(e.home, "tools-without-npx")
	if err := os.MkdirAll(toolBin, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"awk", "basename", "chmod", "curl", "dirname", "grep", "gzip", "head", "mkdir", "mktemp", "mv", "rm", "sed", "sha256sum", "tar", "tr", "uname"} {
		path, err := exec.LookPath(tool)
		if err != nil {
			t.Fatalf("required fixture tool %s: %v", tool, err)
		}
		if err := os.Symlink(path, filepath.Join(toolBin, tool)); err != nil {
			t.Fatal(err)
		}
	}
	e.env = append(e.env, "PATH="+toolBin)
	requireFailure(t, runUnixInstaller(t, e, "--version", installerFixtureVersion, "--no-modify-path", "--with-skill", "--agent", "cursor"), "npx")
	if got := binaryVersion(t, e.binary); got != "ai-history "+installerFixtureVersion {
		t.Fatalf("binary not preserved: %q", got)
	}
}

func TestUnixInstallerReportsPartialAgentFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix installer test")
	}
	f := newReleaseFixture(t, []string{installerFixtureVersion}, true)
	e := newUnixEnvironment(t, f)
	writeFakeNpx(t, e.fakeBin, true)
	result := runUnixInstaller(t, e, "--version", installerFixtureVersion, "--no-modify-path", "--with-skill", "--agent", "codex", "--agent", "cursor")
	requireFailure(t, result, "cursor")
	if !strings.Contains(strings.ToLower(result.output), "partial") {
		t.Fatalf("missing partial failure report:\n%s", result.output)
	}
	if got := binaryVersion(t, e.binary); got != "ai-history "+installerFixtureVersion {
		t.Fatalf("binary not preserved: %q", got)
	}
}

func readOptional(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
