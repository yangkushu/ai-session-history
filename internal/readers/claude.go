package readers

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yangkushu/ai-session-history/internal/core"
)

type ClaudeStorageReader struct {
	roots   []string
	readDir readDirFunc
	stat    statFunc
}

func NewClaudeStorageReader(roots []string) *ClaudeStorageReader {
	return &ClaudeStorageReader{roots: roots, readDir: os.ReadDir, stat: os.Stat}
}

func (r *ClaudeStorageReader) Doctor() core.SourceDiagnostic {
	files, err := r.sessionFiles()
	if err != nil {
		return diagnosticFromError(core.SourceClaude, err)
	}
	if len(files) > 0 {
		return core.SourceDiagnostic{Source: core.SourceClaude, Status: "available"}
	}
	return core.SourceDiagnostic{
		Source:  core.SourceClaude,
		Status:  "unavailable",
		Code:    core.ErrSourceUnavailable,
		Message: "no Claude Code project JSONL sessions found",
	}
}

func (r *ClaudeStorageReader) ListSessions() ([]core.SessionSummary, error) {
	files, err := r.sessionFiles()
	if err != nil {
		return nil, err
	}
	sessions := make([]core.SessionSummary, 0, len(files))
	for _, path := range files {
		rows, err := readClaudeRows(path)
		if err != nil {
			// Skip files that cannot be parsed (e.g. corrupt or truncated) so one
			// bad file does not hide the rest of the source's sessions.
			continue
		}
		turns := claudeTurnsFromRows(rows)
		sessions = append(sessions, claudeSummaryFromRows(path, rows, turns))
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessionTime(sessions[i]).After(sessionTime(sessions[j]))
	})
	return sessions, nil
}

func (r *ClaudeStorageReader) GetSession(nativeID string) (core.SessionDetail, error) {
	files, err := r.sessionFiles()
	if err != nil {
		return core.SessionDetail{}, err
	}
	for _, path := range files {
		rows, err := readClaudeRows(path)
		if err != nil {
			continue
		}
		if claudeSessionID(rows, path) != nativeID {
			continue
		}
		turns := claudeTurnsFromRows(rows)
		return core.SessionDetail{
			Summary: claudeSummaryFromRows(path, rows, turns),
			Turns:   turns,
		}, nil
	}
	return core.SessionDetail{}, core.NewError(core.ErrSessionNotFound, "Claude Code session not found: "+nativeID)
}

func (r *ClaudeStorageReader) sessionFiles() ([]string, error) {
	var files []string
	for _, root := range r.roots {
		projectsPath := filepath.Join(root, "projects")
		projects, err := r.readDir(projectsPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, pathInspectionError(core.SourceClaude, projectsPath, err)
		}
		for _, project := range projects {
			isProjectDir := project.IsDir()
			projectPath := filepath.Join(projectsPath, project.Name())
			if !isProjectDir && (project.Type() == 0 || project.Type()&os.ModeSymlink != 0) {
				info, err := r.stat(projectPath)
				if err != nil {
					if os.IsNotExist(err) {
						continue
					}
					return nil, pathInspectionError(core.SourceClaude, projectPath, err)
				}
				isProjectDir = info.IsDir()
			}
			if !isProjectDir {
				continue
			}
			entries, err := r.readDir(projectPath)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, pathInspectionError(core.SourceClaude, projectPath, err)
			}
			for _, entry := range entries {
				if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
					continue
				}
				files = append(files, filepath.Join(projectPath, entry.Name()))
			}
		}
	}
	sort.Strings(files)
	return files, nil
}

func readClaudeRows(path string) ([]claudeRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, core.WrapSourceError(core.ErrPermissionDenied, core.SourceClaude, path, err)
	}
	defer file.Close()

	var rows []claudeRow
	scanner := bufio.NewScanner(file)
	// AI session transcripts can carry large tool outputs or pasted content on a
	// single JSONL line (observed up to ~1.7 MB), well above bufio.Scanner's 64
	// KiB default. Raise the max token size so normal long lines parse.
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row claudeRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceClaude, path, err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, core.WrapSourceError(core.ErrPermissionDenied, core.SourceClaude, path, err)
	}
	return rows, nil
}

func claudeTurnsFromRows(rows []claudeRow) []core.Turn {
	var turns []core.Turn
	for _, row := range rows {
		roleText := row.Message.Role
		if roleText == "" {
			roleText = row.Type
		}
		role := core.TurnRole(roleText)
		if role != core.RoleUser && role != core.RoleAssistant && role != core.RoleSystem {
			continue
		}
		text := contentText(row.Message.Content)
		if text == "" {
			continue
		}
		turns = append(turns, core.Turn{
			Role:      role,
			Text:      text,
			Timestamp: timeFromISO(row.Timestamp),
			Kind:      core.KindMessage,
		})
	}
	return turns
}

func claudeSummaryFromRows(path string, rows []claudeRow, turns []core.Turn) core.SessionSummary {
	sessionID := claudeSessionID(rows, path)
	cwd := firstClaudeValue(rows, func(row claudeRow) string { return row.CWD })
	textParts := make([]string, 0, len(turns))
	for _, turn := range turns {
		textParts = append(textParts, turn.Text)
	}
	return core.SessionSummary{
		ID:            core.MakeSessionID(core.SourceClaude, sessionID),
		Source:        core.SourceClaude,
		NativeID:      sessionID,
		Title:         titleFromTurns(turns, "Untitled Session"),
		Project:       projectFromCWD(cwd),
		CWD:           cwd,
		CreatedAt:     firstTurnTime(turns),
		UpdatedAt:     lastTurnTime(turns),
		Preview:       previewText(strings.Join(textParts, "\n"), 160),
		TurnCount:     len(turns),
		Available:     true,
		ReaderBackend: core.BackendStorage,
	}
}

func claudeSessionID(rows []claudeRow, path string) string {
	id := firstClaudeValue(rows, func(row claudeRow) string { return row.SessionID })
	if id != "" {
		return id
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func firstClaudeValue(rows []claudeRow, get func(claudeRow) string) string {
	for _, row := range rows {
		if value := get(row); value != "" {
			return value
		}
	}
	return ""
}

func firstTurnTime(turns []core.Turn) *time.Time {
	if len(turns) == 0 {
		return nil
	}
	return turns[0].Timestamp
}

func lastTurnTime(turns []core.Turn) *time.Time {
	if len(turns) == 0 {
		return nil
	}
	return turns[len(turns)-1].Timestamp
}

type claudeRow struct {
	Type      string        `json:"type"`
	SessionID string        `json:"sessionId"`
	Timestamp string        `json:"timestamp"`
	CWD       string        `json:"cwd"`
	Message   claudeMessage `json:"message"`
}

type claudeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}
