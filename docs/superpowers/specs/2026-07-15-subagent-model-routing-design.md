# Sub-agent 模型路由设计

## 总览

本项目为后续 sub-agent 工作增加项目级 custom agent 配置。主 Agent 继续负责拆分任务、
选择 Agent、提供完整上下文并执行审查 gate；custom agent 只定义不同工作角色的模型、
reasoning effort 和行为边界。

路由目标是在保证质量的前提下降低延迟与使用成本：默认实现工作使用 Luna，复杂实现
升级到 Terra，审查使用 Sol。已创建的 sub-agent 不受配置影响；配置在 Codex 重新加载
项目后用于新创建的 sub-agent。

## Agent 配置

项目新增三个文件：

```text
.codex/agents/
├── reviewer.toml
├── worker-terra.toml
└── worker.toml
```

### `worker`

- 模型：`gpt-5.6-luna`
- Reasoning：`medium`
- 覆盖 Codex 内置同名 `worker`，作为默认实现 Agent。
- 适用：已有明确计划、测试契约、文件范围和验收命令的实现、文档与机械修复。
- 要求：遵循 TDD；不扩大范围；长命令前报告；阻塞时立即返回明确状态。

### `worker-terra`

- 模型：`gpt-5.6-terra`
- Reasoning：`medium`
- 适用：跨平台 installer、安全边界、复杂 shell/PowerShell、非显然调试和需要权衡的实现。
- 要求：先验证假设；保留失败证据；不以重复重试替代诊断。

### `reviewer`

- 模型：`gpt-5.6-sol`
- Reasoning：`high`
- 适用：规格符合性、代码质量、安全性与最终集成审查。
- 默认只读，不修改实现；必须读取实际 diff 和测试，不信任 implementer 的完成报告。

配置省略 `sandbox_mode`、MCP 和权限字段，使其继承父会话的实时安全策略。

## 路由规则

主 Agent 在每次委派前判断任务：

| 任务特征 | Agent | 示例 |
| --- | --- | --- |
| 明确、重复、验收固定 | `worker` | CI、README、已定义的小修复、bundle 命令编排 |
| 跨平台、复杂、安全敏感 | `worker-terra` | PowerShell installer、原子替换、权限和恢复逻辑 |
| 独立审查与高价值判断 | `reviewer` | 规格 gate、代码质量、安全审查 |

主 Agent 不仅依赖 description 自动匹配，还应在委派提示中明确指定角色名称。任务执行中
若 Luna Agent 报告 `BLOCKED`、需要新的架构判断，或连续两次未解决同一问题，后续修复
升级给 `worker-terra`；不让 Luna 无界重试。审查发现的问题仍交给原 implementer 修复，
除非该 implementer 明确阻塞并满足升级条件。

## 可观察性

每个 implementer 必须在报告中包含状态、commit SHA、执行的测试、失败或未执行原因。
预计超过 60 秒的命令应先说明用途；出现 sandbox、network、approval 或测试环境限制时
立即报告具体命令与错误。

用户可以通过 Codex CLI 的 `/agent`、IDE 的 background-agent panel 或 App 的
Subagents `Active` / `Done` 列表查看 thread。主 Agent应能按用户要求列出 Agent 状态、
转发新指令、中断运行或关闭已完成 thread。

## 加载与边界

- 项目级 custom agents 只影响信任该仓库并加载 `.codex/agents/` 的新 Codex session。
- 已经创建的 sub-agent 保持原模型和配置。
- 当前会话暴露的 `spawn_agent` 调用没有显式 `agent` 或 `model` 参数，因此配置写入后
  需要重新加载或新开 session，才能可靠验证角色选择。
- 如果当前账户不可用某个模型，Codex 应报告配置或 entitlement 错误，不静默声称已使用
  指定模型。

## 验收标准

- 三个 TOML 文件具有 `name`、`description`、`developer_instructions`。
- 模型与 reasoning effort 分别符合本设计。
- `worker` 使用内置名称实现覆盖；另外两个 Agent 名称唯一。
- TOML 能被标准 parser 解析，无未知的自定义路由字段。
- 配置不包含个人路径、凭据或本机专用地址。
- 新 session 中能识别三个 Agent，并可查看实际 spawn thread 的角色与模型；若当前界面
  不显示模型，至少通过 Agent 类型和配置加载日志确认。

## 参考

- [Codex Subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents.md)
- [Codex Models](https://learn.chatgpt.com/docs/models.md)
