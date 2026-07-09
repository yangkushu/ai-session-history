## Why

P1 manual use surfaced that `context` output is technically available but not
yet shaped enough for reliable agent handoff. Before adding MCP adapters,
Skills, or interactive/TUI flows, the CLI context contract should make the
useful parts of a session easy to carry forward while filtering local runtime
boilerplate and noisy tool traces.

## What Changes

- Improve deterministic `context` Markdown structure so a later agent can scan
  metadata, inferred scope boundaries, initial goal, recent conversation, and
  useful tool outcomes in a stable order.
- Clean the initial goal selection so environment preambles, injected
  instructions, local runtime context, and empty/boilerplate user turns are not
  mistaken for the user's actual task.
- Preserve recent user/assistant conversation and concise tool outcomes, while
  continuing to omit large raw tool output and mark omissions clearly.
- Add explicit notes for omitted or skipped content when that helps explain why
  the handoff context is incomplete.
- Keep output deterministic and local-only; do not add LLM summarization,
  embeddings, remote calls, MCP adapters, Skills integration, or TUI behavior in
  this change.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `cli`: Refine deterministic context handoff requirements, especially initial
  goal cleaning, section ordering, recent conversation selection, tool outcome
  inclusion, and omitted-content markers.

## Impact

- Affected code: `internal/render`, `internal/core`, and source reader
  normalization only if existing turn metadata is insufficient for reliable
  context rendering.
- Affected docs: `README.md`, manual acceptance notes, and the `cli` OpenSpec
  capability.
- Affected tests: renderer tests for context structure and filtering, core or
  CLI tests for `context` output, and fixtures that cover boilerplate first
  turns plus useful tool outcomes.
- No new runtime dependency is expected; keep the implementation deterministic
  and local-first.
