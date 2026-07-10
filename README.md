# AI Session History

[简体中文](README.zh-CN.md) | English

Local-first CLI for reading AI coding session history and building handoff
context.

The command name is `ai-history`. The product name is AI Session History.

## P0 Scope

Implemented P0 commands:

```bash
ai-history doctor --json
ai-history list --here --json
ai-history show codex:<session-id> --mode clean --json
ai-history context codex:<session-id> --target-cwd /new/project
```

P0 intentionally does not include `search`, full `export`, full `import`, or
MCP serving. `context` is the lightweight Markdown handoff export for moving a
prior AI coding session into another agent or working directory.

## Local Build and Install

Download prebuilt binaries from GitHub Releases when available. Each release
publishes platform archives for Linux, macOS, and Windows, plus `checksums.txt`.

For local development, build and install from the repository:

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

## Releases

Maintainers publish release binaries by pushing a version tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions runs tests and GoReleaser on tags matching `v*`. Release builds
inject version metadata, so release binaries report the tag, commit, and build
date:

```bash
ai-history version
```

Validate the release configuration locally before pushing a tag:

```bash
goreleaser check
goreleaser release --snapshot --clean
```

Snapshot builds write artifacts under `dist/` and do not publish a GitHub
Release.

## Usage

Show top-level help:

```bash
ai-history help
ai-history --help
ai-history -h
```

Show command help:

```bash
ai-history help list
ai-history list --help
ai-history list -h
```

Show version information:

```bash
ai-history version
ai-history --version
```

Local development builds print `dev` unless version information is injected at
build time. A later release pipeline can inject version metadata with Go
`ldflags`.

List sessions under the current working directory:

```bash
ai-history list --here --limit 10 --json
```

List all sessions from enabled sources:

```bash
ai-history list --limit 10 --json
```

Common short aliases:

```bash
ai-history doctor -j
ai-history list -s codex -l 10 -j
ai-history show codex:<session-id> -m summary -n 2000 -j
ai-history context codex:<session-id> -t /new/project -n 4000
```

`context` emits deterministic Markdown for handoff. The output includes stable
sections for session metadata, initial goal, recent conversation, useful tool
outcomes, handoff notes, and a continuation instruction. It filters known setup
boilerplate such as injected environment context before selecting the initial
goal, preserves concise tool results and errors, omits large raw tool output,
and marks skipped or truncated content.

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
