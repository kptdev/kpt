# Releasing

This document covers releases of the **kpt CLI and library** (root Go module `github.com/kptdev/kpt`) and the separate **Go API module** [`github.com/kptdev/kpt/api`](../api/).

## Main kpt release

To cut a new kpt release perform the following:

- Check that dependencies are up to date and that all required release content is in the main branch
- Navigate to [the project release page](https://github.com/kptdev/kpt/releases) and select "draft new release"
- Leave the target as "main", and create a new tag to match the release version
  - Versioning follows [semantic versioning rules](http://semver.org/)
  - Alpha and beta versions are used to make releases before content is fully stabilized
  - Increment the number after "alpha" or "beta" by one when making this type of release - e.g. v1.0.0-beta.58 could come after v1.0.0-beta.57
- Release title should be left blank - it will be auto-filled from the tag
- Click "Generate release notes" to auto-generate the content of the release. Edit this as appropriate to add extra context
- If the release is an alpha or beta release and there is already a stable version available, the "set as a pre-release" check-box should be checked. Otherwise, leave it checked as "set as the latest release"
- Check the "create a discussion for this release" check-box
- Click "publish" and then verify that the github action has run and the artefacts have been produced

Pushing a root-level version tag (for example `v1.2.3`) runs the [kpt Release](../.github/workflows/release.yml) workflow (GoReleaser, container images, and provenance).

## kpt API module (`github.com/kptdev/kpt/api`)

The [`api/`](../api/) directory is its own Go module. Consumers pin it with `go get`, for example:

```shell
go get github.com/kptdev/kpt/api@v1.0.0
```

That `@v1.0.0` form is the **module version**; the corresponding **git tag** in this repo is **`api/v1.0.0`**.

Tags must use the **`api/`** prefix so the version matches the module subdirectory (see [Go modules: VCS version](https://go.dev/ref/mod#vcs-version)).

### What runs in CI

When you push a tag that matches **`api/v[0-9]+.[0-9]+.[0-9]+*`** (for example `api/v1.2.3`), [.github/workflows/release-api.yml](../.github/workflows/release-api.yml) runs: it uses the Go version from [`api/go.mod`](../api/go.mod) and runs **`make api`**.

### Monorepo and root `go.mod`

The root module uses `replace github.com/kptdev/kpt/api => ./api` so local and CI builds use the workspace copy.
A tagged archive for `github.com/kptdev/kpt@<version>` still includes `./api` at that commit.
