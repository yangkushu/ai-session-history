# P1 CLI 人工验收用例 - 2026-07-09

## 总览

本文档用于人工验收 P1 CLI 易用性与输出形状改进。它不是覆盖所有行为的自动化测试替代品，
而是面向真实本机环境的操作检查，用来确认帮助、版本、短参数、当前目录过滤和 JSON 空列表形状是否符合用户预期。

## 验收目标

确认 P1 改进在真实命令行中可用：

- CLI 帮助入口符合常见习惯。
- 版本命令可用于确认当前构建。
- 常用短参数可以替代长参数。
- `list --here` 能聚焦当前项目会话。
- 空列表 JSON 使用稳定的数组形状。
- 冲突参数给出清晰错误。

## 前置条件

- 仓库位于最新 `master`，且工作区干净。
- Go 可通过 `PATH` 访问，或测试者显式导出 Go 二进制路径。
- 测试者本机至少有一个 Codex、Claude Code 或 Cursor 历史来源；没有 Cursor 也可接受，但应记录诊断。

## 验收步骤

1. 构建本地二进制：

   ```bash
   PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go build -o ai-history ./cmd/ai-history
   ```

2. 检查顶层帮助：

   ```bash
   ./ai-history help
   ./ai-history --help
   ./ai-history -h
   ```

3. 检查子命令帮助：

   ```bash
   ./ai-history help list
   ./ai-history list --help
   ./ai-history list -h
   ```

4. 检查版本输出：

   ```bash
   ./ai-history version
   ./ai-history --version
   ```

5. 检查短参数：

   ```bash
   ./ai-history doctor -j
   ./ai-history list -s codex -l 10 -j
   ```

6. 检查当前目录过滤和全部列表：

   ```bash
   ./ai-history list --here --limit 10 --json
   ./ai-history list --limit 10 --json
   ```

7. 检查空列表 JSON 形状：

   ```bash
   ./ai-history list --under /definitely/missing --json
   ```

8. 检查冲突参数错误：

   ```bash
   ./ai-history list --here --under /tmp
   ```

9. 选择一个真实 session，检查详情与 context：

   ```bash
   ./ai-history show <source:session-id> --mode clean --max-chars 3000
   ./ai-history show <source:session-id> --mode summary --max-chars 3000
   ./ai-history context <source:session-id> --target-cwd "$PWD" --max-chars 5000
   ```

## 验收判定

整体通过应满足：

- `help`、`--help` 和 `-h` 都成功输出顶层帮助。
- `help <command>`、`<command> --help` 和 `<command> -h` 都成功输出子命令帮助。
- `version` 和 `--version` 成功输出版本信息；本机构建可显示 `ai-history dev`。
- `doctor -j`、`list -s ... -l ... -j` 等短参数行为与长参数一致。
- `list --here` 返回当前目录或其子目录下的会话。
- 普通 `list` 仍返回全部可用来源会话，不隐式套用当前目录过滤。
- 空结果 JSON 中 `sessions` 为 `[]`，不是 `null`。
- `--here` 与 `--cwd` / `--under` 同时使用时返回清晰 usage error。
- `show` 和 `context` 输出仍保持可读，且没有写入来源历史存储。

## 手工记录项

- help 文案是否足够清楚，是否缺少示例。
- `ai-history dev` 是否满足当前阶段的版本识别需求。
- `--here` 的默认体验是否符合项目交接场景。
- `diagnostics` / `unavailable_sources` 对脚本使用是否太吵。
- `context` 中的初始目标、最近对话和 tool outcome 是否适合交接给另一个 agent。

