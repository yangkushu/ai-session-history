# ai-history CLI 与 Skill 一键安装设计

## 总览

本次变更为 `ai-history` 提供跨平台的一键安装入口，并重组中英文 README 的安装与
Agent Skill 说明。用户可以选择只安装 CLI binary，或用一个 bundle 命令同时安装
binary 与 Agent Skill；重新执行原命令即可升级，不在 CLI 中新增 `self update`。

安装器覆盖 macOS、Linux 和 Windows，以及 Codex、Claude Code 和 Cursor。binary
继续由现有 GitHub Releases 和 GoReleaser 制品提供，Skill 继续由 `npx skills`
管理。安装器只负责显式编排两者，不把 binary 安装隐藏在 npm lifecycle 或 Skill
hook 中。

## 目标与边界

### 目标

- 为 macOS、Linux 和 Windows 提供 binary-only 一键安装命令。
- 提供 binary + Skill 的 bundle 一键安装命令。
- 支持 `amd64` 与 `arm64` release 制品。
- bundle 模式支持 Codex、Claude Code 和 Cursor，并默认检测本机已有 Agent。
- 通过重新运行相同命令完成幂等安装和升级。
- 支持指定版本安装与显式回退。
- 下载 release archive 后校验 checksum，再原子替换 binary。
- 让 README 清楚说明 Skill 的安装方式、使用者、触发方式和更新方式。

### 不在本次范围

- 不新增 `ai-history self update`。
- 不发布 npm wrapper 或依赖 npm lifecycle 安装 native binary。
- 不在首版接入 Homebrew、Scoop、WinGet 等 package manager。
- 不修改 Agent sandbox、allowlist、managed policy 或历史目录权限。
- 不自动删除或接管由 Homebrew、Go 或手工方式安装的其他 binary。
- 不为安装或更新引入 telemetry。

## 安装入口

仓库新增两个 canonical installer：

```text
scripts/
├── install.sh
└── install.ps1
```

README 提供以下四类公开入口。默认 URL 指向仓库 `master` 分支中可直接审阅的
canonical script；指定版本安装时由脚本继续解析对应 release，而不是切换 installer
自身的 revision：

```bash
# macOS / Linux：只安装或更新 binary
curl -fsSL https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.sh | sh

# macOS / Linux：安装或更新 binary + Skill
curl -fsSL https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.sh | sh -s -- --with-skill
```

```powershell
# Windows：只安装或更新 binary
irm https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.ps1 | iex

# Windows：安装或更新 binary + Skill
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/yangkushu/ai-session-history/master/scripts/install.ps1))) -WithSkill
```

文档同时提供先下载、审阅、再本地执行的替代步骤。公开命令默认安装 latest release；
脚本参数和环境变量可选择明确版本、安装目录、Agent target 和 PATH 行为。

## Binary 安装与更新

### 平台识别与制品选择

安装器把运行平台映射到现有 GoReleaser archive：

| 平台 | Architecture | Archive |
| --- | --- | --- |
| macOS | `amd64` / `arm64` | `darwin_amd64` / `darwin_arm64` tar.gz |
| Linux | `amd64` / `arm64` | `linux_amd64` / `linux_arm64` tar.gz |
| Windows | `amd64` / `arm64` | `windows_amd64` / `windows_arm64` zip |

unsupported OS 或 architecture 在下载前失败，并显示支持列表。默认从 GitHub latest
release 解析目标 tag；显式版本参数接受规范化的 `vX.Y.Z`。为兼容
GoReleaser 已发布 artifact，二进制身份识别同时接受 `ai-history X.Y.Z` 和
`ai-history vX.Y.Z`，并在 installer 内统一归一化为 `vX.Y.Z`。release 不存在时
不修改本机状态。

### 安装目录与 PATH

- macOS / Linux 默认安装到 `${AI_HISTORY_INSTALL_DIR:-$HOME/.local/bin}`。
- Windows 默认安装到 `%LOCALAPPDATA%\ai-history\bin`。
- Unix 不使用 `sudo`。目标目录不在 `PATH` 时，安装器在当前 shell profile 中写入一条
  带稳定注释标记的配置；`--no-modify-path` 禁止该行为。
- Windows 只更新当前用户 `PATH`，不更新 system `PATH`；`-NoModifyPath` 禁止该行为。
- PATH 写入必须幂等，不能在重复安装时追加重复条目。

安装器无法改变当前父进程的环境，因此完成后应说明新 shell 生效方式，并用目标文件
的绝对路径执行本次 smoke check。

### 完整性、替换与冲突

安装器在临时目录下载 archive 和对应 release 的 `checksums.txt`，验证匹配后解压。
checksum 不匹配时拒绝安装。新 binary 先执行基本可执行性检查，再通过同一文件系统内
的 rename 原子替换目标；任何前置失败均保留旧版本。

若目标路径已有文件，安装器必须先确认它可识别为 `ai-history`。无法确认归属时拒绝
覆盖。若目标安装成功，但 `PATH` 中解析到另一份由 Homebrew、Go 或手工方式安装的
binary，则报告路径冲突，不删除或修改其他副本。

### 重跑与版本语义

- 目标版本尚未安装：执行全新安装。
- 目标路径已经是相同版本：binary 阶段为 no-op，仍执行验证；bundle 模式继续刷新
  Skill。
- latest release 高于当前版本：原子升级。
- 显式指定较旧版本：允许回退，并清楚输出当前版本与目标版本。
- 安装器只管理自己的目标路径，不实现 CLI 内部更新状态或 package manager 接管。

## Bundle 安装与 Skill 更新

`--with-skill` / `-WithSkill` 在 binary 安装和验证成功后运行 Skill 阶段。该阶段使用
公开的 `npx skills add` non-interactive 能力，并将 source 固定到与 binary 相同的
release tag，避免 Skill 与 CLI 版本漂移。source 使用 GitHub tree URL，例如目标版本为
`v0.4.0` 时使用
`https://github.com/yangkushu/ai-session-history/tree/v0.4.0/skills/ai-history`；安装器将
URL 中的版本替换为本次已解析的 release tag。

默认行为如下：

1. 检测本机已有的 Codex、Claude Code 和 Cursor。
2. 展示将安装 Skill 的 Agent target。
3. 用户显式传入 Agent 参数时，只处理指定 target。
4. 无法检测任何 Agent 时停止 Skill 阶段，并给出显式 Agent 参数示例；不猜测 target。
5. 以 global scope 安装 `ai-history` Skill，并使用 non-interactive confirmation。

重跑 bundle 命令时，binary 按版本规则安装或升级，Skill 从同一个目标 release tag
重新安装或刷新。binary-only 命令不修改已安装 Skill。

Node.js 或 `npx` 缺失时，binary 保持已安装状态，bundle 返回非零状态，并输出后续
补装 Skill 的准确命令。某一 Agent 安装失败时，报告逐 Agent 结果；不回滚成功的
binary 或其他 Agent，但 bundle 整体返回非零状态。

## 权限与安全模型

- `curl | sh` 和 `irm | iex` 会执行远程代码，README 必须明确提示并提供审阅方式。
- 安装器不请求管理员权限、不写系统目录，也不修改系统级 PATH。
- checksum 提供传输完整性检查；它与 archive 来自同一 release channel，不表述为
  独立的发布者身份认证。
- Skill 安装只写对应 Agent 的 global Skill 目录，不授予 CLI 执行、历史读取或 export
  写入权限。
- 安装后的 `doctor` 只检查本地数据来源，不上传数据；单个 source unavailable 不是
  安装失败。
- 安装器尊重标准 proxy 和 TLS 行为，不提供跳过 TLS 或 checksum 校验的选项。

## 失败处理

| 失败点 | 结果 |
| --- | --- |
| release/tag 不存在 | 不修改本机状态，返回非零 |
| unsupported platform | 不下载，显示支持范围，返回非零 |
| 下载中断或 checksum 失败 | 保留旧 binary，返回非零 |
| 目标文件归属不明 | 不覆盖，说明冲突路径，返回非零 |
| binary smoke check 失败 | 不进入 Skill 阶段，保留旧 binary |
| Node.js / `npx` 缺失 | 保留成功安装的 binary，Skill 未安装，返回非零 |
| 部分 Agent Skill 失败 | 保留成功项，报告逐项状态，返回非零 |
| `doctor` 部分 source unavailable | 安装成功，输出 warning |

安装完成后，脚本使用目标 binary 的绝对路径执行 `version` 和 `doctor --json`。
`version` 失败属于 binary 安装失败；`doctor` 只有 CLI 自身无法运行或返回结构性错误时
才判定安装失败。

## README 信息架构

中英文 README 保持相同结构和命令语义：

1. 项目简介后增加 `Install CLI`、`Install CLI + Skill`、`How the Skill works`、
   `Quick Start` 快速导航。
2. 安装区按“一键安装 binary”“一键安装 binary + Skill”“其他安装方式”“更新”与
   “安全说明”组织。
3. 单独解释用户与 Agent 的责任：用户安装并授权；Codex、Claude Code、Cursor 读取
   Skill 并选择 CLI 命令；实际历史能力由 binary 提供。
4. 为 Codex 提供 `$ai-history` 调用示例，为 Claude Code 和 Cursor 说明使用当前 UI
   提供的 Skill 或 slash invocation；同时保留直接调用 CLI 的方式。
5. 明确重跑 binary-only 命令只更新 binary，重跑 bundle 命令同时更新 binary 与
   Skill。

README 只保留快速路径和关键安全边界。完整参数、PATH 行为、版本选择、失败恢复和
手动安装说明放在 `docs/installation.md`。

## 测试与验收

### Installer 自动化测试

- shell script syntax 与静态检查。
- PowerShell script syntax 检查。
- 在隔离临时 HOME 下测试全新安装、相同版本重跑、升级、指定版本和回退。
- 覆盖 macOS/Linux/Windows 与 `amd64`/`arm64` 的制品名称映射。
- 使用本地 fixture 或 mock release endpoint 测试 checksum 错误、下载中断和 tag
  不存在，避免测试依赖实时 latest release。
- 验证原子替换、未知目标文件保护、PATH 幂等和 opt-out。
- mock `npx` 验证 Agent 自动检测、显式 target、无 `npx` 和部分失败语义。

### 集成与文档验收

- 在支持的 CI runner 上对已发布 release 执行 binary-only smoke test。
- bundle smoke test 使用隔离 HOME，不能污染真实 Agent 配置。
- 核对安装脚本解析的 archive 名称与 `.goreleaser.yaml` 一致。
- 核对 README 中英文的四条入口、更新规则、Skill 使用者说明与安全提示一致。
- 运行项目现有 Go tests，确认安装器与文档变更未影响 CLI 行为。

## 参考

- [uv 安装文档](https://docs.astral.sh/uv/getting-started/installation/)
- [uv installer options](https://docs.astral.sh/uv/configuration/installer/)
- [Vercel Skills CLI](https://github.com/vercel-labs/skills)
- [GitHub Releases](https://github.com/yangkushu/ai-session-history/releases)
