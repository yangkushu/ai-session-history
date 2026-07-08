package history

import (
	"fmt"
	"strings"
)

type ContextOptions struct {
	TargetCWD string
	MaxChars  int
}

func BuildContext(detail SessionDetail, options ContextOptions) string {
	maxChars := options.MaxChars
	if maxChars <= 0 {
		maxChars = 20000
	}

	summary := detail.Summary
	lines := []string{
		"# AI Session Context",
		"",
		"## Session",
		"",
		fmt.Sprintf("- ID: %s", summary.ID),
		fmt.Sprintf("- Source: %s", summary.Source),
		fmt.Sprintf("- Original CWD: %s", valueOrUnknown(summary.CWD)),
	}
	if options.TargetCWD != "" {
		lines = append(lines, fmt.Sprintf("- Target CWD: %s", options.TargetCWD))
	}
	lines = append(lines,
		fmt.Sprintf("- Created: %s", formatTime(summary.CreatedAt)),
		fmt.Sprintf("- Updated: %s", formatTime(summary.UpdatedAt)),
		"",
		"## Initial Goal",
		"",
		initialGoal(detail.Turns),
		"",
		"## Recent Conversation",
		"",
	)

	for _, turn := range recentConversation(detail.Turns) {
		lines = append(lines, fmt.Sprintf("**%s**", strings.Title(string(turn.Role))), "", turn.Text, "")
	}

	lines = append(lines,
		"## Omitted Content",
		"",
		"- Tool output omitted when applicable.",
		"- Transcript truncated when applicable.",
		"",
		"## Handoff Instruction",
		"",
		"Continue from this prior AI coding session. Treat the original CWD as historical context and the target CWD, when present, as the active working directory.",
	)

	out := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	if len(out) <= maxChars {
		return out
	}
	marker := "\n\n[truncated]"
	if maxChars <= len(marker) {
		return marker[:maxChars]
	}
	return strings.TrimSpace(out[:maxChars-len(marker)]) + marker
}

func initialGoal(turns []Turn) string {
	for _, turn := range turns {
		if turn.Role == RoleUser && strings.TrimSpace(turn.Text) != "" {
			return turn.Text
		}
	}
	return "unknown"
}

func recentConversation(turns []Turn) []Turn {
	messages := make([]Turn, 0, len(turns))
	for _, turn := range turns {
		if (turn.Role == RoleUser || turn.Role == RoleAssistant) && strings.TrimSpace(turn.Text) != "" {
			messages = append(messages, turn)
		}
	}
	if len(messages) <= 3 {
		return messages
	}
	return messages[len(messages)-2:]
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func formatTime(value interface {
	IsZero() bool
	Format(string) string
}) string {
	if value.IsZero() {
		return "unknown"
	}
	return value.Format("2006-01-02T15:04:05Z07:00")
}
