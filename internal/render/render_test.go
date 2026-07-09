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
