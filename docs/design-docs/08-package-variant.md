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

This proposal introduces concepts and a set of resources for automating the
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
- [#3347](https://github.com/GoogleContainerTools/kpt/issues/3347) Bulk package
  creation
- [#3243](https://github.com/GoogleContainerTools/kpt/issues/3243) Support bulk
  package upgrades
- [#3488](https://github.com/GoogleContainerTools/kpt/issues/3488) Porch:
  BaseRevision controller aka Fan Out controller - but more
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
changes that tie back to our different dimensions of scalability. We can use
data about the specific variant we are creating to lookup additional context in
the Porch cluster, and copy that information into the variant. That context is a
well-structured resource, not simply key/value pairs. KRM functions within the
package can interpret the resource, modifying other resources in the package
accordingly.  The context can come from multiple sources that vary differently
along those dimensions of scalability. For example, one piece of information may
vary by region, another by individual site, another by cloud provider, and yet
another based on whether we are deploying to development, staging, or production.
By utilizing resources in the Porch cluster as our input model, we can represent
this complexity in a manageable model, rather than scattered in templates or
key/value pairs without any structure. By using configurable KRM functions to
interpret the resources within the package, we enable re-use of those input
resources across many packages.

We refer to the mechanism described above as *configuration injection*. It
enables dynamic, context-aware creation of variants. Another way to think about
it is as a continuous reconciliation, much like other Kubernetes controllers. In
this case, the inputs are a parent package `P` and a context `C` (which may be a
collection of many independent resources), with the output being the derived
package `D`. When a new version of `C` is created by updates to in-cluster
resources, we get a new revision of `D`, customized based upon the updated
context. Similarly, the user (or an automation) can monitor for new versions of
`P`; when one arrives, the `PackageVariant` can be updated to point to that new
version, resulting in a newly proposed Draft of `P`. This will be explained in
more detail below.

This proposal also introduces a way to "fan-out", or create multiple
`PackageVariant` resources declaratively based upon a list or selector with the
`PackageVariantSet` resource. This is combined with the injection mechanism to
enable generation of large sets of variants that are specialized to a particular
target repository, cluster, or other resource.

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

### `PackageRevision` Metadata

The package variant controller utilizes Porch APIs. This means that it is not
just doing a `clone` operation, but in fact creating a Porch `PackageRevision`
resource. In particular, that resource can contain Kubernetes metadata that is
not part of the package as stored in the repository.

Some of that metadata is necessary for the management of the `PackageRevision`
by the package variant controller - for example, the `ownerRef` indicating which
`PackageVariant` created the `PackageRevision`. These are not under the user's
control. However, the `PackageVariant` resource does make the annotations and
labels of the `PackageRevision` available as values that may be controlled
during the creation of the `PackageRevision`. This can assist in additional
automation workflows.

## Introducing Variance
Just cloning is not that interesting, so the `PackageVariant` resource also
allows you to control various ways of mutating the original package to create
the variant.

### Package Context[^notimplemented]
Every kpt package that is designated `--for-deployment` will contain a
ConfigMap called `kptfile.kpt.dev`. Kpt (or Porch) will automatically add a
key `name` to the ConfigMap data, with the value of the package name. This
ConfigMap can then be used as input to functions in the Kpt function pipeline.

This process holds true for package revisions created via the package variant
controller as well. Additionally, the author of the `PackageVariant` resource
can specify additional key-value pairs to insert into the package
context.

While this is convenient, it can be easily abused, leading to
over-parameterization. The preferred approach is configuration injection, as
described below, since it allows inputs to adhere to a well-defined, reusable
schema, rather than simple key/value pairs.

### KRM Function Pipeline[^notimplemented]
In the manual workflow, one of the ways we edit packages is by running KRM
functions imperatively. `PackageVariant` offers a similar capability, by
allowing the user to list a KRM function pipeline similar to the `Kptfile`
mutators pipeline. By default, these KRM functions are not added to the
`Kptfile` pipeline, but there is an option to do so.

Creating a namespace and/or setting the namespace for a particular package is a
very common operation. However, since namespace provisioning in a cluster is a
privileged operation, the deployer of a package may not have the authority to
provision a namespace. For this reason, upstream packages should not directly
include `Namespace` resources if they want to be truly reusable.

The KRM function pipeline provides an easy mechanism for the deployer to set the
namespace, or even to create one if their downstream package application
pipeline allows it.[^setns]

### Configuration Injection[^pdc]
Adding values to the package context, or executing functions with their
configuration listed in the `PackageVariant`, works for values that are under
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
  cluster matching the Group, Version, and Kind of the package resource, and
  satisfying the *injection selector*.
- The package variant controller will copy the `spec` field from the matching
  in-cluster resource to the in-package resource.

Note that because we are injecting data *from the Kubernetes cluster*, we can
also monitor that data for changes. For each resource we inject, the package
variant controller will establish a Kubernetes "watch" on the resource (or
perhaps on the collection of such resources). A change to that resource will
result in a new Draft package with the updated configuration injected.

There are a number of additional details that will be described in the detailed
design below, along with the specific API definition.

## Lifecycle Management

### Upstream Changes
The package variant controller allows you to specific a specific upstream
package revision to clone, or you can specify a floating tag[^notimplemented].

If you specify a specific upstream revision, then the downstream will not be
changed unless the `PackageVariant` resource itself is modified to point to a new
revision. That is, the user must edit the `PackageVariant`, and change the
upstream package reference. When that is done, the package variant controller
will update any existing Draft package under its ownership by doing the
equivalent of a `kpt pkg update` to rebase the downstream on the new upstream
revision. If a Draft does not exist, then the package variant controller will
create a new Draft based on the current published downstream, and apply the `kpt
pkg update` to rebase that. This updated Draft must then be proposed and
approved like any other package change.

If a floating tag is used[^notimplemented], then explicit modification of the `PackageVariant` is
not needed. Rather, when the floating tag is moved to a new tagged revision of
the upstream package, the package revision controller will notice and
automatically propose and update to that revision. For example, the upstream
package author may designate three floating tags: stable, beta, and alpha. The
upstream package author can move these tags to specific revisions, and any
PackageVariant resource tracking them will propose updates to their downstream
packages.

### Adoption and Deletion Policies
When a `PackageVariant` resource is created, it will have a particular
repository and package name as the downstream. The adoption policy controls
whether the package variant controller takes over an existing package with that
name, in that repository.

Analogously, when a `PackageVariant` resource is deleted, a decision must be
made about whether or not to delete the downstream package. This is controlled
by the deletion policy.

## Fan Out of Variant Generation

When used with a single package, the package variant controller mostly helps us
handle the time dimension - producing new versions of a package as the upstream
changes, or as injected resources are updated. It can also be useful for
automating common, systematic changes made when bringing an external package
into an organization, or an organizational package into a team repository.

That is useful, but not extremely compelling by itself. More interesting is when
we use `PackageVariant` as a primitive for automations that act on other
dimensions of scale. That means writing controllers that emit `PackageVariant`
resources. For example, we can create a controller that instantiates a
`PackageVariant`s for each developer in our organization, or we can create
a controller to manage `PackageVariant`s across environments. The ability to not
only clone a package, but make systematic changes to that package enables
flexible automation.

Workload controllers in Kubernetes are a useful analogy. In Kubernetes, we have
different workload controllers such as Deployment, StatefulSet, and DaemonSet.
Ultimately, all of these result in Pods; however, the decisions about what Pods
to create, how to schedule them across Nodes, how to configure those Pods, and
how to manage those Pods as changes happen are very different with each workload
controller. Similarly, we can build different controllers to handle different
ways in which we want to generate `PackageRevision`s. The `PackageVariant`
resource provides a convenient primitive for all of those controllers, allowing
a them to leverage a range of well-defined operations to mutate the packages as
needed.

A common need is the ability to generate many variants of a package based on
a simple list of some entity. Some examples include generating package variants
to spin up development environments for each developer in an organization;
instantiating the same package, with slight configuration changes, across a
fleet of clusters; or instantiating some package per customer.

The package variant set controller is designed to fill this common need. This
controller consumes `PackageVariantSet` resources, and outputs `PackageVariant`
resources. The `PackageVariantSet` defines:
- the upstream package
- a list of targets
- rules for generating one `PackageVariant` per target

Three types of targeting are supported:
- An explicit list of repository and package names
- A label selector for Repository objects along with a way to generate package
  names from those objects
- An arbitrary object selector, along with ways to generate repository and
  package names from those objects

Rules for generating `PackageVariant` resources can also access the fields of
the target objects, allowing the package context values, injection selectors,
and function inputs to vary based upon the target.

## Example Use Cases

- describe scenarios and the `PackageVariant`, `PackageVariantSet` resources that
  would solve the scenarios

### Automatically Organizational Customization of an External Package
We can use a `PackageVariant` to provide basic changes that are needed when
importing a package from an external repository. For example, suppose we have
our own internal registry from which we pull all our images. When we import an
upstream package, we want to modify any images listed in the upstream package to
point to our registry. Additionally, we have a policy that all Pods must have a
`chargeback-code` label, so we want to add that as a validating policy to the
package.

In a manual CLI-based workflow, we would accomplish this as follows.
...

With `PackageVariant`, we instead create the following resources:
...

### Creating a Namespace Per Tenant

### Customizing A Workload By Region

### Customizing A Workload By Cluster and Region

### Customizing A Workload By Environment

## Detailed Design

### `PackageVariant` API

#### Basic API

- Upstream
- Downstream
- Annotations
- Labels
- Adoption and Deletion Policy

#### Package Context Injection

#### KRM Function Pipeline

#### Configuration Injection

Configuration injection is controlled by a combination of in-package resources
with annotations, and *injection selectors* defined on the PackageVariant
resource.

Injection selectors are defined in the `spec.injectionSelectors` field of the
`PackageVariant`. This field is an ordered array of structs containing group,
version, kind, and name. Only the name is required.

The annotations, along with the GVK of the annotated resource, allow a package
to "advertise" the injections it can accept and understand. These injection
points effectively form a configuration API for the package, and the injection
selectors provide a way for the `PackageVariant` author to specify the inputs
for those APIs from the possible values in the management cluster. For example,
consider a package with a resource with a custom GVK we have defined, named
"service-endpoints", and containing endpoint addresses for services the package
needs. Those endpoints may vary by region, so in our Porch cluster we maybe have
one of these for each region: "useast1-service-endpoints",
"useast2-service-endpoints", "uswest1-service-endpoints", etc. When we
instantiate the PackageVariant for a cluster, we want to inject the resource
corresponding to the region in which the cluster exists. Thus, for each cluster
we will create a `PackageVariant` resource pointing to the upstream package, but
with injection selector values that are specific to the region for that cluster.

It is important to realize that the name of the in-package resource and the in-
cluster resource need not match. In fact, it would be an unusual coincidence if
they did match. The names in the package are the same across `PackageVariant`s
using that upstream, but we want to inject different resources for each one such
`PackageVariant`. We also do not want to change the name in the package, because
it likely has meaning within the package and will be used by functions in the
package. Also, different roles controlling the names of the in-package and in-
cluster resources. The names in the package are in the control of the package
author. The names in the cluster are in the control of whoever populates the
cluster (for example, some infrastructure team). The selector is the glue
between them, and is in control of the `PackageVariant` resource creator.

The GVK on the other hand, has to be the same for the in-package resource and
the in-cluster resource, because it tells us the API schema for the resource.
Also, the namespace of the in-cluster object needs to be the same as the
`PackageVariant` resource, or we could leak resources from namespaces to which
our `PackageVariant` user does not have access.

With that understanding, the injection process works as follows:

1. The controller will examine all in-package resources, looking for those with
   an annotation named `kpt.dev/config-injection`, with one of the following
   values: `required` or `optional`. We will call these "injection points".
1. For each injection point, a condition will be created in the
   downstream PackageRevision, with ConditionType set to the dot-delimited
   concatenation of `config.injection`, with the in-package resource kind and
   name, and the value set to `False`. Note that since the package author
   controls the name of the resource, kind and name are sufficient to
   disambiguate the injection point. We will call this ConditionType the
   "injection point ConditionType".
1. For each required injection point, the injection point ConditionType will be
   added to the PackageRevision `readinessGates`. Optional injection points'
   ConditionTypes must not be added to the readinessGates by the PackageVariant
   controller, but humans or other actors may do so at a later date, and the
   PackageVariant controller should not remove them on subsequent
   reconciliations. Also, this relies upon `readinessGates` preventing
   publishing to a *deployment* repository, but should not blueprint
   repositories.
1. The injection processing will proceed as follows. For each injection point:
   - If the resource schema of the injection point is not available in the
     cluster, then the injection point ConditionType will be set to `False`,
     with a message indicating that the schema is missing, and processing should
     proceed to the next injection point. Note that for `optional` injection
     points, not having the schema may be intentional and not an error.
   - If the resource schema of the injection point does not contain a `spec`
     field, then the injection point ConditionType will be set to `False`, with
     a message explaining the error, and processing should proceed to the next
     injection point.
   - The controller will now identify all in-cluster objects in the same
     namespace as the PackageVariant resource, with GVK matching the injection
     point (the in-package resource).
   - The controller will look through the list of injection selectors in
     order and checking if any of the in-cluster objects match the selector. If
     so, that in-cluster object is selected, and processing of the list of
     injection selectors stops. Note that in the initial version, the selectors
     are only by name; thus, at most one match is possible for any given
     selector.
   - If no in-cluster object is selected, the injection point ConditionType will
     be set to `False` with a message that no matching in-cluster resource was
     found, and processing proceeds to the next injection point.
   - If a matching in-cluster object is selected, then it is injected as
     follows:
     - The `spec` field from the in-cluster resource is copied to the `spec`
       field of the in-package resource (the injection point), overwriting it.
     - An annotation with name `kpt.dev/injected-resource-name` and value set to
       the name of the in-cluster resource is added (or overwritten) in the
       in-package resource.
1. The PackageVariant resource must have a condition of type `ConfigInjected`.
   This will be set to `True` if and only if:
   - All configuration injection processing is complete.
   - There is no resource annotated as an injection point but having an invalid
     annotation value (i.e., other than `required` or `optional`).
   - Matching in-cluster resources were successfully injected for all `required`
     injection points (all required injection point ConditionTypes are `True`).
   - There are no ambiguous condition types due to conflicting GVK and name
     values. These must be disambiguated in the upstream package, if so.

Note that by allowing the use of GVK, not just name, in the selector, more
precision in selection is enabled. This is a way to constrain the injections
that will be done. That is, if the package has 10 different objects with
`config-injection` annotation, the PackageVariant could say it only wants to
replace certain GVKs, allowing better control.

Consider, for example, if the cluster contains these resources:

- GVK1 foo
- GVK1 bar
- GVK2 foo
- GVK2 bar

If we could only define injection selectors based upon name, it would be
impossible to ever inject one GVK with `foo` and another with `bar`. Instead,
by using GVK, we can accomplish this with a list of selectors like:

 - GVK1 foo
 - GVK2 bar

That said, often name will be sufficiently unique when combined with the
in-package resource GVK, and so making the selector GVK optional is more
convenient.

#### Order of Mutation During Creation
Note that the KRM function pipeline defined in the Kptfile runs every time Porch
saves the package. The package variant controller will save the Draft package
after performing all the requested mutations. Thus, the Draft package will
undergo mutations in this order during creation:

1. Package Context Injections
1. PackageVariant KRM Function Pipeline
1. Config Injection
1. Kptfile KRM Function Pipeline

#### Order of Mutation During Updates

Updates to the downstream PackageRevision can happen for several different
reasons:

1. An injected in-cluster object is updated.
1. The PackageVariant resource is updated, which could change any of the options
   for introducing variance, or could also change the upstream package revision
   referenced.
1. A new revision of the upstream package has been selected due to a floating
   tag change, or due to a force retagging of the upstream.

TODO(johnbelamaric): Figure out and document how the mutation process works in
each of these cases.

#### Open Questions
- Need to understand how the `kpt pkg update` process works and explain it here
  in some detail. We also need to think about whether we want to do anything
  special for updates when the PVS, PV, or any injected resource changes.
- It may be necessary for the package variant controller to annotate the
  in-package resource with a hash of the in-cluster resource to detect changes,
  this should be discussed.

### `PackageVariantSet` API

PVS will add a "nameFrom" or something similar, that will resolve to the explicitly "name" fields in the selectors during fan out. This allows the PVS to, for example, set the injector name based upon the target name or another target field, or even an annotation or label defined on the target.

#### Other Considerations
It would appear convenient to automatically inject the PackageVariantSet
targeting resource. However, it is better to require the package advertise
the ways it accepts injections (i.e., the GVKs it understands), and only inject
those. This keeps the separation of concerns cleaner; the package does not
build in an awareness of the context in which it expects to be deployed. For
example, a package should not accept a Porch Repository resource just because
that happens to be the targeting mechanism. That would make the package unusable
in other contexts.

## Future Considerations
- As an alternative to the floating tag proposal, we may instead want to have
  a separate tag tracking controller that can update PV and PVS resources to
  tweak their upstream as the tag moves.
- Probably want to think about groups of packages (that is, a collection of
  upstreams with the same set of mutation to be applied). For now, this would be
  handled with `PackageVariant` / `PackageVariantSet` resources that differ only
  in their upstream / downstream. Theoretically we could do that with label
  selectors on packages but it gets really ugly really fast. I suspect just
  making people copy the PV / PVS is better.

## Footnotes
[^notimplemented]: Proposed here but not yet implemented as of Porch v0.0.16.
[^setns]: As of this writing, the `set-namespace` function does not have a
    `create` option. This should be added to avoid the user needing to also use
    the `upsert-resource` function. Such common operation should be simple for
    users. Another option is to build this into `PackageVariant`, though at this
    time we do not plan to do so.
[^pdc]: A prototype version of this was implemented in Nephio `PackageDeployment`,
    but this has not been implemented in `PackageVariant` as of Porch v0.0.16.
