package readers

import (
	"path/filepath"
	"strings"

	"github.com/yangkushu/ai-session-history/internal/core"
)

type StorageReader interface {
	ListSessions() ([]core.SessionSummary, error)
	GetSession(nativeID string) (core.SessionDetail, error)
}

func projectFromCWD(cwd string) string {
	if cwd == "" {
		return ""
	}
	return filepath.Base(cwd)
}

func previewText(text string, limit int) string {
	collapsed := strings.Join(strings.Fields(text), " ")
	if len(collapsed) <= limit {
		return collapsed
	}
	prefix := strings.TrimSpace(collapsed[:limit])
	if index := strings.LastIndex(prefix, " "); index > 0 {
		prefix = prefix[:index]
	}
	return prefix + "..."
}

func titleFromTurns(turns []core.Turn, fallback string) string {
	for _, turn := range turns {
		if turn.Role == core.RoleUser && strings.TrimSpace(turn.Text) != "" {
			return previewText(turn.Text, 80)
		}
	}
	return fallback
}
