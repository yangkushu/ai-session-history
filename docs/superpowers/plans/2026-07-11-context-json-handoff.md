# Context JSON Handoff Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `ai-history context <session-id> --json` as a versioned structured handoff output while preserving default Markdown behavior.

**Architecture:** Refactor `internal/render` so both Markdown and JSON are built from one structured handoff model. Keep source readers and `show --json` unchanged; wire the CLI `context` command to either render Markdown or encode the structured handoff object.

**Tech Stack:** Go 1.22, standard library `encoding/json`, existing `internal/core`, `internal/render`, and `internal/cli` packages.

---

## File Structure

- Modify `internal/render/render.go`: add structured handoff types, `BuildHandoff`, `ContextFromHandoff`, and JSON-friendly note/session section fields; keep `Context` as the Markdown public wrapper.
- Modify `internal/render/render_test.go`: add tests for the structured handoff model and keep existing Markdown behavior tests passing.
- Modify `internal/cli/cli.go`: add `--json` / `-j` flags for `context` and extend the service interface to expose structured handoff output.
- Modify `internal/cli/service.go`: implement the new service method by loading the session detail once and calling the shared renderer.
- Modify `internal/cli/cli_test.go`: add CLI tests for `context --json`, `context -j`, and unchanged Markdown output.
- Modify `README.md` and `README.zh-CN.md`: document `context --json` and distinguish it from `show --json`.
- Update `openspec/changes/add-context-json-handoff/tasks.md`: mark completed tasks during execution.

## Task 1: Structured Handoff Model

**Files:**
- Modify: `internal/render/render.go`
- Test: `internal/render/render_test.go`

- [ ] **Step 1: Write failing renderer tests for JSON handoff shape**

Add this test to `internal/render/render_test.go` near the existing context tests:

```go
func TestBuildHandoffProducesVersionedJSONShape(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Initial goal", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: "Latest answer", Kind: core.KindMessage},
		{Role: core.RoleTool, Text: "go test ./... passed", Kind: core.KindToolResult},
	})

	handoff := BuildHandoff(detail, ContextOptions{TargetCWD: "/new/project", MaxChars: 3000})

	if handoff.SchemaVersion != "context-handoff.v1" {
		t.Fatalf("unexpected schema version: %q", handoff.SchemaVersion)
	}
	if handoff.Session.ID != "codex:abc" || handoff.Session.TargetCWD != "/new/project" {
		t.Fatalf("unexpected session metadata: %+v", handoff.Session)
	}
	if !handoff.InitialGoal.Available || handoff.InitialGoal.Text != "Initial goal" {
		t.Fatalf("unexpected initial goal: %+v", handoff.InitialGoal)
	}
	if len(handoff.RecentConversation) == 0 {
		t.Fatalf("expected recent conversation: %+v", handoff)
	}
	if len(handoff.ToolOutcomes) != 1 || handoff.ToolOutcomes[0].Text != "go test ./... passed" {
		t.Fatalf("unexpected tool outcomes: %+v", handoff.ToolOutcomes)
	}
	if handoff.HandoffInstruction == "" {
		t.Fatal("expected required handoff instruction")
	}
}
```

- [ ] **Step 2: Run the targeted test and confirm it fails**

Run:

```bash
GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/render -run TestBuildHandoffProducesVersionedJSONShape -count=1
```

Expected: compile failure because `BuildHandoff` and the handoff types do not exist yet.

- [ ] **Step 3: Add handoff types in `internal/render/render.go`**

Add these types after `ContextOptions`:

```go
const HandoffSchemaVersion = "context-handoff.v1"

type HandoffContext struct {
	SchemaVersion      string             `json:"schema_version"`
	Session            HandoffSession     `json:"session"`
	InitialGoal        HandoffInitialGoal `json:"initial_goal"`
	RecentConversation []core.Turn        `json:"recent_conversation"`
	ToolOutcomes       []core.Turn        `json:"tool_outcomes"`
	HandoffNotes       []HandoffNote      `json:"handoff_notes"`
	HandoffInstruction string             `json:"handoff_instruction"`
	Truncated          bool               `json:"truncated"`
}

type HandoffSession struct {
	ID          string       `json:"id"`
	Source      core.Source  `json:"source"`
	NativeID    string       `json:"native_id"`
	Title       string       `json:"title,omitempty"`
	OriginalCWD string       `json:"original_cwd,omitempty"`
	TargetCWD   string       `json:"target_cwd,omitempty"`
	CreatedAt   *time.Time   `json:"created_at,omitempty"`
	UpdatedAt   *time.Time   `json:"updated_at,omitempty"`
}

type HandoffInitialGoal struct {
	Available bool   `json:"available"`
	Text      string `json:"text,omitempty"`
}

type HandoffNote struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
```

- [ ] **Step 4: Implement `BuildHandoff` with shared filtering**

Add this function in `internal/render/render.go` near `Context`:

```go
func BuildHandoff(detail core.SessionDetail, opts ContextOptions) HandoffContext {
	maxChars := opts.MaxChars
	if maxChars <= 0 {
		maxChars = 20000
	}
	clean := Detail(detail, core.ModeClean, maxChars*10)
	filteredTurns, skippedSetup := filterContextTurns(clean.Turns)
	outcomes := toolOutcomes(clean.Turns)
	omittedToolOutput := countOmittedToolOutput(clean.Turns)
	notes := contextNotes{
		skippedSetup:      skippedSetup,
		omittedToolOutput: omittedToolOutput,
		truncated:         clean.Truncated,
	}
	handoff := buildHandoff(clean.Summary, filteredTurns, outcomes, notes, opts, 1200, 1200, false, true)
	if contentSize(handoff) > maxChars {
		handoff = buildHandoff(clean.Summary, filteredTurns, outcomes, contextNotes{
			skippedSetup:      skippedSetup,
			omittedToolOutput: omittedToolOutput,
			truncated:         true,
		}, opts, 360, 180, true, true)
	}
	if contentSize(handoff) > maxChars {
		handoff = buildHandoff(clean.Summary, filteredTurns, outcomes, contextNotes{
			skippedSetup:      skippedSetup,
			omittedToolOutput: omittedToolOutput,
			truncated:         true,
		}, opts, 90, 0, true, false)
	}
	return handoff
}
```

- [ ] **Step 5: Add `buildHandoff`, note helpers, and content budget helper**

Implement helpers in `internal/render/render.go`:

```go
func buildHandoff(s core.SessionSummary, turns []core.Turn, outcomes []core.Turn, notes contextNotes, opts ContextOptions, goalLimit int, turnLimit int, compactRecent bool, includeDetails bool) HandoffContext {
	recent := []core.Turn{}
	toolResults := []core.Turn{}
	if includeDetails {
		recent = limitedTurns(recentConversation(turns, compactRecent), turnLimit)
		toolResults = limitedTurns(outcomes, turnLimit)
	}
	return HandoffContext{
		SchemaVersion: HandoffSchemaVersion,
		Session: HandoffSession{
			ID:          s.ID,
			Source:      s.Source,
			NativeID:    s.NativeID,
			Title:       s.Title,
			OriginalCWD: s.CWD,
			TargetCWD:   opts.TargetCWD,
			CreatedAt:   s.CreatedAt,
			UpdatedAt:   s.UpdatedAt,
		},
		InitialGoal:        handoffInitialGoal(turns, goalLimit),
		RecentConversation: recent,
		ToolOutcomes:       toolResults,
		HandoffNotes:       handoffNotes(notes),
		HandoffInstruction: handoffInstruction(includeDetails),
		Truncated:          notes.truncated,
	}
}

func handoffInitialGoal(turns []core.Turn, limit int) HandoffInitialGoal {
	for _, turn := range turns {
		if turn.Role == core.RoleUser && strings.TrimSpace(turn.Text) != "" {
			return HandoffInitialGoal{Available: true, Text: limitText(turn.Text, limit)}
		}
	}
	return HandoffInitialGoal{Available: false}
}

func limitedTurns(turns []core.Turn, limit int) []core.Turn {
	out := make([]core.Turn, 0, len(turns))
	for _, turn := range turns {
		turn.Text = limitText(turn.Text, limit)
		out = append(out, turn)
	}
	return out
}

func handoffNotes(notes contextNotes) []HandoffNote {
	out := []HandoffNote{}
	if notes.skippedSetup == 0 && notes.omittedToolOutput == 0 && !notes.truncated {
		return append(out, HandoffNote{Code: "no_omissions", Message: "No skipped, omitted, or truncated content."})
	}
	if notes.skippedSetup > 0 {
		out = append(out, HandoffNote{Code: "setup_boilerplate_skipped", Message: fmt.Sprintf("Skipped setup boilerplate turns: %d", notes.skippedSetup)})
	}
	if notes.omittedToolOutput > 0 {
		out = append(out, HandoffNote{Code: "tool_output_omitted", Message: fmt.Sprintf("Omitted noisy tool output turns: %d", notes.omittedToolOutput)})
	}
	if notes.truncated {
		out = append(out, HandoffNote{Code: "context_truncated", Message: "Context truncated to --max-chars."})
	}
	return out
}

func handoffInstruction(includeDetails bool) string {
	if includeDetails {
		return "Continue from this prior AI coding session. Treat the original CWD as historical context and the target CWD, when present, as the active working directory."
	}
	return "Continue from this prior AI coding session."
}

func contentSize(handoff HandoffContext) int {
	total := len(handoff.InitialGoal.Text) + len(handoff.HandoffInstruction)
	for _, turn := range handoff.RecentConversation {
		total += len(turn.Text)
	}
	for _, turn := range handoff.ToolOutcomes {
		total += len(turn.Text)
	}
	for _, note := range handoff.HandoffNotes {
		total += len(note.Message)
	}
	return total
}
```

- [ ] **Step 6: Run the targeted test and confirm it passes**

Run:

```bash
GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/render -run TestBuildHandoffProducesVersionedJSONShape -count=1
```

Expected: PASS.

## Task 2: Preserve Markdown Through Shared Handoff

**Files:**
- Modify: `internal/render/render.go`
- Test: `internal/render/render_test.go`

- [ ] **Step 1: Add a regression test for Markdown output after refactor**

Existing Markdown tests already cover most behavior. Add this focused test:

```go
func TestContextMarkdownUsesSharedHandoffModel(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Initial goal", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: "Latest answer", Kind: core.KindMessage},
	})

	text := Context(detail, ContextOptions{TargetCWD: "/new/project", MaxChars: 3000})

	for _, want := range []string{
		"# AI Session Context",
		"## Session",
		"- ID: codex:abc",
		"- Target CWD: /new/project",
		"## Initial Goal",
		"Initial goal",
		"## Handoff Instruction",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in markdown:\n%s", want, text)
		}
	}
}
```

- [ ] **Step 2: Refactor `Context` to render from `BuildHandoff`**

Replace `Context` with:

```go
func Context(detail core.SessionDetail, opts ContextOptions) string {
	maxChars := opts.MaxChars
	if maxChars <= 0 {
		maxChars = 20000
	}
	text := ContextFromHandoff(BuildHandoff(detail, opts))
	return bound(text, maxChars)
}
```

- [ ] **Step 3: Add `ContextFromHandoff`**

Add this renderer and keep section names identical:

```go
func ContextFromHandoff(handoff HandoffContext) string {
	var b strings.Builder
	b.WriteString("# AI Session Context\n\n")
	b.WriteString("## Session\n\n")
	writeLine(&b, "- ID: %s", handoff.Session.ID)
	writeLine(&b, "- Source: %s", handoff.Session.Source)
	writeLine(&b, "- Original CWD: %s", valueOrUnknown(handoff.Session.OriginalCWD))
	if handoff.Session.TargetCWD != "" {
		writeLine(&b, "- Target CWD: %s", handoff.Session.TargetCWD)
	}
	writeLine(&b, "- Created: %s", timeOrUnknown(handoff.Session.CreatedAt))
	writeLine(&b, "- Updated: %s", timeOrUnknown(handoff.Session.UpdatedAt))
	b.WriteString("\n## Initial Goal\n\n")
	if handoff.InitialGoal.Available {
		b.WriteString(handoff.InitialGoal.Text)
	} else {
		b.WriteString("Unavailable")
	}
	b.WriteString("\n\n## Recent Conversation\n\n")
	if len(handoff.RecentConversation) == 0 && handoff.Truncated {
		b.WriteString("Omitted for size.\n\n")
	} else {
		for _, turn := range handoff.RecentConversation {
			writeLine(&b, "### %s", titleRole(turn.Role))
			b.WriteString("\n")
			b.WriteString(turn.Text)
			b.WriteString("\n\n")
		}
	}
	b.WriteString("## Tool Outcomes\n\n")
	if len(handoff.ToolOutcomes) == 0 && handoff.Truncated {
		b.WriteString("Omitted for size.\n\n")
	} else {
		for _, turn := range handoff.ToolOutcomes {
			writeLine(&b, "### %s", titleKind(turn.Kind))
			b.WriteString("\n")
			b.WriteString(turn.Text)
			b.WriteString("\n\n")
		}
	}
	b.WriteString("## Handoff Notes\n\n")
	for _, note := range handoff.HandoffNotes {
		writeLine(&b, "- %s", note.Message)
		if note.Code == "context_truncated" {
			b.WriteString("- [truncated]\n")
		}
	}
	b.WriteString("\n## Handoff Instruction\n\n")
	b.WriteString(handoff.HandoffInstruction)
	return b.String()
}
```

- [ ] **Step 4: Remove the old direct `buildContext` path after tests pass**

Delete the old `buildContext` function only after `ContextFromHandoff` covers its behavior. Keep shared helper functions such as `writeLine`, `valueOrUnknown`, `timeOrUnknown`, `titleRole`, `titleKind`, `limitText`, and `bound`.

- [ ] **Step 5: Run render package tests**

Run:

```bash
GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/render -count=1
```

Expected: PASS. If `TestContextKeepsHighPrioritySectionsWhenBounded` fails by a small length difference, adjust `BuildHandoff` compact limits, not the test intent.

## Task 3: JSON-Specific Renderer Tests

**Files:**
- Modify: `internal/render/render_test.go`

- [ ] **Step 1: Add tests for notes, missing initial goal, empty arrays, and truncation**

Add:

```go
func TestBuildHandoffStructuredNotesAndMissingGoal(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "<environment_context>\n  <cwd>/tmp/project</cwd>\n</environment_context>", Kind: core.KindMessage},
		{Role: core.RoleTool, Text: strings.Repeat("log\n", 200), Kind: core.KindToolResult, OmittedReason: "tool_output"},
	})

	handoff := BuildHandoff(detail, ContextOptions{MaxChars: 3000})

	if handoff.InitialGoal.Available {
		t.Fatalf("expected unavailable initial goal: %+v", handoff.InitialGoal)
	}
	if handoff.RecentConversation == nil || handoff.ToolOutcomes == nil {
		t.Fatalf("expected empty arrays, got recent=%#v outcomes=%#v", handoff.RecentConversation, handoff.ToolOutcomes)
	}
	codes := map[string]bool{}
	for _, note := range handoff.HandoffNotes {
		codes[note.Code] = true
		if note.Message == "" {
			t.Fatalf("note missing message: %+v", note)
		}
	}
	for _, want := range []string{"setup_boilerplate_skipped", "tool_output_omitted"} {
		if !codes[want] {
			t.Fatalf("expected note code %q in %+v", want, handoff.HandoffNotes)
		}
	}
}

func TestBuildHandoffTruncatesByContentBudget(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Initial goal " + strings.Repeat("long ", 100), Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: strings.Repeat("assistant detail ", 100), Kind: core.KindMessage},
	})

	handoff := BuildHandoff(detail, ContextOptions{MaxChars: 120})

	if !handoff.Truncated {
		t.Fatalf("expected truncated handoff: %+v", handoff)
	}
	if handoff.SchemaVersion == "" || handoff.Session.ID == "" || handoff.HandoffInstruction == "" {
		t.Fatalf("expected core fields preserved: %+v", handoff)
	}
	found := false
	for _, note := range handoff.HandoffNotes {
		if note.Code == "context_truncated" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected truncation note: %+v", handoff.HandoffNotes)
	}
}
```

- [ ] **Step 2: Run the JSON-specific renderer tests**

Run:

```bash
GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/render -run 'TestBuildHandoff' -count=1
```

Expected: PASS.

## Task 4: CLI JSON Wiring

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/service.go`
- Test: `internal/cli/cli_test.go`

- [ ] **Step 1: Extend the CLI service interface**

In `internal/cli/cli.go`, add `ContextHandoff` to `Service`:

```go
type Service interface {
	Doctor() []core.SourceDiagnostic
	List(core.ListOptions) core.ListResult
	Show(string, core.ShowOptions) (core.SessionDetail, error)
	Context(string, core.ContextOptions) (string, error)
	ContextHandoff(string, core.ContextOptions) (render.HandoffContext, error)
}
```

Add the `internal/render` import to `cli.go`.

- [ ] **Step 2: Implement `ContextHandoff` in `internal/cli/service.go`**

Add:

```go
func (s *appService) ContextHandoff(sessionID string, opts core.ContextOptions) (render.HandoffContext, error) {
	if opts.MaxChars <= 0 {
		opts.MaxChars = s.contextLimit
	}
	detail, err := s.Show(sessionID, core.ShowOptions{Mode: core.ModeClean, MaxChars: s.detailLimit})
	if err != nil {
		return render.HandoffContext{}, err
	}
	return render.BuildHandoff(detail, render.ContextOptions{TargetCWD: opts.TargetCWD, MaxChars: opts.MaxChars}), nil
}
```

Then simplify `Context` to use it:

```go
func (s *appService) Context(sessionID string, opts core.ContextOptions) (string, error) {
	handoff, err := s.ContextHandoff(sessionID, opts)
	if err != nil {
		return "", err
	}
	return render.ContextFromHandoff(handoff), nil
}
```

- [ ] **Step 3: Add context JSON flags in `runContext`**

In `runContext`, add:

```go
var jsonOut bool
flags.BoolVar(&jsonOut, "json", false, "write JSON output")
flags.BoolVar(&jsonOut, "j", false, "write JSON output")
```

Before the Markdown path, add:

```go
if jsonOut {
	handoff, err := service.ContextHandoff(sessionID, core.ContextOptions{
		TargetCWD: targetCWD,
		MaxChars:  maxChars,
	})
	if err != nil {
		writeError(stderr, err)
		return 1
	}
	return writeJSON(stdout, handoff)
}
```

- [ ] **Step 4: Update `fakeCLIService` in CLI tests**

Add a field and method:

```go
handoff render.HandoffContext
```

```go
func (f fakeCLIService) ContextHandoff(string, core.ContextOptions) (render.HandoffContext, error) {
	return f.handoff, nil
}
```

Add the `internal/render` import in `internal/cli/cli_test.go`.

- [ ] **Step 5: Run CLI compile check**

Run:

```bash
GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/cli -run TestContextCommandWritesMarkdown -count=1
```

Expected: PASS.

## Task 5: CLI Behavior Tests

**Files:**
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Add `context --json` test**

Add:

```go
func TestContextCommandWritesJSONHandoff(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service := fakeCLIService{handoff: render.HandoffContext{
		SchemaVersion: render.HandoffSchemaVersion,
		Session: render.HandoffSession{
			ID:        "codex:abc",
			Source:    core.SourceCodex,
			NativeID:  "abc",
			TargetCWD: "/new",
		},
		InitialGoal:        render.HandoffInitialGoal{Available: true, Text: "Initial goal"},
		RecentConversation: []core.Turn{},
		ToolOutcomes:       []core.Turn{},
		HandoffNotes:       []render.HandoffNote{{Code: "no_omissions", Message: "No skipped, omitted, or truncated content."}},
		HandoffInstruction: "Continue from this prior AI coding session.",
	}}

	code := RunWithService([]string{"context", "codex:abc", "--target-cwd", "/new", "--json"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	var payload struct {
		SchemaVersion      string          `json:"schema_version"`
		Session            json.RawMessage `json:"session"`
		RecentConversation json.RawMessage `json:"recent_conversation"`
		ToolOutcomes       json.RawMessage `json:"tool_outcomes"`
		HandoffNotes       []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"handoff_notes"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if payload.SchemaVersion != render.HandoffSchemaVersion {
		t.Fatalf("unexpected schema version: %q", payload.SchemaVersion)
	}
	if string(payload.RecentConversation) != "[]" || string(payload.ToolOutcomes) != "[]" {
		t.Fatalf("expected empty arrays, got recent=%s outcomes=%s", payload.RecentConversation, payload.ToolOutcomes)
	}
	if len(payload.HandoffNotes) != 1 || payload.HandoffNotes[0].Code == "" || payload.HandoffNotes[0].Message == "" {
		t.Fatalf("unexpected handoff notes: %+v", payload.HandoffNotes)
	}
}
```

- [ ] **Step 2: Add short alias test**

Add:

```go
func TestContextCommandSupportsJSONShortAlias(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service := fakeCLIService{handoff: render.HandoffContext{
		SchemaVersion:      render.HandoffSchemaVersion,
		Session:            render.HandoffSession{ID: "codex:abc", Source: core.SourceCodex, NativeID: "abc"},
		InitialGoal:        render.HandoffInitialGoal{Available: false},
		RecentConversation: []core.Turn{},
		ToolOutcomes:       []core.Turn{},
		HandoffNotes:       []render.HandoffNote{{Code: "no_omissions", Message: "No skipped, omitted, or truncated content."}},
		HandoffInstruction: "Continue from this prior AI coding session.",
	}}

	code := RunWithService([]string{"context", "codex:abc", "-j"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"schema_version": "context-handoff.v1"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}
```

- [ ] **Step 3: Run CLI tests**

Run:

```bash
GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/cli -count=1
```

Expected: PASS.

## Task 6: Documentation

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`

- [ ] **Step 1: Update English README quick start and command notes**

In `README.md`, add this example after the Markdown `context` example:

```markdown
Generate structured handoff JSON for scripts, Skills, or future MCP adapters:

```bash
ai-history context codex:<session-id> --target-cwd /path/to/project --json
```
```

Add this paragraph near command notes:

```markdown
`show --json` returns normalized session detail. `context --json` returns a
filtered handoff object with `schema_version: "context-handoff.v1"` for
continuing work in another agent or directory.
```

- [ ] **Step 2: Update Chinese README**

In `README.zh-CN.md`, add the equivalent example:

```markdown
为脚本、Skill 或后续 MCP adapter 生成结构化交接 JSON：

```bash
ai-history context codex:<session-id> --target-cwd /path/to/project --json
```
```

Add:

```markdown
`show --json` 返回归一化会话详情。`context --json` 返回用于继续工作的筛选后交接对象，
其中包含 `schema_version: "context-handoff.v1"`。
```

- [ ] **Step 3: Run docs-related smoke check**

Run:

```bash
rg -n "context --json|context-handoff.v1|show --json" README.md README.zh-CN.md
```

Expected: both README files mention `context --json`, `context-handoff.v1`, and the `show --json` distinction.

## Task 7: OpenSpec Task Updates And Verification

**Files:**
- Modify: `openspec/changes/add-context-json-handoff/tasks.md`

- [ ] **Step 1: Mark implemented OpenSpec tasks complete**

Update each completed checkbox in `openspec/changes/add-context-json-handoff/tasks.md` from `- [ ]` to `- [x]` only after the matching implementation and tests pass.

- [ ] **Step 2: Run full Go tests**

Run:

```bash
GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./...
```

Expected: all packages PASS.

- [ ] **Step 3: Run OpenSpec validation**

Run:

```bash
openspec validate add-context-json-handoff --strict
```

Expected:

```text
Change 'add-context-json-handoff' is valid
```

- [ ] **Step 4: Review git diff**

Run:

```bash
git diff --stat
git diff -- openspec/changes/add-context-json-handoff/tasks.md
```

Expected: diff includes implementation, tests, docs, and completed OpenSpec task checkboxes; no unrelated files.

- [ ] **Step 5: Commit after user approval**

After review, commit with a concise Chinese message:

```bash
git add internal/render/render.go internal/render/render_test.go internal/cli/cli.go internal/cli/service.go internal/cli/cli_test.go README.md README.zh-CN.md openspec/changes/add-context-json-handoff/tasks.md
git commit -m "添加 context JSON 交接输出"
```

Do not add automated co-author trailers.

## Self-Review

- Spec coverage: the plan covers schema version, section-shaped JSON, target cwd, empty arrays, structured notes, truncation as content budget, no raw turns, default Markdown preservation, and `show --json` separation.
- Placeholder scan: no unresolved placeholder language or unspecified edge handling remains in the task steps.
- Type consistency: all planned exported names use `render.HandoffContext`, `render.HandoffSession`, `render.HandoffInitialGoal`, `render.HandoffNote`, `render.HandoffSchemaVersion`, `render.BuildHandoff`, and `render.ContextFromHandoff`.
