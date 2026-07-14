# Sub-agent Model Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add project-scoped custom agents that route routine implementation to Luna, complex implementation to Terra, and independent review to Sol.

**Architecture:** Three standalone TOML files under `.codex/agents/` define the roles and inherit the parent session's sandbox and tool configuration. A root Go contract test locks required fields and model routing, while Python standard-library `tomllib` provides syntax validation without adding a project dependency.

**Tech Stack:** Codex custom agent TOML, Go 1.22 tests, Python 3 standard-library `tomllib` for validation.

---

## File map

- Create `.codex/agents/worker.toml`: override the built-in implementation worker with Luna.
- Create `.codex/agents/worker-terra.toml`: opt-in complex implementation worker using Terra.
- Create `.codex/agents/reviewer.toml`: independent read-only reviewer using Sol.
- Create `agent_config_test.go`: prevent required agent names, models, reasoning, and inherited-sandbox behavior from drifting.

### Task 1: Add a failing agent configuration contract

**Files:**
- Create: `agent_config_test.go`

- [ ] **Step 1: Write the failing Go test**

Create `agent_config_test.go` in `package release_test`:

```go
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
```

- [ ] **Step 2: Run the test and verify the RED state**

Run:

```bash
env GOCACHE=/tmp/go-build go test . -run TestProjectAgentModelRouting -count=1
```

Expected: FAIL because `.codex/agents/worker.toml` does not exist.

- [ ] **Step 3: Commit the red test**

```bash
git add agent_config_test.go
git commit -m "测试：定义 sub-agent 模型路由"
```

### Task 2: Add the three custom agents

**Files:**
- Create: `.codex/agents/worker.toml`
- Create: `.codex/agents/worker-terra.toml`
- Create: `.codex/agents/reviewer.toml`
- Test: `agent_config_test.go`

- [ ] **Step 1: Add the Luna default worker**

Create `.codex/agents/worker.toml`:

```toml
name = "worker"
description = "Default implementation worker for clear tasks with a written plan, tests, and explicit acceptance criteria."
model = "gpt-5.6-luna"
model_reasoning_effort = "medium"
nickname_candidates = ["Kepler", "Curie", "Fermi"]

developer_instructions = """
Implement only the bounded task provided by the parent agent.
Follow the supplied plan, file boundaries, tests, and acceptance commands exactly.
Use test-driven development: verify the intended failure, implement the minimum change, then rerun focused and regression checks.
Do not expand scope or modify unrelated files.
Before a command expected to take more than 60 seconds, report what it will verify.
If requirements are ambiguous, the same blocker repeats, or the environment prevents validation, stop and return NEEDS_CONTEXT or BLOCKED with the exact evidence.
Report status, commit SHA, tests run, failures or skipped checks, and residual risks.
"""
```

- [ ] **Step 2: Add the Terra complex worker**

Create `.codex/agents/worker-terra.toml`:

```toml
name = "worker-terra"
description = "Implementation worker for cross-platform, security-sensitive, ambiguous, or difficult debugging tasks."
model = "gpt-5.6-terra"
model_reasoning_effort = "medium"
nickname_candidates = ["Planck", "Gauss", "Noether"]

developer_instructions = """
Implement the bounded complex task provided by the parent agent.
Validate assumptions before editing and preserve concrete failure evidence.
Use test-driven development and run focused checks before broader regression checks.
Treat permissions, atomic writes, platform differences, rollback, quoting, and error paths as first-class requirements.
Do not use repeated retries as a substitute for diagnosis and do not expand scope without approval.
Before a command expected to take more than 60 seconds, report what it will verify.
Return NEEDS_CONTEXT or BLOCKED when a new architecture decision or authority is required.
Report status, commit SHA, tests run, failures or skipped checks, and residual risks.
"""
```

- [ ] **Step 3: Add the Sol reviewer**

Create `.codex/agents/reviewer.toml`:

```toml
name = "reviewer"
description = "Independent specification, code-quality, security, and final integration reviewer."
model = "gpt-5.6-sol"
model_reasoning_effort = "high"
nickname_candidates = ["Atlas", "Delta", "Echo"]

developer_instructions = """
Review as an independent project owner and remain read-only.
Inspect the actual diff, implementation, tests, and commands; do not trust the implementer's completion report.
Check missing requirements and extra scope before reviewing maintainability, portability, security, regressions, and test quality.
Use precise file and line references for every issue and distinguish Critical, Important, and Minor findings.
Do not modify files or create commits.
Approve only when no Critical or Important issue remains.
"""
```

- [ ] **Step 4: Run the focused contract test**

Run:

```bash
env GOCACHE=/tmp/go-build go test . -run TestProjectAgentModelRouting -count=1
```

Expected: PASS.

- [ ] **Step 5: Validate all TOML with the Python standard library**

Run:

```bash
python3 -c 'import pathlib,tomllib; files=sorted(pathlib.Path(".codex/agents").glob("*.toml")); assert len(files)==3; [tomllib.loads(p.read_text()) for p in files]; print("validated", len(files), "agent configs")'
```

Expected: `validated 3 agent configs`.

- [ ] **Step 6: Run repository checks**

Run:

```bash
env GOCACHE=/tmp/go-build go test -c -o /tmp/ai-history-tests .
env GOCACHE=/tmp/go-build go vet ./...
git diff --check
```

Expected: all commands exit 0. `go test ./...` is not used here because installer integration tests require a loopback listener outside the sandbox.

- [ ] **Step 7: Commit the custom agents**

```bash
git add .codex/agents/worker.toml .codex/agents/worker-terra.toml .codex/agents/reviewer.toml
git commit -m "配置：添加 sub-agent 模型路由"
```

### Task 3: Verify configuration loading in a new session

**Files:**
- No repository changes expected.

- [ ] **Step 1: Restart or open Codex from the current feature worktree root**

Exit the current Codex session, change into the feature worktree root, and start a new session there so project-scoped Agent configuration is reloaded.

- [ ] **Step 2: Inspect the loaded Agent list**

Use the Subagents panel or `/agent` and confirm these Agent names are available:

```text
worker
worker-terra
reviewer
```

- [ ] **Step 3: Verify model routing with bounded smoke tasks**

Ask the new session to spawn each role for a read-only one-line status task. Confirm the Agent thread metadata shows:

```text
worker        -> gpt-5.6-luna / medium
worker-terra  -> gpt-5.6-terra / medium
reviewer      -> gpt-5.6-sol / high
```

If the UI does not expose model metadata, confirm the loaded Agent type and inspect Codex configuration logs. Do not infer a successful model selection from the nickname alone.
