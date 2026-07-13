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
	fields := frontmatterFields(t, frontmatter)
	if len(fields) != 2 {
		t.Fatalf("SKILL.md frontmatter must contain only name and description, got:\n%s", frontmatter)
	}
	for _, key := range []string{"name", "description"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("SKILL.md frontmatter missing %q: %s", key, frontmatter)
		}
	}
	if fields["name"] != "ai-history" {
		t.Fatalf("unexpected skill name: %s", fields["name"])
	}
	description := fields["description"]
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
		"ai-history list --here --json",
		"ai-history search <query> --here --json",
		"ai-history show <id> --json",
		"ai-history context <id> --json",
		"ai-history export <id> --output <path>",
		"permission_denied",
		"path",
		"source",
		"available",
		"unavailable",
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
	assertProhibitedTerm(t, "SKILL.md", bodyLower, "`import`", "never", "do not", "not a command")
	for _, want := range []string{
		"execution of the specific `ai-history` command",
		"read access to the reported history path",
		"write access to the user-selected export destination",
		"managed policy",
		"installation alone grants no runtime permission",
	} {
		if !strings.Contains(bodyLower, want) {
			t.Errorf("SKILL.md permission model missing %q", want)
		}
	}

	codex := strings.ToLower(contents[paths[1]])
	assertContainsAll(t, "Codex reference", codex,
		"/permissions",
		"filesystem permission profile",
		"install",
		"runtime permission",
		"managed policy",
	)
	assertProhibitedTerm(t, "Codex reference", codex, "danger-full-access", "never", "do not")

	claude := strings.ToLower(contents[paths[2]])
	assertContainsAll(t, "Claude Code reference", claude,
		"/permissions",
		"bash(ai-history",
		"install",
		"runtime permission",
		"managed policy",
	)
	if !strings.Contains(claude, "--add-dir") && !strings.Contains(claude, "additionaldirectories") {
		t.Errorf("Claude Code reference missing --add-dir/additionalDirectories")
	}
	assertProhibitedTerm(t, "Claude Code reference", claude, "whole bash", "never", "do not", "not permit")

	cursor := strings.ToLower(contents[paths[3]])
	assertContainsAll(t, "Cursor reference", cursor,
		"approvals & execution",
		"sandbox",
		"shell(ai-history)",
		"read(<history-path>)",
		"install",
		"runtime permission",
		"managed policy",
	)
	assertProhibitedTerm(t, "Cursor reference", cursor, "broad allowlist", "avoid", "never", "do not")

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

func frontmatterFields(t *testing.T, content string) map[string]string {
	t.Helper()
	fields := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
				t.Fatalf("invalid frontmatter line %q", line)
			}
			key := strings.TrimSpace(parts[0])
			if _, duplicate := fields[key]; duplicate {
				t.Fatalf("duplicate frontmatter field %q", key)
			}
			fields[key] = strings.TrimSpace(parts[1])
		}
	}
	return fields
}

func assertContainsAll(t *testing.T, name, content string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(content, strings.ToLower(want)) {
			t.Errorf("%s missing %q", name, want)
		}
	}
}

func assertProhibitedTerm(t *testing.T, name, content, term string, prohibitionMarkers ...string) {
	t.Helper()
	term = strings.ToLower(term)
	found := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.ToLower(line)
		if !strings.Contains(line, term) {
			continue
		}
		found = true
		prohibited := false
		for _, marker := range prohibitionMarkers {
			if strings.Contains(line, strings.ToLower(marker)) {
				prohibited = true
				break
			}
		}
		if !prohibited {
			t.Fatalf("%s mentions %q without prohibition context: %s", name, term, line)
		}
	}
	if !found {
		t.Fatalf("%s missing prohibited term %q", name, term)
	}
}
