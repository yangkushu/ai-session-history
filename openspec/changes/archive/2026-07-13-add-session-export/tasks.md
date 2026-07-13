## 1. 导出模型与渲染

- [x] 1.1 先在 `internal/render/render_test.go` 为 `session-export.v1`、默认 `raw`、`clean`/`summary` 模式，以及完整 Markdown turn 转录建立失败用例。
- [x] 1.2 在 `internal/render/` 实现无字符预算的版本化 `SessionExport` 构建与 JSON/Markdown 编码入口，复用既有内容模式语义。
- [x] 1.3 运行渲染包测试，并进行规格符合性与代码质量审查。

## 2. 服务层完整详情

- [x] 2.1 先在 `internal/cli/service.go` 的测试中覆盖 export 绕过 `detail_chars` 限制、从核心服务取得完整归一化详情的行为。
- [x] 2.2 扩展 CLI 服务接口与 `appService`，为 export 返回版本化导出模型，不改变 `show` 或 `context` 既有默认行为。
- [x] 2.3 运行服务与 CLI 相关测试，并进行规格符合性与代码质量审查。

## 3. CLI 与安全文件写入

- [x] 3.1 先在 `internal/cli/cli_test.go` 覆盖 `export --output`、默认 JSON、`--format markdown`、`--mode`、缺少输出路径、无 `--force` 覆盖拒绝、强制替换、0600 权限及失败时无部分目标文件。
- [x] 3.2 实现 `export` 命令、帮助文本、格式/模式校验，以及同目录临时文件与原子替换写入；保留 `import` 的不可用行为。
- [x] 3.3 运行 CLI 测试，并进行规格符合性与代码质量审查。

## 4. 文档、规格同步与验证

- [x] 4.1 更新中英文 README 的 export 示例、敏感文件提示和命令说明。
- [x] 4.2 将 delta spec 同步到 `openspec/specs/cli/spec.md`，并在完成后归档 OpenSpec change。
- [x] 4.3 运行 `gofmt`、`go test ./...`、`go vet ./...`、编译检查、`openspec validate add-session-export --strict` 与 `git diff --check`，再执行最终规格符合性和代码质量审查。
