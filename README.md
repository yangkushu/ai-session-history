# AI Session History

Local-first CLI for reading AI coding session history and building handoff
context.

The command name is `ai-history`. The product name is AI Session History.

## P0 Scope

Implemented P0 commands:

```bash
ai-history doctor --json
ai-history list --under /path/to/workspace --json
ai-history show codex:<session-id> --mode clean --json
ai-history context codex:<session-id> --target-cwd /new/project
```

P0 intentionally does not include `search`, full `export`, full `import`, or
MCP serving. `context` is the lightweight Markdown handoff export for moving a
prior AI coding session into another agent or working directory.

## Development

Run tests:

```bash
PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go test ./...
```

Build the CLI:

```bash
PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go build ./cmd/ai-history
```

Run locally:

```bash
PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go run ./cmd/ai-history doctor --json
```

Use an explicit config:

```bash
PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go run ./cmd/ai-history doctor --json --config examples/config.yaml
```

## Source Support

- Codex: reads `state_5.sqlite` and rollout JSONL files.
- Claude Code: reads `projects/**/*.jsonl`.
- Cursor: P0 target, but real latest macOS and Windows fixture validation is
  still pending. Until then, Cursor storage is diagnosed as unavailable or
  unsupported instead of being parsed speculatively.

## Reference Prototype

The previous Python MCP prototype lives in:

- `<mcp-lab>/servers/ai-history`
- `<mcp-lab>/openspec/specs/ai-history/spec.md`
- `<mcp-lab>/docs/superpowers/status/2026-07-07-ai-history-mcp-status.md`

See `docs/2026-07-07-product-direction.md` for the current design notes.
