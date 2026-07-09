## Why

P0 manual acceptance showed that the core CLI can run, but several command-line
ergonomics and JSON/context semantics are not yet stable enough for daily use or
scripts. Fixing these in P1 should happen before adding MCP adapters or Skills,
so downstream integrations do not wrap unstable CLI behavior.

## What Changes

- Add conventional CLI help entry points: top-level `help`, `--help`, `-h`, and
  subcommand help via `help <command>`, `<command> --help`, and `<command> -h`.
- Add user-visible version output through `version` and `--version`, with a
  clear development-build fallback and future release-build injection path.
- Add stable short flag aliases for common options where they do not conflict
  with future verbose behavior.
- Make `list --json` script-friendly when no sessions match by emitting
  `"sessions": []` instead of `"sessions": null`.
- Decide and implement the P1 listing scope behavior for current-directory
  workflows versus all-history workflows, including explicit switching between
  the two.
- Clarify and improve tool-message handling so useful tool final results,
  command/error summaries, and omitted-content markers have predictable behavior
  across `clean`, `summary`, `raw`, and `context`.
- Keep MCP adapters, Skills integration, full import/export, and interactive/TUI
  mode out of this change.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `cli`: Refine command help/version behavior, flag aliases, list output shape,
  listing scope semantics, and tool-result rendering requirements.

## Impact

- Affected code: `internal/cli`, `internal/core`, `internal/render`, and source
  readers if tool-result extraction needs source-specific changes.
- Affected docs: `README.md`, CLI usage examples, and the `cli` OpenSpec
  capability.
- Affected tests: CLI behavior tests, core service list-result tests, render
  tests, and any reader fixture tests needed to validate tool-result semantics.
- No new runtime dependency is required by default; keep using the existing CLI
  structure unless design shows a clear need for a command framework.
