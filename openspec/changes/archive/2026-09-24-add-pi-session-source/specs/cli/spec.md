## MODIFIED Requirements

### Requirement: Source diagnostics

The system SHALL expose `doctor` diagnostics for Codex, Claude Code, Cursor, and Pi source availability and explicit history-path permission failures.

#### Scenario: All sources checked independently

- **WHEN** a user runs `ai-history doctor`
- **THEN** the system reports each enabled source independently

#### Scenario: One source unavailable

- **WHEN** one enabled source has no readable default or configured path
- **THEN** `doctor` reports that source as unavailable without hiding available sources

#### Scenario: Source history path permission denied

- **WHEN** a reader encounters an explicit OS permission error while inspecting a configured or default history path
- **THEN** `doctor --json` reports that source with code `permission_denied` and the denied path without hiding other source diagnostics

#### Scenario: Unsupported Cursor format

- **WHEN** Cursor storage exists but does not match a supported latest macOS or Windows format
- **THEN** `doctor` reports `unsupported_format` with the inspected path and a concise reason

#### Scenario: Pi has partially readable storage

- **WHEN** Pi has both valid and invalid session files
- **THEN** `doctor --json` reports Pi as `partial` with per-file warnings while other sources remain independently diagnosable

### Requirement: Cross-source session listing

The system SHALL list normalized session summaries from Codex, Claude Code, Pi, and supported Cursor storage.

#### Scenario: List all enabled sources

- **WHEN** a user runs `ai-history list`
- **THEN** the system returns session summaries from every enabled available source

#### Scenario: Source-filtered listing

- **WHEN** a user runs `ai-history list --source codex`
- **THEN** the system returns only Codex session summaries

#### Scenario: Exact cwd listing

- **WHEN** a user runs `ai-history list --cwd <path>`
- **THEN** the system returns only sessions whose normalized working directory equals `<path>`

#### Scenario: Directory subtree listing

- **WHEN** a user runs `ai-history list --under <path>`
- **THEN** the system returns only sessions whose normalized working directory is `<path>` or a descendant of `<path>`

#### Scenario: Current directory subtree listing

- **WHEN** a user runs `ai-history list --here`
- **THEN** the system returns only sessions whose normalized working directory is the process current working directory or a descendant of it

#### Scenario: Bounded listing

- **WHEN** a user runs `ai-history list --limit 50`
- **THEN** the system returns no more than 50 session summaries

#### Scenario: List Pi sessions without other sources

- **WHEN** a user runs `ai-history list --source pi --json`
- **THEN** the system returns only Pi session summaries and any Pi-specific diagnostics

### Requirement: Local session search command

The system SHALL provide `ai-history search <query>` to search normalized local
session titles and turns from enabled Codex, Claude Code, Pi, and supported Cursor
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

#### Scenario: Search Pi sessions

- **WHEN** a user runs `ai-history search <query> --source pi --json`
- **THEN** the system searches Pi's normalized title and text turns without including opaque Pi payloads or other sources
