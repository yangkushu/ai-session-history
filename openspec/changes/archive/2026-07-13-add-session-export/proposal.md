## Why

`show --json` is useful for immediate inspection, but it is bounded and writes
to standard output. Users need a safe, durable way to save a complete
normalized AI coding session for later review, transfer, or tool consumption.

## What Changes

- Add the read-only `ai-history export <session-id>` command.
- Require an output path and write a versioned session export atomically with
  owner-only file permissions.
- Support complete normalized session exports as JSON (default) and Markdown.
- Support the existing `raw`, `clean`, and `summary` content modes; default to
  `raw` so an export is complete by default.
- Replace the P0-only unavailable-export behavior with the P1 export command.

## Capabilities

### New Capabilities

- `session-export`: Create durable, versioned JSON or Markdown files containing
  a complete normalized session.

### Modified Capabilities

- `cli`: Expose the export command and remove its unavailable-command behavior.

## Impact

- `internal/cli/cli.go` gains export parsing, usage text, and safe file output.
- `internal/cli/service.go` exposes unbounded normalized detail for export.
- `internal/render/` gains a versioned export model and Markdown renderer.
- CLI, rendering, and service tests cover formats, content modes, overwrite
  protection, permissions, and atomic-write failures.
- README files and the CLI specification document the new command.
