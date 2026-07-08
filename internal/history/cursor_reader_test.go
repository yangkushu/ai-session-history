package history

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCursorStorageReaderReadsLatestComposerDataFromStateDB(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "User", "globalStorage", "state.vscdb")
	createCursorFixture(t, dbPath)

	reader := NewCursorStorageReader([]string{filepath.Join(root, "User")})
	sessions, err := reader.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions returned error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("ListSessions returned %d sessions, want 1", len(sessions))
	}
	summary := sessions[0]
	if summary.ID != "cursor:session-1" {
		t.Fatalf("summary.ID = %q", summary.ID)
	}
	if summary.Project != "demo" {
		t.Fatalf("summary.Project = %q", summary.Project)
	}
	if summary.TurnCount != 2 {
		t.Fatalf("summary.TurnCount = %d", summary.TurnCount)
	}
	if summary.Preview != "Read Cursor history Found composerData." {
		t.Fatalf("summary.Preview = %q", summary.Preview)
	}

	detail, err := reader.GetSession(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}
	if got := detail.Turns[0].Text; got != "Read Cursor history" {
		t.Fatalf("first turn text = %q", got)
	}
	if got := detail.Turns[1].Role; got != RoleAssistant {
		t.Fatalf("second turn role = %q", got)
	}
}

func TestCursorStorageReaderIgnoresEmptyDraftsAndReturnsNotFound(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "User", "globalStorage", "state.vscdb")
	createCursorFixture(t, dbPath)

	reader := NewCursorStorageReader([]string{filepath.Join(root, "User")})
	if _, err := reader.GetSession(context.Background(), "empty-state-draft"); err == nil {
		t.Fatal("GetSession for empty draft returned nil error")
	}
}

func TestCursorStorageReaderRecognizesMinimizedLatestMacOSEmptyFixture(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "User", "globalStorage", "state.vscdb")
	if err := ensureDir(filepath.Dir(dbPath)); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	fixture, err := os.ReadFile(filepath.Join("testdata", "cursor_macos_v16_empty.sql"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if _, err := db.Exec(string(fixture)); err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close fixture db: %v", err)
	}

	reader := NewCursorStorageReader([]string{filepath.Join(root, "User")})
	diagnostic := reader.Doctor(context.Background())
	if !diagnostic.Available {
		t.Fatalf("Doctor did not recognize latest macOS fixture: %#v", diagnostic)
	}
	sessions, err := reader.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions returned error: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("empty composer fixture returned %d sessions", len(sessions))
	}
}

func createCursorFixture(t *testing.T, dbPath string) {
	t.Helper()
	if err := ensureDir(filepath.Dir(dbPath)); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE cursorDiskKV (key TEXT UNIQUE ON CONFLICT REPLACE, value BLOB)`); err != nil {
		t.Fatalf("create cursorDiskKV: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE composerHeaders (composerId TEXT PRIMARY KEY, workspaceId TEXT, createdAt INTEGER, lastUpdatedAt INTEGER, isArchived INTEGER, isSubagent INTEGER, recency INTEGER, checkpointAt INTEGER, value TEXT)`); err != nil {
		t.Fatalf("create composerHeaders: %v", err)
	}

	payload := `{
		"_v": 16,
		"composerId": "session-1",
		"createdAt": 1783444765134,
		"lastUpdatedAt": 1783444768000,
		"context": {"workspaceRootPath": "/Users/alice/Workspace/demo"},
		"conversation": [
			{"role": "user", "text": "Read Cursor history", "timestamp": 1783444766000},
			{"role": "assistant", "content": [{"text": "Found composerData."}], "timestamp": 1783444767000}
		]
	}`
	if _, err := db.Exec(`INSERT INTO cursorDiskKV (key, value) VALUES (?, ?)`, "composerData:session-1", payload); err != nil {
		t.Fatalf("insert composerData: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO cursorDiskKV (key, value) VALUES (?, NULL)`, "composerData:empty-state-draft"); err != nil {
		t.Fatalf("insert draft: %v", err)
	}
}
