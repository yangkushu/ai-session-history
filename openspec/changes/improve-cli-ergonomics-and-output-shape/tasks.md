## 1. Tests First

- [ ] 1.1 Add failing CLI tests for top-level `help`, `--help`, and `-h`.
- [ ] 1.2 Add failing CLI tests for `help <command>`, `<command> --help`, and `<command> -h`.
- [ ] 1.3 Add failing CLI tests for `version` and `--version` development-build output.
- [ ] 1.4 Add failing CLI tests for short aliases: `-c`, `-j`, `-s`, `-l`, `-m`, `-n`, and `-t`.
- [ ] 1.5 Add failing list tests for `--here`, `--here` conflicts, and empty JSON `sessions: []`.
- [ ] 1.6 Add failing render/core tests for tool result, tool error, and context tool-outcome behavior.

## 2. CLI Help, Version, and Flags

- [ ] 2.1 Centralize top-level and subcommand usage text in `internal/cli`.
- [ ] 2.2 Implement top-level help handling before service construction when possible.
- [ ] 2.3 Implement subcommand help handling with successful exit codes.
- [ ] 2.4 Add version variables with development defaults and implement `version` / `--version`.
- [ ] 2.5 Register short aliases for common flags while preserving existing long flags.
- [ ] 2.6 Update error messages for invalid help or conflicting directory flags.

## 3. List Output and Directory Scope

- [ ] 3.1 Initialize `ListResult.Sessions` as an empty slice so JSON emits `[]`.
- [ ] 3.2 Add `Here` or equivalent option to list filtering without changing default all-history behavior.
- [ ] 3.3 Resolve `--here` to the process current working directory in the CLI layer.
- [ ] 3.4 Reject `--here` when combined with explicit `--cwd` or `--under`.
- [ ] 3.5 Verify source diagnostics and unavailable source maps keep existing omit/emit behavior.

## 4. Tool Result Semantics

- [ ] 4.1 Update render behavior so `clean`, `summary`, `raw`, and `context` match the tool-result spec.
- [ ] 4.2 Audit Codex, Claude Code, and Cursor readers for reliable tool final result fields.
- [ ] 4.3 Implement reader extraction only for source formats with reliable tool result text.
- [ ] 4.4 Keep unknown or ambiguous tool formats omitted with explicit markers rather than guessed content.

## 5. Documentation

- [ ] 5.1 Update README usage examples for help, version, short aliases, and `list --here`.
- [ ] 5.2 Document development-build version behavior and future ldflags release injection.
- [ ] 5.3 Update manual acceptance guidance if command examples change.

## 6. Verification

- [ ] 6.1 Run `gofmt` on changed Go files.
- [ ] 6.2 Run `PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go test ./...`.
- [ ] 6.3 Run representative manual CLI checks for help, version, list empty JSON, and `--here`.
- [ ] 6.4 Run `openspec validate improve-cli-ergonomics-and-output-shape --strict`.
- [ ] 6.5 Commit and push with a concise Chinese commit message after verification.
