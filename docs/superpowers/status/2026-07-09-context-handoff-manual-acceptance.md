# Context Handoff 人工验收用例 - 2026-07-09

## 总览

本文档用于人工验收 `improve-context-handoff-quality` 的 Markdown handoff 输出质量。
它关注真实 session 的可读性和交接效果，不替代 renderer 自动化测试。

## 验收目标

确认 `ai-history context` 适合交给另一个 agent 继续工作：

- 输出仍以 `# AI Session Context` 开头。
- session metadata、initial goal、recent conversation、tool outcomes、handoff notes
  和 handoff instruction 按稳定顺序出现。
- 初始目标不会被 AGENTS/CLAUDE 注入说明、environment context 或空 turn 污染。
- concise tool final result 和 tool error 会保留。
- 大块 tool output 会省略，并在 handoff notes 中记录。
- tight `--max-chars` 下仍保留 metadata、initial goal 和 notes，并标记 truncated。

## 前置条件

- 仓库位于包含 `improve-context-handoff-quality` 实现的提交。
- 已完成本机编译：

  ```bash
  PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go build -o ai-history ./cmd/ai-history
  ```

- 本机至少存在一个可读取的真实 session。优先选择包含工具调用、测试输出或长日志的 session。

## 验收步骤

1. 找一个当前项目相关 session：

   ```bash
   ./ai-history list --here --limit 10
   ```

2. 生成常规 handoff：

   ```bash
   ./ai-history context <source:session-id> --target-cwd "$PWD" --max-chars 5000
   ```

3. 检查 section 顺序：

   ```text
   # AI Session Context
   ## Session
   ## Initial Goal
   ## Recent Conversation
   ## Tool Outcomes
   ## Handoff Notes
   ## Handoff Instruction
   ```

4. 检查初始目标：

   - 如果 session 开头是环境或 AGENTS/CLAUDE 注入说明，`Initial Goal` 不应显示这些注入内容。
   - 如果真实任务存在，`Initial Goal` 应显示第一个有意义的用户任务。
   - 如果没有有意义任务，应显示 `Unavailable`，而不是编造目标。

5. 检查 tool outcomes：

   - 测试通过、测试失败、构建失败、命令错误等 concise result/error 应出现在 `Tool Outcomes`。
   - 大块日志、diff 或终端输出不应完整出现在 `Tool Outcomes`。

6. 检查 handoff notes：

   - 跳过 setup boilerplate 时，应出现 skipped note。
   - 省略 noisy tool output 时，应出现 omitted note。
   - 截断时，应出现 truncated note 和 `[truncated]` marker。

7. 检查 tight budget：

   ```bash
   ./ai-history context <source:session-id> --target-cwd "$PWD" --max-chars 900
   ```

   输出应仍保留 `Session`、`Initial Goal`、`Handoff Notes`，并说明已截断。

## 验收判定

整体通过应满足：

- 输出可作为另一个 agent 的直接交接上下文。
- 初始目标不被本地环境注入内容污染。
- 最近对话保留真实推进信息，不展示明显 setup boilerplate。
- tool outcome 保留对继续工作有价值的结果和错误。
- noisy tool output 不大量污染 handoff。
- 截断输出仍能解释自己缺失了什么。

## 手工记录项

- 初始目标是否选中了真实任务。
- `Recent Conversation` 是否足够新、足够短。
- `Tool Outcomes` 是否保留了继续工作真正需要的结果。
- `Handoff Notes` 是否太吵或不够具体。
- 是否需要后续新增 `context --json` 或目标 agent profile。
