## 1. Tests First

- [x] 1.1 Add a failing CLI test for injected release version, commit, and build date output.
- [x] 1.2 Add a failing repository test or check for release workflow and GoReleaser config presence.

## 2. Release Configuration

- [x] 2.1 Add `.goreleaser.yaml` for cross-platform `ai-history` archives and checksums.
- [x] 2.2 Configure GoReleaser `ldflags` for `internal/cli.version`, `internal/cli.commit`, and `internal/cli.buildDate`.
- [x] 2.3 Add `.github/workflows/release.yaml` with `v*` tag and manual triggers.
- [x] 2.4 Ensure the workflow checks out full history, sets up Go 1.22, runs tests, and runs GoReleaser.

## 3. Documentation

- [x] 3.1 Update README with install-from-release guidance.
- [x] 3.2 Document how maintainers publish a release tag.
- [x] 3.3 Document local snapshot validation before pushing a tag.

## 4. Verification

- [x] 4.1 Run `gofmt` on changed Go files.
- [x] 4.2 Run `PATH="$PATH:/usr/local/go/bin" GOCACHE=/tmp/go-build go test ./...`.
- [x] 4.3 Run a local build with release-style `ldflags` and verify `ai-history version` output.
- [x] 4.4 Run `openspec validate add-release-automation --strict`.
- [x] 4.5 Run `git diff --check`.
- [x] 4.6 Commit and push with a concise Chinese commit message.
