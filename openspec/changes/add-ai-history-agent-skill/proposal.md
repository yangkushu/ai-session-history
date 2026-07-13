## Why

`ai-history` 已经提供稳定的本地 CLI 和 JSON 协议，但 Codex、Claude Code 与
Cursor Agent 仍需自行推断命令选择、项目范围和权限处理，容易造成无关项目搜索、
过度授权或敏感 raw export。现在需要一份跨 Agent 的可安装 Skill，把这些调用规则和
安全边界固化为可复用工作流。

## What Changes

- 新增一份平台无关的 `ai-history` Agent Skill，统一指导 agent 使用 `doctor`、
  `list`、`search`、`show`、`context` 和 `export`。
- 默认只在当前项目查找历史；扩大到全局范围前必须征得用户同意。
- 默认推荐 clean export；只有用户明确要求完整原始内容时才允许 raw export。
- 为 Codex、Claude Code 和 Cursor 提供按需加载的最小权限说明。
- 使用通用 `npx skills add` 作为 Codex、Claude Code 和 Cursor 的主要安装入口，
  并提供不修改 host permission 配置的手动复制 fallback。
- 增强 `doctor --json`：reader 遇到明确的 history 路径权限错误时返回带路径的
  `permission_denied`，使 Skill 能区分权限失败和来源不存在。
- 更新中英文 README，提供简短的 Skill 安装入口。

## Capabilities

### New Capabilities

- `agent-skill`: 定义跨 Codex、Claude Code 和 Cursor 的 CLI 调用、安装、权限与
  export 安全行为。

### Modified Capabilities

- `cli`: 明确 `doctor --json` 对 history 路径权限错误的诊断 code 和 path。

## Impact

- 新增 `skills/ai-history/` 核心 Skill 与三端 permission references。
- 修改 Codex、Claude Code、Cursor reader 的 `Doctor` 权限诊断及相应测试。
- 修改 README、README.zh-CN.md，并验证 `npx skills` 能发现核心 Skill 和三个 agent
  target。
- `npx` 只用于安装 Skill；不增加 CLI 运行时依赖，也不改变正常 session 读取、
  搜索、渲染或导出协议。
