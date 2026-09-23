package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangkushu/ai-session-history/internal/core"
	"github.com/yangkushu/ai-session-history/internal/render"
)

func TestPiCLICommandsUseConfiguredSourceAndSharedHandoffExport(t *testing.T) {
	piRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "pi", "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configText := "sources:\n" +
		"  codex: {enabled: false, use_default_paths: false}\n" +
		"  claude: {enabled: false, use_default_paths: false}\n" +
		"  cursor: {enabled: false, use_default_paths: false}\n" +
		"  pi:\n    enabled: true\n    use_default_paths: false\n    paths:\n      - " + piRoot + "\n"
	if err := os.WriteFile(configPath, []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("create app service: %v", err)
	}

	diagnostics := service.Doctor()
	var piDiagnostic *core.SourceDiagnostic
	for index := range diagnostics {
		if diagnostics[index].Source == core.SourcePi {
			piDiagnostic = &diagnostics[index]
		}
	}
	if piDiagnostic == nil || piDiagnostic.Status != "available" {
		t.Fatalf("Pi source should be available from configured fixture: %+v", diagnostics)
	}

	var stdout, stderr bytes.Buffer
	if code := RunWithService([]string{"list", "--source", "pi", "--json"}, &stdout, &stderr, service); code != 0 {
		t.Fatalf("list failed (%d): %s", code, stderr.String())
	}
	var listed core.ListResult
	if err := json.Unmarshal(stdout.Bytes(), &listed); err != nil {
		t.Fatalf("decode list output: %v", err)
	}
	if len(listed.Sessions) != 2 || listed.Sessions[0].Source != core.SourcePi {
		t.Fatalf("unexpected Pi list: %+v", listed.Sessions)
	}

	stdout.Reset()
	stderr.Reset()
	if code := RunWithService([]string{"search", "active branch", "--source", "pi", "--json"}, &stdout, &stderr, service); code != 0 {
		t.Fatalf("search failed (%d): %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "pi-fixture-v3") || strings.Contains(stdout.String(), "abandoned sibling") {
		t.Fatalf("search should find active Pi branch only: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := RunWithService([]string{"show", "pi:pi-fixture-v3", "--mode", "raw", "--json"}, &stdout, &stderr, service); code != 0 {
		t.Fatalf("show failed (%d): %s", code, stderr.String())
	}
	for _, private := range []string{"PRIVATE THINKING", "PRIVATE TOOL ARGUMENT", "PRIVATE BASE64", "PRIVATE BASH COMMAND"} {
		if strings.Contains(stdout.String(), private) {
			t.Errorf("show leaked %q: %s", private, stdout.String())
		}
	}

	stdout.Reset()
	stderr.Reset()
	if code := RunWithService([]string{"context", "pi:pi-fixture-v3", "--json"}, &stdout, &stderr, service); code != 0 {
		t.Fatalf("context JSON failed (%d): %s", code, stderr.String())
	}
	var handoff render.HandoffContext
	if err := json.Unmarshal(stdout.Bytes(), &handoff); err != nil {
		t.Fatalf("decode handoff: %v", err)
	}
	if handoff.SchemaVersion != render.HandoffSchemaVersion || len(handoff.PersistedSummaries) != 2 || handoff.PersistedSummaries[0].Kind != core.SummaryCompaction || handoff.PersistedSummaries[1].Kind != core.SummaryBranchSummary {
		t.Fatalf("unexpected Pi handoff: %+v", handoff)
	}

	stdout.Reset()
	stderr.Reset()
	markdownPath := filepath.Join(t.TempDir(), "handoff.md")
	if code := RunWithService([]string{"context", "pi:pi-fixture-v3", "--target-cwd", "/example/target"}, &stdout, &stderr, service); code != 0 {
		t.Fatalf("context Markdown failed (%d): %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "## Persisted Pi Summaries") || !strings.Contains(stdout.String(), "Persisted compaction") || !strings.Contains(stdout.String(), "Persisted branch summary") {
		t.Fatalf("Markdown context omitted persisted Pi summary: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := RunWithService([]string{"export", "pi:pi-fixture-v3", "--output", markdownPath, "--format", "markdown"}, &stdout, &stderr, service); code != 0 {
		t.Fatalf("export failed (%d): %s", code, stderr.String())
	}
	payload, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), "Concise tool result") || !strings.Contains(string(payload), "Terminal reports all checks passed.") {
		t.Fatalf("raw session export should preserve normalized visible tool results: %s", payload)
	}
	for _, private := range []string{"PRIVATE THINKING", "PRIVATE TOOL ARGUMENT", "PRIVATE BASE64", "PRIVATE BASH COMMAND"} {
		if strings.Contains(string(payload), private) {
			t.Errorf("raw export leaked %q", private)
		}
	}
	info, err := os.Stat(markdownPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("export must use private permissions, got %o", info.Mode().Perm())
	}

	cleanPath := filepath.Join(filepath.Dir(markdownPath), "clean.md")
	stdout.Reset()
	stderr.Reset()
	if code := RunWithService([]string{"export", "pi:pi-fixture-v3", "--output", cleanPath, "--format", "markdown", "--mode", "clean"}, &stdout, &stderr, service); code != 0 {
		t.Fatalf("clean export failed (%d): %s", code, stderr.String())
	}
	cleanPayload, err := os.ReadFile(cleanPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cleanPayload), "Terminal reports all checks passed.") || !strings.Contains(string(cleanPayload), "[omitted: tool_output]") {
		t.Fatalf("clean export must apply existing tool-output omission rules: %s", cleanPayload)
	}
}
