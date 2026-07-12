## Why

Users can list or open local AI coding sessions, but cannot locate a prior
discussion by its content. A local-first search command makes the existing
history useful once the number of sessions grows.

## What Changes

- Add `ai-history search <query>` for case-insensitive, contiguous literal
  matching across normalized session titles and turns from enabled sources.
- Add the same source and working-directory filters as `list`, plus bounded
  text and JSON result output with match snippets.
- Define deterministic relevance ranking and partial-source failure behavior.
- Introduce an internal search abstraction with a scan-based implementation,
  allowing future indexed implementations without changing the CLI contract.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `cli`: Add a local, read-only search command and its output, filtering,
  ranking, and error-handling requirements.

## Impact

- Affects `internal/core` search models and service behavior, plus the CLI
  service interface, command parsing, rendering, and tests.
- Does not change source reader formats or add remote services, embeddings,
  persistent indexes, or dependencies.
