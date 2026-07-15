package release_test

import (
	"strings"
	"testing"
)

func TestDocumentationAgentSkillInstallationContract(t *testing.T) {
	readmeRoles := []struct {
		name        string
		path        string
		heading     string
		nextHeading string
		want        []string
	}{
		{
			name:        "English",
			path:        "README.md",
			heading:     "## Who uses the Skill",
			nextHeading: "## Quick Start",
			want: []string{
				"You choose the Skill targets and authorize runtime access",
				"$ai-history",
				"Claude Code and Cursor",
				"Skill invocation displayed by the current host UI",
				"docs/installation.md",
			},
		},
		{
			name:        "Chinese",
			path:        "README.zh-CN.md",
			heading:     "## 谁使用 Skill",
			nextHeading: "## 快速开始",
			want: []string{
				"你负责选择 Skill targets，并授权运行时访问",
				"$ai-history",
				"Claude Code 和 Cursor",
				"当前 host UI 显示的 Skill invocation",
				"docs/installation.md",
			},
		},
	}

	for _, tt := range readmeRoles {
		t.Run(tt.name, func(t *testing.T) {
			section := readMarkdownSection(t, tt.path, tt.heading, tt.nextHeading)
			normalized := normalizeWhitespace(section)
			for _, want := range tt.want {
				if !strings.Contains(normalized, want) {
					t.Errorf("%s Skill role section missing %q", tt.path, want)
				}
			}
		})
	}

	docSections := []struct {
		name        string
		heading     string
		nextHeading string
		want        []string
	}{
		{
			name:        "bundle trust boundary",
			heading:     "## Binary and Skill install",
			nextHeading: "## Version, install directory, and PATH options",
			want: []string{
				"review the repository source and [`skills/ai-history/SKILL.md`]",
				"downloads and executes the linked third-party `skills` package",
				"Node.js and `npx` are used only for Skill installation",
				"Go CLI runtime has no Node.js dependency",
			},
		},
		{
			name:        "runtime authorization",
			heading:     "## Agent detection and explicit targets",
			nextHeading: "## Releases, source, and manual fallbacks",
			want: []string{
				"does not grant permission to execute the CLI, read history, or write exports",
				"does not change the host sandbox, allowlist, or managed policy",
				"Codex uses `$ai-history`",
				"Claude Code and Cursor, use the Skill invocation displayed by the current host UI",
			},
		},
		{
			name:        "manual fallback",
			heading:     "## Releases, source, and manual fallbacks",
			nextHeading: "## Remote-script review and checksum trust boundary",
			want: []string{
				"same canonical `skills/ai-history/` source of truth",
				"$HOME/.agents/skills/ai-history",
				"$HOME/.claude/skills/ai-history",
				"$HOME/.cursor/skills/ai-history",
			},
		},
	}

	for _, tt := range docSections {
		t.Run(tt.name, func(t *testing.T) {
			section := readMarkdownSection(t, "docs/installation.md", tt.heading, tt.nextHeading)
			normalized := normalizeWhitespace(section)
			for _, want := range tt.want {
				if !strings.Contains(normalized, want) {
					t.Errorf("docs/installation.md section %q missing %q", tt.heading, want)
				}
			}
		})
	}
}

func readMarkdownSection(t *testing.T, path, heading, nextHeading string) string {
	t.Helper()
	content := readFile(t, path)
	start := strings.Index(content, heading)
	if start < 0 {
		t.Fatalf("%s missing section heading %q", path, heading)
	}
	endOffset := strings.Index(content[start+len(heading):], nextHeading)
	if endOffset < 0 {
		t.Fatalf("%s section %q missing next heading %q", path, heading, nextHeading)
	}
	end := start + len(heading) + endOffset
	section := content[start:end]
	if strings.TrimSpace(section) == "" {
		t.Fatalf("%s section %q is empty", path, heading)
	}
	return section
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
