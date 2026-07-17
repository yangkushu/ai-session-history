## 1. Formatter Tests

- [x] 1.1 Add failing table-driven tests for relative ages at every unit boundary,
  floored values, and future timestamps using a fixed current time.
- [x] 1.2 Add failing tests for local absolute timestamps, independent created and
  updated ages, and `unknown` timestamp handling.
- [x] 1.3 Add failing tests for whitespace normalization and 80-cell truncation of
  ASCII, CJK, emoji, and combining-character titles.

## 2. Text Renderer Implementation

- [x] 2.1 Add the focused Unicode display-width dependency or an equivalently
  tested internal implementation.
- [x] 2.2 Implement isolated relative-age, local-time, and title-formatting helpers
  until the formatter tests pass.
- [x] 2.3 Add failing renderer tests for four-line alignment, complete CWD and ID,
  missing CWD, multiple-block spacing, and the final newline contract.
- [x] 2.4 Implement the deterministic session-block renderer until its tests pass.

## 3. CLI Integration and Compatibility

- [x] 3.1 Add a failing CLI test proving human-readable `list` uses the new block
  renderer with updated time before created time.
- [x] 3.2 Route only non-JSON `list` output through the renderer and keep filtering,
  ordering, limits, and empty output behavior unchanged.
- [x] 3.3 Extend JSON regression tests to prove the existing `ListResult` shape and
  timestamp values remain unchanged.

## 4. Documentation and Verification

- [x] 4.1 Update README examples and release-facing guidance to describe the new
  text layout and direct machine consumers to `--json`.
- [x] 4.2 Run focused CLI tests, the complete Go test suite, formatting checks,
  `git diff --check`, and `openspec validate improve-list-time-display`.
- [x] 4.3 Build the CLI and manually inspect representative current-project and
  all-history text output for CJK alignment, long values, and local timestamps.
