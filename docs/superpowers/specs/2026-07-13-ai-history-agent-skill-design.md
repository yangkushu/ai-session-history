# ai-history Agent Skill 设计

## 总览

本次开发为 `ai-history` 提供一份可移植的 Agent Skill，让 Codex、Claude Code
和 Cursor Agent 能按统一策略调用现有 CLI。Skill 只负责命令选择、JSON 输出使用、
权限预检和安全升级，不读取原始历史文件，也不复制 CLI 的 reader、search、render
或 export 逻辑。

仓库维护一份核心 Skill，通过安装器复制到三端各自支持的用户级 Skill 目录。安装器
不得修改任何 agent permission 配置；运行时权限仍由用户和对应 agent host 授予。

## 目标与边界

### 目标

- 覆盖 Codex、Claude Code 和 Cursor Agent 的本地调用场景。
- 让 agent 能在 `doctor`、`list`、`search`、`show`、`context` 和 `export`
  之间稳定选择。
- 默认限制在当前项目，避免无意读取无关项目历史。
- 对 CLI 执行、历史目录读取和 export 写入分别执行最小权限处理。
- 提供 Linux/macOS shell 安装器和 Windows PowerShell 安装器。

### 不在本次范围

- 不实现或调用 `import`。
- 不实现 MCP adapter、TUI 或新的 CLI 行为。
- 不自动修改 Codex、Claude Code 或 Cursor 的 permission 配置。
- 不自动启用 full access、跳过权限检查或放宽历史文件的 OS 权限。
- 不维护三份独立、可能漂移的 `SKILL.md`。

## 目录与分发

核心内容放在：

```text
skills/ai-history/
├── SKILL.md
├── references/
│   ├── codex-permissions.md
│   ├── claude-code-permissions.md
│   └── cursor-permissions.md
└── scripts/
    ├── install.sh
    └── install.ps1
```

`SKILL.md` 是唯一的调用策略来源。三个 permission reference 只记录各 host 的授权入口
和故障处理差异，不重复 CLI 工作流。安装器从同一核心目录复制文件，支持单个平台和
全部平台，且重复执行保持幂等。

默认用户级安装目标为：

- Codex：`$HOME/.agents/skills/ai-history`
- Claude Code：`$HOME/.claude/skills/ai-history`
- Cursor：`$HOME/.cursor/skills/ai-history`

实现前和发布前均需用当时的官方文档复核这些目录。Codex 同时支持仓库级
`.agents/skills`；Cursor 也支持仓库级 `.agents/skills` 和 `.cursor/skills`，但本次
安装器默认使用用户级目录，使 `ai-history` 可跨项目使用。仓库级手动安装只作为文档
说明，不由安装器猜测目标项目。

安装器不得静默覆盖内容不同的既有 Skill；默认报告冲突并退出，只有显式 `--force`
才替换。安装器只复制本 Skill 自身的文件，不写入 permission、sandbox、allowlist
或 managed policy 配置。

## Agent 调用流程

Agent 首次需要使用 `ai-history` 时执行以下预检：

```text
ai-history version
        │
        ▼
ai-history doctor --json
        │
        ├─ CLI 不可执行：提示安装或 PATH 问题后停止
        ├─ permission_denied：说明最小授权方法后停止
        ├─ 单个 source 不可用：报告该 source，继续使用其他可用 source
        ▼
在当前项目范围执行 list/search --here --json
```

命令选择规则：

- 查找历史讨论：`search <query> --here --json`。
- 浏览最近会话：`list --here --json`。
- 查看指定会话：`show <session-id> --mode clean --json`。
- 在当前目录继续旧工作：`context <session-id> --target-cwd <current-dir> --json`。
- 保存或传递完整会话：`export`，并遵守下述 export 安全规则。

当前项目没有结果时，Skill 不自动扩大到全部历史；应先询问用户是否允许全局搜索。
机器消费优先使用 JSON，Markdown 主要用于人工阅读或明确要求的交接文件。

## Export 安全规则

`export` 默认的 CLI mode 是 `raw`，但 Skill 不得在用户未明确选择时直接执行 raw
export。用户只说“导出”时，Skill 应说明 raw 可能包含提示词、路径、工具输入输出、
凭据或其他敏感内容，并推荐 `clean`。

- 用户明确要求完整、原始或 raw 内容后，才执行 `--mode raw`。
- 未明确要求 raw 时，获得用户同意后使用 `--mode clean`。
- 默认建议把文件写到当前 workspace 内的显式路径。
- 写到 workspace 外时，只请求目标路径所需的写权限。
- 不自动使用 `--force` 覆盖已有文件。

## 权限模型

权限分为三个相互独立的边界：

1. **CLI 执行**：允许调用具体的 `ai-history` 命令，不扩大为整个 shell 的长期授权。
2. **历史读取**：只为实际使用的 Codex、Claude Code 或 Cursor 历史目录增加读权限。
3. **Export 写入**：只为用户指定的目标路径增加写权限。

Skill 自身不能绕过 host sandbox。权限失败后只进行一次有信息增量的诊断，不以相同
参数重复执行失败命令。需要授权时，Skill 应指出被拒绝的命令或路径、所需访问类型
以及授权原因，然后等待用户决定。

平台处理如下：

- Codex：使用 scoped approval、`/permissions` 或受支持的 filesystem permission
  profile；不得自动切换到 `danger-full-access`。
- Claude Code：使用 `/permissions` 管理命令授权，并通过 `--add-dir` 或
  `additionalDirectories` 增加必要目录；不得建议全局允许全部 `Bash`。
- Cursor：使用 `Approvals & Execution`、sandbox directory controls 或 Cursor CLI
  的 `Shell(...)` / `Read(...)` 规则；不得自动扩大 allowlist。

以下方法明确禁止：`sudo`、`chmod 777`、复制全部历史到项目目录、
`bypassPermissions`、`dangerously-skip-permissions`，以及任何未经用户确认的 full
access。若 managed policy 阻止访问，Skill 报告该限制，不尝试规避。

## 错误处理

Skill 应区分以下错误，而不是统一归类为“CLI 不工作”：

- command not found 或 executable denied：安装、PATH 或 shell execution 问题。
- `permission_denied`：sandbox 或 OS 文件读取权限问题。
- `source_unavailable`：对应工具没有历史、路径未发现或当前平台不支持该来源。
- `unsupported_format`：历史格式不受当前 reader 支持。
- export destination exists：保留已有文件并询问是否更换路径；不自动 `--force`。

单个 source 失败不应阻止使用其他可用 source。Skill 不解析或依赖人类可读错误文本来
替代已有 JSON error code；只有 CLI 无法启动时才根据 shell 错误分类。

## 验收标准

### Skill 与安装器

- `SKILL.md` 通过 frontmatter、命名和目录结构校验。
- shell 与 PowerShell 安装器均支持单个平台、全部平台、幂等安装和显式 force。
- 使用临时 HOME/profile 验证安装，不污染真实用户目录。
- 安装测试证明脚本不会创建或修改任何 permission 配置文件。
- README 只提供简短安装入口，详细工作流保留在 Skill 和按需加载的 reference 中。

### 行为场景

- 当前项目的 list、search、show 和 context 能产生正确命令。
- 当前项目无结果时不会自动执行全局搜索。
- 单个 source unavailable 时仍可使用其他来源。
- CLI 不在 PATH、shell execution 被拒绝及 history read 被拒绝时，能给出不同处理。
- 普通 export 推荐 clean；raw export 必须由用户明确授权。
- export 不自动覆盖已有文件，且默认建议写入当前 workspace。

### 三端兼容

- 能运行的平台执行真实安装和调用 smoke test。
- 本机缺少的平台通过临时目录安装测试和对应 OS 的 CI 验证，并记录未执行的人工验收。
- 发布前复核 Codex、Claude Code 和 Cursor 的官方 Skill 目录及 permission 文档。

## 参考资料

- [Codex Build skills](https://learn.chatgpt.com/docs/build-skills#where-to-save-skills)
- [Codex sandbox and approvals](https://learn.chatgpt.com/docs/agent-approvals-security#sandbox-and-approvals)
- [Claude Code CLI reference](https://docs.anthropic.com/en/docs/claude-code/cli-usage)
- [Cursor Agent permissions](https://docs.cursor.com/cli/reference/permissions)
- [Cursor local agent sandbox](https://cursor.com/blog/agent-sandboxing)

