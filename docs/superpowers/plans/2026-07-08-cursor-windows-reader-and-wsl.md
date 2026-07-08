# Cursor Windows Reader 与 WSL 发现 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `ai-history` 实现真实的 Cursor Windows latest 读取器，并让 WSL 主机默认发现 Windows 侧 Cursor 数据，最后在 WSL 上验证整条 CLI 链路。

**Architecture:** discovery 层新增可注入测试的 WSL 探测（`/proc/version` + `/mnt` glob），保留 `DefaultPaths` 纯函数；reader 用 `modernc.org/sqlite` 以 `immutable=1` 打开 `globalStorage/state.vscdb`，从 `composerHeaders` 列会话、从 `cursorDiskKV` 的 `bubbleId:` key 读消息 turn；fixture 在测试内临时生成（沿用 `writeCodexState` 模式），内容全合成。

**Tech Stack:** Go 1.22+（本机 1.26.1）、`modernc.org/sqlite`（纯 Go，无 CGO）、`database/sql`、标准库 `flag`、`encoding/json`。

**提交约定：** 每个 Task 末尾本地 `git commit`（中文 message，无 Co-Authored-By，注意换行符）；`git push` 在最后统一询问用户。

**关键事实（来自真实样本探查）：**
- Windows Cursor 数据在 `<root>/globalStorage/state.vscdb`（`<root>` = `…/Cursor/User`）。
- `composerHeaders` 表：`composerId`(PK) + `value`(JSON)，JSON 含 `name`、`createdAt`(float ms)、`lastUpdatedAt`(float ms)、`isArchived`(bool)、`workspaceIdentifier.uri.path`(cwd)。
- `cursorDiskKV` 表：`key`(UNIQUE) + `value`(BLOB)。消息在 `bubbleId:<composerId>:<bubbleId>`，value JSON 含 `type`(1=user,2=assistant)、`text`、`createdAt`(ISO string)。
- 直读 live DB：`immutable=1` 成功；普通/`mode=ro` 在 WSL+DrvFs 上报 `disk I/O error`。
- 3339 条无 `text` 的 assistant bubble 是内部状态（内容在加密 `conversationState`/`agentKv`），P0 不解析。

---

## File Structure

| 文件 | 责任 | 动作 |
|------|------|------|
| `internal/discovery/discovery.go` | 路径发现：纯 `DefaultPaths` + WSL 探测 + `ResolveRoots` | 修改（追加） |
| `internal/discovery/discovery_test.go` | discovery 单测 | 修改（追加用例） |
| `internal/readers/cursor.go` | Cursor reader：Doctor/ListSessions/GetSession | 重写 |
| `internal/readers/cursor_test.go` | reader 单测 + `writeCursorState` fixture helper | 重写 |
| `internal/cli/service.go` | service 装配：cursor 用 `ResolveRoots` | 修改（1 行） |
| `testdata/cursor/windows/README.md` | 真实存储形状留档 | 修改 |
| `README.md` | Cursor 支持状态 | 修改 |
| `docs/superpowers/status/2026-07-08-cursor-windows-reader.md` | 本次工作状态（中文） | 新建 |

---

## Task 1: WSL 探测辅助函数（discovery，TDD）

**Files:**
- Modify: `internal/discovery/discovery.go`
- Test: `internal/discovery/discovery_test.go`

- [ ] **Step 1: 写失败测试**

追加到 `internal/discovery/discovery_test.go`（顶部 import 补 `"reflect"`、`"sort"` 如缺）：

```go
func TestIsWSLDetectsMicrosoftKernel(t *testing.T) {
	cases := map[string]bool{
		"Linux version 6.6.87.2-microsoft-standard-WSL2 (user@host)": true,
		"Linux version 5.15.153.1-microsoft-standard-WSL2":           true,
		"Linux version 5.15.0-91-generic (user@host)":                 false,
		"":                                                             false,
	}
	for content, want := range cases {
		if got := isWSL(content); got != want {
			t.Errorf("isWSL(%q) = %v, want %v", content, got, want)
		}
	}
}

func TestWindowsCursorRootsUnderGlobsExistingUserDirs(t *testing.T) {
	mount := t.TempDir()
	alice := filepath.Join(mount, "c", "Users", "alice", "AppData", "Roaming", "Cursor", "User")
	bob := filepath.Join(mount, "c", "Users", "bob", "AppData", "Roaming", "Cursor", "User")
	for _, p := range []string{alice, bob} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(mount, "c", "Users", "carol", "AppData", "Roaming", "Other"), 0o700); err != nil {
		t.Fatal(err)
	}

	roots := windowsCursorRootsUnder(mount)
	sort.Strings(roots)
	want := []string{alice, bob}
	sort.Strings(want)
	if !reflect.DeepEqual(roots, want) {
		t.Fatalf("got %v, want %v", roots, want)
	}
}

func TestCursorRootsForAddsWSLWindowsRootsOnLinuxWSL(t *testing.T) {
	mount := t.TempDir()
	win := filepath.Join(mount, "c", "Users", "alice", "AppData", "Roaming", "Cursor", "User")
	if err := os.MkdirAll(win, 0o700); err != nil {
		t.Fatal(err)
	}

	roots := cursorRootsFor("linux", "/home/alice", nil,
		"Linux version 6.6.87.2-microsoft-standard-WSL2", mount)

	if !contains(roots, win) {
		t.Fatalf("WSL Windows root not discovered: %v", roots)
	}
	native := filepath.Join("/home/alice", ".config", "Cursor", "User")
	if !contains(roots, native) {
		t.Fatalf("native linux fallback dropped: %v", roots)
	}
}

func TestCursorRootsForSkipsWSLGlobOnPlainLinux(t *testing.T) {
	mount := t.TempDir()
	win := filepath.Join(mount, "c", "Users", "alice", "AppData", "Roaming", "Cursor", "User")
	if err := os.MkdirAll(win, 0o700); err != nil {
		t.Fatal(err)
	}
	roots := cursorRootsFor("linux", "/home/alice", nil,
		"Linux version 5.15.0-91-generic", mount)
	if contains(roots, win) {
		t.Fatalf("plain linux must not glob /mnt: %v", roots)
	}
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `GOCACHE=/tmp/go-build go test ./internal/discovery/ -run 'TestIsWSL|TestWindowsCursorRoots|TestCursorRootsFor' -v`
Expected: 编译失败，`undefined: isWSL` 等。

- [ ] **Step 3: 写最小实现**

在 `internal/discovery/discovery.go` 顶部 import 加 `"strings"`（已有 `os`/`path/filepath`/`runtime`/`core`）。文件末尾追加：

```go
// isWSL reports whether the given /proc/version content indicates a WSL kernel.
func isWSL(procVersion string) bool {
	return strings.Contains(strings.ToLower(procVersion), "microsoft")
}

// windowsCursorRootsUnder globs a mount tree for Windows Cursor user dirs of
// the form <mount>/<drive>/Users/<user>/AppData/Roaming/Cursor/User and returns
// the ones that exist.
func windowsCursorRootsUnder(mountRoot string) []string {
	pattern := filepath.Join(mountRoot, "*", "Users", "*", "AppData", "Roaming", "Cursor", "User")
	matches, _ := filepath.Glob(pattern)
	roots := []string{}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err == nil && info.IsDir() {
			roots = append(roots, match)
		}
	}
	return roots
}

// cursorRootsFor returns Cursor roots for an injectable environment. It is the
// testable composition of DefaultPaths plus WSL→Windows discovery.
func cursorRootsFor(goos, home string, env map[string]string, procVersion, mountRoot string) []string {
	roots := DefaultPaths(core.SourceCursor, goos, home, env)
	if goos == "linux" && isWSL(procVersion) && mountRoot != "" {
		roots = append(roots, windowsCursorRootsUnder(mountRoot)...)
	}
	return roots
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `GOCACHE=/tmp/go-build go test ./internal/discovery/ -v`
Expected: PASS（含原有用例）。

- [ ] **Step 5: 提交**

```bash
git add internal/discovery/discovery.go internal/discovery/discovery_test.go
git commit -m "添加 WSL 探测辅助函数"
```

---

## Task 2: ResolveRoots 并接入 CLI service

**Files:**
- Modify: `internal/discovery/discovery.go`
- Modify: `internal/cli/service.go:30`
- Test: `internal/discovery/discovery_test.go`（追加 ResolveRoots 不做 FS 注入测试，因其委托给已测的 cursorRootsFor；只加最小冒烟）

- [ ] **Step 1: 在 discovery.go 追加 ResolveRoots**

```go
// ResolveRoots returns the default roots for a source, including WSL→Windows
// Cursor discovery when running on a WSL host. Configured paths from the config
// file are added by the caller; this only resolves default-path roots.
func ResolveRoots(source core.Source) []string {
	if source == core.SourceCursor {
		procVersion, _ := os.ReadFile("/proc/version")
		return cursorRootsFor(runtime.GOOS, "", nil, string(procVersion), "/mnt")
	}
	return DefaultPaths(source, "", "", nil)
}
```

- [ ] **Step 2: 改 service.go 用 ResolveRoots**

`internal/cli/service.go` 第 29-31 行，把：

```go
		if sourceConfig.UseDefaultPaths {
			roots = append(roots, discovery.DefaultPaths(source, "", "", nil)...)
		}
```

改为：

```go
		if sourceConfig.UseDefaultPaths {
			roots = append(roots, discovery.ResolveRoots(source)...)
		}
```

- [ ] **Step 3: 跑全量测试确认不回归**

Run: `GOCACHE=/tmp/go-build go test ./...`
Expected: PASS（`cli`/`config`/`discovery` 等全过）。

- [ ] **Step 4: 冒烟：doctor 在 WSL 上应发现 cursor**

Run: `GOCACHE=/tmp/go-build go run ./cmd/ai-history doctor --json`
Expected: JSON 中 cursor 的 status 为 `available`（reader 仍是脚手架，但发现路径已生效；status 取决于 reader——此时 reader Doctor 仍按 Task 3 改造前逻辑；若 Task 2 在 Task 3 之前，cursor 可能仍是 unavailable/unsupported。**因此本步骤仅确认不 panic、codex/claude 仍 available**，cursor 最终可用性等 Task 3。）

- [ ] **Step 5: 提交**

```bash
git add internal/discovery/discovery.go internal/cli/service.go
git commit -m "接入 WSL Cursor 路径发现"
```

---

## Task 3: 实现 Cursor Windows reader（TDD，重写）

**Files:**
- Rewrite: `internal/readers/cursor.go`
- Rewrite: `internal/readers/cursor_test.go`

- [ ] **Step 1: 写失败测试（重写 cursor_test.go 全文）**

`internal/readers/cursor_test.go` 完整内容：

```go
package readers

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/yangkushu/ai-session-history/internal/core"
)

func TestCursorStorageReaderListsNonArchivedComposers(t *testing.T) {
	root := t.TempDir()
	writeCursorState(t, root,
		[]cursorComposerFixture{
			{ID: "comp-active", Name: "Active composer", CWD: "/example/active",
				CreatedAt: 1783382400000, UpdatedAt: 1783382402000},
			{ID: "comp-archived", Name: "Archived", Archived: true},
		}, nil)

	reader := NewCursorStorageReader([]string{root})
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("want 1 session, got %+v", sessions)
	}
	s := sessions[0]
	if s.ID != "cursor:comp-active" {
		t.Fatalf("id: %s", s.ID)
	}
	if s.Title != "Active composer" {
		t.Fatalf("title: %s", s.Title)
	}
	if s.CWD != "/example/active" {
		t.Fatalf("cwd: %s", s.CWD)
	}
	if s.Project != "active" {
		t.Fatalf("project: %s", s.Project)
	}
}

func TestCursorStorageReaderReadsMessageTurnsAndSkipsEmpty(t *testing.T) {
	root := t.TempDir()
	writeCursorState(t, root,
		[]cursorComposerFixture{
			{ID: "comp-turns", Name: "Turns", CreatedAt: 1783382400000, UpdatedAt: 1783382402000},
		},
		[]cursorBubbleFixture{
			{Composer: "comp-turns", Bubble: "b1", Type: 1, Text: "Example user goal",
				CreatedAt: "2026-07-08T00:00:01.000Z"},
			{Composer: "comp-turns", Bubble: "b2", Type: 2, Text: "",
				CreatedAt: "2026-07-08T00:00:02.000Z"},
			{Composer: "comp-turns", Bubble: "b3", Type: 2, Text: "Example assistant reply",
				CreatedAt: "2026-07-08T00:00:03.000Z"},
		})

	reader := NewCursorStorageReader([]string{root})
	detail, err := reader.GetSession("comp-turns")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(detail.Turns) != 2 {
		t.Fatalf("want 2 turns (empty skipped), got %+v", detail.Turns)
	}
	if detail.Turns[0].Role != core.RoleUser || detail.Turns[0].Text != "Example user goal" {
		t.Fatalf("turn0: %+v", detail.Turns[0])
	}
	if detail.Turns[1].Role != core.RoleAssistant || detail.Turns[1].Text != "Example assistant reply" {
		t.Fatalf("turn1: %+v", detail.Turns[1])
	}
	if detail.Summary.TurnCount != 2 {
		t.Fatalf("turn count: %d", detail.Summary.TurnCount)
	}
}

func TestCursorStorageReaderDoctorAvailable(t *testing.T) {
	root := t.TempDir()
	writeCursorState(t, root, []cursorComposerFixture{{ID: "c1"}}, nil)
	reader := NewCursorStorageReader([]string{root})
	d := reader.Doctor()
	if d.Status != "available" {
		t.Fatalf("status: %+v", d)
	}
}

func TestCursorStorageReaderDoctorUnavailableWithoutDB(t *testing.T) {
	reader := NewCursorStorageReader([]string{t.TempDir()})
	d := reader.Doctor()
	if d.Status != "unavailable" || d.Code != core.ErrSourceUnavailable {
		t.Fatalf("unexpected: %+v", d)
	}
}

func TestCursorStorageReaderGetSessionNotFound(t *testing.T) {
	root := t.TempDir()
	writeCursorState(t, root, []cursorComposerFixture{{ID: "c1"}}, nil)
	reader := NewCursorStorageReader([]string{root})
	_, err := reader.GetSession("missing")
	if !core.IsCode(err, core.ErrSessionNotFound) {
		t.Fatalf("err: %v", err)
	}
}

type cursorComposerFixture struct {
	ID, Name, CWD      string
	CreatedAt, UpdatedAt int64
	Archived           bool
}

type cursorBubbleFixture struct {
	Composer, Bubble string
	Type             int
	Text, CreatedAt  string
}

func writeCursorState(t *testing.T, root string, composers []cursorComposerFixture, bubbles []cursorBubbleFixture) {
	t.Helper()
	dbPath := filepath.Join(root, "globalStorage", "state.vscdb")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE composerHeaders (
		composerId TEXT PRIMARY KEY,
		workspaceId TEXT,
		createdAt INTEGER,
		lastUpdatedAt INTEGER,
		isArchived INTEGER,
		isSubagent INTEGER,
		recency INTEGER,
		checkpointAt INTEGER,
		value TEXT
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE cursorDiskKV (key TEXT UNIQUE ON CONFLICT REPLACE, value BLOB)`); err != nil {
		t.Fatal(err)
	}

	for _, c := range composers {
		val := map[string]interface{}{
			"composerId":    c.ID,
			"name":          c.Name,
			"createdAt":     float64(c.CreatedAt),
			"lastUpdatedAt": float64(c.UpdatedAt),
			"isArchived":    c.Archived,
		}
		if c.CWD != "" {
			val["workspaceIdentifier"] = map[string]interface{}{
				"id": c.ID + "-ws",
				"uri": map[string]interface{}{"path": c.CWD},
			}
		}
		raw, err := json.Marshal(val)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO composerHeaders (composerId, value) VALUES (?, ?)`, c.ID, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	for _, b := range bubbles {
		val := map[string]interface{}{
			"bubbleId":  b.Bubble,
			"type":      float64(b.Type),
			"text":      b.Text,
			"createdAt": b.CreatedAt,
		}
		raw, err := json.Marshal(val)
		if err != nil {
			t.Fatal(err)
		}
		key := "bubbleId:" + b.Composer + ":" + b.Bubble
		if _, err := db.Exec(`INSERT INTO cursorDiskKV (key, value) VALUES (?, ?)`, key, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `GOCACHE=/tmp/go-build go test ./internal/readers/ -run TestCursor -v`
Expected: FAIL（当前 reader 把 state.vscdb 当 unsupported_format，list/doctor 不工作）。

- [ ] **Step 3: 重写 cursor.go 实现**

`internal/readers/cursor.go` 完整内容：

```go
package readers

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yangkushu/ai-session-history/internal/core"
)

type CursorStorageReader struct {
	roots []string
}

func NewCursorStorageReader(roots []string) *CursorStorageReader {
	return &CursorStorageReader{roots: roots}
}

func (r *CursorStorageReader) Doctor() core.SourceDiagnostic {
	dbPath := r.stateDBPath()
	if dbPath == "" {
		return core.SourceDiagnostic{
			Source:  core.SourceCursor,
			Status:  "unavailable",
			Code:    core.ErrSourceUnavailable,
			Message: "no Cursor state.vscdb found",
		}
	}
	db, err := sql.Open("sqlite", cursorDBURI(dbPath))
	if err != nil {
		return core.SourceDiagnostic{
			Source: core.SourceCursor, Status: "unsupported_format",
			Code: core.ErrUnsupportedFormat, Path: dbPath,
			Message: err.Error(),
		}
	}
	defer db.Close()
	var name string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='composerHeaders'`).Scan(&name)
	if err != nil {
		return core.SourceDiagnostic{
			Source: core.SourceCursor, Status: "unsupported_format",
			Code: core.ErrUnsupportedFormat, Path: dbPath,
			Message: "composerHeaders table not found",
		}
	}
	return core.SourceDiagnostic{Source: core.SourceCursor, Status: "available", Path: dbPath}
}

func (r *CursorStorageReader) ListSessions() ([]core.SessionSummary, error) {
	db, dbPath, err := r.openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT composerId, value FROM composerHeaders`)
	if err != nil {
		return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}
	defer rows.Close()

	var sessions []core.SessionSummary
	for rows.Next() {
		var composerID string
		var raw []byte
		if err := rows.Scan(&composerID, &raw); err != nil {
			return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
		}
		var value cursorComposerValue
		if err := json.Unmarshal(raw, &value); err != nil {
			continue
		}
		if value.IsArchived {
			continue
		}
		sessions = append(sessions, cursorSummary(composerID, value))
	}
	if err := rows.Err(); err != nil {
		return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessionTime(sessions[i]).After(sessionTime(sessions[j]))
	})
	return sessions, nil
}

func (r *CursorStorageReader) GetSession(nativeID string) (core.SessionDetail, error) {
	db, dbPath, err := r.openDB()
	if err != nil {
		return core.SessionDetail{}, err
	}
	defer db.Close()

	var raw []byte
	err = db.QueryRow(`SELECT value FROM composerHeaders WHERE composerId = ?`, nativeID).Scan(&raw)
	if err == sql.ErrNoRows {
		return core.SessionDetail{}, core.NewError(core.ErrSessionNotFound, "Cursor session not found: "+nativeID)
	}
	if err != nil {
		return core.SessionDetail{}, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}

	var value cursorComposerValue
	_ = json.Unmarshal(raw, &value)

	turns, err := r.readTurns(db, dbPath, nativeID)
	if err != nil {
		return core.SessionDetail{}, err
	}
	summary := cursorSummary(nativeID, value)
	summary.TurnCount = len(turns)
	return core.SessionDetail{Summary: summary, Turns: turns}, nil
}

func (r *CursorStorageReader) readTurns(db *sql.DB, dbPath, composerID string) ([]core.Turn, error) {
	prefix := "bubbleId:" + composerID + ":"
	rows, err := db.Query(
		`SELECT value FROM cursorDiskKV WHERE key >= ? AND key < ?`,
		prefix, prefix+"\U0010FFFF",
	)
	if err != nil {
		return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}
	defer rows.Close()

	var turns []core.Turn
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
		}
		var bubble cursorBubble
		if err := json.Unmarshal(raw, &bubble); err != nil {
			continue
		}
		if strings.TrimSpace(bubble.Text) == "" {
			continue
		}
		role := cursorBubbleRole(bubble.Type)
		if role == "" {
			continue
		}
		turns = append(turns, core.Turn{
			Role:      role,
			Text:      bubble.Text,
			Timestamp: timeFromISO(bubble.CreatedAt),
			Kind:      core.KindMessage,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}
	sort.SliceStable(turns, func(i, j int) bool {
		return cursorTurnTime(turns[i]).Before(cursorTurnTime(turns[j]))
	})
	return turns, nil
}

func (r *CursorStorageReader) stateDBPath() string {
	for _, root := range r.roots {
		candidate := filepath.Join(root, "globalStorage", "state.vscdb")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func (r *CursorStorageReader) openDB() (*sql.DB, string, error) {
	dbPath := r.stateDBPath()
	if dbPath == "" {
		return nil, "", core.NewError(core.ErrSourceUnavailable, "no Cursor state.vscdb found")
	}
	db, err := sql.Open("sqlite", cursorDBURI(dbPath))
	if err != nil {
		return nil, dbPath, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}
	return db, dbPath, nil
}

func cursorDBURI(dbPath string) string {
	return "file:" + dbPath + "?immutable=1"
}

func cursorBubbleRole(t float64) core.TurnRole {
	switch t {
	case 1:
		return core.RoleUser
	case 2:
		return core.RoleAssistant
	default:
		return ""
	}
}

func cursorTurnTime(turn core.Turn) time.Time {
	if turn.Timestamp != nil {
		return *turn.Timestamp
	}
	return time.Time{}
}

func cursorSummary(composerID string, value cursorComposerValue) core.SessionSummary {
	title := value.Name
	if strings.TrimSpace(title) == "" {
		title = "Untitled Composer"
	}
	cwd := value.WorkspaceIdentifier.URI.Path
	return core.SessionSummary{
		ID:            core.MakeSessionID(core.SourceCursor, composerID),
		Source:        core.SourceCursor,
		NativeID:      composerID,
		Title:         title,
		Project:       projectFromCWD(cwd),
		CWD:           cwd,
		CreatedAt:     cursorTimeFromMillis(value.CreatedAt),
		UpdatedAt:     cursorTimeFromMillis(value.LastUpdatedAt),
		Available:     true,
		ReaderBackend: core.BackendStorage,
	}
}

func cursorTimeFromMillis(ms float64) *time.Time {
	if ms <= 0 {
		return nil
	}
	t := time.UnixMilli(int64(ms)).UTC()
	return &t
}

type cursorComposerValue struct {
	Name          string `json:"name"`
	IsArchived    bool   `json:"isArchived"`
	CreatedAt     float64 `json:"createdAt"`
	LastUpdatedAt float64 `json:"lastUpdatedAt"`
	WorkspaceIdentifier struct {
		ID  string `json:"id"`
		URI struct {
			Path string `json:"path"`
		} `json:"uri"`
	} `json:"workspaceIdentifier"`
}

type cursorBubble struct {
	Type      float64 `json:"type"`
	Text      string  `json:"text"`
	CreatedAt string  `json:"createdAt"`
}
```

> 说明：`timeFromISO`、`sessionTime`、`projectFromCWD` 复用 codex.go/reader.go 中已有同包函数。

- [ ] **Step 4: 跑 reader 测试确认通过**

Run: `GOCACHE=/tmp/go-build go test ./internal/readers/ -run TestCursor -v`
Expected: 5 个测试全 PASS。

- [ ] **Step 5: 跑全量测试**

Run: `GOCACHE=/tmp/go-build go test ./...`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add internal/readers/cursor.go internal/readers/cursor_test.go
git commit -m "实现 Cursor Windows 读取器"
```

---

## Task 4: 文档与 fixture 留档

**Files:**
- Modify: `testdata/cursor/windows/README.md`
- Modify: `README.md`
- Create: `docs/superpowers/status/2026-07-08-cursor-windows-reader.md`

- [ ] **Step 1: 重写 testdata/cursor/windows/README.md**

```markdown
# Cursor Windows Fixture

This directory documents the observed real storage shape of latest Cursor on
Windows. Tests build a synthetic database with this shape at runtime via the
`writeCursorState` helper in `internal/readers/cursor_test.go`; no real or
binary fixture is committed.

Real `globalStorage/state.vscdb` shape (Windows latest, observed 2026-07-08):

- `composerHeaders` table: one row per composer (Agent) conversation.
  - `composerId` (PRIMARY KEY), `workspaceId`, `createdAt` (ms), `lastUpdatedAt`
    (ms), `isArchived` (int), `isSubagent` (int), `recency`, `checkpointAt`,
    `value` (JSON).
  - `value` JSON fields used by the reader: `name`, `isArchived`,
    `createdAt`, `lastUpdatedAt`, `workspaceIdentifier.uri.path`.
- `cursorDiskKV` table: `key` (UNIQUE) + `value` (BLOB).
  - Conversation messages live under `bubbleId:<composerId>:<bubbleId>`.
  - Bubble `value` JSON fields used by the reader: `type` (1 = user,
    2 = assistant), `text`, `createdAt` (ISO 8601 string).
  - Bubbles without `text` carry internal state in an encrypted
    `conversationState` blob and content-addressed `agentKv:blob:` rows; P0
    does not parse them.

The reader opens the database with SQLite `immutable=1` because the file is
owned and live-updated by Cursor and, on a WSL host, sits on a Windows
filesystem mount where default read-only access fails with a disk I/O error.
```

- [ ] **Step 2: 更新 README.md 的 Source Support 段**

把 `README.md` 中：

```text
- Cursor: P0 target, but real latest macOS and Windows fixture validation is
  still pending. Until then, Cursor storage is diagnosed as unavailable or
  unsupported instead of being parsed speculatively.
```

替换为：

```text
- Cursor: Windows latest is supported, reading `globalStorage/state.vscdb`
  (`composerHeaders` + `cursorDiskKV`). Windows Cursor data is auto-discovered
  from a WSL host. macOS latest is still pending a real sample. The database is
  opened with SQLite `immutable=1` to safely read Cursor's live, WAL-mode file.
```

- [ ] **Step 3: 新建中文状态文档**

`docs/superpowers/status/2026-07-08-cursor-windows-reader.md`：

```markdown
# Cursor Windows 读取器 状态 - 2026-07-08

## 总览

本次工作补齐了 P0 的 Cursor Windows latest 读取器，并让 WSL 主机默认发现
Windows 侧 Cursor 数据。Cursor macOS 仍未验证（本机无 macOS 样本），OpenSpec
change `bootstrap-ai-session-history-cli` 保持未归档。

## 已完成

- WSL 探测：`internal/discovery` 新增 `isWSL`、`windowsCursorRootsUnder`、
  `cursorRootsFor`（可注入测试）与 `ResolveRoots`（生产入口）。WSL 下默认 glob
  `/mnt/*/Users/*/AppData/Roaming/Cursor/User`。
- Cursor reader：`internal/readers/cursor.go` 重写。从 `composerHeaders` 列出
  非 archived 的 composer；从 `cursorDiskKV` 的 `bubbleId:<id>:*` 读消息 turn
  （`type=1`→user、`type=2`→assistant，仅取有 `text` 的 bubble）；以
  `immutable=1` 打开 DB。
- Fixture：测试内 `writeCursorState` 临时生成真实表结构 + 合成中性内容，沿用
  `writeCodexState` 模式；不提交二进制或真实数据。
- 文档：`testdata/cursor/windows/README.md` 留档真实存储形状；`README.md` 更新
  Cursor 支持状态。

## P0 已知限制

- Cursor 的工具结果、diff、思考内容存在加密 `conversationState` 与内容寻址的
  `agentKv:blob:`，P0 不解析；故 cursor 的 `clean`/`summary`/`raw` 差异主要为
  尺寸边界，不产生 tool_result turn。
- `immutable=1` 跳过 WAL，极近期未 checkpoint 的数据会滞后；只读历史工具可接受。
- Cursor macOS latest 仍未用真实样本验证。

## 不做的事

- 不 archive OpenSpec change（macOS 未验证）。
- 不实现 `search`/`export`/`import`。
- 不提交任何真实 Cursor 历史内容。

## 验证（WSL2 + Windows 侧 Cursor）

- `gofmt -l .` 干净；`go vet ./...` 通过。
- `go test ./...` 全通过（含新增 discovery WSL 与 cursor reader 用例）。
- `go build ./cmd/ai-history` 成功。
- `doctor --json`：cursor 为 `available`。
- `list --source cursor --json`：列出真实 composer 会话。
- `show cursor:<id> --json`：读到真实消息 turn。
- `context cursor:<id>`：输出 Markdown 移交包。
- `openspec validate bootstrap-ai-session-history-cli --strict` 通过。
```

- [ ] **Step 4: 提交**

```bash
git add testdata/cursor/windows/README.md README.md docs/superpowers/status/2026-07-08-cursor-windows-reader.md
git commit -m "更新 Cursor 文档与 Windows 存储形状留档"
```

---

## Task 5: Win/WSL 兼容性验收

**Files:** 无修改；仅运行验证命令并记录结果。

- [ ] **Step 1: 格式与静态检查**

Run:
```bash
PATH="$PATH:/usr/local/go/bin"
gofmt -l . && echo "gofmt clean"
GOCACHE=/tmp/go-build go vet ./...
```
Expected: `gofmt clean`；vet 无输出（退出码 0）。若有文件列出，`gofmt -w` 后重跑。

- [ ] **Step 2: 全量测试**

Run: `PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go test ./...`
Expected: 全部 `ok`。

- [ ] **Step 3: 构建**

Run: `PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go build ./cmd/ai-history`
Expected: 无输出，生成 `ai-history` 二进制。

- [ ] **Step 4: doctor 真实回归**

Run: `PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go run ./cmd/ai-history doctor --json`
Expected: cursor status = `available`，path 指向 `/mnt/c/Users/<user>/AppData/Roaming/Cursor/User/globalStorage/state.vscdb`；codex/claude 仍 available。

- [ ] **Step 5: cursor list 真实回归**

Run: `PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go run ./cmd/ai-history list --source cursor --limit 5 --json`
Expected: 返回真实 composer 摘要数组（title/cwd/turn_count 合理）。确认输出**不含**任何真实隐私内容被写进仓库（此为终端输出，不提交）。

- [ ] **Step 6: cursor show + context 真实回归**

取 Step 5 输出中的一个 `id`（形如 `cursor:<composerId>`），执行：
```bash
go run ./cmd/ai-history show <id> --mode clean --json | head -40
go run ./cmd/ai-history context <id> | head -30
```
Expected: show 输出 user/assistant turn 文本；context 输出 `# AI Session Context` Markdown，含 initial goal 与 recent conversation。

- [ ] **Step 7: codex/claude 全链路回归（兼容性）**

Run:
```bash
go run ./cmd/ai-history list --source codex --limit 3 --json
go run ./cmd/ai-history list --source claude --limit 3 --json
```
Expected: 在 WSL 上正常列出 codex/claude 会话（验证已开发内容兼容当前环境）。

- [ ] **Step 8: OpenSpec 校验**

Run: `openspec validate bootstrap-ai-session-history-cli --strict`
Expected: `Change 'bootstrap-ai-session-history-cli' is valid`。

- [ ] **Step 9: 清理探查临时文件**

Run: `rm -rf /tmp/cursor-probe /tmp/cursor-snap /tmp/go-build/ai-history 2>/dev/null; ls /tmp/cursor-* 2>/dev/null || echo cleaned`
Expected: `cleaned`。

- [ ] **Step 10: 勾选 OpenSpec tasks 并询问 push**

编辑 `openspec/changes/bootstrap-ai-session-history-cli/tasks.md`：把 Task 3/4 对应的未勾选项改为 `[x]`（WSL 发现、reader 实现、fixture、Windows 测试、文档更新、`list/show/context` 对 live DB 的回归）。macOS 项保持 `[ ]`。

向用户确认是否 `git push`（不自动 push）。

---

## Self-Review

**1. Spec coverage:**
- "Cursor latest Windows reading"（list/show/archived/text-only/immutable）→ Task 3 全覆盖。
- "WSL discovery of Windows Cursor storage"（auto-detect + config respected）→ Task 1+2 覆盖；config 语义由现有 `service.go` `UseDefaultPaths` 保证，未改动。
- "Cursor latest macOS support deferred"（不 archive）→ tasks.md Section 8 + 状态文档说明。
- "Unsupported Cursor variants"（SHALL 报 unsupported_format）→ Doctor 在表缺失时返回 unsupported_format（Task 3 Step 3）。
- 兼容性验收 → Task 5。

**2. Placeholder scan:** 无 TBD/TODO；每步含完整代码或确切命令。

**3. Type consistency:** `cursorComposerValue`/`cursorBubble` 在 reader 与 fixture helper 字段名一致（`name`/`isArchived`/`createdAt`/`lastUpdatedAt`/`workspaceIdentifier.uri.path`；`type`/`text`/`createdAt`）；`cursorBubbleRole`、`cursorTimeFromMillis`、`cursorSummary`、`cursorDBURI` 定义与调用一致；`timeFromISO`/`sessionTime`/`projectFromCWD` 复用已有同包函数（已核对存在于 codex.go / reader.go）。

**风险提示：** Task 2 Step 4 的 doctor 冒烟在 Task 3 完成前 cursor 状态可能仍非 available（reader 尚未改造）；这是预期，Task 3 完成后 Task 5 Step 4 做最终确认。
