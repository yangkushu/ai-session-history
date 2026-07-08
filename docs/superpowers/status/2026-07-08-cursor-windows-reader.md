# Cursor Windows 读取器 状态 - 2026-07-08

## 总览

本次工作补齐了 P0 的 Cursor Windows latest 读取器，并让 WSL 主机默认发现
Windows 侧 Cursor 数据。后续合并已补齐 Cursor macOS latest 的 `composerData`
读取路径，OpenSpec change `bootstrap-ai-session-history-cli` 仅剩归档前确认。

## 已完成

- WSL 探测：`internal/discovery` 新增 `isWSL`、`windowsCursorRootsUnder`、
  `cursorRootsFor`（可注入测试）与 `ResolveRoots`（生产入口）。WSL 下默认 glob
  `/mnt/*/Users/*/AppData/Roaming/Cursor/User`。
- Cursor reader：`internal/readers/cursor.go` 重写。从 `composerHeaders` 列出
  非 archived 的 composer；从 `cursorDiskKV` 的 `bubbleId:<id>:*` 读消息 turn
  （`type=1`→user、`type=2`→assistant，仅取有 `text` 的 bubble）；以
  `immutable=1` 打开 DB。
- Fixture：测试内 `writeCursorState` 临时生成真实表结构 + 合成中性内容，沿用
  `writeCodexState` 模式；不提交二进制或真实数据。
- 文档：`testdata/cursor/windows/README.md` 留档真实存储形状；`README.md` 更新
  Cursor 支持状态。

## P0 已知限制

- Cursor 的工具结果、diff、思考内容存在加密 `conversationState` 与内容寻址的
  `agentKv:blob:`，P0 不解析；故 cursor 的 `clean`/`summary`/`raw` 差异主要为
  尺寸边界，不产生 tool_result turn。
- `immutable=1` 跳过 WAL，极近期未 checkpoint 的数据会滞后；只读历史工具可接受。
- Cursor macOS latest 已用真实 `composerData:<id>` 结构验证；P0 仅解析明文
  conversation 消息，不解析加密或内容寻址 blob。

## 不做的事

- 归档 OpenSpec change 前需要完成最终验证并确认是否保留 P1 后续事项。
- 不实现 `search`/`export`/`import`。
- 不提交任何真实 Cursor 历史内容。

## 验证（WSL2 + Windows 侧 Cursor）

- `gofmt -l .` 干净；`go vet ./...` 通过。
- `go test ./...` 全通过（含新增 discovery WSL 与 cursor reader 用例）。
- `go build ./cmd/ai-history` 成功。
- `doctor --json`：cursor 为 `available`。
- `list --source cursor --json`：列出真实 composer 会话。
- `show cursor:<id> --json`：读到真实消息 turn。
- `context cursor:<id>`：输出 Markdown 移交包。
- `openspec validate bootstrap-ai-session-history-cli --strict` 通过。
