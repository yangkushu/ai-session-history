package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/yangzeqi/ai-session-history/internal/history"
)

func TestRunListCommandOutputsJSON(t *testing.T) {
	svc := history.NewService(history.Config{}, map[history.Source]history.Reader{
		history.SourceCursor: cliFakeReader{
			sessions: []history.SessionSummary{{
				ID:            "cursor:one",
				Source:        history.SourceCursor,
				NativeID:      "one",
				Title:         "Cursor session",
				CWD:           "/tmp/demo",
				Project:       "demo",
				UpdatedAt:     time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
				Available:     true,
				ReaderBackend: "storage",
			}},
		},
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"list", "--source", "cursor", "--json"}, &stdout, &stderr, svc)

	if code != 0 {
		t.Fatalf("run returned %d, stderr: %s", code, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, `"id": "cursor:one"`) {
		t.Fatalf("stdout missing session JSON:\n%s", got)
	}
}

type cliFakeReader struct {
	sessions []history.SessionSummary
	detail   history.SessionDetail
	err      error
}

func (r cliFakeReader) ListSessions(context.Context) ([]history.SessionSummary, error) {
	return r.sessions, r.err
}

func (r cliFakeReader) GetSession(context.Context, string) (history.SessionDetail, error) {
	return r.detail, r.err
}

func (r cliFakeReader) Doctor(context.Context) history.SourceDiagnostic {
	return history.SourceDiagnostic{Source: history.SourceCursor, Available: r.err == nil}
}
