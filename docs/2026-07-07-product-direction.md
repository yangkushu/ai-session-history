# ai-history Product Direction - 2026-07-07

## Summary

`ai-history` should become an independent local AI coding history tool. The
core product should be a native CLI distributed as prebuilt binaries. MCP and
Skill integrations should sit on top of the same core capability instead of
being the product boundary.

The existing Python implementation in `mcp-lab` is a useful prototype and
behavior reference, but the long-term product should live in its own repository.

## Problem

The current `ai-history` implementation is a Python MCP server. Users must have
a Python runtime, create or use a virtual environment, and install package
dependencies before registering the server with an MCP client.

That is acceptable for development, but it is too much friction for a local
utility that should be usable by developers who do not care about the
implementation language.

## Recommended Product Shape

Build `ai-history` as a Go CLI with an optional MCP server subcommand.

P0 command surface:

```bash
ai-history doctor --json
ai-history list --under ~/workspaces --json
ai-history show codex:<session-id> --mode clean --json
ai-history context codex:<session-id> --target-cwd ~/workspaces/new-project
```

The CLI is the capability boundary. It should own source discovery, reading,
normalization, filtering, context rendering, diagnostics, and safety
defaults.

The MCP server should be an adapter over the same core logic. It should expose
structured tools for clients that prefer MCP or cannot safely grant shell access
to an agent. MCP is not part of P0.

Search, full session export, and session import are also excluded from P0.
`context` is the lightweight Markdown handoff export; a dedicated read-only
`export` command can be designed in P1.

The Skill should teach agents how to use the CLI and when to fall back to MCP.
It should not be treated as a security boundary.

## Distribution

The primary user path should be prebuilt binary distribution, not local
compilation.

Recommended channels:

- GitHub Releases with platform-specific binaries:
  - `ai-history-darwin-arm64`
  - `ai-history-darwin-amd64`
  - `ai-history-linux-amd64`
  - `ai-history-linux-arm64`
  - `ai-history-windows-amd64.exe`
- Checksums for release artifacts.
- Homebrew tap for macOS and Linux.
- Optional later support for Scoop or winget on Windows.
- `go install ...@latest` only as a developer-oriented fallback.

GoReleaser is a good fit for automating cross-platform builds, checksums,
GitHub Releases, and Homebrew formula updates.

## CLI, Skill, and MCP Relationship

The recommended layering is:

```text
Go CLI: true local capability
Skill: agent workflow and calling strategy
MCP: structured tool adapter for restricted or MCP-native clients
```

CLI plus Skill is enough for agents that can execute local commands. The Skill
can instruct an agent to call:

```bash
ai-history list --under /path/to/workspace --json
ai-history context <session-id> --target-cwd /new/project
```

However, Skill alone does not solve sandbox or permission issues. Agents may be
blocked from executing the binary or reading local history paths. The CLI should
therefore include `doctor --json` to report executable, source discovery, and
read-permission status clearly.

MCP remains valuable when shell execution is restricted but the user can
preconfigure and authorize a tool server. In that case, the agent calls MCP
tools instead of shell commands.

## Safety Defaults

The CLI should be safe by default:

- Read-only operation.
- No network calls.
- No source history mutation.
- No delete, archive, resume, fork, or rewrite commands in the initial scope.
- No persistent index in the first version unless explicitly designed later.
- Bounded output by default.
- `raw` content mode only via explicit option.
- Custom history paths only through explicit config.
- Clear JSON errors for permission issues and unavailable sources.

Example permission error shape:

```json
{
  "error": "permission_denied",
  "source": "claude",
  "path": "~/.claude/projects",
  "hint": "Grant read access to the history path, run outside the sandbox, or use a preconfigured MCP server."
}
```

## Repository Organization

This project should live outside both `mcp-lab` and `skill-lab`.

Recommended repository layout:

```text
ai-history/
  cmd/ai-history/          # Go CLI entrypoint
  internal/history/        # Source discovery and readers
  internal/search/         # Search, filtering, ranking
  internal/context/        # Context rendering
  internal/mcp/            # MCP adapter
  docs/                    # Product notes and design docs
  skills/ai-history/       # Optional Skill packaged with the project
  examples/                # Example configs
  .github/workflows/       # CI and release automation
  .goreleaser.yaml
```

Existing projects should be treated as follows:

- `mcp-lab`: keep the Python MCP server as prototype/reference, or later mark it
  deprecated once the Go version reaches parity.
- `skill-lab`: use for Skill experiments or syncing a published Skill, but not
  for core product code.
- `ai-history`: own the real CLI, release pipeline, MCP adapter, docs, and Skill
  source if desired.

## Current Reference Material

Use the current Python implementation and specs as behavior references:

- `<mcp-lab>/servers/ai-history`
- `<mcp-lab>/openspec/specs/ai-history/spec.md`
- `<mcp-lab>/openspec/changes/archive/2026-07-06-add-ai-history-mcp`
- `<mcp-lab>/docs/superpowers/status/2026-07-07-ai-history-mcp-status.md`

The Python version already defines useful P0 behavior:

- `list_sessions`
- `search_sessions`
- `get_session`
- `build_context`
- Codex, Claude Code, and Cursor source readers.
- `clean`, `summary`, and `raw` content modes.
- Local read-only operation.

## Open Questions For The New Project

- Should P0 include MCP, or should MCP wait until CLI behavior is stable?
- Should P0 avoid a persistent index, or should the new CLI introduce an
  opt-in local derived index?
- Should the Skill live in this repository under `skills/ai-history/`, or in
  `skill-lab` with this repository as the implementation dependency?
- What should the initial supported source set be: Codex and Claude Code first,
  or Codex, Claude Code, and Cursor together?

## Documentation Language

P0 documentation and OpenSpec artifacts keep their current language as-is. From
P1 onward, new requirement records, design notes, plans, and decision documents
should be written in Chinese by default, while keeping technical identifiers,
command names, API fields, and code terms in English where appropriate.
