package readers

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/yangkushu/ai-session-history/internal/core"
)

const maxDshJSONLLineBytes = 16 << 20

// DshStorageReader reads DeepSeek Harness (dsh) session transcripts stored as
// zstd-compressed JSONL event logs under DSH_HOME/sessions
// (<munged-cwd>--/session-<uuid>/session.v*.jsonl.zstd).
type DshStorageReader struct {
	roots []string
}

type dshEvent struct {
	Type string          `json:"type"`
	Seq  *int64          `json:"seq"`
	Time *int64          `json:"time"`
	Data json.RawMessage `json:"data"`
}

type dshSessionHeader struct {
	Type      string `json:"type"`
	Version   *int   `json:"version"`
	ID        string `json:"id"`
	CreatedAt int64  `json:"createdAt"`
	CWD       string `json:"cwd"`
}

type dshTitleData struct {
	Title string `json:"title"`
}

type dshContentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	Name      string `json:"name"`
	ID        string `json:"id"`
	Arguments string `json:"arguments"`
}

type dshMessageData struct {
	Role    string            `json:"role"`
	Content []dshContentBlock `json:"content"`
	IsError bool              `json:"isError"`
}

type dshParsedSession struct {
	header  dshSessionHeader
	events  []dshEvent
	turns   []core.Turn
	title   string
	updated *time.Time
}

func NewDshStorageReader(roots []string) *DshStorageReader {
	return &DshStorageReader{roots: append([]string(nil), roots...)}
}

func (r *DshStorageReader) Doctor() core.SourceDiagnostic {
	sessions, warnings, err := r.ListSessionsWithDiagnostics()
	if len(sessions) > 0 {
		if len(warnings) > 0 {
			return core.SourceDiagnostic{Source: core.SourceDsh, Status: "partial", Warnings: warnings}
		}
		return core.SourceDiagnostic{Source: core.SourceDsh, Status: "available"}
	}
	if err == nil {
		err = core.NewError(core.ErrSourceUnavailable, "no dsh JSONL sessions found")
	}
	diagnostic := diagnosticFromError(core.SourceDsh, err)
	diagnostic.Warnings = warnings
	return diagnostic
}

func (r *DshStorageReader) ListSessions() ([]core.SessionSummary, error) {
	sessions, _, err := r.ListSessionsWithDiagnostics()
	return sessions, err
}

// ListSessionsWithDiagnostics returns readable sessions together with isolated
// per-file failures. A partial result is successful when at least one session
// can be read.
func (r *DshStorageReader) ListSessionsWithDiagnostics() ([]core.SessionSummary, []core.DiagnosticWarning, error) {
	paths, warnings, rootsFound := r.sessionFiles()
	sessions := make([]core.SessionSummary, 0, len(paths))
	seen := make(map[string]bool)
	for _, path := range paths {
		parsed, err := readDshSession(path)
		if err != nil {
			warnings = append(warnings, dshWarning(path, err))
			continue
		}
		if seen[parsed.header.ID] {
			continue
		}
		seen[parsed.header.ID] = true
		sessions = append(sessions, dshSummary(parsed))
	}
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessionTime(sessions[i]).After(sessionTime(sessions[j]))
	})
	if len(sessions) > 0 {
		return sessions, warnings, nil
	}
	if len(warnings) > 0 {
		return sessions, warnings, dshWarningError(core.SourceDsh, warnings[0])
	}
	if !rootsFound {
		return sessions, nil, core.NewError(core.ErrSourceUnavailable, "no dsh session storage directory found")
	}
	return sessions, nil, core.NewError(core.ErrSourceUnavailable, "no dsh JSONL sessions found")
}

func (r *DshStorageReader) GetSession(nativeID string) (core.SessionDetail, error) {
	paths, warnings, _ := r.sessionFiles()
	var matchedErr error
	var matchedPath string
	for _, path := range paths {
		parsed, err := readDshSession(path)
		if parsed.header.ID != nativeID {
			continue
		}
		if err != nil {
			if matchedErr == nil {
				matchedErr, matchedPath = err, path
			}
			continue
		}
		return core.SessionDetail{Summary: dshSummary(parsed), Turns: parsed.turns}, nil
	}
	if matchedErr != nil {
		return core.SessionDetail{}, dshWarningError(core.SourceDsh, dshWarning(matchedPath, matchedErr))
	}
	for _, warning := range warnings {
		if warning.Code == core.ErrPermissionDenied {
			return core.SessionDetail{}, dshWarningError(core.SourceDsh, warning)
		}
	}
	return core.SessionDetail{}, core.NewError(core.ErrSessionNotFound, "dsh session not found: "+nativeID)
}

// sessionFiles walks each root for compressed dsh transcripts
// (session-<uuid>/session.v*.jsonl.zstd), preserving root priority and
// deterministic ordering within each root.
func (r *DshStorageReader) sessionFiles() ([]string, []core.DiagnosticWarning, bool) {
	var paths []string
	var warnings []core.DiagnosticWarning
	rootsFound := false
	for _, root := range r.roots {
		info, err := os.Lstat(root)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				warnings = append(warnings, dshWarning(root, pathInspectionError(core.SourceDsh, root, err)))
			}
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		rootsFound = true
		if !info.IsDir() {
			warnings = append(warnings, core.DiagnosticWarning{
				Code: core.ErrUnsupportedFormat, Path: root, Message: "dsh session root is not a directory",
			})
			continue
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if !errors.Is(walkErr, fs.ErrNotExist) {
					warnings = append(warnings, dshWarning(path, pathInspectionError(core.SourceDsh, path, walkErr)))
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl.zstd") {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			warnings = append(warnings, dshWarning(root, pathInspectionError(core.SourceDsh, root, err)))
		}
	}
	ordered := make([]string, 0, len(paths))
	seen := make(map[string]bool)
	for _, root := range r.roots {
		var underRoot []string
		for _, path := range paths {
			rel, err := filepath.Rel(root, path)
			if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				underRoot = append(underRoot, path)
			}
		}
		sort.Strings(underRoot)
		for _, path := range underRoot {
			if !seen[path] {
				seen[path] = true
				ordered = append(ordered, path)
			}
		}
	}
	return ordered, warnings, rootsFound
}

// readDshSession decompresses and parses a dsh transcript. When parsing fails
// after the session header was read, the partially-filled header is still
// returned so callers can match native IDs.
func readDshSession(path string) (dshParsedSession, error) {
	file, err := os.Open(path)
	if err != nil {
		return dshParsedSession{}, pathInspectionError(core.SourceDsh, path, err)
	}
	defer file.Close()

	decoder, err := zstd.NewReader(file)
	if err != nil {
		return dshParsedSession{}, dshFormatError(path, 0, err)
	}
	defer decoder.Close()

	scanner := bufio.NewScanner(decoder)
	scanner.Buffer(make([]byte, 64*1024), maxDshJSONLLineBytes)

	parsed := dshParsedSession{}
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if parsed.header.Type == "" {
			if err := json.Unmarshal([]byte(line), &parsed.header); err != nil {
				return parsed, dshFormatError(path, lineNumber, err)
			}
			if parsed.header.Type != "session" || parsed.header.ID == "" {
				return parsed, dshFormatError(path, lineNumber, errors.New("missing dsh session header or session id"))
			}
			if parsed.header.Version != nil && (*parsed.header.Version < 1 || *parsed.header.Version > 4) {
				return parsed, dshFormatError(path, lineNumber, fmt.Errorf("unsupported dsh session version %d (supported: 1-4)", *parsed.header.Version))
			}
			continue
		}
		var event dshEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return parsed, dshFormatError(path, lineNumber, err)
		}
		if event.Type == "" {
			continue
		}
		parsed.events = append(parsed.events, event)
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return parsed, dshFormatError(path, lineNumber+1, fmt.Errorf("JSONL line exceeds %d bytes", maxDshJSONLLineBytes))
		}
		if os.IsPermission(err) {
			return parsed, pathInspectionError(core.SourceDsh, path, err)
		}
		// Remaining stream errors (e.g. zstd magic mismatch) mean the
		// transcript is not a valid compressed dsh session file.
		return parsed, dshFormatError(path, lineNumber+1, err)
	}
	if parsed.header.Type == "" {
		return parsed, dshFormatError(path, 0, errors.New("empty dsh session file"))
	}
	parsed.turns, parsed.title, parsed.updated = dshReduce(parsed.events)
	return parsed, nil
}

// dshReduce converts the event stream into normalized turns, a session title,
// and the latest observed event time.
func dshReduce(events []dshEvent) ([]core.Turn, string, *time.Time) {
	var turns []core.Turn
	title := ""
	var updated *time.Time
	for _, event := range events {
		if ts := timeFromMillisPtr(event.Time); ts != nil && (updated == nil || ts.After(*updated)) {
			updated = ts
		}
		switch event.Type {
		case "session/title":
			var data dshTitleData
			if json.Unmarshal(event.Data, &data) == nil && strings.TrimSpace(data.Title) != "" {
				title = strings.TrimSpace(data.Title)
			}
		case "user/message":
			var data dshMessageData
			if json.Unmarshal(event.Data, &data) != nil {
				continue
			}
			text := dshTextContent(data.Content)
			if text == "" {
				continue
			}
			turns = append(turns, core.Turn{
				Role: core.RoleUser, Text: text, Kind: core.KindMessage,
				Timestamp: timeFromMillisPtr(event.Time),
			})
		case "assistant/message":
			var data struct {
				Message dshMessageData `json:"message"`
			}
			if json.Unmarshal(event.Data, &data) != nil {
				continue
			}
			timestamp := timeFromMillisPtr(event.Time)
			var toolCalls []core.Turn
			var textParts []string
			for _, block := range data.Message.Content {
				switch block.Type {
				case "text":
					if block.Text != "" {
						textParts = append(textParts, block.Text)
					}
				case "tool-call":
					if block.Name != "" {
						toolCalls = append(toolCalls, core.Turn{
							Role: core.RoleTool, Text: "Tool call: " + block.Name,
							Timestamp: timestamp, Kind: core.KindToolCall,
						})
					}
				}
			}
			if text := strings.Join(textParts, "\n"); text != "" {
				turns = append(turns, core.Turn{
					Role: core.RoleAssistant, Text: text, Timestamp: timestamp, Kind: core.KindMessage,
				})
			}
			turns = append(turns, toolCalls...)
		case "tool/result":
			var data struct {
				Message dshMessageData `json:"message"`
			}
			if json.Unmarshal(event.Data, &data) != nil {
				continue
			}
			text := dshTextContent(data.Message.Content)
			if text == "" {
				continue
			}
			kind := core.KindToolResult
			if data.Message.IsError {
				kind = core.KindError
			}
			turn := core.Turn{
				Role: core.RoleTool, Text: text, Timestamp: timeFromMillisPtr(event.Time), Kind: kind,
			}
			if kind == core.KindToolResult && len(text) > 500 {
				turn.OmittedReason = "tool_output"
			}
			turns = append(turns, turn)
		}
	}
	return turns, title, updated
}

func dshTextContent(blocks []dshContentBlock) string {
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func dshSummary(parsed dshParsedSession) core.SessionSummary {
	title := parsed.title
	if title == "" {
		title = titleFromTurns(parsed.turns, "Untitled Session")
	}
	created := timeFromMillisValue(parsed.header.CreatedAt)
	cwd := parsed.header.CWD
	textParts := make([]string, 0, len(parsed.turns))
	for _, turn := range parsed.turns {
		if turn.Role == core.RoleUser || turn.Role == core.RoleAssistant {
			textParts = append(textParts, turn.Text)
		}
	}
	return core.SessionSummary{
		ID: core.MakeSessionID(core.SourceDsh, parsed.header.ID), Source: core.SourceDsh, NativeID: parsed.header.ID,
		Title: title, Project: projectFromCWD(cwd), CWD: cwd, CreatedAt: created, UpdatedAt: parsed.updated,
		Preview: previewText(strings.Join(textParts, "\n"), 160), TurnCount: len(parsed.turns),
		Available: true, ReaderBackend: core.BackendStorage,
	}
}

func timeFromMillisPtr(value *int64) *time.Time {
	if value == nil || *value <= 0 {
		return nil
	}
	t := time.UnixMilli(*value).UTC()
	return &t
}

func timeFromMillisValue(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	t := time.UnixMilli(value).UTC()
	return &t
}

func dshWarning(path string, err error) core.DiagnosticWarning {
	warning := core.DiagnosticWarning{Code: core.ErrUnsupportedFormat, Path: path, Message: err.Error()}
	var appErr *core.AppError
	if errors.As(err, &appErr) {
		warning.Code = appErr.Code
		if appErr.Path != "" {
			warning.Path = appErr.Path
		}
		warning.Message = appErr.Message
	}
	return warning
}

func dshWarningError(source core.Source, warning core.DiagnosticWarning) *core.AppError {
	return &core.AppError{
		Code: warning.Code, Source: source, Path: warning.Path,
		Message: warning.Message,
	}
}

func dshFormatError(path string, line int, err error) error {
	location := path
	if line > 0 {
		location = fmt.Sprintf("%s:%d", path, line)
	}
	return core.WrapSourceError(core.ErrUnsupportedFormat, core.SourceDsh, path, fmt.Errorf("%s: %w", location, err))
}
