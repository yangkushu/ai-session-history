## ADDED Requirements

### Requirement: Portable cross-agent Skill

The system SHALL provide one canonical `ai-history` Skill whose command workflow is
usable by Codex, Claude Code, and Cursor without duplicating CLI business logic.
Platform-specific permission guidance SHALL be stored as separate references that
the Skill loads only for the active host.

#### Scenario: Agent selects an ai-history command

- **WHEN** a supported agent needs to find, inspect, hand off, or export a local AI coding session
- **THEN** the Skill directs the agent to the corresponding `ai-history` CLI command and machine-readable JSON output where available

#### Scenario: Host needs permission guidance

- **WHEN** CLI execution or history access is denied by Codex, Claude Code, or Cursor
- **THEN** the Skill uses the matching host permission reference without changing CLI data-processing behavior

### Requirement: Safe project-first workflow

The Skill SHALL preflight the CLI with `version` and `doctor --json` before first use,
SHALL default listing and searching to the current project, and MUST obtain user
approval before expanding a no-result project search to all local history.

#### Scenario: Search current project by default

- **WHEN** a user asks an agent to find a prior session without specifying scope
- **THEN** the Skill uses `search --here --json` or `list --here --json` as appropriate

#### Scenario: Current project has no result

- **WHEN** the project-scoped list or search returns no matching session
- **THEN** the Skill asks whether to search all local history and does not remove the project scope until the user agrees

#### Scenario: One source is unavailable

- **WHEN** diagnostics report one source unavailable and at least one other source available
- **THEN** the Skill reports the unavailable source and continues with the available sources

### Requirement: Minimum-permission recovery

The Skill SHALL distinguish CLI execution, history read, and export write permission
failures. It MUST request only the command or path access needed for the current
operation, MUST NOT repeat an unchanged denied command, and MUST NOT instruct an
agent to bypass managed policy or disable permission checks.

#### Scenario: History read is denied

- **WHEN** `doctor` or another JSON result reports `permission_denied` with a path
- **THEN** the Skill reports that path and requests read-only access through the active host's supported permission mechanism

#### Scenario: Managed policy blocks access

- **WHEN** the active host reports that an administrator policy prevents the required access
- **THEN** the Skill stops and reports the policy boundary without proposing full access, `sudo`, `chmod 777`, or a permission bypass mode

### Requirement: Explicit raw export consent

The Skill MUST NOT execute raw export unless the user explicitly requests complete,
original, or raw content. When export mode is unspecified, the Skill SHALL disclose
that raw content may contain sensitive data and SHALL recommend clean mode. The Skill
MUST NOT overwrite an existing destination without explicit user approval.

#### Scenario: Export mode is unspecified

- **WHEN** a user asks an agent to export a session without selecting a content mode
- **THEN** the Skill explains the raw-data risk, recommends clean mode, and does not execute the CLI's default raw export

#### Scenario: User explicitly requests raw export

- **WHEN** a user explicitly requests a complete original or raw session export
- **THEN** the Skill may execute `export --mode raw` to an explicit destination

#### Scenario: Export destination already exists

- **WHEN** export reports that the destination exists
- **THEN** the Skill preserves the file and asks the user to choose another path or explicitly approve replacement

### Requirement: Standard cross-agent installation

The canonical Skill SHALL be discoverable and installable through the open `npx skills`
CLI for Codex, Claude Code, and Cursor. The system SHALL document manual copying of the
same canonical Skill as a fallback when Node.js or network access is unavailable.
Neither installation path SHALL claim to grant CLI execution or history access and
MUST NOT require a project-maintained permission-bypass installer.

#### Scenario: Install through the common CLI

- **WHEN** a user installs `ai-history` with `npx skills add` and selects Codex, Claude Code, or Cursor
- **THEN** the common CLI discovers the canonical Skill and targets the selected agent using its supported directory mapping

#### Scenario: Install without Node.js

- **WHEN** a user cannot or does not want to use `npx skills`
- **THEN** the documentation provides a manual copy fallback from the same `skills/ai-history` source directory

#### Scenario: Installation and runtime permission remain separate

- **WHEN** installation succeeds through the common CLI or manual fallback
- **THEN** the documentation still requires runtime permission preflight and does not represent Skill installation as sandbox, allowlist, managed policy, or OS permission authorization
