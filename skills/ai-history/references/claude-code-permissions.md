# Claude Code permissions

Use this reference only when Claude Code runs the Skill or a permission failure occurs.

## Narrow recovery

1. Use `/permissions` to allow only the specific `ai-history ...` command required now. A rule may target a concrete command such as `Bash(ai-history doctor --json)`; do not permit whole Bash.
2. If diagnostics report `permission_denied`, add only the reported `<history-path>` with `--add-dir`, `/add-dir`, or the equivalent `additionalDirectories` setting.
3. An additional directory becomes readable, while edit ability follows the current permission mode. Combine the directory addition with deny/ask rules, sandbox settings, and the appropriate mode so the history path remains non-writable.
4. Grant write access only to the user-selected `<export-path>`, preferably inside the current workspace.
5. Managed deny rules cannot be overridden. If managed policy prevents a narrow rule or directory addition, stop and report the blocked command or path.

Installing the Skill does not grant runtime permissions. Do not broaden command or directory access merely to suppress approval prompts.

## Official reference

- [Claude Code permissions](https://code.claude.com/docs/en/permissions)
