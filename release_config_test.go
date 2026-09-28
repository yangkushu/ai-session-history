package release_test

import (
	"os"
	"path/filepath"
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
		"github.com/yangkushu/ai-session-history/internal/cli.version=v{{.Version}}",
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
		"go-version: '1.26'",
		"goreleaser/goreleaser-action@v6",
		"args: release --clean",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("release workflow missing %q:\n%s", want, workflow)
		}
	}
}

func TestWindowsInstallerUsesSelfContainedChecksum(t *testing.T) {
	script := readFile(t, "scripts/install.ps1")
	if strings.Contains(script, "Get-FileHash") {
		t.Fatal("Windows installer checksum must not depend on an auto-loaded PowerShell cmdlet")
	}
	for _, want := range []string{
		"[System.Security.Cryptography.SHA256]::Create()",
		"[System.IO.File]::OpenRead($ArchivePath)",
		"$Sha256.ComputeHash($Stream)",
		"checksum verification failed",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("Windows installer missing checksum step %q", want)
		}
	}
}

func TestReadFileNormalizesCRLFForTextContracts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("formats:\r\n          - zip\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, path); got != "formats:\n          - zip\n" {
		t.Fatalf("text contract must compare lines independent of checkout EOL: %q", got)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}
