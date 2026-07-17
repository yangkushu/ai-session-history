## ADDED Requirements

### Requirement: Readable list text rendering

The system SHALL render each session in human-readable `list` output as a
deterministic four-line block without ANSI styling or terminal-dependent layout.

#### Scenario: Complete session block

- **WHEN** a session has source, title, updated time, created time, CWD, and ID
- **THEN** the first line contains the source followed by the normalized title
- **AND** the second line contains updated time followed by created time
- **AND** each timestamp uses local `YYYY-MM-DD HH:mm` followed by its compact
  relative age in parentheses
- **AND** the third line contains the complete CWD
- **AND** the fourth line contains the complete source-prefixed session ID
- **AND** lines two through four align with the title content column

#### Scenario: Multiple session blocks

- **WHEN** human-readable list output contains multiple sessions
- **THEN** adjacent session blocks have exactly one blank line between them
- **AND** output has no additional blank line after the final block

#### Scenario: TTY and redirected output match

- **WHEN** human-readable list output is written to a terminal or redirected to
  a pipe or file
- **THEN** the text content and layout are the same
- **AND** no ANSI color or styling sequences are emitted

#### Scenario: Missing summary values

- **WHEN** created time or updated time is missing
- **THEN** its fixed position on the time line contains `unknown`
- **AND** no relative age is appended to that missing time
- **WHEN** CWD is missing
- **THEN** the CWD line contains `unknown`

### Requirement: Compact list time formatting

The system SHALL render updated time before created time and SHALL calculate a
separate compact relative age for each present timestamp using the same captured
current time.

#### Scenario: Relative age units

- **WHEN** elapsed time is less than one minute or the timestamp is in the future
- **THEN** the relative age is `now`
- **WHEN** elapsed time is at least one minute and less than one hour
- **THEN** the relative age is the floored whole-minute count with suffix `m`
- **WHEN** elapsed time is at least one hour and less than one day
- **THEN** the relative age is the floored whole-hour count with suffix `h`
- **WHEN** elapsed time is at least one day and less than 30 days
- **THEN** the relative age is the floored whole-day count with suffix `d`
- **WHEN** elapsed time is at least 30 days and less than 365 days
- **THEN** the relative age is the floored count of 30-day months with suffix
  `mo`
- **WHEN** elapsed time is at least 365 days
- **THEN** the relative age is the floored count of 365-day years with suffix
  `y`

#### Scenario: Local absolute time

- **WHEN** a session timestamp is present
- **THEN** the absolute portion is converted to the process local timezone
- **AND** formatted as `YYYY-MM-DD HH:mm`

### Requirement: Safe list title formatting

The system SHALL keep every list title on one logical output line and SHALL
limit it to 80 terminal display cells without truncating CWD or session ID.

#### Scenario: Multiline and repeated whitespace title

- **WHEN** a title contains newlines, tabs, or repeated whitespace
- **THEN** each whitespace run is replaced with one ASCII space
- **AND** leading and trailing whitespace is removed

#### Scenario: Wide title truncation

- **WHEN** a normalized title exceeds 80 terminal display cells
- **THEN** it is truncated and ends with `…`
- **AND** the rendered title including `…` occupies no more than 80 display cells
- **AND** display width accounts for CJK, emoji, and combining characters rather
  than bytes alone

#### Scenario: Full path and ID preservation

- **WHEN** CWD or session ID exceeds 80 display cells
- **THEN** the complete value is still rendered on its dedicated line

### Requirement: Stable list JSON during text rendering changes

The system MUST preserve the existing `list --json` schema and values while
changing human-readable list output.

#### Scenario: JSON output remains structured

- **WHEN** a user runs `ai-history list --json`
- **THEN** the CLI emits the existing `ListResult` JSON representation
- **AND** does not emit the four-line human-readable blocks
