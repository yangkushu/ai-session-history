# Tasks

## 1. Project Setup

- [x] Initialize Go module and CLI entrypoint for `ai-history`.
- [x] Add repository layout for command, core service, source readers, rendering,
      config, diagnostics, and fixtures.
- [x] Add basic test and lint commands.

## 2. Core Model and Service

- [x] Define normalized source, summary, detail, turn, content mode, and error
      types.
- [x] Implement source-prefixed session ID parsing and formatting.
- [x] Implement service methods for `doctor`, `list`, `show`, and `context`.
- [x] Implement limit handling and deterministic sorting by updated/created
      time.

## 3. Configuration and Discovery

- [x] Implement zero-config defaults for Codex, Claude Code, and Cursor.
- [x] Implement optional YAML config loading.
- [x] Implement source enablement, custom paths, and default path toggles.
- [x] Implement per-source diagnostics for missing paths, permission errors, and
      unsupported formats.

## 4. Source Readers

- [x] Port Codex storage reader behavior from the Python prototype.
- [x] Port Claude Code storage reader behavior from the Python prototype.
- [ ] Install latest Cursor on macOS and capture a minimized real fixture.
- [ ] Implement Cursor macOS latest reader from the real fixture.
- [x] Add Windows Cursor default path discovery and diagnostic scaffolding.
- [ ] Validate latest Cursor on Windows with a minimized real fixture before
      marking P0 complete.

## 5. CLI Commands

- [x] Implement `doctor` with human output and `--json`.
- [x] Implement `list` with `--source`, `--cwd`, `--under`, `--limit`, and
      `--json`.
- [x] Implement `show` with `--mode clean|summary|raw`, `--max-chars`, and
      `--json`.
- [x] Implement `context` Markdown handoff with `--target-cwd` and
      `--max-chars`.

## 6. Rendering

- [x] Implement clean, summary, and raw detail rendering with bounded output.
- [x] Implement deterministic Markdown context handoff.
- [x] Ensure initial goal and recent conversation are preserved where possible.
- [x] Mark omitted and truncated content explicitly.

## 7. Verification

- [x] Add Go unit tests mirroring the Python prototype's model, service, Codex,
      Claude Code, and rendering tests.
- [ ] Add Cursor macOS latest fixture tests from a real sample.
- [ ] Add Cursor Windows latest fixture tests from a real sample.
- [x] Add CLI behavior tests for output shape and error handling.
- [x] Run Go tests, formatting, linting if configured, and OpenSpec validation.

## 8. Completion

- [x] Update documentation with install/build/use examples.
- [ ] Archive the OpenSpec change after implementation and verification.
- [ ] Ask whether to commit and push with a concise Chinese commit message.
