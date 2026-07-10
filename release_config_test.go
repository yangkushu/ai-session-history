package release_test

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseConfiguration(t *testing.T) {
	goreleaser := readFile(t, ".goreleaser.yaml")
	workflow := readFile(t, ".github/workflows/release.yaml")

	for _, want := range []string{
		"project_name: ai-history",
		"main: ./cmd/ai-history",
		"binary: ai-history",
		"CGO_ENABLED=0",
		"github.com/yangkushu/ai-session-history/internal/cli.version={{.Version}}",
		"github.com/yangkushu/ai-session-history/internal/cli.commit={{.Commit}}",
		"github.com/yangkushu/ai-session-history/internal/cli.buildDate={{.CommitDate}}",
		"name_template: checksums.txt",
	} {
		if !strings.Contains(goreleaser, want) {
			t.Fatalf(".goreleaser.yaml missing %q:\n%s", want, goreleaser)
		}
	}

	for _, want := range []string{
		"tags:",
		"- 'v*'",
		"workflow_dispatch:",
		"contents: write",
		"fetch-depth: 0",
		"actions/setup-go@v5",
		"goreleaser/goreleaser-action@v6",
		"args: release --clean",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("release workflow missing %q:\n%s", want, workflow)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
