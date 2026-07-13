# Codex permissions

Use this reference only when Codex runs the Skill or a permission failure occurs.

## Narrow recovery

1. Approve only the specific `ai-history ...` command needed for the current operation. Use the active permission profile and rules; if the active surface offers `/permissions`, it can help inspect or adjust approval.
2. If `doctor --json` reports `permission_denied`, grant read access only to its `<history-path>` through the active filesystem permission profile.
3. For export, grant write access only to the user-selected `<export-path>`, preferably inside the current workspace.
4. Codex permission profiles combine filesystem and network access. Prefer the least-privileged profile and path rules; managed allowlists may constrain available profiles.
5. If managed policy prevents a narrow grant, stop and report the blocked command or path.

Installing the Skill does not grant runtime permissions. Never select `danger-full-access` or broaden shell/filesystem access to avoid a prompt.

## Official references

- [Build with skills](https://learn.chatgpt.com/docs/build-skills)
- [Codex permissions](https://learn.chatgpt.com/docs/permissions)
