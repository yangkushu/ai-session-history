package cli

import (
	"github.com/yangkushu/ai-session-history/internal/config"
	"github.com/yangkushu/ai-session-history/internal/core"
	"github.com/yangkushu/ai-session-history/internal/discovery"
	"github.com/yangkushu/ai-session-history/internal/readers"
	"github.com/yangkushu/ai-session-history/internal/render"
)

type appService struct {
	core         *core.Service
	detailLimit  int
	contextLimit int
}

func NewService(configPath string) (Service, error) {
	cfg, err := config.LoadOptional(configPath)
	if err != nil {
		return nil, err
	}
	coreReaders := map[core.Source]core.Reader{}
	for _, source := range []core.Source{core.SourceCodex, core.SourceClaude, core.SourceCursor} {
		sourceConfig := cfg.Sources[string(source)]
		if !sourceConfig.Enabled {
			continue
		}
		roots := append([]string{}, sourceConfig.Paths...)
		if sourceConfig.UseDefaultPaths {
			roots = append(roots, discovery.ResolveRoots(source)...)
		}
		switch source {
		case core.SourceCodex:
			coreReaders[source] = readers.NewCodexStorageReader(roots)
		case core.SourceClaude:
			coreReaders[source] = readers.NewClaudeStorageReader(roots)
		case core.SourceCursor:
			coreReaders[source] = readers.NewCursorStorageReader(roots)
		}
	}
	return &appService{
		core:         core.NewService(coreReaders),
		detailLimit:  cfg.Limits.DetailChars,
		contextLimit: cfg.Limits.ContextChars,
	}, nil
}

func (s *appService) Doctor() []core.SourceDiagnostic {
	return s.core.Doctor()
}

func (s *appService) List(opts core.ListOptions) core.ListResult {
	return s.core.List(opts)
}

func (s *appService) Search(opts core.SearchOptions) core.SearchResult {
	return s.core.Search(opts)
}

func (s *appService) Show(sessionID string, opts core.ShowOptions) (core.SessionDetail, error) {
	if opts.MaxChars <= 0 {
		opts.MaxChars = s.detailLimit
	}
	detail, err := s.core.Show(sessionID, opts)
	if err != nil {
		return core.SessionDetail{}, err
	}
	mode := opts.Mode
	if mode == "" {
		mode = core.ModeClean
	}
	return render.Detail(detail, mode, opts.MaxChars), nil
}

func (s *appService) Context(sessionID string, opts core.ContextOptions) (string, error) {
	if opts.MaxChars <= 0 {
		opts.MaxChars = s.contextLimit
	}
	detail, err := s.Show(sessionID, core.ShowOptions{Mode: core.ModeClean, MaxChars: s.detailLimit})
	if err != nil {
		return "", err
	}
	return render.Context(detail, render.ContextOptions{TargetCWD: opts.TargetCWD, MaxChars: opts.MaxChars}), nil
}
