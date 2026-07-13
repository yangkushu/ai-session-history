package release_test

import (
	"strings"
	"testing"
)

func TestReadmesDocumentAgentSkillInstallationContract(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "English",
			path: "README.md",
			want: []string{
				"npx skills add yangkushu/ai-session-history",
				"--skill ai-history --global",
				"--agent codex --agent claude-code --agent cursor",
				"skills/ai-history/SKILL.md",
				"skills/ai-history/",
				"$HOME/.agents/skills/ai-history",
				"$HOME/.claude/skills/ai-history",
				"$HOME/.cursor/skills/ai-history",
				"$ai-history",
				"ai-history version",
				"doctor --json",
				"Node.js",
				"runtime permissions",
				"sandbox",
				"allowlist",
				"managed policy",
			},
		},
		{
			name: "Chinese",
			path: "README.zh-CN.md",
			want: []string{
				"npx skills add yangkushu/ai-session-history",
				"--skill ai-history --global",
				"--agent codex --agent claude-code --agent cursor",
				"skills/ai-history/SKILL.md",
				"skills/ai-history/",
				"$HOME/.agents/skills/ai-history",
				"$HOME/.claude/skills/ai-history",
				"$HOME/.cursor/skills/ai-history",
				"$ai-history",
				"ai-history version",
				"doctor --json",
				"Node.js",
				"运行时权限",
				"sandbox",
				"allowlist",
				"managed policy",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := readFile(t, tt.path)
			for _, want := range tt.want {
				if !strings.Contains(content, want) {
					t.Errorf("%s missing %q", tt.path, want)
				}
			}
		})
	}
}
