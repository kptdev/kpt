# Releasing

To cut a new kpt release perform the following:

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

Release artifacts such as binaries and images will be built automatically by Cloud Build in the
`kpt-dev` GCP project.  The binaries linked from the README.md docs will be automatically updated
because they point to the `latest` binaries which are updated for tagged releases.  Images are
also updated with the `latest` tag for tagged releases.
