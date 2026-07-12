# CLI Specification

## Purpose

Define the local-first `ai-history` CLI behavior for reading, listing, showing,
diagnosing, and rendering handoff context from local AI coding session history.
## Requirements
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

### Requirement: CLI help discovery

The system SHALL expose conventional help entry points for the top-level CLI and
each supported command.

#### Scenario: Top-level help command

- **WHEN** a user runs `ai-history help`
- **THEN** the CLI prints top-level usage and supported commands
- **AND** exits successfully

#### Scenario: Top-level help flag

- **WHEN** a user runs `ai-history --help` or `ai-history -h`
- **THEN** the CLI prints top-level usage and supported commands
- **AND** exits successfully

#### Scenario: Subcommand help command

- **WHEN** a user runs `ai-history help list`
- **THEN** the CLI prints usage for the `list` command
- **AND** exits successfully

#### Scenario: Subcommand help flag

- **WHEN** a user runs `ai-history list --help` or `ai-history list -h`
- **THEN** the CLI prints usage for the `list` command
- **AND** exits successfully

### Requirement: CLI version discovery

The system SHALL expose user-visible CLI version information without requiring a
source history read.

#### Scenario: Version command

- **WHEN** a user runs `ai-history version`
- **THEN** the CLI prints the current CLI version information
- **AND** exits successfully

#### Scenario: Version flag

- **WHEN** a user runs `ai-history --version`
- **THEN** the CLI prints the current CLI version information
- **AND** exits successfully

#### Scenario: Development build version

- **WHEN** no release version has been injected into the binary
- **THEN** version output identifies the build as a development build

### Requirement: Stable short flag aliases

The system SHALL support stable short aliases for common non-ambiguous command
flags while preserving all existing long flags.

#### Scenario: JSON short alias

- **WHEN** a user runs `ai-history doctor -j`
- **THEN** the CLI behaves the same as `ai-history doctor --json`

#### Scenario: Config short alias

- **WHEN** a user runs `ai-history doctor -c <path>`
- **THEN** the CLI behaves the same as `ai-history doctor --config <path>`

#### Scenario: List short aliases

- **WHEN** a user runs `ai-history list -s codex -l 10 -j`
- **THEN** the CLI behaves the same as `ai-history list --source codex --limit 10 --json`

#### Scenario: Show short aliases

- **WHEN** a user runs `ai-history show <session-id> -m summary -n 2000 -j`
- **THEN** the CLI behaves the same as `ai-history show <session-id> --mode summary --max-chars 2000 --json`

#### Scenario: Context short aliases

- **WHEN** a user runs `ai-history context <session-id> -t <path> -n 4000`
- **THEN** the CLI behaves the same as `ai-history context <session-id> --target-cwd <path> --max-chars 4000`

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

#### Scenario: Current directory subtree listing

- **WHEN** a user runs `ai-history list --here`
- **THEN** the system returns only sessions whose normalized working directory is
  the process current working directory or a descendant of it

#### Scenario: Bounded listing

- **WHEN** a user runs `ai-history list --limit 50`
- **THEN** the system returns no more than 50 session summaries

### Requirement: Stable empty list JSON

The system SHALL encode empty JSON collections using arrays rather than `null`
for primary list results.

#### Scenario: Empty list returns an empty sessions array

- **WHEN** a user runs `ai-history list --json` and no sessions match
- **THEN** the JSON response includes `"sessions": []`
- **AND** `total_returned` is `0`

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

#### Scenario: Clean mode keeps concise errors

- **WHEN** detail contains tool or command errors
- **THEN** `clean` mode preserves concise error text needed to understand the
  session outcome

#### Scenario: Summary mode keeps lightweight traces

- **WHEN** a user runs `ai-history show <session-id> --mode summary`
- **THEN** the system includes lightweight tool-call, command, result, and error
  summaries without large outputs

#### Scenario: Raw mode is bounded

- **WHEN** a user runs `ai-history show <session-id> --mode raw --max-chars <n>`
- **THEN** the system returns best-effort raw content subject to `<n>`

### Requirement: Tool result output semantics

The system SHALL define predictable output behavior for tool calls, tool
results, and tool errors when a source reader can identify them reliably.

#### Scenario: Clean mode omits noisy tool output but keeps errors

- **WHEN** detail contains tool output and tool errors
- **THEN** `clean` mode omits noisy tool output with an omitted-content marker
- **AND** preserves concise tool error text

#### Scenario: Summary mode includes lightweight tool summaries

- **WHEN** detail contains tool calls, command results, or tool errors
- **THEN** `summary` mode includes lightweight summaries without large raw
  outputs

#### Scenario: Raw mode includes bounded tool text

- **WHEN** detail contains tool text and the user runs `show --mode raw`
- **THEN** the CLI includes best-effort raw tool text subject to `--max-chars`

#### Scenario: Context includes useful tool final results

- **WHEN** a session has concise tool final results or errors that are useful
  for handoff
- **THEN** `context` includes those concise results or errors without dumping
  large raw tool output

### Requirement: Deterministic context handoff

The system SHALL render deterministic handoff context for continuing a prior AI
coding session in another agent or working directory.

#### Scenario: Context emits Markdown by default

- **WHEN** a user runs `ai-history context <session-id>`
- **THEN** the system emits Markdown headed `# AI Session Context`

#### Scenario: Context includes migration metadata

- **WHEN** a user runs `ai-history context <session-id> --target-cwd <path>`
- **THEN** the handoff includes the session ID, source, original cwd, target
  cwd, created time, and updated time when available

#### Scenario: Context uses stable handoff sections

- **WHEN** a user runs `ai-history context <session-id>`
- **THEN** the handoff presents session metadata, initial goal, recent
  conversation, useful tool outcomes, and omitted-content notes in a stable
  deterministic order

#### Scenario: Context cleans the initial goal

- **WHEN** a session begins with injected environment context, AGENTS/CLAUDE
  instructions, local runtime metadata, empty text, or other known setup
  boilerplate before the user's actual task
- **THEN** the handoff uses the first meaningful user task as the initial goal
- **AND** does not present the skipped boilerplate as the user's goal

#### Scenario: Context marks missing initial goal

- **WHEN** no meaningful user task remains after deterministic boilerplate
  filtering
- **THEN** the handoff marks the initial goal as unavailable rather than
  inventing one

#### Scenario: Context preserves recent conversation

- **WHEN** a session has user and assistant messages
- **THEN** the handoff includes recent clean user/assistant conversation turns
  within the size limit
- **AND** excludes known setup boilerplate from the recent conversation unless it
  is needed to explain the task

#### Scenario: Context preserves useful tool outcomes

- **WHEN** a session includes concise tool final results or errors that are
  useful for continuing the work
- **THEN** the handoff includes those outcomes without large raw tool output

#### Scenario: Context marks omitted content

- **WHEN** context rendering skips boilerplate, omits noisy tool output, or
  truncates lower-priority content due to `--max-chars`
- **THEN** the handoff includes deterministic notes that content was skipped,
  omitted, or truncated

#### Scenario: Context does not invent summaries

- **WHEN** rendering context
- **THEN** the system MUST NOT synthesize key decisions, current state, next
  steps, or other interpretive summaries that require model-like understanding

#### Scenario: Context is bounded

- **WHEN** rendered context exceeds `--max-chars`
- **THEN** the system truncates the output and marks it as truncated

#### Scenario: Context emits structured JSON on request

- **WHEN** a user runs `ai-history context <session-id> --json`
- **THEN** the system emits a JSON object with `schema_version`, `session`,
  `initial_goal`, `recent_conversation`, `tool_outcomes`, `handoff_notes`,
  `handoff_instruction`, and `truncated` fields
- **AND** the command exits successfully

#### Scenario: Context JSON declares schema version

- **WHEN** a user runs `ai-history context <session-id> --json`
- **THEN** the JSON includes `schema_version` set to `context-handoff.v1`
- **AND** compatible future releases MUST NOT change the meaning of the core
  v1 fields without introducing a new schema version

#### Scenario: Context JSON uses handoff semantics

- **WHEN** a user runs `ai-history context <session-id> --json`
- **THEN** the JSON represents a continuation handoff rather than raw source
  history or full normalized session detail
- **AND** the JSON hides Codex, Claude Code, and Cursor storage-specific fields
  behind the normalized handoff structure

#### Scenario: Context JSON omits raw turns

- **WHEN** a user runs `ai-history context <session-id> --json`
- **THEN** the JSON MUST NOT include raw source turns, raw source records, or a
  full normalized turn list
- **AND** users who need normalized turn detail can use `show --json`

#### Scenario: Context JSON includes target cwd

- **WHEN** a user runs `ai-history context <session-id> --target-cwd <path> --json`
- **THEN** the JSON `session` object includes the target cwd value

#### Scenario: Context JSON preserves empty collections

- **WHEN** a JSON handoff has no recent conversation or no tool outcomes
- **THEN** the corresponding fields are empty arrays rather than `null`

#### Scenario: Context JSON preserves core fields

- **WHEN** source metadata such as title, created time, or updated time is
  unavailable
- **THEN** the JSON still includes the top-level core fields
- **AND** optional unavailable metadata fields may be omitted

#### Scenario: Context JSON uses structured notes

- **WHEN** `handoff_notes` contains skipped, omitted, or truncated content notes
- **THEN** each note includes a stable `code`
- **AND** each note includes a human-readable `message`

#### Scenario: Context JSON reports truncation

- **WHEN** `ai-history context <session-id> --json --max-chars <n>` must omit or
  reduce lower-priority handoff content to fit the requested content budget
- **THEN** the JSON sets `truncated` to `true`
- **AND** `handoff_notes` includes a deterministic truncation note

#### Scenario: Context JSON max chars is a content budget

- **WHEN** a user runs `ai-history context <session-id> --json --max-chars <n>`
- **THEN** the system treats `<n>` as a handoff content budget rather than a
  strict serialized JSON byte limit
- **AND** the system preserves the JSON object shape and core fields

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

### Requirement: Cursor latest macOS support

The system SHALL support latest Cursor macOS session reading from the observed
real local storage shape.

#### Scenario: Cursor macOS latest validated

- **WHEN** latest Cursor on macOS stores composer data in `cursorDiskKV` using
  `composerData:<composerId>` keys
- **THEN** `list`, `show`, and `context` work for those Cursor macOS sessions

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

### Requirement: CLI release artifacts

The system SHALL provide automated release artifacts for the `ai-history` CLI
from explicit release tags.

#### Scenario: Tag-triggered release workflow

- **WHEN** a maintainer pushes a Git tag matching `v*`
- **THEN** GitHub Actions runs the release workflow
- **AND** the workflow publishes release artifacts through GitHub Releases

#### Scenario: Manual release workflow dispatch

- **WHEN** a maintainer starts the release workflow manually
- **THEN** GitHub Actions runs the same release automation without requiring a
  branch push trigger

#### Scenario: Cross-platform archives

- **WHEN** a release workflow runs
- **THEN** it builds release archives for Linux, macOS, and Windows on supported
  `amd64` and `arm64` targets

#### Scenario: Release checksums

- **WHEN** release artifacts are produced
- **THEN** the release includes a checksum file covering the published archives

#### Scenario: Release version metadata

- **WHEN** a user runs `ai-history version` from a release artifact
- **THEN** the output includes the release version
- **AND** includes commit and build date metadata when available

#### Scenario: Local release validation

- **WHEN** a maintainer validates release configuration locally before pushing a
  tag
- **THEN** the repository documents a snapshot or check command that does not
  publish a GitHub Release
