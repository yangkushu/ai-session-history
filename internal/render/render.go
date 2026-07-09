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

func Context(detail core.SessionDetail, opts ContextOptions) string {
	maxChars := opts.MaxChars
	if maxChars <= 0 {
		maxChars = 20000
	}
	clean := Detail(detail, core.ModeClean, maxChars*10)
	s := clean.Summary
	filteredTurns, skippedSetup := filterContextTurns(clean.Turns)
	outcomes := toolOutcomes(clean.Turns)
	omittedToolOutput := countOmittedToolOutput(clean.Turns)
	text := buildContext(s, filteredTurns, outcomes, contextNotes{
		skippedSetup:      skippedSetup,
		omittedToolOutput: omittedToolOutput,
		truncated:         clean.Truncated,
	}, opts, 1200, 1200, false, true)
	if len(text) > maxChars {
		text = buildContext(s, filteredTurns, outcomes, contextNotes{
			skippedSetup:      skippedSetup,
			omittedToolOutput: omittedToolOutput,
			truncated:         true,
		}, opts, 360, 180, true, true)
	}
	if len(text) > maxChars {
		text = buildContext(s, filteredTurns, outcomes, contextNotes{
			skippedSetup:      skippedSetup,
			omittedToolOutput: omittedToolOutput,
			truncated:         true,
		}, opts, 90, 0, true, false)
	}
	return bound(text, maxChars)
}

type contextNotes struct {
	skippedSetup      int
	omittedToolOutput int
	truncated         bool
}

func buildContext(s core.SessionSummary, turns []core.Turn, outcomes []core.Turn, notes contextNotes, opts ContextOptions, goalLimit int, turnLimit int, compactRecent bool, includeDetails bool) string {
	var b strings.Builder
	b.WriteString("# AI Session Context\n\n")
	b.WriteString("## Session\n\n")
	writeLine(&b, "- ID: %s", s.ID)
	writeLine(&b, "- Source: %s", s.Source)
	writeLine(&b, "- Original CWD: %s", valueOrUnknown(s.CWD))
	if opts.TargetCWD != "" {
		writeLine(&b, "- Target CWD: %s", opts.TargetCWD)
	}
	writeLine(&b, "- Created: %s", timeOrUnknown(s.CreatedAt))
	writeLine(&b, "- Updated: %s", timeOrUnknown(s.UpdatedAt))
	b.WriteString("\n## Initial Goal\n\n")
	b.WriteString(initialGoal(turns, goalLimit))
	b.WriteString("\n\n## Recent Conversation\n\n")
	if includeDetails {
		for _, turn := range recentConversation(turns, compactRecent) {
			writeLine(&b, "### %s", titleRole(turn.Role))
			b.WriteString("\n")
			b.WriteString(limitText(turn.Text, turnLimit))
			b.WriteString("\n\n")
		}
	} else {
		b.WriteString("Omitted for size.\n\n")
	}
	b.WriteString("## Tool Outcomes\n\n")
	if includeDetails && len(outcomes) > 0 {
		for _, turn := range outcomes {
			writeLine(&b, "### %s", titleKind(turn.Kind))
			b.WriteString("\n")
			b.WriteString(limitText(turn.Text, turnLimit))
			b.WriteString("\n\n")
		}
	} else if !includeDetails {
		b.WriteString("Omitted for size.\n\n")
	}
	b.WriteString("## Handoff Notes\n\n")
	if notes.skippedSetup == 0 && notes.omittedToolOutput == 0 && !notes.truncated {
		b.WriteString("- No skipped, omitted, or truncated content.\n")
	}
	if notes.skippedSetup > 0 {
		writeLine(&b, "- Skipped setup boilerplate turns: %d", notes.skippedSetup)
	}
	if notes.omittedToolOutput > 0 {
		writeLine(&b, "- Omitted noisy tool output turns: %d", notes.omittedToolOutput)
	}
	if notes.truncated {
		b.WriteString("- Context truncated to --max-chars.\n")
		b.WriteString("- [truncated]\n")
	}
	b.WriteString("\n## Handoff Instruction\n\n")
	if includeDetails {
		b.WriteString("Continue from this prior AI coding session. Treat the original CWD as historical context and the target CWD, when present, as the active working directory.")
	} else {
		b.WriteString("Continue from this prior AI coding session.")
	}
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

func initialGoal(turns []core.Turn, limit int) string {
	for _, turn := range turns {
		if turn.Role == core.RoleUser && strings.TrimSpace(turn.Text) != "" {
			return limitText(turn.Text, limit)
		}
	}
	return "Unavailable"
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
