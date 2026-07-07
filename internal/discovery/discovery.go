package discovery

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/yangkushu/ai-session-history/internal/core"
)

func DefaultPaths(source core.Source, goos string, home string, env map[string]string) []string {
	if goos == "" {
		goos = runtime.GOOS
	}
	if home == "" {
		home, _ = os.UserHomeDir()
	}

	switch source {
	case core.SourceCodex:
		paths := []string{filepath.Join(home, ".codex")}
		if goos == "darwin" {
			paths = append(paths, filepath.Join(home, "Library", "Application Support", "Codex"))
		}
		return paths
	case core.SourceClaude:
		return []string{filepath.Join(home, ".claude")}
	case core.SourceCursor:
		if goos == "darwin" {
			return []string{filepath.Join(home, "Library", "Application Support", "Cursor", "User")}
		}
		if goos == "windows" {
			appdata := ""
			if env != nil {
				appdata = env["APPDATA"]
			}
			if appdata == "" {
				appdata = filepath.Join(home, "AppData", "Roaming")
			}
			return []string{filepath.Join(appdata, "Cursor", "User")}
		}
		return []string{filepath.Join(home, ".config", "Cursor", "User")}
	default:
		return nil
	}
}
