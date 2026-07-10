# AI Session History

简体中文 | [English](README.md)

一个本地优先的命令行工具，用于查找 AI 编程会话，并生成干净的交接上下文。

`ai-history` 读取受支持工具保存在本机的会话历史。它不会上传你的数据，也不依赖托管服务。

## 功能

- 从当前项目或所有已配置来源发现会话。
- 以 JSON、干净文本或摘要形式查看会话。
- 生成确定性的 Markdown 上下文，方便把工作交接给另一个 Agent 或工作目录。
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

## 快速开始

检查当前可用的本地数据来源：

```bash
ai-history doctor --json
```

列出当前项目的会话：

```bash
ai-history list --here --limit 10 --json
```

查看某个会话：

```bash
ai-history show codex:<session-id> --mode clean
```

为另一个项目生成交接上下文：

```bash
ai-history context codex:<session-id> --target-cwd /path/to/project
```

## 命令

```bash
ai-history doctor
ai-history list
ai-history show <source>:<session-id>
ai-history context <source>:<session-id>
ai-history version
```

运行 `ai-history help` 或 `ai-history help <command>` 查看完整命令说明。

常用短选项：

```bash
ai-history doctor -j
ai-history list -s codex -l 10 -j
ai-history show codex:<session-id> -m summary -n 2000 -j
ai-history context codex:<session-id> -t /path/to/project -n 4000
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
