## 1. Handoff Model And Rendering

- [x] 1.1 Add a structured handoff model for schema version, session metadata, initial goal, recent conversation, tool outcomes, structured handoff notes, instruction, and truncation state.
- [x] 1.2 Refactor Markdown context rendering to build from the structured handoff model while preserving default Markdown behavior.
- [x] 1.3 Implement deterministic JSON handoff encoding with `schema_version: "context-handoff.v1"` and empty arrays instead of null collections.
- [x] 1.4 Apply existing setup filtering, tool outcome filtering, structured note codes/messages, and JSON content-budget truncation behavior through the shared handoff model.

## 2. CLI Integration

- [ ] 2.1 Add `--json` and `-j` support to the `context` command.
- [ ] 2.2 Route `context --json` through the structured handoff renderer instead of returning Markdown.
- [ ] 2.3 Keep `context` Markdown as the default output and keep `show --json` behavior unchanged.
- [ ] 2.4 Ensure config path, target cwd, max chars, and session lookup errors behave consistently with existing context command behavior.

## 3. Tests And Documentation

- [x] 3.1 Add renderer tests for JSON shape, schema version, initial goal availability, empty collections, omitted optional metadata, tool outcomes, structured note codes/messages, and truncation notes.
- [ ] 3.2 Add CLI tests for `context --json`, `context -j`, and default Markdown output.
- [ ] 3.3 Update README documentation for `context --json` and clarify the difference between `show --json` and `context --json`.
- [ ] 3.4 Run `GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./...` and `openspec validate add-context-json-handoff --strict`.
