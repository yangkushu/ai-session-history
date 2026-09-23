## Context

现有 CLI 的 source reader 将本地存储归一化，core.Service 与 scanSearcher 分别服务于 list/show 和 search，appService 还提供 Markdown/JSON context handoff 与 `session-export.v1` 完整导出。Pi 0.87.1 将持久化 session 保存为 JSONL，默认根目录是 `~/.pi/agent/sessions/`；每个文件含一个 session header 和 append-only entry tree。Pi 重开文件时把最后一个非 header entry 设为 active leaf。详见 proposal.md 与 `specs/pi-session-source/spec.md`。

## Goals / Non-Goals

**Goals:**

- 在不改变既有命令与 JSON 核心字段语义的前提下接入 Pi source；搜索、完整导出、Markdown/JSON handoff 均可用。
- 以 Pi 的 active-branch 规则读取 v1–v3 会话，并在保守内容过滤下输出稳定的 normalized detail。
- 让默认发现、环境变量发现及 YAML 路径覆盖可单元测试。
- 保持 source 之间和单个 session 文件之间的故障隔离。

**Non-Goals:**

- 不写入、恢复、fork 或删除 Pi session。
- 不扫描 Pi 项目 settings 以推测历史的自定义 `sessionDir`；用户必须通过环境变量或 `sources.pi.paths` 提供该位置。
- 不解析图片、thinking/signature、tool arguments、extension private data 或未知 entry payload。
- 不承诺读取 v4+，不复刻 Pi 的完整上下文投影（例如 context edit 的模型可见替换）。

## Decisions

### 使用独立 `PiStorageReader` 与 `pi` source

在 core 的 source 枚举、默认 config、reader 注册和 doctor source 列表中加入 `pi`，由独立 reader 读取 JSONL。

- 选择原因：与 Codex、Claude Code、Cursor 的 storage-reader 架构一致，CLI、filter、renderer 与 error model 无需分叉。
- 替代方案：调用 Pi CLI/SDK 列举会话。拒绝，因为这会引入 Node/Pi 运行时依赖，破坏 Go CLI 的 local-first、零额外运行时边界。

### 路径优先级与递归发现

显式配置的 `paths` 按配置顺序优先；若 `use_default_paths: true`，再按 `PI_CODING_AGENT_SESSION_DIR`（若设置）→ `PI_CODING_AGENT_DIR/sessions`（若设置）→ `~/.pi/agent/sessions` **择一**追加，不扫描被覆盖的默认 root。若 `use_default_paths: false`，仅保留配置路径。空环境变量视为未设置；相对 session root 按当前工作目录解析。reader 在各生效 root 内递归查找 `.jsonl`，不跟随 symlink directory，并按路径排序；重复 native ID 保留优先 root 的首个结果。

- 选择原因：默认目录下以编码 CWD 分组，必须递归；环境变量是 Pi 官方支持的运行时覆盖，不应让显式隔离的 session directory 混入默认历史；明确优先级避免 `pi:<id>` 不稳定。
- 替代方案：读取所有 Pi user/project settings 推导 `sessionDir`。拒绝，因为 project-level relative `sessionDir` 需要枚举未知项目，且会将配置发现扩展成不可靠的全盘扫描。

### 以最后有效 entry 重建 active branch

reader 对每个文件解析 header 与有效 entries；对于带 `id/parentId` 的 v2/v3，从文件最后一个有效 entry 沿 parent 链回溯并反转为 root-to-leaf。v1 没有树链接时按有效文件顺序视为单一分支。遇到孤儿 entry 时止于可达链，且不使整个 source 失败。

- 选择原因：与 Pi `SessionManager` 重开文件时的 leaf 选择一致，可使 `show/context` 与 Pi 当前会话一致。
- 替代方案：展示所有分支。拒绝，因为会混入已放弃的对话，并使 handoff 与 Pi active context 不一致。

### 最小可见内容投影与 shared renderer

reader 仅输出用户和助手的 text blocks；工具结果输出为 `RoleTool/KindToolResult`，使现有 clean/summary/raw renderer 继续负责内容模式。`search` 只匹配 normalized turns，保留既有分类得分和 snippet 上限；`export` 复用 source-neutral 的 `BuildSessionExport`，默认 raw 不额外施加字数上限，但只能导出 reader 已归一化的可见内容。`compaction` 与 `branch_summary` 使用独立的摘要 turn kind（例如 `KindPersistedSummary`，并记录摘要来源）。扩展现有 `HandoffContext` 为 Pi 可选的 `persisted_summaries` 数组（元素为 `kind/text`）；它属于 handoff 内容预算，不改变 `context-handoff.v1` 已有字段语义。Markdown 与 `context --json` 从同一个 `BuildHandoff` 选出 active branch 最近一条 compaction 与最近一条 branch summary，Markdown 在 `## Persisted Pi Summaries` 区块展示；不能仅当作 assistant 消息：当前 `render.BuildHandoff` 的 `recentConversation()` 只收集 user/assistant，会漏掉摘要。上下文按 metadata、initial goal、最近的持久化摘要、recent conversation 的优先序分配 `MaxChars`；工具 outcomes 与其它 notes 沿用现有规则且不被 Pi 摘要静默替换。各摘要有单独文本上限，超长时明确标记截断；默认上限下也只能保证展示有界摘要文本，而非保证输出全文。Markdown 极小 `MaxChars` 不能容纳完整结构时仍须保持上限；能容纳标记时显示截断标记，否则只输出有界前缀。JSON 的 `--max-chars` 仍是 handoff 内容预算，并非序列化 JSON 字节上限。reader detail 在渲染前保留完整活跃分支，避免 `appService.ContextHandoff` 先经 `Show` 的 detail limit 截断后把最后摘要/近期对话丢掉；可从 core 获取未渲染详情后由 context renderer 一次性限长。

系统 prompt、thinking、images、tool call arguments、usage、labels、custom state 和未知 entry 一律不进入 normalized text；即使显式 `show --mode raw` 或默认 raw 的 `export` 也不得恢复这些结构化 payload。show 遵守 detail limit，完整 export 保留所有 normalized turns（不受 detail/context limit 截断）。普通可见文本与既有摘要可能本身敏感，不承诺语义脱敏。`context_edit` 作为元数据忽略，保留原始历史文本。

- 选择原因：默认安全，同时对 handoff 中的 Pi 已持久化摘要提供显式位置；不新增 source-specific JSON API 或泄露 opaque provider/extension data。
- 替代方案：完整复刻 Pi `buildSessionProjection`。拒绝，因为其目标是未来模型 context 而非历史查看，会改变历史文本并扩大对 Pi 内部格式的耦合。

### 版本与容错策略

仅接受 header version 缺失/v1/v2/v3；高于 v3 的单文件标记 `unsupported_format`。解析使用有上限的行读取；空行跳过，单行超过上限、非空 JSON 无法解析或 header 缺失时跳过整文件并记录其失败。即使 header 可读取，损坏正文也不能当作成功的会话摘要，否则 `show` 会与 `list` 不一致。对 header ID 能匹配的损坏/不可读文件，`GetSession` 返回带路径的 `unsupported_format`/`permission_denied`，而不是 `session_not_found`。

现有 `Reader.ListSessions() ([]SessionSummary, error)` 无法同时返回 sessions 和 warnings，因此新增**可选**的 reader 列表诊断接口，仅 Pi 实现，core.Service 与 scanSearcher 在调用时检测它；其它 reader 保持原接口。扩展 `SourceDiagnostic` 增加 `warnings`（元素含 `code/path/message`），`status: "partial"` 表示有可读会话也有失败文件；`ListResult.diagnostics.pi`、`SearchResult.diagnostics.pi` 与 doctor 都使用同一结构，部分成功时 `unavailable_sources` 不含 `pi`。搜索依然遵守 `GetSession` 二次读取失败时跳过该 hit 的现有语义。无可读会话时，根据存储不存在、权限失败或格式失败返回最具体的 `source_unavailable`、`permission_denied` 或 `unsupported_format`。

- 选择原因：Pi JSONL 可包含大型 tool output/image；需要避免无界内存，同时不能用一个坏文件隐藏其它历史，又不必破坏现有 reader 的公开接口。
- 替代方案：宽松接受未知版本。拒绝，因为 entry 含义和分支语义变化时会产生静默错误 handoff。

## Risks / Trade-offs

- [Pi 更新存储格式或新增必要 entry 类型] → 仅承诺 v1–v3；future version 明确报 `unsupported_format`，以 fixture 驱动后续扩展。
- [大型 JSONL 行被上限跳过] → 记录单文件部分失败；保留可配置外的安全固定上限，测试覆盖长行和过大行。
- [自定义 sessionDir 不在当前环境变量中] → 提供 `sources.pi.paths`，在 doctor 诊断中提示配置该路径。
- [session ID 在多 root 重复] → 固定 root/path 顺序并首个优先，保证 list/show 解析一致。
- [已持久化 summary 含敏感内容] → 它是原会话可见上下文的一部分；沿用现有 `show/context` 的本地只读和输出边界，不增加传输。

## Migration Plan

1. 发布时将 `pi` 默认启用，因此未提供 config 的现有用户会在 doctor/list/search JSON 中看到额外 Pi source 或 unavailable 诊断。Markdown context 与 JSON handoff 来自同一模型；仅 Pi 有摘要时附加可选 `persisted_summaries`，完整 export 仍使用 `session-export.v1`。
2. 现有 source、session ID 和命令参数保持不变；无数据迁移。
3. 若 Pi reader 造成问题，用户可通过 `sources.pi.enabled: false` 回退；代码回滚只移除新 source，不影响现有历史文件。
