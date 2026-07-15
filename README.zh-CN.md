# AI Session History

简体中文 | [English](README.md)

一个本地优先的命令行工具，用于查找 AI 编程会话，并生成干净的交接上下文。

`ai-history` 读取受支持工具保存在本机的会话历史。它不会上传你的数据，也不依赖托管服务。

[CLI 安装](#安装) · [组合安装](#安装-binary-和-skill) ·
[Skill 角色](#谁使用-skill) · [快速开始](#快速开始)

## 功能

- 从当前项目或所有已配置来源发现会话。
- 对本地会话标题和对话内容进行确定性排序搜索。
- 以 JSON、干净文本或摘要形式查看会话。
- 生成确定性的 Markdown 上下文，方便把工作交接给另一个 Agent 或工作目录。
- 将完整归一化会话导出为私有、可长期保存的 JSON 或 Markdown 文件。
- 读取 Codex、Claude Code 和 Cursor 的本地历史。

## 安装

在 Linux 或 macOS 上只安装 CLI：

```sh
curl -fsSL https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.sh | sh
```

在 PowerShell 中只安装 CLI：

```powershell
irm https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.ps1 | iex
```

版本固定、自定义安装目录、PATH、更新、验证和卸载说明见
[安装指南](docs/installation.md)。

## 安装 binary 和 Skill

添加 `--with-skill` 可同时安装 CLI 和可选的 Agent Skill。installer 可以自动
检测受支持的 host，也可以显式选择 targets。

Linux 或 macOS：

```sh
curl -fsSL https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.sh | sh -s -- --with-skill
```

PowerShell：

```powershell
$script = irm https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.ps1
& ([scriptblock]::Create($script)) -WithSkill
```

完整选项、显式 target 示例、安全边界和故障恢复见
[安装指南](docs/installation.md)。

## 谁使用 Skill

你负责选择 Skill targets，并授权运行时访问。Codex、Claude Code 或 Cursor
读取已安装的 Skill，再选择合适的 CLI commands。Codex 可这样调用：

```text
$ai-history 查找这个项目之前关于发布流程的讨论。
```

Claude Code 和 Cursor 请使用当前 host UI 显示的 Skill invocation。

`ai-history` binary 负责 discovery、search、show、context 和 export。安装 Skill
不会授予读取历史、执行 CLI 或写入 export 的权限；这些权限仍由当前 host 控制。
不安装 Skill 时，仍然完整支持直接使用 CLI。

Skill contract 见 [`skills/ai-history/SKILL.md`](skills/ai-history/SKILL.md)，
安装细节见 [installation.md](docs/installation.md)。

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
