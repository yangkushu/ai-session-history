## Context

The CLI currently lists normalized session summaries and reads individual
session details through `core.Service`. The `search` command is reserved but
returns a P0-unavailable error. Source readers already expose the detail data
needed to search titles, messages, tool calls, and tool results.

The first implementation must remain local, read-only, and dependency-free,
while avoiding an API shape that prevents a future local full-text index.

## Goals / Non-Goals

**Goals:**

- Search enabled local sources with the same source and CWD scope semantics as
  `list`.
- Provide deterministic literal matching, relevance ranking, snippets, and
  machine-readable output.
- Isolate the search algorithm behind an internal abstraction.

**Non-Goals:**

- Persistent indexing, SQLite FTS, embeddings, fuzzy matching, query language,
  remote search, or source-data mutation.
- Exposing a search-engine selection flag or configuration setting.

## Decisions

### Use an internal `Searcher` abstraction

`core.Service.Search` will delegate to a `Searcher` implementation while
retaining ownership of configured readers and the public result contract. The
initial scan implementation reads candidate summaries, applies source/CWD
filters, reads each candidate detail, and constructs results. Future FTS or
cached implementations can replace the algorithm without changing CLI flags,
JSON fields, or source readers.

### Match normalized content using case-insensitive contiguous literals

The scan implementation will match the query as one contiguous literal in the
title and non-empty turns after Unicode-aware case normalization. It will not
split terms, tokenize, stem, or apply fuzzy matching.

### Rank with fixed, capped category weights

Each result receives at most one contribution from each category: title +100,
user message +30, assistant message +20, and tool call/result +10. Results are
ordered by score descending, then session update time descending. Repeated
matches do not increase a category's score.

### Return bounded snippets rather than full matched content

Each match records its first matching location and returns a text snippet around
that occurrence, capped at 200 characters. Terminal output includes the
session identity and snippet; JSON exposes structured match data.

### Preserve partial results for unavailable sources

Search uses the same source availability model as `list`: unavailable sources
are represented in diagnostics and do not discard results from readable
sources. Individual details that cannot be read are skipped. A valid search
with no matches is successful and returns an empty result array.

## Risks / Trade-offs

- [Large local histories make scanning slow] → Keep `--limit` output-bounded
  and make the scanner replaceable by a later local index.
- [Case normalization is not locale-specific collation] → Document literal,
  case-insensitive matching and defer language-aware query semantics.
- [Snippets can contain sensitive local text] → Keep snippets short and expose
  them only to the invoking local user; do not transmit or persist them.
- [Some source details fail after summary listing] → Skip the individual
  session and preserve the rest of the result set.

## Migration Plan

The command is additive and has no stored-state migration. Existing callers of
`list`, `show`, and `context` retain their behavior.

## Open Questions

None. The initial matching, scoring, snippet, abstraction, and failure
semantics have been agreed.
