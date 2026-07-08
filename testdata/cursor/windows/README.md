# Cursor Windows Fixture

This directory documents the observed real storage shape of latest Cursor on
Windows. Tests build a synthetic database with this shape at runtime via the
`writeCursorState` helper in `internal/readers/cursor_test.go`; no real or
binary fixture is committed.

Real `globalStorage/state.vscdb` shape (Windows latest, observed 2026-07-08):

- `composerHeaders` table: one row per composer (Agent) conversation.
  - `composerId` (PRIMARY KEY), `workspaceId`, `createdAt` (ms), `lastUpdatedAt`
    (ms), `isArchived` (int), `isSubagent` (int), `recency`, `checkpointAt`,
    `value` (JSON).
  - `value` JSON fields used by the reader: `name`, `isArchived`,
    `createdAt`, `lastUpdatedAt`, `workspaceIdentifier.uri.path`.
- `cursorDiskKV` table: `key` (UNIQUE) + `value` (BLOB).
  - Conversation messages live under `bubbleId:<composerId>:<bubbleId>`.
  - Bubble `value` JSON fields used by the reader: `type` (1 = user,
    2 = assistant), `text`, `createdAt` (ISO 8601 string).
  - Bubbles without `text` carry internal state in an encrypted
    `conversationState` blob and content-addressed `agentKv:blob:` rows; P0
    does not parse them.

The reader opens the database with SQLite `immutable=1` because the file is
owned and live-updated by Cursor and, on a WSL host, sits on a Windows
filesystem mount where default read-only access fails with a disk I/O error.
