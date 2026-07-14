# CLI and Skill Installer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add safe, repeatable one-command installers for the `ai-history` binary alone or the binary plus its Agent Skill on macOS, Linux, and Windows.

**Architecture:** Keep `scripts/install.sh` and `scripts/install.ps1` self-contained so GitHub can serve them directly. Both resolve a release, select the GoReleaser artifact, verify `checksums.txt`, install atomically into a user directory, manage user PATH idempotently, and optionally delegate tagged Skill installation to `npx skills`; native Go integration tests serve local release fixtures and isolate HOME.

**Tech Stack:** POSIX shell, PowerShell 5.1+, Go 1.22 test harness, GitHub Releases, GoReleaser, Vercel `skills` CLI, GitHub Actions.

---

## File map

- Create `scripts/install.sh`: macOS/Linux binary and bundle installer.
- Create `scripts/install.ps1`: Windows binary and bundle installer.
- Create `installer_test.go`: release fixture, Unix integration tests, and README contract tests.
- Create `installer_windows_test.go`: Windows-only PowerShell integration tests.
- Create `docs/installation.md`: full install, update, security, recovery, and uninstall guide.
- Modify `.github/workflows/ci.yaml`: native installer matrix.
- Modify `ci_config_test.go`: assert that matrix remains present.
- Modify `README.md` and `README.zh-CN.md`: prominent install paths and Skill roles.

The two installers stay standalone because a piped script cannot rely on sibling libraries. The Go contract tests keep their observable behavior aligned.

### Task 1: Lock the installer contract with failing integration tests

**Files:**
- Create: `installer_test.go`
- Create: `installer_windows_test.go`

- [ ] **Step 1: Create the local release fixture**

Use `package release_test`, matching the existing root tests. In `installer_test.go`, define `const installerFixtureVersion = "v1.2.3"` and these helpers:

```go
type releaseFixture struct {
    server *httptest.Server
    archiveName string
    archiveHits atomic.Int32
}

func runCommand(t *testing.T, env []string, name string, args ...string) (string, error)
func buildFixtureBinary(t *testing.T, dir, version string) string
func newReleaseFixture(t *testing.T, versions []string, checksumValid bool) *releaseFixture
func writeTarGz(t *testing.T, archivePath, binaryPath string)
func writeZip(t *testing.T, archivePath, binaryPath string)
```

`buildFixtureBinary` must embed its `version` argument so the fixture prints `ai-history v1.2.3` when called with `v1.2.3`, and it must print `[]` for `doctor --json`. For the baseline `v1.2.3`, `newReleaseFixture` packages it for the current runner and serves the matching route from this complete matrix, plus the checksum and latest endpoints:

```text
/v1.2.3/ai-history_1.2.3_linux_amd64.tar.gz
/v1.2.3/ai-history_1.2.3_linux_arm64.tar.gz
/v1.2.3/ai-history_1.2.3_darwin_amd64.tar.gz
/v1.2.3/ai-history_1.2.3_darwin_arm64.tar.gz
/v1.2.3/ai-history_1.2.3_windows_amd64.zip
/v1.2.3/ai-history_1.2.3_windows_arm64.zip
/v1.2.3/checksums.txt
/latest
```

Select the runner's actual OS/architecture filename. The valid checksum response is `hex-sha256`, two spaces, then the archive filename; the invalid fixture returns 64 zeroes. `/latest` returns `{"tag_name":"v1.2.3"}`. Count archive requests with `archiveHits`. A multi-version fixture creates the same routes and version-reporting binary for every supplied version.

- [ ] **Step 2: Add failing Unix behavior tests**

Add these tests and skip them on Windows:

```go
func TestUnixInstallerInstallsAndReruns(t *testing.T)
func TestUnixInstallerUpgradesAndExplicitlyDowngrades(t *testing.T)
func TestUnixInstallerResolvesLatest(t *testing.T)
func TestUnixInstallerReportsLatestReleaseFailure(t *testing.T)
func TestUnixInstallerRejectsBadChecksum(t *testing.T)
func TestUnixInstallerPreservesOldVersionAfterInterruptedDownload(t *testing.T)
func TestUnixInstallerPreservesUnknownTarget(t *testing.T)
func TestUnixInstallerRejectsUnsupportedPlatformBeforeDownload(t *testing.T)
func TestInstallerArtifactMappingMatchesGoReleaser(t *testing.T)
func TestUnixInstallerUpdatesPathOnce(t *testing.T)
func TestUnixInstallerHonorsNoModifyPath(t *testing.T)
func TestUnixInstallerWarnsAboutAnotherPathBinary(t *testing.T)
func TestUnixInstallerStopsBeforeSkillWhenDoctorIsInvalid(t *testing.T)
func TestUnixInstallerInstallsTaggedSkillForExplicitAgents(t *testing.T)
func TestUnixInstallerDetectsInstalledAgents(t *testing.T)
func TestUnixInstallerRequiresAgentWhenNoneDetected(t *testing.T)
func TestUnixInstallerKeepsBinaryWhenNpxIsMissing(t *testing.T)
func TestUnixInstallerReportsPartialAgentFailure(t *testing.T)
```

Each explicit-version test runs `sh scripts/install.sh --version v1.2.3 --no-modify-path` with isolated `HOME`, `AI_HISTORY_INSTALL_DIR`, and `AI_HISTORY_RELEASE_BASE_URL=fixture.server.URL`. Latest-version tests also set `AI_HISTORY_LATEST_RELEASE_URL=fixture.server.URL + "/latest"`. The idempotency test runs twice, requires `already installed`, and asserts `archiveHits` did not increase. Upgrade/downgrade installs `v1.2.2`, then `v1.2.3`, then explicitly returns to `v1.2.2` and checks output at every stage. The checksum, interrupted-download, and conflict tests assert the target is absent or retains the earlier bytes/version. The platform test sets `AI_HISTORY_TEST_OS=plan9` and requires zero archive hits.

For Skill tests, prepend a fake executable named `npx` to PATH. It appends arguments to `AI_HISTORY_NPX_LOG`; configure it to fail only for `cursor` in the partial-failure case. Assert the log contains one invocation per requested Agent and this exact source:

```text
https://github.com/yangkushu/ai-session-history/tree/v1.2.3/skills/ai-history
```

- [ ] **Step 3: Add failing Windows behavior tests**

Create `installer_windows_test.go` with `//go:build windows` and the equivalent tests named `TestPowerShellInstaller...`. Invoke:

```go
args := []string{
    "-NoProfile", "-ExecutionPolicy", "Bypass",
    "-File", `scripts\install.ps1`,
    "-Version", installerFixtureVersion,
    "-NoModifyPath",
}
output, err := runCommand(t, env, "powershell", args...)
```

Cover install/rerun, upgrade/explicit downgrade, latest resolution/error, invalid checksum, interrupted download, unknown target, user PATH idempotency, conflicting PATH binary, invalid `doctor` output before Skill, explicit and auto-detected Agents, no detected Agent, missing `npx`, and one-Agent failure. Use `USERPROFILE` and `AI_HISTORY_INSTALL_DIR` under `t.TempDir()`. For the PATH test, set `AI_HISTORY_TEST_USER_PATH_FILE` to a file under `t.TempDir()`; the installer must use this file instead of the Windows user environment registry so tests never mutate the developer's real PATH.

- [ ] **Step 4: Verify the tests fail for the expected reason**

Run on Unix:

```bash
go test ./... -run 'TestUnixInstaller' -count=1
```

Expected: FAIL because `scripts/install.sh` does not exist.

Run on Windows:

```powershell
go test ./... -run 'TestPowerShellInstaller' -count=1
```

Expected: FAIL because `scripts\install.ps1` does not exist.

- [ ] **Step 5: Commit the red tests**

```bash
git add installer_test.go installer_windows_test.go
git commit -m "测试：定义跨平台安装器行为"
```

### Task 2: Implement the macOS/Linux binary installer

**Files:**
- Create: `scripts/install.sh`
- Test: `installer_test.go`

- [ ] **Step 1: Implement the public options and platform mapping**

Use this public state and option contract:

```sh
#!/bin/sh
set -eu
REPOSITORY="yangkushu/ai-session-history"
VERSION="${AI_HISTORY_VERSION:-}"
INSTALL_DIR="${AI_HISTORY_INSTALL_DIR:-$HOME/.local/bin}"
RELEASE_BASE_URL="${AI_HISTORY_RELEASE_BASE_URL:-https://github.com/$REPOSITORY/releases/download}"
MODIFY_PATH=1
WITH_SKILL=0
AGENTS=""
```

Parse `--version`, `--install-dir`, `--no-modify-path`, `--with-skill`, repeated `--agent`, and `--help`; reject unknown or value-less options with exit code 2. Map `Linux` to `linux`, `Darwin` to `darwin`, `x86_64` to `amd64`, and `arm64`/`aarch64` to `arm64`. `AI_HISTORY_TEST_OS` and `AI_HISTORY_TEST_ARCH` override detection only for tests. Reject all other values before network access.

- [ ] **Step 2: Resolve and validate the release**

When no version is supplied, call `${AI_HISTORY_LATEST_RELEASE_URL:-https://api.github.com/repos/yangkushu/ai-session-history/releases/latest}` and extract `tag_name` without requiring `jq`. A non-2xx response, missing tag, or rate-limit response must name the failed URL and suggest `--version`. Require a `v` followed by three dot-separated non-negative integers. Strip `v` only in the archive name:

```sh
ARCHIVE_VERSION=${VERSION#v}
ARCHIVE_NAME="ai-history_${ARCHIVE_VERSION}_${OS}_${ARCH}.tar.gz"
ARCHIVE_URL="$RELEASE_BASE_URL/$VERSION/$ARCHIVE_NAME"
CHECKSUMS_URL="$RELEASE_BASE_URL/$VERSION/checksums.txt"
```

- [ ] **Step 3: Implement safe download and atomic installation**

If `$INSTALL_DIR/ai-history version` identifies the exact target version, print `already installed` and skip archive download. If an existing target cannot identify itself with an `ai-history vX.Y.Z` prefix, refuse to overwrite it.

Download into `mktemp -d`, trap cleanup, and require `curl`, `tar`, plus either `sha256sum` or `shasum`. Select exactly one checksum line whose filename equals `$ARCHIVE_NAME`; reject missing, duplicate, or mismatched entries. Extract and require the staged binary to report the target version. Copy to a sibling `$INSTALL_DIR/.ai-history.new.$$`, set mode `0755`, then `mv -f` it to `$INSTALL_DIR/ai-history`.

- [ ] **Step 4: Implement PATH and smoke checks**

When the install directory is absent from PATH and modification is enabled, add one marked line:

```text
bash ~/.bashrc: export PATH="$INSTALL_DIR:$PATH" # ai-history installer
zsh ~/.zshrc: export PATH="$INSTALL_DIR:$PATH" # ai-history installer
fish ~/.config/fish/config.fish: fish_add_path "$INSTALL_DIR" # ai-history installer
```

Render `$INSTALL_DIR` as the shell-quoted actual directory rather than writing the variable name literally. Never duplicate the marker. For unknown shells print, but do not write, an exact PATH instruction. After installation, compare `command -v ai-history` with the managed target; warn with both paths when another installation wins. Run the target by absolute path with `version`, then capture `doctor --json`. Source-unavailable diagnostics are a valid JSON array and succeed; a nonzero exit or output that is not a JSON array returns nonzero, retains the installed binary, and prevents the Skill stage.

- [ ] **Step 5: Run the binary tests**

Run:

```bash
sh -n scripts/install.sh
go test ./... -run 'TestUnixInstaller' -count=1
```

Expected: binary, checksum, conflict, platform, and PATH cases PASS; Skill cases remain failing until Task 3.

- [ ] **Step 6: Commit the Unix binary installer**

```bash
git add scripts/install.sh installer_test.go
git commit -m "功能：添加 Unix 二进制安装器"
```

### Task 3: Implement Unix Skill bundle orchestration

**Files:**
- Modify: `scripts/install.sh`
- Test: `installer_test.go`

- [ ] **Step 1: Add explicit validation and default detection**

Accept only `codex`, `claude-code`, and `cursor`. With no `--agent`, detect targets as follows:

```sh
command -v codex || test -d "$HOME/.codex"
command -v claude || test -d "$HOME/.claude"
command -v cursor || test -d "$HOME/.cursor"
```

If no Agent is detected, fail only the Skill stage and print examples for all three explicit values. Tests create only the relevant home directories, so detection never depends on the developer's real PATH or home.

- [ ] **Step 2: Install each Agent independently**

After binary success, require `npx` and run once per Agent:

```sh
SKILL_SOURCE="https://github.com/$REPOSITORY/tree/$VERSION/skills/ai-history"
npx --yes skills add "$SKILL_SOURCE" --skill ai-history --global --agent "$agent" --yes
```

Continue after a single Agent failure, print success/failure per Agent, and return nonzero if any failed. Missing `npx` also returns nonzero while retaining the installed binary. A same-version bundle rerun skips the archive but executes the Skill stage again.

- [ ] **Step 3: Run and commit bundle behavior**

Run:

```bash
sh -n scripts/install.sh
go test ./... -run 'TestUnixInstaller' -count=1
```

Expected: all Unix installer tests PASS.

```bash
git add scripts/install.sh installer_test.go
git commit -m "功能：支持一键安装二进制与 Skill"
```

### Task 4: Implement the Windows PowerShell installer

**Files:**
- Create: `scripts/install.ps1`
- Test: `installer_windows_test.go`

- [ ] **Step 1: Implement the public parameter block**

Start with:

```powershell
param(
    [string]$Version = $env:AI_HISTORY_VERSION,
    [string]$InstallDir = $env:AI_HISTORY_INSTALL_DIR,
    [switch]$NoModifyPath,
    [switch]$WithSkill,
    [ValidateSet('codex', 'claude-code', 'cursor')]
    [string[]]$Agent
)
$ErrorActionPreference = 'Stop'
$Repository = 'yangkushu/ai-session-history'
if (-not $InstallDir) { $InstallDir = Join-Path $env:LOCALAPPDATA 'ai-history\bin' }
$ReleaseBaseUrl = if ($env:AI_HISTORY_RELEASE_BASE_URL) { $env:AI_HISTORY_RELEASE_BASE_URL.TrimEnd('/') } else { "https://github.com/$Repository/releases/download" }
```

Resolve latest via `$env:AI_HISTORY_LATEST_RELEASE_URL` or the GitHub Releases API when needed. Map `AMD64`/`x86_64` to `amd64` and `ARM64`/`aarch64` to `arm64`; reject other platforms and architectures. After assigning `$ArchiveVersion = $Version.TrimStart('v')`, build `ai-history_${ArchiveVersion}_windows_${Architecture}.zip`.

- [ ] **Step 2: Implement integrity, identity, and atomic replacement**

Use `Invoke-WebRequest`, `Expand-Archive`, and `Get-FileHash -Algorithm SHA256`. Require exactly one checksum entry for the archive and verify the staged executable reports the target version. Refuse an existing unrecognized target. Replace through `.ai-history.new.$PID.exe`; retain a recognized old target as a sibling `.ai-history.backup.$PID.exe` until the new executable passes `version`, restoring it on replacement failure. Clean temporary and backup files in `finally`.

- [ ] **Step 3: Implement user PATH and bundle behavior**

Unless `-NoModifyPath`, read and update `[Environment]::GetEnvironmentVariable('Path', 'User')`; compare entries case-insensitively and write only the user target. When `AI_HISTORY_TEST_USER_PATH_FILE` is set, read and write that file instead of the registry; do not document this test seam as a public option. Capture `doctor --json` and require a successful JSON array before entering the Skill stage, while retaining an installed binary on diagnostic failure. Detect the same three Agents through `Get-Command` or home directories. Invoke tag-pinned `npx --yes skills add` independently per Agent and use the same partial-success exit semantics as Unix.

- [ ] **Step 4: Run and commit Windows behavior**

Run on Windows:

```powershell
$null = [scriptblock]::Create((Get-Content scripts\install.ps1 -Raw))
go test ./... -run 'TestPowerShellInstaller' -count=1
```

Expected: all PowerShell installer tests PASS.

```bash
git add scripts/install.ps1 installer_windows_test.go
git commit -m "功能：添加 Windows 一键安装器"
```

### Task 5: Add native installer CI coverage

**Files:**
- Modify: `.github/workflows/ci.yaml`
- Modify: `ci_config_test.go`

- [ ] **Step 1: Add a failing CI contract test**

Add `TestCIExercisesNativeInstallers` and require these strings in `.github/workflows/ci.yaml`:

```go
wants := []string{
    "installer-test:",
    "ubuntu-latest",
    "macos-latest",
    "windows-latest",
    "go test ./... -run TestUnixInstaller",
    "go test ./... -run TestPowerShellInstaller",
}
```

Run `go test ./... -run TestCIExercisesNativeInstallers`; expect FAIL with missing `installer-test:`.

- [ ] **Step 2: Add the native matrix**

Add:

```yaml
  installer-test:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true
      - if: runner.os != 'Windows'
        run: go test ./... -run TestUnixInstaller -count=1
      - if: runner.os == 'Windows'
        run: go test ./... -run TestPowerShellInstaller -count=1
```

- [ ] **Step 3: Verify and commit CI**

Run `go test ./... -run 'TestCIExercisesNativeInstallers|TestCIConfig' -count=1`, then `go test ./...`; expect PASS.

```bash
git add .github/workflows/ci.yaml ci_config_test.go
git commit -m "持续集成：验证跨平台安装器"
```

### Task 6: Rewrite installation and Skill usage documentation

**Files:**
- Create: `docs/installation.md`
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `installer_test.go`

- [ ] **Step 1: Add a failing README contract test**

Add `TestReadmesDocumentInstallerAndSkillRoles`. For `README.md`, require `scripts/install.sh | sh`, `--with-skill`, `scripts/install.ps1`, `Who uses the Skill`, `$ai-history`, and `docs/installation.md`. For `README.zh-CN.md`, require the same command tokens plus `谁使用 Skill`. Run the test and expect failure before editing docs.

- [ ] **Step 2: Write `docs/installation.md`**

Use this exact order: binary-only commands; binary + Skill commands; version/install-dir/PATH options; repeat-to-update behavior; Agent detection and explicit targets; Releases/source/manual fallbacks; remote-script review and checksum trust boundary; partial-failure recovery; PATH conflicts; verification; uninstall.

Executable examples must use only these canonical URLs:

```text
https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.sh
https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.ps1
```

State that binary-only reruns update only binary, bundle reruns refresh binary and selected Skills, and Skill installation grants no history permission.

- [ ] **Step 3: Reorganize both README files**

Below the summary add links to CLI install, bundle install, Skill roles, and Quick Start. Replace the long current install sections with concise Unix and PowerShell command blocks and link to `docs/installation.md`.

The role section must say: the user chooses targets and runtime authorization; Codex/Claude Code/Cursor reads the Skill and selects CLI commands; the binary performs discovery/search/show/context/export; direct CLI use remains supported. Include `$ai-history Find earlier discussions about the release process in this project.` in English and `$ai-history 查找这个项目之前关于发布流程的讨论。` in Chinese. For Claude Code and Cursor, refer to the invocation displayed by the current host UI rather than inventing a slash command.

- [ ] **Step 4: Verify and commit documentation**

Run:

```bash
go test ./... -run TestReadmesDocumentInstallerAndSkillRoles -count=1
git diff --check
```

Expected: PASS.

```bash
git add README.md README.zh-CN.md docs/installation.md installer_test.go
git commit -m "文档：补充一键安装与 Skill 使用说明"
```

### Task 7: Perform release-shaped verification

**Files:**
- Modify only when a verification step exposes an earlier defect.

- [ ] **Step 1: Run static and repository checks**

```bash
sh -n scripts/install.sh
go test ./...
go vet ./...
goreleaser check
git diff --check
```

Expected: every command exits 0.

- [ ] **Step 2: Verify snapshot artifact mapping**

Run `goreleaser release --snapshot --clean` and `find dist -maxdepth 2 -type f | sort`. Expect Linux/Darwin tarballs and Windows zip files for `amd64`/`arm64`, plus `checksums.txt`, with names matching both installers.

- [ ] **Step 3: Smoke-test a published binary in isolation**

```bash
tmp_dir=$(mktemp -d)
AI_HISTORY_INSTALL_DIR="$tmp_dir/bin" sh scripts/install.sh --version v0.4.0 --no-modify-path
"$tmp_dir/bin/ai-history" version
```

Expected: checksum passes and output contains `ai-history v0.4.0`. Remove the temporary directory afterward.

- [ ] **Step 4: Smoke-test a real Skill install in an isolated HOME**

```bash
tmp_home=$(mktemp -d)
HOME="$tmp_home" AI_HISTORY_INSTALL_DIR="$tmp_home/bin" sh scripts/install.sh --version v0.4.0 --no-modify-path --with-skill --agent codex
find "$tmp_home" -maxdepth 5 \( -type f -o -type l \) -print | sort
```

Expected: binary `v0.4.0` and a global `ai-history` Skill under the temporary HOME; no real Agent configuration changes.

- [ ] **Step 5: Confirm clean completion**

Run `git status --short` and `git log -8 --oneline`. Expect no uncommitted files and one focused commit per implementation stage. If a verification fix was necessary, rerun its affected checks and commit it with a short Chinese message.
