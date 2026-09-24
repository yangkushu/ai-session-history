# Pi Session Fixtures

These are minimized synthetic Pi JSONL fixtures with neutral project paths and
text. They contain no real session transcript, credentials, private filesystem
paths, or user data.

- `sessions/--example-pi-demo--/*_pi-fixture-v1.jsonl` covers linear v1 records.
- `sessions/--example-pi-demo--/*_pi-fixture-v3.jsonl` covers v3 entry IDs,
  parent-linked active branching, `session_info`, compaction and branch
  summaries, text/tool-call blocks, tool results, bash output, and fields that
  the reader must exclude.

The format references Pi's documented session header and entry structure. Tests
also create temporary malformed, unsupported-version, oversized, duplicate-ID,
symlink, and pi-subagents sidecar cases without committing those generated files.
