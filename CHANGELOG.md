# Changelog

All notable user-facing changes to this project are documented here.

This project follows semantic versioning where practical.

## Unreleased

## 0.3.0 - 2026-07-13

### Added

- Added complete, versioned session exports in JSON or Markdown through the
  `export` command, with `raw`, `clean`, and `summary` content modes and explicit
  overwrite protection.

## 0.2.0 - 2026-07-12

### Added

- Added this changelog.
- Added a regular CI workflow for push, pull request, and manual checks.
- Added local session search through the `search` command, with text and JSON
  output, source and directory scopes, and deterministic relevance ranking.
- Added versioned JSON handoff output for the `context` command through
  `--json` / `-j`.

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
