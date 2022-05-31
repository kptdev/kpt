# kpt roadmap

Last updated: May 31st, 2022

We added new components `package orchestrator`, `Config as Data UI`,
`Config Sync` and `Go function SDK` to the `kpt` toolchain recently. We are
looking for feedback for all the components that will help us prioritize work.
If you have feedback about what you think we should be working on, we encourage
you to get in touch (e.g. by filing an issue, or using the "thumbs-up" emoji
reaction on an issue's first comment).

## 2022

---

## Areas of Focus

---

A few areas of work are ongoing. (This is not exhaustive.)

### Package orchestration

This is an area where we want to support critical scenario associated with
day 2 package management at scale, for example supporting bulk package update,
restoring packages to previous versions, preview package changes, admission
control at the repository level to enforce guardrails.

See the [detailed package orchestration roadmap](https://github.com/GoogleContainerTools/kpt/blob/main/porch/docs/porch-roadmap.md)
or see [the porch issues](https://github.com/GoogleContainerTools/kpt/issues?q=is%3Aopen+is%3Aissue+label%3Aarea%2Fporch)
for more details.

### Package Workflow

We intend to improve workflows involving package update, constructing package
variants, package rendering that involves generator functions, documentation
and examples.
See the [packaging and rendering issues](https://github.com/GoogleContainerTools/kpt/issues?q=is%3Aopen+is%3Aissue+label%3Aarea%2Fhydrate%2Carea%2Fpkg%2Carea%2Fpkg-update)
for more details.

### Function Runtime

We have made some progress in this area by adding support for [`podman`](https://podman.io)
container runtime and in-cluster function evaluator for `package orchestration`
and we we plan to explore other ways to addresss performance and dependency
issues for running functions. Few areas that are in active considerations are:
exploring runtimes such as `WASM` for function execution, builtin support for
Starlark, providing common functions as built-ins in `kpt` CLI.

See the [function runtime issues](https://github.com/GoogleContainerTools/kpt/issues?q=is%3Aopen+is%3Aissue+label%3Aarea%2Ffn-runtime)
for more details.

### Function Catalog and SDK

We plan to improve the `Go function SDK` in areas such as filtering and
indexing resources, handling OpenAPI schema and scaffolding support to enable
boostrapping a function project quickly.
See the [function catalog and SDK issues](https://github.com/GoogleContainerTools/kpt/issues?q=is%3Aopen+is%3Aissue+label%3Aarea%2Ffn-catalog%2Carea%2Ffn-sdk)
for more details.

### Additional storage beyond Git

Currently, `kpt pkg` workflows only support Git repositories. There is some increasing
demand from users to support other storage options beyond git(e.g., OCI). We will
be spending time understanding the use-cases and need for this project. [Tracking issue](https://github.com/GoogleContainerTools/kpt/issues/2300).

## Feedback channels

1. File a [new issue] on Github, but please search first.
1. Reach out to us on kpt-users@googlegroups.com or our [Slack channel](https://app.slack.com/client/T09NY5SBT/C0155NSPJSZ/thread/C0155NSPJSZ-1653582471.722719).

*Note that we have assigned `area/<component>` label to the github issues, for
example `area/porch` label is assigned to all the package orchestration
related issues. This should help in browsing issues by different components.*

[new issue]: https://github.com/GoogleContainerTools/kpt/issues/new/choose
[The Kpt Book]: https://kpt.dev/book/
[apply chapter]: https://kpt.dev/book/06-apply/
[cli-utils]: https://github.com/kubernetes-sigs/cli-utils
[function catalog]: https://catalog.kpt.dev/
[kpt milestones]: https://github.com/GoogleContainerTools/kpt/milestones
[migration guide]: https://kpt.dev/installation/migration
[release notes]: https://github.com/GoogleContainerTools/kpt/releases
