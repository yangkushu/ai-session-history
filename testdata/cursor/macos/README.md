# Cursor macOS Fixture

This directory documents the observed real storage shape of latest Cursor on
macOS. Tests build synthetic databases with this shape at runtime via helpers in
`internal/readers/cursor_test.go`; no real or binary fixture is committed.

Real `globalStorage/state.vscdb` shape (macOS latest, observed 2026-07-08):

- `composerHeaders` may exist but be empty.
- `cursorDiskKV` stores composer state under `composerData:<composerId>`.
- `composerData` JSON fields used by the reader:
  - `composerId`
  - `createdAt`, `updatedAt`, `lastUpdatedAt` in epoch milliseconds
  - `context.workspaceRootPath` or `context.workspaceFolderPath`
  - `conversation[]` messages with `role`, `text`, and `timestamp`

The minimized SQL sample in this directory captures the empty-composer shape
observed locally. Parser tests additionally synthesize a non-empty conversation
with the same table/key shape so `list`, `show`, and `context` behavior remains
covered without committing private history content.
