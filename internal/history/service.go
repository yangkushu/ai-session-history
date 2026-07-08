package history

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type Reader interface {
	ListSessions(context.Context) ([]SessionSummary, error)
	GetSession(context.Context, string) (SessionDetail, error)
	Doctor(context.Context) SourceDiagnostic
}

type Config struct {
	Sources map[Source]SourceConfig
	Limits  Limits
}

type SourceConfig struct {
	Enabled         bool
	UseDefaultPaths bool
	Paths           []string
}

type Limits struct {
	DetailChars  int
	ContextChars int
}

type Service struct {
	config  Config
	readers map[Source]Reader
}

type ListOptions struct {
	Source Source
	CWD    string
	Under  string
	Limit  int
}

type ListResult struct {
	Sessions           []SessionSummary
	UnavailableSources map[Source]string
}

func NewService(config Config, readers map[Source]Reader) Service {
	return Service{config: config, readers: readers}
}

func (s Service) List(ctx context.Context, options ListOptions) ListResult {
	limit := options.Limit
	if limit <= 0 {
		limit = 50
	}
	result := ListResult{Sessions: []SessionSummary{}, UnavailableSources: map[Source]string{}}
	for _, source := range s.sources(options.Source) {
		if !s.sourceEnabled(source) {
			continue
		}
		reader := s.readers[source]
		if reader == nil {
			result.UnavailableSources[source] = "source_unavailable"
			continue
		}
		sessions, err := reader.ListSessions(ctx)
		if err != nil {
			result.UnavailableSources[source] = err.Error()
			continue
		}
		for _, session := range sessions {
			if matchesCWD(session.CWD, options.CWD) && matchesUnder(session.CWD, options.Under) {
				result.Sessions = append(result.Sessions, session)
			}
		}
	}
	sort.SliceStable(result.Sessions, func(i, j int) bool {
		left := result.Sessions[i].UpdatedAt
		if left.IsZero() {
			left = result.Sessions[i].CreatedAt
		}
		right := result.Sessions[j].UpdatedAt
		if right.IsZero() {
			right = result.Sessions[j].CreatedAt
		}
		return left.After(right)
	})
	if len(result.Sessions) > limit {
		result.Sessions = result.Sessions[:limit]
	}
	return result
}

func (s Service) Show(ctx context.Context, sessionID string) (SessionDetail, error) {
	source, nativeID, err := ParseSessionID(sessionID)
	if err != nil {
		return SessionDetail{}, err
	}
	reader := s.readers[source]
	if reader == nil {
		return SessionDetail{}, fmt.Errorf("source_unavailable: %s", source)
	}
	detail, err := reader.GetSession(ctx, nativeID)
	if err != nil {
		return SessionDetail{}, err
	}
	return detail, nil
}

func (s Service) Context(ctx context.Context, sessionID string, options ContextOptions) (string, error) {
	detail, err := s.Show(ctx, sessionID)
	if err != nil {
		return "", err
	}
	return BuildContext(detail, options), nil
}

func (s Service) Doctor(ctx context.Context) []SourceDiagnostic {
	diagnostics := make([]SourceDiagnostic, 0, 3)
	for _, source := range []Source{SourceCodex, SourceClaude, SourceCursor} {
		if !s.sourceEnabled(source) {
			diagnostics = append(diagnostics, SourceDiagnostic{Source: source, Status: "disabled"})
			continue
		}
		reader := s.readers[source]
		if reader == nil {
			diagnostics = append(diagnostics, SourceDiagnostic{
				Source: source,
				Status: "unavailable",
				Code:   "source_unavailable",
				Reason: "source reader missing",
			})
			continue
		}
		diagnostics = append(diagnostics, reader.Doctor(ctx))
	}
	return diagnostics
}

func (s Service) sources(only Source) []Source {
	if only != "" {
		return []Source{only}
	}
	return []Source{SourceCodex, SourceClaude, SourceCursor}
}

func (s Service) sourceEnabled(source Source) bool {
	cfg, ok := s.config.Sources[source]
	return !ok || cfg.Enabled
}

func matchesCWD(sessionCWD, filter string) bool {
	if filter == "" {
		return true
	}
	return cleanPath(sessionCWD) == cleanPath(filter)
}

func matchesUnder(sessionCWD, filter string) bool {
	if filter == "" {
		return true
	}
	session := cleanPath(sessionCWD)
	parent := cleanPath(filter)
	return session == parent || strings.HasPrefix(session, parent+string(filepath.Separator))
}

func cleanPath(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}
