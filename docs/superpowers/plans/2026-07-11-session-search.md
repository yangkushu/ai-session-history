# Session Search Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only local `ai-history search` command that finds prior AI sessions by literal text and returns ranked, bounded results.

**Architecture:** `core.Service` exposes a stable search contract and delegates the algorithm to an internal `Searcher`. The first `scanSearcher` reuses configured readers to list filtered summaries, load candidate details, score matching categories, and return result records. The CLI only parses flags and formats the stable `SearchResult` contract.

**Tech Stack:** Go 1.22, standard library `strings`, `sort`, `unicode/utf8`, existing Cobra-free flag parsing, OpenSpec.

## Global Constraints

- Search is local-only and read-only: no network, embeddings, persistent index, source mutation, or new dependency.
- Query matching is case-insensitive contiguous literal matching.
- Scores are capped per category: title 100, user 30, assistant 20, tool call/result 10.
- Snippets are at most 200 characters; results sort by score then updated time.
- `--source`, location flags, availability diagnostics, and JSON conventions follow `list`; search defaults to limit 20.

---

### Task 1: Core search contract and scan implementation

**Files:**
- Create: `internal/core/search.go`
- Modify: `internal/core/models.go`
- Modify: `internal/core/service.go`
- Test: `internal/core/search_test.go`

**Interfaces:**
- Consumes: `Reader.ListSessions()`, `Reader.GetSession(string)`, `SessionSummary`, and `SessionDetail`.
- Produces: `SearchOptions`, `SearchMatch`, `SearchHit`, `SearchResult`, `Searcher`, and `(*Service).Search(SearchOptions) SearchResult`.

- [ ] **Step 1: Write the failing core tests**

Create fake Codex and Claude readers with titled sessions and turns that cover title, user, assistant, tool, case-insensitive, CWD, unavailable-source, tie-break, limit, and long-snippet cases. Assert category scores are capped and `Hits` is a non-nil empty slice.

- [ ] **Step 2: Run the focused test to verify it fails**

Run: `GOCACHE=/private/tmp/ai-history-search-gocache go test ./internal/core -run TestSearch -count=1`

Expected: FAIL because `SearchOptions` and `Service.Search` are undefined.

- [ ] **Step 3: Add search data types and delegation boundary**

Add models with JSON names `hits`, `session`, `score`, `matches`, `snippet`, `diagnostics`, `unavailable_sources`, and `total_returned`. Make `Service.Search` delegate to an internal `Searcher` initialized with the service readers.

- [ ] **Step 4: Implement the minimal scan searcher**

Use existing `matchesCWD`, `matchesUnder`, `sources`, and `summaryTime`; ignore unreadable individual details. Normalize with `strings.ToLower`, score each category once, choose the first text match for the bounded UTF-8 snippet, sort, and apply the limit after sorting.

- [ ] **Step 5: Run the focused core tests**

Run: `GOCACHE=/private/tmp/ai-history-search-gocache go test ./internal/core -run TestSearch -count=1`

Expected: PASS.

### Task 2: CLI parsing and output

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/service.go`
- Test: `internal/cli/cli_test.go`

**Interfaces:**
- Consumes: `core.SearchOptions`, `core.SearchResult`, and `Service.Search(core.SearchOptions) core.SearchResult`.
- Produces: `ai-history search <query>` text and JSON output, usage text, and short aliases.

- [ ] **Step 1: Write failing CLI tests**

Replace the P0-unavailable test with tests for missing/blank query, `-s/-l/-j`, `--here` conflict handling, default limit 20, text output containing ID/title/snippet, and JSON output with an empty `hits` array.

- [ ] **Step 2: Run focused CLI tests to verify failure**

Run: `GOCACHE=/private/tmp/ai-history-search-gocache go test ./internal/cli -run TestSearch -count=1`

Expected: the existing unavailable-command behavior fails the new assertions.

- [ ] **Step 3: Implement the search command**

Add `Search` to the CLI service interface and `appService`; replace the reserved command case with `runSearch`. Reuse `list` parsing rules, require one nonblank positional query, default limit to 20, emit `SearchResult` for JSON, and print one tab-separated line per hit in text mode.

- [ ] **Step 4: Update help discovery**

Include `search` in top-level usage and route `help search`, `search --help`, and `search -h` to explicit search usage.

- [ ] **Step 5: Run focused CLI tests**

Run: `GOCACHE=/private/tmp/ai-history-search-gocache go test ./internal/cli -run TestSearch -count=1`

Expected: PASS when the local test binary can execute; otherwise use compile-only verification and record the LC_UUID limitation.

### Task 3: Completion checks and OpenSpec state

**Files:**
- Modify: `openspec/changes/add-session-search/tasks.md`

**Interfaces:**
- Consumes: completed Tasks 1 and 2 and their automated tests.
- Produces: completed OpenSpec checklist and validated change.

- [ ] **Step 1: Format modified Go files**

Run: `gofmt -w internal/core/models.go internal/core/service.go internal/core/search.go internal/core/search_test.go internal/cli/cli.go internal/cli/service.go internal/cli/cli_test.go`

- [ ] **Step 2: Run verification**

Run: `git diff --check`, focused core tests, `go vet ./...`, `go test ./...`, and `openspec validate add-session-search --strict` with `GOCACHE=/private/tmp/ai-history-search-gocache` where needed.

- [ ] **Step 3: Record completed OpenSpec tasks**

Mark each completed checkbox in `openspec/changes/add-session-search/tasks.md` as `[x]` only after its verification passes or its environment limitation is documented.

- [ ] **Step 4: Commit the feature**

Run: `git add internal/core internal/cli openspec/changes/add-session-search docs/superpowers/plans/2026-07-11-session-search.md && git commit -m "feat: add local session search"`
