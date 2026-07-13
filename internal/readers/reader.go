package readers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/yangkushu/ai-session-history/internal/core"
)

type statFunc func(string) (os.FileInfo, error)
type readDirFunc func(string) ([]os.DirEntry, error)

func diagnosticFromError(source core.Source, err error) core.SourceDiagnostic {
	diagnostic := core.SourceDiagnostic{
		Source: source, Status: "unavailable",
		Code: core.ErrSourceUnavailable, Message: err.Error(),
	}
	var appErr *core.AppError
	if errors.As(err, &appErr) {
		diagnostic.Code = appErr.Code
		diagnostic.Path = appErr.Path
	}
	return diagnostic
}

func pathInspectionError(source core.Source, path string, err error) error {
	code := core.ErrSourceUnavailable
	if os.IsPermission(err) {
		code = core.ErrPermissionDenied
	}
	return core.WrapSourceError(code, source, path, err)
}

type StorageReader interface {
	ListSessions() ([]core.SessionSummary, error)
	GetSession(nativeID string) (core.SessionDetail, error)
}

func projectFromCWD(cwd string) string {
	if cwd == "" {
		return ""
	}
	return filepath.Base(cwd)
}

func previewText(text string, limit int) string {
	collapsed := strings.Join(strings.Fields(text), " ")
	if len(collapsed) <= limit {
		return collapsed
	}
	prefix := strings.TrimSpace(collapsed[:limit])
	if index := strings.LastIndex(prefix, " "); index > 0 {
		prefix = prefix[:index]
	}
	return prefix + "..."
}

func titleFromTurns(turns []core.Turn, fallback string) string {
	for _, turn := range turns {
		if turn.Role == core.RoleUser && strings.TrimSpace(turn.Text) != "" {
			return previewText(turn.Text, 80)
		}
	}
	return fallback
}
