## Context

`ai-history` v0.3.0 已提供本地只读 CLI、三种 source reader、项目范围搜索、结构化
handoff 和完整 session export。`context-handoff.v1` 与 `session-export.v1` 已适合作为
agent 的机器接口，但仓库尚未提供可安装的 Agent Skill，三种 host 也使用不同的 Skill
目录和 permission 入口。

Skill 不能提升子进程权限；`ai-history` 在 agent terminal 中运行时仍受 host sandbox
约束。当前 reader 的 `Doctor` 对部分 `os.Stat` 或目录遍历权限错误会退化为
`source_unavailable`，因此还需要一个窄范围的诊断增强，才能为 Skill 提供可靠的
`permission_denied` 和路径。

## Goals / Non-Goals

**Goals:**

- 维护一份能被 Codex、Claude Code 和 Cursor 使用的核心 Skill。
- 固化项目优先、JSON 优先、最小授权和敏感 export 确认策略。
- 提供安全、幂等、不修改 permission 配置的跨平台安装器。
- 让 `doctor --json` 对明确的 history 权限错误返回可操作的结构化诊断。

**Non-Goals:**

- 不实现 MCP、TUI、import 或新的 session 数据协议。
- 不让 Skill 读取 source history 或复制任何 CLI 业务逻辑。
- 不自动改变 host sandbox、allowlist、managed policy 或 OS 文件权限。
- 不保证识别被 host 有意伪装为 `not found` 的权限拒绝。

## Decisions

### 维护一份核心 Skill，安装时复制到 host 原生目录

核心内容位于 `skills/ai-history/`，包含 `SKILL.md` 和三个按需加载的 permission
reference。仓库级 shell 与 PowerShell 安装器把相同内容复制到 Codex
`$HOME/.agents/skills`、Claude Code `$HOME/.claude/skills` 或 Cursor
`$HOME/.cursor/skills`。

这比维护三份 `SKILL.md` 更不容易漂移，也比同时制作三种 plugin 更符合本次只分发
Skill 的范围。安装器放在仓库 `scripts/`，避免把安装工具复制进运行时 Skill。

### 安装器默认保守处理已有目标

安装器支持单个 target 和 `all`。目标不存在时复制；内容相同时成功退出；内容不同时
默认报冲突，只有显式 force 才替换。测试通过临时 HOME/profile 覆盖目标路径，不接触
真实用户目录。

安装器不生成或修改任何 permission 文件。相比自动配置，这会多一个人工授权步骤，
但避免安装 Skill 时静默扩大 agent 权限。

### Skill 以 CLI JSON 为机器接口

首次使用先运行 `ai-history version` 和 `ai-history doctor --json`。正常工作流使用
`list/search/show/context` 的 JSON 输出；Skill 不解析原始 source 文件，也不依赖人类
可读输出作为稳定协议。

默认 `list` 与 `search` 使用 `--here`。当前项目无结果后，必须先获得用户同意才能
移除范围限制。该选择优先保护其他项目历史，也减少 agent 选择错误 session 的概率。

### Skill 收紧 export 的 CLI 默认值

CLI 的 raw 默认值保证显式命令行为向后兼容，但 Skill 是更高层的 agent policy。用户
未明确选择 mode 时，Skill 必须说明 raw 风险并推荐 clean；只有明确要求完整、原始或
raw 内容后才执行 raw export。Skill 不自动使用 `--force`，默认建议写到当前 workspace。

### 权限按执行、读取、写入三层处理

Permission reference 分别说明 host 的 scoped approval 入口，但共享以下规则：只允许
具体 `ai-history` 命令；只读实际 source history 路径；只写用户指定的 export 目标。
同一失败不得无变化重试，managed policy 不得绕过。

### 在 reader 边界增强 Doctor 诊断

Codex、Claude Code 和 Cursor reader 在检查候选 history 路径时保留明确的 OS
permission error，并返回 `SourceDiagnostic{Status: "unavailable", Code:
permission_denied, Path: ...}`。不存在的候选仍按现有规则继续查找，正常 available 和
unsupported format 行为不变。

该逻辑留在 reader，而不是 CLI renderer，因为 reader 最接近发生错误的真实路径，且
`doctor`、`list` 和 `search` 已共享 `SourceDiagnostic` / `AppError` 结构。

## Risks / Trade-offs

- [Host 版本改变 Skill 路径或权限入口] → 发布前复核官方文档，路径和 host 指引集中在
  安装器与独立 reference 中。
- [Sandbox 把拒绝伪装成不存在] → 不误报 `permission_denied`；Skill 说明可能的 host
  限制并引导用户手动检查。
- [Skill 指令不能像代码一样强制执行] → 使用明确 MUST 规则、代表性场景测试和真实
  smoke test 验证 agent 行为。
- [PowerShell 无法在所有开发机本地运行] → 用 Windows CI 验证脚本，本地明确记录未
  执行的人工验收。
- [Force 替换会丢失用户自定义 Skill] → 默认冲突退出，force 必须显式提供。

## Migration Plan

1. 先发布 Skill、安装器和非破坏性的 doctor 诊断增强。
2. 现有 CLI 用户无需迁移；未安装 Skill 的行为完全不变。
3. 安装 Skill 的用户按 host 选择用户级目标，并按需授予最小权限。
4. 回滚时删除新增 Skill/安装器并恢复 reader doctor 逻辑；不涉及用户数据迁移。

## Open Questions

无。实现和发布验收时仅复核外部 host 文档是否改变了安装目录或 permission 入口。
