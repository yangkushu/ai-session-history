# P0 CLI 手工验收指南 - 2026-07-09

## 总览

本文档用于指导当前 P0 CLI 的人工验收，重点确认用户能否在本机找到历史 AI 编码会话、
查看会话详情，并生成 Markdown 交接上下文。

这不是严格意义上的“测试用例”。它依赖测试者本机真实历史数据，输入不可完全固定，因此更适合作为
手工验收流程和记录模板。可重复、可自动化的测试用例应使用固定 fixture 或受控输入另行设计。

## 验收目标

确认用户可以完成端到端会话交接流程：

- 诊断 Codex、Claude Code、Cursor 来源是否可用。
- 列出最近本地 AI 编码会话。
- 选择一个会话并查看规范化详情。
- 基于该会话生成 Markdown 交接上下文。
- 在整个过程中不修改各来源拥有的历史存储。

## 前置条件

- 测试者本机至少有一个来自 Codex、Claude Code 或 Cursor 的本地会话。
- 仓库位于 `master` 分支，且工作区干净。
- Go 可通过 `PATH` 访问，或测试者已导出本地 Go 二进制路径。

## 验收流程

1. 构建 CLI：

   ```bash
   GOCACHE=/tmp/go-build go build -o ai-history ./cmd/ai-history
   ```

2. 检查帮助和版本信息：

   ```bash
   ./ai-history help
   ./ai-history list --help
   ./ai-history version
   ```

3. 检查来源可用性：

   ```bash
   ./ai-history doctor --json
   ```

4. 列出当前项目下的最近会话：

   ```bash
   ./ai-history list --here --limit 10 --json
   ```

5. 列出所有来源的最近会话：

   ```bash
   ./ai-history list --limit 10 --json
   ```

6. 选择一个返回的 `id` 并查看详情：

   ```bash
   ./ai-history show <source:session-id> --mode clean --max-chars 2000 --json
   ./ai-history show <source:session-id> -m summary -n 2000 -j
   ```

7. 生成交接上下文：

   ```bash
   ./ai-history context <source:session-id> --target-cwd <target-project-path> --max-chars 4000
   ```

## 验收判定

整体通过应满足：

- `doctor --json` 独立报告每个来源。
- 不可用来源会作为诊断信息报告，并且不会阻止可用来源继续工作。
- `help`、子命令 `--help` 和 `version` 能成功输出信息。
- `list` 返回稳定的、带来源前缀的 ID，例如 `codex:<id>`、`claude:<id>` 或
  `cursor:<id>`。
- `list --here` 只返回当前目录或子目录下的会话；普通 `list` 仍返回所有可用来源会话。
- 空列表 JSON 使用 `"sessions": []`，而不是 `"sessions": null`。
- `show` 返回所选会话的规范化摘要和对话轮次。
- `context` 返回以 `# AI Session Context` 开头的 Markdown。
- Markdown 包含会话元数据、原始 cwd、目标 cwd、初始目标、最近对话、省略内容说明和交接指令。
- 这些命令不会写入各来源拥有的历史存储。

## 手工记录项

验收时记录以下信息，作为后续 P1 设计输入：

- 测试环境中哪些来源可用。
- 会话标题、cwd、对话轮次数和预览内容是否合理。
- 所选 `Initial Goal` 是否有用，或是否被启动引导消息污染。
- 空结果是否以便于脚本处理的形状渲染。
- 是否存在来源特定的解析缺口，尤其是 Cursor 存储变体。
- 验收过程中发现的产品体验、文档或 CLI 易用性问题。
