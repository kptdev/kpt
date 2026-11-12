# Releasing

kpt consists of three Go modules that are released together with **one locked version**:

| Module | Path | Git tag |
|---|---|---|
| API | [`github.com/kptdev/kpt/api`](../api/) | `api/v1.2.3` |
| Root library | `github.com/kptdev/kpt` | `v1.2.3` |
| CLI | [`github.com/kptdev/kpt/cli`](../cli/) | `cli/v1.2.3` |

The three tags point at different commits, because each module requires the previous one:
the root `go.mod` requires `api/v1.2.3` and `cli/go.mod` requires `v1.2.3` and `api/v1.2.3`.
Consumers pin modules with `go get github.com/kptdev/kpt/api@v1.2.3`; the `api/` and `cli/` prefixes only
exist on the git tags (see [Go modules: VCS version](https://go.dev/ref/mod#vcs-version)).

## Creating a release

Run the [kpt multimodule release](../.github/workflows/release.yml) workflow (Actions, "Run workflow"):

| Input | Description |
|---|---|
| `version` | New version, for example `v1.2.3` or `v1.2.3-pre.1`. Must be valid semver and newer than **all** existing `v*`, `api/v*` and `cli/v*` tags. |
| `branch` | Branch to release from (default `main`). |
| `dry_run` | Run everything locally in the runner, but push and publish nothing. Use it to try a release. |

The workflow:

1. Validates the version (semver, none of the three tags exist, newer than every existing release) and the branch.
2. Fails if any `go.mod` has a `replace` directive, or requires a kpt module at a pseudo-version
   (for example `v1.1.1-0.20260101000000-abcdef123456`). Pseudo-versions of other modules are fine.
3. Runs `make all` and `make test-docker` on the branch.
4. Tags `api/v1.2.3`, bumps the api requirement in the root `go.mod` and commits, tags `v1.2.3`, bumps the kpt and api
   requirements in `cli/go.mod` and commits, tags `cli/v1.2.3`. The checks of step 2 run again on the result and the CLI is built.
5. Builds the CLI archives, Linux packages and checksums with GoReleaser ([`release/tag/goreleaser.yaml`](tag/goreleaser.yaml)).
6. Pushes the branch and the three tags in a single atomic push.
7. Creates the GitHub release **named `v1.2.3`, attached to the `cli/v1.2.3` tag**, with all artifacts, and uploads SLSA provenance.
   Versions with a pre-release suffix are marked as pre-releases.

Nothing is pushed before the build and tests have succeeded. The steps are implemented in
[`scripts/multimodule-release/`](../scripts/multimodule-release/) (usage is documented in the header of `release.sh`).

### Requirements

- The `RELEASE_TOKEN` repository secret: a GitHub App installation token or PAT that may push to the (protected) release branch and
  create tags. The default `GITHUB_TOKEN` is not sufficient for protected branches.
- The branch must not receive other commits while the workflow runs. If it does, the atomic push is rejected and nothing is published;
  run the workflow again.

### Release notes

The release notes are the GitHub generated notes for the whole repository since the previous `v*` release, followed by the
changes scoped to `api/` and `cli/` since their previous tags. GoReleaser's changelog is disabled, because the open-source
version cannot scope changelogs to a module (no `monorepo` support).

[`scripts/generate-folder-release-notes.sh`](../scripts/generate-folder-release-notes.sh) generates the folder scoped notes. It
calls the GitHub Release Notes API, then keeps only bullets whose PRs also appear in `git log` for the folder:

```shell
scripts/generate-folder-release-notes.sh --folder api --new-tag v1.2.3
```

Prerequisites: `gh` (authenticated to this repository) and a clone with tags fetched (`git fetch --tags`).

### Download URLs

Because the release is attached to the CLI tag, URLs look like:

```text
https://github.com/kptdev/kpt/releases/download/cli/v1.2.3/kpt_linux_amd64-1.2.3.tar.gz
```

`go install github.com/kptdev/kpt/cli@v1.2.3` uses the `cli/v1.2.3` tag.

## When something fails

- Before the push (validation, checks, tests, build): nothing was published. Fix the problem and run the workflow again with the same version.
- After the push (release creation or provenance failed): the tags exist and **must not be reused**. Either create the missing
  release manually from the `cli/v1.2.3` tag, or delete the three tags and the bump commits and run again. Do not delete tags
  that were already fetched by the Go module proxy; release a new version instead.

## Local development

`api/` and `cli/` are nested modules, so a tag for one module does not publish the others. A module zip excludes nested modules.
Local development can use a gitignored `go.work` (`make work`) so builds see the sibling modules without committed `replace`
directives.

To test the version logic: `scripts/multimodule-release/test-semver.sh`.
