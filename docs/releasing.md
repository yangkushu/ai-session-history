# Releasing

Maintainers publish release binaries by pushing a new version tag (replace
`vX.Y.Z` with the chosen unused version):

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

GitHub Actions runs tests and GoReleaser on tags matching `v*`. Release builds
use Go 1.26 so macOS binaries include Mach-O `LC_UUID` (required by newer
macOS); the Linux CI job still checks the module's Go 1.22 minimum. Release
binaries report the tag, commit, and build date:

```bash
ai-history version
```

Before tagging, confirm that the tag is unused, `master` is clean and its CI
is green, and move the release notes from `Unreleased` to the new version and
date in `CHANGELOG.md`. Validate the release configuration locally:

```bash
goreleaser check
goreleaser release --snapshot --clean
```

Snapshot builds write artifacts under `dist/` and do not publish a GitHub
Release. Snapshot versions use the synthetic next-patch label from the previous
tag (for example, `0.5.1-next` after `v0.5.0`), not the intended release
version. Check the expected platform archives, `checksums.txt`, and binary
version metadata before pushing the tag. After GitHub Actions publishes the
release, verify the uploaded assets and an installer download; do not treat a
green tag workflow alone as a completed release.
