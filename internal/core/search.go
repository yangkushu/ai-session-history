package core

import (
	"sort"
	"strings"
	"unicode"
)

const maxSearchSnippetRunes = 200

type Searcher interface {
	Search(SearchOptions) SearchResult
}

type scanSearcher struct {
	readers map[Source]Reader
}

func NewScanSearcher(readers map[Source]Reader) Searcher {
	return scanSearcher{readers: readers}
}

func (s scanSearcher) Search(opts SearchOptions) SearchResult {
	result := SearchResult{
		Hits:        []SearchHit{},
		Diagnostics: map[Source]SourceDiagnostic{},
		Unavailable: map[Source]string{},
	}
	for _, source := range searchSources(opts.Source) {
		reader := s.readers[source]
		if reader == nil {
			result.Unavailable[source] = "source reader is not configured"
			continue
		}
		summaries, err := reader.ListSessions()
		if err != nil {
			result.Unavailable[source] = err.Error()
			result.Diagnostics[source] = searchDiagnostic(source, err)
			continue
		}
		for _, summary := range summaries {
			if !matchesCWD(summary.CWD, opts.CWD) || !matchesUnder(summary.CWD, opts.Under) {
				continue
			}
			detail, err := reader.GetSession(summary.NativeID)
			if err != nil {
				continue
			}
			if hit, ok := searchSessionDetail(summary, detail, opts.Query); ok {
				result.Hits = append(result.Hits, hit)
			}
		}
	}
	sort.Slice(result.Hits, func(i, j int) bool {
		if result.Hits[i].Score != result.Hits[j].Score {
			return result.Hits[i].Score > result.Hits[j].Score
		}
		leftTime := summaryTime(result.Hits[i].Session)
		rightTime := summaryTime(result.Hits[j].Session)
		if !leftTime.Equal(rightTime) {
			return leftTime.After(rightTime)
		}
		return result.Hits[i].Session.ID < result.Hits[j].Session.ID
	})
	if opts.Limit > 0 && len(result.Hits) > opts.Limit {
		result.Hits = result.Hits[:opts.Limit]
	}
	result.TotalReturned = len(result.Hits)
	if len(result.Diagnostics) == 0 {
		result.Diagnostics = nil
	}
	if len(result.Unavailable) == 0 {
		result.Unavailable = nil
	}
	return result
}

func searchSources(source Source) []Source {
	if source != "" {
		return []Source{source}
	}
	return []Source{SourceCodex, SourceClaude, SourceCursor}
}

func searchDiagnostic(source Source, err error) SourceDiagnostic {
	diagnostic := SourceDiagnostic{
		Source:  source,
		Status:  "unavailable",
		Message: err.Error(),
	}
	if appErr, ok := err.(*AppError); ok {
		diagnostic.Code = appErr.Code
		diagnostic.Path = appErr.Path
	}
	return diagnostic
}

func searchSessionDetail(summary SessionSummary, detail SessionDetail, query string) (SearchHit, bool) {
	hit := SearchHit{Session: summary, Matches: []SearchMatch{}}
	if snippet, ok := searchText(summary.Title, query); ok {
		hit.Score += 100
		hit.Matches = append(hit.Matches, SearchMatch{Category: SearchMatchTitle, Snippet: snippet})
		hit.Snippet = snippet
	}
	seen := map[SearchMatchCategory]bool{}
	for _, turn := range detail.Turns {
		category, weight, ok := searchCategory(turn)
		if !ok || seen[category] {
			continue
		}
		snippet, ok := searchText(turn.Text, query)
		if !ok {
			continue
		}
		seen[category] = true
		hit.Score += weight
		hit.Matches = append(hit.Matches, SearchMatch{Category: category, Snippet: snippet})
		if hit.Snippet == "" {
			hit.Snippet = snippet
		}
	}
	return hit, hit.Score > 0
}

func searchCategory(turn Turn) (SearchMatchCategory, int, bool) {
	if turn.Kind == KindToolCall || turn.Kind == KindToolResult {
		return SearchMatchTool, 10, true
	}
	if turn.Kind != KindMessage {
		return "", 0, false
	}
	switch turn.Role {
	case RoleUser:
		return SearchMatchUser, 30, true
	case RoleAssistant:
		return SearchMatchAssistant, 20, true
	default:
		return "", 0, false
	}
}

func searchText(text string, query string) (string, bool) {
	lowerQuery := lowerRunes(query)
	if len(lowerQuery) == 0 {
		return "", false
	}
	matchStart := runeIndex(lowerRunes(text), lowerQuery)
	if matchStart < 0 {
		return "", false
	}
	return searchSnippet(text, matchStart, len(lowerQuery)), true
}

func lowerRunes(text string) []rune {
	return []rune(strings.Map(unicode.ToLower, text))
}

func runeIndex(text []rune, query []rune) int {
	for start := 0; start+len(query) <= len(text); start++ {
		matched := true
		for offset, queryRune := range query {
			if text[start+offset] != queryRune {
				matched = false
				break
			}
		}
		if matched {
			return start
		}
	}
	return -1
}

func searchSnippet(text string, matchStart int, matchRunes int) string {
	runes := []rune(text)
	start := matchStart - 80
	if start < 0 {
		start = 0
	}
	end := start + maxSearchSnippetRunes
	if end > len(runes) {
		end = len(runes)
	}
	if matchStart+matchRunes > end {
		start = matchStart + matchRunes - maxSearchSnippetRunes
		if start < 0 {
			start = 0
		}
		end = start + maxSearchSnippetRunes
		if end > len(runes) {
			end = len(runes)
		}
	}
	return string(runes[start:end])
}
