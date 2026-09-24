## 1. Source contract and configuration

- [x] 1.1 在 `internal/core/models.go`、ID 校验、doctor、list 与 `searchSources` 中加入 `pi`，并更新 core/CLI 单元测试验证 `pi:<native-id>`、`list/search --source pi` 与 Pi doctor JSON 输出。
- [x] 1.2 扩展 YAML 默认配置、`examples/config.yaml` 与 config tests，加入默认启用的 `sources.pi`，并验证 `enabled`、`paths`、`use_default_paths` 的既有合并语义。
- [x] 1.3 在 discovery 中实现 Pi session root 的可注入解析：配置路径之外仅从 `PI_CODING_AGENT_SESSION_DIR`、`PI_CODING_AGENT_DIR/sessions`、默认 agent sessions root 中择一追加；以跨平台单元测试验证环境变量覆盖不叠加、相对路径解析、配置路径优先和 `use_default_paths: false`。

## 2. Pi reader and partial diagnostics

- [x] 2.1 为 Pi reader 增加可选的 sessions+warnings 接口（不改其它 reader 的 `ListSessions` 签名），扩展 `SourceDiagnostic.warnings`（code/path/message）与 core `ListResult`/`SearchResult` 的 Pi partial diagnostics；以 list/search/doctor tests 验证有效 sessions 或 hits 与 warning 并存、Pi 不列入 `unavailable_sources`。
- [x] 2.2 创建脱敏 `testdata/pi/` fixture 与 `internal/readers/pi_test.go` 的基础失败测试，覆盖 v1、v2、v3 header、名称、CWD、时间和 source-prefixed ID；确认 fixture 不含真实历史、凭据或个人路径。
- [x] 2.3 实现 Pi JSONL 文件发现和 header 读取：递归扫描 `.jsonl`、跳过已知 `subagent-artifacts/` 扩展目录、不跟随 symlink directory、按 root/path 确定性排序、首个重复 native ID 优先；以 reader tests 验证默认会话分组目录、重复 ID 和不可用 storage 诊断。
- [x] 2.4 实现有界 JSONL 解析及单文件故障隔离，接受缺失 version/v1/v2/v3，拒绝 v4+；以测试验证空行可跳过，非空损坏行、超大行、缺 header、未知版本导致整文件跳过并出现在 `warnings`，且有效邻居文件仍可列出。
- [x] 2.5 实现 active branch 投影：v2/v3 从最后有效 entry 回溯 `parentId`，v1 保留线性顺序并容忍孤儿链；以 fixture tests 验证兄弟分支不会出现在 `show/context`。
- [x] 2.6 实现 Pi entry 到 normalized turn 的安全映射：提取 user/assistant text，转换 tool result，以独立 kind 保留 compaction 与 branch summary 来源，忽略 system prompt、thinking/signature、image data、tool arguments、custom payload、usage/label/context edit；以 clean/summary/raw reader-renderer tests 验证 show 均不泄露结构化 opaque 字段，显式 show raw 仅可额外显示有界可见 tool result。
- [x] 2.7 实现 `Doctor` 与 `GetSession` 的 Pi 错误映射，并以单元测试验证 `available`/`partial`、`source_unavailable`、`permission_denied`、`unsupported_format` 与 `session_not_found`；header ID 匹配但正文损坏时 `show` 必须返回文件错误而非 not found。
- [x] 2.8 根据真实 Pi 0.87.1 存储验证识别 pi-subagents child transcript sidecar；只跳过 `subagent-artifacts/`，保留其它未知 JSONL 的文件级 warning，并加入 reader 回归测试。

## 3. CLI integration and documentation

- [x] 3.1 在 `internal/cli/service.go` 注册 Pi reader；以 CLI integration tests 验证 doctor、list、search、show 对 Pi 的默认路径及自定义路径工作，且失败诊断一致。
- [x] 3.2 在 `render.BuildHandoff` 与 `ContextFromHandoff` 中增加 Pi 可选 `persisted_summaries` 和 Markdown 独立区块，避免 `ContextHandoff` 经 `Show` 的 detail limit 提前截断；以 Markdown/JSON tests 验证相同摘要来源、默认预算有界显示、极小预算按既有 JSON 与 Markdown 规则截断、旧源的 `context-handoff.v1` 核心字段不变。
- [x] 3.3 验证 `export pi:<id>` 复用 `session-export.v1`：raw 包含完整 normalized turns 与可见 tool result 且不受 show 上限截断，clean 遵守工具省略行为，两者均排除结构化 opaque payload；以 export tests 验证。
- [x] 3.4 更新 `README.md`、`README.zh-CN.md`、`docs/source-support.md`、`examples/config.yaml`，说明 Pi 支持版本、默认发现路径、环境变量覆盖、search/export、完整 raw 导出与自定义 session directory 配置；核对文档不包含个人机器路径、用户名或会话内容。

## 4. Verification and release readiness

- [x] 4.1 执行 `gofmt -w cmd internal`，运行 `go test ./...`，并修复所有失败；验证测试覆盖 Pi branch、format、partial failure 与安全过滤场景。
- [x] 4.2 使用无敏感输出的本地 Pi fixture、本机 Pi 0.87.1 session storage 及实际 `ai-history doctor`、`list`、`search`、`show`、`context`（含 `--json`）和 `export` 命令完成手工 smoke check，确认默认 root 和 YAML alternate root 均可工作；导出仅写入新建的临时私有文件。
- [x] 4.3 执行 `openspec validate add-pi-session-source --strict`，确认 change 可实施；实现完成且验收通过后再决定是否 archive、git commit 与 git push。
