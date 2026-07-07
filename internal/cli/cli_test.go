package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/yangkushu/ai-session-history/internal/core"
)

func TestRunShowsHelpForNoArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(nil, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Usage: ai-history")) {
		t.Fatalf("expected usage in stderr, got %q", stderr.String())
	}
}

func TestSearchCommandIsUnavailable(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"search", "query"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("search is not available in P0")) {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestContextCommandWritesMarkdown(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service := fakeCLIService{contextText: "# AI Session Context\n\nbody"}

	code := RunWithService([]string{"context", "codex:abc", "--target-cwd", "/new"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("# AI Session Context")) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestDoctorCommandWritesJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service := fakeCLIService{diagnostics: []core.SourceDiagnostic{
		{Source: core.SourceCodex, Status: "available"},
	}}

	code := RunWithService([]string{"doctor", "--json"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"source": "codex"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunDoctorBuildsDefaultService(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"doctor", "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"source": "codex"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestListCommandWritesJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	now := time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC)
	service := fakeCLIService{listResult: core.ListResult{Sessions: []core.SessionSummary{
		{ID: "codex:abc", Source: core.SourceCodex, NativeID: "abc", Title: "Title", CWD: "/work/a", UpdatedAt: &now},
	}}}

	code := RunWithService([]string{"list", "--source", "codex", "--under", "/work", "--limit", "5", "--json"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"id": "codex:abc"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestShowCommandWritesJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service := fakeCLIService{detail: core.SessionDetail{
		Summary: core.SessionSummary{ID: "codex:abc", Source: core.SourceCodex, NativeID: "abc", Title: "Title"},
		Turns:   []core.Turn{{Role: core.RoleUser, Text: "hello", Kind: core.KindMessage}},
	}}

	code := RunWithService([]string{"show", "codex:abc", "--mode", "clean", "--json"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"text": "hello"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestImportAndExportCommandsAreUnavailable(t *testing.T) {
	for _, command := range []string{"import", "export"} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := Run([]string{command, "anything"}, &stdout, &stderr)

		if code != 2 {
			t.Fatalf("%s: expected exit code 2, got %d", command, code)
		}
		if !strings.Contains(stderr.String(), command+" is not available in P0") {
			t.Fatalf("%s: unexpected stderr: %s", command, stderr.String())
		}
	}
}

type fakeCLIService struct {
	contextText string
	diagnostics []core.SourceDiagnostic
	listResult  core.ListResult
	detail      core.SessionDetail
}

func (f fakeCLIService) Doctor() []core.SourceDiagnostic {
	return f.diagnostics
}

func (f fakeCLIService) List(core.ListOptions) core.ListResult {
	return f.listResult
}

func (f fakeCLIService) Show(string, core.ShowOptions) (core.SessionDetail, error) {
	return f.detail, nil
}

func (f fakeCLIService) Context(string, core.ContextOptions) (string, error) {
	return f.contextText, nil
}
