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

	"github.com/yangkushu/ai-session-history/internal/core"
)

const maxPiJSONLLineBytes = 16 << 20
const piSubagentArtifactsDirName = "subagent-artifacts"

type PiStorageReader struct {
	roots []string
}

type piHeader struct {
	Type          string `json:"type"`
	Version       *int   `json:"version"`
	ID            string `json:"id"`
	Timestamp     string `json:"timestamp"`
	CWD           string `json:"cwd"`
	ParentSession string `json:"parentSession"`
}

type piEntry struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	ParentID  *string         `json:"parentId"`
	Timestamp string          `json:"timestamp"`
	Message   json.RawMessage `json:"message"`
	Name      *string         `json:"name"`
	Summary   string          `json:"summary"`
	FromID    string          `json:"fromId"`
}

type piMessage struct {
	Role        string          `json:"role"`
	Content     json.RawMessage `json:"content"`
	Timestamp   float64         `json:"timestamp"`
	IsError     bool            `json:"isError"`
	ToolName    string          `json:"toolName"`
	Command     string          `json:"command"`
	Output      string          `json:"output"`
	ExcludeFrom bool            `json:"excludeFromContext"`
}

type piContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Name string `json:"name"`
}

func NewPiStorageReader(roots []string) *PiStorageReader {
	return &PiStorageReader{roots: append([]string(nil), roots...)}
}

func (r *PiStorageReader) Doctor() core.SourceDiagnostic {
	sessions, warnings, err := r.ListSessionsWithDiagnostics()
	if len(sessions) > 0 {
		if len(warnings) > 0 {
			return core.SourceDiagnostic{Source: core.SourcePi, Status: "partial", Warnings: warnings}
		}
		return core.SourceDiagnostic{Source: core.SourcePi, Status: "available"}
	}
	if err == nil {
		err = core.NewError(core.ErrSourceUnavailable, "no Pi JSONL sessions found")
	}
	diagnostic := diagnosticFromError(core.SourcePi, err)
	diagnostic.Warnings = warnings
	return diagnostic
}

func (r *PiStorageReader) ListSessions() ([]core.SessionSummary, error) {
	sessions, _, err := r.ListSessionsWithDiagnostics()
	return sessions, err
}

// ListSessionsWithDiagnostics returns readable sessions together with isolated
// per-file failures. A partial result is successful when at least one session
// can be read.
func (r *PiStorageReader) ListSessionsWithDiagnostics() ([]core.SessionSummary, []core.DiagnosticWarning, error) {
	paths, warnings, rootsFound := r.sessionFiles()
	sessions := make([]core.SessionSummary, 0, len(paths))
	seen := make(map[string]bool)
	for _, path := range paths {
		header, entries, err := readPiSession(path)
		if err != nil {
			warnings = append(warnings, piWarning(path, err))
			continue
		}
		if seen[header.ID] {
			continue
		}
		seen[header.ID] = true
		sessions = append(sessions, piSummary(path, header, entries))
	}
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessionTime(sessions[i]).After(sessionTime(sessions[j]))
	})
	if len(sessions) > 0 {
		return sessions, warnings, nil
	}
	if len(warnings) > 0 {
		return sessions, warnings, piWarningError(core.SourcePi, warnings[0])
	}
	if !rootsFound {
		return sessions, nil, core.NewError(core.ErrSourceUnavailable, "no Pi session storage directory found")
	}
	return sessions, nil, core.NewError(core.ErrSourceUnavailable, "no Pi JSONL sessions found")
}

func (r *PiStorageReader) GetSession(nativeID string) (core.SessionDetail, error) {
	paths, warnings, _ := r.sessionFiles()
	var matchedErr error
	var matchedPath string
	for _, path := range paths {
		header, entries, err := readPiSession(path)
		if header.ID != nativeID {
			continue
		}
		if err != nil {
			if matchedErr == nil {
				matchedErr, matchedPath = err, path
			}
			continue
		}
		return core.SessionDetail{Summary: piSummary(path, header, entries), Turns: piTurns(header, entries)}, nil
	}
	if matchedErr != nil {
		return core.SessionDetail{}, piWarningError(core.SourcePi, piWarning(matchedPath, matchedErr))
	}
	if len(warnings) > 0 {
		for _, warning := range warnings {
			if warning.Code == core.ErrPermissionDenied {
				return core.SessionDetail{}, piWarningError(core.SourcePi, warning)
			}
		}
	}
	return core.SessionDetail{}, core.NewError(core.ErrSessionNotFound, "Pi session not found: "+nativeID)
}

func (r *PiStorageReader) sessionFiles() ([]string, []core.DiagnosticWarning, bool) {
	var paths []string
	var warnings []core.DiagnosticWarning
	rootsFound := false
	for _, root := range r.roots {
		info, err := os.Lstat(root)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				warnings = append(warnings, piWarning(root, pathInspectionError(core.SourcePi, root, err)))
			}
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		rootsFound = true
		if !info.IsDir() {
			warnings = append(warnings, core.DiagnosticWarning{
				Code: core.ErrUnsupportedFormat, Path: root, Message: "Pi session root is not a directory",
			})
			continue
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if !errors.Is(walkErr, fs.ErrNotExist) {
					warnings = append(warnings, piWarning(path, pathInspectionError(core.SourcePi, path, walkErr)))
				}
				return nil
			}
			if path == root || entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if entry.IsDir() {
				// pi-subagents stores child transcripts as JSONL beside Pi sessions;
				// these are event artifacts, not native Pi session files.
				if entry.Name() == piSubagentArtifactsDirName {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.EqualFold(filepath.Ext(entry.Name()), ".jsonl") {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			warnings = append(warnings, piWarning(root, pathInspectionError(core.SourcePi, root, err)))
		}
	}
	// Preserve configured root priority, then sort paths deterministically inside
	// each root. Duplicates across roots are removed without changing precedence.
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

func readPiSession(path string) (piHeader, []piEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return piHeader{}, nil, pathInspectionError(core.SourcePi, path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxPiJSONLLineBytes)
	var header piHeader
	var entries []piEntry
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if header.Type == "" {
			if err := json.Unmarshal([]byte(line), &header); err != nil {
				return header, nil, piFormatError(path, lineNumber, err)
			}
			if header.Type != "session" || header.ID == "" {
				return header, nil, piFormatError(path, lineNumber, errors.New("missing Pi session header or session id"))
			}
			version := 1
			if header.Version != nil {
				version = *header.Version
			}
			if version < 1 || version > 3 {
				return header, nil, piFormatError(path, lineNumber, fmt.Errorf("unsupported Pi session version %d (supported: 1-3)", version))
			}
			continue
		}
		var entry piEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return header, nil, piFormatError(path, lineNumber, err)
		}
		if entry.Type == "" {
			return header, nil, piFormatError(path, lineNumber, errors.New("session entry has no type"))
		}
		if header.Version != nil && *header.Version >= 2 {
			if entry.ID == "" {
				return header, nil, piFormatError(path, lineNumber, errors.New("tree entry has no id"))
			}
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return header, nil, piFormatError(path, lineNumber+1, fmt.Errorf("JSONL line exceeds %d bytes", maxPiJSONLLineBytes))
		}
		return header, nil, pathInspectionError(core.SourcePi, path, err)
	}
	if header.Type == "" {
		return header, nil, piFormatError(path, 0, errors.New("empty Pi session file"))
	}
	if header.Version != nil && *header.Version >= 2 {
		if err := validatePiEntryTree(entries); err != nil {
			return header, nil, piFormatError(path, 0, err)
		}
	}
	return header, entries, nil
}

func validatePiEntryTree(entries []piEntry) error {
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if seen[entry.ID] {
			return fmt.Errorf("duplicate Pi entry id %q", entry.ID)
		}
		seen[entry.ID] = true
	}
	return nil
}

func piActiveEntries(header piHeader, entries []piEntry) []piEntry {
	if header.Version == nil || *header.Version < 2 || len(entries) == 0 {
		return append([]piEntry(nil), entries...)
	}
	byID := make(map[string]piEntry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	current := entries[len(entries)-1]
	path := make([]piEntry, 0, len(entries))
	seen := make(map[string]bool)
	for current.ID != "" && !seen[current.ID] {
		seen[current.ID] = true
		path = append(path, current)
		if current.ParentID == nil || *current.ParentID == "" {
			break
		}
		parent, ok := byID[*current.ParentID]
		if !ok {
			break
		}
		current = parent
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

func piTurns(header piHeader, entries []piEntry) []core.Turn {
	active := piActiveEntries(header, entries)
	var turns []core.Turn
	for _, entry := range active {
		switch entry.Type {
		case "message":
			turns = append(turns, piMessageTurns(entry)...)
		case "compaction":
			if strings.TrimSpace(entry.Summary) != "" {
				turns = append(turns, core.Turn{
					Role: core.RoleAssistant, Text: entry.Summary, Kind: core.KindPersistedSummary,
					SummaryKind: core.SummaryCompaction, Timestamp: timeFromISO(entry.Timestamp),
				})
			}
		case "branch_summary":
			if strings.TrimSpace(entry.Summary) != "" {
				turns = append(turns, core.Turn{
					Role: core.RoleAssistant, Text: entry.Summary, Kind: core.KindPersistedSummary,
					SummaryKind: core.SummaryBranchSummary, Timestamp: timeFromISO(entry.Timestamp),
				})
			}
		}
	}
	return turns
}

func piMessageTurns(entry piEntry) []core.Turn {
	var message piMessage
	if len(entry.Message) == 0 || json.Unmarshal(entry.Message, &message) != nil {
		return nil
	}
	timestamp := timeFromMillisFloat(message.Timestamp)
	if timestamp == nil {
		timestamp = timeFromISO(entry.Timestamp)
	}
	switch message.Role {
	case "user":
		text := piTextContent(message.Content)
		if text == "" {
			return nil
		}
		return []core.Turn{{Role: core.RoleUser, Text: text, Timestamp: timestamp, Kind: core.KindMessage}}
	case "assistant":
		var blocks []piContentBlock
		if json.Unmarshal(message.Content, &blocks) != nil {
			var text string
			if json.Unmarshal(message.Content, &text) == nil && text != "" {
				return []core.Turn{{Role: core.RoleAssistant, Text: text, Timestamp: timestamp, Kind: core.KindMessage}}
			}
			return nil
		}
		var turns []core.Turn
		var textParts []string
		for _, block := range blocks {
			switch block.Type {
			case "text":
				if block.Text != "" {
					textParts = append(textParts, block.Text)
				}
			case "toolCall":
				if block.Name != "" {
					turns = append(turns, core.Turn{
						Role: core.RoleTool, Text: "Tool call: " + block.Name,
						Timestamp: timestamp, Kind: core.KindToolCall,
					})
				}
			}
		}
		if text := strings.Join(textParts, "\n"); text != "" {
			turns = append([]core.Turn{{Role: core.RoleAssistant, Text: text, Timestamp: timestamp, Kind: core.KindMessage}}, turns...)
		}
		return turns
	case "toolResult":
		text := piTextContent(message.Content)
		if text == "" {
			return nil
		}
		kind := core.KindToolResult
		if message.IsError {
			kind = core.KindError
		}
		turn := core.Turn{Role: core.RoleTool, Text: text, Timestamp: timestamp, Kind: kind}
		if kind == core.KindToolResult && len(text) > 500 {
			turn.OmittedReason = "tool_output"
		}
		return []core.Turn{turn}
	case "bashExecution":
		if message.ExcludeFrom {
			return nil
		}
		// Pi stores the shell command alongside its output; the command is a
		// tool argument, so only retain the user-visible execution output.
		text := strings.TrimSpace(message.Output)
		if text == "" {
			return nil
		}
		return []core.Turn{{
			Role: core.RoleTool, Text: text, Timestamp: timestamp,
			Kind: core.KindToolResult, OmittedReason: "tool_output",
		}}
	default:
		return nil
	}
}

func piTextContent(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var blocks []piContentBlock
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func piSummary(path string, header piHeader, entries []piEntry) core.SessionSummary {
	active := piActiveEntries(header, entries)
	turns := piTurns(header, entries)
	name := ""
	for _, entry := range entries {
		if entry.Type == "session_info" && entry.Name != nil {
			name = strings.TrimSpace(*entry.Name)
		}
	}
	if name == "" {
		name = titleFromTurns(turns, "Untitled Session")
	}
	cwd := header.CWD
	created := timeFromISO(header.Timestamp)
	updated := piUpdatedAt(entries, active, created)
	textParts := make([]string, 0, len(turns))
	for _, turn := range turns {
		if turn.Role == core.RoleUser || turn.Role == core.RoleAssistant {
			textParts = append(textParts, turn.Text)
		}
	}
	return core.SessionSummary{
		ID: core.MakeSessionID(core.SourcePi, header.ID), Source: core.SourcePi, NativeID: header.ID,
		Title: name, Project: projectFromCWD(cwd), CWD: cwd, CreatedAt: created, UpdatedAt: updated,
		Preview: previewText(strings.Join(textParts, "\n"), 160), TurnCount: len(turns),
		Available: true, ReaderBackend: core.BackendStorage,
	}
}

func piUpdatedAt(entries, active []piEntry, fallback *time.Time) *time.Time {
	var latest *time.Time
	for _, entry := range entries {
		if entry.Type != "message" {
			continue
		}
		var message piMessage
		if json.Unmarshal(entry.Message, &message) != nil || (message.Role != "user" && message.Role != "assistant") {
			continue
		}
		timestamp := timeFromMillisFloat(message.Timestamp)
		if timestamp == nil {
			timestamp = timeFromISO(entry.Timestamp)
		}
		if timestamp != nil && (latest == nil || timestamp.After(*latest)) {
			latest = timestamp
		}
	}
	if latest != nil {
		return latest
	}
	for _, entry := range active {
		if timestamp := timeFromISO(entry.Timestamp); timestamp != nil && (latest == nil || timestamp.After(*latest)) {
			latest = timestamp
		}
	}
	if latest != nil {
		return latest
	}
	return fallback
}

func timeFromMillisFloat(value float64) *time.Time {
	if value <= 0 {
		return nil
	}
	t := time.UnixMilli(int64(value)).UTC()
	return &t
}

func piWarning(path string, err error) core.DiagnosticWarning {
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

func piWarningError(source core.Source, warning core.DiagnosticWarning) *core.AppError {
	return &core.AppError{
		Code: warning.Code, Source: source, Path: warning.Path,
		Message: warning.Message,
	}
}

func piFormatError(path string, line int, err error) error {
	location := path
	if line > 0 {
		location = fmt.Sprintf("%s:%d", path, line)
	}
	return core.WrapSourceError(core.ErrUnsupportedFormat, core.SourcePi, path, fmt.Errorf("%s: %w", location, err))
}
