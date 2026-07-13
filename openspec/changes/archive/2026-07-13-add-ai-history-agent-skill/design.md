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
- 通过通用 `npx skills` 提供三端安装，并保留不修改 permission 配置的手动 fallback。
- 让 `doctor --json` 对明确的 history 权限错误返回可操作的结构化诊断。

**Non-Goals:**

- 不实现 MCP、TUI、import 或新的 session 数据协议。
- 不让 Skill 读取 source history 或复制任何 CLI 业务逻辑。
- 不自动改变 host sandbox、allowlist、managed policy 或 OS 文件权限。
- 不保证识别被 host 有意伪装为 `not found` 的权限拒绝。

## Decisions

### 维护一份开放格式核心 Skill，由通用 CLI 安装

核心内容位于 `skills/ai-history/`，包含 `SKILL.md` 和三个按需加载的 permission
reference。主要安装入口使用 Vercel 开源的 `npx skills add`，由该 CLI 识别
`codex`、`claude-code` 和 `cursor` target，并管理全局或项目级安装。

这比维护三份 `SKILL.md` 更不容易漂移，也比同时制作三种 plugin 更符合本次只分发
Skill 的范围；同时避免本项目自行维护 shell、PowerShell、目标目录映射、更新和删除
逻辑。

### Node.js 只属于安装路径

`ai-history` CLI 继续以预编译 Go binary 分发，运行时不需要 Node.js。`npx` 和网络只在
用户选择通用 Skill installer 时需要。README 同时提供手动复制 fallback，供离线、
无 Node.js 或不愿运行第三方 installer 的用户使用。

无论通过 `npx` 还是手动复制，安装 Skill 都不等于授予 CLI execution 或 history read
权限。文档必须把安装和运行时授权分开说明，不提供自动 permission 配置。

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

- [通用 CLI 或 host 改变 agent identifier、Skill 路径或权限入口] → 发布前复核
  `npx skills` target discovery 和三端官方文档，权限指引集中在独立 reference 中。
- [Sandbox 把拒绝伪装成不存在] → 不误报 `permission_denied`；Skill 说明可能的 host
  限制并引导用户手动检查。
- [Skill 指令不能像代码一样强制执行] → 使用明确 MUST 规则、代表性场景测试和真实
  smoke test 验证 agent 行为。
- [安装需要 Node.js 和网络] → 明确这不是 CLI 运行时依赖，并提供手动复制 fallback。
- [远程安装带来 supply-chain 风险] → 文档要求核对 GitHub 来源和 Skill 内容，不自动
  使用跳过确认或无审阅安装作为唯一入口。

## Migration Plan

1. 先发布标准 Skill 目录和非破坏性的 doctor 诊断增强。
2. 现有 CLI 用户无需迁移；未安装 Skill 的行为完全不变。
3. 安装 Skill 的用户通过 `npx skills add` 选择 host，或按文档手动复制，再按需授予
   最小权限。
4. 回滚时删除新增 Skill 并恢复 reader doctor 逻辑；不涉及用户数据迁移。

## Open Questions

无。实现和发布验收时仅复核外部 host 文档是否改变了安装目录或 permission 入口。
