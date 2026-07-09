## MODIFIED Requirements

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

#### Scenario: Context uses stable handoff sections

- **WHEN** a user runs `ai-history context <session-id>`
- **THEN** the Markdown presents session metadata, initial goal, recent
  conversation, useful tool outcomes, and omitted-content notes in a stable
  deterministic order

#### Scenario: Context cleans the initial goal

- **WHEN** a session begins with injected environment context, AGENTS/CLAUDE
  instructions, local runtime metadata, empty text, or other known setup
  boilerplate before the user's actual task
- **THEN** the Markdown uses the first meaningful user task as the initial goal
- **AND** does not present the skipped boilerplate as the user's goal

#### Scenario: Context marks missing initial goal

- **WHEN** no meaningful user task remains after deterministic boilerplate
  filtering
- **THEN** the Markdown marks the initial goal as unavailable rather than
  inventing one

#### Scenario: Context preserves recent conversation

- **WHEN** a session has user and assistant messages
- **THEN** the Markdown includes recent clean user/assistant conversation turns
  within the size limit
- **AND** excludes known setup boilerplate from the recent conversation unless it
  is needed to explain the task

#### Scenario: Context preserves useful tool outcomes

- **WHEN** a session includes concise tool final results or errors that are
  useful for continuing the work
- **THEN** the Markdown includes those outcomes without large raw tool output

#### Scenario: Context marks omitted content

- **WHEN** context rendering skips boilerplate, omits noisy tool output, or
  truncates lower-priority content due to `--max-chars`
- **THEN** the Markdown includes deterministic notes that content was skipped,
  omitted, or truncated

#### Scenario: Context does not invent summaries

- **WHEN** rendering context
- **THEN** the system MUST NOT synthesize key decisions, current state, next
  steps, or other interpretive summaries that require model-like understanding

#### Scenario: Context is bounded

- **WHEN** rendered context exceeds `--max-chars`
- **THEN** the system truncates the output and marks it as truncated
