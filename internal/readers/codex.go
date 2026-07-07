package readers

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yangkushu/ai-session-history/internal/core"
)

type CodexStorageReader struct {
	roots []string
}

type codexThreadRow struct {
	ID          string
	RolloutPath string
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
	CWD         string
	Title       string
	Preview     string
}

func NewCodexStorageReader(roots []string) *CodexStorageReader {
	return &CodexStorageReader{roots: roots}
}

func (r *CodexStorageReader) ListSessions() ([]core.SessionSummary, error) {
	rows, err := r.threadRows()
	if err != nil {
		return nil, err
	}
	sessions := make([]core.SessionSummary, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, r.summaryFromRow(row, nil))
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessionTime(sessions[i]).After(sessionTime(sessions[j]))
	})
	return sessions, nil
}

func (r *CodexStorageReader) GetSession(nativeID string) (core.SessionDetail, error) {
	rows, err := r.threadRows()
	if err != nil {
		return core.SessionDetail{}, err
	}
	for _, row := range rows {
		if row.ID != nativeID {
			continue
		}
		turns, err := readCodexRollout(row.RolloutPath)
		if err != nil {
			return core.SessionDetail{}, err
		}
		return core.SessionDetail{
			Summary: r.summaryFromRow(row, turns),
			Turns:   turns,
		}, nil
	}
	return core.SessionDetail{}, core.NewError(core.ErrSessionNotFound, "Codex session not found: "+nativeID)
}

func (r *CodexStorageReader) threadRows() ([]codexThreadRow, error) {
	var rows []codexThreadRow
	for _, root := range r.roots {
		dbPath := filepath.Join(root, "state_5.sqlite")
		if _, err := os.Stat(dbPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, core.WrapSourceError(core.ErrPermissionDenied, core.SourceCodex, dbPath, err)
		}
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			return nil, core.WrapSourceError(core.ErrSourceUnavailable, core.SourceCodex, dbPath, err)
		}
		dbRows, queryErr := db.Query(`
			SELECT id, rollout_path, created_at_ms, updated_at_ms, cwd, title, preview
			FROM threads
		`)
		if queryErr != nil {
			_ = db.Close()
			return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCodex, dbPath, queryErr)
		}
		for dbRows.Next() {
			var row codexThreadRow
			var createdMS, updatedMS sql.NullInt64
			if err := dbRows.Scan(&row.ID, &row.RolloutPath, &createdMS, &updatedMS, &row.CWD, &row.Title, &row.Preview); err != nil {
				_ = dbRows.Close()
				_ = db.Close()
				return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCodex, dbPath, err)
			}
			row.CreatedAt = timeFromMillis(createdMS)
			row.UpdatedAt = timeFromMillis(updatedMS)
			rows = append(rows, row)
		}
		if err := dbRows.Err(); err != nil {
			_ = dbRows.Close()
			_ = db.Close()
			return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCodex, dbPath, err)
		}
		_ = dbRows.Close()
		_ = db.Close()
	}
	return rows, nil
}

func (r *CodexStorageReader) summaryFromRow(row codexThreadRow, turns []core.Turn) core.SessionSummary {
	title := row.Title
	if title == "" && turns != nil {
		title = titleFromTurns(turns, "Untitled Session")
	}
	if title == "" {
		title = "Untitled Session"
	}
	return core.SessionSummary{
		ID:            core.MakeSessionID(core.SourceCodex, row.ID),
		Source:        core.SourceCodex,
		NativeID:      row.ID,
		Title:         title,
		Project:       projectFromCWD(row.CWD),
		CWD:           row.CWD,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		Preview:       row.Preview,
		TurnCount:     len(turns),
		Available:     true,
		ReaderBackend: core.BackendStorage,
	}
}

func readCodexRollout(path string) ([]core.Turn, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, core.WrapSourceError(core.ErrPermissionDenied, core.SourceCodex, path, err)
	}
	defer file.Close()

	var turns []core.Turn
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record codexRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceCodex, path, err)
		}
		if record.Type != "response_item" || record.Payload.Type != "message" {
			continue
		}
		role := core.TurnRole(record.Payload.Role)
		if role != core.RoleUser && role != core.RoleAssistant && role != core.RoleSystem {
			continue
		}
		text := contentText(record.Payload.Content)
		if text == "" {
			continue
		}
		turns = append(turns, core.Turn{
			Role:      role,
			Text:      text,
			Timestamp: timeFromISO(record.Timestamp),
			Kind:      core.KindMessage,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, core.WrapSourceError(core.ErrPermissionDenied, core.SourceCodex, path, err)
	}
	return turns, nil
}

type codexRecord struct {
	Timestamp string       `json:"timestamp"`
	Type      string       `json:"type"`
	Payload   codexPayload `json:"payload"`
}

type codexPayload struct {
	Type    string          `json:"type"`
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func contentText(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return ""
	}
	textParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
	}
	return strings.Join(textParts, "\n")
}

func timeFromMillis(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	t := time.UnixMilli(value.Int64).UTC()
	return &t
}

func timeFromISO(value string) *time.Time {
	if value == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil
	}
	return &t
}

func sessionTime(summary core.SessionSummary) time.Time {
	if summary.UpdatedAt != nil {
		return *summary.UpdatedAt
	}
	if summary.CreatedAt != nil {
		return *summary.CreatedAt
	}
	return time.Time{}
}
