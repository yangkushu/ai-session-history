# Real Acceptance - 2026-07-08

## Summary

Ran the P0 CLI against local real session data after the initial CLI spec was
archived. The acceptance run covered source diagnostics, listing, detail
reading, and Markdown context rendering.

## Commands Covered

- `doctor --json`
- `list --limit 10 --json`
- `show <claude-session> --mode clean --max-chars 1200 --json`
- `context <claude-session> --target-cwd <path> --max-chars 1600`
- `show <codex-session> --mode clean --max-chars 1200 --json`
- `list --source codex --limit 3 --json`
- `context <codex-session> --target-cwd <path> --max-chars 1600`
- `list --under <project-path> --limit 5 --json`

## Results

- Codex and Claude Code were available in the current environment.
- Cursor was unavailable in the current environment because no `state.vscdb`
  was discovered.
- Claude Code `show` and `context` returned normalized turns and Markdown
  handoff output.
- Codex `show` returned real turns from the rollout transcript.
- Directory filtering returned no sessions for the tested project path, which is
  valid for the current local history.

## Fix Applied

The run found a Codex list/detail inconsistency: `list` returned `turn_count = 0`
for Codex sessions while `show` could read hundreds of turns from the same
session. Root cause: `CodexStorageReader.ListSessions()` built summaries without
reading the rollout transcript, so `summaryFromRow` always saw a nil turn slice.

Fix:

- `ListSessions()` now reads each Codex rollout before building the summary.
- `TestCodexStorageReaderListsAndReadsRollout` now asserts that list summaries
  include the expected turn count.

## Follow-up Candidates

- P1 should consider source-specific clean-up rules for environment bootstrap
  messages, such as injected project instructions and local command caveats,
  before selecting the `Initial Goal` for context handoff.
- P1 should consider whether empty `list` JSON responses should render
  `sessions: []` instead of `sessions: null`.
