# How to Contribute

We'd love to accept your patches and contributions to this project. To learn more about the project structure and
organization, please refer to Project [Governance](https://github.com/kptdev/governance) information. There are just a
few small guidelines you need to follow.

## Developer Certificate of Origin (DCO)

Contributors of this project should state that they agree with the terms published at https://developercertificate.org/
for their contribution. To do this when creating a commit with the Git CLI, a sign-off can be added with
[the -s option](https://git-scm.com/docs/git-commit#git-commit--s). The sign-off is stored as part of the commit message
itself. 

## Copyright notices

All files should have the copyright notice.
```
// Copyright 2026 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
```

If the file has never been modified: use the creation year only

* Example: `Copyright 2026 The kpt Authors`

If the file has been modified: use a year range from creation to last modification

* Example: `Copyright 2024-2026 The kpt Authors`

## Contributing large features

Larger features and all the features that affect the interface (CLI or API) of
kpt components need to have a reviewed and merged design document. It is OK to
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

Process for code reviews. Before requesting human review, a PR must:

* All tests passing
* All linting passing
* Meeting project code quality requirements, including passing all configured static analysis / SonarCloud quality gates and not reducing automated test coverage for the affected components
* The comments from the first run of automatically generated comments (AI generated comments, SonarCloud comments, bot generated comments, etc.) of the PR are addressed (addressing further re-runs of AI are optional)
* If it is not possible to resolve an automatic comment, please add a sub-comment indicating why the automated comment cannot be resolved or ask for help in resolving the comment

## Declare any use of AI

> The use of AI in the creation of PRs is allowed but you must declare any use of AI and you must be able to explain the PR code independently of any AI tools.

Update the PR description to state whether you used AI to help you create this PR; if so, list the AI tools you have used and in what areas.

For example:
```text
I have used AI in the creation of this PR.

I have used the following AI tools:
- GitHub Copilot to analyse the code
- Claude Code to generate the function someNewFunctionIAdded()
- Amazon Q to generate unit tests
```

### Attribute AI in the Git commit messages

Following the [guidance of the Linux kernel](https://docs.kernel.org/process/coding-assistants.html#attribution)
we recommend the attribution of AI tools in the commit messages using the following format:

```shell
Assisted-by: AGENT_NAME:MODEL_VERSION [TOOL1] [TOOL2]
```

Where:

- `AGENT_NAME` is the name of the AI tool or framework
- `MODEL_VERSION` is the specific model version used
- `[TOOL1] [TOOL2]` are optional specialized analysis tools used (e.g., coccinelle, sparse, smatch, clang-tidy)

Basic development tools (git, gcc, make, editors) should not be listed.

Example:

```shell
Assisted-by: Claude:claude-3-opus coccinelle sparse
```

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

### Function Catalog

Function catalog has its own repository. Refer to the
[documentation in the krm-functions-catalog](https://github.com/kptdev/krm-functions-catalog/blob/master/CONTRIBUTING.md)
repo.

### Documentation

If you are updating the documentation, please do it in separate PRs from code
changes and PR description should start with `Docs:`.

#### Run the docs locally

Make docs changes and test them by running the site in a docker container with
`make serve`.

It's usually a good idea to test locally for the following:

- Broken links
- Rendering static content

#### Update docs

Docs are under [documentation/](documentation/). Refer to the [README.md](documentation/README.md) in the folder for details about
documentation contributions. 


[error message style guide]: docs/style-guides/errors.md
[documentation style guide]: docs/style-guides/docs.md
[github help]: https://help.github.com/articles/about-pull-requests/
[google's open source community guidelines]:
  https://opensource.google.com/conduct/
[code of conduct]: CODE_OF_CONDUCT.md
[docsify]: https://docsify.js.org/
[site/]: site/
[w3 link checker]: https://validator.w3.org/checklink/
