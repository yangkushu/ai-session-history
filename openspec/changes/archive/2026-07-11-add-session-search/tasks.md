## 1. Search Domain Contract

- [x] 1.1 Add search option, result, match, and diagnostic models in `internal/core` with stable JSON fields.
- [x] 1.2 Introduce the internal `Searcher` boundary and connect it to `core.Service` without changing existing command behavior.
- [x] 1.3 Implement a scan-based searcher that applies source/CWD filters, literal matching, fixed capped scoring, deterministic ordering, bounded snippets, and partial-source diagnostics.

## 2. CLI Integration

- [x] 2.1 Extend the CLI service contract and implement `search` parsing, help text, short flags, validation, and plain-text output.
- [x] 2.2 Add structured JSON output with stable empty arrays and propagated source diagnostics.

## 3. Verification

- [x] 3.1 Add core tests for all searchable content categories, case-insensitive literal matching, relevance scoring, tie-breaking, snippets, filters, limits, and unavailable sources.
- [x] 3.2 Add CLI tests for usage, flags, invalid input, text output, JSON output, and empty results.
- [x] 3.3 Run formatting, focused tests, the full Go test suite, `go vet ./...`, and OpenSpec validation; document any environment-specific limitation.

## Verification Note

- `CGO_ENABLED=0 go test ./...` passes for all packages.
- The default `go test ./...` remains blocked only for `internal/cli` and
  `internal/readers` by the local macOS `dyld: missing LC_UUID load command`
  failure observed before this change. `go vet ./...` and an all-package
  compile-only test invocation pass.
