# kpt roadmap for 2021

Last updated: September 14th, 2021

Draft of *v1* is released. Please follow the [installation](https://kpt.dev/installation/) guide.

### 1. Declarative function pipeline

kpt has added a declarative way to customize and validate configuration.  
This allows you to run several mutation and validation 
functions in a pipeline alleviating the need to create shell scripts that do 
the same thing.  Further information can be found in the 
[declarative function execution]  section of the [The Kpt Book].

### 2. Setters

Setters used to be a special entity without sufficient differentiation from
KRM functions. In kpt v1 setters become just another function with drastically
simplified syntax.  Configuring 4 setters used to take 20 lines of yaml 
and now takes 6.  Setters also get recursively applied to sub-packages by
default.  For further information on the setter function please visit: 
[apply-setters documentation]. 

### 3. Resource-merge

kpt package updates now default to the resource-merge strategy 
which allows you to edit configuration with an text editor of your choice 
and still be able to get updates with upstream changes. 

### 4. Live apply

_kpt live_ used to use ConfigMap to store inventory information. This was
convenient as it didn't require any CRDs, but it had challenges around encoding
of the GroupKind, name and namespace, and it didn't allow us to easily add
additional metadata about a package, such as the package version. This is
is now migrated to use ResourceGroup CRD.  You can learn more about in the
[The Kpt Book](https://kpt.dev/book/06-deploying-packages/).

### 5. Updated documentation

Documentation was a major area of investment, including the [The Kpt Book].
The book is a methodical way to introduce some unique kpt concepts like 
in place editing and hydration.  It's meant to be a hands on guide where the user
configures and deploys wordpress and nginx while learning about the kpt
concepts.

### 6. Function catalog

All of the hydration and validation logic has been moved from the kpt binary 
to functions allowing for flexibility and security. This enables new 
scenarios like limiting the customization and validation to a subset of 
allowed functions.  The function catalog has received additional functions, 
examples and help. Please visit the [function catalog] for further information.

### 7. Targeting resources in `kpt fn` commands

Users want to invoke a kpt function (imperatively and declaratively) on a subset of
resources in the package by selecting them on the basis of GVKNN, package-path,
file path etc. For example, set work-load identity annotation on all Kubernetes
Service Account resources in this package [Tracker](https://github.com/GoogleContainerTools/kpt/issues/2015).
**Estimated release date:** Last week of September 2021.

## In Progress

### 8. Metrics for kpt

We need a way to measure the adoption of kpt. This will help us understand the current
impact of kpt and help us identify the areas where we can improve. Measuring KCC resources under
management(RUM) is the primary goal of this work. **Estimated completion date:** End of November 2021.
This will be an ongoing effort and other metrics about GKE cluster resources, site visits etc. will
be implemented in the near future.

### 9. Merging pipeline section during `kpt pkg update`

Currently, `kpt pkg update` doesn't merge pipeline section in the Kptfile as expected.
The fact that pipeline section is non-associative list with defined ordering makes it 
very difficult to merge with upstream counterpart. This is forcing users to use setters
and discouraging them from declaring other functions in the pipeline as they will be
deleted during `kpt pkg update`. Merging pipeline correctly will reduce
huge amount of friction in declaring new functions which in turn helps to avoid
excessive parameterization. **Estimated completion date:** End of December 2021.

### 10. Best practices for kpt with idiomatic package examples

We need to publish best practices to use kpt inorder to create and use kpt packages.
This will help users to understand the right way of using kpt. These best practice
guidelines should be backed by idiomatic kpt package examples. These packages should
be designed reflecting the best practices, easily discoverable and simple to understand.
This is an ongoing effort. For best practices guide **Estimated completion date:** End of November 2021.
Idiomatic package examples **Estimated completion date:** End of December 2021.

### 11. Improve Function Authoring Experience

We need a rich ecosystem of third party functions. Users should be able to write 
functions with custom logic very quickly using the tools they are familiar with. 
So we are investing on making function authoring experience very easy. This is an
ongoing effort. [Starlark enhancements](https://github.com/GoogleContainerTools/kpt/issues/2504) 
(and docs improvement) will be delivered by the end of September 2021.
For Golang SDK, **Estimated release date:** End of November 2021.

### 12. Explore various options for function runtime

Currently, `kpt fn render` has dependency on docker to execute functions in pipeline.
There are performance and docker dependency issues reported by customers. We will
be exploring different function runtimes in order to address those issues. This is
just an exploratory step and actual implementation if any, will be taken up after December 2021.

### 13. Integrate kpt with Cloud Code

One of the major areas of investment is to integrate [Cloud Code](https://cloud.google.com/code) with kpt to provide
package authoring assistance. Users can author Kptfile and functionConfig files with
features like auto-complete and error detection. This significantly improves the
discoverability of Kptfile schema, catalog functions and their functionConfigs.
**Estimated release date:** Last week of September 2021 for Kptfile schema integration,
function catalog integration will be released by end of November 2021.

## Ongoing work
Since this is a draft of the release notes you should be aware of the
ongoing work. The list of all the current and future milestones can be
found here: [kpt milestones]

## Upgrading from previous version of kpt.
There are a number of breaking changes that had to be done to clean up the
CLI and the data format for kpt.  Please visit the [migration guide] for 
your existing kpt content.

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
