package readers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/yangkushu/ai-session-history/internal/core"
)

func TestCursorStorageReaderReportsPermissionDeniedStatePath(t *testing.T) {
	root := t.TempDir()
	deniedPath := filepath.Join(root, "globalStorage", "state.vscdb")
	reader := NewCursorStorageReader([]string{root})
	reader.stat = func(path string) (os.FileInfo, error) {
		return nil, &os.PathError{Op: "stat", Path: path, Err: fs.ErrPermission}
	}

	diagnostic := reader.Doctor()
	if diagnostic.Status != "unavailable" || diagnostic.Code != core.ErrPermissionDenied || diagnostic.Path != deniedPath {
		t.Fatalf("unexpected diagnostic: %+v", diagnostic)
	}

	_, err := reader.ListSessions()
	var appErr *core.AppError
	if !errors.As(err, &appErr) || appErr.Code != core.ErrPermissionDenied || appErr.Path != deniedPath {
		t.Fatalf("unexpected list error: %#v", err)
	}
}

func TestCursorStorageReaderListsNonArchivedComposers(t *testing.T) {
	root := t.TempDir()
	writeCursorState(t, root,
		[]cursorComposerFixture{
			{ID: "comp-active", Name: "Active composer", CWD: "/example/active",
				CreatedAt: 1783382400000, UpdatedAt: 1783382402000},
			{ID: "comp-archived", Name: "Archived", Archived: true},
		}, nil)

	reader := NewCursorStorageReader([]string{root})
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("want 1 session, got %+v", sessions)
	}
	s := sessions[0]
	if s.ID != "cursor:comp-active" {
		t.Fatalf("id: %s", s.ID)
	}
	if s.Title != "Active composer" {
		t.Fatalf("title: %s", s.Title)
	}
	if s.CWD != "/example/active" {
		t.Fatalf("cwd: %s", s.CWD)
	}
	if s.Project != "active" {
		t.Fatalf("project: %s", s.Project)
	}
}

func TestCursorStorageReaderReadsMessageTurnsAndSkipsEmpty(t *testing.T) {
	root := t.TempDir()
	writeCursorState(t, root,
		[]cursorComposerFixture{
			{ID: "comp-turns", Name: "Turns", CreatedAt: 1783382400000, UpdatedAt: 1783382402000},
		},
		[]cursorBubbleFixture{
			{Composer: "comp-turns", Bubble: "b1", Type: 1, Text: "Example user goal",
				CreatedAt: "2026-07-08T00:00:01.000Z"},
			{Composer: "comp-turns", Bubble: "b2", Type: 2, Text: "",
				CreatedAt: "2026-07-08T00:00:02.000Z"},
			{Composer: "comp-turns", Bubble: "b3", Type: 2, Text: "Example assistant reply",
				CreatedAt: "2026-07-08T00:00:03.000Z"},
			{Composer: "comp-turns", Bubble: "b4", Type: 3, Text: "unknown type bubble",
				CreatedAt: "2026-07-08T00:00:04.000Z"},
		})

	reader := NewCursorStorageReader([]string{root})
	detail, err := reader.GetSession("comp-turns")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(detail.Turns) != 2 {
		t.Fatalf("want 2 turns (empty skipped), got %+v", detail.Turns)
	}
	if detail.Turns[0].Role != core.RoleUser || detail.Turns[0].Text != "Example user goal" {
		t.Fatalf("turn0: %+v", detail.Turns[0])
	}
	if detail.Turns[1].Role != core.RoleAssistant || detail.Turns[1].Text != "Example assistant reply" {
		t.Fatalf("turn1: %+v", detail.Turns[1])
	}
	if detail.Summary.TurnCount != 2 {
		t.Fatalf("turn count: %d", detail.Summary.TurnCount)
	}
}

func TestCursorStorageReaderReadsMacOSComposerDataSessions(t *testing.T) {
	root := t.TempDir()
	writeCursorMacState(t, root, []cursorMacComposerFixture{
		{
			ID:        "mac-composer",
			CWD:       "/Users/alice/Workspace/demo",
			CreatedAt: 1783444765134,
			UpdatedAt: 1783444768000,
			Messages: []cursorMacMessageFixture{
				{Role: "user", Text: "Read Cursor history", Timestamp: 1783444766000},
				{Role: "assistant", Text: "Found composerData.", Timestamp: 1783444767000},
			},
		},
	})

	reader := NewCursorStorageReader([]string{root})
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("want 1 session, got %+v", sessions)
	}
	s := sessions[0]
	if s.ID != "cursor:mac-composer" {
		t.Fatalf("id: %s", s.ID)
	}
	if s.Title != "Read Cursor history" {
		t.Fatalf("title: %s", s.Title)
	}
	if s.CWD != "/Users/alice/Workspace/demo" || s.Project != "demo" {
		t.Fatalf("cwd/project: %s / %s", s.CWD, s.Project)
	}
	if s.Preview != "Read Cursor history Found composerData." {
		t.Fatalf("preview: %s", s.Preview)
	}

	detail, err := reader.GetSession("mac-composer")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(detail.Turns) != 2 {
		t.Fatalf("want 2 turns, got %+v", detail.Turns)
	}
	if detail.Turns[0].Role != core.RoleUser || detail.Turns[1].Role != core.RoleAssistant {
		t.Fatalf("roles: %+v", detail.Turns)
	}
}

func TestCursorStorageReaderRecognizesMacOSEmptyComposerData(t *testing.T) {
	root := t.TempDir()
	writeCursorMacState(t, root, []cursorMacComposerFixture{
		{ID: "sample-empty", CreatedAt: 1783444765134},
	})

	reader := NewCursorStorageReader([]string{root})
	d := reader.Doctor()
	if d.Status != "available" {
		t.Fatalf("status: %+v", d)
	}
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("want no sessions for empty composer, got %+v", sessions)
	}
}

func TestCursorStorageReaderDoctorAvailable(t *testing.T) {
	root := t.TempDir()
	writeCursorState(t, root, []cursorComposerFixture{{ID: "c1"}}, nil)
	reader := NewCursorStorageReader([]string{root})
	d := reader.Doctor()
	if d.Status != "available" {
		t.Fatalf("status: %+v", d)
	}
}

func TestCursorStorageReaderDoctorUnavailableWithoutDB(t *testing.T) {
	reader := NewCursorStorageReader([]string{t.TempDir()})
	d := reader.Doctor()
	if d.Status != "unavailable" || d.Code != core.ErrSourceUnavailable {
		t.Fatalf("unexpected: %+v", d)
	}
}

func TestCursorStorageReaderGetSessionNotFound(t *testing.T) {
	root := t.TempDir()
	writeCursorState(t, root, []cursorComposerFixture{{ID: "c1"}}, nil)
	reader := NewCursorStorageReader([]string{root})
	_, err := reader.GetSession("missing")
	if !core.IsCode(err, core.ErrSessionNotFound) {
		t.Fatalf("err: %v", err)
	}
}

func TestCursorStorageReaderDoctorUnsupportedWithoutComposerTable(t *testing.T) {
	root := t.TempDir()
	writeCursorState(t, root, []cursorComposerFixture{{ID: "c1"}}, nil)
	dbPath := filepath.Join(root, "globalStorage", "state.vscdb")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TABLE composerHeaders`); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	reader := NewCursorStorageReader([]string{root})
	d := reader.Doctor()
	if d.Status != "unsupported_format" || d.Code != core.ErrUnsupportedFormat {
		t.Fatalf("unexpected: %+v", d)
	}
}

type cursorComposerFixture struct {
	ID, Name, CWD        string
	CreatedAt, UpdatedAt int64
	Archived             bool
}

type cursorBubbleFixture struct {
	Composer, Bubble string
	Type             int
	Text, CreatedAt  string
}

type cursorMacComposerFixture struct {
	ID, CWD              string
	CreatedAt, UpdatedAt int64
	Messages             []cursorMacMessageFixture
}

type cursorMacMessageFixture struct {
	Role      string
	Text      string
	Timestamp int64
}

func writeCursorState(t *testing.T, root string, composers []cursorComposerFixture, bubbles []cursorBubbleFixture) {
	t.Helper()
	dbPath := filepath.Join(root, "globalStorage", "state.vscdb")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE composerHeaders (
		composerId TEXT PRIMARY KEY,
		workspaceId TEXT,
		createdAt INTEGER,
		lastUpdatedAt INTEGER,
		isArchived INTEGER,
		isSubagent INTEGER,
		recency INTEGER,
		checkpointAt INTEGER,
		value TEXT
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE cursorDiskKV (key TEXT UNIQUE ON CONFLICT REPLACE, value BLOB)`); err != nil {
		t.Fatal(err)
	}

	for _, c := range composers {
		val := map[string]interface{}{
			"composerId":    c.ID,
			"name":          c.Name,
			"createdAt":     float64(c.CreatedAt),
			"lastUpdatedAt": float64(c.UpdatedAt),
			"isArchived":    c.Archived,
		}
		if c.CWD != "" {
			val["workspaceIdentifier"] = map[string]interface{}{
				"id":  c.ID + "-ws",
				"uri": map[string]interface{}{"path": c.CWD},
			}
		}
		raw, err := json.Marshal(val)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO composerHeaders (composerId, value) VALUES (?, ?)`, c.ID, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	for _, b := range bubbles {
		val := map[string]interface{}{
			"bubbleId":  b.Bubble,
			"type":      float64(b.Type),
			"text":      b.Text,
			"createdAt": b.CreatedAt,
		}
		raw, err := json.Marshal(val)
		if err != nil {
			t.Fatal(err)
		}
		key := "bubbleId:" + b.Composer + ":" + b.Bubble
		if _, err := db.Exec(`INSERT INTO cursorDiskKV (key, value) VALUES (?, ?)`, key, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
}

func writeCursorMacState(t *testing.T, root string, composers []cursorMacComposerFixture) {
	t.Helper()
	dbPath := filepath.Join(root, "globalStorage", "state.vscdb")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE composerHeaders (
		composerId TEXT PRIMARY KEY,
		workspaceId TEXT,
		createdAt INTEGER,
		lastUpdatedAt INTEGER,
		isArchived INTEGER,
		isSubagent INTEGER,
		recency INTEGER,
		checkpointAt INTEGER,
		value TEXT
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE cursorDiskKV (key TEXT UNIQUE ON CONFLICT REPLACE, value BLOB)`); err != nil {
		t.Fatal(err)
	}

	for _, c := range composers {
		messages := make([]map[string]interface{}, 0, len(c.Messages))
		for _, m := range c.Messages {
			messages = append(messages, map[string]interface{}{
				"role":      m.Role,
				"text":      m.Text,
				"timestamp": float64(m.Timestamp),
			})
		}
		val := map[string]interface{}{
			"_v":              float64(16),
			"composerId":      c.ID,
			"richText":        "",
			"hasLoaded":       true,
			"text":            "",
			"conversation":    messages,
			"conversationMap": map[string]interface{}{},
			"createdAt":       float64(c.CreatedAt),
			"lastUpdatedAt":   float64(c.UpdatedAt),
			"context": map[string]interface{}{
				"workspaceRootPath": c.CWD,
			},
		}
		raw, err := json.Marshal(val)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO cursorDiskKV (key, value) VALUES (?, ?)`, "composerData:"+c.ID, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
}
