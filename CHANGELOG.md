# Changelog

All notable user-facing changes to this project are documented here.

This project follows semantic versioning where practical.

## Unreleased

### Added

- Added Pi session history support for formats v1–v3, including active-branch
  browsing, search, export, and Markdown/JSON handoffs with persisted summaries.

### Fixed

- Excluded Pi-subagents transcript artifacts from Pi session discovery without
  hiding warnings for other malformed JSONL files.
- Made Windows installer checksum verification independent of PowerShell module
  auto-loading and updated macOS release builds for current macOS compatibility.

## 0.5.0 - 2026-07-17

### Added

- Added one-step Unix and Windows installers for the CLI and optional Agent
  Skill, with version pinning, checksum verification, and safe replacement.

### Changed

- Improved session list readability with formatted timestamps, titles, and
  session blocks.

## 0.4.0 - 2026-07-13

### Added

- Added the cross-Agent `ai-history` Skill for discovering and handing off
  coding sessions through the CLI.

### Fixed

- Improved Claude Code project traversal compatibility and history path
  permission diagnostics.

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
