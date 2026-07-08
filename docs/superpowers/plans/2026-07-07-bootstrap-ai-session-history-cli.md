# Bootstrap AI Session History CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the P0 `ai-history` Go CLI for local AI coding session history discovery, reading, and deterministic Markdown handoff context.

**Architecture:** Implement a small Go CLI over a reusable internal core. Source readers normalize Codex, Claude Code, and Cursor storage into shared session models; the service layer handles filtering, rendering, and diagnostics; command packages only parse flags and format output.

**Tech Stack:** Go, `modernc.org/sqlite` for portable SQLite access, `gopkg.in/yaml.v3` for optional config, standard `testing` package, OpenSpec for requirement validation.

---

## File Structure

- Create `go.mod`: module metadata and dependencies.
- Create `cmd/ai-history/main.go`: CLI entrypoint.
- Create `internal/cli/cli.go`: command parsing, flag handling, output formatting.
- Create `internal/core/models.go`: normalized source, session, turn, diagnostics, and error types.
- Create `internal/core/id.go`: source-prefixed session ID helpers.
- Create `internal/core/service.go`: `Doctor`, `List`, `Show`, and `Context` orchestration.
- Create `internal/config/config.go`: zero-config defaults and YAML loading.
- Create `internal/discovery/discovery.go`: default source path discovery.
- Create `internal/readers/reader.go`: reader interface and shared helpers.
- Create `internal/readers/codex.go`: Codex storage reader.
- Create `internal/readers/claude.go`: Claude Code storage reader.
- Create `internal/readers/cursor.go`: Cursor latest macOS/Windows reader and diagnostics.
- Create `internal/render/render.go`: clean/summary/raw rendering and Markdown context handoff.
- Create `internal/testutil/fixtures.go`: fixture helpers for tests.
- Create `testdata/codex/...`: minimized Codex fixture.
- Create `testdata/claude/...`: minimized Claude Code fixture.
- Create `testdata/cursor/macos/...`: minimized latest Cursor macOS fixture after real sample capture.
- Create `testdata/cursor/windows/...`: minimized latest Cursor Windows fixture after Windows validation.
- Modify `README.md`: current build and P0 usage.
- Modify `openspec/changes/bootstrap-ai-session-history-cli/tasks.md`: check off completed tasks during implementation.

## Task 1: Go Module and CLI Skeleton

**Files:**
- Create: `go.mod`
- Create: `cmd/ai-history/main.go`
- Create: `internal/cli/cli.go`
- Test: `internal/cli/cli_test.go`

- [ ] **Step 1: Write the failing CLI skeleton test**

Create `internal/cli/cli_test.go`:

```go
package cli

import (
	"bytes"
	"testing"
)

func TestRunShowsHelpForNoArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(nil, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Usage: ai-history")) {
		t.Fatalf("expected usage in stderr, got %q", stderr.String())
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```bash
go test ./internal/cli
```

Expected: fail because `go.mod` or package `internal/cli` does not exist.

- [ ] **Step 3: Add minimal module and CLI code**

Create `go.mod`:

```go
module github.com/yangkushu/ai-session-history

go 1.22
```

Create `internal/cli/cli.go`:

```go
package cli

import (
	"fmt"
	"io"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: ai-history <command> [flags]")
		fmt.Fprintln(stderr, "Commands: doctor, list, show, context")
		return 2
	}
	fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
	return 2
}
```

Create `cmd/ai-history/main.go`:

```go
package main

import (
	"os"

	"github.com/yangkushu/ai-session-history/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
```

- [ ] **Step 4: Run the test and verify it passes**

Run:

```bash
go test ./internal/cli
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add go.mod cmd/ai-history/main.go internal/cli/cli.go internal/cli/cli_test.go
git commit -m "初始化 Go CLI 骨架"
```

## Task 2: Normalized Models and Session IDs

**Files:**
- Create: `internal/core/models.go`
- Create: `internal/core/id.go`
- Test: `internal/core/id_test.go`

- [ ] **Step 1: Write failing ID tests**

Create `internal/core/id_test.go`:

```go
package core

import "testing"

func TestSessionIDRoundTrip(t *testing.T) {
	id := MakeSessionID(SourceCodex, "abc")
	if id != "codex:abc" {
		t.Fatalf("unexpected id: %s", id)
	}

	source, native, err := ParseSessionID(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != SourceCodex || native != "abc" {
		t.Fatalf("unexpected parsed id: %s %s", source, native)
	}
}

func TestParseSessionIDRejectsInvalidInput(t *testing.T) {
	_, _, err := ParseSessionID("invalid")
	if err == nil {
		t.Fatal("expected invalid session id error")
	}
	if !IsCode(err, ErrInvalidSessionID) {
		t.Fatalf("expected %s, got %v", ErrInvalidSessionID, err)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```bash
go test ./internal/core
```

Expected: fail because `internal/core` does not exist.

- [ ] **Step 3: Implement normalized model and ID helpers**

Create `internal/core/models.go`:

```go
package core

import (
	"errors"
	"fmt"
	"time"
)

type Source string

const (
	SourceCodex  Source = "codex"
	SourceClaude Source = "claude"
	SourceCursor Source = "cursor"
)

type ReaderBackend string

const (
	BackendStorage ReaderBackend = "storage"
)

type ContentMode string

const (
	ModeClean   ContentMode = "clean"
	ModeSummary ContentMode = "summary"
	ModeRaw     ContentMode = "raw"
)

type TurnRole string

const (
	RoleUser      TurnRole = "user"
	RoleAssistant TurnRole = "assistant"
	RoleSystem    TurnRole = "system"
	RoleTool      TurnRole = "tool"
)

type TurnKind string

const (
	KindMessage    TurnKind = "message"
	KindToolCall   TurnKind = "tool_call"
	KindToolResult TurnKind = "tool_result"
	KindError      TurnKind = "error"
)

type Turn struct {
	Role          TurnRole   `json:"role"`
	Text          string     `json:"text"`
	Timestamp     *time.Time `json:"timestamp,omitempty"`
	Kind          TurnKind   `json:"kind"`
	Omitted       bool       `json:"omitted"`
	OmittedReason string     `json:"omitted_reason,omitempty"`
}

type SessionSummary struct {
	ID            string        `json:"id"`
	Source        Source        `json:"source"`
	NativeID      string        `json:"native_id"`
	Title         string        `json:"title"`
	Project       string        `json:"project,omitempty"`
	CWD           string        `json:"cwd,omitempty"`
	CreatedAt     *time.Time    `json:"created_at,omitempty"`
	UpdatedAt     *time.Time    `json:"updated_at,omitempty"`
	Preview       string        `json:"preview"`
	TurnCount     int           `json:"turn_count"`
	Available     bool          `json:"available"`
	ReaderBackend ReaderBackend `json:"reader_backend"`
}

type SessionDetail struct {
	Summary   SessionSummary `json:"summary"`
	Turns     []Turn         `json:"turns"`
	Truncated bool           `json:"truncated"`
}

type ErrorCode string

const (
	ErrPermissionDenied  ErrorCode = "permission_denied"
	ErrSourceUnavailable ErrorCode = "source_unavailable"
	ErrUnsupportedFormat ErrorCode = "unsupported_format"
	ErrSessionNotFound   ErrorCode = "session_not_found"
	ErrInvalidSessionID  ErrorCode = "invalid_session_id"
	ErrInvalidConfig     ErrorCode = "invalid_config"
	ErrReaderUnavailable ErrorCode = "reader_unavailable"
)

type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Source  Source    `json:"source,omitempty"`
	Path    string    `json:"path,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

func NewError(code ErrorCode, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func WrapSourceError(code ErrorCode, source Source, path string, err error) *AppError {
	return &AppError{
		Code:    code,
		Source:  source,
		Path:    path,
		Message: fmt.Sprintf("%s: %v", source, err),
	}
}

func IsCode(err error, code ErrorCode) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Code == code
}
```

Create `internal/core/id.go`:

```go
package core

import "strings"

func MakeSessionID(source Source, nativeID string) string {
	return string(source) + ":" + nativeID
}

func ParseSessionID(sessionID string) (Source, string, error) {
	sourceText, nativeID, ok := strings.Cut(sessionID, ":")
	source := Source(sourceText)
	if !ok || nativeID == "" || !IsSource(source) {
		return "", "", NewError(ErrInvalidSessionID, "invalid session id: "+sessionID)
	}
	return source, nativeID, nil
}

func IsSource(source Source) bool {
	return source == SourceCodex || source == SourceClaude || source == SourceCursor
}
```

- [ ] **Step 4: Run the test and verify it passes**

Run:

```bash
go test ./internal/core
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/core
git commit -m "添加会话模型"
```

## Task 3: Config Loading and Default Discovery

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/discovery/discovery.go`
- Test: `internal/config/config_test.go`
- Test: `internal/discovery/discovery_test.go`
- Modify: `go.mod`

- [ ] **Step 1: Write failing config and discovery tests**

Create `internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigEnablesAllSources(t *testing.T) {
	cfg := Default()
	for _, source := range []string{"codex", "claude", "cursor"} {
		if !cfg.Sources[source].Enabled {
			t.Fatalf("expected %s enabled", source)
		}
		if !cfg.Sources[source].UseDefaultPaths {
			t.Fatalf("expected %s default paths enabled", source)
		}
	}
	if cfg.Limits.DetailChars != 50000 || cfg.Limits.ContextChars != 20000 {
		t.Fatalf("unexpected limits: %+v", cfg.Limits)
	}
}

func TestLoadConfigOverridesSourceAndLimits(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(`
sources:
  cursor:
    enabled: false
    use_default_paths: false
    paths:
      - ~/cursor-fixture
limits:
  detail_chars: 100
  context_chars: 80
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Sources["cursor"].Enabled {
		t.Fatal("expected cursor disabled")
	}
	if cfg.Sources["cursor"].UseDefaultPaths {
		t.Fatal("expected default paths disabled")
	}
	if cfg.Limits.DetailChars != 100 || cfg.Limits.ContextChars != 80 {
		t.Fatalf("unexpected limits: %+v", cfg.Limits)
	}
}
```

Create `internal/discovery/discovery_test.go`:

```go
package discovery

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yangkushu/ai-session-history/internal/core"
)

func TestDiscoverCodexAndClaudeDefaults(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "Users", "alice")

	codex := DefaultPaths(core.SourceCodex, "darwin", home, nil)
	claude := DefaultPaths(core.SourceClaude, "darwin", home, nil)

	if codex[0] != filepath.Join(home, ".codex") {
		t.Fatalf("unexpected codex path: %v", codex)
	}
	if claude[0] != filepath.Join(home, ".claude") {
		t.Fatalf("unexpected claude path: %v", claude)
	}
}

func TestDiscoverCursorDefaultsForMacAndWindows(t *testing.T) {
	macHome := filepath.Join(string(filepath.Separator), "Users", "alice")
	winHome := `C:\Users\Alice`

	mac := DefaultPaths(core.SourceCursor, "darwin", macHome, nil)
	win := DefaultPaths(core.SourceCursor, "windows", winHome, map[string]string{"APPDATA": `C:\Users\Alice\AppData\Roaming`})

	if runtime.GOOS != "windows" && mac[0] != filepath.Join(macHome, "Library", "Application Support", "Cursor", "User") {
		t.Fatalf("unexpected mac cursor path: %v", mac)
	}
	if win[0] != filepath.Join(`C:\Users\Alice\AppData\Roaming`, "Cursor", "User") {
		t.Fatalf("unexpected windows cursor path: %v", win)
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./internal/config ./internal/discovery
```

Expected: fail because packages do not exist.

- [ ] **Step 3: Add config dependency**

Run:

```bash
go get gopkg.in/yaml.v3
```

Expected: `go.mod` and `go.sum` update.

- [ ] **Step 4: Implement config and discovery**

Create `internal/config/config.go`:

```go
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/yangkushu/ai-session-history/internal/core"
)

type SourceConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Paths           []string `yaml:"paths"`
	UseDefaultPaths bool     `yaml:"use_default_paths"`
}

type Limits struct {
	DetailChars  int `yaml:"detail_chars"`
	ContextChars int `yaml:"context_chars"`
}

type Config struct {
	Sources map[string]SourceConfig `yaml:"sources"`
	Limits  Limits                  `yaml:"limits"`
}

func Default() Config {
	return Config{
		Sources: map[string]SourceConfig{
			"codex":  {Enabled: true, UseDefaultPaths: true},
			"claude": {Enabled: true, UseDefaultPaths: true},
			"cursor": {Enabled: true, UseDefaultPaths: true},
		},
		Limits: Limits{DetailChars: 50000, ContextChars: 20000},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(expandHome(path))
	if err != nil {
		return Config{}, core.NewError(core.ErrInvalidConfig, err.Error())
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, core.NewError(core.ErrInvalidConfig, err.Error())
	}
	normalize(&cfg)
	return cfg, nil
}

func LoadOptional(path string) (Config, error) {
	if path == "" {
		return Default(), nil
	}
	return Load(path)
}

func normalize(cfg *Config) {
	defaults := Default()
	if cfg.Sources == nil {
		cfg.Sources = defaults.Sources
	}
	for name, def := range defaults.Sources {
		src, ok := cfg.Sources[name]
		if !ok {
			cfg.Sources[name] = def
			continue
		}
		if !src.Enabled && !src.UseDefaultPaths && len(src.Paths) == 0 {
			cfg.Sources[name] = src
			continue
		}
		if !src.Enabled {
			src.UseDefaultPaths = cfg.Sources[name].UseDefaultPaths
		}
		cfg.Sources[name] = src
	}
	if cfg.Limits.DetailChars <= 0 {
		cfg.Limits.DetailChars = defaults.Limits.DetailChars
	}
	if cfg.Limits.ContextChars <= 0 {
		cfg.Limits.ContextChars = defaults.Limits.ContextChars
	}
}

func expandHome(path string) string {
	if len(path) < 2 || path[:2] != "~/" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
```

Create `internal/discovery/discovery.go`:

```go
package discovery

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/yangkushu/ai-session-history/internal/core"
)

func DefaultPaths(source core.Source, goos string, home string, env map[string]string) []string {
	if goos == "" {
		goos = runtime.GOOS
	}
	if home == "" {
		home, _ = os.UserHomeDir()
	}

	switch source {
	case core.SourceCodex:
		paths := []string{filepath.Join(home, ".codex")}
		if goos == "darwin" {
			paths = append(paths, filepath.Join(home, "Library", "Application Support", "Codex"))
		}
		return paths
	case core.SourceClaude:
		return []string{filepath.Join(home, ".claude")}
	case core.SourceCursor:
		if goos == "darwin" {
			return []string{filepath.Join(home, "Library", "Application Support", "Cursor", "User")}
		}
		if goos == "windows" {
			appdata := ""
			if env != nil {
				appdata = env["APPDATA"]
			}
			if appdata == "" {
				appdata = filepath.Join(home, "AppData", "Roaming")
			}
			return []string{filepath.Join(appdata, "Cursor", "User")}
		}
		return []string{filepath.Join(home, ".config", "Cursor", "User")}
	default:
		return nil
	}
}
```

- [ ] **Step 5: Run tests and verify they pass**

Run:

```bash
go test ./internal/config ./internal/discovery
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/config internal/discovery
git commit -m "添加配置和路径发现"
```

## Task 4: Rendering and Context Handoff

**Files:**
- Create: `internal/render/render.go`
- Test: `internal/render/render_test.go`

- [ ] **Step 1: Write failing rendering tests**

Create `internal/render/render_test.go`:

```go
package render

import (
	"strings"
	"testing"
	"time"

	"github.com/yangkushu/ai-session-history/internal/core"
)

func TestRenderDetailModesAreBounded(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Can you run tests?", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: "I will check.", Kind: core.KindMessage},
		{Role: core.RoleTool, Text: strings.Repeat("line\n", 200), Kind: core.KindToolResult, OmittedReason: "tool_output"},
		{Role: core.RoleTool, Text: "pytest failed", Kind: core.KindError},
	})

	clean := Detail(detail, core.ModeClean, 500)
	summary := Detail(detail, core.ModeSummary, 500)
	raw := Detail(detail, core.ModeRaw, 80)

	if clean.Turns[2].Text != "[omitted: tool_output]" {
		t.Fatalf("unexpected clean tool text: %q", clean.Turns[2].Text)
	}
	if summary.Turns[2].Text != "[tool_result omitted: tool_output]" {
		t.Fatalf("unexpected summary tool text: %q", summary.Turns[2].Text)
	}
	if !raw.Truncated {
		t.Fatal("expected raw output truncated")
	}
}

func TestContextIncludesInitialGoalRecentConversationAndTargetCWD(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Initial goal", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: "Early answer", Kind: core.KindMessage},
		{Role: core.RoleUser, Text: "Latest question", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: "Latest answer", Kind: core.KindMessage},
	})

	text := Context(detail, ContextOptions{TargetCWD: "/new/project", MaxChars: 1000})

	for _, want := range []string{"# AI Session Context", "Original CWD: /old/project", "Target CWD: /new/project", "Initial goal", "Latest question", "Latest answer"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in context:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Key Decisions") {
		t.Fatal("P0 context must not invent interpretive summaries")
	}
}

func fixtureDetail(turns []core.Turn) core.SessionDetail {
	now := time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC)
	return core.SessionDetail{
		Summary: core.SessionSummary{
			ID: "codex:abc", Source: core.SourceCodex, NativeID: "abc",
			Title: "Test Session", Project: "project", CWD: "/old/project",
			CreatedAt: &now, UpdatedAt: &now, Preview: "preview", TurnCount: len(turns),
			Available: true, ReaderBackend: core.BackendStorage,
		},
		Turns: turns,
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./internal/render
```

Expected: fail because package `internal/render` does not exist.

- [ ] **Step 3: Implement rendering**

Create `internal/render/render.go`:

```go
package render

import (
	"fmt"
	"strings"

	"github.com/yangkushu/ai-session-history/internal/core"
)

type ContextOptions struct {
	TargetCWD string
	MaxChars  int
}

func Detail(detail core.SessionDetail, mode core.ContentMode, maxChars int) core.SessionDetail {
	if maxChars <= 0 {
		maxChars = 50000
	}
	out := detail
	out.Turns = nil
	remaining := maxChars
	for _, turn := range detail.Turns {
		if remaining <= 0 {
			out.Truncated = true
			break
		}
		turn.Text = turnText(turn, mode)
		if len(turn.Text) > remaining {
			turn.Text = turn.Text[:remaining]
			out.Truncated = true
		}
		if strings.HasPrefix(turn.Text, "[omitted") || strings.Contains(turn.Text, " omitted: ") {
			turn.Omitted = true
		}
		remaining -= len(turn.Text)
		out.Turns = append(out.Turns, turn)
	}
	return out
}

func Context(detail core.SessionDetail, opts ContextOptions) string {
	maxChars := opts.MaxChars
	if maxChars <= 0 {
		maxChars = 20000
	}
	clean := Detail(detail, core.ModeClean, maxChars*10)
	var b strings.Builder
	s := clean.Summary
	b.WriteString("# AI Session Context\n\n")
	b.WriteString("## Session\n\n")
	writeLine(&b, "- ID: %s", s.ID)
	writeLine(&b, "- Source: %s", s.Source)
	writeLine(&b, "- Original CWD: %s", valueOrUnknown(s.CWD))
	if opts.TargetCWD != "" {
		writeLine(&b, "- Target CWD: %s", opts.TargetCWD)
	}
	writeLine(&b, "- Created: %s", timeOrUnknown(s.CreatedAt))
	writeLine(&b, "- Updated: %s", timeOrUnknown(s.UpdatedAt))
	b.WriteString("\n## Initial Goal\n\n")
	b.WriteString(initialGoal(clean.Turns))
	b.WriteString("\n\n## Recent Conversation\n\n")
	for _, turn := range recentConversation(clean.Turns) {
		writeLine(&b, "### %s", strings.Title(string(turn.Role)))
		b.WriteString("\n")
		b.WriteString(limitText(turn.Text, 1200))
		b.WriteString("\n\n")
	}
	b.WriteString("## Omitted Content\n\n")
	b.WriteString("- Tool output omitted when applicable.\n")
	if clean.Truncated {
		b.WriteString("- Transcript truncated.\n")
	}
	b.WriteString("\n## Handoff Instruction\n\n")
	b.WriteString("Continue from this prior AI coding session. Treat the original CWD as historical context and the target CWD, when present, as the active working directory.")
	return bound(b.String(), maxChars)
}

func turnText(turn core.Turn, mode core.ContentMode) string {
	if mode == core.ModeRaw {
		return turn.Text
	}
	if turn.Role == core.RoleTool && turn.Kind == core.KindToolResult {
		reason := turn.OmittedReason
		if reason == "" {
			reason = "tool_output"
		}
		if mode == core.ModeSummary {
			return fmt.Sprintf("[%s omitted: %s]", turn.Kind, reason)
		}
		return fmt.Sprintf("[omitted: %s]", reason)
	}
	if turn.Role == core.RoleTool && mode == core.ModeClean && turn.Kind != core.KindError {
		reason := turn.OmittedReason
		if reason == "" {
			reason = string(turn.Kind)
		}
		return fmt.Sprintf("[omitted: %s]", reason)
	}
	return turn.Text
}

func initialGoal(turns []core.Turn) string {
	for _, turn := range turns {
		if turn.Role == core.RoleUser && strings.TrimSpace(turn.Text) != "" {
			return limitText(turn.Text, 1200)
		}
	}
	return "Unknown"
}

func recentConversation(turns []core.Turn) []core.Turn {
	var messages []core.Turn
	for _, turn := range turns {
		if (turn.Role == core.RoleUser || turn.Role == core.RoleAssistant) && strings.TrimSpace(turn.Text) != "" {
			messages = append(messages, turn)
		}
	}
	if len(messages) <= 3 {
		return messages
	}
	return messages[len(messages)-2:]
}

func writeLine(b *strings.Builder, format string, args ...any) {
	b.WriteString(fmt.Sprintf(format, args...))
	b.WriteString("\n")
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func timeOrUnknown(value interface{ String() string }) string {
	if value == nil {
		return "unknown"
	}
	return value.String()
}

func limitText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return strings.TrimSpace(text[:limit]) + "..."
}

func bound(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	marker := "\n\n[truncated]"
	if maxChars <= len(marker) {
		return text[:maxChars]
	}
	return strings.TrimSpace(text[:maxChars-len(marker)]) + marker
}
```

- [ ] **Step 4: Run tests and fix compile issues**

Run:

```bash
go test ./internal/render
```

Expected: pass. If `timeOrUnknown` does not compile with `*time.Time`, change its signature to `func timeOrUnknown(value *time.Time) string` and import `time`.

- [ ] **Step 5: Commit**

```bash
git add internal/render
git commit -m "添加上下文渲染"
```

## Task 5: Source Reader Interfaces and Codex Reader

**Files:**
- Create: `internal/readers/reader.go`
- Create: `internal/readers/codex.go`
- Test: `internal/readers/codex_test.go`
- Modify: `go.mod`
- Create fixture files under `testdata/codex/.codex/`

- [ ] **Step 1: Write failing Codex reader test**

Create `internal/readers/codex_test.go` with helper code that creates a temp
`state_5.sqlite`, writes a rollout JSONL, lists sessions, and reads detail:

```go
package readers

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCodexStorageReaderListsAndReadsRollout(t *testing.T) {
	root := t.TempDir()
	rollout := filepath.Join(root, "sessions", "2026", "07", "07", "rollout.jsonl")
	if err := os.MkdirAll(filepath.Dir(rollout), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rollout, []byte(`{"timestamp":"2026-07-07T00:00:01Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"Design cross AI history"}]}}
{"timestamp":"2026-07-07T00:00:02Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Use a CLI."}]}}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	writeCodexState(t, filepath.Join(root, "state_5.sqlite"), "abc", rollout)

	reader := NewCodexStorageReader([]string{root})
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "codex:abc" || sessions[0].Project != "demo" {
		t.Fatalf("unexpected sessions: %+v", sessions)
	}

	detail, err := reader.GetSession("abc")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if len(detail.Turns) != 2 || detail.Turns[0].Text != "Design cross AI history" {
		t.Fatalf("unexpected turns: %+v", detail.Turns)
	}
}

func writeCodexState(t *testing.T, path string, id string, rollout string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE threads (
		id TEXT PRIMARY KEY,
		rollout_path TEXT NOT NULL,
		created_at_ms INTEGER,
		updated_at_ms INTEGER,
		cwd TEXT,
		title TEXT,
		preview TEXT,
		archived INTEGER
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO threads VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, rollout, int64(1783382400000), int64(1783382402000),
		"/Users/alice/Workspace/demo", "Codex Title", "Design cross AI history", 0)
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run test and verify it fails**

Run:

```bash
go test ./internal/readers -run TestCodexStorageReaderListsAndReadsRollout
```

Expected: fail because package and dependency do not exist.

- [ ] **Step 3: Add SQLite dependency**

Run:

```bash
go get modernc.org/sqlite
```

Expected: `go.mod` and `go.sum` update.

- [ ] **Step 4: Implement reader interface and Codex reader**

Implement:

```go
type StorageReader interface {
	ListSessions() ([]core.SessionSummary, error)
	GetSession(nativeID string) (core.SessionDetail, error)
}
```

`NewCodexStorageReader(roots []string)` should:

- Scan each root for `state_5.sqlite`.
- Query `threads` for `id`, `rollout_path`, `created_at_ms`, `updated_at_ms`, `cwd`, `title`, `preview`.
- Build `core.SessionSummary` with `codex:<id>`.
- Read rollout JSONL and keep `response_item` payloads where `payload.type == "message"` and role is `user`, `assistant`, or `system`.
- Extract content text from string or array items with a `text` field.

- [ ] **Step 5: Run test and verify it passes**

Run:

```bash
go test ./internal/readers -run TestCodexStorageReaderListsAndReadsRollout
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/readers
git commit -m "添加 Codex 读取器"
```

## Task 6: Claude Code Reader

**Files:**
- Create: `internal/readers/claude.go`
- Test: `internal/readers/claude_test.go`

- [ ] **Step 1: Write failing Claude reader test**

Create `internal/readers/claude_test.go`:

```go
package readers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeStorageReaderListsAndReadsProjectJSONL(t *testing.T) {
	root := t.TempDir()
	session := filepath.Join(root, "projects", "-Users-alice-Workspace-demo", "claude-session.jsonl")
	if err := os.MkdirAll(filepath.Dir(session), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(session, []byte(`{"type":"user","sessionId":"claude-session","timestamp":"2026-07-07T00:00:01Z","cwd":"/Users/alice/Workspace/demo","message":{"role":"user","content":"How do I recover a session?"}}
{"type":"assistant","sessionId":"claude-session","timestamp":"2026-07-07T00:00:02Z","cwd":"/Users/alice/Workspace/demo","message":{"role":"assistant","content":[{"type":"text","text":"Use the session id."}]}}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	reader := NewClaudeStorageReader([]string{root})
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "claude:claude-session" || sessions[0].Title != "How do I recover a session?" {
		t.Fatalf("unexpected sessions: %+v", sessions)
	}

	detail, err := reader.GetSession("claude-session")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if len(detail.Turns) != 2 || detail.Turns[1].Text != "Use the session id." {
		t.Fatalf("unexpected turns: %+v", detail.Turns)
	}
}
```

- [ ] **Step 2: Run test and verify it fails**

Run:

```bash
go test ./internal/readers -run TestClaudeStorageReaderListsAndReadsProjectJSONL
```

Expected: fail because `NewClaudeStorageReader` is undefined.

- [ ] **Step 3: Implement Claude reader**

`NewClaudeStorageReader(roots []string)` should:

- Scan `<root>/projects/**/*.jsonl`.
- Parse non-empty JSONL rows.
- Use first `sessionId`, or file stem as fallback.
- Use first `cwd` for working directory and project.
- Convert `message.role` or row `type` values `user`, `assistant`, `system` into turns.
- Extract `message.content` from string or array items with `text`.

- [ ] **Step 4: Run test and verify it passes**

Run:

```bash
go test ./internal/readers -run TestClaudeStorageReaderListsAndReadsProjectJSONL
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/readers/claude.go internal/readers/claude_test.go
git commit -m "添加 Claude Code 读取器"
```

## Task 7: Service Layer, Filters, and Diagnostics

**Files:**
- Create: `internal/core/service.go`
- Test: `internal/core/service_test.go`
- Modify: `internal/readers/reader.go`

- [ ] **Step 1: Write failing service tests**

Create `internal/core/service_test.go` with fake readers:

```go
package core

import (
	"testing"
	"time"
)

func TestListFiltersBySourceCWDAndUnder(t *testing.T) {
	now := time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC)
	service := NewService(map[Source]Reader{
		SourceCodex: fakeReader{summaries: []SessionSummary{
			{ID: "codex:one", Source: SourceCodex, NativeID: "one", CWD: "/work/a", UpdatedAt: &now, Available: true},
		}},
		SourceClaude: fakeReader{summaries: []SessionSummary{
			{ID: "claude:two", Source: SourceClaude, NativeID: "two", CWD: "/work/b", UpdatedAt: &now, Available: true},
		}},
	})

	result := service.List(ListOptions{Under: "/work", Limit: 10})
	if len(result.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %+v", result.Sessions)
	}

	result = service.List(ListOptions{CWD: "/work/a", Limit: 10})
	if len(result.Sessions) != 1 || result.Sessions[0].ID != "codex:one" {
		t.Fatalf("unexpected exact cwd result: %+v", result.Sessions)
	}
}

func TestShowReturnsSessionNotFoundForMissingNativeID(t *testing.T) {
	service := NewService(map[Source]Reader{SourceCodex: fakeReader{}})
	_, err := service.Show("codex:missing", ShowOptions{Mode: ModeClean, MaxChars: 100})
	if !IsCode(err, ErrSessionNotFound) {
		t.Fatalf("expected session_not_found, got %v", err)
	}
}

type fakeReader struct {
	summaries []SessionSummary
	details   map[string]SessionDetail
}

func (f fakeReader) ListSessions() ([]SessionSummary, error) {
	return f.summaries, nil
}

func (f fakeReader) GetSession(nativeID string) (SessionDetail, error) {
	if f.details != nil {
		if detail, ok := f.details[nativeID]; ok {
			return detail, nil
		}
	}
	return SessionDetail{}, NewError(ErrSessionNotFound, nativeID)
}

func (f fakeReader) Doctor() SourceDiagnostic {
	return SourceDiagnostic{Source: SourceCodex, Status: "available"}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./internal/core -run 'TestList|TestShow'
```

Expected: fail because service types do not exist.

- [ ] **Step 3: Implement service layer**

Implement `Reader`, `SourceDiagnostic`, `Service`, `ListOptions`, `ShowOptions`,
and `ContextOptions` in `internal/core/service.go`. Service should:

- List enabled readers and degrade per-source failures into diagnostics.
- Sort by `updated_at` then `created_at`, descending.
- Apply `Source`, `CWD`, `Under`, and `Limit` filters.
- Parse session IDs with `ParseSessionID`.
- Return `ErrSessionNotFound` for missing sessions.
- Delegate rendering through function fields or a small adapter added in the next task if import cycles appear.

- [ ] **Step 4: Run tests and verify they pass**

Run:

```bash
go test ./internal/core
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/core/service.go internal/core/service_test.go internal/readers/reader.go
git commit -m "添加核心服务"
```

## Task 8: CLI Commands

**Files:**
- Modify: `internal/cli/cli.go`
- Test: `internal/cli/cli_test.go`
- Create: `internal/cli/output.go`

- [ ] **Step 1: Write failing CLI command tests**

Extend `internal/cli/cli_test.go` with tests for unknown `search`, `context`, and
JSON output using an injectable service factory:

```go
func TestSearchCommandIsUnavailable(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"search", "query"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("search is not available in P0")) {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}
```

Add service-backed tests after `RunWithService` exists:

```go
func TestContextCommandWritesMarkdown(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service := fakeCLIService{contextText: "# AI Session Context\n\nbody"}

	code := RunWithService([]string{"context", "codex:abc", "--target-cwd", "/new"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("# AI Session Context")) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}
```

- [ ] **Step 2: Run CLI tests and verify they fail**

Run:

```bash
go test ./internal/cli
```

Expected: fail because commands are not implemented.

- [ ] **Step 3: Implement commands**

Implement:

- `doctor [--json] [--config path]`
- `list [--source source] [--cwd path] [--under path] [--limit n] [--json] [--config path]`
- `show <session-id> [--mode mode] [--max-chars n] [--json] [--config path]`
- `context <session-id> [--target-cwd path] [--max-chars n] [--config path]`
- `search` returns exit code `2` with `search is not available in P0`.

Use `flag.NewFlagSet` per command. Keep command parsing in `internal/cli`; create
the real service from config/discovery/readers through a small builder function.

- [ ] **Step 4: Run CLI tests and verify they pass**

Run:

```bash
go test ./internal/cli
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/cli cmd/ai-history/main.go
git commit -m "添加 CLI 命令"
```

## Task 9: Cursor Latest macOS Reader

**Files:**
- Modify: `internal/readers/cursor.go`
- Test: `internal/readers/cursor_test.go`
- Add: `testdata/cursor/macos/README.md`
- Add: minimized real Cursor macOS fixture files

- [ ] **Step 1: Capture latest macOS Cursor sample**

Install latest Cursor on macOS, create a small AI chat with non-sensitive text,
then inspect likely storage under:

```text
~/Library/Application Support/Cursor/User
```

Minimize the fixture to the smallest database or JSON files required to prove
session listing and reading. Replace private content with neutral text while
preserving keys and storage shape.

- [ ] **Step 2: Write failing Cursor macOS test from the fixture**

Create `internal/readers/cursor_test.go`:

```go
package readers

import "testing"

func TestCursorMacOSLatestReaderListsAndReadsRealFixture(t *testing.T) {
	reader := NewCursorStorageReader([]string{"../../testdata/cursor/macos/User"})
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("expected at least one Cursor session")
	}
	detail, err := reader.GetSession(sessions[0].NativeID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if len(detail.Turns) == 0 {
		t.Fatalf("expected turns for cursor fixture: %+v", detail)
	}
}
```

- [ ] **Step 3: Run the test and verify it fails**

Run:

```bash
go test ./internal/readers -run TestCursorMacOSLatestReaderListsAndReadsRealFixture
```

Expected: fail until the real fixture shape is implemented.

- [ ] **Step 4: Implement Cursor reader for latest macOS fixture**

Implement only the observed latest macOS storage shape. If multiple candidate
databases or keys exist, keep the parser narrow and return `ErrUnsupportedFormat`
for unknown shapes.

- [ ] **Step 5: Run the test and verify it passes**

Run:

```bash
go test ./internal/readers -run TestCursorMacOSLatestReaderListsAndReadsRealFixture
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add internal/readers/cursor.go internal/readers/cursor_test.go testdata/cursor/macos
git commit -m "添加 Cursor macOS 读取器"
```

## Task 10: Windows Cursor Validation Gate

**Files:**
- Modify: `internal/discovery/discovery.go`
- Modify: `internal/readers/cursor.go`
- Add: `testdata/cursor/windows/README.md`
- Add later: minimized Windows latest Cursor fixture

- [ ] **Step 1: Add Windows discovery test if not already covered**

Ensure `internal/discovery/discovery_test.go` verifies:

```go
win := DefaultPaths(core.SourceCursor, "windows", `C:\Users\Alice`, map[string]string{"APPDATA": `C:\Users\Alice\AppData\Roaming`})
if win[0] != filepath.Join(`C:\Users\Alice\AppData\Roaming`, "Cursor", "User") {
	t.Fatalf("unexpected windows cursor path: %v", win)
}
```

- [ ] **Step 2: Add Windows fixture README**

Create `testdata/cursor/windows/README.md`:

```markdown
# Cursor Windows Fixture

This directory is intentionally empty until latest Cursor on Windows is
validated with a real local sample. P0 is not complete until a minimized fixture
derived from that sample is added and the Cursor Windows reader test passes.
```

- [ ] **Step 3: Add pending validation task to OpenSpec tasks**

Keep `openspec/changes/bootstrap-ai-session-history-cli/tasks.md` Windows Cursor
validation unchecked until the real Windows fixture passes.

- [ ] **Step 4: Commit**

```bash
git add internal/discovery/discovery_test.go testdata/cursor/windows/README.md openspec/changes/bootstrap-ai-session-history-cli/tasks.md
git commit -m "添加 Cursor Windows 验收门槛"
```

## Task 11: Documentation and OpenSpec Task Sync

**Files:**
- Modify: `README.md`
- Modify: `docs/2026-07-07-product-direction.md`
- Modify: `openspec/changes/bootstrap-ai-session-history-cli/tasks.md`

- [ ] **Step 1: Update README examples**

Document:

```bash
go test ./...
go run ./cmd/ai-history doctor
go run ./cmd/ai-history list --under /path/to/workspace
go run ./cmd/ai-history show codex:<session-id> --mode clean
go run ./cmd/ai-history context codex:<session-id> --target-cwd /new/project
```

- [ ] **Step 2: Update product direction**

Make P0 command examples match the accepted scope:

```bash
ai-history doctor --json
ai-history list --under ~/workspaces --json
ai-history show codex:<session-id> --mode clean --json
ai-history context codex:<session-id> --target-cwd ~/workspaces/new-project
```

Remove `search` and `mcp serve` from P0 examples; keep MCP as future adapter.

- [ ] **Step 3: Sync OpenSpec tasks**

Check off completed tasks in `openspec/changes/bootstrap-ai-session-history-cli/tasks.md`.
Leave Cursor Windows latest validation unchecked until it passes on Windows.

- [ ] **Step 4: Commit**

```bash
git add README.md docs/2026-07-07-product-direction.md openspec/changes/bootstrap-ai-session-history-cli/tasks.md
git commit -m "更新使用文档"
```

## Task 12: Full Verification and Archive

**Files:**
- Modify after implementation: OpenSpec archive output under `openspec/specs/`

- [ ] **Step 1: Run Go tests**

Run:

```bash
go test ./...
```

Expected: all tests pass. If dependency download is blocked by network sandboxing,
rerun the same command with escalation.

- [ ] **Step 2: Run formatting**

Run:

```bash
gofmt -w cmd internal
```

Expected: no output.

- [ ] **Step 3: Re-run tests after formatting**

Run:

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 4: Validate OpenSpec change**

Run:

```bash
openspec validate bootstrap-ai-session-history-cli --strict
```

Expected: `Change 'bootstrap-ai-session-history-cli' is valid`.

- [ ] **Step 5: Archive only after P0 completion**

Do not archive if Cursor Windows latest validation is still pending. Once all P0
tasks including Windows fixture validation are complete, run:

```bash
openspec archive bootstrap-ai-session-history-cli --yes
openspec validate --strict
```

Expected: archive succeeds and specs validate.

- [ ] **Step 6: Final commit and push**

```bash
git status --short
git add .
git commit -m "完成 AI Session History CLI"
git push
```

## Self-Review

- Spec coverage: local-first CLI, diagnostics, listing filters, session detail,
  content modes, deterministic context handoff, optional config, Cursor macOS and
  Windows validation gates, no-search P0, and no full import/export P0 are all
  mapped to tasks.
- Placeholder scan: no placeholder markers or open-ended "add appropriate
  handling" steps remain; Windows validation is explicitly gated by a fixture
  task.
- Type consistency: source, session, content mode, error code, reader, and
  service names are defined before later tasks use them.
