# Releasing

## Steps

To cut a new kpt release perform the following:

- Ensure kpt is importing the latest dependent releases
  - [cli-utils](https://github.com/kubernetes-sigs/cli-utils/tree/master/release)
  - Within kustomize: [kyaml](https://github.com/kubernetes-sigs/kustomize/blob/master/releasing)
  - Within kustomize: [cmd/config](https://github.com/kubernetes-sigs/kustomize/blob/master/releasing)
  - Update `go.mod` file with correct versions of `cli-utils`, `kyaml`, and `cmd/config`
  - Run `make all` (which should update `go.sum` and run `go mod tidy`)
  - Create a `kpt` PR with previous `go.mod` and `go.sum` changes, and submit. [Example PR](https://github.com/GoogleContainerTools/kpt/pull/594)
- Fetch the latest master changes to a clean branch
  - `git checkout -b release`
  - `git fetch upstream`
  - `git reset --hard upstream/master`
- Tag the commit
  - `git tag v0.MINOR.0`
  - `git push upstream v0.MINOR.0`
- Update the Homebrew release
  - `go run ./release/formula/main.go v0.MINOR.0`
  - `git add . && git commit -m "update homebrew to v0.MINOR.0"`
  - create a PR for this change and merge it
  - [example PR](https://github.com/GoogleContainerTools/kpt/pull/331/commits/baf33d8ed214f2c5e106ec6e963ad736e5ff4d98#diff-d69e3adb302ee3e84814136422cbf872)

## Artifacts

Release artifacts such as binaries and images will be built automatically by Cloud Build in the
`kpt-dev` GCP project.  The binaries linked from the README.md docs will be automatically updated
because they point to the `latest` binaries which are updated for tagged releases.  Images are
also updated with the `latest` tag for tagged releases.

- `kpt-dev` release buckets
  - `gs://kpt-dev/latest`
  - `gs://kpt-dev/releases`

# Testing the Release Process

## Running Cloud Build Locally

You can use [`cloud-build-local`](https://github.com/GoogleCloudPlatform/cloud-build-local)
to run kpt's Cloud Build builds locally with custom parameters (`--substitutions`)
and dry-runs (`--dryrun`) to validate the builds syntax.

You will need to provide `--substitutions` for `TAG_NAME`, `_VERSION`,
`_GCS_BUCKET` and `_GITHUB_USER`. In a `--dryrun` these do not need to align
with existing resources. For example:

```sh
cloud-build-local --config=release/tag/cloudbuild.yaml --substitutions=TAG_NAME=test,_VERSION=test,_GCS_BUCKET=test,_GITHUB_USER=test --dryrun=true .
```

## Dry-Run Goreleaser

To test local changes to the [`goreleaser.yaml`](./tag/goreleaser.yaml) config. You may
[install goreleaser](https://goreleaser.com/install/) locally and provide the
`--skip-verify --skip-publish` flags.

From the kpt directory you would run:

```sh
goreleaser release --skip-validate --skip-publish -f release/tag/goreleaser.yaml
```

The resulting release artifacts will be stored in the `./dist` directory.