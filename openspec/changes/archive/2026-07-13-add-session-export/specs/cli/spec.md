## MODIFIED Requirements

### Requirement: Full import and export excluded from P0

The system SHALL expose the read-only P1 `export` command for complete
normalized session files. The system SHALL continue to reject session import.

#### Scenario: Export command is available

- **WHEN** a user runs `ai-history export <session-id> --output <path>` with
  valid options
- **THEN** the CLI creates the selected session export file rather than
  reporting the P0 unavailable-command error

#### Scenario: Import command unavailable

- **WHEN** a user attempts to run `ai-history import <path>`
- **THEN** the CLI reports that session import is not available in P0
