package readers

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCodexStorageReaderListsAndReadsRollout(t *testing.T) {
	root := t.TempDir()
	rollout := filepath.Join(root, "sessions", "2026", "07", "07", "rollout.jsonl")
	if err := os.MkdirAll(filepath.Dir(rollout), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rollout, []byte(`{"timestamp":"2026-07-07T00:00:01Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"Design cross AI history"}]}}
{"timestamp":"2026-07-07T00:00:02Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Use a CLI."}]}}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	writeCodexState(t, filepath.Join(root, "state_5.sqlite"), "abc", rollout)

	reader := NewCodexStorageReader([]string{root})
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "codex:abc" || sessions[0].Project != "demo" {
		t.Fatalf("unexpected sessions: %+v", sessions)
	}
	if sessions[0].TurnCount != 2 {
		t.Fatalf("list turn count: %d", sessions[0].TurnCount)
	}

	detail, err := reader.GetSession("abc")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if len(detail.Turns) != 2 || detail.Turns[0].Text != "Design cross AI history" {
		t.Fatalf("unexpected turns: %+v", detail.Turns)
	}
}

func TestCodexStorageReaderHandlesLongRolloutLine(t *testing.T) {
	root := t.TempDir()
	rollout := filepath.Join(root, "sessions", "2026", "07", "08", "rollout.jsonl")
	if err := os.MkdirAll(filepath.Dir(rollout), 0o700); err != nil {
		t.Fatal(err)
	}
	longText := strings.Repeat("x", 200000) // exceeds the 64 KiB default scanner token limit
	line := fmt.Sprintf(`{"timestamp":"2026-07-08T00:00:01Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":%s}]}}`+"\n",
		strconv.Quote(longText))
	if err := os.WriteFile(rollout, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	writeCodexState(t, filepath.Join(root, "state_5.sqlite"), "abc", rollout)

	reader := NewCodexStorageReader([]string{root})
	detail, err := reader.GetSession("abc")
	if err != nil {
		t.Fatalf("long rollout line must not fail: %v", err)
	}
	if len(detail.Turns) != 1 || len(detail.Turns[0].Text) != 200000 {
		t.Fatalf("unexpected turns: %+v", detail.Turns)
	}
}

func writeCodexState(t *testing.T, path string, id string, rollout string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE threads (
		id TEXT PRIMARY KEY,
		rollout_path TEXT NOT NULL,
		created_at_ms INTEGER,
		updated_at_ms INTEGER,
		cwd TEXT,
		title TEXT,
		preview TEXT,
		archived INTEGER
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO threads VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, rollout, int64(1783382400000), int64(1783382402000),
		"/Users/alice/Workspace/demo", "Codex Title", "Design cross AI history", 0)
	if err != nil {
		t.Fatal(err)
	}
}
