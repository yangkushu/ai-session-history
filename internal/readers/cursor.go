package readers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yangkushu/ai-session-history/internal/core"
)

type CursorStorageReader struct {
	roots []string
}

func NewCursorStorageReader(roots []string) *CursorStorageReader {
	return &CursorStorageReader{roots: roots}
}

func (r *CursorStorageReader) Doctor() core.SourceDiagnostic {
	dbPath := r.stateDBPath()
	if dbPath == "" {
		return core.SourceDiagnostic{
			Source:  core.SourceCursor,
			Status:  "unavailable",
			Code:    core.ErrSourceUnavailable,
			Message: "no Cursor state.vscdb found",
		}
	}
	db, err := sql.Open("sqlite", cursorDBURI(dbPath))
	if err != nil {
		return core.SourceDiagnostic{
			Source: core.SourceCursor, Status: "unsupported_format",
			Code: core.ErrUnsupportedFormat, Path: dbPath,
			Message: err.Error(),
		}
	}
	defer db.Close()
	var name string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='composerHeaders'`).Scan(&name)
	if err != nil {
		return core.SourceDiagnostic{
			Source: core.SourceCursor, Status: "unsupported_format",
			Code: core.ErrUnsupportedFormat, Path: dbPath,
			Message: "composerHeaders table not found",
		}
	}
	return core.SourceDiagnostic{Source: core.SourceCursor, Status: "available", Path: dbPath}
}

func (r *CursorStorageReader) ListSessions() ([]core.SessionSummary, error) {
	db, dbPath, err := r.openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT composerId, value FROM composerHeaders`)
	if err != nil {
		return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}
	defer rows.Close()

	sessions := make([]core.SessionSummary, 0, 16)
	for rows.Next() {
		var composerID string
		var raw []byte
		if err := rows.Scan(&composerID, &raw); err != nil {
			return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
		}
		var value cursorComposerValue
		if err := json.Unmarshal(raw, &value); err != nil {
			continue
		}
		if value.IsArchived {
			continue
		}
		sessions = append(sessions, cursorSummary(composerID, value))
	}
	if err := rows.Err(); err != nil {
		return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessionTime(sessions[i]).After(sessionTime(sessions[j]))
	})
	return sessions, nil
}

func (r *CursorStorageReader) GetSession(nativeID string) (core.SessionDetail, error) {
	db, dbPath, err := r.openDB()
	if err != nil {
		return core.SessionDetail{}, err
	}
	defer db.Close()

	var raw []byte
	err = db.QueryRow(`SELECT value FROM composerHeaders WHERE composerId = ?`, nativeID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return core.SessionDetail{}, core.NewError(core.ErrSessionNotFound, "Cursor session not found: "+nativeID)
	}
	if err != nil {
		return core.SessionDetail{}, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}

	var value cursorComposerValue
	_ = json.Unmarshal(raw, &value)

	turns, err := r.readTurns(db, dbPath, nativeID)
	if err != nil {
		return core.SessionDetail{}, err
	}
	summary := cursorSummary(nativeID, value)
	summary.TurnCount = len(turns)
	return core.SessionDetail{Summary: summary, Turns: turns}, nil
}

func (r *CursorStorageReader) readTurns(db *sql.DB, dbPath, composerID string) ([]core.Turn, error) {
	prefix := "bubbleId:" + composerID + ":"
	rows, err := db.Query(
		`SELECT value FROM cursorDiskKV WHERE key >= ? AND key < ?`,
		prefix, prefix+"\U0010FFFF",
	)
	if err != nil {
		return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}
	defer rows.Close()

	var turns []core.Turn
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
		}
		var bubble cursorBubble
		if err := json.Unmarshal(raw, &bubble); err != nil {
			continue
		}
		if strings.TrimSpace(bubble.Text) == "" {
			continue
		}
		role := cursorBubbleRole(bubble.Type)
		if role == "" {
			continue
		}
		turns = append(turns, core.Turn{
			Role:      role,
			Text:      bubble.Text,
			Timestamp: timeFromISO(bubble.CreatedAt),
			Kind:      core.KindMessage,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}
	sort.SliceStable(turns, func(i, j int) bool {
		return cursorTurnTime(turns[i]).Before(cursorTurnTime(turns[j]))
	})
	return turns, nil
}

func (r *CursorStorageReader) stateDBPath() string {
	for _, root := range r.roots {
		candidate := filepath.Join(root, "globalStorage", "state.vscdb")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func (r *CursorStorageReader) openDB() (*sql.DB, string, error) {
	dbPath := r.stateDBPath()
	if dbPath == "" {
		return nil, "", core.NewError(core.ErrSourceUnavailable, "no Cursor state.vscdb found")
	}
	db, err := sql.Open("sqlite", cursorDBURI(dbPath))
	if err != nil {
		return nil, dbPath, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}
	return db, dbPath, nil
}

func cursorDBURI(dbPath string) string {
	return "file:" + dbPath + "?immutable=1"
}

func cursorBubbleRole(t float64) core.TurnRole {
	switch t {
	case 1:
		return core.RoleUser
	case 2:
		return core.RoleAssistant
	default:
		return ""
	}
}

func cursorTurnTime(turn core.Turn) time.Time {
	if turn.Timestamp != nil {
		return *turn.Timestamp
	}
	return time.Time{}
}

func cursorSummary(composerID string, value cursorComposerValue) core.SessionSummary {
	title := value.Name
	if strings.TrimSpace(title) == "" {
		title = "Untitled Composer"
	}
	cwd := value.WorkspaceIdentifier.URI.Path
	return core.SessionSummary{
		ID:            core.MakeSessionID(core.SourceCursor, composerID),
		Source:        core.SourceCursor,
		NativeID:      composerID,
		Title:         title,
		Project:       projectFromCWD(cwd),
		CWD:           cwd,
		CreatedAt:     cursorTimeFromMillis(value.CreatedAt),
		UpdatedAt:     cursorTimeFromMillis(value.LastUpdatedAt),
		Available:     true,
		ReaderBackend: core.BackendStorage,
	}
}

func cursorTimeFromMillis(ms float64) *time.Time {
	if ms <= 0 {
		return nil
	}
	t := time.UnixMilli(int64(ms)).UTC()
	return &t
}

type cursorComposerValue struct {
	Name                string  `json:"name"`
	IsArchived          bool    `json:"isArchived"`
	CreatedAt           float64 `json:"createdAt"`
	LastUpdatedAt       float64 `json:"lastUpdatedAt"`
	WorkspaceIdentifier struct {
		ID  string `json:"id"`
		URI struct {
			Path string `json:"path"`
		} `json:"uri"`
	} `json:"workspaceIdentifier"`
}

type cursorBubble struct {
	Type      float64 `json:"type"`
	Text      string  `json:"text"`
	CreatedAt string  `json:"createdAt"`
}
