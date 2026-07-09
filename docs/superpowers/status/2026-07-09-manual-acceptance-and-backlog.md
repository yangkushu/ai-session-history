# Manual Acceptance and Backlog - 2026-07-09

## Summary

This note provides a manual acceptance test case for the current P0 CLI and
records one backlog idea for a later interactive workflow. The CLI remains the
core interface; any interactive mode should be additive.

## Manual Acceptance Test Case: Session Handoff Works End-to-End

### Goal

Verify that a user can find a prior local AI coding session, inspect it, and
generate a Markdown handoff context without mutating source history.

### Preconditions

- The tester has at least one local session from Codex, Claude Code, or Cursor.
- The repository is on `master` and the working tree is clean.
- Go is available on `PATH`, or the tester exports the local Go binary path.

### Steps

1. Build the CLI:

   ```bash
   GOCACHE=/tmp/go-build go build ./cmd/ai-history
   ```

2. Check source availability:

   ```bash
   GOCACHE=/tmp/go-build go run ./cmd/ai-history doctor --json
   ```

3. List recent sessions:

   ```bash
   GOCACHE=/tmp/go-build go run ./cmd/ai-history list --limit 10 --json
   ```

4. Pick one returned `id` and inspect it:

   ```bash
   GOCACHE=/tmp/go-build go run ./cmd/ai-history show <source:session-id> --mode clean --max-chars 2000 --json
   ```

5. Generate handoff context:

   ```bash
   GOCACHE=/tmp/go-build go run ./cmd/ai-history context <source:session-id> --target-cwd <target-project-path> --max-chars 4000
   ```

6. Run a directory filter:

   ```bash
   GOCACHE=/tmp/go-build go run ./cmd/ai-history list --under <project-root> --limit 10 --json
   ```

### Expected Results

- `doctor --json` reports each source independently.
- Unavailable sources are reported as diagnostics and do not prevent available
  sources from working.
- `list` returns stable source-prefixed IDs such as `codex:<id>`,
  `claude:<id>`, or `cursor:<id>`.
- `show` returns normalized summary and turns for the selected session.
- `context` returns Markdown headed `# AI Session Context`.
- The Markdown includes session metadata, original cwd, target cwd, initial
  goal, recent conversation, omitted-content notes, and the handoff instruction.
- The commands do not write to source-owned history stores.

### Manual Notes To Capture

- Which sources were available in the test environment.
- Whether session titles, cwd, turn counts, and previews looked reasonable.
- Whether the selected `Initial Goal` was useful or polluted by bootstrap
  messages.
- Whether empty results render in a script-friendly shape.
- Any source-specific parsing gaps, especially Cursor storage variants.

## Backlog Idea: Interactive CLI Mode

Add an optional interactive workflow in a later version while keeping the
command-oriented CLI as the primary stable interface.

Candidate shape:

- `ai-history tui` or `ai-history interactive` opens a terminal menu.
- The menu lets users select source, filter by directory, browse sessions,
  preview details, and generate context.
- The interactive mode should call the same core service used by `doctor`,
  `list`, `show`, and `context`; it should not introduce a separate behavior
  path.
- The command mode remains required for scripts, agent handoff, and predictable
  automation.

This should be considered after P1 command improvements such as export and
context clean-up rules.
