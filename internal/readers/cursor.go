package readers

import (
	"os"
	"path/filepath"

	"github.com/yangkushu/ai-session-history/internal/core"
)

type CursorStorageReader struct {
	roots []string
}

func NewCursorStorageReader(roots []string) *CursorStorageReader {
	return &CursorStorageReader{roots: roots}
}

func (r *CursorStorageReader) Doctor() core.SourceDiagnostic {
	dbPath := r.firstStateDB()
	if dbPath == "" {
		return core.SourceDiagnostic{
			Source:  core.SourceCursor,
			Status:  "unavailable",
			Code:    core.ErrSourceUnavailable,
			Message: "no Cursor state.vscdb found",
		}
	}
	return core.SourceDiagnostic{
		Source:  core.SourceCursor,
		Status:  "unsupported_format",
		Code:    core.ErrUnsupportedFormat,
		Path:    dbPath,
		Message: "Cursor latest fixture validation is pending",
	}
}

func (r *CursorStorageReader) ListSessions() ([]core.SessionSummary, error) {
	if dbPath := r.firstStateDB(); dbPath != "" {
		return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, errCursorPending{})
	}
	return nil, core.NewError(core.ErrSourceUnavailable, "no Cursor state.vscdb found")
}

func (r *CursorStorageReader) GetSession(nativeID string) (core.SessionDetail, error) {
	if dbPath := r.firstStateDB(); dbPath != "" {
		return core.SessionDetail{}, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, errCursorPending{})
	}
	return core.SessionDetail{}, core.NewError(core.ErrSessionNotFound, "Cursor session not found: "+nativeID)
}

func (r *CursorStorageReader) firstStateDB() string {
	for _, root := range r.roots {
		var found string
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || found != "" {
				return nil
			}
			if !entry.IsDir() && entry.Name() == "state.vscdb" {
				found = path
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}

type errCursorPending struct{}

func (errCursorPending) Error() string {
	return "Cursor latest fixture validation is pending"
}
