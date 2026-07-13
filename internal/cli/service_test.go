package cli

import (
	"strings"
	"testing"

	"github.com/yangkushu/ai-session-history/internal/core"
)

func TestAppServiceExportGetsCompleteNormalizedDetailWithoutDetailLimit(t *testing.T) {
	fullText := strings.Repeat("complete normalized detail ", 20)
	detail := core.SessionDetail{
		Summary: core.SessionSummary{ID: "codex:abc", Source: core.SourceCodex, NativeID: "abc"},
		Turns:   []core.Turn{{Role: core.RoleUser, Kind: core.KindMessage, Text: fullText}},
	}
	service := &appService{
		core:        core.NewService(map[core.Source]core.Reader{core.SourceCodex: exportDetailReader{detail: detail}}),
		detailLimit: 10,
	}

	export, err := service.Export("codex:abc", core.ModeRaw)

	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if export.Session.Truncated {
		t.Fatalf("export must not be truncated: %+v", export.Session)
	}
	if got := export.Session.Turns[0].Text; got != fullText {
		t.Fatalf("export used the detail limit: got %d chars, want %d", len(got), len(fullText))
	}
}

type exportDetailReader struct {
	detail core.SessionDetail
}

func (r exportDetailReader) ListSessions() ([]core.SessionSummary, error) {
	return []core.SessionSummary{r.detail.Summary}, nil
}

func (r exportDetailReader) GetSession(nativeID string) (core.SessionDetail, error) {
	if nativeID != r.detail.Summary.NativeID {
		return core.SessionDetail{}, core.NewError(core.ErrSessionNotFound, nativeID)
	}
	return r.detail, nil
}

func (r exportDetailReader) Doctor() core.SourceDiagnostic {
	return core.SourceDiagnostic{Source: r.detail.Summary.Source, Status: "available"}
}
