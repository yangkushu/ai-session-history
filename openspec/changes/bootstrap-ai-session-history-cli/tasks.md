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
- [x] Add WSL-aware Cursor discovery: detect WSL via `/proc/version` and glob
      `/mnt/<drive>/Users/<user>/AppData/Roaming/Cursor/User`, composed through a
      resolver the CLI service calls. Keep `DefaultPaths` pure and add testable
      `isWSL` and mount-glob helpers.

## 4. Source Readers

- [x] Port Codex storage reader behavior from the Python prototype.
- [x] Port Claude Code storage reader behavior from the Python prototype.
- [x] Add Windows Cursor default path discovery and diagnostic scaffolding.
- [x] Document the real Cursor Windows latest storage shape (the `composerHeaders`
      and `cursorDiskKV` tables, `bubbleId:<composerId>:<bubbleId>` keys, and the
      `type`/`text`/`createdAt` bubble fields) in the fixture README.
- [x] Capture a minimized Cursor Windows fixture with synthetic content but the
      real table shape, plus a `writeCursorState` test helper mirroring
      `writeCodexState`.
- [x] Implement the Cursor Windows latest reader: `list` from `composerHeaders`
      excluding archived composers; `show` turns from `bubbleId` `text` by
      `type`.
- [x] Open Cursor `state.vscdb` with SQLite `immutable=1` and read bubble fields
      via `json_extract` so the large `conversationState` blob is never loaded
      whole.
- [ ] Capture a minimized Cursor macOS fixture and implement the macOS reader
      (deferred until a macOS sample is available).

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
- [x] Add Cursor Windows latest fixture tests from the real storage shape.
- [x] Add WSL Cursor discovery tests with a fake `/proc/version` and a fake
      `/mnt` mount tree.
- [x] Verify `list`, `show`, and `context` against the live Windows Cursor
      database on a WSL host (manual integration check; the live database is not
      committed).
- [ ] Add Cursor macOS latest fixture tests from a real sample (deferred).
- [x] Add CLI behavior tests for output shape and error handling.
- [x] Run Go tests, formatting, linting if configured, and OpenSpec validation.

## 8. Completion

- [x] Update documentation with install/build/use examples.
- [x] Update README and status notes to state Windows Cursor is supported and
      macOS Cursor is deferred, and record the `immutable=1` read tradeoff.
- [ ] Archive the OpenSpec change after implementation and verification. Blocked
      until Cursor macOS latest is also validated against a real sample; until
      then the change stays open with Windows complete and macOS deferred.
- [ ] Ask whether to commit and push with a concise Chinese commit message.
