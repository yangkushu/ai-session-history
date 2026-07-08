# Cursor Reader 续作 Prompt

把下面这段 prompt 复制给能访问 Cursor 本地历史数据的 AI agent 使用。

```text
请继续开发 AI Session History 项目的 Cursor reader 部分。

项目仓库：
<ai-session-history-repo>

请先执行：

git fetch origin
git switch bootstrap-ai-session-history-cli

然后阅读这些文件，恢复上下文：

1. docs/superpowers/status/2026-07-08-ai-session-history-cli-status.md
2. openspec/changes/bootstrap-ai-session-history-cli/proposal.md
3. openspec/changes/bootstrap-ai-session-history-cli/design.md
4. openspec/changes/bootstrap-ai-session-history-cli/specs/cli/spec.md
5. openspec/changes/bootstrap-ai-session-history-cli/tasks.md
6. internal/readers/cursor.go
7. internal/readers/cursor_test.go

当前状态：

- P0 非 Cursor 部分已经实现并验证通过。
- Codex reader、Claude Code reader、CLI、config、rendering 都已完成。
- Cursor 目前只有诊断脚手架。
- 当前 Cursor reader 在发现 state.vscdb 时会返回 unsupported_format。
- 不能使用旧 Python 原型里的 synthetic ai-history.sessions fixture 冒充真实 Cursor latest 格式。
- 不能提交真实隐私历史内容。
- 不能 archive OpenSpec，直到 macOS 和 Windows latest Cursor 样本都验证完成。

你的任务：

1. 在当前环境中定位 Cursor latest 的真实本地数据目录。

macOS 常见路径：

~/Library/Application Support/Cursor/User

Windows 常见路径：

%APPDATA%\Cursor\User

2. 先探查目录结构和候选数据库/文件，不要立刻复制隐私内容进仓库。

重点查找：

- state.vscdb
- workspaceStorage
- globalStorage
- any chat / composer / ai / conversation related keys
- SQLite ItemTable 中可能包含 AI chat/session 数据的 key

3. 从真实样本中归纳最小 fixture。

要求：

- 放到 testdata/cursor/macos/ 或 testdata/cursor/windows/
- 只保留证明 list/show/context 能工作的最小数据
- 替换掉私人文本、路径、用户名、项目名
- 保持真实 storage shape、表结构、key 名称、JSON shape
- 不要提交完整原始 Cursor 数据目录

4. 按 TDD 实现 Cursor reader。

流程必须是：

- 先写 failing test
- 运行测试确认失败
- 再实现最小代码
- 再运行测试确认通过

建议测试覆盖：

- NewCursorStorageReader(...).ListSessions() 能列出真实 fixture 中的 session
- GetSession(nativeID) 能读取 turns
- summary 包含 id/source/native_id/title/cwd/project/preview/turn_count
- unsupported/unrecognized format 继续返回 unsupported_format
- doctor 在真实 fixture 上返回 available
- doctor 在无数据时返回 source_unavailable

5. 实现完成后运行：

export PATH="$PATH:/usr/local/go/bin"
GOCACHE=/tmp/go-build go test ./...
openspec validate bootstrap-ai-session-history-cli --strict
GOCACHE=/tmp/go-build go run ./cmd/ai-history doctor --json

6. 更新文档和 OpenSpec tasks：

- 如果完成 macOS fixture 和 reader，勾选 macOS Cursor 相关任务
- 如果完成 Windows fixture 和 reader，勾选 Windows Cursor 相关任务
- 如果只完成其中一个，不要把 P0 标记为全部完成
- README 中更新 Cursor 支持状态

7. 提交：

git add .
git commit -m "添加 Cursor 读取器"

注意：

- Commit message 用中文
- 不要添加 Co-Authored-By
- 不要 archive OpenSpec，除非 macOS 和 Windows latest Cursor 都已经用真实 fixture 验证通过
```
