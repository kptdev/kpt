# Releasing

## Steps

To cut a new kpt release perform the following:

- Ensure kpt is importing the latest dependent releases
  - [cli-utils](https://github.com/kubernetes-sigs/cli-utils/tree/master/release)
  - Within kustomize:
    [kyaml](https://github.com/kubernetes-sigs/kustomize/blob/master/releasing)
  - Within kustomize:
    [cmd/config](https://github.com/kubernetes-sigs/kustomize/blob/master/releasing)
  - Update `go.mod` file with correct versions of `cli-utils`, `kyaml`, and
    `cmd/config`
  - Run `make all` (which should update `go.sum` and run `go mod tidy`)
  - Create a `kpt` PR with previous `go.mod` and `go.sum` changes, and submit.
    [Example PR](https://github.com/GoogleContainerTools/kpt/pull/594)
- Fetch the latest master changes to a clean branch
  - `git checkout -b release`
  - `git fetch upstream`
  - `git reset --hard upstream/master`
- Tag the commit
  - `git tag v1.0.0-(alpha|beta|rc).*`
  - `git push upstream v1.0.0-(alpha|beta|rc).*`
- This will trigger a Github Action that will use goreleaser to make the
  release. The result will be a github release in the draft state and upload
  docker images to GCR.
  - Verify that the release looks good. If it does, publish the release through
    the github UI.
- Update the Homebrew release
  - `go run ./release/formula/main.go <tag>` (example: `go run ./release/formula/main.go v1.0.0-beta.31`)
  - `git add . && git commit -m "update homebrew to <tag>"`
  - create a PR for this change and merge it
  - [example PR](https://github.com/GoogleContainerTools/kpt/pull/331/commits/baf33d8ed214f2c5e106ec6e963ad736e5ff4d98#diff-d69e3adb302ee3e84814136422cbf872)

## Artifacts

Release artifacts such as binaries and images will be built automatically by the
Github Action. The binaries linked from the README.md docs will be automatically
updated because they point to the `latest` binaries which are updated for tagged
releases. Images created from the `main` branch will not be tagged with
`latest`.
