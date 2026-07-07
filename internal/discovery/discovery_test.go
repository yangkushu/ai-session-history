package discovery

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yangkushu/ai-session-history/internal/core"
)

func TestDiscoverCodexAndClaudeDefaults(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "Users", "alice")

	codex := DefaultPaths(core.SourceCodex, "darwin", home, nil)
	claude := DefaultPaths(core.SourceClaude, "darwin", home, nil)

	if codex[0] != filepath.Join(home, ".codex") {
		t.Fatalf("unexpected codex path: %v", codex)
	}
	if claude[0] != filepath.Join(home, ".claude") {
		t.Fatalf("unexpected claude path: %v", claude)
	}
}

func TestDiscoverCursorDefaultsForMacAndWindows(t *testing.T) {
	macHome := filepath.Join(string(filepath.Separator), "Users", "alice")
	winHome := `C:\Users\Alice`

	mac := DefaultPaths(core.SourceCursor, "darwin", macHome, nil)
	win := DefaultPaths(core.SourceCursor, "windows", winHome, map[string]string{"APPDATA": `C:\Users\Alice\AppData\Roaming`})

	if runtime.GOOS != "windows" && mac[0] != filepath.Join(macHome, "Library", "Application Support", "Cursor", "User") {
		t.Fatalf("unexpected mac cursor path: %v", mac)
	}
	if win[0] != filepath.Join(`C:\Users\Alice\AppData\Roaming`, "Cursor", "User") {
		t.Fatalf("unexpected windows cursor path: %v", win)
	}
}
