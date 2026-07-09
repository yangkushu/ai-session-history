## 1. Tests First

- [x] 1.1 Add failing renderer tests for stable context section ordering and headings.
- [x] 1.2 Add failing renderer tests where setup boilerplate appears before the real user task.
- [x] 1.3 Add failing renderer tests for missing initial goal after boilerplate filtering.
- [x] 1.4 Add failing renderer tests for recent conversation selection and boilerplate exclusion.
- [x] 1.5 Add failing renderer tests for useful tool outcome inclusion and noisy output omission markers.
- [x] 1.6 Add failing max-chars tests that verify truncation markers and priority ordering.

## 2. Context Rendering

- [x] 2.1 Define narrow deterministic boilerplate filters for known injected setup shapes.
- [x] 2.2 Update initial-goal selection to use the first meaningful user task after filtering.
- [x] 2.3 Update context Markdown layout to use stable handoff sections.
- [x] 2.4 Preserve recent clean user/assistant turns while excluding setup boilerplate when safe.
- [x] 2.5 Preserve concise tool final results and errors while omitting large raw output.
- [x] 2.6 Add deterministic skipped, omitted, and truncated notes.
- [x] 2.7 Ensure context rendering stays bounded by `--max-chars`.

## 3. Documentation and Manual Acceptance

- [x] 3.1 Update README `context` examples and notes to describe the improved handoff shape.
- [x] 3.2 Add or update manual acceptance guidance for context handoff quality.
- [x] 3.3 Record any remaining feedback items that should stay outside this change.

## 4. Verification

- [x] 4.1 Run `gofmt` on changed Go files.
- [x] 4.2 Run `PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go test ./...`.
- [x] 4.3 Run representative manual `ai-history context` checks on at least one real local session.
- [x] 4.4 Run `openspec validate improve-context-handoff-quality --strict`.
- [x] 4.5 Commit and push with a concise Chinese commit message after verification.
