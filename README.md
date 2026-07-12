# AI Session History

[简体中文](README.zh-CN.md) | English

Local-first CLI for finding AI coding sessions and turning them into clean
handoff context.

`ai-history` reads local session history from supported AI coding tools. It does
not upload your data or require a hosted service.

## Features

- Discover sessions from the current project or all configured sources.
- Search local session titles and conversation content with deterministic,
  ranked matches.
- Inspect a session in JSON, clean text, or summary form.
- Generate deterministic Markdown context for handing work to another agent or
  working directory.
- Read local history from Codex, Claude Code, and Cursor.

## Install

Download a prebuilt archive from
[GitHub Releases](https://github.com/yangkushu/ai-session-history/releases).
Release artifacts include Linux, macOS, and Windows builds plus checksums.

You can also build from source:

```bash
go build -o ai-history ./cmd/ai-history
```

Run the binary directly:

```bash
./ai-history doctor --json
```

Or place it somewhere on your `PATH`, such as `~/bin`.

## Quick Start

Check which local sources are available:

```bash
ai-history doctor --json
```

List sessions for the current project:

```bash
ai-history list --here --limit 10 --json
```

Search the current project's prior conversations:

```bash
ai-history search "release checklist" --here --json
```

Show a session:

```bash
ai-history show codex:<session-id> --mode clean
```

Create handoff context for another project:

```bash
ai-history context codex:<session-id> --target-cwd /path/to/project
```

Generate structured handoff JSON for scripts, Skills, or future MCP adapters:

```bash
ai-history context codex:<session-id> --target-cwd /path/to/project --json
```

## Commands

```bash
ai-history doctor
ai-history list
ai-history search <query>
ai-history show <source>:<session-id>
ai-history context <source>:<session-id>
ai-history version
```

Run `ai-history help` or `ai-history help <command>` for the full command
reference.

`show --json` returns normalized session detail. `context --json` returns a
filtered handoff object with `schema_version: "context-handoff.v1"` for
continuing work in another agent or directory.

Useful short flags:

```bash
ai-history doctor -j
ai-history list -s codex -l 10 -j
ai-history search "release checklist" -s codex -l 20 -j
ai-history show codex:<session-id> -m summary -n 2000 -j
ai-history context codex:<session-id> -t /path/to/project -n 4000
```

## Supported Sources

- Codex local session state and rollout JSONL files.
- Claude Code project JSONL history.
- Cursor local storage on macOS and Windows, including WSL discovery for Windows
  data.

See [Source support](docs/source-support.md) for storage details and current
limitations.

## Development

Run tests:

```bash
go test ./...
```

Run locally:

```bash
go run ./cmd/ai-history doctor --json
```

Use an explicit config:

```bash
go run ./cmd/ai-history doctor --json --config examples/config.yaml
```

Maintainer release notes live in [Releasing](docs/releasing.md). Historical
scope notes and prototype references live in [Project notes](docs/project-notes.md).
