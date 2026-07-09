## Context

The CLI currently renders `context` as deterministic Markdown with session
metadata, an initial goal, recent conversation, and useful tool outcomes. That
baseline is enough to prove the feature works, but real sessions often start
with injected environment context, AGENTS/CLAUDE instructions, or local runtime
metadata before the user's actual task appears.

The next step is to make `context` a practical handoff artifact for another
agent while preserving the project constraints: local-first, read-only, and no
LLM-generated summaries.

## Goals / Non-Goals

**Goals:**

- Make context output easier to scan by using stable, predictable sections.
- Select the initial goal from the first meaningful user task rather than
  boilerplate or injected setup text.
- Preserve recent conversation and concise tool outcomes needed for a handoff.
- Mark skipped or omitted content clearly when that affects completeness.
- Keep output deterministic and bounded by `--max-chars`.

**Non-Goals:**

- No LLM summarization or inferred project status.
- No MCP adapter work.
- No Skills integration work.
- No import/export command work.
- No interactive/TUI behavior.
- No mutation of source histories.

## Decisions

### Keep context rendering deterministic

Continue rendering from normalized session turns with deterministic filtering.
Do not call a model to summarize, classify, or infer current state. This keeps
the CLI usable offline and prevents handoff output from inventing facts.

Alternative considered: add a generated summary section. This would be more
compact, but it conflicts with the current local-first boundary and would create
trust questions before the raw renderer contract is stable.

### Filter boilerplate before choosing the initial goal

Choose the initial goal from the first meaningful user turn after filtering
known boilerplate shapes such as environment context blocks, AGENTS/CLAUDE
instruction injections, empty turns, and command/runtime preambles. Keep this as
rule-based filtering over normalized message text.

Alternative considered: always use the literal first user turn. That is simple
and stable, but it makes context misleading when the first user turn is injected
setup rather than the user's request.

### Preserve evidence, not interpretation

Context may include session metadata, recent messages, concise tool final
results, tool errors, truncation markers, and skipped-content notes. It must not
add sections such as "current state" or "next steps" unless those words came
from source messages or deterministic metadata.

Alternative considered: derive a current-state summary from recent messages and
tool outcomes. That would be useful, but it crosses into interpretation without
a model or user validation.

### Prefer stable sections over new flags

Improve the default `context` output rather than adding a new mode flag in this
change. The existing command is already the handoff surface; adding a parallel
mode would increase the compatibility surface before the default is good.

Alternative considered: add `context --mode handoff`. That can be revisited if
future users need both legacy and rich layouts, but P1 has no released binary
compatibility burden yet.

## Risks / Trade-offs

- Boilerplate filters can over-filter real user content. Mitigation: keep rules
  narrow, fixture-driven, and transparent through skipped-content notes.
- Context can still omit useful details under tight `--max-chars` limits.
  Mitigation: preserve metadata and omission markers before lower-priority
  recent turns.
- Source readers may not expose enough tool metadata. Mitigation: use existing
  reliable normalized fields first and avoid speculative parsing.
- Stable section ordering may require updating snapshots or manual docs.
  Mitigation: update tests and README examples in the same change.

## Migration Plan

This is a non-breaking output-quality change. Existing `ai-history context`
commands continue to work. Scripts that parse Markdown headings may need to
adapt to improved section names or ordering before a formal release.

Rollback is straightforward: revert the context renderer changes and associated
tests if the filtering rules hide important real-world content.

## Open Questions

- Should skipped boilerplate be shown as a count only, or should the first
  skipped reason be included for debuggability?
- Should `context` reserve budget for recent user turns over assistant turns, or
  preserve strict reverse chronological order within the recent conversation?
- Should future release builds offer a machine-readable `context --json` shape,
  or keep JSON limited to `show --json` for normalized detail?
