package cli

import (
	"bytes"
	"encoding/json"
	"errors"
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

func TestSearchCommandWritesJSONAndSupportsShortAliases(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var got core.SearchOptions
	service := fakeCLIService{
		searchHook: func(opts core.SearchOptions) {
			got = opts
		},
		searchResult: core.SearchResult{Hits: []core.SearchHit{{
			Session: core.SessionSummary{ID: "codex:abc", Source: core.SourceCodex, Title: "Needle session", CWD: "/work"},
			Score:   130,
			Matches: []core.SearchMatch{{Category: core.SearchMatchTitle, Snippet: "Needle session"}},
			Snippet: "Needle session",
		}}, Unavailable: map[core.Source]string{core.SourceClaude: "not readable"}, Diagnostics: map[core.Source]core.SourceDiagnostic{
			core.SourceClaude: {Source: core.SourceClaude, Status: "unavailable", Message: "not readable"},
		}},
	}

	code := RunWithService([]string{"search", "needle", "-s", "codex", "-l", "5", "-j"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if got.Query != "needle" || got.Source != core.SourceCodex || got.Limit != 5 {
		t.Fatalf("unexpected search options: %+v", got)
	}
	if !strings.Contains(stdout.String(), `"hits": [`) || !strings.Contains(stdout.String(), `"id": "codex:abc"`) || !strings.Contains(stdout.String(), `"unavailable_sources"`) || !strings.Contains(stdout.String(), `"diagnostics"`) {
		t.Fatalf("unexpected JSON output: %s", stdout.String())
	}
}

func TestSearchCommandValidatesQueryAndLocationFilters(t *testing.T) {
	for _, args := range [][]string{
		{"search"},
		{"search", "   "},
		{"search", "needle", "--here", "--under", "/tmp"},
		{"search", "needle", "--source", "invalid"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := RunWithService(args, &stdout, &stderr, fakeCLIService{})

		if code != 2 {
			t.Fatalf("%v: expected usage error, got %d", args, code)
		}
	}
}

func TestSearchCommandWritesTextAndEmptyJSONArray(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service := fakeCLIService{searchResult: core.SearchResult{Hits: []core.SearchHit{{
		Session: core.SessionSummary{ID: "codex:abc", Source: core.SourceCodex, Title: "Needle session"},
		Score:   100,
		Snippet: "needle context",
	}}}}

	code := RunWithService([]string{"search", "needle"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	for _, want := range []string{"codex:abc", "Needle session", "needle context"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected %q in text output: %s", want, stdout.String())
		}
	}

	stdout.Reset()
	stderr.Reset()
	service.searchResult = core.SearchResult{Hits: []core.SearchHit{}}
	code = RunWithService([]string{"search", "missing", "--json"}, &stdout, &stderr, service)
	if code != 0 {
		t.Fatalf("expected empty result success, got %d stderr=%s", code, stderr.String())
	}
	var payload struct {
		Hits json.RawMessage `json:"hits"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if string(payload.Hits) != "[]" {
		t.Fatalf("expected empty hits array, got %s", stdout.String())
	}
}

func TestSearchCommandUsesDefaultLimitAndHere(t *testing.T) {
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
	var got core.SearchOptions
	service := fakeCLIService{
		searchHook: func(opts core.SearchOptions) {
			got = opts
		},
		searchResult: core.SearchResult{Hits: []core.SearchHit{}},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunWithService([]string{"search", "needle", "--here"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if got.Limit != 20 || got.Under != workingDir {
		t.Fatalf("unexpected default search options: %+v", got)
	}
}

func TestSearchHelpIsDiscoverable(t *testing.T) {
	for _, args := range [][]string{{"help", "search"}, {"search", "--help"}, {"search", "-h"}} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := RunWithService(args, &stdout, &stderr, fakeCLIService{})

		if code != 0 {
			t.Fatalf("%v: expected success, got %d stderr=%s", args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Usage: ai-history search") {
			t.Fatalf("%v: expected search usage, got %s", args, stdout.String())
		}
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
	created := time.Date(2026, 7, 6, 23, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC)
	service := fakeCLIService{listResult: core.ListResult{Sessions: []core.SessionSummary{
		{ID: "codex:abc", Source: core.SourceCodex, NativeID: "abc", Title: "Title", CWD: "/work/a", CreatedAt: &created, UpdatedAt: &updated},
	}}}

	code := RunWithService([]string{"list", "--source", "codex", "--under", "/work", "--limit", "5", "--json"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	var payload struct {
		Sessions []core.SessionSummary `json:"sessions"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if len(payload.Sessions) != 1 || payload.Sessions[0].ID != "codex:abc" {
		t.Fatalf("unexpected sessions: %+v", payload.Sessions)
	}
	if payload.Sessions[0].CreatedAt == nil || !payload.Sessions[0].CreatedAt.Equal(created) {
		t.Fatalf("created_at changed: %+v", payload.Sessions[0].CreatedAt)
	}
	if payload.Sessions[0].UpdatedAt == nil || !payload.Sessions[0].UpdatedAt.Equal(updated) {
		t.Fatalf("updated_at changed: %+v", payload.Sessions[0].UpdatedAt)
	}
}

func TestListCommandWritesReadableText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	created := time.Date(2026, 7, 16, 19, 28, 0, 0, time.UTC)
	updated := time.Date(2026, 7, 17, 10, 4, 0, 0, time.UTC)
	service := fakeCLIService{listResult: core.ListResult{Sessions: []core.SessionSummary{
		{
			ID:        "codex:complete-session-id",
			Source:    core.SourceCodex,
			Title:     "Readable\n title",
			CWD:       "/work/complete/path",
			CreatedAt: &created,
			UpdatedAt: &updated,
		},
	}}}

	code := RunWithService([]string{"list"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("list output has %d lines, want 4: %q", len(lines), stdout.String())
	}
	if lines[0] != "codex  Readable title" {
		t.Fatalf("title line = %q", lines[0])
	}
	updatedPrefix := "       " + updated.In(time.Local).Format("2006-01-02 15:04") + " ("
	createdFragment := ")  " + created.In(time.Local).Format("2006-01-02 15:04") + " ("
	if !strings.HasPrefix(lines[1], updatedPrefix) || !strings.Contains(lines[1], createdFragment) || !strings.HasSuffix(lines[1], ")") {
		t.Fatalf("time line = %q", lines[1])
	}
	if lines[2] != "       /work/complete/path" {
		t.Fatalf("cwd line = %q", lines[2])
	}
	if lines[3] != "       codex:complete-session-id" {
		t.Fatalf("id line = %q", lines[3])
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

func TestExportCommandWritesDefaultJSONWithPrivatePermissions(t *testing.T) {
	output := filepath.Join(t.TempDir(), "session.json")
	service := fakeCLIService{export: exportFixture()}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunWithService([]string{"export", "codex:abc", "--output", output}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	payload, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if !bytes.Contains(payload, []byte("\"schema_version\": \"session-export.v1\"")) || !bytes.Contains(payload, []byte("\"content_mode\": \"raw\"")) {
		t.Fatalf("expected default JSON export, got %s", payload)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("stat export: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected 0600 export permissions, got %#o", got)
	}
}

func TestExportCommandSupportsMarkdownAndContentMode(t *testing.T) {
	output := filepath.Join(t.TempDir(), "session.md")
	var gotSessionID string
	var gotMode core.ContentMode
	service := fakeCLIService{
		export: exportFixture(),
		exportHook: func(sessionID string, mode core.ContentMode) {
			gotSessionID = sessionID
			gotMode = mode
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunWithService([]string{"export", "codex:abc", "-o", output, "-f", "markdown", "-m", "summary"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	if gotSessionID != "codex:abc" || gotMode != core.ModeSummary {
		t.Fatalf("unexpected export request: id=%q mode=%q", gotSessionID, gotMode)
	}
	payload, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read markdown export: %v", err)
	}
	if !bytes.Contains(payload, []byte("# AI Session Export")) || !bytes.Contains(payload, []byte("Content mode: summary")) {
		t.Fatalf("expected Markdown export, got %s", payload)
	}
}

func TestExportCommandValidatesOutputFormatAndMode(t *testing.T) {
	output := filepath.Join(t.TempDir(), "session.json")
	for _, args := range [][]string{
		{"export", "codex:abc"},
		{"export", "codex:abc", "--output", output, "--format", "yaml"},
		{"export", "codex:abc", "--output", output, "--mode", "invalid"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := RunWithService(args, &stdout, &stderr, fakeCLIService{export: exportFixture()})

		if code != 2 {
			t.Fatalf("%v: expected usage error, got %d stderr=%s", args, code, stderr.String())
		}
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatalf("%v: invalid export created output: %v", args, err)
		}
	}
}

func TestExportCommandProtectsExistingFileUnlessForced(t *testing.T) {
	output := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(output, []byte("original"), 0o600); err != nil {
		t.Fatalf("create output: %v", err)
	}
	service := fakeCLIService{export: exportFixture()}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunWithService([]string{"export", "codex:abc", "--output", output}, &stdout, &stderr, service)

	if code != 2 {
		t.Fatalf("expected usage error, got %d stderr=%s", code, stderr.String())
	}
	payload, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	if string(payload) != "original" {
		t.Fatalf("existing export changed without --force: %q", payload)
	}
}

func TestExportCommandForceAtomicallyReplacesExistingFile(t *testing.T) {
	output := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(output, []byte("original"), 0o600); err != nil {
		t.Fatalf("create output: %v", err)
	}
	service := fakeCLIService{export: exportFixture()}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunWithService([]string{"export", "codex:abc", "--output", output, "--force"}, &stdout, &stderr, service)

	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	payload, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read replacement: %v", err)
	}
	if bytes.Equal(payload, []byte("original")) || !bytes.Contains(payload, []byte("session-export.v1")) {
		t.Fatalf("expected replacement export, got %s", payload)
	}
}

func TestExportCommandWriteFailureLeavesNoPartialDestination(t *testing.T) {
	output := filepath.Join(t.TempDir(), "missing", "session.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunWithService([]string{"export", "codex:abc", "--output", output}, &stdout, &stderr, fakeCLIService{export: exportFixture()})

	if code != 1 {
		t.Fatalf("expected runtime error, got %d stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("failed export left destination: %v", err)
	}
}

func TestExportCommandRenameFailureCleansTemporaryFile(t *testing.T) {
	parent := t.TempDir()
	output := filepath.Join(parent, "existing-directory")
	if err := os.Mkdir(output, 0o700); err != nil {
		t.Fatalf("create destination directory: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunWithService([]string{"export", "codex:abc", "--output", output, "--force"}, &stdout, &stderr, fakeCLIService{export: exportFixture()})

	if code != 1 {
		t.Fatalf("expected runtime error, got %d stderr=%s", code, stderr.String())
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatalf("list output directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "existing-directory" || !entries[0].IsDir() {
		t.Fatalf("failed export left temporary output: %+v", entries)
	}
}

func TestWritePrivateFileAtomicDoesNotReplaceConcurrentDestinationWithoutForce(t *testing.T) {
	output := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(output, []byte("concurrent"), 0o600); err != nil {
		t.Fatalf("create output: %v", err)
	}

	err := writePrivateFileAtomic(output, []byte("replacement"), false)

	if !errors.Is(err, errExportDestinationExists) {
		t.Fatalf("expected existing destination error, got %v", err)
	}
	payload, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatalf("read original: %v", readErr)
	}
	if string(payload) != "concurrent" {
		t.Fatalf("concurrent destination was replaced: %q", payload)
	}
}

func TestImportCommandRemainsUnavailable(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"import", "anything"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "import is not available in P0") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

type fakeCLIService struct {
	contextText  string
	handoff      render.HandoffContext
	handoffHook  func(core.ContextOptions)
	diagnostics  []core.SourceDiagnostic
	listResult   core.ListResult
	searchResult core.SearchResult
	detail       core.SessionDetail
	export       render.SessionExport
	listHook     func(core.ListOptions)
	searchHook   func(core.SearchOptions)
	exportHook   func(string, core.ContentMode)
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

func (f fakeCLIService) Search(opts core.SearchOptions) core.SearchResult {
	if f.searchHook != nil {
		f.searchHook(opts)
	}
	return f.searchResult
}

func (f fakeCLIService) Show(string, core.ShowOptions) (core.SessionDetail, error) {
	return f.detail, nil
}

func (f fakeCLIService) Export(sessionID string, mode core.ContentMode) (render.SessionExport, error) {
	if f.exportHook != nil {
		f.exportHook(sessionID, mode)
	}
	export := f.export
	export.ContentMode = mode
	return export, nil
}

func exportFixture() render.SessionExport {
	return render.SessionExport{
		SchemaVersion: render.SessionExportSchemaVersion,
		ExportedAt:    time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC),
		ContentMode:   core.ModeRaw,
		Session: core.SessionDetail{
			Summary: core.SessionSummary{ID: "codex:abc", Source: core.SourceCodex, NativeID: "abc", Title: "Export title"},
			Turns:   []core.Turn{{Role: core.RoleUser, Kind: core.KindMessage, Text: "Export this session"}},
		},
	}
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
