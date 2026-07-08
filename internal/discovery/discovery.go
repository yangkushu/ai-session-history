package discovery

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

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

// isWSL reports whether the given /proc/version content indicates a WSL kernel.
func isWSL(procVersion string) bool {
	return strings.Contains(strings.ToLower(procVersion), "microsoft")
}

// windowsCursorRootsUnder globs a mount tree for Windows Cursor user dirs of
// the form <mount>/<drive>/Users/<user>/AppData/Roaming/Cursor/User and returns
// the ones that exist.
func windowsCursorRootsUnder(mountRoot string) []string {
	pattern := filepath.Join(mountRoot, "*", "Users", "*", "AppData", "Roaming", "Cursor", "User")
	matches, _ := filepath.Glob(pattern)
	roots := []string{}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err == nil && info.IsDir() {
			roots = append(roots, match)
		}
	}
	return roots
}

// cursorRootsFor returns Cursor roots for an injectable environment. It is the
// testable composition of DefaultPaths plus WSL→Windows discovery.
func cursorRootsFor(goos, home string, env map[string]string, procVersion, mountRoot string) []string {
	roots := DefaultPaths(core.SourceCursor, goos, home, env)
	if goos == "linux" && isWSL(procVersion) && mountRoot != "" {
		roots = append(roots, windowsCursorRootsUnder(mountRoot)...)
	}
	return roots
}
