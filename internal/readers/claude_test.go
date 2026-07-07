package readers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeStorageReaderListsAndReadsProjectJSONL(t *testing.T) {
	root := t.TempDir()
	session := filepath.Join(root, "projects", "-Users-alice-Workspace-demo", "claude-session.jsonl")
	if err := os.MkdirAll(filepath.Dir(session), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(session, []byte(`{"type":"user","sessionId":"claude-session","timestamp":"2026-07-07T00:00:01Z","cwd":"/Users/alice/Workspace/demo","message":{"role":"user","content":"How do I recover a session?"}}
{"type":"assistant","sessionId":"claude-session","timestamp":"2026-07-07T00:00:02Z","cwd":"/Users/alice/Workspace/demo","message":{"role":"assistant","content":[{"type":"text","text":"Use the session id."}]}}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	reader := NewClaudeStorageReader([]string{root})
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "claude:claude-session" || sessions[0].Title != "How do I recover a session?" {
		t.Fatalf("unexpected sessions: %+v", sessions)
	}

	detail, err := reader.GetSession("claude-session")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if len(detail.Turns) != 2 || detail.Turns[1].Text != "Use the session id." {
		t.Fatalf("unexpected turns: %+v", detail.Turns)
	}
}
