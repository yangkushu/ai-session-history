package core

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Reader interface {
	ListSessions() ([]SessionSummary, error)
	GetSession(nativeID string) (SessionDetail, error)
	Doctor() SourceDiagnostic
}

type SourceDiagnostic struct {
	Source  Source    `json:"source"`
	Status  string    `json:"status"`
	Code    ErrorCode `json:"code,omitempty"`
	Path    string    `json:"path,omitempty"`
	Message string    `json:"message,omitempty"`
}

type Service struct {
	readers map[Source]Reader
}

type ListOptions struct {
	Source Source
	CWD    string
	Under  string
	Limit  int
}

type ListResult struct {
	Sessions      []SessionSummary            `json:"sessions"`
	Diagnostics   map[Source]SourceDiagnostic `json:"diagnostics,omitempty"`
	Unavailable   map[Source]string           `json:"unavailable_sources,omitempty"`
	TotalReturned int                         `json:"total_returned"`
}

type ShowOptions struct {
	Mode     ContentMode
	MaxChars int
}

type ContextOptions struct {
	TargetCWD string
	MaxChars  int
}

func NewService(readers map[Source]Reader) *Service {
	return &Service{readers: readers}
}

func (s *Service) Doctor() []SourceDiagnostic {
	sources := []Source{SourceCodex, SourceClaude, SourceCursor}
	diagnostics := make([]SourceDiagnostic, 0, len(sources))
	for _, source := range sources {
		reader := s.readers[source]
		if reader == nil {
			diagnostics = append(diagnostics, SourceDiagnostic{
				Source:  source,
				Status:  "unavailable",
				Code:    ErrSourceUnavailable,
				Message: "source reader is not configured",
			})
			continue
		}
		diagnostics = append(diagnostics, reader.Doctor())
	}
	return diagnostics
}

func (s *Service) List(opts ListOptions) ListResult {
	result := ListResult{
		Sessions:    []SessionSummary{},
		Diagnostics: map[Source]SourceDiagnostic{},
		Unavailable: map[Source]string{},
	}
	for _, source := range s.sources(opts.Source) {
		reader := s.readers[source]
		if reader == nil {
			result.Unavailable[source] = "source reader is not configured"
			continue
		}
		sessions, err := reader.ListSessions()
		if err != nil {
			result.Unavailable[source] = err.Error()
			if appErr, ok := err.(*AppError); ok {
				result.Diagnostics[source] = SourceDiagnostic{
					Source:  source,
					Status:  "unavailable",
					Code:    appErr.Code,
					Path:    appErr.Path,
					Message: appErr.Message,
				}
			}
			continue
		}
		for _, session := range sessions {
			if !matchesCWD(session.CWD, opts.CWD) || !matchesUnder(session.CWD, opts.Under) {
				continue
			}
			result.Sessions = append(result.Sessions, session)
		}
	}
	sort.Slice(result.Sessions, func(i, j int) bool {
		return summaryTime(result.Sessions[i]).After(summaryTime(result.Sessions[j]))
	})
	if opts.Limit > 0 && len(result.Sessions) > opts.Limit {
		result.Sessions = result.Sessions[:opts.Limit]
	}
	result.TotalReturned = len(result.Sessions)
	if len(result.Diagnostics) == 0 {
		result.Diagnostics = nil
	}
	if len(result.Unavailable) == 0 {
		result.Unavailable = nil
	}
	return result
}

func (s *Service) Show(sessionID string, opts ShowOptions) (SessionDetail, error) {
	source, nativeID, err := ParseSessionID(sessionID)
	if err != nil {
		return SessionDetail{}, err
	}
	reader := s.readers[source]
	if reader == nil {
		return SessionDetail{}, NewError(ErrSourceUnavailable, string(source)+" source reader is not configured")
	}
	detail, err := reader.GetSession(nativeID)
	if err != nil {
		if IsCode(err, ErrSessionNotFound) {
			return SessionDetail{}, NewError(ErrSessionNotFound, sessionID)
		}
		return SessionDetail{}, err
	}
	return detail, nil
}

func (s *Service) Context(sessionID string, opts ContextOptions) (string, error) {
	return "", NewError(ErrReaderUnavailable, "context renderer is not configured")
}

func (s *Service) sources(source Source) []Source {
	if source != "" {
		return []Source{source}
	}
	return []Source{SourceCodex, SourceClaude, SourceCursor}
}

func matchesCWD(cwd string, filter string) bool {
	if filter == "" {
		return true
	}
	if cwd == "" {
		return false
	}
	return cleanPath(cwd) == cleanPath(filter)
}

func matchesUnder(cwd string, filter string) bool {
	if filter == "" {
		return true
	}
	if cwd == "" {
		return false
	}
	cleanCWD := cleanPath(cwd)
	cleanFilter := cleanPath(filter)
	return cleanCWD == cleanFilter || strings.HasPrefix(cleanCWD, cleanFilter+string(filepath.Separator))
}

func cleanPath(path string) string {
	if path == "" {
		return ""
	}
	if expanded, ok := strings.CutPrefix(path, "~/"); ok {
		path = filepath.Join("~", expanded)
	}
	return filepath.Clean(path)
}

func summaryTime(summary SessionSummary) time.Time {
	if summary.UpdatedAt != nil {
		return *summary.UpdatedAt
	}
	if summary.CreatedAt != nil {
		return *summary.CreatedAt
	}
	return time.Time{}
}
