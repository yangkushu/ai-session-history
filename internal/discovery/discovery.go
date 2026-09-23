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
	case core.SourcePi:
		if env != nil && env["PI_CODING_AGENT_SESSION_DIR"] != "" {
			return []string{expandPiPath(env["PI_CODING_AGENT_SESSION_DIR"], home)}
		}
		if env != nil && env["PI_CODING_AGENT_DIR"] != "" {
			return []string{filepath.Join(expandPiPath(env["PI_CODING_AGENT_DIR"], home), "sessions")}
		}
		return []string{filepath.Join(home, ".pi", "agent", "sessions")}
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

// ResolveRoots returns the default roots for a source, including WSL→Windows
// Cursor discovery when running on a WSL host. Configured paths from the config
// file are added by the caller; this only resolves default-path roots.
func ResolveRoots(source core.Source) []string {
	if source == core.SourcePi {
		cwd, _ := os.Getwd()
		return PiRoots(nil, true, cwd, "", processEnvironment("PI_CODING_AGENT_DIR", "PI_CODING_AGENT_SESSION_DIR"))
	}
	if source == core.SourceCursor {
		procVersion, _ := os.ReadFile("/proc/version")
		return cursorRootsFor(runtime.GOOS, "", nil, string(procVersion), "/mnt")
	}
	return DefaultPaths(source, "", "", nil)
}

// cursorRootsFor returns Cursor roots for an injectable environment. It is the
// testable composition of DefaultPaths plus WSL→Windows discovery.
// PiRoots returns configured roots followed by at most one environment/default root.
// Relative paths resolve against cwd; '~' and '~/...' expand against home.
func PiRoots(configured []string, useDefaultPaths bool, cwd string, home string, env map[string]string) []string {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	roots := make([]string, 0, len(configured)+1)
	seen := map[string]bool{}
	appendRoot := func(path string) {
		if path == "" {
			return
		}
		path = expandPiPath(path, home)
		if !filepath.IsAbs(path) {
			if cwd != "" {
				path = filepath.Join(cwd, path)
			} else {
				path, _ = filepath.Abs(path)
			}
		}
		path = filepath.Clean(path)
		if !seen[path] {
			seen[path] = true
			roots = append(roots, path)
		}
	}
	for _, path := range configured {
		appendRoot(path)
	}
	if useDefaultPaths {
		for _, path := range DefaultPaths(core.SourcePi, "", home, env) {
			appendRoot(path)
		}
	}
	return roots
}

func expandPiPath(path string, home string) string {
	if path == "~" {
		return home
	}
	if expanded, ok := strings.CutPrefix(path, "~/"); ok {
		return filepath.Join(home, expanded)
	}
	return path
}

func processEnvironment(keys ...string) map[string]string {
	env := make(map[string]string, len(keys))
	for _, key := range keys {
		env[key] = os.Getenv(key)
	}
	return env
}

func cursorRootsFor(goos, home string, env map[string]string, procVersion, mountRoot string) []string {
	roots := DefaultPaths(core.SourceCursor, goos, home, env)
	if goos == "linux" && isWSL(procVersion) && mountRoot != "" {
		roots = append(roots, windowsCursorRootsUnder(mountRoot)...)
	}
	return roots
}
