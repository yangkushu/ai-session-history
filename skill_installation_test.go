package release_test

import (
	"strings"
	"testing"
)

func TestReadmesDocumentAgentSkillInstallationContract(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		heading     string
		nextHeading string
		want        []string
	}{
		{
			name:        "English",
			path:        "README.md",
			heading:     "## Agent Skill",
			nextHeading: "## Quick Start",
			want: []string{
				"review the repository source and",
				"before installing",
				"downloads and executes the linked third-party `skills` package",
				"only for this skill installation",
				"no node.js runtime dependency",
				"same canonical `skills/ai-history/` directory",
				"source of truth",
				"does not grant runtime permissions for cli execution, history reads, or export writes",
				"does not change the sandbox, allowlist, or managed policy",
				"codex uses `$ai-history`",
				"slash or skill invocation shown by the current host ui",
			},
		},
		{
			name:        "Chinese",
			path:        "README.zh-CN.md",
			heading:     "## Agent Skill 安装",
			nextHeading: "## 快速开始",
			want: []string{
				"安装前请先审阅仓库源码和",
				"会下载并执行所链接的第三方 `skills` package",
				"只在安装 skill 时需要",
				"运行时不依赖 node.js",
				"同一份 canonical `skills/ai-history/` 目录",
				"source of truth",
				"不会授予 cli 执行、历史读取或导出写入等运行时权限",
				"不会修改 `sandbox`、`allowlist` 或 `managed policy`",
				"codex 使用 `$ai-history`",
				"host ui 显示的 slash 或 skill invocation",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := readMarkdownSection(t, tt.path, tt.heading, tt.nextHeading)
			lower := strings.ToLower(normalizeWhitespace(section))
			for _, want := range tt.want {
				if !strings.Contains(lower, strings.ToLower(want)) {
					t.Errorf("%s Agent Skill section missing %q", tt.path, want)
				}
			}

			command := "npx skills add yangkushu/ai-session-history --skill ai-history --global --agent codex --agent claude-code --agent cursor"
			if !strings.Contains(normalizeWhitespace(section), command) {
				t.Errorf("%s Agent Skill section missing normalized primary command", tt.path)
			}
			if !strings.Contains(section, command) {
				t.Errorf("%s Agent Skill section missing single-line command", tt.path)
			}

			for _, target := range []string{
				"$HOME/.agents/skills/ai-history",
				"$HOME/.claude/skills/ai-history",
				"$HOME/.cursor/skills/ai-history",
			} {
				if !strings.Contains(section, target) {
					t.Errorf("%s Agent Skill section missing manual target %q", tt.path, target)
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
	return strings.Join(strings.Fields(strings.ReplaceAll(value, "\\", "")), " ")
}
