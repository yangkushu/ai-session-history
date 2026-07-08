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
