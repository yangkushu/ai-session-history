## ADDED Requirements

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

### Requirement: Current-directory list shortcut
The system SHALL provide an explicit shortcut for listing sessions under the
current working directory without changing the default all-history list
behavior.

#### Scenario: Default list remains all-history
- **WHEN** a user runs `ai-history list`
- **THEN** the CLI lists sessions from every enabled available source without
  applying an implicit current-directory filter

#### Scenario: Here list filters by current directory subtree
- **WHEN** a user runs `ai-history list --here`
- **THEN** the CLI returns only sessions whose normalized working directory is
  the process current working directory or a descendant of it

#### Scenario: Here conflicts with explicit directory filters
- **WHEN** a user runs `ai-history list --here --under <path>`
- **THEN** the CLI returns a usage error explaining that `--here` cannot be
  combined with explicit directory filters

### Requirement: Stable empty list JSON
The system SHALL encode empty JSON collections using arrays rather than `null`
for primary list results.

#### Scenario: Empty list returns an empty sessions array
- **WHEN** a user runs `ai-history list --json` and no sessions match
- **THEN** the JSON response includes `"sessions": []`
- **AND** `total_returned` is `0`

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

## MODIFIED Requirements

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

#### Scenario: Context preserves useful tool outcomes
- **WHEN** a session includes concise tool final results or errors that are
  useful for continuing the work
- **THEN** the Markdown includes those outcomes without large raw tool output

#### Scenario: Context does not invent summaries
- **WHEN** rendering P0 context
- **THEN** the system MUST NOT synthesize key decisions, current state, or other
  interpretive summaries that require model-like understanding

#### Scenario: Context is bounded
- **WHEN** rendered context exceeds `--max-chars`
- **THEN** the system truncates the output and marks it as truncated
