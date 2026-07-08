# AI Session History CLI 状态 - 2026-07-08

## 总览

当前工作在分支 `bootstrap-ai-session-history-cli` 上。P0 的 Go CLI 主体已经实现到可验证状态，但 Cursor 真实读取仍未完成，因为当前执行环境无法访问 latest Cursor macOS/Windows 的真实本地历史样本。

OpenSpec change：

- `openspec/changes/bootstrap-ai-session-history-cli`

实施计划：

- `docs/superpowers/plans/2026-07-07-bootstrap-ai-session-history-cli.md`

## 已完成内容

- Go module 和 CLI 入口：`cmd/ai-history`
- 标准库 `flag` 实现 CLI，不使用 Cobra 等 CLI framework
- 命令：
  - `ai-history doctor`
  - `ai-history list`
  - `ai-history show`
  - `ai-history context`
- P0 明确拒绝：
  - `ai-history search`
  - `ai-history export`
  - `ai-history import`
- 统一模型：
  - source
  - source-prefixed session ID
  - session summary/detail
  - turn
  - content mode
  - structured error code
- 配置和路径发现：
  - 默认零配置
  - 可选 YAML config
  - Codex / Claude Code / Cursor 默认路径发现
- Reader：
  - Codex：读取 `state_5.sqlite` 和 rollout JSONL
  - Claude Code：读取 `projects/**/*.jsonl`
  - Cursor：已接入诊断脚手架；未验证格式时返回 unavailable 或 unsupported_format
- Rendering：
  - `clean`
  - `summary`
  - `raw`
  - deterministic Markdown context handoff
- 文档：
  - README 已更新为当前 P0 状态
  - 产品方向文档已更新 P0 命令面
  - OpenSpec tasks 已同步已完成和未完成项

## 当前阻塞点

Cursor 真实读取未完成。

原因：

- OpenSpec 要求 Cursor 不能依赖旧版 synthetic fixture。
- 必须用 latest Cursor macOS 和 Windows 的真实本地样本归纳 fixture。
- 当前执行环境是 Linux，未找到 Cursor 应用历史目录。
- 当前环境中：
  - `cursor` 命令不存在
  - `~/.config/Cursor/User` 不存在
  - `~/Library/Application Support/Cursor/User` 不存在
  - `~/.cursor` 只有 skills，不是 Cursor 应用历史

当前 Cursor reader 的行为是保守的：

- 找不到 `state.vscdb`：返回 `source_unavailable`
- 发现 `state.vscdb` 但未验证格式：返回 `unsupported_format`
- 不会猜测解析，也不会把 Cursor 标记为完成

## 切换环境后的恢复步骤

1. 拉取并切到实现分支：

   ```bash
   git fetch origin
   git switch bootstrap-ai-session-history-cli
   ```

2. 确认 Go 可用：

   ```bash
   export PATH="$PATH:/usr/local/go/bin"
   go version
   ```

3. 跑基线验证：

   ```bash
   GOCACHE=/tmp/go-build go test ./...
   openspec validate bootstrap-ai-session-history-cli --strict
   ```

4. 在能访问 Cursor 数据的环境里定位真实目录。

   macOS 常见路径：

   ```text
   ~/Library/Application Support/Cursor/User
   ```

   Windows 常见路径：

   ```text
   %APPDATA%\Cursor\User
   ```

5. 继续 Cursor 任务：

   - 先只探查结构，不提交隐私内容。
   - 从真实样本归纳最小 fixture 到 `testdata/cursor/macos/` 或 `testdata/cursor/windows/`。
   - 写 failing test。
   - 实现 latest Cursor reader。
   - 再运行 `go test ./...` 和 `openspec validate bootstrap-ai-session-history-cli --strict`。

## 不能做的事

- 不要用旧 Python 原型里的 synthetic `ai-history.sessions` fixture 冒充 Cursor latest 格式。
- 不要把用户真实历史全文提交到仓库。
- 不要 archive OpenSpec change，直到 Cursor macOS 和 Windows latest 样本验证完成。
- 不要实现 `search`、完整 `export` 或 `import` 到 P0。

## 最近验证

在 2026-07-08 已运行：

```bash
PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go test ./...
openspec validate bootstrap-ai-session-history-cli --strict
PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go run ./cmd/ai-history doctor --json
```

结果：

- Go tests：通过
- OpenSpec validate：通过
- `doctor --json`：
  - Codex available
  - Claude Code available
  - Cursor unavailable，原因是当前环境没有 Cursor `state.vscdb`
