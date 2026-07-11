package render

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yangkushu/ai-session-history/internal/core"
)

type ContextOptions struct {
	TargetCWD string
	MaxChars  int
}

const HandoffSchemaVersion = "context-handoff.v1"

type HandoffContext struct {
	SchemaVersion             string             `json:"schema_version"`
	Session                   HandoffSession     `json:"session"`
	InitialGoal               HandoffInitialGoal `json:"initial_goal"`
	RecentConversation        []core.Turn        `json:"recent_conversation"`
	ToolOutcomes              []core.Turn        `json:"tool_outcomes"`
	HandoffNotes              []HandoffNote      `json:"handoff_notes"`
	HandoffInstruction        string             `json:"handoff_instruction"`
	Truncated                 bool               `json:"truncated"`
	RecentConversationOmitted bool               `json:"recent_conversation_omitted,omitempty"`
	ToolOutcomesOmitted       bool               `json:"tool_outcomes_omitted,omitempty"`
}

type HandoffSession struct {
	ID          string      `json:"id"`
	Source      core.Source `json:"source"`
	NativeID    string      `json:"native_id"`
	Title       string      `json:"title,omitempty"`
	OriginalCWD string      `json:"original_cwd,omitempty"`
	TargetCWD   string      `json:"target_cwd,omitempty"`
	CreatedAt   *time.Time  `json:"created_at,omitempty"`
	UpdatedAt   *time.Time  `json:"updated_at,omitempty"`
}

type HandoffInitialGoal struct {
	Available bool   `json:"available"`
	Text      string `json:"text,omitempty"`
}

type HandoffNote struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Detail(detail core.SessionDetail, mode core.ContentMode, maxChars int) core.SessionDetail {
	if maxChars <= 0 {
		maxChars = 50000
	}
	out := detail
	out.Turns = nil
	remaining := maxChars
	for _, turn := range detail.Turns {
		if remaining <= 0 {
			out.Truncated = true
			break
		}
		turn.Text = turnText(turn, mode)
		if len(turn.Text) > remaining {
			turn.Text = turn.Text[:remaining]
			out.Truncated = true
		}
		if strings.HasPrefix(turn.Text, "[omitted") || strings.Contains(turn.Text, " omitted: ") {
			turn.Omitted = true
		}
		remaining -= len(turn.Text)
		out.Turns = append(out.Turns, turn)
	}
	return out
}

func BuildHandoff(detail core.SessionDetail, opts ContextOptions) HandoffContext {
	maxChars := opts.MaxChars
	if maxChars <= 0 {
		maxChars = 20000
	}
	clean := Detail(detail, core.ModeClean, maxChars*10)
	filteredTurns, skippedSetup := filterContextTurns(clean.Turns)
	outcomes := toolOutcomes(clean.Turns)
	omittedToolOutput := countOmittedToolOutput(clean.Turns)
	notes := contextNotes{
		skippedSetup:      skippedSetup,
		omittedToolOutput: omittedToolOutput,
		truncated:         clean.Truncated,
	}
	handoff := buildHandoff(clean.Summary, filteredTurns, outcomes, notes, opts, 1200, 1200, false, true)
	if handoffContentBytes(handoff) > maxChars {
		handoff = buildHandoff(clean.Summary, filteredTurns, outcomes, contextNotes{
			skippedSetup:      skippedSetup,
			omittedToolOutput: omittedToolOutput,
			truncated:         true,
		}, opts, 360, 180, true, true)
	}
	if handoffContentBytes(handoff) > maxChars {
		handoff = buildHandoff(clean.Summary, filteredTurns, outcomes, contextNotes{
			skippedSetup:      skippedSetup,
			omittedToolOutput: omittedToolOutput,
			truncated:         true,
		}, opts, 90, 0, true, false)
	}
	if handoffContentBytes(handoff) > maxChars {
		handoff = fitHandoffContentBudget(handoff, maxChars)
	}
	return handoff
}

func Context(detail core.SessionDetail, opts ContextOptions) string {
	maxChars := opts.MaxChars
	if maxChars <= 0 {
		maxChars = 20000
	}
	handoff := BuildHandoff(detail, opts)
	text := ContextFromHandoff(handoff)
	if len(text) > maxChars {
		text = ContextFromHandoff(fitHandoffContentBudget(handoff, maxChars))
	}
	return bound(text, maxChars)
}

type contextNotes struct {
	skippedSetup      int
	omittedToolOutput int
	truncated         bool
}

func buildHandoff(s core.SessionSummary, turns []core.Turn, outcomes []core.Turn, notes contextNotes, opts ContextOptions, goalLimit int, turnLimit int, compactRecent bool, includeDetails bool) HandoffContext {
	recent := []core.Turn{}
	toolResults := []core.Turn{}
	recentSource := recentConversation(turns, compactRecent)
	if includeDetails {
		recent = limitedTurns(recentSource, turnLimit)
		toolResults = limitedTurns(outcomes, turnLimit)
	}
	return HandoffContext{
		SchemaVersion: HandoffSchemaVersion,
		Session: HandoffSession{
			ID:          s.ID,
			Source:      s.Source,
			NativeID:    s.NativeID,
			Title:       s.Title,
			OriginalCWD: s.CWD,
			TargetCWD:   opts.TargetCWD,
			CreatedAt:   s.CreatedAt,
			UpdatedAt:   s.UpdatedAt,
		},
		InitialGoal:               handoffInitialGoal(turns, goalLimit),
		RecentConversation:        recent,
		ToolOutcomes:              toolResults,
		HandoffNotes:              handoffNotes(notes),
		HandoffInstruction:        handoffInstruction(includeDetails),
		Truncated:                 notes.truncated,
		RecentConversationOmitted: !includeDetails && len(recentSource) > 0,
		ToolOutcomesOmitted:       !includeDetails && len(outcomes) > 0,
	}
}

func ContextFromHandoff(handoff HandoffContext) string {
	var b strings.Builder
	b.WriteString("# AI Session Context\n\n")
	b.WriteString("## Session\n\n")
	writeLine(&b, "- ID: %s", handoff.Session.ID)
	writeLine(&b, "- Source: %s", handoff.Session.Source)
	writeLine(&b, "- Original CWD: %s", valueOrUnknown(handoff.Session.OriginalCWD))
	if handoff.Session.TargetCWD != "" {
		writeLine(&b, "- Target CWD: %s", handoff.Session.TargetCWD)
	}
	writeLine(&b, "- Created: %s", timeOrUnknown(handoff.Session.CreatedAt))
	writeLine(&b, "- Updated: %s", timeOrUnknown(handoff.Session.UpdatedAt))
	b.WriteString("\n## Initial Goal\n\n")
	if handoff.InitialGoal.Available {
		b.WriteString(handoff.InitialGoal.Text)
	} else {
		b.WriteString("Unavailable")
	}
	b.WriteString("\n\n## Recent Conversation\n\n")
	if handoff.RecentConversationOmitted {
		b.WriteString("Omitted for size.\n\n")
	} else {
		for _, turn := range handoff.RecentConversation {
			writeLine(&b, "### %s", titleRole(turn.Role))
			b.WriteString("\n")
			b.WriteString(turn.Text)
			b.WriteString("\n\n")
		}
	}
	b.WriteString("## Tool Outcomes\n\n")
	if handoff.ToolOutcomesOmitted {
		b.WriteString("Omitted for size.\n\n")
	} else {
		for _, turn := range handoff.ToolOutcomes {
			writeLine(&b, "### %s", titleKind(turn.Kind))
			b.WriteString("\n")
			b.WriteString(turn.Text)
			b.WriteString("\n\n")
		}
	}
	b.WriteString("## Handoff Notes\n\n")
	for _, note := range handoff.HandoffNotes {
		writeLine(&b, "- %s", note.Message)
		if note.Code == "context_truncated" {
			b.WriteString("- [truncated]\n")
		}
	}
	b.WriteString("\n## Handoff Instruction\n\n")
	b.WriteString(handoff.HandoffInstruction)
	return b.String()
}

func turnText(turn core.Turn, mode core.ContentMode) string {
	if mode == core.ModeRaw {
		return turn.Text
	}
	if turn.Role == core.RoleTool && turn.Kind == core.KindToolResult {
		if preserveToolResult(turn) {
			return turn.Text
		}
		reason := turn.OmittedReason
		if reason == "" {
			reason = "tool_output"
		}
		if mode == core.ModeSummary {
			return fmt.Sprintf("[%s omitted: %s]", turn.Kind, reason)
		}
		return fmt.Sprintf("[omitted: %s]", reason)
	}
	if turn.Role == core.RoleTool && mode == core.ModeClean && turn.Kind != core.KindError {
		reason := turn.OmittedReason
		if reason == "" {
			reason = string(turn.Kind)
		}
		return fmt.Sprintf("[omitted: %s]", reason)
	}
	return turn.Text
}

func preserveToolResult(turn core.Turn) bool {
	if turn.OmittedReason != "" {
		return false
	}
	text := strings.TrimSpace(turn.Text)
	return text != "" && len(text) <= 500
}

func handoffInitialGoal(turns []core.Turn, limit int) HandoffInitialGoal {
	for _, turn := range turns {
		if turn.Role == core.RoleUser && strings.TrimSpace(turn.Text) != "" {
			return HandoffInitialGoal{Available: true, Text: limitText(turn.Text, limit)}
		}
	}
	return HandoffInitialGoal{Available: false}
}

func recentConversation(turns []core.Turn, compact bool) []core.Turn {
	var messages []core.Turn
	for _, turn := range turns {
		if (turn.Role == core.RoleUser || turn.Role == core.RoleAssistant) && strings.TrimSpace(turn.Text) != "" {
			messages = append(messages, turn)
		}
	}
	if compact && len(messages) > 1 {
		return messages[len(messages)-1:]
	}
	if len(messages) <= 3 {
		return messages
	}
	return messages[len(messages)-2:]
}

func toolOutcomes(turns []core.Turn) []core.Turn {
	var outcomes []core.Turn
	for _, turn := range turns {
		if turn.Role != core.RoleTool || strings.TrimSpace(turn.Text) == "" || turn.Omitted {
			continue
		}
		if turn.Kind == core.KindToolResult || turn.Kind == core.KindError {
			outcomes = append(outcomes, turn)
		}
	}
	if len(outcomes) <= 5 {
		return outcomes
	}
	return outcomes[len(outcomes)-5:]
}

func limitedTurns(turns []core.Turn, limit int) []core.Turn {
	out := make([]core.Turn, 0, len(turns))
	for _, turn := range turns {
		turn.Text = limitText(turn.Text, limit)
		out = append(out, turn)
	}
	return out
}

func filterContextTurns(turns []core.Turn) ([]core.Turn, int) {
	filtered := make([]core.Turn, 0, len(turns))
	skippedSetup := 0
	for _, turn := range turns {
		if turn.Role == core.RoleUser && isContextSetupBoilerplate(turn.Text) {
			skippedSetup++
			continue
		}
		filtered = append(filtered, turn)
	}
	return filtered, skippedSetup
}

func isContextSetupBoilerplate(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(trimmed, "# AGENTS.md instructions") ||
		strings.HasPrefix(trimmed, "# CLAUDE.md instructions") ||
		strings.HasPrefix(trimmed, "# Global Claude Code Instructions") {
		return true
	}
	for _, marker := range []string{
		"<environment_context>",
		"<app-context>",
		"<permissions instructions>",
		"<collaboration_mode>",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func countOmittedToolOutput(turns []core.Turn) int {
	count := 0
	for _, turn := range turns {
		if turn.Role == core.RoleTool && turn.Omitted {
			count++
		}
	}
	return count
}

func handoffNotes(notes contextNotes) []HandoffNote {
	out := []HandoffNote{}
	if notes.skippedSetup == 0 && notes.omittedToolOutput == 0 && !notes.truncated {
		return append(out, HandoffNote{Code: "no_omissions", Message: "No skipped, omitted, or truncated content."})
	}
	if notes.skippedSetup > 0 {
		out = append(out, HandoffNote{Code: "setup_boilerplate_skipped", Message: fmt.Sprintf("Skipped setup boilerplate turns: %d", notes.skippedSetup)})
	}
	if notes.omittedToolOutput > 0 {
		out = append(out, HandoffNote{Code: "tool_output_omitted", Message: fmt.Sprintf("Omitted noisy tool output turns: %d", notes.omittedToolOutput)})
	}
	if notes.truncated {
		out = append(out, HandoffNote{Code: "context_truncated", Message: "Context truncated to --max-chars."})
	}
	return out
}

func handoffInstruction(includeDetails bool) string {
	if includeDetails {
		return "Continue from this prior AI coding session. Treat the original CWD as historical context and the target CWD, when present, as the active working directory."
	}
	return "Continue from this prior AI coding session."
}

func fitHandoffContentBudget(handoff HandoffContext, maxChars int) HandoffContext {
	handoff.Truncated = true
	if len(handoff.RecentConversation) > 0 {
		handoff.RecentConversationOmitted = true
	}
	if len(handoff.ToolOutcomes) > 0 {
		handoff.ToolOutcomesOmitted = true
	}
	handoff.RecentConversation = []core.Turn{}
	handoff.ToolOutcomes = []core.Turn{}
	handoff.HandoffNotes = ensureTruncationNote(handoff.HandoffNotes)

	available := maxChars - len(handoff.HandoffInstruction)
	for _, note := range handoff.HandoffNotes {
		available -= len(note.Message)
	}
	if !handoff.InitialGoal.Available || strings.TrimSpace(handoff.InitialGoal.Text) == "" {
		return handoff
	}
	goalLimit := available
	if goalLimit <= 0 || goalLimit > 90 {
		goalLimit = 90
	}
	handoff.InitialGoal.Text = strings.TrimSpace(truncateAtRuneBoundary(handoff.InitialGoal.Text, goalLimit))
	return handoff
}

func ensureTruncationNote(notes []HandoffNote) []HandoffNote {
	out := append([]HandoffNote{}, notes...)
	for _, note := range out {
		if note.Code == "context_truncated" {
			return out
		}
	}
	return append(out, HandoffNote{Code: "context_truncated", Message: "Context truncated to --max-chars."})
}

// handoffContentBytes counts only trim-prone handoff text content, not serialized JSON size:
// initial goal, instruction, recent/tool text, and note messages.
func handoffContentBytes(handoff HandoffContext) int {
	size := len(handoff.InitialGoal.Text) + len(handoff.HandoffInstruction)
	for _, turn := range handoff.RecentConversation {
		size += len(turn.Text)
	}
	for _, turn := range handoff.ToolOutcomes {
		size += len(turn.Text)
	}
	for _, note := range handoff.HandoffNotes {
		size += len(note.Message)
	}
	return size
}

func writeLine(b *strings.Builder, format string, args ...any) {
	b.WriteString(fmt.Sprintf(format, args...))
	b.WriteString("\n")
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func timeOrUnknown(value *time.Time) string {
	if value == nil {
		return "unknown"
	}
	return value.String()
}

func titleRole(role core.TurnRole) string {
	text := string(role)
	if text == "" {
		return ""
	}
	return strings.ToUpper(text[:1]) + text[1:]
}

func titleKind(kind core.TurnKind) string {
	text := strings.ReplaceAll(string(kind), "_", " ")
	if text == "" {
		return "Tool"
	}
	return strings.ToUpper(text[:1]) + text[1:]
}

func limitText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return strings.TrimSpace(truncateAtRuneBoundary(text, limit)) + "..."
}

func bound(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	marker := "\n\n[truncated]"
	if maxChars <= len(marker) {
		return truncateAtRuneBoundary(text, maxChars)
	}
	return strings.TrimSpace(truncateAtRuneBoundary(text, maxChars-len(marker))) + marker
}

func truncateAtRuneBoundary(text string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(text) <= maxBytes {
		return text
	}
	for maxBytes > 0 && !utf8.ValidString(text[:maxBytes]) {
		maxBytes--
	}
	return text[:maxBytes]
}
