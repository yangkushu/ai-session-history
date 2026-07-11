## MODIFIED Requirements

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
