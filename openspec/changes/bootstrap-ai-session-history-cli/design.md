# Design

## Product Boundary

`ai-history` is the product capability boundary. It owns local source discovery,
storage reading, normalization, filtering, rendering, diagnostics, and safety
defaults. MCP can be added later as an adapter over the CLI/core library, but P0
does not expose MCP.

The CLI is read-only. It never mutates source histories and never writes a
persistent index.

P0 does not include a full session import/export workflow. `context` is the
lightweight Markdown handoff export for continuing work in another agent or
directory, and `show --json` is the machine-readable normalized detail output.
A dedicated read-only `export` command can be designed in P1. Importing into
Codex, Claude Code, Cursor, or their native history stores is out of scope.

## Command Surface

```bash
ai-history doctor [--json] [--config <path>]
ai-history list [--source codex|claude|cursor] [--cwd <path>] [--under <path>] [--limit <n>] [--json] [--config <path>]
ai-history show <source:id> [--mode clean|summary|raw] [--max-chars <n>] [--json] [--config <path>]
ai-history context <source:id> [--target-cwd <path>] [--max-chars <n>] [--config <path>]
```

`context` defaults to Markdown because its primary use is direct handoff to
another agent. JSON is a secondary machine interface for diagnostics, listing,
and detail inspection.

There is no `export` or `import` command in P0.

## Normalized Model

The Go implementation should preserve the old prototype's stable concepts:

- Source names: `codex`, `claude`, `cursor`.
- Session IDs: `<source>:<native-id>`.
- Session summary: ID, source, native ID, title, project, cwd, created time,
  updated time, preview, turn count, availability, reader backend.
- Session detail: summary, normalized turns, truncation flag.
- Turn: role, text, timestamp, kind, omitted flag, omitted reason.
- Content modes: `clean`, `summary`, `raw`.

The implementation can use Go naming conventions, but externally visible JSON
fields should be stable, snake_case, and documented by tests.

## Source Readers

### Codex

Read default local Codex paths and parse storage equivalent to the existing
prototype:

- Thread metadata from `state_5.sqlite`.
- Transcript turns from rollout JSONL files referenced by the thread metadata.

Only message turns with user, assistant, or system roles become normalized
message turns in P0.

### Claude Code

Read default local Claude Code paths and parse project JSONL sessions under
`projects/**/*.jsonl`, preserving the existing prototype behavior for
sessionId, cwd, timestamps, and message content.

### Cursor

Cursor support is part of P0, but storage format support must be sample-driven:

- Windows latest: implemented from a real latest Cursor Windows local history
  sample accessed through a WSL host.
- macOS latest: deferred until a macOS environment and a real local sample are
  available.

Observed Windows latest Cursor storage shape, in `globalStorage/state.vscdb`:

- `composerHeaders` table: one row per composer (Agent) conversation, with
  `composerId`, `workspaceId`, `createdAt`, `lastUpdatedAt`, `isArchived`,
  `isSubagent`, and a `value` JSON blob holding `name`, `subtitle`,
  `workspaceIdentifier.uri.path`, timestamps, and classification flags.
- `cursorDiskKV` table: namespaced key/value rows. Conversation messages live
  under `bubbleId:<composerId>:<bubbleId>`. Each bubble `value` JSON has `type`
  (1 = user, 2 = assistant), `text`, and `createdAt`. Most assistant bubbles
  carry no `text`; their content lives in an encrypted `conversationState` blob
  and content-addressed `agentKv:blob:` rows that P0 does not interpret.

Reader behavior:

- `list` reads `composerHeaders`, excludes composers whose `value` has
  `isArchived = true`, and maps each row to a normalized summary using `name` as
  the title and `workspaceIdentifier.uri.path` as the cwd.
- `show` reads the `bubbleId:<composerId>:*` rows for one composer and emits one
  message turn per bubble whose `text` is non-empty (user or assistant by
  `type`). Bubbles without `text` are not rendered as turns because their
  content is not practically parseable in P0.
- The reader opens `state.vscdb` with SQLite `immutable=1` because the database
  is owned and live-updated by Cursor and may sit on a WSL-mounted Windows
  filesystem where default read-only access fails with a disk I/O error. Bubble
  fields are read with `json_extract` so the large `conversationState` blob is
  never loaded whole.

P0 limitation: Cursor's per-bubble tool results, diffs, and thinking tokens are
stored in `conversationState` and `agentKv:blob:` entries, not in bubble fields.
P0 therefore extracts user and assistant message text only. `clean`, `summary`,
and `raw` modes all apply to Cursor through the shared renderer, but because P0
does not model Cursor tool output as turns, the three modes differ mainly in
size bounds for Cursor sessions. A later revision can reverse-engineer
`agentKv` references for richer tool extraction.

Cursor support must not rely on the old synthetic `ai-history.sessions` fixture
as proof of correctness. Fixtures are derived from the real observed table
shape, column names, key prefixes, and JSON field names, with private content
replaced by neutral placeholders.

If Cursor storage exists but the format is unsupported or unrecognized, `doctor`
and relevant commands report `unsupported_format` with the inspected path and a
concise reason.

## Path Discovery

Default path discovery is per source and per OS:

- Codex: `~/.codex` plus macOS `~/Library/Application Support/Codex`.
- Claude Code: `~/.claude`.
- Cursor macOS: `~/Library/Application Support/Cursor/User` (deferred).
- Cursor Windows: `%APPDATA%\Cursor\User`.
- Cursor on a WSL host: the Windows path is reachable at
  `/mnt/<drive>/Users/<user>/AppData/Roaming/Cursor/User`. When the CLI runs on
  WSL (detected via `/proc/version` mentioning Microsoft), it globs these mounts
  and adds any existing Windows Cursor user directory to the default Cursor
  roots, alongside the native Linux `~/.config/Cursor/User` fallback.

Configured `paths` are always respected. `use_default_paths: false` disables the
default discovery additions, so users can override discovery entirely.

The pure `DefaultPaths(source, goos, home, env)` helper stays free of filesystem
access and remains unit-testable. WSL detection and `/mnt` globbing live in
small testable helpers (`isWSL(versionText)`, glob over an injectable mount
root) and are composed by a resolver the CLI service calls.

## Listing Filters

`list` is the P0 discovery command because P0 has no search.

- `--source` restricts to one source.
- `--limit` bounds returned summaries.
- `--cwd <path>` matches normalized session cwd exactly.
- `--under <path>` matches sessions whose normalized cwd is the supplied path or
  a descendant of it.

Path comparison should expand `~`, clean paths, and compare absolute paths when
possible. If a session has no cwd, it does not match cwd filters.

## Content Rendering

`show` defaults to `clean` mode:

- `clean`: preserve user and assistant text, omit large tool output, terminal
  output, diffs, and logs.
- `summary`: include lightweight tool-call, command, and error summaries without
  large outputs.
- `raw`: return best-effort raw content, still bounded by `--max-chars`.

All modes must be bounded. Truncated output must be explicit.

## Context Handoff

`context` produces deterministic Markdown and does not call an LLM. It should
not invent "Key Decisions" or "Current State" summaries.

The P0 Markdown structure is:

```markdown
# AI Session Context

## Session

- ID: <source:id>
- Source: <source>
- Original CWD: <cwd or unknown>
- Target CWD: <target-cwd or omitted>
- Created: <timestamp or unknown>
- Updated: <timestamp or unknown>

## Initial Goal

<first user message, clean and bounded>

## Recent Conversation

<recent user/assistant turns in clean mode>

## Omitted Content

- Tool output omitted when applicable.
- Transcript truncated when applicable.

## Handoff Instruction

Continue from this prior AI coding session. Treat the original CWD as historical
context and the target CWD, when present, as the active working directory.
```

The renderer should preserve the initial user goal and recent conversation
within `--max-chars`.

## Configuration

The CLI works without a config file. Optional YAML config can be supplied with
`--config`.

Minimum config shape:

```yaml
sources:
  codex:
    enabled: true
    paths: []
    use_default_paths: true
  claude:
    enabled: true
    paths: []
    use_default_paths: true
  cursor:
    enabled: true
    paths: []
    use_default_paths: true
limits:
  detail_chars: 50000
  context_chars: 20000
```

Custom paths are additive unless `use_default_paths` is false.

## Diagnostics and Errors

Errors should be actionable and stable enough for agents:

- `permission_denied`
- `source_unavailable`
- `unsupported_format`
- `session_not_found`
- `invalid_session_id`
- `invalid_config`
- `reader_unavailable`

`doctor --json` should report each source independently so one broken source
does not hide other usable sources.

## Verification Strategy

- Unit tests for normalized IDs, path filters, rendering, config loading, and
  bounded output.
- Fixture tests for Codex and Claude Code based on the existing Python tests.
- Cursor Windows fixture test derived from the real latest Windows storage
  shape, plus a live `list`/`show` check against the real Windows database on a
  WSL host.
- WSL discovery test with a fake `/proc/version` and a fake `/mnt` mount tree.
- Cursor macOS fixture test from a real latest macOS sample (deferred).
- CLI tests for command parsing and JSON/Markdown output shape.
- OpenSpec validation for this change.
