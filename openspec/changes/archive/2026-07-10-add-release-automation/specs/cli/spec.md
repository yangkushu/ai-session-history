## ADDED Requirements

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
