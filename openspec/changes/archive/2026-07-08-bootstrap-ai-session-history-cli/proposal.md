# Bootstrap AI Session History CLI

## Why

AI Session History needs to move from the existing Python MCP prototype into a
standalone local-first CLI product. The first product version should help a
developer find prior AI coding sessions, inspect them, and hand off useful
context from one agent or working directory to another without requiring MCP,
Python dependencies, network calls, or model calls.

The existing Python MCP server remains the behavior reference for normalized
sessions, source readers, safe local operation, and context rendering.

## Scope

- Create a Go CLI named `ai-history`.
- Support local read-only session discovery for Codex, Claude Code, and Cursor.
- Support zero-config default path discovery plus optional YAML config.
- Provide `doctor`, `list`, `show`, and `context` commands.
- Support directory-based listing filters.
- Render `context` as a deterministic Markdown handoff pack suitable for moving
  a session from Codex to Claude Code or from one working directory to another.
- Support Cursor latest Windows reading from a real local sample, including
  discovering Windows Cursor storage from a WSL host.
- Support Cursor latest macOS reading from a real local sample.

## Non-Goals

- No `search` command in P0.
- No MCP server in P0.
- No semantic search, embeddings, vector database, LLM calls, or remote service.
- No persistent index or cache database.
- No source history mutation, deletion, archive, resume, fork, or rewrite.
- No compatibility promise for Cursor on native Linux, old Cursor versions, or
  unrecognized future storage formats. Reading Windows latest Cursor data from a
  WSL host is supported because the underlying storage is still Windows Cursor.
- No automatic summarization of key decisions or current state in P0 context
  output.
- No full session export or import command in P0. `context` is the P0 lightweight
  Markdown handoff export; a dedicated read-only `export` command is a P1
  candidate.
- No import into Codex, Claude Code, Cursor, or any source-owned history store.

## Acceptance Criteria

- `ai-history doctor` reports source availability and actionable diagnostics.
- `ai-history list` can list Codex, Claude Code, and supported Cursor sessions,
  with source, limit, exact cwd, and directory-subtree filters.
- `ai-history show <source:id>` reads a session by stable source-prefixed ID and
  supports `clean`, `summary`, and `raw` content modes with size bounds.
- `ai-history context <source:id>` emits a Markdown handoff pack containing
  metadata, initial goal, recent clean conversation, omitted-content notes, and
  optional target cwd.
- The CLI works without a config file when default paths are present.
- Optional config can disable sources, add custom paths, choose default-path
  usage, and set size limits.
- Cursor Windows latest is validated against a real local Windows sample
  (accessed through a WSL host) and converted into a fixture derived from the
  real storage shape, with private content replaced by neutral placeholders.
- The CLI discovers Windows Cursor storage from a WSL host by default, without
  requiring manual config.
- Cursor macOS latest is validated against a real local macOS sample and
  converted into a minimized fixture derived from the real storage shape, with
  private content replaced by neutral placeholders.
- P0 exposes no full session import/export workflow beyond `context` and
  `show --json`.
