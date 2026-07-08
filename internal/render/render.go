package render

import (
	"fmt"
	"strings"
	"time"

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
	var b strings.Builder
	s := clean.Summary
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
	b.WriteString(initialGoal(clean.Turns))
	b.WriteString("\n\n## Recent Conversation\n\n")
	for _, turn := range recentConversation(clean.Turns) {
		writeLine(&b, "### %s", titleRole(turn.Role))
		b.WriteString("\n")
		b.WriteString(limitText(turn.Text, 1200))
		b.WriteString("\n\n")
	}
	b.WriteString("## Omitted Content\n\n")
	b.WriteString("- Tool output omitted when applicable.\n")
	if clean.Truncated {
		b.WriteString("- Transcript truncated.\n")
	}
	b.WriteString("\n## Handoff Instruction\n\n")
	b.WriteString("Continue from this prior AI coding session. Treat the original CWD as historical context and the target CWD, when present, as the active working directory.")
	return bound(b.String(), maxChars)
}

func turnText(turn core.Turn, mode core.ContentMode) string {
	if mode == core.ModeRaw {
		return turn.Text
	}
	if turn.Role == core.RoleTool && turn.Kind == core.KindToolResult {
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

func initialGoal(turns []core.Turn) string {
	for _, turn := range turns {
		if turn.Role == core.RoleUser && strings.TrimSpace(turn.Text) != "" {
			return limitText(turn.Text, 1200)
		}
	}
	return "Unknown"
}

func recentConversation(turns []core.Turn) []core.Turn {
	var messages []core.Turn
	for _, turn := range turns {
		if (turn.Role == core.RoleUser || turn.Role == core.RoleAssistant) && strings.TrimSpace(turn.Text) != "" {
			messages = append(messages, turn)
		}
	}
	if len(messages) <= 3 {
		return messages
	}
	return messages[len(messages)-2:]
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

func limitText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return strings.TrimSpace(text[:limit]) + "..."
}

func bound(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	marker := "\n\n[truncated]"
	if maxChars <= len(marker) {
		return text[:maxChars]
	}
	return strings.TrimSpace(text[:maxChars-len(marker)]) + marker
}
