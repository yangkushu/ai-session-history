package render

import (
	"encoding/json"
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

func TestContextMarkdownUsesSharedHandoffModel(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Initial goal", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: "Latest answer", Kind: core.KindMessage},
	})

	text := ContextFromHandoff(BuildHandoff(detail, ContextOptions{TargetCWD: "/new/project", MaxChars: 3000}))

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

func TestBuildHandoffStructuredNotesAndMissingGoal(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "# AGENTS.md instructions\n\n<INSTRUCTIONS>\nUse Chinese.\n</INSTRUCTIONS>", Kind: core.KindMessage},
		{Role: core.RoleUser, Text: "   ", Kind: core.KindMessage},
		{Role: core.RoleTool, Text: strings.Repeat("download log\n", 200), Kind: core.KindToolResult, OmittedReason: "tool_output"},
	})

	handoff := BuildHandoff(detail, ContextOptions{MaxChars: 3000})

	if handoff.InitialGoal.Available || handoff.InitialGoal.Text != "" {
		t.Fatalf("expected missing initial goal after setup filtering: %+v", handoff.InitialGoal)
	}
	assertHandoffNote(t, handoff, "setup_boilerplate_skipped", "Skipped setup boilerplate turns: 2")
	assertHandoffNote(t, handoff, "tool_output_omitted", "Omitted noisy tool output turns: 1")
	if len(handoff.RecentConversation) != 0 {
		t.Fatalf("expected no recent conversation after setup filtering: %+v", handoff.RecentConversation)
	}
	if len(handoff.ToolOutcomes) != 0 {
		t.Fatalf("expected omitted noisy tool output to stay out of outcomes: %+v", handoff.ToolOutcomes)
	}
}

func TestBuildHandoffUsesRealGoalAfterSetupBoilerplate(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "<environment_context>\n  <cwd>/tmp/project</cwd>\n</environment_context>", Kind: core.KindMessage},
		{Role: core.RoleUser, Text: "Implement structured handoff", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: "I will add tests first.", Kind: core.KindMessage},
	})

	handoff := BuildHandoff(detail, ContextOptions{MaxChars: 3000})

	if !handoff.InitialGoal.Available || handoff.InitialGoal.Text != "Implement structured handoff" {
		t.Fatalf("expected real user goal after setup filtering: %+v", handoff.InitialGoal)
	}
	assertHandoffNote(t, handoff, "setup_boilerplate_skipped", "Skipped setup boilerplate turns: 1")
	for _, turn := range handoff.RecentConversation {
		if strings.Contains(turn.Text, "environment_context") {
			t.Fatalf("setup boilerplate should be excluded from recent conversation: %+v", handoff.RecentConversation)
		}
	}
}

func TestBuildHandoffTruncatesByContentBudget(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Initial goal " + strings.Repeat("long ", 80), Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: strings.Repeat("assistant detail ", 80), Kind: core.KindMessage},
		{Role: core.RoleTool, Text: "go test ./... passed", Kind: core.KindToolResult},
	})

	handoff := BuildHandoff(detail, ContextOptions{TargetCWD: "/new/project", MaxChars: 20})

	if !handoff.Truncated {
		t.Fatalf("expected handoff to be marked truncated: %+v", handoff)
	}
	if handoff.SchemaVersion != HandoffSchemaVersion || handoff.Session.ID != "codex:abc" || handoff.Session.TargetCWD != "/new/project" {
		t.Fatalf("expected core fields preserved: %+v", handoff)
	}
	if handoff.HandoffInstruction == "" {
		t.Fatal("expected handoff instruction to be preserved")
	}
	if handoff.RecentConversation == nil || len(handoff.RecentConversation) != 0 {
		t.Fatalf("expected recent conversation empty slice under tiny budget: %+v", handoff.RecentConversation)
	}
	if handoff.ToolOutcomes == nil || len(handoff.ToolOutcomes) != 0 {
		t.Fatalf("expected tool outcomes empty slice under tiny budget: %+v", handoff.ToolOutcomes)
	}
	if strings.TrimSpace(handoff.InitialGoal.Text) == "" {
		t.Fatalf("expected initial goal text fragment when instruction and notes exceed budget: %+v", handoff.InitialGoal)
	}
	assertHandoffNote(t, handoff, "context_truncated", "Context truncated to --max-chars.")
	if got := handoffContentBytes(handoff); got <= 0 {
		t.Fatalf("expected content budget helper to count preserved text, got %d", got)
	}
}

func TestBuildHandoffBudgetIgnoresMarkdownOverhead(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Goal", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: "Answer", Kind: core.KindMessage},
		{Role: core.RoleTool, Text: "ok", Kind: core.KindToolResult},
	})
	maxChars := 220

	handoff := BuildHandoff(detail, ContextOptions{TargetCWD: "/new/project", MaxChars: maxChars})

	if handoffContentBytes(handoff) > maxChars {
		t.Fatalf("test setup expected content budget within max chars, got %d > %d", handoffContentBytes(handoff), maxChars)
	}
	if len(ContextFromHandoff(handoff)) <= maxChars {
		t.Fatalf("test setup expected markdown overhead to exceed max chars, got %d <= %d", len(ContextFromHandoff(handoff)), maxChars)
	}
	if handoff.Truncated {
		t.Fatalf("handoff should not be truncated by markdown overhead: %+v", handoff)
	}
	if len(handoff.RecentConversation) == 0 || len(handoff.ToolOutcomes) == 0 {
		t.Fatalf("handoff should preserve sections within content budget: recent=%+v outcomes=%+v", handoff.RecentConversation, handoff.ToolOutcomes)
	}
}

func TestContextFromHandoffDoesNotOmitEmptyToolOutcomesWhenTruncated(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Initial goal", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: strings.Repeat("assistant detail ", 80), Kind: core.KindMessage},
	})

	handoff := BuildHandoff(detail, ContextOptions{MaxChars: 20})
	text := ContextFromHandoff(handoff)
	outcomes := sectionText(text, "## Tool Outcomes", "## Handoff Notes")

	if !handoff.Truncated {
		t.Fatalf("test setup expected truncated handoff: %+v", handoff)
	}
	if len(handoff.ToolOutcomes) != 0 {
		t.Fatalf("test setup expected no tool outcomes: %+v", handoff.ToolOutcomes)
	}
	if strings.Contains(outcomes, "Omitted for size.") {
		t.Fatalf("empty tool outcomes should not be reported as omitted:\n%s", text)
	}
}

func TestContextMarkdownCompressionMarksOmittedSections(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Goal", Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: strings.Repeat("assistant detail ", 18), Kind: core.KindMessage},
		{Role: core.RoleTool, Text: "ok", Kind: core.KindToolResult},
	})
	opts := ContextOptions{MaxChars: 700}
	handoff := BuildHandoff(detail, opts)
	if handoff.Truncated || handoffContentBytes(handoff) > opts.MaxChars || len(ContextFromHandoff(handoff)) <= opts.MaxChars {
		t.Fatalf("test setup expected only markdown overhead to exceed budget: content=%d markdown=%d handoff=%+v", handoffContentBytes(handoff), len(ContextFromHandoff(handoff)), handoff)
	}

	text := Context(detail, opts)
	recent := sectionText(text, "## Recent Conversation", "## Tool Outcomes")
	outcomes := sectionText(text, "## Tool Outcomes", "## Handoff Notes")

	if !strings.Contains(recent, "Omitted for size.") && !strings.Contains(outcomes, "Omitted for size.") {
		t.Fatalf("expected markdown compression to mark non-empty omitted sections:\n%s", text)
	}
}

func TestContextFromHandoffPreservesOmittedSectionsAfterJSONRoundTrip(t *testing.T) {
	handoff := HandoffContext{
		SchemaVersion: HandoffSchemaVersion,
		Session: HandoffSession{
			ID: "codex:abc",
		},
		InitialGoal:               HandoffInitialGoal{Available: true, Text: "Goal"},
		RecentConversationOmitted: true,
		HandoffNotes:              []HandoffNote{{Code: "context_truncated", Message: "Context truncated to --max-chars."}},
		HandoffInstruction:        "Continue from this prior AI coding session.",
		Truncated:                 true,
	}
	payload, err := json.Marshal(handoff)
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	var decoded HandoffContext
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal handoff: %v", err)
	}

	text := ContextFromHandoff(decoded)
	recent := sectionText(text, "## Recent Conversation", "## Tool Outcomes")

	if !strings.Contains(recent, "Omitted for size.") {
		t.Fatalf("expected omitted state to survive JSON round trip:\n%s", text)
	}
}

func TestBuildHandoffTinyBudgetPreservesAvailableInitialGoalText(t *testing.T) {
	detail := fixtureDetail([]core.Turn{
		{Role: core.RoleUser, Text: "Initial goal " + strings.Repeat("long ", 80), Kind: core.KindMessage},
		{Role: core.RoleAssistant, Text: strings.Repeat("assistant detail ", 80), Kind: core.KindMessage},
	})

	handoff := BuildHandoff(detail, ContextOptions{MaxChars: 20})

	if !handoff.Truncated {
		t.Fatalf("expected handoff to be marked truncated: %+v", handoff)
	}
	if !handoff.InitialGoal.Available {
		t.Fatalf("expected initial goal to remain available: %+v", handoff.InitialGoal)
	}
	if strings.TrimSpace(handoff.InitialGoal.Text) == "" {
		t.Fatalf("expected tiny budget to preserve an initial goal fragment: %+v", handoff.InitialGoal)
	}
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

func assertHandoffNote(t *testing.T, handoff HandoffContext, code string, message string) {
	t.Helper()
	for _, note := range handoff.HandoffNotes {
		if note.Code == code && note.Message == message {
			return
		}
	}
	t.Fatalf("expected handoff note %q/%q in %+v", code, message, handoff.HandoffNotes)
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
