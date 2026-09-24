## Purpose

为 AI Session History 提供 Pi 本地持久化会话的安全只读发现、查看和上下文交接能力，使其与既有 Agent source 使用相同 CLI 工作流。

## ADDED Requirements

### Requirement: Pi source is available through the existing CLI surface
系统 SHALL 将 `pi` 作为有效 source，接受 `pi:<session-id>` 会话标识，并将其纳入 `doctor`、`list`、`search`、`show`、`context`（Markdown/JSON）与 `export` 的既有行为。

#### Scenario: Filter Pi sessions
- **WHEN** 用户执行 `ai-history list --source pi --json`
- **THEN** 输出仅包含 source 为 `pi` 的会话及 Pi source 诊断

#### Scenario: Read a Pi session
- **WHEN** 用户执行 `ai-history show pi:<session-id> --mode clean --json`
- **THEN** 系统返回该 Pi 会话的规范化详情，或返回现有错误模型中的可操作错误

#### Scenario: Search a Pi session
- **WHEN** 用户执行 `ai-history search <query> --source pi --json`
- **THEN** 系统仅在 Pi 的规范化标题及可见 turns 中搜索，并保留与 `list` 一致的 Pi 部分失败诊断

#### Scenario: Export a Pi session
- **WHEN** 用户执行 `ai-history export pi:<session-id> --output <path>`
- **THEN** 系统沿用 `session-export.v1` 输出完整的规范化 Pi turns；默认 raw 不施加 show/context 的字数上限，但仍排除未归一化的结构化 opaque payload

### Requirement: Pi session paths are discoverable and configurable
系统 SHALL 默认发现 Pi 的标准 session storage。启用默认路径时，系统 SHALL 按 `PI_CODING_AGENT_SESSION_DIR`、`PI_CODING_AGENT_DIR/sessions`、`~/.pi/agent/sessions` 的优先级仅选择一个生效的默认 root，不得叠加扫描较低优先级 root。系统 SHALL 允许用户通过 `sources.pi.paths` 额外添加 root（优先于选定的默认 root），并允许以 `use_default_paths: false` 完全禁用默认路径与环境变量路径。

#### Scenario: Discover default Pi storage
- **WHEN** 默认 Pi session storage 中存在有效持久化会话
- **THEN** `ai-history doctor --json` 将 Pi 报告为 `available`，且 `list` 可发现该会话

#### Scenario: Use configured alternate session directory
- **WHEN** `sources.pi.paths` 指向非默认 Pi session directory 且 `use_default_paths` 为 false
- **THEN** 系统仅从配置路径发现 Pi 会话

#### Scenario: Session directory environment override is exclusive
- **WHEN** 设置 `PI_CODING_AGENT_SESSION_DIR` 且标准 Pi session storage 也包含会话，并且 `use_default_paths` 为 true
- **THEN** 系统从该环境变量指定的 root 发现会话，不从标准 root 自动发现会话

### Requirement: Pi v1 through v3 session formats are read as the active branch
系统 SHALL 读取 Pi session format v1、v2 与 v3 的 JSONL 文件。对于含树结构的会话，系统 SHALL 从最后一个有效非 header 条目开始沿 `parentId` 选择 active branch；对于线性 v1 会话，系统 SHALL 保留文件顺序。

#### Scenario: Branched session selects the active branch
- **WHEN** Pi session 含有多个分支且最后一个有效条目位于其中一个分支
- **THEN** 展示与 context 输出仅包含根至该最后条目的分支内容，不包含兄弟分支消息

#### Scenario: Supported legacy session is read
- **WHEN** session header 的 version 为 1、2 或 3
- **THEN** 系统以该版本可用的字段构建会话摘要与详情

#### Scenario: Future session format is rejected safely
- **WHEN** session header 的 version 高于 3
- **THEN** 该文件被报告为 `unsupported_format`，且同一 Pi source 中其它有效会话仍可用

### Requirement: Pi transcript content is normalized without exposing opaque sensitive payloads
系统 SHALL 从 active branch 提取用户与助手的可见 text content，并 SHALL 保留已有的 compaction 与 branch summary 文本以供 handoff 使用。系统 SHALL 把工具输出归一化为可被现有 content mode 省略的 tool result。所有模式（含显式 `raw` 和默认 raw 的完整 `export`）MUST NOT 从 Pi 的结构化字段提取或输出 image base64 data、thinking/signature、tool arguments 或 extension private payload；`show --mode raw` MAY 在 show 字数上限内输出可见 tool result，`export --mode raw` SHALL 保留全部已归一化的可见 tool result 文本，不额外施加字数上限。普通用户/助手文本及 Pi 已持久化的摘要原文不做语义脱敏。

#### Scenario: Clean output omits tool output and opaque content
- **WHEN** Pi session 包含 assistant thinking、tool call arguments、图片内容和 tool result
- **THEN** `show --mode clean` 不输出结构化 thinking、签名、图片数据、tool arguments 或完整 tool result

#### Scenario: Raw output keeps opaque payloads excluded
- **WHEN** 用户执行 `show pi:<session-id> --mode raw` 且会话包含结构化 thinking、image data、tool arguments 和可见 tool result 文本
- **THEN** 输出可以包含有界 tool result 文本，但不包含上述结构化 opaque payload

#### Scenario: Context preserves persisted Pi summary
- **WHEN** active branch 含 compaction 或 branch summary，且使用默认 `context` 长度上限
- **THEN** Markdown `context` 在独立的「Persisted Pi Summaries」区块中呈现最近一条 compaction 与最近一条 branch summary（若存在）的有界文本，不生成新的解释性摘要，并保留 initial goal 与 recent conversation

#### Scenario: JSON context carries the same persisted summaries
- **WHEN** 用户执行 `ai-history context pi:<session-id> --json` 且 active branch 含持久化摘要
- **THEN** JSON handoff 仍使用 `context-handoff.v1` 的既有核心字段语义，并额外包含可选的 `persisted_summaries` 数组；每项包含 `kind`（`compaction` 或 `branch_summary`）及有界的 `text`，与 Markdown 展示选取相同的摘要，不混入 `recent_conversation` 或 `handoff_notes`

#### Scenario: Bounded context cannot fit all summaries
- **WHEN** 用户设置的 `--max-chars` 小于完整 handoff 所需长度
- **THEN** `context` 仍不超过该上限，优先保留 session metadata、initial goal、最近的持久化摘要与 recent conversation；无法完整保留时在长度足够容纳标记的情况下明确标记截断

### Requirement: Pi source failures are isolated and diagnosable
系统 SHALL 忽略单个 Pi JSONL 的空行；递归发现时 SHALL 跳过 Pi-subagents 使用的 `subagent-artifacts/` 子目录，其中的 child transcript 是扩展 event JSONL，而非原生 Pi session。对于其它损坏、不支持或无法读取的文件，系统 SHALL 继续处理同一 source 下其它有效会话。部分失败时 `list --json` 和 `search --json` SHALL 保留有效 sessions 或 hits，并在 `diagnostics.pi` 中返回 `status: "partial"` 及 `warnings` 数组；每项 warning SHALL 包含 `code`、`path`、`message`，`code` 使用现有的 `unsupported_format` 或 `permission_denied`。部分失败不得将 Pi 记入 `unavailable_sources`。`doctor --json` 在有可读会话与文件失败并存时 SHALL 同样报告 `status: "partial"` 与 warnings；无可用会话时 SHALL 将缺失存储、权限失败或无法识别的格式映射为现有 source diagnostic/error code。

#### Scenario: Pi-subagents transcripts are not treated as sessions
- **WHEN** Pi storage root 中的 `subagent-artifacts/` 子目录包含 child transcript JSONL，且旁边存在有效 Pi session
- **THEN** transcript 不会列为 session，也不会产生 Pi source warning；其它目录中的损坏或未知 JSONL 仍按文件报告 warning

#### Scenario: One corrupt file does not hide valid Pi sessions
- **WHEN** Pi storage 同时包含损坏 JSONL 与有效 JSONL
- **THEN** `list --source pi --json` 仍返回有效会话，`diagnostics.pi.status` 为 `partial` 且 `warnings` 中含有损坏文件的 `unsupported_format`、`path` 与 `message`，`unavailable_sources` 不含 `pi`

#### Scenario: Search stays partial with a corrupt Pi file
- **WHEN** Pi storage 同时包含一个损坏文件和一个匹配查询的有效会话
- **THEN** `search <query> --source pi --json` 返回匹配 hit 和 Pi 的 partial warnings，而不将 Pi 放入 `unavailable_sources`

#### Scenario: Show a corrupt Pi session
- **WHEN** `show pi:<session-id>` 匹配到 header ID 正确但正文损坏的文件
- **THEN** 命令返回带该文件路径的 `unsupported_format` 或 `permission_denied`，而非 `session_not_found`

#### Scenario: Pi storage cannot be accessed
- **WHEN** 默认及配置的 Pi storage 均不存在或无法读取
- **THEN** `doctor --json` 为 Pi 返回 `source_unavailable` 或 `permission_denied` 的可操作诊断
