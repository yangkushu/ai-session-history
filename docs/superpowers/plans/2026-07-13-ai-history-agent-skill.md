# ai-history Agent Skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 Codex、Claude Code 和 Cursor 提供一份安全调用 `ai-history` CLI 的标准 Agent Skill，并让 `doctor --json` 对明确的 history 路径权限错误返回可操作诊断。

**Architecture:** `skills/ai-history` 只保存一份跨 Agent 工作流，host 权限差异拆到三个按需 reference；安装交给开放的 `npx skills` CLI，手动复制只作为 fallback。Reader 在自己的路径发现边界保留 OS permission error，通过既有 `SourceDiagnostic` 暴露给 Skill，不改变 session、handoff 或 export 数据协议。

**Tech Stack:** Go 1.22、标准库 `os`/`io/fs`/`errors`、Markdown Agent Skills、OpenSpec、Vercel `npx skills` CLI。

---

## 文件结构

新增：

- `skills/ai-history/SKILL.md`：唯一 CLI 调用与安全策略入口。
- `skills/ai-history/references/{codex,claude-code,cursor}-permissions.md`：host 权限差异。
- `skill_content_test.go`：Skill 安全契约和 README 安装入口回归测试。

修改：

- `internal/readers/reader.go`：共享 stat/read-dir 注入与错误转换。
- `internal/readers/{codex,claude,cursor}.go` 及测试：权限诊断。
- `internal/core/service_test.go`：多 source 聚合。
- `README.md`、`README.zh-CN.md`：`npx skills` 安装与权限边界。
- `openspec/changes/add-ai-history-agent-skill/tasks.md`：任务状态。

不新增项目 installer、Node.js package 或 CLI 运行时依赖。

### Task 1: 先建立失败的 reader 权限测试

**Files:**
- Modify: `internal/readers/codex_test.go`
- Modify: `internal/readers/claude_test.go`
- Modify: `internal/readers/cursor_test.go`
- Modify: `internal/readers/reader.go`

- [ ] **Step 1: 给三个 reader 写可移植的权限失败测试**

使用真实 `*os.PathError`，避免依赖当前用户或 ACL：

```go
func permissionError(path string) error {
	return &os.PathError{Op: "stat", Path: path, Err: fs.ErrPermission}
}

func TestCodexStorageReaderDoctorReportsPermissionDenied(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "state_5.sqlite")
	reader := NewCodexStorageReader([]string{root})
	reader.stat = func(got string) (os.FileInfo, error) {
		if got != path {
			t.Fatalf("stat path = %q, want %q", got, path)
		}
		return nil, permissionError(path)
	}
	diagnostic := reader.Doctor()
	if diagnostic.Code != core.ErrPermissionDenied || diagnostic.Path != path {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
}
```

Claude 用例注入 `reader.readDir`，拒绝 `<root>/projects`；Cursor 用例注入
`reader.stat`，拒绝 `<root>/globalStorage/state.vscdb`。补入 `io/fs`、`os` 和
`internal/core` import。

- [ ] **Step 2: 运行测试确认先失败**

Run:

```bash
go test ./internal/readers -run 'Test(Codex|Claude|Cursor)StorageReaderDoctorReportsPermissionDenied' -count=1
```

Expected: FAIL，提示 reader 没有 `stat` / `readDir` 字段。

- [ ] **Step 3: 添加共享错误分类 helper**

在 `reader.go` import `errors`、`os`，添加：

```go
type statFunc func(string) (os.FileInfo, error)
type readDirFunc func(string) ([]os.DirEntry, error)

func diagnosticFromError(source core.Source, err error) core.SourceDiagnostic {
	diagnostic := core.SourceDiagnostic{
		Source: source, Status: "unavailable",
		Code: core.ErrSourceUnavailable, Message: err.Error(),
	}
	var appErr *core.AppError
	if errors.As(err, &appErr) {
		diagnostic.Code = appErr.Code
		diagnostic.Path = appErr.Path
	}
	return diagnostic
}

func pathInspectionError(source core.Source, path string, err error) error {
	code := core.ErrSourceUnavailable
	if os.IsPermission(err) {
		code = core.ErrPermissionDenied
	}
	return core.WrapSourceError(code, source, path, err)
}
```

Run `gofmt -w internal/readers/reader.go internal/readers/*_test.go` 和
`git diff --check`。Expected: 格式检查通过；定向测试仍因 reader 未接线而失败。

### Task 2: 实现三个 reader 的权限诊断

**Files:**
- Modify: `internal/readers/codex.go`
- Modify: `internal/readers/claude.go`
- Modify: `internal/readers/cursor.go`

- [ ] **Step 1: Codex 使用可注入 stat**

```go
type CodexStorageReader struct {
	roots []string
	stat  statFunc
}

func NewCodexStorageReader(roots []string) *CodexStorageReader {
	return &CodexStorageReader{roots: roots, stat: os.Stat}
}
```

`Doctor` 对 `err == nil` 返回 available，对 `os.IsNotExist(err)` 继续下一个 root，
其他错误返回
`diagnosticFromError(core.SourceCodex, pathInspectionError(...))`。`threadRows` 同样把
`os.Stat` 换成 `r.stat`，确保 doctor/list/search code 一致。

- [ ] **Step 2: Claude 改为可诊断目录遍历**

```go
type ClaudeStorageReader struct {
	roots   []string
	readDir readDirFunc
}

func NewClaudeStorageReader(roots []string) *ClaudeStorageReader {
	return &ClaudeStorageReader{roots: roots, readDir: os.ReadDir}
}
```

把 `sessionFiles` 改成：读取每个 `<root>/projects`；不存在则继续，其他错误通过
`pathInspectionError` 返回；读取每个 project 子目录并收集非目录的 `.jsonl` 文件；
子目录读取失败同样返回带 project path 的错误；最后 `sort.Strings(files)`。Doctor
error 分支改用 `diagnosticFromError(core.SourceClaude, err)`。

- [ ] **Step 3: Cursor 路径发现返回 error**

```go
type CursorStorageReader struct {
	roots []string
	stat  statFunc
}

func (r *CursorStorageReader) stateDBPath() (string, error) {
	for _, root := range r.roots {
		candidate := filepath.Join(root, "globalStorage", "state.vscdb")
		_, err := r.stat(candidate)
		switch {
		case err == nil:
			return candidate, nil
		case os.IsNotExist(err):
			continue
		default:
			return "", pathInspectionError(core.SourceCursor, candidate, err)
		}
	}
	return "", nil
}
```

构造时设 `stat: os.Stat`。Doctor 处理 `(dbPath, err)`，error 时用
`diagnosticFromError`；`openDB` 传播 error，空路径仍返回 `source_unavailable`。

- [ ] **Step 4: 运行 reader 回归**

```bash
gofmt -w internal/readers/*.go
go test ./internal/readers -count=1
```

Expected: PASS，包括新增 permission tests 和全部既有 fixture tests。

- [ ] **Step 5: 提交**

```bash
git add internal/readers
git commit -m "fix: 完善历史路径权限诊断"
```

### Task 3: 验证多 source 聚合

**Files:**
- Modify: `internal/core/service_test.go`

- [ ] **Step 1: 添加聚合测试**

给已有 `fakeReader` 增加 `diagnostic SourceDiagnostic` 字段，并让 `Doctor()` 在该字段
的 `Source` 非空时返回它，否则保持原有 Codex available 默认值：

```go
func (f fakeReader) Doctor() SourceDiagnostic {
	if f.diagnostic.Source != "" {
		return f.diagnostic
	}
	return SourceDiagnostic{Source: SourceCodex, Status: "available"}
}
```

随后构造 Codex/Cursor available、Claude `ErrPermissionDenied + Path`，调用
`service.Doctor()`，断言长度为 3 且 Claude 的 code/path 未丢失：

```go
if len(diagnostics) != 3 ||
	diagnostics[1].Code != ErrPermissionDenied ||
	diagnostics[1].Path != deniedPath {
	t.Fatalf("diagnostics = %+v", diagnostics)
}
```

- [ ] **Step 2: 运行定向测试**

```bash
gofmt -w internal/core/service_test.go
go test ./internal/core ./internal/cli -count=1
```

Expected: PASS；既有 CLI test 继续覆盖 `doctor --json` 序列化。

- [ ] **Step 3: 提交**

```bash
git add internal/core/service_test.go
git commit -m "test: 覆盖多来源权限诊断"
```

### Task 4: 创建核心 Agent Skill

**Files:**
- Create: `skills/ai-history/SKILL.md`
- Create: `skills/ai-history/references/codex-permissions.md`
- Create: `skills/ai-history/references/claude-code-permissions.md`
- Create: `skills/ai-history/references/cursor-permissions.md`
- Create: `skill_content_test.go`

- [ ] **Step 1: 使用官方 initializer**

```bash
skill_creator_dir="${CODEX_HOME:-$HOME/.codex}/skills/.system/skill-creator"
python3 "$skill_creator_dir/scripts/init_skill.py" ai-history --path skills --resources references --interface 'display_name=AI History' --interface 'short_description=Find and hand off local AI coding sessions' --interface 'default_prompt=Use $ai-history to find a relevant local coding session and prepare safe handoff context.'
```

Expected: 创建 Skill 模板。为保持批准的跨 Agent 最小格式，用 `apply_patch` 删除生成的
`agents/openai.yaml`；不使用 shell 删除命令。

- [ ] **Step 2: 先创建失败的内容契约测试**

`skill_content_test.go` 读取 Skill 和 references，至少断言包含：

```go
required := []string{
	"name: ai-history",
	"ai-history version",
	"ai-history doctor --json",
	"search --here --json",
	"list --here --json",
	"--mode clean",
	"--mode raw",
	"references/codex-permissions.md",
	"references/claude-code-permissions.md",
	"references/cursor-permissions.md",
}
```

并断言核心 Skill 不包含 `chmod 777`、`dangerously-skip-permissions`、
`bypassPermissions`。额外断言当前项目无结果先询问、raw 必须明确请求、不得自动
force、不得调用 import。

Run: `go test . -run TestAIHistorySkillSafetyContract -count=1`

Expected: FAIL，模板缺少工作流内容。

- [ ] **Step 3: 写入完整 SKILL.md**

Frontmatter：

```yaml
---
name: ai-history
description: Use when an agent needs to find, search, inspect, hand off, or export local Codex, Claude Code, or Cursor coding sessions through the ai-history CLI.
---
```

正文依次包含：

1. CLI 是 capability boundary，不直接读 native history。
2. 首次运行 `ai-history version`、`ai-history doctor --json`。
3. list/search 默认 `--here --json`；无结果先问再扩大。
4. show 用 clean JSON，context 用 `--target-cwd <current-dir> --json`。
5. 禁止 import。
6. 未指定 export mode 时披露 raw 风险并推荐 clean；明确请求才 raw；不自动 force。
7. 区分 command execution、history read、export write；相同 denial 不重试。
8. 只加载当前 host 对应 reference。

- [ ] **Step 4: 写三份 host permission reference**

每份使用 `Command execution`、`History read`、`Export write`、`Stop conditions`
四节：

- Codex：scoped approval、`/permissions`、filesystem permission profile；禁止
  `danger-full-access`。
- Claude Code：`/permissions`、`--add-dir` / `additionalDirectories`、只允许
  具体 `ai-history` Bash pattern；禁止允许整个 Bash。
- Cursor：`Approvals & Execution`、sandbox directory controls、
  `Shell(ai-history)` / `Read(path)`；禁止扩大整个 allowlist。

三份都要求读取诊断 path、export 只写目标、managed policy 阻止时停止、不得使用
`sudo` 或修改 source history 权限。

- [ ] **Step 5: 校验并提交**

```bash
skill_creator_dir="${CODEX_HOME:-$HOME/.codex}/skills/.system/skill-creator"
python3 "$skill_creator_dir/scripts/quick_validate.py" skills/ai-history
gofmt -w skill_content_test.go
go test . -run TestAIHistorySkillSafetyContract -count=1
git diff --check
```

Expected: 全部 PASS。

```bash
git add skills/ai-history skill_content_test.go
git commit -m "feat: 添加跨 Agent 会话 Skill"
```

### Task 5: 验证 npx 安装并更新 README

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `skill_content_test.go`

- [ ] **Step 1: 先添加 README 失败测试**

对两个 README 断言包含：

```go
required := []string{
	"npx skills add yangkushu/ai-session-history",
	"--skill ai-history",
	"--agent codex",
	"--agent claude-code",
	"--agent cursor",
}
```

Run: `go test . -run TestReadmesDocumentCommonSkillInstallation -count=1`

Expected: FAIL，缺少 npx 安装入口。

- [ ] **Step 2: 更新中英文 README**

加入同一条命令：

```bash
npx skills add yangkushu/ai-session-history \
  --skill ai-history --global \
  --agent codex --agent claude-code --agent cursor
```

同时明确：npx 只用于安装；安装前审阅 GitHub 来源和 Skill；无 Node.js/网络时手动复制
`skills/ai-history` 到 host 官方目录；安装不授予 shell/history 权限，首次调用仍由
`doctor --json` 引导最小授权。

- [ ] **Step 3: 运行本地 discovery**

```bash
npx skills add . --list
```

Expected: 输出发现 `ai-history`。若 npm 下载被 sandbox 阻止，按正常网络授权重试。

- [ ] **Step 4: 隔离验证三 target**

```bash
repo_root="$PWD"
smoke_dir="$(mktemp -d)"
(cd "$smoke_dir" && npx skills add "$repo_root" \
  --skill ai-history --agent codex --agent claude-code --agent cursor --copy -y)
find "$smoke_dir" -type f -name SKILL.md -print
```

Expected: 安装成功且包含 `ai-history`；临时目录没有 permission/sandbox/allowlist/managed
policy 配置。验收后仅删除该次 `mktemp` 返回的 `smoke_dir`。

- [ ] **Step 5: 测试并提交**

```bash
gofmt -w skill_content_test.go
go test . -run 'Test(AIHistorySkillSafetyContract|ReadmesDocumentCommonSkillInstallation)' -count=1
git diff --check
git add README.md README.zh-CN.md skill_content_test.go
git commit -m "docs: 添加通用 Skill 安装说明"
```

Expected: 测试和提交成功。

### Task 6: 全量验证、前向评估、审查和归档

**Files:**
- Modify: `openspec/changes/add-ai-history-agent-skill/tasks.md`
- Archive into: `openspec/changes/archive/2026-07-13-add-ai-history-agent-skill/`
- Sync: `openspec/specs/agent-skill/spec.md`
- Sync: `openspec/specs/cli/spec.md`

- [ ] **Step 1: 更新 OpenSpec checkbox**

仅在对应实现与验证完成后把 `- [ ]` 改为 `- [x]`；4.3/4.4 不提前勾选。

- [ ] **Step 2: 运行全部验证**

```bash
gofmt -w internal/readers/*.go internal/core/service_test.go skill_content_test.go
go test ./...
go vet ./...
GOCACHE=/tmp/go-build go build -o /tmp/ai-history-skill-build-check ./cmd/ai-history
skill_creator_dir="${CODEX_HOME:-$HOME/.codex}/skills/.system/skill-creator"
python3 "$skill_creator_dir/scripts/quick_validate.py" skills/ai-history
openspec validate add-ai-history-agent-skill --strict
git diff --check
```

Expected: 每条命令 exit 0。

- [ ] **Step 3: 运行只读 CLI smoke**

```bash
/tmp/ai-history-skill-build-check version
/tmp/ai-history-skill-build-check doctor --json
/tmp/ai-history-skill-build-check list --here --limit 3 --json
```

Expected: version 可执行；doctor 为三个 source 返回独立诊断；list 输出合法 JSON。

- [ ] **Step 4: 做 fresh-agent 前向评估**

只传 Skill 路径，依次测试：“找当前项目发布会话”“当前项目没找到怎么办”“导出这个
session”“完整 raw 导出”“处理带 permission_denied/path 的 doctor JSON”。验收
`--here --json`、扩大范围前询问、普通 export 不直接 raw、明确 raw 才执行、只请求
诊断 path。偏差只修改 Skill/reference 和契约测试，然后重跑 Step 2。

- [ ] **Step 5: 请求规格符合性与代码质量审查**

审查 `git diff "$(git merge-base HEAD origin/master)"..HEAD`，确认全部 scenario、仅 `os.IsPermission` 映射
permission_denied、Skill 不扩大搜索/raw/权限范围、README 命令与 npx smoke 一致。
修复后重跑 Step 2。

- [ ] **Step 6: 归档 OpenSpec**

```bash
openspec archive add-ai-history-agent-skill --yes
openspec validate --all --strict
git diff --check
```

Expected: change 归档，主 specs 新增 `agent-skill` 并更新 `cli`。

- [ ] **Step 7: 提交归档**

```bash
git add openspec docs README.md README.zh-CN.md skills internal skill_content_test.go
git commit -m "docs: 归档跨 Agent Skill 变更"
git status --short --branch
```

Expected: commit 成功、工作区干净；push 与 release/tag 另行确认。
