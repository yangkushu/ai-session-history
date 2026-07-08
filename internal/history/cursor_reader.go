package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type CursorStorageReader struct {
	roots []string
}

func NewCursorStorageReader(roots []string) CursorStorageReader {
	return CursorStorageReader{roots: roots}
}

func (r CursorStorageReader) ListSessions(ctx context.Context) ([]SessionSummary, error) {
	sessions, err := r.sessions(ctx)
	if err != nil {
		return nil, err
	}
	summaries := make([]SessionSummary, 0, len(sessions))
	for _, session := range sessions {
		summaries = append(summaries, session.summary())
	}
	return summaries, nil
}

func (r CursorStorageReader) GetSession(ctx context.Context, nativeID string) (SessionDetail, error) {
	sessions, err := r.sessions(ctx)
	if err != nil {
		return SessionDetail{}, err
	}
	for _, session := range sessions {
		if session.nativeID == nativeID {
			return SessionDetail{Summary: session.summary(), Turns: session.turns}, nil
		}
	}
	return SessionDetail{}, fmt.Errorf("%w: cursor:%s", ErrSessionNotFound, nativeID)
}

func (r CursorStorageReader) Doctor(ctx context.Context) SourceDiagnostic {
	paths, err := r.stateDBPaths()
	if err != nil {
		return SourceDiagnostic{Source: SourceCursor, Status: "unavailable", Code: "source_unavailable", Reason: err.Error()}
	}
	if len(paths) == 0 {
		return SourceDiagnostic{Source: SourceCursor, Status: "unavailable", Code: "source_unavailable", Reason: "no Cursor state.vscdb files found"}
	}
	if _, err := r.sessions(ctx); err != nil {
		return SourceDiagnostic{Source: SourceCursor, Status: "unsupported", Code: "unsupported_format", Path: paths[0], Reason: err.Error()}
	}
	return SourceDiagnostic{Source: SourceCursor, Status: "available", Path: paths[0], Available: true}
}

type cursorSession struct {
	nativeID  string
	title     string
	cwd       string
	createdAt time.Time
	updatedAt time.Time
	turns     []Turn
}

func (s cursorSession) summary() SessionSummary {
	texts := make([]string, 0, len(s.turns))
	for _, turn := range s.turns {
		texts = append(texts, turn.Text)
	}
	title := strings.TrimSpace(s.title)
	if title == "" {
		title = titleFromTurns(s.turns)
	}
	return SessionSummary{
		ID:            MakeSessionID(SourceCursor, s.nativeID),
		Source:        SourceCursor,
		NativeID:      s.nativeID,
		Title:         title,
		Project:       projectFromCWD(s.cwd),
		CWD:           s.cwd,
		CreatedAt:     s.createdAt,
		UpdatedAt:     firstNonZeroTime(s.updatedAt, s.createdAt),
		Preview:       previewText(strings.Join(texts, " "), 160),
		TurnCount:     len(s.turns),
		Available:     true,
		ReaderBackend: "storage",
	}
}

func (r CursorStorageReader) sessions(ctx context.Context) ([]cursorSession, error) {
	paths, err := r.stateDBPaths()
	if err != nil {
		return nil, err
	}
	var out []cursorSession
	var sawSupportedDB bool
	for _, path := range paths {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		sessions, supported, err := cursorSessionsFromStateDB(path)
		if err != nil {
			return nil, err
		}
		if supported {
			sawSupportedDB = true
		}
		out = append(out, sessions...)
	}
	if len(paths) > 0 && !sawSupportedDB {
		return nil, errors.New("Cursor state database does not contain cursorDiskKV or composerHeaders")
	}
	sort.SliceStable(out, func(i, j int) bool {
		return firstNonZeroTime(out[i].updatedAt, out[i].createdAt).After(firstNonZeroTime(out[j].updatedAt, out[j].createdAt))
	})
	return out, nil
}

func (r CursorStorageReader) stateDBPaths() ([]string, error) {
	var paths []string
	for _, root := range r.roots {
		expanded := expandHome(root)
		info, err := os.Stat(expanded)
		if err != nil || !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(expanded, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			if entry.Name() == "state.vscdb" {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func cursorSessionsFromStateDB(path string) ([]cursorSession, bool, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, false, err
	}
	defer db.Close()

	if !tableExists(db, "cursorDiskKV") && !tableExists(db, "composerHeaders") {
		return nil, false, nil
	}

	var sessions []cursorSession
	if tableExists(db, "cursorDiskKV") {
		rows, err := db.Query(`SELECT key, value FROM cursorDiskKV WHERE key LIKE 'composerData:%' AND value IS NOT NULL`)
		if err != nil {
			return nil, true, err
		}
		defer rows.Close()
		for rows.Next() {
			var key string
			var raw []byte
			if err := rows.Scan(&key, &raw); err != nil {
				return nil, true, err
			}
			session, ok := cursorSessionFromComposerJSON(strings.TrimPrefix(key, "composerData:"), raw)
			if ok {
				sessions = append(sessions, session)
			}
		}
		if err := rows.Err(); err != nil {
			return nil, true, err
		}
	}

	if tableExists(db, "composerHeaders") {
		rows, err := db.Query(`SELECT composerId, createdAt, lastUpdatedAt, value FROM composerHeaders WHERE value IS NOT NULL`)
		if err != nil {
			return nil, true, err
		}
		defer rows.Close()
		for rows.Next() {
			var composerID string
			var createdAt, updatedAt sql.NullInt64
			var value sql.NullString
			if err := rows.Scan(&composerID, &createdAt, &updatedAt, &value); err != nil {
				return nil, true, err
			}
			if !value.Valid {
				continue
			}
			session, ok := cursorSessionFromComposerJSON(composerID, []byte(value.String))
			if ok {
				if session.createdAt.IsZero() && createdAt.Valid {
					session.createdAt = timeFromCursorNumber(float64(createdAt.Int64))
				}
				if session.updatedAt.IsZero() && updatedAt.Valid {
					session.updatedAt = timeFromCursorNumber(float64(updatedAt.Int64))
				}
				sessions = append(sessions, session)
			}
		}
		if err := rows.Err(); err != nil {
			return nil, true, err
		}
	}
	return sessions, true, nil
}

func cursorSessionFromComposerJSON(fallbackID string, raw []byte) (cursorSession, bool) {
	var payload map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &payload) != nil {
		return cursorSession{}, false
	}
	nativeID := firstString(payload["composerId"], fallbackID)
	if nativeID == "" || nativeID == "empty-state-draft" {
		return cursorSession{}, false
	}
	turns := turnsFromComposerPayload(payload)
	if len(turns) == 0 && strings.TrimSpace(firstString(payload["text"], payload["richText"])) == "" {
		return cursorSession{}, false
	}
	return cursorSession{
		nativeID:  nativeID,
		title:     firstString(payload["title"], payload["name"]),
		cwd:       cwdFromComposerPayload(payload),
		createdAt: timeFromAny(payload["createdAt"]),
		updatedAt: firstNonZeroTime(timeFromAny(payload["lastUpdatedAt"]), timeFromAny(payload["updatedAt"]), newestTurnTime(turns)),
		turns:     turns,
	}, true
}

func turnsFromComposerPayload(payload map[string]any) []Turn {
	var candidates []any
	for _, key := range []string{"conversation", "messages", "fullConversation"} {
		if value, ok := payload[key]; ok {
			candidates = append(candidates, valuesFromAny(value)...)
		}
	}
	if value, ok := payload["conversationMap"]; ok {
		candidates = append(candidates, valuesFromAny(value)...)
	}

	turns := make([]Turn, 0, len(candidates))
	for _, candidate := range candidates {
		obj, ok := candidate.(map[string]any)
		if !ok {
			continue
		}
		role := roleFromAny(firstString(obj["role"], obj["speaker"], obj["type"]))
		if role == "" {
			continue
		}
		text := textFromMessageObject(obj)
		if strings.TrimSpace(text) == "" {
			continue
		}
		turns = append(turns, Turn{
			Role:      role,
			Text:      text,
			Timestamp: timeFromAny(firstPresent(obj, "timestamp", "createdAt", "time")),
			Kind:      KindMessage,
		})
	}
	sort.SliceStable(turns, func(i, j int) bool {
		if turns[i].Timestamp.IsZero() || turns[j].Timestamp.IsZero() {
			return i < j
		}
		return turns[i].Timestamp.Before(turns[j].Timestamp)
	})
	return turns
}

func valuesFromAny(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		values := make([]any, 0, len(keys))
		for _, key := range keys {
			values = append(values, typed[key])
		}
		return values
	default:
		return nil
	}
}

func textFromMessageObject(obj map[string]any) string {
	for _, key := range []string{"text", "content", "message", "richText"} {
		if text := textFromAny(obj[key]); text != "" {
			return text
		}
	}
	return ""
}

func textFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := textFromAny(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		for _, key := range []string{"text", "content", "value"} {
			if text := textFromAny(typed[key]); text != "" {
				return text
			}
		}
	}
	return ""
}

func cwdFromComposerPayload(payload map[string]any) string {
	for _, key := range []string{"cwd", "workspaceRootPath", "workspaceFolderPath", "projectPath"} {
		if value := firstString(payload[key]); value != "" {
			return value
		}
	}
	context, ok := payload["context"].(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range []string{"workspaceRootPath", "workspaceFolderPath", "projectPath", "cwd"} {
		if value := firstString(context[key]); value != "" {
			return value
		}
	}
	return ""
}

func roleFromAny(value string) TurnRole {
	switch strings.ToLower(value) {
	case "user", "human":
		return RoleUser
	case "assistant", "ai":
		return RoleAssistant
	case "system":
		return RoleSystem
	default:
		return ""
	}
}

func firstString(values ...any) string {
	for _, value := range values {
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func firstPresent(obj map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := obj[key]; ok {
			return value
		}
	}
	return nil
}

func timeFromAny(value any) time.Time {
	switch typed := value.(type) {
	case float64:
		return timeFromCursorNumber(typed)
	case int64:
		return timeFromCursorNumber(float64(typed))
	case string:
		if typed == "" {
			return time.Time{}
		}
		parsed, err := time.Parse(time.RFC3339Nano, strings.ReplaceAll(typed, "Z", "+00:00"))
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func timeFromCursorNumber(value float64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	if value > 1_000_000_000_000 {
		return time.UnixMilli(int64(value)).UTC()
	}
	return time.Unix(int64(value), 0).UTC()
}

func newestTurnTime(turns []Turn) time.Time {
	var newest time.Time
	for _, turn := range turns {
		if turn.Timestamp.After(newest) {
			newest = turn.Timestamp
		}
	}
	return newest
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func tableExists(db *sql.DB, name string) bool {
	var found string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&found)
	return err == nil
}

func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
