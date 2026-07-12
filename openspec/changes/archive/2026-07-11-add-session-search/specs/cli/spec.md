## ADDED Requirements

### Requirement: Local session search command

The system SHALL provide `ai-history search <query>` to search normalized local
session titles and turns from enabled Codex, Claude Code, and supported Cursor
sources without calling a remote service or mutating source data.

#### Scenario: Search enabled local sources

- **WHEN** a user runs `ai-history search <query>` with a non-empty query
- **THEN** the system searches all enabled available sources and returns matching sessions

#### Scenario: Reject a missing query

- **WHEN** a user runs `ai-history search` without a query
- **THEN** the CLI prints search usage and exits with code `2`

#### Scenario: Reject a blank query

- **WHEN** a user runs `ai-history search "   "`
- **THEN** the CLI rejects the query and exits with code `2`

### Requirement: Search filters and limits

The search command SHALL support `--source` (`-s`), `--cwd`, `--under`,
`--here`, `--limit` (`-l`), and `--json` (`-j`) with the same validation and
working-directory scope semantics as `list`. The default limit SHALL be `20`.

#### Scenario: Search a source and current directory subtree

- **WHEN** a user runs `ai-history search <query> --source codex --here`
- **THEN** only matching Codex sessions whose CWD is the current directory or a descendant are returned

#### Scenario: Reject conflicting location filters

- **WHEN** a user combines `--here` with `--cwd` or `--under`
- **THEN** the CLI reports the conflict and exits with code `2`

### Requirement: Literal matching and deterministic ranking

The system SHALL perform case-insensitive contiguous literal matching. A result
score SHALL add at most once per category: title `100`, user message `30`,
assistant message `20`, and tool call or tool result `10`. Results SHALL sort
by descending score and then descending session update time.

#### Scenario: Rank title hits before turn-only hits

- **WHEN** one matching session contains the query in its title and another only contains it in a user message
- **THEN** the title-matching session appears first

#### Scenario: Repeated tool output does not increase rank

- **WHEN** a session repeats the query multiple times in tool output
- **THEN** its tool-content contribution is exactly `10`

### Requirement: Bounded search result representation

Each search result SHALL include the normalized session summary, relevance
score, matching content categories, and a snippet around the first match. A
snippet SHALL contain at most 200 characters. JSON output SHALL include an
array for search results even when no sessions match.

#### Scenario: Search result reports a bounded snippet

- **WHEN** a matching turn is longer than 200 characters
- **THEN** the result returns a snippet around the first match that contains no more than 200 characters

#### Scenario: Empty JSON search result

- **WHEN** a user runs `ai-history search <query> --json` and no session matches
- **THEN** the JSON response includes a search-results array with value `[]`

### Requirement: Partial source availability during search

The system SHALL return matches from readable sources when another enabled
source is unavailable. JSON output SHALL include unavailable-source and
diagnostic information consistent with `list`; a valid partial result SHALL
exit with code `0`.

#### Scenario: One source is unavailable

- **WHEN** one enabled source cannot be read and another source has matching sessions
- **THEN** the matching sessions are returned with unavailable-source diagnostics
- **AND** the command exits with code `0`
