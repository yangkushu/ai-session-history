# AI Session History

简体中文 | [English](README.md)

一个本地优先的命令行工具，用于读取 AI 编程会话历史并生成交接上下文。

命令名称为 `ai-history`，产品名称为 AI Session History。

## P0 范围

当前已实现的 P0 命令：

```bash
ai-history doctor --json
ai-history list --here --json
ai-history show codex:<session-id> --mode clean --json
ai-history context codex:<session-id> --target-cwd /new/project
```

P0 有意不包含 `search`、完整的 `export`、完整的 `import` 或 MCP 服务。
`context` 是轻量级的 Markdown 交接导出，用于把先前的 AI 编程会话移交给另一个
Agent 或工作目录。

## 本地构建与安装

如有可用版本，请从 GitHub Releases 下载预编译二进制文件。每个版本都会发布适用于
Linux、macOS 和 Windows 的平台归档文件，以及 `checksums.txt`。

本地开发时，可以从仓库构建：

```bash
PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go build -o ai-history ./cmd/ai-history
```

直接运行构建出的二进制文件：

```bash
./ai-history doctor --json
```

将它安装到用户目录下的 bin 目录：

```bash
mkdir -p ~/bin
cp ai-history ~/bin/
```

确保 `~/bin` 已加入 `PATH`。如果使用 bash，可按需将下面这一行写入 `~/.bashrc`：

```bash
export PATH="$HOME/bin:$PATH"
```

然后验证已安装的命令：

```bash
ai-history doctor --json
```

## 发布版本

维护者通过推送版本标签发布二进制文件：

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions 会对匹配 `v*` 的标签运行测试和 GoReleaser。发布构建会注入版本元数据，
因此发布的二进制文件会报告标签、提交和构建日期：

```bash
ai-history version
```

推送标签前，可在本地校验发布配置：

```bash
goreleaser check
goreleaser release --snapshot --clean
```

快照构建会将制品写入 `dist/`，但不会发布 GitHub Release。

## 使用方法

显示顶层帮助：

```bash
ai-history help
ai-history --help
ai-history -h
```

显示子命令帮助：

```bash
ai-history help list
ai-history list --help
ai-history list -h
```

显示版本信息：

```bash
ai-history version
ai-history --version
```

本地开发构建在未注入版本信息时会显示 `dev`。发布流水线可在构建时通过 Go
`ldflags` 注入版本信息。

列出当前工作目录下的会话：

```bash
ai-history list --here --limit 10 --json
```

列出所有启用来源的会话：

```bash
ai-history list --limit 10 --json
```

常用短选项：

```bash
ai-history doctor -j
ai-history list -s codex -l 10 -j
ai-history show codex:<session-id> -m summary -n 2000 -j
ai-history context codex:<session-id> -t /new/project -n 4000
```

`context` 会输出确定性的 Markdown 交接文档。输出包含稳定的章节：会话元数据、初始
目标、最近对话、有用的工具结果、交接说明和继续执行的指令。它会在选择初始目标前过滤
已知的环境初始化样板内容，保留简洁的工具结果和错误，省略较大的原始工具输出，并标记
被跳过或截断的内容。

## 开发

运行测试：

```bash
PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go test ./...
```

构建 CLI：

```bash
PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go build ./cmd/ai-history
```

本地运行：

```bash
PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go run ./cmd/ai-history doctor --json
```

使用指定配置：

```bash
PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go run ./cmd/ai-history doctor --json --config examples/config.yaml
```

## 支持的数据来源

- Codex：读取 `state_5.sqlite` 和 rollout JSONL 文件。
- Claude Code：读取 `projects/**/*.jsonl`。
- Cursor：已支持 Windows 最新版，读取 `globalStorage/state.vscdb`（`composerHeaders`
  与 `cursorDiskKV`）及 `bubbleId:<composerId>:<bubbleId>` 记录；可从 WSL 主机
  自动发现 Windows Cursor 数据。macOS 最新版也已支持，基于已观察到的
  `cursorDiskKV` `composerData:<composerId>` 格式读取。数据库以 SQLite `immutable=1`
  方式打开，可安全读取 Cursor 正在使用的 WAL 模式文件。

## 参考原型

此前的 Python MCP 原型和 OpenSpec 说明仅作为行为参考，位于本仓库之外；不应将其视为
Go CLI 源码树的一部分。

当前设计说明见 `docs/2026-07-07-product-direction.md`。
