# Cursor permissions

Use this reference only when Cursor runs the Skill or a permission failure occurs.

## Narrow recovery

1. In Cursor CLI, grant a narrow `Shell(ai-history)` rule only for the operation being performed.
2. If `doctor --json` reports `permission_denied`, grant `Read(<history-path>)` only for the reported path in Cursor CLI permissions.
3. Cursor editor's Approvals & Execution and sandbox directory controls are a separate surface from Cursor CLI `Shell(commandBase)` and `Read(pathOrGlob)` tokens. Apply the narrow rule in the surface that is running the Skill.
4. Grant write access only to the user-selected `<export-path>`, preferably inside the current workspace.
5. Avoid a broad allowlist. Do not bypass managed policy or administrator restrictions; if they prevent a narrow command or directory rule, stop and report the block.

Installing the Skill does not grant runtime permissions. Do not broaden shell or filesystem access merely to suppress approval prompts.

## Official references

- [Cursor CLI permissions](https://docs.cursor.com/cli/reference/permissions)
- [Cursor 2.5 sandbox update](https://cursor.com/changelog/2-5)
- [Cursor agent sandboxing](https://cursor.com/blog/agent-sandboxing)
