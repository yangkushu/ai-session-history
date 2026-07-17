## Why

The human-readable `ai-history list` output omits session timestamps even though
the normalized summaries already contain them. Users need a compact view that
shows when each session was created and last updated, while keeping long paths
and source-prefixed session IDs easy to copy.

## What Changes

- **BREAKING** Replace the current one-line tab-separated human-readable list
  output with a four-line session block separated by a blank line.
- Show updated time before created time, with each absolute local timestamp
  followed by a compact relative age.
- Put the full working directory and full session ID on separate aligned lines.
- Normalize and display-width-truncate titles while preserving full paths and
  IDs.
- Keep `--json`, filtering, ordering, limits, and source-reading behavior
  unchanged; scripts should use JSON as the stable machine interface.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cli`: Change the human-readable `list` rendering contract to include compact
  absolute and relative timestamps and structured multi-line session blocks.

## Impact

- Affects the `list` text renderer in `internal/cli` and its tests.
- Adds reusable formatting logic for relative ages, local timestamps, title
  normalization, and terminal display-width truncation.
- May add a focused Unicode display-width dependency; source readers and stored
  history remain unchanged.
- Existing consumers that parse the legacy tab-separated text must migrate to
  `ai-history list --json`.
