## Context

The CLI can be built locally and already exposes development-build version
output. The product direction document calls for prebuilt binary distribution,
checksums, and version metadata, but the repository currently has no release
workflow or GoReleaser configuration.

The user confirmed that GitHub Free can trigger builds and approved adding tag
based release automation.

## Goals / Non-Goals

**Goals:**

- Publish release archives from pushed `v*` tags.
- Support manual workflow dispatch for release validation or re-runs.
- Build `ai-history` binaries for Linux, macOS, and Windows on `amd64` and
  `arm64`.
- Inject `version`, `commit`, and `buildDate` into CLI output.
- Generate checksums for artifacts.
- Document release usage and local snapshot validation.

**Non-Goals:**

- No Homebrew tap or package-manager publishing.
- No code signing, notarization, SBOM, or container images.
- No automatic release on every branch push.
- No release tag creation by CI; maintainers create tags deliberately.

## Decisions

### Use GoReleaser for artifact generation

Use GoReleaser because it is the standard tool for Go CLI release archives,
checksums, changelogs, and GitHub Releases. It keeps the workflow small while
still producing consistent cross-platform artifacts.

Alternative considered: hand-written `go build` matrix plus upload-artifact
steps. That is simpler initially but duplicates archive/checksum/version logic
and becomes harder to maintain as platforms are added.

### Trigger releases only from `v*` tags

Configure GitHub Actions to run on tags matching `v*` and through
`workflow_dispatch`. Avoid releasing on branch pushes so normal development does
not consume release minutes or create accidental artifacts.

Alternative considered: build snapshots on every `master` push. That is useful
later for nightly artifacts, but it is unnecessary for the first release path.

### Inject version metadata into `internal/cli`

The version variables live in `github.com/yangkushu/ai-session-history/internal/cli`.
GoReleaser `ldflags` must target that package path, not `main`. Release builds
will set:

- `internal/cli.version` from the tag version
- `internal/cli.commit` from the commit SHA
- `internal/cli.buildDate` from the commit date

### Keep CGO enabled status explicit

The project uses `modernc.org/sqlite`, so release builds can run with
`CGO_ENABLED=0`. This simplifies cross-compilation and keeps archives portable.

## Risks / Trade-offs

- GitHub-hosted runners may fail if dependencies require unexpected CGO support.
  Mitigation: set `CGO_ENABLED=0` and run local snapshot validation.
- First release may need repository settings adjustment for workflow write
  permissions. Mitigation: set workflow `permissions.contents: write` and
  document that GitHub Actions must be enabled.
- Free private repositories have limited Actions minutes. Mitigation: trigger
  only on tags and manual dispatch.
- Windows archives may need `.zip` behavior. Mitigation: rely on GoReleaser
  archive defaults unless local validation shows a problem.

## Migration Plan

This is additive. Existing local build and install commands continue to work.
After merge, maintainers can test locally with a snapshot build, then publish a
release by pushing a tag such as `v0.1.0`.

Rollback is straightforward: remove the workflow and GoReleaser config before a
release tag is pushed, or delete a faulty GitHub Release after correcting the
configuration.

## Open Questions

- Should the first public tag be `v0.1.0` or a lower preview such as `v0.0.1`?
- Should package-manager distribution begin with Homebrew in a separate change?
