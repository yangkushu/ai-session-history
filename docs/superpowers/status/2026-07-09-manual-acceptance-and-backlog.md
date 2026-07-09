# 手工验收与后续待办 - 2026-07-09

## 总览

本文档提供当前 P0 CLI 的手工验收用例，并记录一个后续交互式工作流的待办想法。
CLI 仍然是核心接口；任何交互模式都应作为增量能力补充。

## 手工验收用例：会话交接端到端可用

### 目标

验证用户可以找到之前的本地 AI 编码会话、查看会话详情，并在不修改源历史数据的前提下生成
Markdown 交接上下文。

### 前置条件

- 测试者本机至少有一个来自 Codex、Claude Code 或 Cursor 的本地会话。
- 仓库位于 `master` 分支，且工作区干净。
- Go 可通过 `PATH` 访问，或测试者已导出本地 Go 二进制路径。

### 步骤

1. 构建 CLI：

   ```bash
   GOCACHE=/tmp/go-build go build ./cmd/ai-history
   ```

2. 检查来源可用性：

   ```bash
   GOCACHE=/tmp/go-build go run ./cmd/ai-history doctor --json
   ```

3. 列出最近会话：

   ```bash
   GOCACHE=/tmp/go-build go run ./cmd/ai-history list --limit 10 --json
   ```

4. 选择一个返回的 `id` 并查看详情：

   ```bash
   GOCACHE=/tmp/go-build go run ./cmd/ai-history show <source:session-id> --mode clean --max-chars 2000 --json
   ```

5. 生成交接上下文：

   ```bash
   GOCACHE=/tmp/go-build go run ./cmd/ai-history context <source:session-id> --target-cwd <target-project-path> --max-chars 4000
   ```

6. 执行目录过滤：

   ```bash
   GOCACHE=/tmp/go-build go run ./cmd/ai-history list --under <project-root> --limit 10 --json
   ```

### 预期结果

- `doctor --json` 会独立报告每个来源。
- 不可用来源会作为诊断信息报告，并且不会阻止可用来源继续工作。
- `list` 返回稳定的、带来源前缀的 ID，例如 `codex:<id>`、`claude:<id>` 或
  `cursor:<id>`。
- `show` 返回所选会话的规范化摘要和对话轮次。
- `context` 返回以 `# AI Session Context` 开头的 Markdown。
- Markdown 包含会话元数据、原始 cwd、目标 cwd、初始目标、最近对话、省略内容说明和交接指令。
- 这些命令不会写入各来源拥有的历史存储。

### 手工记录项

- 测试环境中哪些来源可用。
- 会话标题、cwd、对话轮次数和预览内容是否合理。
- 所选 `Initial Goal` 是否有用，或是否被启动引导消息污染。
- 空结果是否以便于脚本处理的形状渲染。
- 是否存在来源特定的解析缺口，尤其是 Cursor 存储变体。

## 后续待办想法：交互式 CLI 模式

在后续版本中添加可选的交互式工作流，同时保持命令式 CLI 作为主要稳定接口。

候选形态：

- `ai-history tui` 或 `ai-history interactive` 打开终端菜单。
- 菜单允许用户选择来源、按目录过滤、浏览会话、预览详情并生成上下文。
- 交互模式应调用与 `doctor`、`list`、`show` 和 `context` 相同的核心服务；
  不应引入单独的行为路径。
- 命令模式仍然是脚本、agent 交接和可预测自动化所必需的接口。

这应在 P1 命令改进之后考虑，例如 export 命令和 context 清理规则。
