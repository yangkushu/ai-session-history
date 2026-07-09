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

## Local Build and Install

This project does not publish prebuilt binaries yet. Build and install locally
from the repository:

```bash
PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go build -o ai-history ./cmd/ai-history
```

Run the built binary directly:

```bash
./ai-history doctor --json
```

Install it into a user-local bin directory:

```bash
mkdir -p ~/bin
cp ai-history ~/bin/
```

Make sure `~/bin` is on `PATH`. For bash, add this to `~/.bashrc` if needed:

```bash
export PATH="$HOME/bin:$PATH"
```

Then verify the installed command:

```bash
ai-history doctor --json
```

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
- Cursor: Windows latest is supported, reading `globalStorage/state.vscdb`
  (`composerHeaders` + `cursorDiskKV`) and `bubbleId:<composerId>:<bubbleId>`
  rows. Windows Cursor data is auto-discovered from a WSL host. macOS latest is
  supported from the observed `cursorDiskKV` `composerData:<composerId>` shape.
  The database is opened with SQLite `immutable=1` to safely read Cursor's live,
  WAL-mode file.

## Reference Prototype

The previous Python MCP prototype and its OpenSpec notes are behavior
references only. They live outside this repository and should not be treated as
part of the Go CLI source tree.

See `docs/2026-07-07-product-direction.md` for the current design notes.
