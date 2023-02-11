# kpt roadmap

Last updated: February 9th, 2023

Please follow the [installation](https://kpt.dev/installation/) guide for installing the latest version of kpt.

## Latest releases

https://github.com/GoogleContainerTools/kpt/releases

## Detailed release notes
Please refer to the [release notes] page for more information about the latest features.

## Upcoming features

A few areas of work are ongoing. (This is not exhaustive.)

### Package orchestration

See the [package orchestration roadmap](https://github.com/GoogleContainerTools/kpt/blob/main/porch/docs/porch-roadmap.md)
for more details.

### Explore various options for function runtime

Currently, `kpt fn render` has dependency on docker to execute functions in pipeline.
There are performance and docker dependency issues reported by customers. We will
be exploring different function runtimes in order to address those issues. This is
just an exploratory step and actual implementation if any, will be taken up after December 2021.
[Tracking issue](https://github.com/GoogleContainerTools/kpt/issues/2567).

### Additional storage beyond Git

Currently, `kpt pkg` workflows only support Git repositories. There is some increasing
demand from users to support other storage options beyond git(e.g., OCI). We will
be spending time understanding the use-cases and need for this project. [Tracking issue](https://github.com/GoogleContainerTools/kpt/issues/2300).

## Feedback channels:
1. File a [new issue] on Github, but please search first. 
1. kpt-users@googlegroups.com

[new issue]: https://github.com/GoogleContainerTools/kpt/issues/new/choose
[The Kpt Book]: https://kpt.dev/book/
[apply chapter]: https://kpt.dev/book/06-apply/
[cli-utils]: https://github.com/kubernetes-sigs/cli-utils
[function catalog]: https://catalog.kpt.dev/
[kpt milestones]: https://github.com/GoogleContainerTools/kpt/milestones
[migration guide]: https://kpt.dev/installation/migration
[release notes]: https://github.com/GoogleContainerTools/kpt/releases
