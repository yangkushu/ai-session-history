## Context

P0 shipped a local-first CLI with `doctor`, `list`, `show`, and `context`.
Manual acceptance confirmed that the main flow is usable, but the CLI still has
rough edges for humans and scripts:

- No conventional top-level help or version command.
- No short aliases for common flags.
- Empty `list --json` results encode `sessions` as `null`.
- Project-local workflows require typing `--under .`, while plain `list`
  returns all sessions.
- Tool result behavior is unclear: render modes know about tool turns, but
  readers mostly emit user/assistant/system messages.

The P1 goal is to stabilize CLI behavior before building MCP adapters or Skills
on top of it.

## Goals / Non-Goals

**Goals:**

- Make CLI discovery predictable through help and version commands.
- Improve common flag ergonomics without introducing ambiguous aliases.
- Preserve script compatibility by keeping `ai-history list` as all-history by
  default while adding an explicit current-directory shortcut.
- Make JSON output shapes stable for empty collections.
- Define tool-result rendering semantics that are useful for handoff without
  dumping noisy tool output.

**Non-Goals:**

- No MCP adapter work.
- No Skills integration work.
- No interactive/TUI mode implementation.
- No full import/export command in this change.
- No speculative parsing of unknown source-specific tool formats.

## Decisions

### Keep the existing CLI parser

Continue using the Go standard library `flag` package and the existing small
dispatcher for P1. A command framework such as Cobra would help with rich help
generation, but it adds a dependency and migration churn before the command set
has stabilized.

Implementation should centralize usage text and flag registration enough to
avoid divergent help output, while staying inside the current structure.

### Preserve `list` default behavior and add `--here`

Keep `ai-history list` as all-history listing. This preserves the existing
specification and avoids breaking scripts that already expect global history.

Add an explicit `--here` flag as a shortcut for filtering to the current working
directory subtree. `--here` should be equivalent to `--under <process cwd>` and
must conflict with explicit `--cwd` or `--under`, because combining them would
make the filter ambiguous.

### Use `--version` and `version`; reserve `-v`

Support `ai-history version` and `ai-history --version`. Do not assign `-v` to
version in P1, because `-v` is commonly used for verbose output and this project
may need verbosity later. If users strongly expect `-v`, decide that separately
after the CLI flag vocabulary is clearer.

Version values should default to a development value such as `dev`. Release
builds can inject `version`, `commit`, and `buildDate` via Go `ldflags` later;
P1 should prepare that path without requiring a release pipeline.

### Short aliases are stable, not exhaustive

Add aliases only where the meaning is conventional and unlikely to conflict:

- `-c` for `--config`
- `-j` for `--json`
- `-s` for `--source`
- `-l` for `--limit`
- `-m` for `--mode`
- `-n` for `--max-chars`
- `-t` for `--target-cwd`

Do not add a short alias for `--under`, `--cwd`, or `--here` in P1; these are
path-scoping flags where clarity is more valuable than compactness.

### Stable empty JSON arrays

Initialize list result slices to empty slices before encoding JSON. Keep
`diagnostics` and `unavailable_sources` as omitted when empty, because they are
diagnostic maps rather than primary collection results.

### Tool results are summarized when reliable

Do not treat all tool messages as equivalent. The normalized model should keep
the distinction between tool calls, tool results, and errors. Rendering should
preserve the useful signal:

- `clean`: omit noisy tool output, but keep tool errors and concise final-result
  summaries when available.
- `summary`: include lightweight command/result/error summaries.
- `raw`: keep raw tool text subject to bounds.
- `context`: include useful final results or error summaries when they help a
  later agent continue, but never include large raw tool output.

Reader changes should only extract tool results from source fields that are
well understood. Unknown or ambiguous tool formats should remain omitted rather
than guessed.

## Risks / Trade-offs

- Help output may drift from flag registration if usage text is duplicated.
  Mitigation: keep usage helpers close to flag registration and cover them with
  CLI tests.
- Keeping `list` default as all-history may be less convenient for the most
  common project-local workflow. Mitigation: add `--here`, document it
  prominently, and let future interactive mode default to current directory if
  that proves better.
- Tool-result extraction can become source-specific and brittle. Mitigation:
  require tests from known storage shapes and avoid speculative parsing.
- Adding aliases creates long-term compatibility obligations. Mitigation: add
  only conservative aliases and leave ambiguous flags without short forms.

## Migration Plan

This is a non-breaking CLI improvement. Existing long flags and commands remain
valid. Scripts using `list --json` should only see a more stable empty result
shape.

Rollback is straightforward: revert the CLI behavior change commit before
release if any new alias or help behavior conflicts with existing usage.

## Open Questions

- Should `-v` ever mean version, or should it remain reserved for future verbose
  logging?
- Which source reader should be the first to preserve reliable tool final
  results: Codex, Claude Code, or Cursor?
- Should a future interactive mode default to `--here` even though command mode
  keeps all-history as the default?
