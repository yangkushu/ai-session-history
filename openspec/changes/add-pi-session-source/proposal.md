## Why

`ai-history` 目前只能读取 Codex、Claude Code 和 Cursor 的本地会话，无法发现或交接 Pi 的持久化 coding session。Pi 已使用稳定的 JSONL session format（当前 v3），将其纳入同一只读、local-first CLI 可让用户跨 Agent 查找和延续工作。

## What Changes

- 新增 `pi` source，支持 `pi:<session-id>` 的稳定会话标识。
- 将 Pi 纳入现有 `doctor`、`list`、`search`、`show`、`context`（Markdown/JSON）与 `export` 的能力边界，不新增命令。
- 新增 Pi 默认路径发现：环境变量按 Pi 的覆盖优先级择一生效，YAML 自定义路径可额外添加或完全替代默认 root。
- 读取 Pi JSONL session v1、v2、v3，按 Pi 重开会话时的 active leaf 规则选择当前分支。
- 归一化 Pi 的用户、助手、工具结果、compaction 与 branch summary 内容；context 显示有界的既有摘要，所有内容模式均排除结构化图片、thinking/signature、tool arguments 与扩展私有 payload。
- 对未知未来版本、损坏单文件和权限问题提供现有错误模型中的可操作诊断；部分失败时同时返回有效会话和逐文件 warning，不阻塞同一 source 的其它会话。

## Capabilities

### New Capabilities
- `pi-session-source`: 发现、读取、归一化和安全展示 Pi 的本地持久化会话，包括搜索、导出和 handoff。

### Modified Capabilities

- `cli`: 扩展现有诊断、跨源列表与本地搜索的 source 集合，纳入 Pi，且保持其它 CLI 行为不变。

## Impact

- 代码：`internal/core`（含 search）、`internal/config`、`internal/discovery`、`internal/readers`、`internal/cli`、`internal/render`，以及对应测试。
- 文档：`README.md`、`README.zh-CN.md`、`docs/source-support.md`、`examples/config.yaml`。
- 外部接口：`--source` 新增 `pi`，`doctor --json` 默认增加一个 source 诊断，YAML 配置新增 `sources.pi`；Pi 的 `context --json` 可选增加 `persisted_summaries` 字段，保留现有 `context-handoff.v1` 核心字段语义。
- 不增加网络访问、持久索引、写入 Pi 历史或新的第三方依赖。
