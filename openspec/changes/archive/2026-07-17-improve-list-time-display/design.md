## Context

`core.SessionSummary` already carries `CreatedAt` and `UpdatedAt`, and list
results are sorted by updated time with created time as a fallback. The CLI's
JSON output exposes both timestamps, but the current human-readable renderer
emits only ID, source, title, and CWD as one tab-separated line. Real titles can
contain line breaks or long pasted prompts, while paths and source-prefixed IDs
must remain copyable.

The human-readable output is for people. JSON remains the stable machine
interface. Rendering must be deterministic in terminals and pipes and must not
inspect terminal width or emit ANSI styling.

## Goals / Non-Goals

**Goals:**

- Show updated and created local timestamps, each with a compact relative age.
- Keep title, timestamps, CWD, and full session ID visually separated and
  aligned in a compact session block.
- Preserve complete CWD and session ID values.
- Keep malformed, missing, future, multiline, and wide-Unicode input from
  breaking the layout.
- Make time and title formatting independently testable with a fixed clock and
  location.

**Non-Goals:**

- Changing JSON fields or encoding.
- Changing default scope, source discovery, filtering, sorting, or limits.
- Adding color, an interactive picker, terminal-width detection, or a new flag.
- Implementing or revalidating desktop and CLI history support for Claude Code,
  Codex, or Cursor; that work remains a separate source-compatibility review.

## Decisions

### Render one deterministic four-line block per session

The text renderer will emit the source and normalized title on the first line,
updated then created timestamps on the second, full CWD on the third, and full
source-prefixed session ID on the fourth. Lines two through four start at the
same content column as the title. Consecutive blocks have exactly one blank
line; output ends after the final ID line's newline without an additional blank
line.

Example:

```text
codex   A normalized session title…
        2026-07-17 10:04 (20m)  2026-07-16 19:28 (15h)
        /workspace/ai-session-history
        codex:019f6aaf-29f9-7023-a67f-32ba88094b8e
```

The content column is based on the widest source label in the current result,
plus two spaces. This keeps every block aligned without a fixed source-specific
branch. A plain formatter is used for TTY and redirected output alike.

Alternatives considered:

- A single wide table was rejected because two timestamps, CWD, ID, and a
  Unicode title exceed common terminal widths.
- Per-field labels were rejected because fixed ordering is sufficient and the
  repeated `created`/`updated` text adds noise.
- TTY-only color and adaptive widths were rejected because redirected output
  would differ and snapshots would be less predictable.

### Format absolute and relative times together

Each present timestamp is converted to the process local location and rendered
as `YYYY-MM-DD HH:mm (<age>)`. Updated time is always first. A missing timestamp
is rendered as `unknown` in its position with no relative suffix.

Relative age uses elapsed duration from an injected `now`, floors to an integer,
and uses these ranges:

- future or less than one minute: `now`
- less than one hour: `Nm`
- less than one day: `Nh`
- less than 30 days: `Nd`
- less than 365 days: `Nmo`, using 30-day months
- otherwise: `Ny`, using 365-day years

Created and updated ages are calculated independently. This intentionally uses
rough duration units rather than calendar arithmetic because the display is a
recency aid, not a second authoritative timestamp.

### Normalize and truncate titles by terminal display width

All title whitespace, including newlines and tabs, is collapsed to single ASCII
spaces. The normalized title is truncated to at most 80 terminal display cells,
including a trailing Unicode ellipsis when truncation occurs. CWD and session ID
are never truncated. A focused Unicode display-width implementation or library
must account for CJK and emoji rather than counting bytes or runes.

### Isolate human rendering from the JSON path

`runList` will keep returning `ListResult` as it does today. The `--json` branch
continues to call the existing JSON writer unchanged. The text branch delegates
to a small list renderer whose clock and location can be controlled in tests;
relative-age, timestamp, title, and block rendering behavior are tested at their
own boundaries.

## Risks / Trade-offs

- **Text output is a breaking change for ad-hoc parsers** → Mark it explicitly
  and retain `--json` as the stable machine interface.
- **Four lines increase vertical space** → Keep labels out, align content, and
  use only one blank line between sessions.
- **Unicode width rules are non-trivial** → Use a focused, maintained width
  implementation and test CJK, emoji, combining characters, and the ellipsis.
- **Local timestamps vary by machine** → Inject the location in formatter tests
  and test the CLI contract without assuming the developer machine's zone.
- **Very long paths still wrap visually** → Preserve them intentionally because
  copyability is more important than forcing every block to four physical
  terminal rows.

## Migration Plan

Release the new human-readable renderer with release notes that direct scripts
to `ai-history list --json`. No data migration is needed. Rollback consists of
reverting the text-renderer change; normalized data and JSON are unaffected.

## Open Questions

None.
