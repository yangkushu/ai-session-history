## Why

Manual local builds are enough for development, but they make real adoption and
version verification clumsy. The CLI already exposes version fields, so the next
step is to publish reproducible release artifacts from tags and inject release
metadata into binaries.

## What Changes

- Add GoReleaser configuration for `ai-history` release builds.
- Add a GitHub Actions workflow that runs tests and publishes GitHub Release
  artifacts when a `v*` tag is pushed, plus a manual workflow trigger.
- Build platform archives for Linux, macOS, and Windows on `amd64` and `arm64`
  where supported by the current pure-Go build.
- Generate checksums for release artifacts.
- Inject `version`, `commit`, and `buildDate` into the CLI via Go `ldflags`.
- Update README with release install, tag publishing, and local snapshot
  validation instructions.
- Keep package-manager distribution such as Homebrew outside this change.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `cli`: Add release artifact and version metadata requirements for the
  published `ai-history` CLI.

## Impact

- Affected files: `.goreleaser.yaml`, `.github/workflows/release.yaml`,
  `README.md`, and CLI tests for injected version metadata.
- Affected systems: GitHub Actions and GitHub Releases.
- No new runtime dependency is added to the CLI binary.
- Release automation depends on repository tags and GitHub-provided
  `GITHUB_TOKEN` permissions.
