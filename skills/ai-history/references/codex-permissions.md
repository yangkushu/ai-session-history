# Codex permissions

Use this reference only when Codex runs the Skill or a permission failure occurs.

## Narrow recovery

1. Approve only the specific `ai-history ...` command needed for the current operation. Use the active permission configuration and rules; if the active surface offers `/permissions`, it can help inspect or adjust approval.
2. Permission profiles are Beta and combine filesystem and network access. When Beta permission profiles are active, use the least-privileged active filesystem permission profile to grant read access only to the `<history-path>` reported by `doctor --json`.
3. When the active configuration instead uses legacy `sandbox_mode` or `sandbox_workspace_write`, follow that active legacy sandbox/filesystem mechanism. Do not mix permission profiles with legacy settings.
4. For export, grant write access only to the user-selected `<export-path>`, preferably inside the current workspace.
5. Managed allowlists may constrain available profiles. If managed policy prevents a narrow grant, stop and report the blocked command or path.

Installing the Skill does not grant runtime permissions. Never select `danger-full-access` or broaden shell/filesystem access to avoid a prompt.

## Official references

- [Build with skills](https://learn.chatgpt.com/docs/build-skills)
- [Codex permissions](https://learn.chatgpt.com/docs/permissions)
