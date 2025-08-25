# Releasing

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

