package release_test

import (
	"os"
	"strings"
	"testing"
)

func TestProjectAgentModelRouting(t *testing.T) {
	cases := []struct {
		path  string
		wants []string
	}{
		{
			path: ".codex/agents/worker.toml",
			wants: []string{
				`name = "worker"`,
				`model = "gpt-5.6-luna"`,
				`model_reasoning_effort = "medium"`,
				`developer_instructions = """`,
			},
		},
		{
			path: ".codex/agents/worker-terra.toml",
			wants: []string{
				`name = "worker-terra"`,
				`model = "gpt-5.6-terra"`,
				`model_reasoning_effort = "medium"`,
				`developer_instructions = """`,
			},
		},
		{
			path: ".codex/agents/reviewer.toml",
			wants: []string{
				`name = "reviewer"`,
				`model = "gpt-5.6-sol"`,
				`model_reasoning_effort = "high"`,
				`developer_instructions = """`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			payload, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			text := string(payload)
			for _, want := range tc.wants {
				if !strings.Contains(text, want) {
					t.Errorf("%s missing %q", tc.path, want)
				}
			}
			for _, inherited := range []string{"sandbox_mode", "approval_policy", "mcp_servers"} {
				if strings.Contains(text, inherited) {
					t.Errorf("%s must inherit %s from the parent", tc.path, inherited)
				}
			}
		})
	}
}
