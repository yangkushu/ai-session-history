## Context

`show --json` writes a bounded, inspection-oriented normalized detail to
standard output. `context` deliberately filters and bounds content for an
agent handoff. Neither command creates a durable complete-session artifact.
The existing `export` dispatcher branch therefore still reports a P0
unavailable-command error.

The new capability must remain local and read-only, preserve the existing
source-neutral model, and avoid accidental disclosure or destructive file
replacement.

## Goals / Non-Goals

**Goals:**

- Export one source-prefixed session ID to an explicit local file.
- Provide a stable `session-export.v1` JSON envelope and a full Markdown
  rendering of the same normalized content.
- Export all turns without the configured detail/context character limits.
- Reuse `raw`, `clean`, and `summary` semantics, with `raw` as the default.
- Write new files atomically with owner-only permissions and protect existing
  paths unless the caller supplies `--force`.

**Non-Goals:**

- Import, restore, resume, or modify source-owned history.
- Emit source-specific storage records or unknown private fields.
- Add automatic redaction, remote transfer, compression, encryption, indexing,
  or multi-session archive files.
- Add a new truncation flag; a complete export must not silently become
  incomplete.

## Decisions

### Build a versioned export model before encoding

`render` will define a `SessionExport` model with `schema_version`,
`exported_at`, `content_mode`, and a normalized `core.SessionDetail`. A single
builder will apply the selected mode to every turn without applying a character
budget. JSON encodes this model directly; Markdown renders this same model as
metadata followed by all turns.

This keeps JSON and Markdown semantically aligned and makes the JSON contract
safe for later consumers. Directly reusing `show --json` was rejected because
that path applies the configured detail limit and has no export metadata.

### Fetch export detail from the core service directly

`appService.Export` will call `core.Service.Show` rather than `appService.Show`.
The latter deliberately applies `detail_chars`; the former returns the full
normalized reader result. The export builder then applies only the requested
content mode. This preserves the existing source normalization while making
unbounded export behavior explicit.

### Keep file policy at the CLI boundary

`runExport` parses `--output/-o`, `--format/-f`, `--mode/-m`, and `--force`.
It serializes the export and writes a temporary file in the destination
directory with `0600` permissions, then renames it into place. Without
`--force`, an existing destination is rejected before any write. Temporary
files are removed on failure.

The service and renderer remain testable without filesystem concerns. Writing
directly to the destination was rejected because a write failure could leave a
partial session export.

### Preserve existing content-mode meanings

`raw` preserves all normalized turn text, while `clean` and `summary` reuse the
existing omission behavior for tool content. This does not expose raw
Codex/Claude/Cursor records: normalization happens in readers before the export
model is built. A separate export-specific mode was rejected because it would
duplicate user-facing semantics.

## Risks / Trade-offs

- [Full exports can contain secrets or personal data] → Require an explicit
  output path, create files with `0600`, and document that exports are local
  sensitive artifacts.
- [Large sessions create large files] → Preserve completeness in the first
  version; users can select `clean` or `summary` rather than relying on hidden
  truncation.
- [JSON consumers depend on the schema] → Start with an explicit version and
  permit future compatible additive fields only.
- [Atomic rename behavior differs across platforms] → Create the temporary file
  in the output directory and cover error/overwrite behavior with tests.

## Migration Plan

This is a new P1 command. The former `export is not available in P0` response
is replaced in place; `import` remains unavailable. No data migration or
rollback action is required. Removing the feature later restores the prior
unavailable-command response without altering source histories.

## Open Questions

None.
