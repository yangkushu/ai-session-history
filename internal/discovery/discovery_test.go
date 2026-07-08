package discovery

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
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

func TestIsWSLDetectsMicrosoftKernel(t *testing.T) {
	cases := map[string]bool{
		"Linux version 6.6.87.2-microsoft-standard-WSL2 (user@host)": true,
		"Linux version 5.15.153.1-microsoft-standard-WSL2":           true,
		"Linux version 5.15.0-91-generic (user@host)":                false,
		"": false,
	}
	for content, want := range cases {
		if got := isWSL(content); got != want {
			t.Errorf("isWSL(%q) = %v, want %v", content, got, want)
		}
	}
}

func TestWindowsCursorRootsUnderGlobsExistingUserDirs(t *testing.T) {
	mount := t.TempDir()
	alice := filepath.Join(mount, "c", "Users", "alice", "AppData", "Roaming", "Cursor", "User")
	bob := filepath.Join(mount, "c", "Users", "bob", "AppData", "Roaming", "Cursor", "User")
	for _, p := range []string{alice, bob} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(mount, "c", "Users", "carol", "AppData", "Roaming", "Other"), 0o700); err != nil {
		t.Fatal(err)
	}

	roots := windowsCursorRootsUnder(mount)
	sort.Strings(roots)
	want := []string{alice, bob}
	sort.Strings(want)
	if !reflect.DeepEqual(roots, want) {
		t.Fatalf("got %v, want %v", roots, want)
	}
}

func TestCursorRootsForAddsWSLWindowsRootsOnLinuxWSL(t *testing.T) {
	mount := t.TempDir()
	win := filepath.Join(mount, "c", "Users", "alice", "AppData", "Roaming", "Cursor", "User")
	if err := os.MkdirAll(win, 0o700); err != nil {
		t.Fatal(err)
	}

	roots := cursorRootsFor("linux", "/home/alice", nil,
		"Linux version 6.6.87.2-microsoft-standard-WSL2", mount)

	if !contains(roots, win) {
		t.Fatalf("WSL Windows root not discovered: %v", roots)
	}
	native := filepath.Join("/home/alice", ".config", "Cursor", "User")
	if !contains(roots, native) {
		t.Fatalf("native linux fallback dropped: %v", roots)
	}
}

func TestCursorRootsForSkipsWSLGlobOnPlainLinux(t *testing.T) {
	mount := t.TempDir()
	win := filepath.Join(mount, "c", "Users", "alice", "AppData", "Roaming", "Cursor", "User")
	if err := os.MkdirAll(win, 0o700); err != nil {
		t.Fatal(err)
	}
	roots := cursorRootsFor("linux", "/home/alice", nil,
		"Linux version 5.15.0-91-generic", mount)
	if contains(roots, win) {
		t.Fatalf("plain linux must not glob /mnt: %v", roots)
	}
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
