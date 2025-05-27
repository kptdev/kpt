# How to Contribute

We'd love to accept your patches and contributions to this project. To learn more about the project structure and
organization, please refer to Project [Governance](governance.md) information. There are just a few small guidelines you
need to follow.

## Developer Certificate of Origin (DCO)

Contributors of this project should state that they agree with the terms published at https://developercertificate.org/
for their contribution. To do this when creating a commit with the Git CLI, a sign-off can be added with
[the -s option](https://git-scm.com/docs/git-commit#git-commit--s). The sign-off is stored as part of the commit message
itself. 

## Contributing large features

Larger features and all the features that affect the interface (CLI or API) of
kpt components need to have a reviewed and merged design document.  It is OK to
start with a prototype in your private fork but if you intend for your feature
to be shipped in kpt please create a design document with this
[design template](/docs/design-docs/00-template.md).

You should create a copy of the template and submit a PR for comments and 
review by maintainers.  Once the PR is merged the design is considered approved.
The actual code change PRs should link to the design documents, even though it
is well understood that the design can drift during implementation.

## Code reviews

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Consult [GitHub Help] for more
information on using pull requests.

## Community Guidelines

This project follows a [Code of Conduct].

## Community Discussion Groups

Join following groups/channels to discuss ideas with other kpt contributors.

1. Join our [Slack channel](https://kubernetes.slack.com/channels/kpt)
1. Join our [Discussions](https://github.com/kptdev/kpt/discussions)

## Style Guides

Contributions are required to follow these style guides:

- [Error Message Style Guide]
- [Documentation Style Guide]

## Contributing to `kpt`

The kpt toolchain has several components such as `kpt CLI`, `package orchestrator`,
`function catalog`, `function SDKs`, `Backstage UI plugin` and `config sync`. Each
component has their own development process.
Refer to the pointers below to learn more:

#### Building the Source

1. Clone the project

   ```shell
   git clone https://github.com/kptdev/kpt
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

### Package Orchestrator

Package orchestrator code has been moved to the Nephio project: https://github.com/nephio-project/porch

### Function Catalog

Function catalog has its own repository. Refer to the
[documentation in the krm-functions-catalog](https://github.com/kptdev/krm-functions-catalog/blob/master/CONTRIBUTING.md)
repo.

### Documentation

If you are updating the documentation, please do it in separate PRs from code
changes and PR description should start with `Docs:`.

#### Run the docs locally

Make docs changes and test them by running the site in a docker container with
`make site-run-server`.

It's usually a good idea to test locally for the following:

- Broken links
- Rendering static content

#### Update docs

Docs are under [site/] and use [docsify] to present the source markdown files.
The sidebar is automatically updated for the site at deployment time.

#### Docs Hygiene

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
