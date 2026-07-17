package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"

	"github.com/yangkushu/ai-session-history/internal/core"
)

const listTitleWidth = 80

func compactAge(at time.Time, now time.Time) string {
	age := now.Sub(at)
	switch {
	case age < time.Minute:
		return "now"
	case age < time.Hour:
		return fmt.Sprintf("%dm", age/time.Minute)
	case age < 24*time.Hour:
		return fmt.Sprintf("%dh", age/time.Hour)
	case age < 30*24*time.Hour:
		return fmt.Sprintf("%dd", age/(24*time.Hour))
	case age < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", age/(30*24*time.Hour))
	default:
		return fmt.Sprintf("%dy", age/(365*24*time.Hour))
	}
}

func formatListTime(value *time.Time, now time.Time, loc *time.Location) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%s (%s)", value.In(loc).Format("2006-01-02 15:04"), compactAge(*value, now))
}

func formatListTitle(title string) string {
	normalized := strings.Join(strings.Fields(title), " ")
	if runewidth.StringWidth(normalized) <= listTitleWidth {
		return normalized
	}
	return runewidth.Truncate(normalized, listTitleWidth-runewidth.StringWidth("…"), "") + "…"
}

func writeListText(w io.Writer, sessions []core.SessionSummary, now time.Time, loc *time.Location) error {
	sourceWidth := 0
	for _, session := range sessions {
		sourceWidth = max(sourceWidth, runewidth.StringWidth(string(session.Source)))
	}
	indent := strings.Repeat(" ", sourceWidth+2)

	for i, session := range sessions {
		if i > 0 {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}

		source := string(session.Source)
		gap := strings.Repeat(" ", sourceWidth-runewidth.StringWidth(source)+2)
		cwd := session.CWD
		if cwd == "" {
			cwd = "unknown"
		}
		block := source + gap + formatListTitle(session.Title) + "\n" +
			indent + formatListTime(session.UpdatedAt, now, loc) + "  " + formatListTime(session.CreatedAt, now, loc) + "\n" +
			indent + cwd + "\n" +
			indent + session.ID + "\n"
		if _, err := io.WriteString(w, block); err != nil {
			return err
		}
	}
	return nil
}
