# Versioning

## SemVer

We use [semantic versioning] for kpt releases. Releases with a fully specified version
(e.g. `vX.Y.Z` for the CLI, `api/vX.Y.Z` for the Go API module) are immutable and will
never be changed.

We also use alpha and beta pre-releases (e.g. `v1.0.0-beta.65`) before content is fully
stabilized. See [release/README.md](release/README.md) for maintainer instructions on
cutting releases.

### Release surfaces

This repository publishes two independently versioned surfaces:

- **kpt CLI** (root Go module `github.com/kptdev/kpt`): release tags look like `v1.2.3`.
- **kpt API** (Go module `github.com/kptdev/kpt/api`): release tags look like `api/v1.2.3`.
  Consumers pin with `go get github.com/kptdev/kpt/api@v1.2.3` (see
  [Go modules: VCS version](https://go.dev/ref/mod#vcs-version)).

### Breaking Changes

We define a breaking change as: For any given valid input, kpt produces a different
result on a user-facing surface, or a previously supported input is no longer accepted.

### Backwards Compatibility

We follow semantic versioning, with one important difference from the
[semver specification]: **minor versions are not guaranteed to be backwards compatible.**

For post v1.0.0 versions, we will:

- Bump major version: There are breaking changes.

- Bump minor version: There are new features, improvements, or breaking changes.

- Bump patch version: There are only bug fixes and security fixes (e.g. dependency
  non-breaking version bumps).

For pre v1.0.0 versions, the major version is always `0` and we will:

- Bump minor version: There may be breaking changes.

- Bump patch version: In all other cases, including backward-compatible features, bug
  fixes and security fixes.

For CLI pre-releases tagged as `v1.0.0-beta.N`, treat each beta release as potentially
containing breaking changes until a stable `v1.0.0` release is published.

## Best Practices

- Pin the full SemVer for the kpt CLI in CI and production. Download a specific release
  binary or use a fully qualified container image tag (e.g. `ghcr.io/kptdev/kpt:v1.2.3`).
- Pin the Go module version explicitly, for example
  `go get github.com/kptdev/kpt/api@v1.2.3`.
- Read release notes before upgrading, especially when bumping the minor version.

[semantic versioning]: https://semver.org/
