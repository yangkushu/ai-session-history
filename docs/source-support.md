# Source Support

`ai-history` reads local files produced by AI coding tools. It opens those files
read-only and does not upload session data.

## Codex

Codex support reads `state_5.sqlite` and rollout JSONL files from local session
state.

## Claude Code

Claude Code support reads project history from `projects/**/*.jsonl`.

## Cursor

Cursor support reads `globalStorage/state.vscdb` using `composerHeaders`,
`cursorDiskKV`, and `bubbleId:<composerId>:<bubbleId>` rows where available.

Windows Cursor data can be auto-discovered from a WSL host. macOS support covers
the observed `cursorDiskKV` `composerData:<composerId>` shape.

The Cursor database is opened with SQLite `immutable=1` so the CLI can safely
read a live WAL-mode database without mutating it.
