# ai-history

Local AI coding history reader and context builder.

This repository is intended to become the standalone home for `ai-history`.
The current Python MCP prototype lives in:

- `/home/yangzeqi/workspaces/mcp-lab/servers/ai-history`
- `/home/yangzeqi/workspaces/mcp-lab/openspec/specs/ai-history/spec.md`
- `/home/yangzeqi/workspaces/mcp-lab/docs/superpowers/status/2026-07-07-ai-history-mcp-status.md`

## Direction

`ai-history` should be designed as a local-first CLI product, not only as an MCP
server.

Recommended shape:

- Core: native CLI binary, likely implemented in Go.
- Distribution: prebuilt binaries for macOS, Linux, and Windows.
- Agent workflow: Skill documentation that teaches agents how to call the CLI.
- MCP integration: optional adapter exposed as `ai-history mcp serve`.

See `docs/2026-07-07-product-direction.md` for the current design notes.
