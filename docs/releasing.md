# Releasing

Maintainers publish release binaries by pushing a version tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions runs tests and GoReleaser on tags matching `v*`. Release builds
use Go 1.26 so macOS binaries include Mach-O `LC_UUID` (required by newer
macOS); the Linux CI job still checks the module's Go 1.22 minimum. Release
binaries report the tag, commit, and build date:

```bash
ai-history version
```

Validate the release configuration locally before pushing a tag:

```bash
goreleaser check
goreleaser release --snapshot --clean
```

Snapshot builds write artifacts under `dist/` and do not publish a GitHub
Release.
