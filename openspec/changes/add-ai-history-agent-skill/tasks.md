## 1. 权限诊断增强

- [ ] 1.1 先为 Codex、Claude Code 和 Cursor reader 增加 history 路径权限失败的测试，覆盖 `permission_denied`、被拒绝路径以及其他 source 不受影响的诊断聚合。
- [ ] 1.2 在三个 reader 的 `Doctor` 路径检查中保留明确的 OS permission error，同时维持不存在路径、available 和 unsupported format 的既有行为。
- [ ] 1.3 运行 reader、core 和 CLI 定向测试，确认 `doctor --json` 输出稳定且正常 session 行为未改变。

## 2. 核心 Agent Skill

- [ ] 2.1 创建 `skills/ai-history/SKILL.md`，固化 CLI 预检、项目优先、JSON 优先、命令选择、单 source 降级和禁止调用 import 的规则。
- [ ] 2.2 创建 Codex、Claude Code、Cursor permission references，分别记录 scoped command、history read、export write 和 managed policy 处理，且不包含权限绕过方法。
- [ ] 2.3 在 Skill 中实现 export 安全策略：未指定 mode 时披露 raw 风险并推荐 clean，只有明确同意后使用 raw，且不自动 force 覆盖。
- [ ] 2.4 使用 Skill 校验器和代表性 agent 场景检查 frontmatter、按需引用与命令选择行为。

## 3. 跨平台安装

- [ ] 3.1 先增加临时 HOME/profile 安装测试，覆盖单 host、all、重复安装、内容冲突、显式 force 和 permission 配置不变。
- [ ] 3.2 实现 `scripts/install-ai-history-skill.sh`，复制同一核心 Skill 到 Codex、Claude Code 和 Cursor 用户级目录。
- [ ] 3.3 实现等价的 `scripts/install-ai-history-skill.ps1`，保持参数、冲突和退出码语义一致。
- [ ] 3.4 在可用本机环境和对应 OS CI 中运行安装测试，并明确记录无法执行的人工平台验收。

## 4. 文档与验收

- [ ] 4.1 更新中英文 README，提供简短的 Skill 安装、调用和最小权限入口，不复制完整 Skill 内容。
- [ ] 4.2 复核三端官方 Skill 目录及 permission 文档，把最终引用和已知 host 限制同步到 references。
- [ ] 4.3 运行 Skill 校验、安装测试、`gofmt`、`go test ./...`、`go vet ./...`、编译检查、`openspec validate add-ai-history-agent-skill --strict` 和 `git diff --check`。
- [ ] 4.4 执行规格符合性、Skill 前向场景和代码质量审查，修复发现的问题后归档 change。
