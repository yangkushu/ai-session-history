package release_test

import (
	"strings"
	"testing"
)

func TestCIConfiguration(t *testing.T) {
	workflow := readFile(t, ".github/workflows/ci.yaml")
	changelog := readFile(t, "CHANGELOG.md")

	for _, want := range []string{
		"name: CI",
		"push:",
		"pull_request:",
		"workflow_dispatch:",
		"contents: read",
		"actions/setup-go@v5",
		"go-version: '1.22'",
		"go mod tidy",
		"git diff --exit-code -- go.mod go.sum",
		"go test ./...",
		"go vet ./...",
		"goreleaser/goreleaser-action@v6",
		"args: check",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("CI workflow missing %q:\n%s", want, workflow)
		}
	}

	for _, want := range []string{
		"# Changelog",
		"## Unreleased",
		"## 0.1.0 - 2026-07-10",
		"Initial `ai-history` CLI release.",
		"Added release automation with GoReleaser",
	} {
		if !strings.Contains(changelog, want) {
			t.Fatalf("CHANGELOG.md missing %q:\n%s", want, changelog)
		}
	}
}

func TestCIExercisesNativeInstallers(t *testing.T) {
	workflow := readFile(t, ".github/workflows/ci.yaml")

	for _, want := range []string{
		"installer-test:",
		"ubuntu-latest",
		"macos-latest",
		"windows-latest",
		"go test ./... -run TestUnixInstaller",
		"go test ./... -run TestPowerShellInstaller",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("CI workflow missing %q:\n%s", want, workflow)
		}
	}
}
