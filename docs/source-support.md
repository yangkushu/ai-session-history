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

## Pi

Pi support reads persistent session JSONL files in format v1, v2, and v3. The
standard storage root is `~/.pi/agent/sessions/`; session files are grouped
recursively by working directory. For tree-format sessions, `show`, `search`,
and handoff use the active branch ending at the final entry in the file.

When default paths are enabled, the effective root is selected exclusively in
this order:

1. `PI_CODING_AGENT_SESSION_DIR`
2. `PI_CODING_AGENT_DIR/sessions`
3. `~/.pi/agent/sessions/`

Set `sources.pi.paths` in the YAML config to add alternate roots. Set
`use_default_paths: false` to use only configured roots. Relative configured
paths resolve from the current working directory. A Pi-specific custom
`sessionDir` that is not reflected in these environment variables must be
configured explicitly.

The reader is read-only. It exports visible user/assistant text, tool results,
and persisted compaction/branch summaries through the existing CLI. For example,
`list --source pi`, `search "query" --source pi`, and
`export pi:<id> --mode raw` use the same Pi session IDs. Structured
thinking/signatures, image data, tool arguments, and extension-private payloads
are not extracted, including by `show --mode raw` and full raw export. A corrupt
or unsupported file is isolated from other sessions and reported as a per-file
warning; versions newer than v3 are not parsed speculatively.
