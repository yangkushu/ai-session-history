package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yangkushu/ai-session-history/internal/core"
	"github.com/yangkushu/ai-session-history/internal/render"
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

func TestRunShowsTopLevelHelp(t *testing.T) {
	for _, args := range [][]string{
		{"help"},
		{"--help"},
		{"-h"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := RunWithService(args, &stdout, &stderr, fakeCLIService{})

		if code != 0 {
			t.Fatalf("%v: expected exit code 0, got %d stderr=%s", args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Usage: ai-history") {
			t.Fatalf("%v: expected usage in stdout, got %q", args, stdout.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("%v: expected no stderr, got %q", args, stderr.String())
		}
	}
}

func TestRunShowsSubcommandHelp(t *testing.T) {
	for _, args := range [][]string{
		{"help", "list"},
		{"list", "--help"},
		{"list", "-h"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := RunWithService(args, &stdout, &stderr, fakeCLIService{})

		if code != 0 {
			t.Fatalf("%v: expected exit code 0, got %d stderr=%s", args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Usage: ai-history list") {
			t.Fatalf("%v: expected list usage in stdout, got %q", args, stdout.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("%v: expected no stderr, got %q", args, stderr.String())
		}
	}
}

func TestRunHelpDoesNotLoadConfig(t *testing.T) {
	for _, args := range [][]string{
		{"help", "list", "--config", "/path/does/not/exist.yaml"},
		{"list", "--help", "--config", "/path/does/not/exist.yaml"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := Run(args, &stdout, &stderr)

		if code != 0 {
			t.Fatalf("%v: expected exit code 0, got %d stderr=%s", args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Usage: ai-history list") {
			t.Fatalf("%v: expected list usage in stdout, got %q", args, stdout.String())
		}
	}
}

func TestRunShowsVersion(t *testing.T) {
	for _, args := range [][]string{
		{"version"},
		{"--version"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := RunWithService(args, &stdout, &stderr, fakeCLIService{})

		if code != 0 {
			t.Fatalf("%v: expected exit code 0, got %d stderr=%s", args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "ai-history") || !strings.Contains(stdout.String(), "dev") {
			t.Fatalf("%v: expected development version output, got %q", args, stdout.String())
		}
	}
}

func TestRunShowsInjectedVersionMetadata(t *testing.T) {
	oldVersion := version
	oldCommit := commit
	oldBuildDate := buildDate
	t.Cleanup(func() {
		version = oldVersion
		commit = oldCommit
		buildDate = oldBuildDate
	})
	version = "v0.1.0"
	commit = "abc1234"
	buildDate = "2026-07-10T00:00:00Z"

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunWithService([]string{"version"}, &stdout, &stderr, fakeCLIService{})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	for _, want := range []string{"ai-history v0.1.0", "commit=abc1234", "buildDate=2026-07-10T00:00:00Z"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected %q in version output, got %q", want, stdout.String())
		}
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

func TestContextCommandWritesJSONHandoff(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var got core.ContextOptions
	service := fakeCLIService{
		handoff: contextHandoffFixture(),
		handoffHook: func(opts core.ContextOptions) {
			got = opts
		},
	}

	code := RunWithService([]string{"context", "codex:abc", "--target-cwd", "/new", "--max-chars", "123", "--json"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	var payload struct {
		SchemaVersion      string          `json:"schema_version"`
		Session            json.RawMessage `json:"session"`
		RecentConversation json.RawMessage `json:"recent_conversation"`
		ToolOutcomes       json.RawMessage `json:"tool_outcomes"`
		HandoffNotes       []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"handoff_notes"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if payload.SchemaVersion != render.HandoffSchemaVersion {
		t.Fatalf("unexpected schema version: %q", payload.SchemaVersion)
	}
	if got.TargetCWD != "/new" || got.MaxChars != 123 {
		t.Fatalf("unexpected context options: %+v", got)
	}
	if string(payload.RecentConversation) != "[]" || string(payload.ToolOutcomes) != "[]" {
		t.Fatalf("expected empty arrays, got recent=%s outcomes=%s", payload.RecentConversation, payload.ToolOutcomes)
	}
	if len(payload.HandoffNotes) != 1 || payload.HandoffNotes[0].Code == "" || payload.HandoffNotes[0].Message == "" {
		t.Fatalf("unexpected handoff notes: %+v", payload.HandoffNotes)
	}
}

func TestContextCommandSupportsJSONShortAlias(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service := fakeCLIService{handoff: contextHandoffFixture()}

	code := RunWithService([]string{"context", "codex:abc", "-j"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"schema_version": "context-handoff.v1"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestContextCommandKeepsMarkdownDefault(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service := fakeCLIService{
		contextText: "# AI Session Context\n\nbody",
		handoff:     contextHandoffFixture(),
	}

	code := RunWithService([]string{"context", "codex:abc", "--target-cwd", "/new"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if stdout.String() != "# AI Session Context\n\nbody" {
		t.Fatalf("unexpected Markdown output: %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "schema_version") {
		t.Fatalf("default context output must remain Markdown: %q", stdout.String())
	}
}

func TestAppServiceContextKeepsMarkdownWithinMaxChars(t *testing.T) {
	detail := core.SessionDetail{
		Summary: core.SessionSummary{ID: "codex:abc", Source: core.SourceCodex, NativeID: "abc", CWD: "/old"},
		Turns: []core.Turn{
			{Role: core.RoleUser, Text: "Goal", Kind: core.KindMessage},
			{Role: core.RoleAssistant, Text: "Answer", Kind: core.KindMessage},
			{Role: core.RoleTool, Text: "ok", Kind: core.KindToolResult},
		},
	}
	service := &appService{
		core:         core.NewService(map[core.Source]core.Reader{core.SourceCodex: contextDetailReader{detail: detail}}),
		detailLimit:  2000,
		contextLimit: 220,
	}

	text, err := service.Context("codex:abc", core.ContextOptions{MaxChars: 220})

	if err != nil {
		t.Fatalf("context: %v", err)
	}
	if len(text) > 220 {
		t.Fatalf("expected Markdown bounded to 220 chars, got %d:\n%s", len(text), text)
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

func TestDoctorCommandSupportsJSONShortAlias(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service := fakeCLIService{diagnostics: []core.SourceDiagnostic{
		{Source: core.SourceCodex, Status: "available"},
	}}

	code := RunWithService([]string{"doctor", "-j"}, &stdout, &stderr, service)

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
	if !strings.Contains(stdout.String(), `"source": "cursor"`) {
		t.Fatalf("expected cursor diagnostic in stdout: %s", stdout.String())
	}
}

func TestRunUsesCommandConfigFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"doctor", "--config", "/path/does/not/exist.yaml", "--json"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected config load failure, got %d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "invalid_config") {
		t.Fatalf("expected invalid_config error, got %s", stderr.String())
	}
}

func TestRunUsesConfigShortFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"doctor", "-c", "/path/does/not/exist.yaml", "--json"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected config load failure, got %d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "invalid_config") {
		t.Fatalf("expected invalid_config error, got %s", stderr.String())
	}
}

func TestRunUsesConfigForSourceEnablement(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(configPath, []byte(`
sources:
  codex:
    enabled: false
  claude:
    enabled: false
  cursor:
    enabled: false
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"doctor", "--json", "--config", configPath}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), `"status": "available"`) {
		t.Fatalf("expected no available sources with all disabled, got %s", stdout.String())
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

func TestListCommandSupportsShortAliases(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var got core.ListOptions
	service := fakeCLIService{
		listHook: func(opts core.ListOptions) {
			got = opts
		},
		listResult: core.ListResult{Sessions: []core.SessionSummary{
			{ID: "codex:abc", Source: core.SourceCodex, NativeID: "abc", Title: "Title", CWD: "/work/a"},
		}},
	}

	code := RunWithService([]string{"list", "-s", "codex", "-l", "5", "-j"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if got.Source != core.SourceCodex || got.Limit != 5 {
		t.Fatalf("unexpected list options: %+v", got)
	}
	if !strings.Contains(stdout.String(), `"id": "codex:abc"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestListCommandHereFiltersCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	}()
	var got core.ListOptions
	service := fakeCLIService{
		listHook: func(opts core.ListOptions) {
			got = opts
		},
		listResult: core.ListResult{Sessions: []core.SessionSummary{}},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunWithService([]string{"list", "--here", "--json"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if got.Under != workingDir {
		t.Fatalf("expected --here to use cwd %q, got %+v", workingDir, got)
	}
}

func TestListCommandHereConflictsWithDirectoryFilters(t *testing.T) {
	for _, args := range [][]string{
		{"list", "--here", "--under", "/tmp"},
		{"list", "--here", "--cwd", "/tmp"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := RunWithService(args, &stdout, &stderr, fakeCLIService{})

		if code != 2 {
			t.Fatalf("%v: expected usage error, got %d", args, code)
		}
		if !strings.Contains(stderr.String(), "--here cannot be combined") {
			t.Fatalf("%v: unexpected stderr: %s", args, stderr.String())
		}
	}
}

func TestListCommandEmptyJSONUsesArray(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service := fakeCLIService{listResult: core.ListResult{Sessions: []core.SessionSummary{}}}

	code := RunWithService([]string{"list", "--json"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	var payload struct {
		Sessions json.RawMessage `json:"sessions"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if string(payload.Sessions) != "[]" {
		t.Fatalf("expected sessions [], got %s in %s", payload.Sessions, stdout.String())
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

func TestShowCommandSupportsShortAliases(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service := fakeCLIService{detail: core.SessionDetail{
		Summary: core.SessionSummary{ID: "codex:abc", Source: core.SourceCodex, NativeID: "abc", Title: "Title"},
		Turns:   []core.Turn{{Role: core.RoleUser, Text: "hello", Kind: core.KindMessage}},
	}}

	code := RunWithService([]string{"show", "codex:abc", "-m", "summary", "-n", "2000", "-j"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"text": "hello"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestContextCommandSupportsShortAliases(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service := fakeCLIService{contextText: "# AI Session Context\n\nbody"}

	code := RunWithService([]string{"context", "codex:abc", "-t", "/new", "-n", "4000"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("# AI Session Context")) {
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
	handoff     render.HandoffContext
	handoffHook func(core.ContextOptions)
	diagnostics []core.SourceDiagnostic
	listResult  core.ListResult
	detail      core.SessionDetail
	listHook    func(core.ListOptions)
}

type contextDetailReader struct {
	detail core.SessionDetail
}

func (r contextDetailReader) ListSessions() ([]core.SessionSummary, error) {
	return []core.SessionSummary{r.detail.Summary}, nil
}

func (r contextDetailReader) GetSession(nativeID string) (core.SessionDetail, error) {
	if nativeID != r.detail.Summary.NativeID {
		return core.SessionDetail{}, core.NewError(core.ErrSessionNotFound, nativeID)
	}
	return r.detail, nil
}

func (r contextDetailReader) Doctor() core.SourceDiagnostic {
	return core.SourceDiagnostic{Source: r.detail.Summary.Source, Status: "available"}
}

func (f fakeCLIService) Doctor() []core.SourceDiagnostic {
	return f.diagnostics
}

func (f fakeCLIService) List(opts core.ListOptions) core.ListResult {
	if f.listHook != nil {
		f.listHook(opts)
	}
	return f.listResult
}

func (f fakeCLIService) Show(string, core.ShowOptions) (core.SessionDetail, error) {
	return f.detail, nil
}

func (f fakeCLIService) Context(string, core.ContextOptions) (string, error) {
	return f.contextText, nil
}

func (f fakeCLIService) ContextHandoff(_ string, opts core.ContextOptions) (render.HandoffContext, error) {
	if f.handoffHook != nil {
		f.handoffHook(opts)
	}
	return f.handoff, nil
}

func contextHandoffFixture() render.HandoffContext {
	return render.HandoffContext{
		SchemaVersion: render.HandoffSchemaVersion,
		Session: render.HandoffSession{
			ID:        "codex:abc",
			Source:    core.SourceCodex,
			NativeID:  "abc",
			TargetCWD: "/new",
		},
		InitialGoal:        render.HandoffInitialGoal{Available: true, Text: "Initial goal"},
		RecentConversation: []core.Turn{},
		ToolOutcomes:       []core.Turn{},
		HandoffNotes: []render.HandoffNote{{
			Code:    "no_omissions",
			Message: "No skipped, omitted, or truncated content.",
		}},
		HandoffInstruction: "Continue from this prior AI coding session.",
	}
}
