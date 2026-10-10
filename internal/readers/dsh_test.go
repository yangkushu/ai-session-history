package readers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/yangkushu/ai-session-history/internal/core"
)

func writeDshTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir dsh fixture: %v", err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create dsh fixture: %v", err)
	}
	defer file.Close()
	writer, err := zstd.NewWriter(file)
	if err != nil {
		t.Fatalf("init zstd writer: %v", err)
	}
	if _, err := writer.Write([]byte(content)); err != nil {
		t.Fatalf("write dsh fixture: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("flush dsh fixture: %v", err)
	}
}

func dshFixtureJSONL(id string) string {
	return `{"type":"session","version":4,"id":"` + id + `","createdAt":1791613914004,"cwd":"/work/dsh-demo"}` + "\n" +
		`{"type":"user/message","seq":1,"time":1791613928800,"data":{"content":[{"type":"text","text":"dsh goal"}],"role":"user"}}` + "\n" +
		`{"type":"assistant/message","seq":2,"time":1791613941767,"data":{"message":{"role":"assistant","content":[{"type":"text","text":"dsh answer"}]}}}` + "\n"
}

func TestDshStorageReaderReadsCompressedFixtures(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "dsh", "sessions")
	reader := NewDshStorageReader([]string{root})
	sessions, warnings, err := reader.ListSessionsWithDiagnostics()
	if err != nil {
		t.Fatalf("list dsh sessions: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %+v", warnings)
	}
	if len(sessions) != 1 {
		t.Fatalf("want one dsh session, got %+v", sessions)
	}
	summary := sessions[0]
	if summary.ID != "dsh:session-dsh-fixture-a" || summary.Source != core.SourceDsh || summary.NativeID != "session-dsh-fixture-a" {
		t.Fatalf("unexpected dsh summary id fields: %+v", summary)
	}
	if summary.Title != "Add dsh history support" {
		t.Fatalf("unexpected title: %q", summary.Title)
	}
	if summary.CWD != "/home/demo/workspaces/dsh-demo" || summary.Project != "dsh-demo" {
		t.Fatalf("unexpected cwd/project: %+v", summary)
	}
	if summary.CreatedAt == nil || summary.UpdatedAt == nil {
		t.Fatalf("missing timestamps: %+v", summary)
	}
	if summary.UpdatedAt.UnixMilli() != 1791614025204 {
		t.Fatalf("updated at should track the latest event time, got %v", summary.UpdatedAt)
	}

	detail, err := reader.GetSession("session-dsh-fixture-a")
	if err != nil {
		t.Fatalf("get dsh session: %v", err)
	}
	joined := joinDshTurnText(detail.Turns)
	for _, want := range []string{
		"Help me add dsh support to the history tool",
		"I will inspect the project layout first.",
		"Tool call: bash",
		"internal main.go",
		"The reader is implemented; run the tests next.",
		"FAILED: expected no error",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected %q in dsh turns:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "PRIVATE REASONING") || strings.Contains(joined, "duplicated spliced message") {
		t.Errorf("internal or spliced content leaked into dsh turns:\n%s", joined)
	}
	errorTurns := 0
	for _, turn := range detail.Turns {
		if turn.Kind == core.KindError {
			errorTurns++
		}
	}
	if errorTurns != 1 {
		t.Fatalf("want exactly one error tool result, got %d in %+v", errorTurns, detail.Turns)
	}
}

func TestDshStorageReaderListAndShowTempSessions(t *testing.T) {
	root := t.TempDir()
	writeDshTestFile(t, filepath.Join(root, "--work-dsh-demo--", "session-temp-1", "session.v4.jsonl.zstd"), dshFixtureJSONL("session-temp-1"))
	writeDshTestFile(t, filepath.Join(root, "--work-other--", "session-temp-2", "session.v4.jsonl.zstd"), dshFixtureJSONL("session-temp-2"))

	reader := NewDshStorageReader([]string{root})
	sessions, _, err := reader.ListSessionsWithDiagnostics()
	if err != nil {
		t.Fatalf("list dsh sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("want two dsh sessions, got %+v", sessions)
	}
	detail, err := reader.GetSession("session-temp-2")
	if err != nil {
		t.Fatalf("get dsh session: %v", err)
	}
	if len(detail.Turns) != 2 || detail.Turns[0].Role != core.RoleUser || detail.Turns[1].Role != core.RoleAssistant {
		t.Fatalf("unexpected dsh turns: %+v", detail.Turns)
	}
	if detail.Turns[0].Kind != core.KindMessage || detail.Turns[0].Timestamp == nil {
		t.Fatalf("missing turn metadata: %+v", detail.Turns[0])
	}

	if _, err := reader.GetSession("session-missing"); core.IsCode(err, core.ErrSessionNotFound) == false {
		t.Fatalf("want session-not-found, got %v", err)
	}
}

func TestDshStorageReaderReportsUnreadableSessionsAsWarnings(t *testing.T) {
	root := t.TempDir()
	writeDshTestFile(t, filepath.Join(root, "--work-dsh-demo--", "session-good", "session.v4.jsonl.zstd"), dshFixtureJSONL("session-good"))
	if err := os.MkdirAll(filepath.Join(root, "--work-dsh-demo--", "session-bad"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "--work-dsh-demo--", "session-bad", "session.v4.jsonl.zstd"), []byte("not zstd"), 0o644); err != nil {
		t.Fatalf("write broken fixture: %v", err)
	}

	reader := NewDshStorageReader([]string{root})
	sessions, warnings, err := reader.ListSessionsWithDiagnostics()
	if err != nil {
		t.Fatalf("partial result should not fail: %v", err)
	}
	if len(sessions) != 1 || sessions[0].NativeID != "session-good" {
		t.Fatalf("want the readable session only, got %+v", sessions)
	}
	if len(warnings) != 1 || warnings[0].Code != core.ErrUnsupportedFormat {
		t.Fatalf("want one unsupported-format warning, got %+v", warnings)
	}
}

func TestDshStorageReaderUnavailableWithoutRoots(t *testing.T) {
	reader := NewDshStorageReader([]string{filepath.Join(t.TempDir(), "missing")})
	_, _, err := reader.ListSessionsWithDiagnostics()
	if !core.IsCode(err, core.ErrSourceUnavailable) {
		t.Fatalf("want source-unavailable, got %v", err)
	}
	diagnostic := reader.Doctor()
	if diagnostic.Status != "unavailable" || diagnostic.Code != core.ErrSourceUnavailable {
		t.Fatalf("unexpected doctor diagnostic: %+v", diagnostic)
	}
}

func joinDshTurnText(turns []core.Turn) string {
	parts := make([]string, 0, len(turns))
	for _, turn := range turns {
		parts = append(parts, turn.Text)
	}
	return strings.Join(parts, "\n")
}
