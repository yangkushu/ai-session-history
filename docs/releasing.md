# Releasing

Maintainers publish release binaries by pushing a version tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions runs tests and GoReleaser on tags matching `v*`. Release builds
inject version metadata, so release binaries report the tag, commit, and build
date:

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
