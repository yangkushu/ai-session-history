package release_test

import (
	"strings"
	"testing"
)

func TestAIHistorySkillContentContract(t *testing.T) {
	paths := []string{
		"skills/ai-history/SKILL.md",
		"skills/ai-history/references/codex-permissions.md",
		"skills/ai-history/references/claude-code-permissions.md",
		"skills/ai-history/references/cursor-permissions.md",
	}

	contents := make(map[string]string, len(paths))
	for _, path := range paths {
		contents[path] = readFile(t, path)
	}

	skill := contents[paths[0]]
	frontmatter, body := splitFrontmatter(t, skill)
	lines := nonEmptyLines(frontmatter)
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "name:") || !strings.HasPrefix(lines[1], "description:") {
		t.Fatalf("SKILL.md frontmatter must contain only name and description, got:\n%s", frontmatter)
	}
	if strings.TrimSpace(strings.TrimPrefix(lines[0], "name:")) != "ai-history" {
		t.Fatalf("unexpected skill name: %s", lines[0])
	}
	description := strings.TrimSpace(strings.TrimPrefix(lines[1], "description:"))
	descriptionLower := strings.ToLower(description)
	for _, want := range []string{"use when", "ai-history", "session", "history"} {
		if !strings.Contains(descriptionLower, want) {
			t.Errorf("description missing trigger %q: %s", want, description)
		}
	}

	bodyLower := strings.ToLower(body)
	for _, want := range []string{
		"ai-history version",
		"ai-history doctor --json",
		"project-first",
		"--here --json",
		"list",
		"search",
		"show",
		"context",
		"export",
		"permission_denied",
		"path",
		"source",
		"available",
		"unavailable",
		"never call or invent `import`",
		"--mode raw",
		"clean",
		"explicit",
		"--force",
		"current user",
		"references/codex-permissions.md",
		"references/claude-code-permissions.md",
		"references/cursor-permissions.md",
	} {
		if !strings.Contains(bodyLower, want) {
			t.Errorf("SKILL.md body missing %q", want)
		}
	}

	all := strings.ToLower(strings.Join([]string{
		contents[paths[0]],
		contents[paths[1]],
		contents[paths[2]],
		contents[paths[3]],
	}, "\n"))
	for _, forbidden := range []string{
		"sudo",
		"chmod 777",
		"--dangerously-bypass-approvals-and-sandbox",
		"allow all shell",
		"full filesystem access",
	} {
		if strings.Contains(all, forbidden) {
			t.Errorf("skill content contains unsafe guidance %q", forbidden)
		}
	}
}

func splitFrontmatter(t *testing.T, content string) (string, string) {
	t.Helper()
	parts := strings.SplitN(content, "---", 3)
	if len(parts) != 3 || strings.TrimSpace(parts[0]) != "" {
		t.Fatalf("SKILL.md must begin with YAML frontmatter")
	}
	return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])
}

func nonEmptyLines(content string) []string {
	var lines []string
	for _, line := range strings.Split(content, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
