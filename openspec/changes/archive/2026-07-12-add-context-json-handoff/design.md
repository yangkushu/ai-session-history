## Context

`context` currently renders Markdown directly from `core.SessionDetail`.
The renderer performs several handoff-specific transformations: clean-mode
turn rendering, setup boilerplate filtering, initial goal selection, recent
conversation selection, tool outcome extraction, note generation, and bounded
output truncation. Those rules are exactly the rules that future Skills and MCP
tools need, but today they are only available as a Markdown string.

The next version should expose those handoff semantics as JSON without changing
the default Markdown behavior. This keeps the CLI as the local capability
boundary while giving later Skill and MCP work a stable machine-readable
contract.

## Goals / Non-Goals

**Goals:**

- Add `ai-history context <session-id> --json` with a deterministic structured
  handoff object.
- Keep Markdown as the default `context` output.
- Share handoff construction logic between Markdown and JSON output.
- Provide a versioned handoff schema with stable core semantics for future Skill
  and MCP consumers.
- Preserve the local-first, read-only, no-network boundary.
- Make the JSON shape suitable for future Skill and MCP adapters without
  requiring those integrations in this change.

**Non-Goals:**

- No MCP server, MCP tool schema, or MCP transport implementation.
- No packaged Skill or agent-specific prompt profile.
- No LLM-generated summary, remote call, embedding model, or vector index.
- No full session export/import command.
- No source-specific raw history JSON export.

## Decisions

### Build a handoff model before rendering

Introduce an internal handoff model, conceptually:

```text
core.SessionDetail
  -> render.BuildHandoff(...)
  -> HandoffContext
      -> Markdown renderer
      -> JSON encoder
```

`HandoffContext` should contain the structured sections that already appear in
Markdown: `session`, `initial_goal`, `recent_conversation`, `tool_outcomes`,
`handoff_notes`, `handoff_instruction`, and `truncated`.

Alternative considered: generate JSON by parsing Markdown. That would preserve
current output quickly, but it would make the machine contract fragile and
couple future integrations to presentation text.

### Version the JSON handoff schema

The JSON object should include `schema_version: "context-handoff.v1"`. Version
1 guarantees the presence and meaning of the core fields:

- `session`
- `initial_goal`
- `recent_conversation`
- `tool_outcomes`
- `handoff_notes`
- `handoff_instruction`
- `truncated`

Compatible additions may add fields without changing the schema version.
Breaking changes to those core meanings require a later version.

Alternative considered: rely on `ai-history version`. That would make
consumers correlate CLI releases with JSON behavior and would be weaker for
Skill and MCP adapters that only see tool output.

### Keep `show --json` and `context --json` separate

`show --json` remains normalized session detail. `context --json` represents a
filtered continuation handoff with target cwd, notes, omission markers, and
handoff instruction. They can share `core.Turn` fields where appropriate, but
the top-level meaning is different.

Alternative considered: add handoff fields to `show --json`. That would blur
the detail-vs-handoff boundary and make it harder for agents to choose the
right command.

`context --json` should not include raw turns, source turns, or a hidden full
detail export. Users and tools that need normalized turns should call
`show --json`.

### Use section-shaped JSON, not source-shaped JSON

The JSON output should hide Codex, Claude Code, and Cursor source differences
behind the existing readers and normalized detail model. It should not expose
source-specific storage fields or raw history records.

Top-level core fields are always present. Arrays are encoded as empty arrays
instead of `null`. Optional source metadata such as title and timestamps may be
omitted when unavailable.

Alternative considered: provide a raw source export mode. That belongs to a
future read-only export capability, not this handoff-focused change.

### Keep handoff notes machine-readable and human-readable

Each handoff note should include a stable `code` and a readable `message`.
Known codes should cover setup boilerplate skipped, noisy tool output omitted,
and context truncated. This avoids forcing future Skill or MCP consumers to
parse English text while keeping CLI JSON usable during manual debugging.

Alternative considered: make notes a string array. That is simpler but too weak
for a machine contract.

### Keep handoff instruction as a required field

`handoff_instruction` remains a required JSON field so the JSON output preserves
the current "ready to hand to another agent" behavior. Future agent profiles can
add fields or override mechanisms without removing the default instruction.

### Apply size limits to handoff sections

`--max-chars` should be treated as a handoff content budget for JSON, not a
strict serialized payload byte limit. JSON field names and escaping are part of
transport representation, not handoff content. The command should reduce
lower-priority content, set `truncated: true`, and include a truncation note
when the handoff content would exceed the requested budget.

The minimum retained data under tight budgets should be session metadata,
initial goal when available, handoff notes, and the truncation state.

## Risks / Trade-offs

- JSON truncation can be less intuitive than Markdown truncation because field
  names and escaping consume serialized output size. Mitigation: document that
  `--max-chars` is a handoff content budget for JSON, keep section priority
  deterministic, and assert the truncation flag and note rather than promising
  exact serialized byte counts.
- Sharing Markdown and JSON construction may require refactoring renderer tests.
  Mitigation: add tests around the handoff model first, then keep the Markdown
  output snapshots focused on user-visible structure.
- Future MCP schemas may need stricter typing than the first JSON output.
  Mitigation: keep field names conservative and section-based; avoid embedding
  source-specific or presentation-only strings as required fields.

## Migration Plan

No data migration is required. Existing users continue to receive Markdown from
`ai-history context <session-id>` unless they pass `--json`. Rollback is safe:
removing the flag before release restores the previous behavior without
affecting source history data.
