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
- Export a complete normalized session as a private, durable JSON or Markdown file.
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

## Agent Skill

Install the `ai-history` binary first. The optional Agent Skill teaches Codex,
Claude Code, and Cursor when and how to call that CLI; it is not a second
runtime. Review the repository source and
[`skills/ai-history/SKILL.md`](skills/ai-history/SKILL.md) before installing:

```bash
npx skills add yangkushu/ai-session-history \
  --skill ai-history --global \
  --agent codex --agent claude-code --agent cursor
```

Node.js, `npx`, and network access are needed only for this Skill installation
path through the [skills CLI](https://github.com/vercel-labs/skills). The Go
`ai-history` CLI has no Node.js runtime dependency. The installer uses project
scope by default; the command above deliberately selects global scope.

For a manual fallback, copy the same canonical `skills/ai-history/` directory
in full to the active host's Skill directory—do not maintain a second copy:

| Host | Global manual target |
| --- | --- |
| Codex | `$HOME/.agents/skills/ai-history` |
| Claude Code | `$HOME/.claude/skills/ai-history` |
| Cursor | `$HOME/.cursor/skills/ai-history` |

An installer may maintain its own supported host mapping. After installation,
invoke `$ai-history` (or the host's Skill invocation) for session lookup and
handoff requests. The Agent should preflight with `ai-history version` and
`ai-history doctor --json`.

Skill installation does not grant runtime permissions for CLI execution,
history reads, or export writes. It does not change the sandbox, allowlist, or
managed policy. Follow the Skill's host reference and grant only the minimum
access needed for the requested operation.

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

Export a complete session for local archival or transfer:

```bash
ai-history export codex:<session-id> --output session-export.json
ai-history export codex:<session-id> --output session-export.md --format markdown --mode clean
```

Exports can contain sensitive prompts, paths, tool input, and tool output. The
output path is required; new export files use owner-only (`0600`) permissions
and existing files require explicit `--force` to replace them. The default
`raw` mode preserves all normalized turns without a character limit; choose
`clean` or `summary` to reduce noisy tool content.

## Commands

```bash
ai-history doctor
ai-history list
ai-history search <query>
ai-history show <source>:<session-id>
ai-history context <source>:<session-id>
ai-history export <source>:<session-id> --output <path>
ai-history version
```

Run `ai-history help` or `ai-history help <command>` for the full command
reference.

`show --json` returns normalized session detail. `context --json` returns a
filtered handoff object with `schema_version: "context-handoff.v1"` for
continuing work in another agent or directory. `export` writes a versioned,
complete `session-export.v1` file; JSON is the default format and Markdown is
available with `--format markdown`.

Useful short flags:

```bash
ai-history doctor -j
ai-history list -s codex -l 10 -j
ai-history search "release checklist" -s codex -l 20 -j
ai-history show codex:<session-id> -m summary -n 2000 -j
ai-history context codex:<session-id> -t /path/to/project -n 4000
ai-history export codex:<session-id> -o session-export.json -m raw
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
