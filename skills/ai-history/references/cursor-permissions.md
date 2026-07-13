# Cursor permissions

Use this reference only when Cursor runs the Skill or a permission failure occurs.

## Narrow recovery

1. In Approvals & Execution, grant a narrow `Shell(ai-history)` rule only for the operation being performed.
2. If `doctor --json` reports `permission_denied`, use sandbox directory controls to grant `Read(<history-path>)` only for the reported path.
3. Grant write access only to the user-selected `<export-path>`, preferably inside the current workspace.
4. Avoid a broad allowlist. If managed policy prevents a narrow command or directory rule, stop and report the block.

Installing the Skill does not grant runtime permissions. Do not broaden shell or filesystem access merely to suppress approval prompts.
