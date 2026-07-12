package core

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSearchMatchesAndRanksContentCategories(t *testing.T) {
	newer := time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)
	older := newer.Add(-time.Hour)
	service := NewService(map[Source]Reader{
		SourceCodex: searchReader{
			summaries: []SessionSummary{
				searchSummary("title", SourceCodex, "Deploy checklist", "/work", older),
				searchSummary("user", SourceCodex, "User discussion", "/work", newer),
				searchSummary("assistant", SourceCodex, "Assistant discussion", "/work", newer),
				searchSummary("tool", SourceCodex, "Tool discussion", "/work", newer),
			},
			details: map[string]SessionDetail{
				"title":     searchDetail("title"),
				"user":      searchDetail("user", Turn{Role: RoleUser, Kind: KindMessage, Text: "Please DEPLOY this."}),
				"assistant": searchDetail("assistant", Turn{Role: RoleAssistant, Kind: KindMessage, Text: "We can deploy now."}),
				"tool": searchDetail("tool",
					Turn{Role: RoleTool, Kind: KindToolResult, Text: "deploy deploy deploy"},
				),
			},
		},
	})

	result := service.Search(SearchOptions{Query: "deploy", Limit: 10})

	if got, want := len(result.Hits), 4; got != want {
		t.Fatalf("expected %d hits, got %+v", want, result)
	}
	for index, want := range []struct {
		id       string
		score    int
		category SearchMatchCategory
	}{
		{"codex:title", 100, SearchMatchTitle},
		{"codex:user", 30, SearchMatchUser},
		{"codex:assistant", 20, SearchMatchAssistant},
		{"codex:tool", 10, SearchMatchTool},
	} {
		hit := result.Hits[index]
		if hit.Session.ID != want.id || hit.Score != want.score {
			t.Fatalf("hit %d: expected %s score=%d, got %+v", index, want.id, want.score, hit)
		}
		if len(hit.Matches) != 1 || hit.Matches[0].Category != want.category {
			t.Fatalf("hit %d: expected %s match, got %+v", index, want.category, hit.Matches)
		}
	}
}

func TestSearchAccumulatesEachCategoryOnlyOnce(t *testing.T) {
	now := time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)
	service := NewService(map[Source]Reader{
		SourceCodex: searchReader{
			summaries: []SessionSummary{searchSummary("all", SourceCodex, "needle title", "/work", now)},
			details: map[string]SessionDetail{"all": searchDetail("all",
				Turn{Role: RoleUser, Kind: KindMessage, Text: "needle one"},
				Turn{Role: RoleUser, Kind: KindMessage, Text: "needle two"},
				Turn{Role: RoleAssistant, Kind: KindMessage, Text: "needle answer"},
				Turn{Role: RoleTool, Kind: KindToolCall, Text: "needle command"},
			)},
		},
	})

	result := service.Search(SearchOptions{Query: "needle", Limit: 10})

	if got, want := len(result.Hits), 1; got != want {
		t.Fatalf("expected %d hit, got %+v", want, result)
	}
	hit := result.Hits[0]
	if got, want := hit.Score, 160; got != want {
		t.Fatalf("expected capped cumulative score %d, got %+v", want, hit)
	}
	if got, want := len(hit.Matches), 4; got != want {
		t.Fatalf("expected %d categories, got %+v", want, hit.Matches)
	}
}

func TestSearchAppliesFiltersLimitsAndTieBreaksByUpdateTime(t *testing.T) {
	newer := time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)
	older := newer.Add(-time.Hour)
	service := NewService(map[Source]Reader{
		SourceCodex: searchReader{
			summaries: []SessionSummary{
				searchSummary("new", SourceCodex, "Notes", "/work/a", newer),
				searchSummary("old", SourceCodex, "Notes", "/work/a/child", older),
				searchSummary("other", SourceCodex, "Notes", "/other", newer),
			},
			details: map[string]SessionDetail{
				"new":   searchDetail("new", Turn{Role: RoleUser, Kind: KindMessage, Text: "needle"}),
				"old":   searchDetail("old", Turn{Role: RoleUser, Kind: KindMessage, Text: "needle"}),
				"other": searchDetail("other", Turn{Role: RoleUser, Kind: KindMessage, Text: "needle"}),
			},
		},
		SourceClaude: searchReader{
			summaries: []SessionSummary{searchSummary("claude", SourceClaude, "Notes", "/work/a", newer)},
			details:   map[string]SessionDetail{"claude": searchDetail("claude", Turn{Role: RoleUser, Kind: KindMessage, Text: "needle"})},
		},
	})

	result := service.Search(SearchOptions{Query: "needle", Source: SourceCodex, Under: "/work", Limit: 1})

	if got, want := len(result.Hits), 1; got != want {
		t.Fatalf("expected %d hit, got %+v", want, result)
	}
	if got, want := result.Hits[0].Session.ID, "codex:new"; got != want {
		t.Fatalf("expected newer tie-break hit %q, got %q", want, got)
	}
}

func TestSearchReturnsBoundedSnippetsAndPartialSourceDiagnostics(t *testing.T) {
	now := time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)
	longText := strings.Repeat("前", 120) + "needle" + strings.Repeat("后", 120)
	service := NewService(map[Source]Reader{
		SourceCodex: searchReader{
			summaries: []SessionSummary{searchSummary("one", SourceCodex, "Notes", "/work", now)},
			details:   map[string]SessionDetail{"one": searchDetail("one", Turn{Role: RoleUser, Kind: KindMessage, Text: longText})},
		},
		SourceClaude: searchReader{listErr: errors.New("not readable")},
	})

	result := service.Search(SearchOptions{Query: "needle", Limit: 10})

	if len(result.Hits) != 1 {
		t.Fatalf("expected one hit, got %+v", result)
	}
	if got := len([]rune(result.Hits[0].Snippet)); got > 200 {
		t.Fatalf("expected bounded snippet, got %d runes: %q", got, result.Hits[0].Snippet)
	}
	if !strings.Contains(result.Hits[0].Snippet, "needle") {
		t.Fatalf("expected snippet to contain match: %q", result.Hits[0].Snippet)
	}
	if result.Unavailable[SourceClaude] != "not readable" {
		t.Fatalf("expected unavailable source, got %+v", result.Unavailable)
	}
	if result.Diagnostics[SourceClaude].Status != "unavailable" {
		t.Fatalf("expected source diagnostic, got %+v", result.Diagnostics)
	}
}

func TestSearchSnippetKeepsUnicodeCaseFoldedMatch(t *testing.T) {
	text := strings.Repeat("Ω", 1000) + "needle" + strings.Repeat("后", 120)

	snippet, ok := searchText(text, "NEEDLE")

	if !ok {
		t.Fatal("expected case-insensitive Unicode text to match")
	}
	if got := len([]rune(snippet)); got > 200 {
		t.Fatalf("expected bounded snippet, got %d runes", got)
	}
	if !strings.Contains(snippet, "needle") {
		t.Fatalf("expected snippet to keep the matched text, got %q", snippet)
	}
}

func TestSearchReturnsEmptyNonNilHits(t *testing.T) {
	service := NewService(map[Source]Reader{
		SourceCodex: searchReader{},
	})

	result := service.Search(SearchOptions{Query: "missing"})

	if result.Hits == nil {
		t.Fatal("expected non-nil hits slice")
	}
	if len(result.Hits) != 0 || result.TotalReturned != 0 {
		t.Fatalf("expected no hits, got %+v", result)
	}
}

type searchReader struct {
	summaries []SessionSummary
	details   map[string]SessionDetail
	listErr   error
}

func (r searchReader) ListSessions() ([]SessionSummary, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.summaries, nil
}

func (r searchReader) GetSession(nativeID string) (SessionDetail, error) {
	detail, ok := r.details[nativeID]
	if !ok {
		return SessionDetail{}, NewError(ErrSessionNotFound, nativeID)
	}
	return detail, nil
}

func (r searchReader) Doctor() SourceDiagnostic {
	return SourceDiagnostic{Source: SourceCodex, Status: "available"}
}

func searchSummary(nativeID string, source Source, title string, cwd string, updated time.Time) SessionSummary {
	return SessionSummary{
		ID:        MakeSessionID(source, nativeID),
		Source:    source,
		NativeID:  nativeID,
		Title:     title,
		CWD:       cwd,
		UpdatedAt: &updated,
	}
}

func searchDetail(nativeID string, turns ...Turn) SessionDetail {
	return SessionDetail{Summary: SessionSummary{NativeID: nativeID}, Turns: turns}
}
