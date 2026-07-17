# Improve List Time Display Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace human-readable `ai-history list` rows with aligned four-line session blocks that show updated and created local timestamps with compact relative ages.

**Architecture:** Keep normalized session data and JSON serialization unchanged. Add a focused text renderer in `internal/cli/list_text.go` with pure helpers for relative time, absolute time, title normalization, display-width truncation, and block alignment; `runList` captures the current time once and delegates only its non-JSON branch.

**Tech Stack:** Go 1.22, `github.com/mattn/go-runewidth`, standard `time`/`io`/`strings`, existing CLI fake service tests, OpenSpec.

---

### Task 1: Relative and Absolute Time Formatting

**Files:**
- Create: `internal/cli/list_text.go`
- Create: `internal/cli/list_text_test.go`

- [ ] **Step 1: Write failing relative-age boundary tests**

Add table-driven tests around a fixed `now` for future timestamps, `59s`, `1m`, `59m`, `1h`, `23h`, `1d`, `29d`, `30d`, `364d`, and `365d`. The test calls `compactAge(at, now)` and expects `now`, `1m`, `59m`, `1h`, `23h`, `1d`, `29d`, `1mo`, `12mo`, and `1y` at the approved boundaries.

```go
func TestCompactAge(t *testing.T) {
	now := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		age  time.Duration
		want string
	}{
		{name: "future", age: -time.Minute, want: "now"},
		{name: "seconds", age: 59 * time.Second, want: "now"},
		{name: "minute", age: time.Minute, want: "1m"},
		{name: "minutes", age: 59 * time.Minute, want: "59m"},
		{name: "hour", age: time.Hour, want: "1h"},
		{name: "hours", age: 23 * time.Hour, want: "23h"},
		{name: "day", age: 24 * time.Hour, want: "1d"},
		{name: "days", age: 29 * 24 * time.Hour, want: "29d"},
		{name: "month", age: 30 * 24 * time.Hour, want: "1mo"},
		{name: "months", age: 364 * 24 * time.Hour, want: "12mo"},
		{name: "year", age: 365 * 24 * time.Hour, want: "1y"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compactAge(now.Add(-tt.age), now); got != tt.want {
				t.Fatalf("compactAge() = %q, want %q", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `GOCACHE=/tmp/go-build go test ./internal/cli -run TestCompactAge -count=1`

Expected: FAIL because `compactAge` is undefined.

- [ ] **Step 3: Implement the minimal time helpers**

Create `compactAge(at, now time.Time) string` using floored duration divisions and the approved minute/hour/day/30-day-month/365-day-year boundaries. Add `formatListTime(value *time.Time, now time.Time, loc *time.Location) string`, returning `unknown` for nil and otherwise `value.In(loc).Format("2006-01-02 15:04") + " (" + compactAge(*value, now) + ")"`.

```go
func compactAge(at, now time.Time) string {
	age := now.Sub(at)
	switch {
	case age < time.Minute:
		return "now"
	case age < time.Hour:
		return fmt.Sprintf("%dm", age/time.Minute)
	case age < 24*time.Hour:
		return fmt.Sprintf("%dh", age/time.Hour)
	case age < 30*24*time.Hour:
		return fmt.Sprintf("%dd", age/(24*time.Hour))
	case age < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", age/(30*24*time.Hour))
	default:
		return fmt.Sprintf("%dy", age/(365*24*time.Hour))
	}
}
```

- [ ] **Step 4: Add local-time, independent-age, and nil tests**

Use `time.FixedZone("UTC+8", 8*60*60)` and assert a UTC timestamp becomes `2026-07-17 18:00 (2h)`. Assert created and updated values are formatted independently, and nil returns exactly `unknown`.

- [ ] **Step 5: Run focused tests and verify GREEN**

Run: `GOCACHE=/tmp/go-build go test ./internal/cli -run 'TestCompactAge|TestFormatListTime' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit the time formatter**

```bash
git add internal/cli/list_text.go internal/cli/list_text_test.go
git commit -m "功能：格式化列表时间"
```

### Task 2: Unicode-Safe Title Formatting

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `internal/cli/list_text.go`
- Modify: `internal/cli/list_text_test.go`

- [ ] **Step 1: Write failing title normalization tests**

Add cases proving `strings.Fields`-style whitespace collapsing and that ASCII, CJK, emoji, and combining-character titles at or below 80 display cells remain unchanged. Add over-limit cases that end in `…` and satisfy `runewidth.StringWidth(got) <= 80`.

```go
func TestFormatListTitle(t *testing.T) {
	if got := formatListTitle("  first\n\tsecond   third  "); got != "first second third" {
		t.Fatalf("formatListTitle() = %q", got)
	}
	for _, input := range []string{
		strings.Repeat("a", 81),
		strings.Repeat("界", 41),
		strings.Repeat("🙂", 41),
		strings.Repeat("e\u0301", 81),
	} {
		got := formatListTitle(input)
		if !strings.HasSuffix(got, "…") || runewidth.StringWidth(got) > listTitleWidth {
			t.Fatalf("invalid truncated title %q width=%d", got, runewidth.StringWidth(got))
		}
	}
}
```

- [ ] **Step 2: Run the title test and verify RED**

Run: `GOCACHE=/tmp/go-build go test ./internal/cli -run TestFormatListTitle -count=1`

Expected: FAIL because the title formatter and width dependency are absent.

- [ ] **Step 3: Add display-width support and minimal implementation**

Run: `go get github.com/mattn/go-runewidth@v0.0.19`

Implement a constant `listTitleWidth = 80` and:

```go
func formatListTitle(title string) string {
	normalized := strings.Join(strings.Fields(title), " ")
	if runewidth.StringWidth(normalized) <= listTitleWidth {
		return normalized
	}
	return runewidth.Truncate(normalized, listTitleWidth-runewidth.StringWidth("…"), "") + "…"
}
```

- [ ] **Step 4: Run focused and package tests**

Run: `GOCACHE=/tmp/go-build go test ./internal/cli -run TestFormatListTitle -count=1`

Expected: PASS.

Run: `GOCACHE=/tmp/go-build go test ./internal/cli -count=1`

Expected: PASS.

- [ ] **Step 5: Commit title formatting**

```bash
git add go.mod go.sum internal/cli/list_text.go internal/cli/list_text_test.go
git commit -m "功能：规范列表标题宽度"
```

### Task 3: Four-Line Session Block Renderer

**Files:**
- Modify: `internal/cli/list_text.go`
- Modify: `internal/cli/list_text_test.go`

- [ ] **Step 1: Write failing renderer snapshot tests**

Build two summaries with `codex` and `claude` sources, fixed timestamps, one missing CWD, a multiline title, and long untruncated IDs. Call `writeListText` with a fixed clock and location and assert the complete string, including content-column alignment, updated-before-created order, `unknown`, one blank line between blocks, and exactly one final newline.

```go
want := "codex   First title\n" +
	"        2026-07-17 18:00 (20m)  2026-07-17 17:00 (1h)\n" +
	"        /work/a\n" +
	"        codex:complete-id\n\n" +
	"claude  Second title\n" +
	"        unknown  unknown\n" +
	"        unknown\n" +
	"        claude:complete-id\n"
```

Also assert an empty session slice writes no bytes.

- [ ] **Step 2: Run the renderer test and verify RED**

Run: `GOCACHE=/tmp/go-build go test ./internal/cli -run TestWriteListText -count=1`

Expected: FAIL because `writeListText` is undefined.

- [ ] **Step 3: Implement deterministic block rendering**

Implement `writeListText(w io.Writer, sessions []core.SessionSummary, now time.Time, loc *time.Location) error`. Compute the widest source display width once, use two spaces after that column, emit a blank line only before blocks after the first, preserve full non-empty CWD and ID, use `unknown` for empty CWD, and return write errors.

```go
func writeListText(w io.Writer, sessions []core.SessionSummary, now time.Time, loc *time.Location) error {
	sourceWidth := 0
	for _, session := range sessions {
		sourceWidth = max(sourceWidth, runewidth.StringWidth(string(session.Source)))
	}
	indent := strings.Repeat(" ", sourceWidth+2)
	for i, session := range sessions {
		if i > 0 {
			if _, err := io.WriteString(w, "\n"); err != nil { return err }
		}
		source := string(session.Source)
		gap := strings.Repeat(" ", sourceWidth-runewidth.StringWidth(source)+2)
		cwd := session.CWD
		if cwd == "" { cwd = "unknown" }
		block := source + gap + formatListTitle(session.Title) + "\n" +
			indent + formatListTime(session.UpdatedAt, now, loc) + "  " + formatListTime(session.CreatedAt, now, loc) + "\n" +
			indent + cwd + "\n" + indent + session.ID + "\n"
		if _, err := io.WriteString(w, block); err != nil { return err }
	}
	return nil
}
```

- [ ] **Step 4: Run renderer and package tests**

Run: `GOCACHE=/tmp/go-build go test ./internal/cli -run TestWriteListText -count=1`

Expected: PASS.

Run: `GOCACHE=/tmp/go-build go test ./internal/cli -count=1`

Expected: PASS.

- [ ] **Step 5: Commit block rendering**

```bash
git add internal/cli/list_text.go internal/cli/list_text_test.go
git commit -m "功能：渲染列表会话块"
```

### Task 4: CLI Integration, Documentation, and Verification

**Files:**
- Modify: `internal/cli/cli.go:185-192`
- Modify: `internal/cli/cli_test.go:510-530`
- Modify: `README.md:90-105`
- Modify: `openspec/changes/improve-list-time-display/tasks.md`

- [ ] **Step 1: Write a failing CLI text-output test**

Add `TestListCommandWritesReadableText` with created and updated timestamps based on one captured `now`. Assert source/title, updated-before-created formatted values, full CWD, and full ID are present in the expected line order. Keep `TestListCommandWritesJSON` and extend it to unmarshal `created_at` and `updated_at` values exactly.

- [ ] **Step 2: Run the CLI tests and verify RED**

Run: `GOCACHE=/tmp/go-build go test ./internal/cli -run 'TestListCommandWritesReadableText|TestListCommandWritesJSON' -count=1`

Expected: the readable-text test FAILS against the legacy tab-separated output while the JSON test PASSES.

- [ ] **Step 3: Route non-JSON output through the renderer**

Replace the legacy loop with:

```go
if err := writeListText(stdout, result.Sessions, time.Now(), time.Local); err != nil {
	fmt.Fprintf(stderr, "cannot write list output: %v\n", err)
	return 1
}
return 0
```

Add the `time` import to `internal/cli/cli.go`. Do not change the preceding JSON branch.

- [ ] **Step 4: Update user documentation**

Add a concise non-JSON example to README showing the four-line layout. State that absolute timestamps use local time, relative ages are approximate, and scripts must use `--json` because the text view is human-oriented.

- [ ] **Step 5: Run automated verification**

Run: `gofmt -w internal/cli/cli.go internal/cli/cli_test.go internal/cli/list_text.go internal/cli/list_text_test.go`

Run: `GOCACHE=/tmp/go-build go test ./internal/cli -count=1`

Run: `GOCACHE=/tmp/go-build go test ./... -count=1`

Run: `GOCACHE=/tmp/go-build go build ./cmd/ai-history`

Run: `git diff --check`

Run: `openspec validate improve-list-time-display`

Expected: all commands succeed and OpenSpec reports the change as valid.

- [ ] **Step 6: Manually inspect representative output**

Run the built CLI with `list --here --limit 3` and `list --limit 3`; verify updated time appears first, both timestamps use local time, relative ages are compact, Chinese titles align, long CWD/ID remain complete, and no ANSI escapes appear when redirected to a file.

- [ ] **Step 7: Mark OpenSpec tasks complete and commit implementation**

Update every implemented checkbox in `openspec/changes/improve-list-time-display/tasks.md` to `[x]`, then run:

```bash
git add README.md go.mod go.sum internal/cli/cli.go internal/cli/cli_test.go internal/cli/list_text.go internal/cli/list_text_test.go openspec/changes/improve-list-time-display/tasks.md docs/superpowers/plans/2026-07-17-improve-list-time-display.md
git commit -m "功能：改进列表时间展示"
```

- [ ] **Step 8: Archive after verification**

Run `openspec archive improve-list-time-display`, validate the archived result and main specs, then commit the archive with:

```bash
git add openspec
git commit -m "规格：归档列表时间展示变更"
```
