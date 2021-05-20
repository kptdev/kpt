# How to Contribute

We'd love to accept your patches and contributions to this project. There are
just a few small guidelines you need to follow.

## Contributor License Agreement

Contributions to this project must be accompanied by a Contributor License
Agreement. You (or your employer) retain the copyright to your contribution;
this simply gives us permission to use and redistribute your contributions as
part of the project. Head over to <https://cla.developers.google.com/> to see
your current agreements on file or to sign a new one.

You generally only need to submit a CLA once, so if you've already submitted one
(even if it was for a different project), you probably don't need to do it
again.

## Code reviews

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Consult [GitHub Help] for more
information on using pull requests.

## Community Guidelines

This project follows [Google's Open Source Community Guidelines] and a [Code of
Conduct].

## Style Guides

Contributions are required to follow these style guides:

- [Error Message Style Guide]
- [Documentation Style Guide]

## Building the Source

1. Clone the project

   ```shell
   git clone https://github.com/GoogleContainerTools/kpt
   cd kpt
   ```

2. Build `kpt` to `$(go env GOPATH)/bin/kpt`

   ```shell
   make
   ```

3. Run test

   ```shell
   make all
   ```

## Contribution to the Functions Catalog

Refer to the
[documentation in the kpt-functions-catalog](https://github.com/GoogleContainerTools/kpt-functions-catalog/blob/master/CONTRIBUTING.md)
repo.

## Contributing to Documentation

If you are updating the documentation, please do it in separate PRs from code
changes and PR description should start with `Docs:`.

### Run the docs locally

Make docs changes and test them by running the site in a docker container with
`make site-run-server`.

It's usually a good idea to test locally for the following:

- Broken links
- Rendering static content

### Update docs

Docs are under [site/] and use [docsify] to present the source markdown files.
The sidebar is automatically updated for the site at deployment time.

### Docs Hygiene

The kpt website uses markdownlint to lint docs for formatting and style. Use
prettier with the `"prettier.proseWrap": "always"` setting to auto-format docs
in VSCode.

This includes:

- Lint docs with markdownlint to standardize them and make them easier to
  update.
- Run the kpt website through the [W3 Link Checker] in recursive mode and fix
  warnings and errors.

[error message style guide]: docs/style-guides/errors.md
[documentation style guide]: docs/style-guides/docs.md
[github help]: https://help.github.com/articles/about-pull-requests/
[google's open source community guidelines]:
  https://opensource.google.com/conduct/
[code of conduct]: CODE_OF_CONDUCT.md
[docsify]: https://docsify.js.org/
[site/]: site/
[w3 link checker]: https://validator.w3.org/checklink/
