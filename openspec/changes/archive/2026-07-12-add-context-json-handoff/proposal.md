## Why

`ai-history context` already produces deterministic Markdown for human-readable
agent handoff, but Skills and future MCP adapters need the same handoff contract
as structured data instead of parsing Markdown. Adding `context --json` now
stabilizes the handoff shape before building Skill or MCP integrations on top.

## What Changes

- Add `ai-history context <session-id> --json` as a structured handoff output
  mode.
- Keep Markdown as the default `context` output and preserve the current
  deterministic handoff semantics.
- Represent the same handoff sections as JSON: session metadata, initial goal,
  recent conversation, tool outcomes, handoff notes, handoff instruction, and
  truncation state.
- Treat the JSON as a semantic contract for future Skill and MCP consumers:
  v0.2.0 stabilizes the core fields and allows compatible additive fields later.
- Keep `show --json` and `context --json` semantically distinct: `show --json`
  remains normalized session detail, while `context --json` is a continuation
  handoff object.
- Do not add MCP serving, Skill packaging, LLM summaries, embeddings, remote
  calls, or full import/export behavior in this change.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `cli`: Add structured JSON output for deterministic context handoff.

## Impact

- Affected code: `internal/cli`, `internal/core`, and `internal/render`.
- Affected docs: README command examples, release or status notes if needed,
  and the `cli` OpenSpec capability.
- Affected tests: CLI tests for `context --json`, renderer tests for structured
  handoff shape, and truncation / omission behavior.
