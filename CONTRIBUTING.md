# How to Contribute

We'd love to accept your patches and contributions to this project. To learn more about the project structure and organization, please refer to Project [Governance](governance.md) information. There are
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

This project follows [Google's Open Source Community Guidelines] and a [Code of
Conduct].

## Community Discussion Groups

Join following groups/channels to discuss ideas with other kpt contributors.

1. For developers join our [email list](https://groups.google.com/forum/?oldui=1#!forum/kpt-dev)
1. You can add this Google [calendar](https://calendar.google.com/calendar/u/0?cid=Y183cWI2ZTY5MW4zMmhxdmxncTdyMWhmOTFta0Bncm91cC5jYWxlbmRhci5nb29nbGUuY29t) to get access to all our meetings. If you'd like to be automatically added to all meetings please join this [list](https://groups.google.com/forum/?oldui=1#!forum/kpt-contribx). Note, our meetings are open to anyone and you do not need an agenda to come. You may be asked for an introduction. For your first meeting, we would suggest you either start with Office Hours or the SIG meeting.
1. Join our [Slack channel](https://kubernetes.slack.com/channels/kpt)

## Style Guides

Contributions are required to follow these style guides:

- [Error Message Style Guide]
- [Documentation Style Guide]

## Contributing to `kpt`

The kpt toolchain has several components such as `kpt CLI`, `package orchestrator`,
`function catalog`, `function SDKs`, `Backstage UI plugin` and `config sync`. Each
component has their own development process.
Refer to the pointers below to learn more:

## Attend Meetings
* All [SIGs](governance.md) have a regular scheduled meeting for feature/roadmap management and planning. For example if you are interested in discussing the roadmap for Config Sync you should attend the **Kpt SIG Config Sync** meeting
* Sub-projects are focused on more tactical items and other sub-project lifecycle items. Sub-projects generally have standups associated with them.
* Working Groups are nimble. Focused a lot on experimentation. Attend if you'd like to see demos and discuss future looking user experiences.
* We regularly have an office hours where we invite the entire community to attend.
* Meeting notes will be kept for all meetings.
* We will not record meetings unless someone explictly asks for one and there are no objections by attendees. If a meeting is recorded a link to it will be left in the notes.

### Meeting Notes
Links to meeting notes. Some may require access to mailing lists above. If in doubt join [kpt-contribx](https://groups.google.com/forum/?oldui=1#!forum/kpt-contribx)

* [Kpt Office Hours](https://docs.google.com/document/d/1I5CJDk9xkDj1vvvwvZNgvaNusE2TanX0Iiy9G1oitz0/view)
* [App Wow Working Group](https://docs.google.com/document/d/1pHsmYjHr9XMwJ_fdJtPiodd8WSg5ilCLIrP_8KE-yKE/view)

Coming soon
* SIG Config as Data
* SIG Config Sync

### kpt CLI

#### Building the Source

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

### Package Orchestrator

Package orchestrator code live under `porch` directory in this repo. Please see the
[developer docs for porch](porch/docs/development.md) to learn more.

### Function Catalog

Function catalog has its own repository. Refer to the
[documentation in the kpt-functions-catalog](https://github.com/GoogleContainerTools/kpt-functions-catalog/blob/master/CONTRIBUTING.md)
repo.

### Config Sync

Config Sync has its own [repository](https://github.com/GoogleContainerTools/kpt-config-sync).
Refer to the [documentation in the config-sync repo](https://github.com/GoogleContainerTools/kpt-config-sync/blob/main/docs/contributing.md).

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
