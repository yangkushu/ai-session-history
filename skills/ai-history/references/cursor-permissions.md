# Cursor permissions

Use this reference only when Cursor runs the Skill or a permission failure occurs.

## Narrow recovery

1. For approval limited to the current operation, keep Cursor CLI's interactive per-call approval. A persisted `Shell(ai-history)` rule permits every `ai-history` subcommand because `Shell(commandBase)` matches the command base; add it only when the user accepts that persistent scope.
2. If `doctor --json` reports `permission_denied`, grant `Read(<history-path>)` only for the reported path in Cursor CLI permissions.
3. Cursor editor's Approvals & Execution and sandbox directory controls are a separate surface from Cursor CLI `Shell(commandBase)` and `Read(pathOrGlob)` tokens. Apply the narrow rule in the surface that is running the Skill.
4. A Shell rule does not authorize the user's intent to export or replace a file with `--force`. Grant write access only to the user-selected `<export-path>`, preferably inside the current workspace, and preserve the Skill's explicit confirmation requirements.
5. Avoid a broad allowlist. Do not bypass managed policy or administrator restrictions; if they prevent a narrow command or directory rule, stop and report the block.

Installing the Skill does not grant runtime permissions. Do not broaden shell or filesystem access merely to suppress approval prompts.

## Official references

- [Cursor CLI permissions](https://docs.cursor.com/cli/reference/permissions)
- [Cursor 2.5 sandbox update](https://cursor.com/changelog/2-5)
- [Cursor agent sandboxing](https://cursor.com/blog/agent-sandboxing)
