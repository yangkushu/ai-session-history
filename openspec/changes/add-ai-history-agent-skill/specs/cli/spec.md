## MODIFIED Requirements

### Requirement: Source diagnostics

The system SHALL expose `doctor` diagnostics for Codex, Claude Code, and Cursor
source availability and explicit history-path permission failures.

#### Scenario: All sources checked independently

- **WHEN** a user runs `ai-history doctor`
- **THEN** the system reports each enabled source independently

#### Scenario: One source unavailable

- **WHEN** one enabled source has no readable default or configured path
- **THEN** `doctor` reports that source as unavailable without hiding available
  sources

#### Scenario: Source history path permission denied

- **WHEN** a reader encounters an explicit OS permission error while inspecting a configured or default history path
- **THEN** `doctor --json` reports that source with code `permission_denied` and the denied path without hiding other source diagnostics

#### Scenario: Unsupported Cursor format

- **WHEN** Cursor storage exists but does not match a supported latest macOS or
  Windows format
- **THEN** `doctor` reports `unsupported_format` with the inspected path and a
  concise reason
