# CLI Specification Delta

## ADDED Requirements

### Requirement: Local-first CLI boundary

The system SHALL provide a native CLI named `ai-history` as the primary product
interface for local AI coding session history.

#### Scenario: CLI works without MCP

- **WHEN** a user runs a P0 command
- **THEN** the command executes without requiring an MCP client or MCP server

#### Scenario: CLI stays local

- **WHEN** any P0 command runs
- **THEN** the system MUST NOT call a remote service, LLM, embedding model, or
  vector database

#### Scenario: CLI is read-only

- **WHEN** any P0 command reads source histories
- **THEN** the system MUST NOT write, delete, archive, resume, fork, rewrite, or
  otherwise mutate source session data

### Requirement: Source diagnostics

The system SHALL expose `doctor` diagnostics for Codex, Claude Code, and Cursor
source availability.

#### Scenario: All sources checked independently

- **WHEN** a user runs `ai-history doctor`
- **THEN** the system reports each enabled source independently

#### Scenario: One source unavailable

- **WHEN** one enabled source has no readable default or configured path
- **THEN** `doctor` reports that source as unavailable without hiding available
  sources

#### Scenario: Unsupported Cursor format

- **WHEN** Cursor storage exists but does not match a supported latest macOS or
  Windows format
- **THEN** `doctor` reports `unsupported_format` with the inspected path and a
  concise reason

### Requirement: Cross-source session listing

The system SHALL list normalized session summaries from Codex, Claude Code, and
supported Cursor storage.

#### Scenario: List all enabled sources

- **WHEN** a user runs `ai-history list`
- **THEN** the system returns session summaries from every enabled available
  source

#### Scenario: Source-filtered listing

- **WHEN** a user runs `ai-history list --source codex`
- **THEN** the system returns only Codex session summaries

#### Scenario: Exact cwd listing

- **WHEN** a user runs `ai-history list --cwd <path>`
- **THEN** the system returns only sessions whose normalized working directory
  equals `<path>`

#### Scenario: Directory subtree listing

- **WHEN** a user runs `ai-history list --under <path>`
- **THEN** the system returns only sessions whose normalized working directory is
  `<path>` or a descendant of `<path>`

#### Scenario: Bounded listing

- **WHEN** a user runs `ai-history list --limit 50`
- **THEN** the system returns no more than 50 session summaries

### Requirement: Session detail reading

The system SHALL read a session by source-prefixed session ID and return
normalized detail.

#### Scenario: Read by session ID

- **WHEN** a user runs `ai-history show codex:<native-id>`
- **THEN** the system reads the matching Codex local session and renders
  normalized turns

#### Scenario: Unknown session ID

- **WHEN** a user runs `ai-history show <source:missing>`
- **THEN** the system returns `session_not_found` with the requested session ID

#### Scenario: Invalid session ID

- **WHEN** a user runs `ai-history show invalid`
- **THEN** the system returns `invalid_session_id`

### Requirement: Content modes

The system SHALL support `clean`, `summary`, and `raw` content modes for session
detail output.

#### Scenario: Clean mode is default

- **WHEN** a user runs `ai-history show <session-id>` without `--mode`
- **THEN** the system uses `clean` mode

#### Scenario: Clean mode omits noisy output

- **WHEN** detail contains large tool output, terminal output, diffs, or logs
- **THEN** `clean` mode omits that content and marks the omission

#### Scenario: Summary mode keeps lightweight traces

- **WHEN** a user runs `ai-history show <session-id> --mode summary`
- **THEN** the system includes lightweight tool-call, command, and error
  summaries without large outputs

#### Scenario: Raw mode is bounded

- **WHEN** a user runs `ai-history show <session-id> --mode raw --max-chars <n>`
- **THEN** the system returns best-effort raw content subject to `<n>`

### Requirement: Deterministic context handoff

The system SHALL render deterministic Markdown handoff context for continuing a
prior AI coding session in another agent or working directory.

#### Scenario: Context emits Markdown by default

- **WHEN** a user runs `ai-history context <session-id>`
- **THEN** the system emits Markdown headed `# AI Session Context`

#### Scenario: Context includes migration metadata

- **WHEN** a user runs `ai-history context <session-id> --target-cwd <path>`
- **THEN** the Markdown includes the session ID, source, original cwd, target
  cwd, created time, and updated time when available

#### Scenario: Context preserves initial goal and recent conversation

- **WHEN** a session has user and assistant messages
- **THEN** the Markdown includes the first user message as the initial goal and
  recent clean user/assistant conversation turns within the size limit

#### Scenario: Context does not invent summaries

- **WHEN** rendering P0 context
- **THEN** the system MUST NOT synthesize key decisions, current state, or other
  interpretive summaries that require model-like understanding

#### Scenario: Context is bounded

- **WHEN** rendered context exceeds `--max-chars`
- **THEN** the system truncates the output and marks it as truncated

### Requirement: Optional configuration

The system SHALL work without a config file and SHALL support optional YAML
configuration for local environment overrides.

#### Scenario: Zero-config defaults

- **WHEN** no config file is supplied
- **THEN** the system uses default source discovery paths

#### Scenario: Configured source paths

- **WHEN** config supplies custom source paths
- **THEN** the system reads those paths in addition to defaults unless
  `use_default_paths` is false

#### Scenario: Disabled source

- **WHEN** config disables a source
- **THEN** the system MUST NOT read that source

#### Scenario: Configured limits

- **WHEN** config supplies detail or context size limits
- **THEN** commands use those limits unless overridden by command flags

### Requirement: Cursor latest Windows reading

The system SHALL read latest Cursor sessions on Windows from the real
`globalStorage/state.vscdb` storage shape, validated against a real local
Windows sample.

#### Scenario: Cursor Windows list and read

- **WHEN** latest Cursor on Windows has a readable `globalStorage/state.vscdb`
  with the `composerHeaders` and `cursorDiskKV` tables
- **THEN** `list` returns non-archived composer sessions and `show` returns
  their message turns

#### Scenario: Cursor Windows excludes archived composers

- **WHEN** a composer row has `isArchived` set in its `value` JSON
- **THEN** `list` MUST NOT include that composer

#### Scenario: Cursor Windows reads message text only

- **WHEN** a composer bubble has non-empty `text`
- **THEN** `show` renders it as a user or assistant message turn based on the
  bubble `type`
- **AND WHEN** a bubble has no `text`
- **THEN** `show` MUST NOT render it as a turn, because its content is not
  practically parseable in P0

#### Scenario: Cursor Windows live database opened immutably

- **WHEN** the Cursor database is owned and live-updated by Cursor, possibly on
  a WSL-mounted Windows filesystem
- **THEN** the reader opens it with SQLite immutable mode to avoid lock and WAL
  contention

### Requirement: WSL discovery of Windows Cursor storage

The system SHALL discover Windows Cursor storage from a WSL host by default.

#### Scenario: WSL host auto-detects Windows Cursor

- **WHEN** the CLI runs on a WSL host, detected via `/proc/version` mentioning
  Microsoft
- **AND** a Windows user directory contains `AppData/Roaming/Cursor/User`
- **THEN** the CLI adds that directory to the default Cursor roots without
  manual config

#### Scenario: Configured Cursor paths still respected on WSL

- **WHEN** config supplies Cursor `paths`
- **THEN** the system reads those paths in addition to WSL defaults unless
  `use_default_paths` is false

### Requirement: Cursor latest macOS support deferred

The system SHALL NOT mark Cursor macOS latest reading complete until it is
validated against a real macOS sample.

#### Scenario: Cursor macOS latest gated

- **WHEN** Cursor macOS has not been validated with a real local sample
- **THEN** P0 MUST NOT mark macOS Cursor reading complete
- **AND** the OpenSpec change MUST NOT be archived while macOS Cursor remains
  unvalidated

### Requirement: Unsupported Cursor variants

The system SHALL report `unsupported_format` for Cursor storage that is not a
supported latest Windows or macOS format, instead of attempting a speculative
parse.

#### Scenario: Native Linux or old Cursor storage

- **WHEN** Cursor storage is from native Linux, an old Cursor version, or an
  unrecognized future format
- **THEN** the system reports `unsupported_format` with the inspected path
  instead of attempting a best-effort parse

### Requirement: Search excluded from P0

The system SHALL NOT expose a P0 `search` command.

#### Scenario: Search command unavailable

- **WHEN** a user attempts to run `ai-history search`
- **THEN** the CLI reports that the command is unavailable rather than running a
  metadata-only or transcript search

### Requirement: Full import and export excluded from P0

The system SHALL NOT expose full session import or export commands in P0.

#### Scenario: Export command unavailable

- **WHEN** a user attempts to run `ai-history export <session-id>`
- **THEN** the CLI reports that full session export is not available in P0 and
  points users to `context` for Markdown handoff or `show --json` for normalized
  detail output

#### Scenario: Import command unavailable

- **WHEN** a user attempts to run `ai-history import <path>`
- **THEN** the CLI reports that session import is not available in P0

#### Scenario: Source histories remain read-only

- **WHEN** any command runs
- **THEN** the system MUST NOT import into, write back to, or synthesize native
  history records for Codex, Claude Code, Cursor, or other source-owned stores
