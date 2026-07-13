package readers

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/yangkushu/ai-session-history/internal/core"
)

func TestClaudeStorageReaderReportsPermissionDeniedProjectPath(t *testing.T) {
	root := t.TempDir()
	projectsPath := filepath.Join(root, "projects")
	reader := NewClaudeStorageReader([]string{root})
	reader.readDir = func(path string) ([]os.DirEntry, error) {
		return nil, &os.PathError{Op: "readdir", Path: path, Err: fs.ErrPermission}
	}

	diagnostic := reader.Doctor()
	if diagnostic.Status != "unavailable" || diagnostic.Code != core.ErrPermissionDenied || diagnostic.Path != projectsPath {
		t.Fatalf("unexpected diagnostic: %+v", diagnostic)
	}

	_, err := reader.ListSessions()
	var appErr *core.AppError
	if !errors.As(err, &appErr) || appErr.Code != core.ErrPermissionDenied || appErr.Path != projectsPath {
		t.Fatalf("unexpected list error: %#v", err)
	}
}

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

func TestClaudeStorageReaderHandlesLongJSONLLine(t *testing.T) {
	root := t.TempDir()
	session := filepath.Join(root, "projects", "-example-big", "big.jsonl")
	if err := os.MkdirAll(filepath.Dir(session), 0o700); err != nil {
		t.Fatal(err)
	}
	longText := strings.Repeat("x", 200000) // exceeds the 64 KiB default scanner token limit
	line := fmt.Sprintf(`{"type":"user","sessionId":"big","timestamp":"2026-07-08T00:00:01Z","cwd":"/example/big","message":{"role":"user","content":%s}}`+"\n",
		strconv.Quote(longText))
	if err := os.WriteFile(session, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}

	reader := NewClaudeStorageReader([]string{root})
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatalf("long line must not fail the source: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("want 1 session, got %d", len(sessions))
	}
	detail, err := reader.GetSession("big")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(detail.Turns) != 1 || len(detail.Turns[0].Text) != 200000 {
		t.Fatalf("unexpected turns: %+v", detail.Turns)
	}
}

func TestClaudeStorageReaderSkipsUnparseableFile(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, "projects", "-example-good", "good.jsonl")
	bad := filepath.Join(root, "projects", "-example-bad", "bad.jsonl")
	for _, p := range []string{good, bad} {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(good, []byte(`{"type":"user","sessionId":"good","timestamp":"2026-07-08T00:00:01Z","cwd":"/example/good","message":{"role":"user","content":"hello"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("{not valid json\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	reader := NewClaudeStorageReader([]string{root})
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatalf("one bad file must not fail the whole source: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "claude:good" {
		t.Fatalf("want only the good session, got %+v", sessions)
	}
}
