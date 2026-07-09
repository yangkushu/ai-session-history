package render

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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

func TestRenderPreservesConciseToolOutcomes(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Run tests", Kind: core.KindMessage},
		{Role: core.RoleTool, Text: "go test ./... passed", Kind: core.KindToolResult},
		{Role: core.RoleTool, Text: strings.Repeat("line\n", 200), Kind: core.KindToolResult, OmittedReason: "tool_output"},
		{Role: core.RoleTool, Text: "go test failed: missing symbol", Kind: core.KindError},
	})

	clean := Detail(detail, core.ModeClean, 1000)
	summary := Detail(detail, core.ModeSummary, 1000)
	context := Context(detail, ContextOptions{MaxChars: 2000})

	if clean.Turns[1].Text != "go test ./... passed" {
		t.Fatalf("expected concise tool result in clean mode, got %q", clean.Turns[1].Text)
	}
	if clean.Turns[2].Text != "[omitted: tool_output]" {
		t.Fatalf("expected noisy tool output omitted, got %q", clean.Turns[2].Text)
	}
	if !strings.Contains(summary.Turns[1].Text, "go test ./... passed") {
		t.Fatalf("expected concise tool result in summary mode, got %q", summary.Turns[1].Text)
	}
	if !strings.Contains(context, "go test ./... passed") || !strings.Contains(context, "go test failed") {
		t.Fatalf("expected useful tool outcomes in context:\n%s", context)
	}
	if strings.Contains(context, strings.Repeat("line\n", 20)) {
		t.Fatalf("context should not include large raw tool output:\n%s", context)
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

func TestContextUsesStableHandoffSections(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Initial goal", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: "Latest answer", Kind: core.KindMessage},
		{Role: core.RoleTool, Text: "go test ./... passed", Kind: core.KindToolResult},
	})

	text := Context(detail, ContextOptions{TargetCWD: "/new/project", MaxChars: 3000})

	assertInOrder(t, text, []string{
		"# AI Session Context",
		"## Session",
		"## Initial Goal",
		"## Recent Conversation",
		"## Tool Outcomes",
		"## Handoff Notes",
		"## Handoff Instruction",
	})
}

func TestContextSkipsSetupBoilerplateBeforeInitialGoal(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "# AGENTS.md instructions\n\n<INSTRUCTIONS>\nUse Chinese.\n</INSTRUCTIONS>", Kind: core.KindMessage},
		{Role: core.RoleUser, Text: "请帮我修复 context 初始目标清洗", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: "我会先补测试。", Kind: core.KindMessage},
	})

	text := Context(detail, ContextOptions{MaxChars: 3000})

	if !strings.Contains(text, "请帮我修复 context 初始目标清洗") {
		t.Fatalf("expected meaningful user task as initial goal:\n%s", text)
	}
	if strings.Contains(sectionText(text, "## Initial Goal", "## Recent Conversation"), "AGENTS.md instructions") {
		t.Fatalf("setup boilerplate should not be initial goal:\n%s", text)
	}
	if strings.Contains(sectionText(text, "## Recent Conversation", "## Handoff Notes"), "AGENTS.md instructions") {
		t.Fatalf("setup boilerplate should be excluded from recent conversation:\n%s", text)
	}
	if !strings.Contains(text, "- Skipped setup boilerplate turns: 1") {
		t.Fatalf("expected skipped boilerplate note:\n%s", text)
	}
}

func TestContextMarksMissingInitialGoalAfterBoilerplateFiltering(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "<environment_context>\n  <cwd>/tmp/project</cwd>\n</environment_context>", Kind: core.KindMessage},
		{Role: core.RoleUser, Text: "   ", Kind: core.KindMessage},
	})

	text := Context(detail, ContextOptions{MaxChars: 2000})

	if !strings.Contains(sectionText(text, "## Initial Goal", "## Recent Conversation"), "Unavailable") {
		t.Fatalf("expected unavailable initial goal, got:\n%s", text)
	}
	if strings.Contains(text, "<environment_context>") {
		t.Fatalf("setup boilerplate should be filtered from context:\n%s", text)
	}
}

func TestContextPreservesRecentConversationWithoutBoilerplate(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "# AGENTS.md instructions\n\n<INSTRUCTIONS>\nInjected setup.\n</INSTRUCTIONS>", Kind: core.KindMessage},
		{Role: core.RoleUser, Text: "Initial goal", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: "Early answer", Kind: core.KindMessage},
		{Role: core.RoleUser, Text: "Latest question", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: "Latest answer", Kind: core.KindMessage},
	})

	text := Context(detail, ContextOptions{MaxChars: 3000})
	recent := sectionText(text, "## Recent Conversation", "## Handoff Notes")

	for _, want := range []string{"Latest question", "Latest answer"} {
		if !strings.Contains(recent, want) {
			t.Fatalf("expected %q in recent conversation:\n%s", want, text)
		}
	}
	if strings.Contains(recent, "AGENTS.md instructions") {
		t.Fatalf("setup boilerplate should be excluded from recent conversation:\n%s", text)
	}
}

func TestContextNotesUsefulToolOutcomesAndOmittedNoisyOutput(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Run verification", Kind: core.KindMessage},
		{Role: core.RoleTool, Text: "go test ./... passed", Kind: core.KindToolResult},
		{Role: core.RoleTool, Text: strings.Repeat("download log\n", 200), Kind: core.KindToolResult, OmittedReason: "tool_output"},
		{Role: core.RoleTool, Text: "openspec validate failed: missing scenario", Kind: core.KindError},
	})

	text := Context(detail, ContextOptions{MaxChars: 3000})
	outcomes := sectionText(text, "## Tool Outcomes", "## Handoff Notes")
	notes := sectionText(text, "## Handoff Notes", "## Handoff Instruction")

	for _, want := range []string{"go test ./... passed", "openspec validate failed"} {
		if !strings.Contains(outcomes, want) {
			t.Fatalf("expected useful tool outcome %q:\n%s", want, text)
		}
	}
	if strings.Contains(outcomes, strings.Repeat("download log\n", 10)) {
		t.Fatalf("context should not include noisy raw tool output:\n%s", text)
	}
	if !strings.Contains(notes, "- Omitted noisy tool output turns: 1") {
		t.Fatalf("expected omitted tool output note:\n%s", text)
	}
}

func TestContextKeepsHighPrioritySectionsWhenBounded(t *testing.T) {
	turns := []core.Turn{
		{Role: core.RoleUser, Text: "Initial goal " + strings.Repeat("这是一段较长的目标描述。", 20), Kind: core.KindMessage},
	}
	for i := 0; i < 20; i++ {
		turns = append(turns, core.Turn{Role: core.RoleAssistant, Text: strings.Repeat("long assistant answer ", 20), Kind: core.KindMessage})
	}
	detail := fixtureDetail(turns)

	text := Context(detail, ContextOptions{MaxChars: 700})

	for _, want := range []string{"# AI Session Context", "## Session", "## Initial Goal", "Initial goal", "## Handoff Notes", "## Handoff Instruction", "[truncated]"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected bounded context to keep high priority section %q:\n%s", want, text)
		}
	}
	if len(text) > 700 {
		t.Fatalf("expected bounded context length <= 700, got %d", len(text))
	}
}

func TestContextTruncationPreservesUTF8(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "请帮我检查 context 输出", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: strings.Repeat("这是一段较长的中文回答。", 80), Kind: core.KindMessage},
	})

	text := Context(detail, ContextOptions{MaxChars: 1000})

	if !utf8.ValidString(text) {
		t.Fatalf("expected valid UTF-8 after context truncation:\n%s", text)
	}
	if strings.ContainsRune(text, '\uFFFD') {
		t.Fatalf("context should not contain replacement characters after truncation:\n%s", text)
	}
}

func fixtureDetail(turns []core.Turn) core.SessionDetail {
	now := time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC)
	return core.SessionDetail{
		Summary: core.SessionSummary{
			ID:            "codex:abc",
			Source:        core.SourceCodex,
			NativeID:      "abc",
			Title:         "Test Session",
			Project:       "project",
			CWD:           "/old/project",
			CreatedAt:     &now,
			UpdatedAt:     &now,
			Preview:       "preview",
			TurnCount:     len(turns),
			Available:     true,
			ReaderBackend: core.BackendStorage,
		},
		Turns: turns,
	}
}

func assertInOrder(t *testing.T, text string, wants []string) {
	t.Helper()
	last := -1
	for _, want := range wants {
		idx := strings.Index(text, want)
		if idx == -1 {
			t.Fatalf("expected %q in text:\n%s", want, text)
		}
		if idx <= last {
			t.Fatalf("expected %q after prior section in text:\n%s", want, text)
		}
		last = idx
	}
}

func sectionText(text, start, end string) string {
	startIdx := strings.Index(text, start)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(start)
	endIdx := strings.Index(text[startIdx:], end)
	if endIdx == -1 {
		return text[startIdx:]
	}
	return text[startIdx : startIdx+endIdx]
}
