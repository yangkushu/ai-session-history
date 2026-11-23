# 下一版：宿主与运行时查证

这是[整体设计](next-version-design.md)的可行性记录，不是兼容性承诺。宿主 API 查证来自上游文档和本机安装包；实测只使用仓库合成样例与临时 SQLite 库，未读取真实会话，也未安装或运行新版插件。

## 宿主入口与交接

| 入口 | 已查证 | 设计建议与待验收 |
| --- | --- | --- |
| DSH 交互端 | [人类命令 API](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/subsystems/commands.md)提供 `ctx.commands.register`，处理过程不发起模型轮次。上游[命令实现](https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/interaction/commands/src/index.ts)和本机 DSH `0.2.1-rc.5` 均把命令返回的 `text` 写入 `command/done`，而 `recordInput: false` 只影响命令输入记录。 | 用户命令可作为操作入口，但不能把完整草稿放在命令的 `text` 里当不落盘预览。优先验证宿主能否提供既不进模型、也不写当前会话的预览；不行则用独立 TS CLI 供用户预览和手动带入。无交互界面的配置不通过模型工具返回交接正文。 |
| Pi | 本机 Pi `1.6.0` 的扩展文档提供 `pi.registerCommand`、`ctx.ui.custom` 和显式提交消息的 API。该安装的 `package.json` 声明 Node `>=22.19.0`。 | `/history` 在 TUI 内预览，确认后才提交；检查 `ctx.mode`，无 TUI 时不走预览确认流程。验收要检查取消时既不调用模型，也不新增包含草稿的会话消息。 |
| Cursor | [官方插件文档](https://cursor.com/docs/plugins#the-agent-plugins-standard)明确支持根级 `plugin.json` 的 Agent Plugin 以及内含的 Skill。 | 用 Agent Plugin 打包共用 Skill，Skill 通过终端调用独立 TS CLI。不假定插件会一并安装或提供 Node；缺少 CLI 或运行时时给出可操作的提示。尚未在 Cursor 中试装。 |

DSH 的 `ctx.approval` 是[已经打开的模型轮次中的操作审批](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/subsystems/approval.md)，不是正文的用户预览通道。DSH 的搜索工具如返回标题和片段，须明确这些内容会进入模型上下文，和用户专看的交接预览分开验收。

## Node.js 与数据读取

| 项目 | 已查证 | 对设计的影响 |
| --- | --- | --- |
| DSH npm CLI | [上游升级说明](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/upgrade-guide/v0.2.1-rc.2/node-24/guide.md)说明从 `0.2.1-rc.2` 起 Node 最低为 `24.9`。本机 DSH `0.2.1-rc.5` 通过 PATH 上的 Node `24.16.0` 启动。 | npm CLI 可基于 Node 24 的能力设计；不把这个下限套到旧 DSH 版本或 Desktop。 |
| DSH Desktop | [Desktop 技术文档](https://github.com/deepseek-ai/deepseek-harness/blob/master/apps/desktop/README.md)说明 Host 由 Electron RunAsNode 子进程启动，和 npm CLI 不是同一个运行时。 | 必须在目标 Desktop 安装包的插件进程里探测 `node:sqlite` 和 zstd，并验证用户预览与确认路径；本机未验。 |
| DSH zstd | [Node 文档](https://nodejs.org/download/release/v22.19.0/docs/api/zlib.html#zlibcreatezstddecompressoptions)说明流式解压 API 从 Node `22.15.0` 起提供，仍标记实验性。本机 Node `24.16.0` 用 `createZstdDecompress()` 逐行解析了仓库 DSH v4 合成压缩样例的全部 18 条记录。 | 优先使用内置流式解压，不因 zstd 预先引入原生 npm 依赖。实测证明解压可行，不代表验证了 reader 的事件映射、坏文件处理或目标宿主。 |
| Cursor SQLite/WAL | 本机 Node `24.16.0`、[内置 SQLite](https://nodejs.org/download/release/v22.19.0/docs/api/sqlite.html)的 `readOnly: true` 读到了临时库 WAL 中已提交但未 checkpoint 的记录，也看到了写端随后提交的记录；写 SQL 被拒绝，读取前后主库和 WAL 哈希一致。但在一个可写目录里读取无 sidecar 的 WAL 模式库时，SQLite 创建了 `-wal` 和 `-shm`。这符合 [SQLite 的 WAL 说明](https://sqlite.org/wal.html#read_only_databases)。 | 不把 `readOnly: true` 当成绝不写原目录的保证，也不照搬旧 Go 对活动库使用 `immutable=1` 的做法；[SQLite 只建议对确实不会变化的数据库使用该标志](https://sqlite.org/uri.html#uriimmutable)。设计评审前应确定安全读取策略，验证活跃库、新提交可见、sidecar 缺失或不可访问时的行为；无法安全读取时报告部分失败。 |

### 核心批次补证：不改写 Cursor 源文件

- 对已存在 WAL/SHM 的合成活动库做系统调用跟踪：`readOnly: true` 以 `O_RDONLY` 打开主库，却以 `O_RDWR|O_CREAT` 打开 `-wal` 和 `-shm`；读完观察到 `-shm` 内容改变。因此不只是缺文件时可能新建 sidecar。
- 在写端于复制期间不操作的合成库上，把主库和对应 WAL 复制到私有临时目录，不复制 SHM。Node 从副本读到了仅在 WAL 中的已提交记录；源目录的文件名和 SHA-256 均不变，新的 SHM 只出现在临时目录。这证明一致副本的可读性，不证明复制仍在写入的原库可以得到一致副本。

当前证据支持将 Node 内置功能作为首选，并用 Node 24 作为核心开发与首轮验收基线。独立 CLI、Pi 的发行最低版本，以及 DSH Desktop 目标版本留给入口设计，待对应运行时实测后公布。
