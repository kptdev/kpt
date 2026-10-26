# Releasing

This document covers releases of the **root library** (`github.com/kptdev/kpt`), the **CLI** (`github.com/kptdev/kpt/cli`), and the **API module** ([`github.com/kptdev/kpt/api`](../api/)).

The CLI module depends on the root library, so the library tag has to exist before the CLI module can require that version. A paired release is two commits and two tags. Version numbers can match (`v1.2.3` and `cli/v1.2.3`); they still point at different commits.

When `api/` changed, tag that module first and bump the root `go.mod` before tagging the library.

## Root library (`github.com/kptdev/kpt`)

Pushing a tag such as `v1.2.3` publishes the root module. Consumers pin it with `go get github.com/kptdev/kpt@v1.2.3`. That tag does not build CLI binaries.

[.github/workflows/release.yml](../.github/workflows/release.yml) checks that the root `go.mod` has no `replace` directives and runs `make lib`.

## CLI (`github.com/kptdev/kpt/cli`)

After the root tag exists:

1. In [`cli/go.mod`](../cli/go.mod), set `require github.com/kptdev/kpt` to that version (and the API module to the version this release should use).
2. Remove any `replace` directives. The CLI workflow fails if one is still present.
3. Run `go mod tidy` in `cli/`.
4. Commit that change, then push a tag `cli/v1.2.3` (same version number as the library tag when this is a paired release). Pre-releases use the same suffix, for example `cli/v1.2.3-beta.1`.

[.github/workflows/release-cli.yml](../.github/workflows/release-cli.yml) runs tests, GoReleaser, container images, and provenance. The workflow passes the tag with the `cli/` prefix removed, so image tags and archive filenames stay `v1.2.3` / `1.2.3`. The GitHub Release is attached to the git tag, so download URLs look like:

```text
https://github.com/kptdev/kpt/releases/download/cli/v1.2.3/kpt_linux_amd64-1.2.3.tar.gz
```

`go install github.com/kptdev/kpt/cli@v1.2.3` uses that `cli/v1.2.3` tag.

## kpt API module (`github.com/kptdev/kpt/api`)

The [`api/`](../api/) directory is its own Go module. Consumers pin it with `go get`, for example:

```shell
go get github.com/kptdev/kpt/api@v1.0.0
```

That `@v1.0.0` form is the **module version**; the corresponding **git tag** in this repo is **`api/v1.0.0`**.

Tags must use the **`api/`** prefix so the version matches the module subdirectory (see [Go modules: VCS version](https://go.dev/ref/mod#vcs-version)).

### Release notes

GitHub's built-in "Generate release notes" includes PRs for the whole repository. For an API release, generate notes scoped to `api/` instead:

```shell
scripts/generate-folder-release-notes.sh --folder api --new-tag v1.2.3
```

That calls the GitHub Release Notes API, then keeps only bullets whose PRs also appear in `git log` for `api/` between the previous `api/v*` semver tag and `HEAD`.
Pass `--previous-tag` to pick a different start (version or full tag, for example `v1.2.2` or `api/v1.2.2`).
Use `-o FILE` to write the notes, and `--attribution` to prepend a note that they are folder-scoped.

Prerequisites: `gh` (authenticated to this repository) and a clone with tags fetched (`git fetch --tags`).

### What runs in CI

When you push a tag that matches **`api/v[0-9]+.[0-9]+.[0-9]+*`** (for example `api/v1.2.3`), [.github/workflows/release-api.yml](../.github/workflows/release-api.yml) runs: it uses the Go version from [`api/go.mod`](../api/go.mod) and runs **`make api`**.

### Monorepo

`api/` and `cli/` are nested modules, so a tag for one module does not publish the others. A module zip excludes nested modules. Local development can use a gitignored `go.work` (`make work`) so builds see the sibling modules without committed `replace` directives.
