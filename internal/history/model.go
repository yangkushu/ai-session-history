package history

import (
	"errors"
	"path/filepath"
	"strings"
	"time"
)

type Source string

const (
	SourceCodex  Source = "codex"
	SourceClaude Source = "claude"
	SourceCursor Source = "cursor"
)

type TurnRole string

const (
	RoleUser      TurnRole = "user"
	RoleAssistant TurnRole = "assistant"
	RoleSystem    TurnRole = "system"
	RoleTool      TurnRole = "tool"
)

type TurnKind string

const (
	KindMessage    TurnKind = "message"
	KindToolCall   TurnKind = "tool_call"
	KindToolResult TurnKind = "tool_result"
	KindError      TurnKind = "error"
)

type ContentMode string

const (
	ContentModeClean   ContentMode = "clean"
	ContentModeSummary ContentMode = "summary"
	ContentModeRaw     ContentMode = "raw"
)

type Turn struct {
	Role          TurnRole  `json:"role"`
	Text          string    `json:"text"`
	Timestamp     time.Time `json:"timestamp,omitempty"`
	Kind          TurnKind  `json:"kind"`
	Omitted       bool      `json:"omitted"`
	OmittedReason string    `json:"omitted_reason,omitempty"`
}

type SessionSummary struct {
	ID            string    `json:"id"`
	Source        Source    `json:"source"`
	NativeID      string    `json:"native_id"`
	Title         string    `json:"title"`
	Project       string    `json:"project,omitempty"`
	CWD           string    `json:"cwd,omitempty"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
	Preview       string    `json:"preview"`
	TurnCount     int       `json:"turn_count"`
	Available     bool      `json:"available"`
	ReaderBackend string    `json:"reader_backend"`
}

type SessionDetail struct {
	Summary   SessionSummary `json:"summary"`
	Turns     []Turn         `json:"turns"`
	Truncated bool           `json:"truncated"`
}

type SourceDiagnostic struct {
	Source  Source `json:"source"`
	Status  string `json:"status"`
	Path    string `json:"path,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`

	Available bool `json:"available"`
}

var ErrSessionNotFound = errors.New("session_not_found")

func MakeSessionID(source Source, nativeID string) string {
	return string(source) + ":" + nativeID
}

func ParseSessionID(sessionID string) (Source, string, error) {
	sourcePart, nativeID, ok := strings.Cut(sessionID, ":")
	source := Source(sourcePart)
	if !ok || nativeID == "" || !isKnownSource(source) {
		return "", "", errors.New("invalid_session_id")
	}
	return source, nativeID, nil
}

func previewText(text string, limit int) string {
	collapsed := strings.Join(strings.Fields(text), " ")
	if len(collapsed) <= limit {
		return collapsed
	}
	prefix := strings.TrimSpace(collapsed[:limit])
	if idx := strings.LastIndex(prefix, " "); idx > 0 {
		prefix = prefix[:idx]
	}
	return prefix + "..."
}

func titleFromTurns(turns []Turn) string {
	for _, turn := range turns {
		if turn.Role == RoleUser && strings.TrimSpace(turn.Text) != "" {
			return previewText(turn.Text, 80)
		}
	}
	return "Untitled Session"
}

func projectFromCWD(cwd string) string {
	if cwd == "" {
		return ""
	}
	clean := filepath.Clean(cwd)
	base := filepath.Base(clean)
	if base == "." || base == string(filepath.Separator) {
		return cwd
	}
	return base
}

func isKnownSource(source Source) bool {
	return source == SourceCodex || source == SourceClaude || source == SourceCursor
}
