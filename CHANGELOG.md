# Changelog

All notable user-facing changes to this project are documented here.

This project follows semantic versioning where practical.

## Unreleased

### Added

- Added this changelog.
- Added a regular CI workflow for push, pull request, and manual checks.

### Changed

- Streamlined the English and Chinese README files.
- Moved maintainer release notes, source storage details, and project history
  notes into `docs/`.

## 0.1.0 - 2026-07-10

### Added

- Initial `ai-history` CLI release.
- Added `doctor`, `list`, `show`, `context`, and `version` commands.
- Added local session readers for Codex, Claude Code, and Cursor.
- Added deterministic Markdown handoff context generation.
- Added release automation with GoReleaser for Linux, macOS, and Windows
  binaries plus checksums.
