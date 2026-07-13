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

func TestListReturnsEmptySessionsSlice(t *testing.T) {
	service := NewService(map[Source]Reader{
		SourceCodex: fakeReader{summaries: []SessionSummary{}},
	})

	result := service.List(ListOptions{Under: "/missing"})

	if result.Sessions == nil {
		t.Fatal("expected empty sessions slice, got nil")
	}
	if len(result.Sessions) != 0 {
		t.Fatalf("expected no sessions, got %+v", result.Sessions)
	}
}

func TestShowReturnsSessionNotFoundForMissingNativeID(t *testing.T) {
	service := NewService(map[Source]Reader{SourceCodex: fakeReader{}})
	_, err := service.Show("codex:missing", ShowOptions{Mode: ModeClean, MaxChars: 100})
	if !IsCode(err, ErrSessionNotFound) {
		t.Fatalf("expected session_not_found, got %v", err)
	}
}

func TestDoctorPreservesIndependentSourceDiagnostics(t *testing.T) {
	deniedPath := "/history/claude/projects"
	service := NewService(map[Source]Reader{
		SourceCodex: fakeReader{diagnostic: SourceDiagnostic{Source: SourceCodex, Status: "available", Path: "/history/codex/state_5.sqlite"}},
		SourceClaude: fakeReader{diagnostic: SourceDiagnostic{
			Source: SourceClaude, Status: "unavailable", Code: ErrPermissionDenied, Path: deniedPath,
		}},
		SourceCursor: fakeReader{diagnostic: SourceDiagnostic{
			Source: SourceCursor, Status: "unsupported_format", Code: ErrUnsupportedFormat, Path: "/history/cursor/state.vscdb",
		}},
	})

	diagnostics := service.Doctor()
	if len(diagnostics) != 3 {
		t.Fatalf("want three independent diagnostics, got %+v", diagnostics)
	}
	if diagnostics[0].Source != SourceCodex || diagnostics[0].Status != "available" {
		t.Fatalf("Codex diagnostic hidden or changed: %+v", diagnostics[0])
	}
	if diagnostics[1].Source != SourceClaude || diagnostics[1].Code != ErrPermissionDenied || diagnostics[1].Path != deniedPath {
		t.Fatalf("Claude permission diagnostic hidden or changed: %+v", diagnostics[1])
	}
	if diagnostics[2].Source != SourceCursor || diagnostics[2].Code != ErrUnsupportedFormat {
		t.Fatalf("Cursor diagnostic hidden or changed: %+v", diagnostics[2])
	}
}

type fakeReader struct {
	summaries  []SessionSummary
	details    map[string]SessionDetail
	diagnostic SourceDiagnostic
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
	if f.diagnostic.Source != "" {
		return f.diagnostic
	}
	return SourceDiagnostic{Source: SourceCodex, Status: "available"}
}
