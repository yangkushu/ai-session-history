# CLI 验收反馈 - 2026-07-09

## 总览

本文档记录 P0 CLI 手工验收和文档审查过程中发现的问题、反馈和后续候选工作。
这些内容不属于手工验收流程本身，应作为后续 OpenSpec change 或 P1 backlog 的输入。

## 文档结构反馈

原 `2026-07-09-manual-acceptance-and-backlog.md` 同时包含手工验收流程、发现问题和后续待办，
职责不清。

已调整方向：

- 手工验收内容保留为独立验收指南。
- 反馈、问题和后续待办迁移到本文档。
- 原“手工验收测试用例”的表述改为“手工验收指南”，避免暗示它是可重复、可自动化的测试用例。

## CLI 帮助与短参数缺口

当前 CLI 缺少符合常见命令行习惯的帮助入口和短参数别名，需要在后续统一修补。

已观察到的问题：

- `ai-history help` 当前返回 `unknown command: help`，没有顶层 help 命令。
- `ai-history -h` 当前被当作未知命令处理，而不是显示顶层帮助。
- 子命令依赖 Go 标准库 `flag` 的默认 `-h` 行为，能显示 flag 列表，但退出码和整体
  usage 体验还不统一。
- 常用参数没有短别名，例如 `--json`、`--limit`、`--source`、`--mode`、
  `--max-chars`、`--target-cwd` 和 `--config`。

后续修补时建议统一设计：

- 顶层支持 `help`、`--help` 和 `-h`。
- 子命令支持 `help <command>`、`<command> --help` 和 `<command> -h`。
- 为常用参数补充稳定短别名，并在 README 与 CLI 规格中同步记录。

## CLI 版本信息缺口

当前 CLI 没有面向用户的版本号入口，也没有清晰的构建版本注入机制。用户无法通过命令确认
当前运行的是哪个版本、哪个提交或哪个构建产物。

已观察到的问题：

- 没有 `ai-history version` 命令。
- 没有 `ai-history --version` 或 `ai-history -v`。
- README 的本机编译说明没有说明版本信息如何产生。
- 当前仓库还没有 release tag 或二进制发布流程，因此版本语义需要和后续发布策略一起确定。

后续修补时建议统一设计：

- 支持 `ai-history version` 和 `ai-history --version`。
- 谨慎决定 `-v`：它常被用作 verbose，也可能被用户期待为 version；需要在 CLI 参数体系中统一约定。
- 版本输出至少包含语义版本；在本机构建或开发构建中可包含 commit、构建时间或 `dev` 标记。
- 若后续引入 GoReleaser/GitHub Releases，应通过 ldflags 注入版本信息，并在 README 中记录。

## 当前目录会话优先展示

当前 `list` 支持 `--cwd` 和 `--under`，但默认 `ai-history list` 会列出所有可用来源的会话。
人工使用时，用户往往是在某个项目目录中寻找“当前项目相关的历史 session”，全部列表可能噪声较大。

反馈问题：

- 是否应优先展示当前目录下的历史 session。
- 是否需要提供命令或参数在“当前目录范围”和“全部历史”之间切换。
- 默认行为需要设计：默认当前目录更贴近项目交接场景；默认全部更符合全局历史浏览和脚本兼容预期。

后续修补时建议统一设计：

- 明确 `ai-history list` 的默认范围：当前目录、当前目录子树，或全部历史。
- 若默认改为当前目录范围，应提供显式 `--all` 或类似参数查看全部历史。
- 若默认保持全部历史，应提供更顺手的当前目录快捷方式，例如 `--here`、`--under .` 简写或交互模式默认过滤。
- 需要评估对脚本兼容性的影响，并在 README 与 CLI 规格中同步记录。

## Tool 消息过滤与最终结果保留

当前工具消息处理边界不够清晰，需要确认是否把 tool 的全部消息都过滤掉，以及 tool 的最终返回是否应该保留。

已观察到的现状：

- 模型层定义了 `tool_call`、`tool_result` 和 `error` 等 turn kind。
- 渲染层在 `clean` 模式会省略 tool result 和非 error 的 tool turn，并用 omitted marker 标记。
- 渲染层在 `summary` 模式会把 tool result 替换为轻量 omitted marker；`raw` 模式保留原始文本。
- 当前 Codex、Claude Code、Cursor reader 主要只把 user/assistant/system message 转成标准化 turns。
  真实历史里的 tool call、tool result 或最终工具返回，大多还没有作为独立 turn 进入输出。

待确认问题：

- 是否应区分 tool call、tool streaming output、tool final result 和 tool error。
- `clean` 模式是否应完全过滤 tool 内容，还是保留 final result 的摘要。
- `summary` 模式是否应保留 command、exit status、错误摘要和最终返回的首尾片段。
- `context` 交接输出是否需要保留对继续工作有价值的 tool final result，例如测试结果、构建失败原因或文件路径。
- 各 source 的原始历史格式中，哪些字段可以可靠表示“最终返回”，需要分别验证。

后续修补时建议先用真实样本定义 tool turn 语义，再更新 reader、render 和 CLI 规格。

## 空列表 JSON 形状

当前 `list --json` 在没有匹配 session 时会输出 `"sessions": null`，这对脚本处理不够友好。
空列表更适合稳定输出为 `"sessions": []`。

已观察到的示例：

```json
{
  "sessions": null,
  "diagnostics": {
    "cursor": {
      "source": "cursor",
      "status": "unavailable",
      "code": "source_unavailable",
      "message": "no Cursor state.vscdb found"
    }
  },
  "unavailable_sources": {
    "cursor": "no Cursor state.vscdb found"
  },
  "total_returned": 0
}
```

已确认的根因：

- `ListResult.Sessions` 是 slice 字段。
- `Service.List` 初始化 `ListResult` 时没有把 `Sessions` 初始化为空 slice。
- 没有匹配会话时，Go JSON 编码会把 nil slice 输出为 `null`。

后续修补时建议：

- 将空结果稳定输出为 `"sessions": []`。
- 添加测试覆盖无匹配 session 的 JSON 形状。
- 明确 `diagnostics` 和 `unavailable_sources` 是否继续使用 `omitempty`，避免改变已有诊断语义。
- 顺带确认 `-under` 与 `--under` 的参数写法是否都应被支持并写入帮助文档。

## 后续待办想法：交互式 CLI 模式

在后续版本中添加可选的交互式工作流，同时保持命令式 CLI 作为主要稳定接口。

候选形态：

- `ai-history tui` 或 `ai-history interactive` 打开终端菜单。
- 菜单允许用户选择来源、按目录过滤、浏览会话、预览详情并生成上下文。
- 交互模式应调用与 `doctor`、`list`、`show` 和 `context` 相同的核心服务；
  不应引入单独的行为路径。
- 命令模式仍然是脚本、agent 交接和可预测自动化所必需的接口。

这应在 P1 命令改进之后考虑，例如 export 命令和 context 清理规则。
