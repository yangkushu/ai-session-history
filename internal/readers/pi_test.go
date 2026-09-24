package readers

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangkushu/ai-session-history/internal/core"
)

func TestPiStorageReaderReadsVersionedFixturesAndActiveBranch(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "pi", "sessions")
	reader := NewPiStorageReader([]string{root})
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatalf("list Pi sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("want v1 and v3 sessions, got %+v", sessions)
	}
	var v3 core.SessionSummary
	for _, session := range sessions {
		if session.NativeID == "pi-fixture-v3" {
			v3 = session
		}
	}
	if v3.ID != "pi:pi-fixture-v3" || v3.Source != core.SourcePi || v3.Title != "Pi Fixture Session" || v3.CWD != "/example/pi-demo" {
		t.Fatalf("unexpected v3 summary: %+v", v3)
	}
	if v3.Project != "pi-demo" || v3.CreatedAt == nil || v3.UpdatedAt == nil {
		t.Fatalf("missing normalized project or timestamps: %+v", v3)
	}

	detail, err := reader.GetSession("pi-fixture-v3")
	if err != nil {
		t.Fatalf("get v3 session: %v", err)
	}
	if len(detail.Turns) == 0 {
		t.Fatal("expected active branch turns")
	}
	joined := joinPiTurnText(detail.Turns)
	for _, want := range []string{
		"Build a Pi history reader",
		"Continue with the active branch",
		"Persisted compaction: preserve the parser and branch semantics",
		"Tool call: read",
		"Concise tool result",
		"The active branch is ready.",
		"Terminal reports all checks passed.",
		"Persisted branch summary: verify the active branch output",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected %q in Pi turns:\n%s", want, joined)
		}
	}
	for _, excluded := range []string{"abandoned sibling branch", "PRIVATE THINKING", "PRIVATE TOOL ARGUMENT", "PRIVATE BASE64", "PRIVATE BASH COMMAND"} {
		if strings.Contains(joined, excluded) {
			t.Errorf("opaque or inactive content %q leaked into Pi turns:\n%s", excluded, joined)
		}
	}
	if !containsPiSummary(detail.Turns, core.SummaryCompaction) || !containsPiSummary(detail.Turns, core.SummaryBranchSummary) {
		t.Fatalf("expected persisted compaction and branch summary kinds in turns: %+v", detail.Turns)
	}

	legacy, err := reader.GetSession("pi-fixture-v1")
	if err != nil {
		t.Fatalf("get v1 session: %v", err)
	}
	if len(legacy.Turns) != 2 || legacy.Turns[0].Text != "Legacy linear session" || legacy.Turns[1].Text != "Legacy response" {
		t.Fatalf("unexpected v1 linear turns: %+v", legacy.Turns)
	}
}

func TestPiStorageReaderSupportsVersionTwoActiveBranch(t *testing.T) {
	root := t.TempDir()
	writePiTestFile(t, root, "v2.jsonl", `{"type":"session","version":2,"id":"v2-pi","timestamp":"2026-01-01T00:00:00Z","cwd":"/work/pi-v2"}`+"\n"+
		`{"type":"message","id":"root","parentId":null,"message":{"role":"user","content":"v2 goal"}}`+"\n"+
		`{"type":"message","id":"inactive","parentId":"root","message":{"role":"user","content":"v2 inactive branch"}}`+"\n"+
		`{"type":"message","id":"active","parentId":"root","message":{"role":"assistant","content":[{"type":"text","text":"v2 active branch"}]}}`+"\n")
	reader := NewPiStorageReader([]string{root})
	detail, err := reader.GetSession("v2-pi")
	if err != nil {
		t.Fatal(err)
	}
	joined := joinPiTurnText(detail.Turns)
	if !strings.Contains(joined, "v2 goal") || !strings.Contains(joined, "v2 active branch") || strings.Contains(joined, "v2 inactive branch") {
		t.Fatalf("v2 active branch projection is incorrect: %s", joined)
	}
}

func TestPiStorageReaderMapsMissingAndPermissionErrors(t *testing.T) {
	reader := NewPiStorageReader(nil)
	if diagnostic := reader.Doctor(); diagnostic.Status != "unavailable" || diagnostic.Code != core.ErrSourceUnavailable {
		t.Fatalf("missing storage must map to source_unavailable: %+v", diagnostic)
	}
	if _, err := reader.GetSession("absent"); !core.IsCode(err, core.ErrSessionNotFound) {
		t.Fatalf("unknown Pi ID must map to session_not_found: %v", err)
	}
	warning := piWarning("/denied/session.jsonl", pathInspectionError(core.SourcePi, "/denied/session.jsonl", fs.ErrPermission))
	if warning.Code != core.ErrPermissionDenied || warning.Path != "/denied/session.jsonl" {
		t.Fatalf("permission error must retain code and path: %+v", warning)
	}
}

func TestPiStorageReaderIsolatesCorruptAndFutureFiles(t *testing.T) {
	root := t.TempDir()
	writePiTestFile(t, root, "valid.jsonl", `{"type":"session","version":3,"id":"valid-pi","timestamp":"2026-01-01T00:00:00Z","cwd":"/work/pi"}`+"\n"+
		`{"type":"message","id":"u1","parentId":null,"timestamp":"2026-01-01T00:00:01Z","message":{"role":"user","content":"hello pi"}}`+"\n")
	writePiTestFile(t, root, "bad.jsonl", "{\"type\":\"session\",\"version\":3,\"id\":\"bad-pi\",\"timestamp\":\"2026-01-01T00:00:00Z\",\"cwd\":\"/work/pi\"}\n{\"type\":\"message\",\"id\":\"x\",\"parentId\":null,\"message\":oops}\n")
	writePiTestFile(t, root, "future.jsonl", `{"type":"session","version":4,"id":"future-pi","timestamp":"2026-01-01T00:00:00Z","cwd":"/work/pi"}`+"\n")
	writePiTestFile(t, root, "missing-header.jsonl", `{"type":"message","role":"user"}`+"\n")

	reader := NewPiStorageReader([]string{root})
	sessions, warnings, err := reader.ListSessionsWithDiagnostics()
	if err != nil {
		t.Fatalf("one valid session should make listing partial-success: %v", err)
	}
	if len(sessions) != 1 || sessions[0].NativeID != "valid-pi" {
		t.Fatalf("invalid neighbors hid valid session: %+v", sessions)
	}
	if len(warnings) != 3 {
		t.Fatalf("expected warnings for corrupt, future, and missing-header files, got %+v", warnings)
	}
	for _, warning := range warnings {
		if warning.Code != core.ErrUnsupportedFormat || warning.Path == "" || warning.Message == "" {
			t.Errorf("malformed warning: %+v", warning)
		}
	}
	diagnostic := reader.Doctor()
	if diagnostic.Status != "partial" || len(diagnostic.Warnings) != 3 {
		t.Fatalf("unexpected partial doctor result: %+v", diagnostic)
	}
	_, err = reader.GetSession("bad-pi")
	if !core.IsCode(err, core.ErrUnsupportedFormat) || !strings.Contains(err.Error(), "bad.jsonl") {
		t.Fatalf("known corrupt session must return pathful unsupported_format: %v", err)
	}
}

func TestPiStorageReaderSkipsEmptyLinesAndReportsOversizedLine(t *testing.T) {
	root := t.TempDir()
	writePiTestFile(t, root, "valid.jsonl", "\n"+
		`{"type":"session","version":3,"id":"valid-pi","timestamp":"2026-01-01T00:00:00Z","cwd":"/work/pi"}`+"\n\n"+
		`{"type":"message","id":"u1","parentId":null,"timestamp":"2026-01-01T00:00:01Z","message":{"role":"user","content":"valid"}}`+"\n")
	writePiTestFile(t, root, "too-large.jsonl", `{"type":"session","version":3,"id":"too-large","timestamp":"2026-01-01T00:00:00Z","cwd":"/work/pi"}`+"\n"+
		strings.Repeat("x", maxPiJSONLLineBytes+1)+"\n")

	reader := NewPiStorageReader([]string{root})
	sessions, warnings, err := reader.ListSessionsWithDiagnostics()
	if err != nil {
		t.Fatalf("oversized neighbor must not hide valid session: %v", err)
	}
	if len(sessions) != 1 || sessions[0].NativeID != "valid-pi" {
		t.Fatalf("unexpected sessions: %+v", sessions)
	}
	if len(warnings) != 1 || warnings[0].Code != core.ErrUnsupportedFormat || !strings.Contains(warnings[0].Message, "exceeds") {
		t.Fatalf("expected bounded-line warning: %+v", warnings)
	}
}

func TestPiStorageReaderPreservesFirstDuplicateIDByRootPriority(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	writePiTestFile(t, first, "session.jsonl", `{"type":"session","version":3,"id":"duplicate","timestamp":"2026-01-01T00:00:00Z","cwd":"/first"}`+"\n"+
		`{"type":"message","id":"u1","parentId":null,"timestamp":"2026-01-01T00:00:01Z","message":{"role":"user","content":"first root"}}`+"\n")
	writePiTestFile(t, second, "session.jsonl", `{"type":"session","version":3,"id":"duplicate","timestamp":"2026-01-02T00:00:00Z","cwd":"/second"}`+"\n"+
		`{"type":"message","id":"u2","parentId":null,"timestamp":"2026-01-02T00:00:01Z","message":{"role":"user","content":"second root"}}`+"\n")

	reader := NewPiStorageReader([]string{first, second})
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].CWD != "/first" {
		t.Fatalf("expected first root to win duplicate ID: %+v", sessions)
	}
	detail, err := reader.GetSession("duplicate")
	if err != nil || !strings.Contains(joinPiTurnText(detail.Turns), "first root") {
		t.Fatalf("show must resolve same duplicate winner: detail=%+v err=%v", detail, err)
	}
}

func TestPiStorageReaderDoesNotLetMalformedDuplicateShadowValidSession(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	writePiTestFile(t, first, "broken.jsonl", `{"type":"session","version":3,"id":"same-id","timestamp":"2026-01-01T00:00:00Z","cwd":"/broken"}`+"\n"+
		`{"type":"message","id":"broken","parentId":null,"message":invalid}`+"\n")
	writePiTestFile(t, second, "valid.jsonl", `{"type":"session","version":3,"id":"same-id","timestamp":"2026-01-02T00:00:00Z","cwd":"/valid"}`+"\n"+
		`{"type":"message","id":"valid","parentId":null,"message":{"role":"user","content":"valid duplicate"}}`+"\n")
	reader := NewPiStorageReader([]string{first, second})
	sessions, warnings, err := reader.ListSessionsWithDiagnostics()
	if err != nil || len(sessions) != 1 || sessions[0].CWD != "/valid" || len(warnings) != 1 {
		t.Fatalf("malformed duplicate must not shadow valid copy: sessions=%+v warnings=%+v err=%v", sessions, warnings, err)
	}
	detail, err := reader.GetSession("same-id")
	if err != nil || !strings.Contains(joinPiTurnText(detail.Turns), "valid duplicate") {
		t.Fatalf("show must resolve valid duplicate after malformed copy: detail=%+v err=%v", detail, err)
	}
}

func TestPiStorageReaderIgnoresPiSubagentTranscriptArtifacts(t *testing.T) {
	root := t.TempDir()
	writePiTestFile(t, root, filepath.Join("--example-project--", "session.jsonl"), `{"type":"session","version":3,"id":"valid-session","timestamp":"2026-01-01T00:00:00Z","cwd":"/example/project"}`+"\n"+
		`{"type":"message","id":"u1","parentId":null,"message":{"role":"user","content":"valid Pi session"}}`+"\n")
	writePiTestFile(t, root, filepath.Join("--example-project--", "subagent-artifacts", "run_transcript.jsonl"), `{"version":1,"recordType":"message","source":"subagent","runId":"run-1","agent":"worker","cwd":"/example/project","sourceEventType":"initial_prompt","role":"user","text":"synthetic sidecar transcript","message":{"role":"user","content":[{"type":"text","text":"synthetic sidecar transcript"}]}}`+"\n")

	reader := NewPiStorageReader([]string{root})
	sessions, warnings, err := reader.ListSessionsWithDiagnostics()
	if err != nil || len(sessions) != 1 || sessions[0].NativeID != "valid-session" || len(warnings) != 0 {
		t.Fatalf("Pi-subagents transcripts must not be parsed or reported as Pi sessions: sessions=%+v warnings=%+v err=%v", sessions, warnings, err)
	}
}

func TestPiStorageReaderDoesNotFollowSymlinkDirectory(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	writePiTestFile(t, outside, "outside.jsonl", `{"type":"session","version":3,"id":"outside","timestamp":"2026-01-01T00:00:00Z","cwd":"/work"}`+"\n")
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	reader := NewPiStorageReader([]string{root})
	if sessions, err := reader.ListSessions(); err == nil || len(sessions) != 0 {
		t.Fatalf("must not follow symlink directory: sessions=%+v err=%v", sessions, err)
	}
}

func writePiTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func joinPiTurnText(turns []core.Turn) string {
	parts := make([]string, 0, len(turns))
	for _, turn := range turns {
		parts = append(parts, turn.Text)
	}
	return strings.Join(parts, "\n")
}

func containsPiSummary(turns []core.Turn, kind core.PersistedSummaryKind) bool {
	for _, turn := range turns {
		if turn.Kind == core.KindPersistedSummary && turn.SummaryKind == kind {
			return true
		}
	}
	return false
}
