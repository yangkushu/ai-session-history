package history

import (
	"strings"
	"testing"
	"time"
)

func TestParseSessionIDRequiresKnownSourceAndNativeID(t *testing.T) {
	source, nativeID, err := ParseSessionID("cursor:abc-123")
	if err != nil {
		t.Fatalf("ParseSessionID returned error: %v", err)
	}
	if source != SourceCursor || nativeID != "abc-123" {
		t.Fatalf("ParseSessionID = %q, %q", source, nativeID)
	}

	for _, input := range []string{"missing-prefix", "cursor:", "unknown:id"} {
		if _, _, err := ParseSessionID(input); err == nil {
			t.Fatalf("ParseSessionID(%q) returned nil error", input)
		}
	}
}

func TestRenderContextPreservesInitialGoalAndRecentConversation(t *testing.T) {
	detail := SessionDetail{
		Summary: SessionSummary{
			ID:        "cursor:session-1",
			Source:    SourceCursor,
			NativeID:  "session-1",
			Title:     "Cursor work",
			CWD:       "/Users/alice/Workspace/demo",
			Project:   "demo",
			CreatedAt: time.Date(2026, 7, 8, 1, 2, 3, 0, time.UTC),
			UpdatedAt: time.Date(2026, 7, 8, 1, 3, 3, 0, time.UTC),
		},
		Turns: []Turn{
			{Role: RoleUser, Text: "Initial goal"},
			{Role: RoleAssistant, Text: "Middle answer"},
			{Role: RoleUser, Text: "Recent ask"},
			{Role: RoleAssistant, Text: "Recent answer"},
		},
	}

	out := BuildContext(detail, ContextOptions{TargetCWD: "/tmp/next", MaxChars: 2000})

	for _, want := range []string{
		"# AI Session Context",
		"ID: cursor:session-1",
		"Original CWD: /Users/alice/Workspace/demo",
		"Target CWD: /tmp/next",
		"## Initial Goal",
		"Initial goal",
		"Recent ask",
		"Recent answer",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("context output missing %q:\n%s", want, out)
		}
	}
}
