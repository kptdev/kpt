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

## In Progress

### 7. Targeting resources in `kpt fn` commands

Users want to invoke a kpt function (imperatively and declaratively) on a subset of 
resources in the package by selecting them on the basis of GVKNN, package-path, 
file path etc. For example, set work-load identity annotation on all Kubernetes 
Service Account resources in this package [Tracker](https://github.com/GoogleContainerTools/kpt/issues/2015).
**Estimated release date:** Last week of September 2021.


### 8. Integrate kpt with Cloud Code

One of the major areas of investment is to integrate [Cloud Code](https://cloud.google.com/code) with kpt to provide 
package authoring assistance. Users can author Kptfile and functionConfig files with
features like auto-complete and error detection. This significantly improves the 
discoverability of Kptfile schema, catalog functions and their functionConfigs.
**Estimated release date:** Last week of September 2021 for Kptfile schema integration,
function catalog integration will be released by mid of November 2021.

### 9. Pre/Post processing step in `kpt fn render`

We need a way to pre(/post) process inputs(/outputs) involved in customizing the 
package or any of its sub-packages before(/after) invoking the function pipeline [Tracker](https://github.com/GoogleContainerTools/kpt/issues/2419).
e.g. Use-case to propagate common setters values in the package hierarchy before
rendering the package resources.
**Estimated release date:** Mid of November 2021.

### 10. Improve functionConfig UX

Package publishers want to provide command driven instructions (semi-automated)
to configure a package that makes the user-guide/examples easy to follow.
e.g. Porcelain to set values in functionConfig or generate functionConfig.
**Estimated release date:** End of October 2021.

Package consumers should be able to detect configuration errors early before the
package is assumed to be ready. e.g. Introduce validation for functionConfig.
**Estimated release date:** End of November 2021.

### 11. Improve Function Authoring Experience

We need a rich ecosystem of third party functions. Users should be able to write 
functions with custom logic very quickly using the tools they are familiar with. 
So we are investing on making function authoring experience very easy. This is an
ongoing effort. [Starlark enhancements](https://github.com/GoogleContainerTools/kpt/issues/2504) 
(and docs improvement) will be delivered by the end of September 2021. 
**Estimated completion date:** End of December 2021.

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
