package core

import (
	"errors"
	"fmt"
	"time"
)

type Source string

const (
	SourceCodex  Source = "codex"
	SourceClaude Source = "claude"
	SourceCursor Source = "cursor"
)

type ReaderBackend string

const (
	BackendStorage ReaderBackend = "storage"
)

type ContentMode string

const (
	ModeClean   ContentMode = "clean"
	ModeSummary ContentMode = "summary"
	ModeRaw     ContentMode = "raw"
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

type Turn struct {
	Role          TurnRole   `json:"role"`
	Text          string     `json:"text"`
	Timestamp     *time.Time `json:"timestamp,omitempty"`
	Kind          TurnKind   `json:"kind"`
	Omitted       bool       `json:"omitted"`
	OmittedReason string     `json:"omitted_reason,omitempty"`
}

type SessionSummary struct {
	ID            string        `json:"id"`
	Source        Source        `json:"source"`
	NativeID      string        `json:"native_id"`
	Title         string        `json:"title"`
	Project       string        `json:"project,omitempty"`
	CWD           string        `json:"cwd,omitempty"`
	CreatedAt     *time.Time    `json:"created_at,omitempty"`
	UpdatedAt     *time.Time    `json:"updated_at,omitempty"`
	Preview       string        `json:"preview"`
	TurnCount     int           `json:"turn_count"`
	Available     bool          `json:"available"`
	ReaderBackend ReaderBackend `json:"reader_backend"`
}

type SessionDetail struct {
	Summary   SessionSummary `json:"summary"`
	Turns     []Turn         `json:"turns"`
	Truncated bool           `json:"truncated"`
}

type ErrorCode string

const (
	ErrPermissionDenied  ErrorCode = "permission_denied"
	ErrSourceUnavailable ErrorCode = "source_unavailable"
	ErrUnsupportedFormat ErrorCode = "unsupported_format"
	ErrSessionNotFound   ErrorCode = "session_not_found"
	ErrInvalidSessionID  ErrorCode = "invalid_session_id"
	ErrInvalidConfig     ErrorCode = "invalid_config"
	ErrReaderUnavailable ErrorCode = "reader_unavailable"
)

type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Source  Source    `json:"source,omitempty"`
	Path    string    `json:"path,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

func NewError(code ErrorCode, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func WrapSourceError(code ErrorCode, source Source, path string, err error) *AppError {
	return &AppError{
		Code:    code,
		Source:  source,
		Path:    path,
		Message: fmt.Sprintf("%s: %v", source, err),
	}
}

func IsCode(err error, code ErrorCode) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Code == code
}
