## ADDED Requirements

### Requirement: Versioned normalized session export

The system SHALL build a source-neutral `session-export.v1` object containing
the export timestamp, selected content mode, and one normalized
`core.SessionDetail`. The default content mode SHALL be `raw`; it SHALL include
all available turns without a configured character limit. `clean` and `summary`
SHALL retain the established content-mode omission behavior. The export object
MUST NOT contain source-private storage records.

#### Scenario: Default raw export preserves every normalized turn
- **WHEN** a user exports a session without supplying `--mode`
- **THEN** the generated export has `schema_version` equal to
  `session-export.v1`, records `raw` as its content mode, and contains every
  normalized turn without character-budget truncation

#### Scenario: Clean export applies established omission behavior
- **WHEN** a user exports a session with `--mode clean`
- **THEN** the generated export applies the same clean-mode tool-content
  omissions used by session detail rendering while preserving the normalized
  session metadata and turn order

### Requirement: JSON and Markdown export encodings

The system SHALL encode the same session export model as JSON or Markdown. JSON
SHALL be the default format. Markdown SHALL include export metadata and every
turn represented by the selected content mode, including tool calls, tool
results, and errors when present.

#### Scenario: JSON is the default format
- **WHEN** a user exports a session without supplying `--format`
- **THEN** the output file contains an indented JSON representation of the
  versioned session export object

#### Scenario: Markdown represents the full selected export
- **WHEN** a user exports a session with `--format markdown`
- **THEN** the output file contains identifiable session metadata and all turns
  represented by the selected content mode

### Requirement: Safe durable export files

The system SHALL require an explicit output path, write the completed export to
that path atomically, and create new export files with owner-only `0600`
permissions. The system SHALL refuse to replace an existing destination unless
the user explicitly supplies `--force`; failed writes MUST NOT leave a partial
destination file.

#### Scenario: Export writes a new private file
- **WHEN** a user supplies a non-existing output path
- **THEN** the system writes the selected complete export atomically and the
  resulting file is readable and writable only by the current user

#### Scenario: Existing export is protected by default
- **WHEN** a user supplies an output path that already exists without `--force`
- **THEN** the command reports a usage error and leaves the existing file
  unchanged

#### Scenario: Explicit force replaces an existing export
- **WHEN** a user supplies an existing output path with `--force`
- **THEN** the system atomically replaces that path with the new selected export

#### Scenario: Failed export leaves no partial destination
- **WHEN** encoding or writing the export fails before replacement
- **THEN** the system returns a runtime error and does not create or modify the
  destination file
