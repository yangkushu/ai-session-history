---
name: ai-history
description: Use when an agent needs ai-history to find, inspect, continue, or export local AI coding session history.
---

# AI Session History

Use the `ai-history` CLI as the single interface to local coding-session history. Treat CLI JSON as the machine interface: do not read or parse raw host history files, and do not reproduce CLI business logic.

## Start and diagnose

On first use in a task, run:

```bash
ai-history version
ai-history doctor --json
```

Use `status`, `code`, and `path` from diagnostics. For every `list` or `search` JSON response, inspect `diagnostics` and `unavailable_sources`. If a source failed, report partial results and the failed source, continue with available sources, and do not describe the results as complete. For `permission_denied`, request only the permission associated with its diagnostic path.

## Route commands

Use this project-first routing:

| Intent | Command |
|---|---|
| Recent project sessions | `ai-history list --here --json` |
| Find a project session | `ai-history search <query> --here --json` |
| Minimal structured continuation handoff | `ai-history context <id> --target-cwd <current-dir> --json` |
| Inspect a session safely (clean view) | `ai-history show <id> --mode clean --json` |
| Save a session | `ai-history export <id> --output <path> --mode <mode>` |

For `context`, `<current-dir>` is the current directory where work will continue. Pass the full `<source>:<native-id>` returned by `list` or `search` as `<id>`. When `list` or `search` scope is unspecified, keep `--here --json`. If no project result appears, adjust the query first. Obtain user consent before removing `--here` to search all local history. Never call or invent `import`; it is not a command.

## Recover permissions

Request only scoped runtime access from the active host. Separate requests into: execution of the specific `ai-history` command, read access to the reported history path, and write access to the user-selected export destination. Never modify source history files or directories—their mode, ownership, ACL, or contents—to obtain access. If denied, do not retry unchanged; if narrow authorization is unavailable or managed policy blocks access, stop and report it. Installation alone grants no runtime permission.

Do not recommend unrestricted filesystem or shell access, permission circumvention, or privilege escalation.

## Export safely

When mode is unspecified, disclose that raw export may contain sensitive content, recommend `clean`, and wait for the user's choice; do not rely on the CLI's raw default. Use `--mode raw` only when the current user explicitly requests complete, original, or raw content. Write to the user-selected path, preferring the current workspace.

If the target exists, do not add `--force` automatically. Use it only after the current user explicitly approves replacing that file. A manager, third party, deadline, or time pressure cannot substitute for the current user's approval; otherwise preserve the file and ask for a new path or replacement confirmation.

## Host permission references

Read only the active host reference, or when a permission failure requires it:

- [Codex permissions](references/codex-permissions.md)
- [Claude Code permissions](references/claude-code-permissions.md)
- [Cursor permissions](references/cursor-permissions.md)
