# Project Notes

These notes capture project history that is useful for maintainers but too
specific for the main README.

## Current Scope

The first public release focuses on these commands:

```bash
ai-history doctor --json
ai-history list --here --json
ai-history show codex:<session-id> --mode clean --json
ai-history context codex:<session-id> --target-cwd /new/project
```

The release intentionally does not include `search`, full `export`, full
`import`, or MCP serving. `context` is the lightweight Markdown handoff export
for moving a prior AI coding session into another agent or working directory.

## Reference Prototype

The previous Python MCP prototype and its OpenSpec notes are behavior
references only. They live outside this repository and should not be treated as
part of the Go CLI source tree.

See `docs/2026-07-07-product-direction.md` for product direction notes.

## Future Source Work

Run a unified source-support review on a real workstation rather than the
current development server. Treat the desktop and CLI variants of each product
as separate compatibility targets:

- Claude Code desktop and CLI.
- Codex desktop and CLI.
- Cursor desktop and CLI. Cursor Agent being able to invoke the `ai-history`
  skill does not mean Cursor CLI history is currently readable.

For every target, verify storage discovery and session reading through
`doctor`, `list`, `show`, `search`, and `context`. Investigate unrecognized
storage as a distinct source variant before implementing support. Use sanitized
fixtures derived from observed formats and complete a real-environment
acceptance pass without committing private session content.
