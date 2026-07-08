package history

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestServiceListFiltersAndSortsSessions(t *testing.T) {
	old := sessionSummary(SourceCursor, "old", "/tmp/demo", time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC))
	newer := sessionSummary(SourceCursor, "new", "/tmp/demo/sub", time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC))
	other := sessionSummary(SourceClaude, "other", "/tmp/other", time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC))
	service := NewService(Config{}, map[Source]Reader{
		SourceCursor: staticReader{sessions: []SessionSummary{old, newer}},
		SourceClaude: staticReader{sessions: []SessionSummary{other}},
	})

	result := service.List(context.Background(), ListOptions{Under: "/tmp/demo", Limit: 10})

	if len(result.Sessions) != 2 {
		t.Fatalf("List returned %d sessions", len(result.Sessions))
	}
	if result.Sessions[0].NativeID != "new" || result.Sessions[1].NativeID != "old" {
		t.Fatalf("sessions not sorted newest first: %#v", result.Sessions)
	}
}

func TestServiceShowAndContextUseSourcePrefixedID(t *testing.T) {
	detail := SessionDetail{
		Summary: sessionSummary(SourceCursor, "one", "/tmp/demo", time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)),
		Turns: []Turn{
			{Role: RoleUser, Text: "Move this session", Kind: KindMessage},
			{Role: RoleAssistant, Text: "Here is context", Kind: KindMessage},
		},
	}
	service := NewService(Config{}, map[Source]Reader{
		SourceCursor: staticReader{detail: detail},
	})

	got, err := service.Show(context.Background(), "cursor:one")
	if err != nil {
		t.Fatalf("Show returned error: %v", err)
	}
	if got.Summary.ID != "cursor:one" {
		t.Fatalf("Show summary ID = %q", got.Summary.ID)
	}

	contextText, err := service.Context(context.Background(), "cursor:one", ContextOptions{TargetCWD: "/tmp/next", MaxChars: 2000})
	if err != nil {
		t.Fatalf("Context returned error: %v", err)
	}
	if !strings.Contains(contextText, "Move this session") || !strings.Contains(contextText, "Target CWD: /tmp/next") {
		t.Fatalf("unexpected context:\n%s", contextText)
	}
}

func sessionSummary(source Source, nativeID string, cwd string, updated time.Time) SessionSummary {
	return SessionSummary{
		ID:            MakeSessionID(source, nativeID),
		Source:        source,
		NativeID:      nativeID,
		Title:         nativeID,
		Project:       projectFromCWD(cwd),
		CWD:           cwd,
		CreatedAt:     updated,
		UpdatedAt:     updated,
		TurnCount:     1,
		Available:     true,
		ReaderBackend: "storage",
	}
}

type staticReader struct {
	sessions []SessionSummary
	detail   SessionDetail
	err      error
}

func (r staticReader) ListSessions(context.Context) ([]SessionSummary, error) {
	return r.sessions, r.err
}

func (r staticReader) GetSession(context.Context, string) (SessionDetail, error) {
	return r.detail, r.err
}

func (r staticReader) Doctor(context.Context) SourceDiagnostic {
	return SourceDiagnostic{Source: SourceCursor, Available: r.err == nil}
}
