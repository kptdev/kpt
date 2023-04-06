# Package Variant Controller

* Author(s): @johnbelamaric, @natasha41575
* Status: Work-in-Progress
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
- Different types or characteristics of those clusters
- Complexity of the organizations deploying those workloads
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

The basic idea here is to introduce a PackageVariant resource that manages the
derivation of a variant of a package from the original source package, and to
manage the evolution of that variant over time. This effectively automates the
human-centered process for variant creation one might use with `kpt`:
1. Clone an upstream package locally
1. Make changes to the local package, setting values in resources and
   executing KRM functions
1. Push the package to a new repository and tag it as a new version

Similarly, PackageVariant can manage the process of updating a package when a
new version of the upstream package is published. In the human-centered
workflow, a user would use `kpt pkg update` to pull in changes to their
derivative package. When using a PackageVariant resource, the change would be
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
this complexity in a manageable model that is reused across many packages,
rather than scattered in package-specific templates or key/value pairs without
any structure. KRM functions, also reused across packages but configured as
needed for the specific package, are used to interpret the resources within the
package. This decouples authoring of the packages, creation of the input model,
and deploy-time use of that input model within the packages, allowing those
activities to be performed by different teams or organizations.

We refer to the mechanism described above as *configuration injection*. It
enables dynamic, context-aware creation of variants. Another way to think about
it is as a continuous reconciliation, much like other Kubernetes controllers. In
this case, the inputs are a parent package `P` and a context `C` (which may be a
collection of many independent resources), with the output being the derived
package `D`. When a new version of `C` is created by updates to in-cluster
resources, we get a new revision of `D`, customized based upon the updated
context. Similarly, the user (or an automation) can monitor for new versions of
`P`; when one arrives, the PackageVariant can be updated to point to that new
version, resulting in a newly proposed Draft of `D`, updated to reflect the
upstream changes. This will be explained in more detail below.

This proposal also introduces a way to "fan-out", or create multiple
PackageVariant resources declaratively based upon a list or selector, with the
PackageVariantSet resource. This is combined with the injection mechanism to
enable generation of large sets of variants that are specialized to a particular
target repository, cluster, or other resource.

## Basic Package Cloning

The PackageVariant resource controls the creation and lifecycle of a variant
of a package. That is, it defines the original (upstream) package, the new
(downstream) package, and the changes (mutations) that need to be made to
transform the upstream into the downstream. It also allows the user to specify
policies around adoption, deletion, and update of package revisions that are
under the control of the package variant controller.

The simple clone operation is shown in *Figure 1* (also see the
[legend](#figure-legend)).

![Figure 1: Basic Package Cloning](packagevariant-clone.png)

Note that *proposal* and *approval* are not handled by the package variant
controller. Those are left to humans or other controllers. The exception is the
proposal of deletion (there is no concept of a "Draft" deletion), which the
package variant control will do, depending upon the specified deletion policy.

### PackageRevision Metadata

The package variant controller utilizes Porch APIs. This means that it is not
just doing a `clone` operation, but in fact creating a Porch PackageRevision
resource. In particular, that resource can contain Kubernetes metadata that is
not part of the package as stored in the repository.

Some of that metadata is necessary for the management of the PackageRevision
by the package variant controller - for example, the owner reference indicating
which PackageVariant created the PackageRevision. These are not under the user's
control. However, the PackageVariant resource does make the annotations and
labels of the PackageRevision available as values that may be controlled
during the creation of the PackageRevision. This can assist in additional
automation workflows.

## Introducing Variance
Just cloning is not that interesting, so the PackageVariant resource also
allows you to control various ways of mutating the original package to create
the variant.

### Package Context[^notimplemented]
Every kpt package that is fetched with `--for-deployment` will contain a
ConfigMap called `kptfile.kpt.dev`. Analogously, when Porch creates a package
in a deployment repository, it will create this ConfigMap, if it does not
already exist. Kpt (or Porch) will automatically add a key `name` to the
ConfigMap data, with the value of the package name. This ConfigMap can then
be used as input to functions in the Kpt function pipeline.

This process holds true for package revisions created via the package variant
controller as well. Additionally, the author of the PackageVariant resource
can specify additional key-value pairs to insert into the package
context, as shown in *Figure 2*.

![Figure 2: Package Context](packagevariant-context.png)

While this is convenient, it can be easily abused, leading to
over-parameterization. The preferred approach is configuration injection, as
described below, since it allows inputs to adhere to a well-defined, reusable
schema, rather than simple key/value pairs.

### KRM Function Pipeline[^notimplemented]
In the manual workflow, one of the ways we edit packages is by running KRM
functions imperatively. PackageVariant offers a similar capability, by
allowing the user to add functions to the beginning of the downstream package
`Kptfile` mutators pipeline. These functions will then execute before the
functions present in the upstream pipeline. It is not exactly the same as
running functions imperatively, because they will also be run in every
subsequent execution of the downstream package function pipeline. But it can
achieve the same goals.

For example, consider an upstream package that includes a Namespace resource.
In many organizations, the deployer of the workload may not have the permissions
to provision cluster-scoped resources like Namespaces. This means that they
would not be able to use this upstream package without removing the Namespace
resource (assuming that they only have access to a pipeline that deploys with
constrained permissions). By adding a function that removes Namespace resources,
and a call to `set-namespace`, they can take advantage of the upstream package.

Similarly, the KRM function pipeline feature provides an easy mechanism for the
deployer to create and set the namespace if their downstream package application
pipeline allows it, as seen in *Figure 3*.[^setns]

![Figure 3: KRM Function](packagevariant-function.png)

### Configuration Injection[^pdc]
Adding values to the package context or functions to the pipeline works
for configuration that is under the control of the creator of the PackageVariant
resource. However, in more advanced use cases, we may need to specialize the
package based upon other contextual information. This particularly comes into
play when the user deploying the workload does not have direct control over the
context in which it is being deployed. For example, one part of the organization
may manage the infrastructure - that is, the cluster in which we are deploying
the workload - and another part the actual workload. We would like to be able to
pull in inputs specified by the infrastructure team automatically, based the
cluster to which we are deploying the workload, or perhaps the region in which
that cluster is deployed.

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

![Figure 4: Configuration Injection](packagevariant-config-injection.png)


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
changed unless the PackageVariant resource itself is modified to point to a new
revision. That is, the user must edit the PackageVariant, and change the
upstream package reference. When that is done, the package variant controller
will update any existing Draft package under its ownership by doing the
equivalent of a `kpt pkg update` to update the downstream to be based upon
the new upstream revision. If a Draft does not exist, then the package variant
controller will create a new Draft based on the current published downstream,
and apply the `kpt pkg update`. This updated Draft must then be proposed and
approved like any other package change.

If a floating tag is used, then explicit modification of the PackageVariant is
not needed. Rather, when the floating tag is moved to a new tagged revision of
the upstream package, the package revision controller will notice and
automatically propose and update to that revision. For example, the upstream
package author may designate three floating tags: stable, beta, and alpha. The
upstream package author can move these tags to specific revisions, and any
PackageVariant resource tracking them will propose updates to their downstream
packages.

### Adoption and Deletion Policies
When a PackageVariant resource is created, it will have a particular
repository and package name as the downstream. The adoption policy controls
whether the package variant controller takes over an existing package with that
name, in that repository.

Analogously, when a PackageVariant resource is deleted, a decision must be
made about whether or not to delete the downstream package. This is controlled
by the deletion policy.

## Fan Out of Variant Generation

When used with a single package, the package variant controller mostly helps us
handle the time dimension - producing new versions of a package as the upstream
changes, or as injected resources are updated. It can also be useful for
automating common, systematic changes made when bringing an external package
into an organization, or an organizational package into a team repository.

That is useful, but not extremely compelling by itself. More interesting is when
we use PackageVariant as a primitive for automations that act on other
dimensions of scale. That means writing controllers that emit PackageVariant
resources. For example, we can create a controller that instantiates a
PackageVariant for each developer in our organization, or we can create
a controller to manage PackageVariants across environments. The ability to not
only clone a package, but make systematic changes to that package enables
flexible automation.

Workload controllers in Kubernetes are a useful analogy. In Kubernetes, we have
different workload controllers such as Deployment, StatefulSet, and DaemonSet.
Ultimately, all of these result in Pods; however, the decisions about what Pods
to create, how to schedule them across Nodes, how to configure those Pods, and
how to manage those Pods as changes happen are very different with each workload
controller. Similarly, we can build different controllers to handle different
ways in which we want to generate PackageRevisions. The PackageVariant
resource provides a convenient primitive for all of those controllers, allowing
a them to leverage a range of well-defined operations to mutate the packages as
needed.

A common need is the ability to generate many variants of a package based on
a simple list of some entity. Some examples include generating package variants
to spin up development environments for each developer in an organization;
instantiating the same package, with slight configuration changes, across a
fleet of clusters; or instantiating some package per customer.

The package variant set controller is designed to fill this common need. This
controller consumes PackageVariantSet resources, and outputs PackageVariant
resources. The PackageVariantSet defines:
- the upstream package
- a list of targets
- rules for generating one PackageVariant per target

Three types of targeting are supported:
- An explicit list of repository and package names
- A label selector for Repository objects along with a way to generate package
  names from those objects
- An arbitrary object selector, along with ways to generate repository and
  package names from those objects

*Figure 5* shows an example of creating PackageVariant resources based upon the
explicity list of repository and package names.

![Figure 5: List of Targets](packagevariantset-target-package.png)

Rules for generating PackageVariant resources can also access the fields of
the target objects, allowing the package context values, injection selectors,
and function inputs to vary based upon the target.

## Example Use Cases

### Automatically Organizational Customization of an External Package
We can use a PackageVariant to provide basic changes that are needed when
importing a package from an external repository. For example, suppose we have
our own internal registry from which we pull all our images. When we import an
upstream package, we want to modify any images listed in the upstream package to
point to our registry. Additionally, we have a policy that all Pods must have a
`chargeback-code` label, so we want to add that as a validating policy to the
package.

In a manual CLI-based workflow, we would accomplish this as follows.
...

With PackageVariant, we instead create the following resources:
...

### Creating a Namespace Per Tenant

### Customizing A Workload By Region

### Customizing A Workload By Cluster and Region

### Customizing A Workload By Environment

## Detailed Design

### PackageVariant API

#### Basic API

- Upstream
- Downstream
- Annotations
- Labels
- Adoption and Deletion Policy
- Status
  - `Valid` Condition Type
  - `DownstreamEnsured` Condition Type (`Ready`?)
  - `DraftExists` Condition Type (?)

#### Package Context Injection

PackageVariant resource authors may specify key-value pairs in the
`spec.packageContext.data` field of the resource. These key-value pairs will be
automatically added to the `data` of the `kptfile.kpt.dev` ConfigMap, if it
exists.

Specifying the key `name` is invalid and must fail validation of the
PackageVariant. This key is reserved for kpt or Porch to set to the package
name.

The `spec.packageContext.removeKeys` field can also be used to specify a list of
keys that the package variant controller should remove from the `data` field of
the `kptfile.kpt.dev` ConfigMap.

When creating or updating a package, the package variant controller will ensure
that:
- The `kptfile.kpt.dev` ConfigMap exists (creating it if not)
- All of the key-value pairs in `spec.packageContext.data` exist in the `data`
  field of the ConfigMap.
- None of the keys listed in `spec.packageContext.removeKeys` exist in the
  ConfigMap.

Note that if a user adds a key via PackageVariant, then changes the
PackageVariant to no longer add that key, it will NOT be removed automatically,
unless the user also lists the key in the `removeKeys` list. This avoids the
need to track which keys were added by PackageVariant.

Similarly, if a user manually adds a key in the downstream that is also listed
in the `removeKeys` field, the package variant controller will remove that key
the next time it needs to update the downstream package. There will be no
attempt to coordinate "ownership" of these keys.

The package variant controller will add or set a condition of type
`ContextInjected` in the PackageVariant resource status. This will be set to
`True` if and only if:
- The user specified `spec.packageContext` (either or both fields).
- The `kptfile.kpt.dev` ConfigMap exists.
- The controller successfully modified it as specified.

Otherwise, the condition should be set to `False`.

If the controller is unable to modify the ConfigMap for some reason, this is
considered an error and should prevent generation of the Draft. This will result
in the condition `DownstreamEnsured` being set to `False`.

#### KRM Function Pipeline

PackageVariant resource creators may specify a list of KRM functions to execute
on package creation and update. These functions are listed in the field
`spec.mutators`, which is a slice of [`Function`](https://github.com/GoogleContainerTools/kpt/blob/cf1f326486214f6b4469d8432287a2fa705b48f5/pkg/api/kptfile/v1/types.go#L283) structs, just as in the Kptfile.

Note that there is no equivalent to the Kptfile `validators` list, as those
should live within the Kptfile itself if they are needed. The PackageVariant
function list is intended to act in a manner similar to a human executing
functions imperatively while customizing a package. Nonetheless, these functions
will be called whenever the PackageVariant resource, the upstream
package, or one of the injected configuration objects are updated, so they
should be idempotent, just as in other uses of KRM functions.

#### Configuration Injection Details

As described [above](#configuration-injection), configuration injection is a
process whereby in-package resources are matched to in-cluster resources, and
the `spec` of the in-cluster resources is copied to the in-package resource.

Configuration injection is controlled by a combination of in-package resources
with annotations, and *injectors* (also known as *injection selectors*) defined
on the PackageVariant resource. Package authors control the injection points
they allow in their packages, by flagging specific resources as *injection
points* with an annotation. Creators of the PackageVariant resource specify how
to map in-cluster resources to those injection points using the injection
selectors. Injection selectors are defined in the `spec.injectors` field of the
PackageVariant. This field is an ordered array of structs containing a GVK
(group, version, kind) tuple as separate fields, and name. Only the name is
required. To identify a match, all fields present must match the in-cluster
object, and all *GVK* fields present must match the in-package resource. In
general the name will not match the in-package resource; this is discussed in
more detail below.

The annotations, along with the GVK of the annotated resource, allow a package
to "advertise" the injections it can accept and understand. These injection
points effectively form a configuration API for the package, and the injection
selectors provide a way for the PackageVariant author to specify the inputs
for those APIs from the possible values in the management cluster. If we define
those APIs carefully, they can be used across many packages; since they are
KRM resources, we can apply versioning and schema validation to them as well.
This creates a more maintainable, automatable set of APIs for package
customization than simple key/value pairs.

As an example, we may define a GVK that contains service endpoints that many
applications use. In each application package, we would then include an instance
of that resource, say called "service-endpoints", and configure a function to
propagate the values from that resource to others within our package. As those
endpoints may vary by region, in our Porch cluster we can create an instance of
this GVK for each region: "useast1-service-endpoints",
"useast2-service-endpoints", "uswest1-service-endpoints", etc. When we
instantiate the PackageVariant for a cluster, we want to inject the resource
corresponding to the region in which the cluster exists. Thus, for each cluster
we will create a PackageVariant resource pointing to the upstream package, but
with injection selector name values that are specific to the region for that
cluster.

It is important to realize that the name of the in-package resource and the in-
cluster resource need not match. In fact, it would be an unusual coincidence if
they did match. The names in the package are the same across PackageVariants
using that upstream, but we want to inject different resources for each one such
PackageVariant. We also do not want to change the name in the package, because
it likely has meaning within the package and will be used by functions in the
package. Also, different owners control the names of the in-package and in-
cluster resources. The names in the package are in the control of the package
author. The names in the cluster are in the control of whoever populates the
cluster (for example, some infrastructure team). The selector is the glue
between them, and is in control of the PackageVariant resource creator.

The GVK on the other hand, has to be the same for the in-package resource and
the in-cluster resource, because it tells us the API schema for the resource.
Also, the namespace of the in-cluster object needs to be the same as the
PackageVariant resource, or we could leak resources from namespaces to which
our PackageVariant user does not have access.

With that understanding, the injection process works as follows:

1. The controller will examine all in-package resources, looking for those with
   an annotation named `kpt.dev/config-injection`, with one of the following
   values: `required` or `optional`. We will call these "injection points". It
   is the responsibility of the package author to define these injection points,
   and to specify which are required and which are optional. Optional injection
   points are a way of specifying default values.
1. For each injection point, a condition will be created in the
   downstream PackageRevision, with ConditionType set to the dot-delimited
   concatenation of `config.injection`, with the in-package resource kind and
   name, and the value set to `False`. Note that since the package author
   controls the name of the resource, kind and name are sufficient to
   disambiguate the injection point. We will call this ConditionType the
   "injection point ConditionType".
1. For each required injection point, the injection point ConditionType will
   be added to the PackageRevision `readinessGates` by the package variant
   controller. Optional injection points' ConditionTypes must not be added to
   the `readinessGates` by the package variant controller, but humans or other
   actors may do so at a later date, and the package variant controller should
   not remove them on subsequent reconciliations. Also, this relies upon
   `readinessGates` gating publishing the package to a *deployment* repository,
   but not gating publishing to a blueprint repository.
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
     injection selectors stops. Note that the namespace is set based upon the
     PackageVariant resource, the GVK is set based upon the in-package resource,
     and all selectors require name. Thus, at most one match is possible for any
     given selector. Also note that *all fields present in the selector* must
     match the in-cluster resource, and only the *GVK fields present in the
     selector* must match the in-package resoruce.
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
1. The package variant controller will add or set a condition of type
   `ConfigInjected` in the PackageVariant resource status.
   This will be set to `True` if and only if:
   - All configuration injection processing is complete.
   - There is no resource annotated as an injection point but having an invalid
     annotation value (i.e., other than `required` or `optional`).
   - Matching in-cluster resources were successfully injected for all `required`
     injection points (i.e., all required injection point ConditionTypes are
     `True`).
   - There are no ambiguous condition types due to conflicting GVK and name
     values. These must be disambiguated in the upstream package, if so.

If `ConfigInjected` is `False`, then `DownstreamEnsured` must also be `False`.

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
convenient. This allows a single injector to apply to multiple injection points
with different GVKs.

#### Order of Mutations

During creation, the first thing the controller does is clone the upstream
package to create the downstream package.

For update, first note that changes to the downstream PackageRevision can be
triggered for several different reasons:

1. The PackageVariant resource is updated, which could change any of the options
   for introducing variance, or could also change the upstream package revision
   referenced.
1. A new revision of the upstream package has been selected due to a floating
   tag change, or due to a force retagging of the upstream.
1. An injected in-cluster object is updated.

The downstream PackageRevision may have been updated by humans or other
automation actors since creation, so we cannot simply recreate the downstream
PackageRevision from scratch when one of these changes happens. Instead, the
controller must maintain the later edits by doing the equivalent of a `kpt pkg
update`, in the case of changes to the upstream for any reason. Any other
changes require reapplication of the PackageVariant functionality. With that
understanding, we can see that the controller will perform mutations on the
downstream package in this order, for both creation and update:

1. Create (via Clone) or Update (via `kpt pkg update` equivalent)
1. Package Context Injections
1. Kptfile KRM Function Pipeline Additions/Changes
1. Config Injection
1. Kptfile KRM Function Pipeline Execution

Since the middle three of these just edit resources (including the Kptfile) in
the package, their ordering does not matter; they cannot affect one another.
The execution of the KRM function pipeline depends on the others, but there are
no direct dependencies otherwise.

### PackageVariantSet API

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
  handled with PackageVariant / PackageVariantSet resources that differ only
  in their upstream / downstream. Theoretically we could do that with label
  selectors on packages but it gets really ugly really fast. I suspect just
  making people copy the PV / PVS is better.

## Footnotes
[^notimplemented]: Proposed here but not yet implemented as of Porch v0.0.16.
[^setns]: As of this writing, the `set-namespace` function does not have a
    `create` option. This should be added to avoid the user needing to also use
    the `upsert-resource` function. Such common operation should be simple for
    users. Another option is to build this into PackageVariant, though at this
    time we do not plan to do so.
[^pdc]: A prototype version of this was implemented in Nephio PackageDeployment,
    but this has not been implemented in PackageVariant as of Porch v0.0.16.

## Figure Legend

![Figure Legend](packagevariant-legend.png)

