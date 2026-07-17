# AI Session History

[简体中文](README.zh-CN.md) | English

Local-first CLI for finding AI coding sessions and turning them into clean
handoff context.

`ai-history` reads local session history from supported AI coding tools. It does
not upload your data or require a hosted service.

[CLI install](#install) · [Bundle install](#install-the-binary-and-skill) ·
[Skill roles](#who-uses-the-skill) · [Quick Start](#quick-start)

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

Install only the CLI on Linux or macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.sh | sh
```

Install only the CLI from PowerShell:

```powershell
irm https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.ps1 | iex
```

See the [installation guide](docs/installation.md) for version pinning, custom
install directories, PATH behavior, updates, verification, and uninstall steps.

## Install the binary and Skill

Add `--with-skill` to install the CLI and the optional Agent Skill together.
You can let the installer detect supported hosts or select targets explicitly.

Linux or macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.sh | sh -s -- --with-skill
```

PowerShell:

```powershell
$script = irm https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.ps1
& ([scriptblock]::Create($script)) -WithSkill
```

Full options, explicit target examples, security boundaries, and recovery steps
are in the [installation guide](docs/installation.md).

## Who uses the Skill

You choose the Skill targets and authorize runtime access. Codex, Claude Code,
or Cursor reads the installed Skill and selects suitable CLI commands. Codex
can invoke it with:

```text
$ai-history Find earlier discussions about the release process in this project.
```

For Claude Code and Cursor, use the Skill invocation displayed by the current
host UI.

The `ai-history` binary performs discovery, search, show, context generation,
and export. Installing the Skill does not grant permission to read history,
execute the CLI, or write exports; the active host still controls those
permissions. Direct CLI use remains fully supported without the Skill.

Review [`skills/ai-history/SKILL.md`](skills/ai-history/SKILL.md) for the Skill
contract and [installation.md](docs/installation.md) for installation details.

## Quick Start

Check which local sources are available:

```bash
ai-history doctor --json
```

List sessions for the current project:

```bash
ai-history list --here --limit 10 --json
```

Omit `--json` for the compact human-readable view:

```text
codex   Example session title
        2026-07-17 10:04 (20m)  2026-07-16 19:28 (15h)
        /workspace/example
        codex:019f6aaf-29f9-7023-a67f-32ba88094b8e
```

The updated timestamp appears before the created timestamp. Absolute timestamps
use the local timezone and parenthesized ages are approximate. Text output is
optimized for reading and may evolve; scripts should use `--json` as the stable
machine interface.

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
