# Package Variant Controller

* Author(s): @johnbelamaric, @natasha41575
* Approver: @mortent

## Why

When deploying workloads across large fleets of clusters, it is often necessary
to modify the workload configuration for a specific cluster. Additionally, those
workloads may evolve over time with security or other patches that require
updates. [Configuration as Data](06-config-as-data.md) in general and [Package
Orchestration](07-package-orchestration.md) in particular can assist in this.
However, they are still centered around manual, one-by-one hydration and
configuration of a workload.

This proposal introduces concepts and a set of resources that for automating the
creation and lifecycle management of package variants. These are designed to
address several different dimensions of scalability:
- Number of different workloads for a given cluster
- Number of clusters across which those workloads are deployed
- Complexity of the organizations deploying those workloads (NOTE: actually this
  is more the `conditions` stuff, will probably move this out of here and create
  a separate proposal for that)
- Changes to those workloads over time

## See Also
- [Package Orchestration](07-package-orchestration.md)
- [#3488](https://github.com/GoogleContainerTools/kpt/issues/3488)
- [Managing Package
   Revisions](https://docs.google.com/document/d/1EzUUDxLm5jlEG9d47AQOxA2W6HmSWVjL1zqyIFkqV1I/edit?usp=sharing)
- [Porch UpstreamPolicy Resource
  API](https://docs.google.com/document/d/1OxNon_1ri4YOqNtEQivBgeRzIPuX9sOyu-nYukjwN1Q/edit?usp=sharing&resourcekey=0-2nDYYH5Kw58IwCatA4uDQw)

## Core Concepts

For this solution, "workloads" are represented by packages. "Package" is a more
general concept, being an arbitrary bundle of resources, and therefore is
sufficient to solve the originally stated problem.

The basic idea here is to introduce a `PackageVariant` resource that manages the
derivation of variant of a package from the original source package, and to
manage the evolution of that variant over time. This effectively automates the
human-centered process for variant creation one might use with `kpt`:
1. Clone and upstream package locally
1. Make changes to the local package, including executing KRM functions
1. Push the package to a new repository and tag it as a new version

Similarly, `PackageVariant` can manage the process of updating a package when a
new version of the upstream package is published. In the human-centered
workflow, a user would use `kpt pkg update` to pull in changes to their
derivative package. When using a `PackageVariant` resource, the change would be
made to the upstream specification in the resource, and the controller would
propose a new Draft package reflecting the outcome of `kpt pkg update`.

By automating this process, we open up the possibility of performing systematic
changes that tie back to our different dimensions of scalability. For example,
we can use data about the specific variant we are creating to lookup additional
context in the Porch cluster, and copy that information into the variant. We
refer to this as *injection*, and it enables dynamic, context-aware creation of
variants. This will be explained in more detail below.

This proposal also introduces a way to "fan-out", or create multiple
`PackageVariant` resources declaratively based upon a list or selector with the
`PackageVariantSet` resource.. This is combined with the injection mechanism to
enable generation of large sets of variants that are specialized to a particular
target repository, cluster, or other target.

## Basic Package Cloning

The `PackageVariant` resource controls the creation and lifecycle of a variant
of a package. That is, it defines the original (upstream) package, the new
(downstream) package, and the changes (mutations) that need to be made to
transform the upstream into the downstream. It also allows the user to specify
policies around adoption, deletion, and update of package revisions that are
under the control of the package variant controller.

![Basic Package Variant](PackageVariant%20-%201.png)

Note that *proposal* and *approval* are not handled by the package variant
controller. Those are left to humans or other controllers. The exception is the
proposal of deletion (there is no concept of a "Draft" deletion), which the
package variant control will do, depending upon the specified deletion policy.

### Package Revision Meta Data

The package variant controller utilizes Porch APIs. This means that it is not
just doing a `clone` operation, but in fact creating a Porch `PackageRevision`
resource. In particular, that resource can contain Kubernetes metadata that is
not part of the package as stored in the repository.

Some of that metadata is necessary for the management of the PackageRevision by
the PackageVariant controller - for example, the `ownerRef` indicating which
`PackageVariant` created the `PackageRevision`. These are not in control of the
user. However, the `PackageVariant` resource does make the annotations and
lables of the `PackageRevision` available as values that may be controlled
during the creation of the `PackageRevision`. This can assist in additional
automation workflows.

## Introducting Variance
Just cloning is not that interesting, so the `PackageVariant` resource also
allows you to control various ways of mutating the original package to create
the variant.

### Package Context
[Note: I believe right now we *always* produce `--for-deployment` package
revisions out of package variant controller. We will likely need to allows some
control over that, as we already have a use case in Nephio to use a
PackageVariant to create a package intended as a base for fan out]

Every kpt package that is designated `--for-deployment` will contain a
ConfigMap called `kptfile.kpt.dev`. Kpt (or Porch) will automatically add a
key `name` to the ConfigMap data, with the value of the package name. This
ConfigMap can then be used as input to functions in the Kpt function pipeline.

This process holds true for package revisions created via the package variant
controller as well. Additionally, the author of the `PackageVariant` resource
can specify additional key/value pairs to insert into the package context [note:
this additional functionality is proposed here but not yet implemented].

### KRM Function Calls
[Note: not yet implemented]
TODO(johnbelamaric): describe adding a KRM function pipeline to the
PackageVariant to allow arbitrary mutations
- question: should these allow adding to the package pipeline a la `--save`? Is
  that a *separate* pipeline?

### Configuration Injection
[Note: this is implemented in the Nephio `PackageDeployment` controller, but not
yet in `PackageVariant`.]

Adding values to the package context, or executing functions with their
configuration listed in the `PackageVariant` works for values that are under
control of the author of the `PackageVariant` resource. However, in more
advanced use cases, we may need to specialize the package based upon other
contextual information. This particularly comes into play when the user
deploying the workload does not have direct control over the context in which
it is being deployed. For example, one part of the organization may manage the
infrastructure - that is, the cluster in which we are deploying the workload -
and another part the actual workload. We would like to be able to pull in inputs
specified by the infrastructure team automatically, based the cluster to which
we are deploying the workload, or perhaps the region in which that cluster is
deployed.

To facilitate this, the package variant controller can "inject" configuration
directly into the package. This means it will use information specific to this
instance of the package to lookup a resource in the Porch cluster, and copy that
information into the package. Of course, the package has to be ready to receive
this information. So, there is a protocol for facilitating this dance:
- Packages may contain resources annotated with `kpt.dev/config-injection`
- Usually, these will also be `config.kubernetes.io/local-config` resources, as
  they are likely just used by local functions as input. But this is not
  mandatory.
- The package variant controller will look for any resource in the Kubernetes
  clsuter matching the Group, Version, and Kind of the package resource, and
  satisfying the *injection selector*.
- The package variant controller will copy the `spec` field from the matching
  in-cluster resource to the in-package resource.

Note that because we are injecting data *from the Kubernetes cluster*, we can
also monitor that data for changes. For each resource we inject, the package
variant controller will establish a Kubernetse "watch" on the resource. A change
to that resource will result in a new Draft package with the updated
configuration injected.

[Note: it may be necessary for the package variant controller to annotate the
in-package resource with a hash of the in-cluster resource to detect changes,
this should be discussed].

### Namespace Configuration and Injection
[Note: not yet implemented]
Creating a namespace and/or setting the namespace for a particular package is a
very common operation. However, since namespace provisioning in a cluster is a
privileged operation, the deployer of a package may not have the authority to
provision a namespace. For this reason, upstream packages should not directly
include `Namespace` resources if they want to be truly reusable.

So, the package variant controller provides some convenience features for
targeting the package at a particular namespace. First, it is possible to use
the KRM function capability to call the well-known `set-namespace` function.
For convenience, you can alternatively set the `namespace.value` field in the
`PackageVariant` resource, and this will be done for you automatically.

Similarly, you can specify `namespace.create: true`, and the package variant
controller will inject a `Namespace` resource of the given name directly into
the package (in addition to calling `set-namespace`).

### Order of Mutation During Provisioning
TODO(johnbelamaric): diagram showing when each of the above stages is done, and
when during that process the standard Kpt package pipeline is run, during the
initial creation / clone of the package

## Lifecycle Management

### Upstream Changes

### Order of Mutation During Update
TODO(johnbelamaric): diagram showing when each of the above stages is done, and
when during that process the standard Kpt package pipeline is run, during the
package update process

### Adoption Policy

### Deletion Policy

## Fan Out of Variant Generation

TODO(johnbelamaric): Describe `PackageVariantSet`.

## PackageVariant API


## PackageVariantSet API

## Example Use Cases

- describe scenarios and the PackageVariant, PackageVariantSet resources that
  would solve the scenarios
