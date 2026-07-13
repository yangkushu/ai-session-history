# AI Session History

简体中文 | [English](README.md)

一个本地优先的命令行工具，用于查找 AI 编程会话，并生成干净的交接上下文。

`ai-history` 读取受支持工具保存在本机的会话历史。它不会上传你的数据，也不依赖托管服务。

## 功能

- 从当前项目或所有已配置来源发现会话。
- 对本地会话标题和对话内容进行确定性排序搜索。
- 以 JSON、干净文本或摘要形式查看会话。
- 生成确定性的 Markdown 上下文，方便把工作交接给另一个 Agent 或工作目录。
- 将完整归一化会话导出为私有、可长期保存的 JSON 或 Markdown 文件。
- 读取 Codex、Claude Code 和 Cursor 的本地历史。

## 安装

从 [GitHub Releases](https://github.com/yangkushu/ai-session-history/releases)
下载预编译归档。发布制品包含 Linux、macOS 和 Windows 构建，以及校验和文件。

也可以从源码构建：

```bash
go build -o ai-history ./cmd/ai-history
```

直接运行构建出的二进制：

```bash
./ai-history doctor --json
```

或者把它放到 `PATH` 中的目录，例如 `~/bin`。

## Agent Skill 安装

请先安装 `ai-history` binary。可选的 Agent Skill 只负责告诉 Codex、Claude Code
和 Cursor 何时、如何调用这个 CLI，并不是另一套运行时。安装前请先审阅仓库源码和
[`skills/ai-history/SKILL.md`](skills/ai-history/SKILL.md)：

```bash
npx skills add yangkushu/ai-session-history \
  --skill ai-history --global \
  --agent codex --agent claude-code --agent cursor
```

通过 [skills CLI](https://github.com/vercel-labs/skills) 安装时，Node.js、`npx`
和网络只在安装 Skill 时需要；Go 编写的 `ai-history` CLI 运行时不依赖 Node.js。
installer 默认使用项目级作用域，上面的命令明确选择全局作用域。

PowerShell 等非 POSIX shell 可直接复制以下单行命令：

```text
npx skills add yangkushu/ai-session-history --skill ai-history --global --agent codex --agent claude-code --agent cursor
```

`npx` 命令会下载并执行所链接的第三方 `skills` package。运行前请在受控环境中
审阅该 CLI 及其源码；也可用 `npx skills@<reviewed-version> add ...` 固定已经审核
的版本。不愿运行第三方 installer 时，请使用下面的手动方式。

若需手动安装，把同一份 canonical `skills/ai-history/` 目录完整复制到当前 host 的
Skill 目录即可，不要维护第二个 source of truth：

| Host | 全局手动安装目标 |
| --- | --- |
| Codex | `$HOME/.agents/skills/ai-history` |
| Claude Code | `$HOME/.claude/skills/ai-history` |
| Cursor | `$HOME/.cursor/skills/ai-history` |

installer mapping 可能与这些 native manual target 不同。更新时应沿用同一种安装
方式，不要分别编辑 installed copies。安装后，Codex 使用 `$ai-history`；Claude
Code 和 Cursor 使用当前 host UI 显示的 slash 或 Skill invocation。Agent 应先
运行 `ai-history version` 和 `ai-history doctor --json` 做 preflight。

安装 Skill 不会授予 CLI 执行、历史读取或导出写入等运行时权限。安装 Skill
不会修改 `sandbox`、`allowlist` 或 `managed policy`。请按 Skill 的 host
reference 仅授予当前操作所需的最小权限。

## 快速开始

检查当前可用的本地数据来源：

```bash
ai-history doctor --json
```

列出当前项目的会话：

```bash
ai-history list --here --limit 10 --json
```

搜索当前项目的历史对话：

```bash
ai-history search "发布检查清单" --here --json
```

查看某个会话：

```bash
ai-history show codex:<session-id> --mode clean
```

为另一个项目生成交接上下文：

```bash
ai-history context codex:<session-id> --target-cwd /path/to/project
```

为脚本、Skill 或后续 MCP adapter 生成结构化交接 JSON：

```bash
ai-history context codex:<session-id> --target-cwd /path/to/project --json
```

将完整会话导出以便本地归档或传递：

```bash
ai-history export codex:<session-id> --output session-export.json
ai-history export codex:<session-id> --output session-export.md --format markdown --mode clean
```

导出可能包含敏感提示、路径、工具输入和工具输出。必须显式提供输出路径；新文件使用仅当前用户可读写的
`0600` 权限，替换已有文件必须显式指定 `--force`。默认 `raw` 模式会完整保留所有归一化 turn，且不受字符上限限制；
可选择 `clean` 或 `summary` 减少嘈杂的工具内容。

## 命令

```bash
ai-history doctor
ai-history list
ai-history search <关键词>
ai-history show <source>:<session-id>
ai-history context <source>:<session-id>
ai-history export <source>:<session-id> --output <路径>
ai-history version
```

运行 `ai-history help` 或 `ai-history help <command>` 查看完整命令说明。

`show --json` 返回归一化会话详情。`context --json` 返回用于继续工作的筛选后交接对象，
其中包含 `schema_version: "context-handoff.v1"`。`export` 会写入完整、版本化的
`session-export.v1` 文件；默认格式为 JSON，也可通过 `--format markdown` 导出 Markdown。

常用短选项：

```bash
ai-history doctor -j
ai-history list -s codex -l 10 -j
ai-history search "发布检查清单" -s codex -l 20 -j
ai-history show codex:<session-id> -m summary -n 2000 -j
ai-history context codex:<session-id> -t /path/to/project -n 4000
ai-history export codex:<session-id> -o session-export.json -m raw
```

## 支持的数据来源

- Codex 本地会话状态和 rollout JSONL 文件。
- Claude Code 项目 JSONL 历史。
- Cursor 在 macOS 和 Windows 上的本地存储，包括从 WSL 自动发现 Windows 数据。

存储细节和当前限制见 [Source support](docs/source-support.md)。

## 开发

运行测试：

```bash
go test ./...
```

本地运行：

```bash
go run ./cmd/ai-history doctor --json
```

使用指定配置：

```bash
go run ./cmd/ai-history doctor --json --config examples/config.yaml
```

维护者发布说明见 [Releasing](docs/releasing.md)。历史范围说明和原型参考见
[Project notes](docs/project-notes.md)。
