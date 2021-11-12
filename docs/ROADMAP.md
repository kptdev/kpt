# kpt roadmap for 2021

Last updated: November 5th, 2021

Please follow the [installation](https://kpt.dev/installation/) guide for installing the latest version of kpt.

## Latest releases

### Targeting resources in `kpt fn` commands

Users want to invoke a kpt function (imperatively and declaratively) on a subset of
resources in the package by selecting them on the basis of GVKNN(Group, Version, Kind, Name, Namespace), package-path,
file path etc. For example, set work-load identity annotation on all Kubernetes
Service Account resources in this package. Here is the documentation for list of
available selectors for [render] and [eval]. Available in [v1.0.0-beta.7]+ versions of kpt.
More selectors will be added incrementally.

## Detailed release notes
Please refer to the [release notes] page for more information about the latest features.

## Upcoming features

### Improve Function Authoring Experience

We need a rich ecosystem of third party functions. Users should be able to write
functions with custom logic very quickly using the tools they are familiar with.
So we are investing on making function authoring experience very easy. This is an
ongoing effort. 
- [Starlark enhancements](https://catalog.kpt.dev/starlark/v0.3/?id=developing-starlark-script)
have been released. 
- For Golang SDK([issue](https://github.com/GoogleContainerTools/kpt/issues/2568)), **Estimated release date:** End of November 2021.

### Best practices for kpt with idiomatic package examples

We need to publish best practices to use kpt inorder to create and use kpt packages.
This will help users to understand the right way of using kpt. These best practice
guidelines should be backed by idiomatic kpt package examples. These packages should
be designed reflecting the best practices, easily discoverable and simple to understand.
This is an ongoing effort. [Tracking issue](https://github.com/GoogleContainerTools/kpt/issues/2541)
- Best practices guide, **Estimated release date:** End of November 2021.
- Idiomatic package examples, **Estimated release date:** End of December 2021.

### Merging pipeline section during `kpt pkg update`

Currently, `kpt pkg update` doesn't merge pipeline section in the Kptfile as expected.
The fact that pipeline section is non-associative list with defined ordering makes it 
very difficult to merge with upstream counterpart. This is forcing users to use setters
and discouraging them from declaring other functions in the pipeline as they will be
deleted during `kpt pkg update`. Merging pipeline correctly will reduce
huge amount of friction in declaring new functions which in turn helps to avoid
excessive parameterization. [Tracking issue](https://github.com/GoogleContainerTools/kpt/issues/2529). 
**Estimated release date:** End of December 2021.

### Explore various options for function runtime

Currently, `kpt fn render` has dependency on docker to execute functions in pipeline.
There are performance and docker dependency issues reported by customers. We will
be exploring different function runtimes in order to address those issues. This is
just an exploratory step and actual implementation if any, will be taken up after December 2021.
[Tracking issue](https://github.com/GoogleContainerTools/kpt/issues/2567).

### Integrate kpt with Cloud Code

One of the major areas of investment is to integrate [Cloud Code](https://cloud.google.com/code) with kpt to provide
package authoring assistance. Users can author Kptfile and functionConfig files with
features like auto-complete and error detection. This significantly improves the
discoverability of Kptfile schema, catalog functions and their functionConfigs.
- Cloud code integration with Kptfile schema has been [released](https://github.com/GoogleCloudPlatform/cloud-code-intellij/blob/main/CHANGELOG.md) and available from 21.10.1+ versions of Cloud Code.
- Function catalog integration, **Estimated release date:** End of December 2021.

### Additional storage beyond Git

Currently, `kpt pkg` workflows only support Git repositories. There is some increasing
demand from users to support other storage options beyond git(e.g., OCI). We will
be spending time understanding the use-cases and need for this project. [Tracking issue](https://github.com/GoogleContainerTools/kpt/issues/2300).

## Feedback channels:
1. File a [new issue] on Github, but please search first. 
1. kpt-users@googlegroups.com

[new issue]: https://github.com/GoogleContainerTools/kpt/issues/new/choose
[declarative function execution]: https://kpt.dev/book/04-using-functions/01-declarative-function-execution
[apply-setters documentation]: https://catalog.kpt.dev/apply-setters/v0.1/ 
[The Kpt Book]: https://kpt.dev/book/
[apply chapter]: https://kpt.dev/book/06-apply/
[cli-utils]: https://github.com/kubernetes-sigs/cli-utils
[function catalog]: https://catalog.kpt.dev/
[kpt milestones]: https://github.com/GoogleContainerTools/kpt/milestones
[migration guide]: https://kpt.dev/installation/migration
[render]: https://kpt.dev/book/04-using-functions/01-declarative-function-execution?id=specifying-selectors
[eval]: https://kpt.dev/book/04-using-functions/02-imperative-function-execution?id=specifying-selectors
[v1.0.0-beta.7]: https://github.com/GoogleContainerTools/kpt/releases/tag/v1.0.0-beta.7
[release notes]: https://github.com/GoogleContainerTools/kpt/releases