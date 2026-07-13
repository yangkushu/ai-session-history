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
	stat  statFunc
}

func NewCursorStorageReader(roots []string) *CursorStorageReader {
	return &CursorStorageReader{roots: roots, stat: os.Stat}
}

func (r *CursorStorageReader) Doctor() core.SourceDiagnostic {
	dbPath, err := r.stateDBPath()
	if err != nil {
		return diagnosticFromError(core.SourceCursor, err)
	}
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
	hasComposerHeaders := cursorTableExists(db, "composerHeaders")
	hasMacComposerData := cursorHasComposerData(db)
	if !hasComposerHeaders && !hasMacComposerData {
		return core.SourceDiagnostic{
			Source: core.SourceCursor, Status: "unsupported_format",
			Code: core.ErrUnsupportedFormat, Path: dbPath,
			Message: "composerHeaders table and composerData entries not found",
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

	sessions := make([]core.SessionSummary, 0, 16)
	seen := map[string]struct{}{}
	if cursorTableExists(db, "composerHeaders") {
		rows, err := db.Query(`SELECT composerId, value FROM composerHeaders`)
		if err != nil {
			return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
		}
		defer rows.Close()

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
			seen[composerID] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
		}
	}

	if cursorTableExists(db, "cursorDiskKV") {
		macSessions, err := r.listMacComposerData(db, dbPath, seen)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, macSessions...)
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

	if cursorTableExists(db, "composerHeaders") {
		var raw []byte
		err = db.QueryRow(`SELECT value FROM composerHeaders WHERE composerId = ?`, nativeID).Scan(&raw)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return core.SessionDetail{}, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
		}
		if err == nil {
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
	}

	if cursorTableExists(db, "cursorDiskKV") {
		detail, ok, err := r.macComposerDetail(db, dbPath, nativeID)
		if err != nil {
			return core.SessionDetail{}, err
		}
		if ok {
			return detail, nil
		}
	}
	return core.SessionDetail{}, core.NewError(core.ErrSessionNotFound, "Cursor session not found: "+nativeID)
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

func (r *CursorStorageReader) stateDBPath() (string, error) {
	for _, root := range r.roots {
		candidate := filepath.Join(root, "globalStorage", "state.vscdb")
		_, err := r.stat(candidate)
		switch {
		case err == nil:
			return candidate, nil
		case os.IsNotExist(err):
			continue
		default:
			return "", pathInspectionError(core.SourceCursor, candidate, err)
		}
	}
	return "", nil
}

func (r *CursorStorageReader) openDB() (*sql.DB, string, error) {
	dbPath, err := r.stateDBPath()
	if err != nil {
		return nil, "", err
	}
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

func (r *CursorStorageReader) listMacComposerData(db *sql.DB, dbPath string, seen map[string]struct{}) ([]core.SessionSummary, error) {
	rows, err := db.Query(`SELECT key, value FROM cursorDiskKV WHERE key LIKE 'composerData:%' AND value IS NOT NULL`)
	if err != nil {
		return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}
	defer rows.Close()

	var sessions []core.SessionSummary
	for rows.Next() {
		var key string
		var raw []byte
		if err := rows.Scan(&key, &raw); err != nil {
			return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
		}
		fallbackID := strings.TrimPrefix(key, "composerData:")
		if _, ok := seen[fallbackID]; ok {
			continue
		}
		summary, ok := cursorMacSummaryFromRaw(fallbackID, raw)
		if ok {
			sessions = append(sessions, summary)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}
	return sessions, nil
}

func (r *CursorStorageReader) macComposerDetail(db *sql.DB, dbPath, nativeID string) (core.SessionDetail, bool, error) {
	var raw []byte
	err := db.QueryRow(`SELECT value FROM cursorDiskKV WHERE key = ? AND value IS NOT NULL`, "composerData:"+nativeID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return core.SessionDetail{}, false, nil
	}
	if err != nil {
		return core.SessionDetail{}, false, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCursor, dbPath, err)
	}
	summary, turns, ok := cursorMacSessionFromRaw(nativeID, raw)
	if !ok {
		return core.SessionDetail{}, false, nil
	}
	return core.SessionDetail{Summary: summary, Turns: turns}, true, nil
}

func cursorMacSummaryFromRaw(fallbackID string, raw []byte) (core.SessionSummary, bool) {
	summary, _, ok := cursorMacSessionFromRaw(fallbackID, raw)
	return summary, ok
}

func cursorMacSessionFromRaw(fallbackID string, raw []byte) (core.SessionSummary, []core.Turn, bool) {
	var value cursorMacComposerValue
	if len(raw) == 0 || json.Unmarshal(raw, &value) != nil {
		return core.SessionSummary{}, nil, false
	}
	nativeID := strings.TrimSpace(value.ComposerID)
	if nativeID == "" {
		nativeID = fallbackID
	}
	if nativeID == "" || nativeID == "empty-state-draft" {
		return core.SessionSummary{}, nil, false
	}
	turns := cursorMacTurns(value.Conversation)
	if len(turns) == 0 && strings.TrimSpace(value.Text+value.RichText) == "" {
		return core.SessionSummary{}, nil, false
	}
	title := strings.TrimSpace(value.Title)
	if title == "" {
		title = strings.TrimSpace(value.Name)
	}
	if title == "" {
		title = titleFromTurns(turns, "Untitled Composer")
	}
	texts := make([]string, 0, len(turns))
	for _, turn := range turns {
		texts = append(texts, turn.Text)
	}
	cwd := value.Context.WorkspaceRootPath
	if cwd == "" {
		cwd = value.Context.WorkspaceFolderPath
	}
	summary := core.SessionSummary{
		ID:            core.MakeSessionID(core.SourceCursor, nativeID),
		Source:        core.SourceCursor,
		NativeID:      nativeID,
		Title:         title,
		Project:       projectFromCWD(cwd),
		CWD:           cwd,
		CreatedAt:     cursorTimeFromMillis(value.CreatedAt),
		UpdatedAt:     cursorTimeFromMillis(firstNonZeroFloat(value.LastUpdatedAt, value.UpdatedAt)),
		Preview:       previewText(strings.Join(texts, " "), 160),
		TurnCount:     len(turns),
		Available:     true,
		ReaderBackend: core.BackendStorage,
	}
	return summary, turns, true
}

func cursorMacTurns(messages []cursorMacMessage) []core.Turn {
	turns := make([]core.Turn, 0, len(messages))
	for _, message := range messages {
		role := cursorMacRole(message.Role)
		if role == "" {
			continue
		}
		text := strings.TrimSpace(message.Text)
		if text == "" {
			continue
		}
		turns = append(turns, core.Turn{
			Role:      role,
			Text:      text,
			Timestamp: cursorTimeFromMillis(message.Timestamp),
			Kind:      core.KindMessage,
		})
	}
	sort.SliceStable(turns, func(i, j int) bool {
		return cursorTurnTime(turns[i]).Before(cursorTurnTime(turns[j]))
	})
	return turns
}

func cursorMacRole(role string) core.TurnRole {
	switch strings.ToLower(role) {
	case "user", "human":
		return core.RoleUser
	case "assistant", "ai":
		return core.RoleAssistant
	case "system":
		return core.RoleSystem
	default:
		return ""
	}
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

type cursorMacComposerValue struct {
	ComposerID    string  `json:"composerId"`
	Title         string  `json:"title"`
	Name          string  `json:"name"`
	Text          string  `json:"text"`
	RichText      string  `json:"richText"`
	CreatedAt     float64 `json:"createdAt"`
	UpdatedAt     float64 `json:"updatedAt"`
	LastUpdatedAt float64 `json:"lastUpdatedAt"`
	Context       struct {
		WorkspaceRootPath   string `json:"workspaceRootPath"`
		WorkspaceFolderPath string `json:"workspaceFolderPath"`
	} `json:"context"`
	Conversation []cursorMacMessage `json:"conversation"`
}

type cursorMacMessage struct {
	Role      string  `json:"role"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

func cursorTableExists(db *sql.DB, name string) bool {
	var found string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&found)
	return err == nil
}

func cursorHasComposerData(db *sql.DB) bool {
	if !cursorTableExists(db, "cursorDiskKV") {
		return false
	}
	var key string
	err := db.QueryRow(`SELECT key FROM cursorDiskKV WHERE key LIKE 'composerData:%' LIMIT 1`).Scan(&key)
	return err == nil
}

func firstNonZeroFloat(values ...float64) float64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
