# Package Orchestration

* Author(s): Martin Maly, @martinmaly
* Approver: @mortent

## Why

Customers who want to take advantage of the benefits of [Configuration as Data
](./06-config-as-data.md) can do so today using a [kpt](https://kpt.dev) CLI and
kpt function ecosystem, including [functions catalog](https://catalog.kpt.dev/).
Package authoring is possible using a variety of editors with
[YAML](https://yaml.org/) support. That said, a delightful UI experience
of WYSIWYG package authoring which supports broader package lifecycle, including
package authoring with *guardrails*, approval workflow, package deployment, and
more, is not yet available.

*Package Orchestration* service is the component that completes the set of core
Configuration as Data components - *CaD Core* - and enables building the
delightful UI experience supporting the configuration lifecycle.

## Core Concepts

This section briefly describes core concepts of package orchestration:

***Package***: Package is a collection of related configuration files containing
configuration of [KRM][krm] **resources**. Specifically, configuration
packages are [kpt packages](https://kpt.dev/).

***Repository***: Repositories store packages or [functions][].
For example [git][] or [OCI](#oci). Functions may be associated with
repositories to enforce constraints or invariants on packages (guardrails).
([more details](#repositories))

Packages are sequentially ***versioned***; multiple versions of the same package
may exist in a repository. [more details](#package-versioning))

A package may have a link (URL) to an ***upstream package*** (a specific
version) from which it was cloned. ([more details](#package-relationships))

Package may be in one of several lifecycle stages:
* ***Draft*** - package is being created or edited. The package contents can be
  modified but package is not ready to be used (i.e. deployed)
* ***Proposed*** - author of the package proposed that the package be published
* ***Published*** - the changes to the package have been approved and the
  package is ready to be used. Published packages can be deployed or cloned

***Function*** (specifically, [KRM functions][krm functions]) can be applied to
packages to mutate or validate resources within them. Functions can be applied
to a package to create specific package mutation while editing a package draft,
functions can be added to package's Kptfile [pipeline][], or associated with a
repository to be applied to all packages on changes.
([more details](#functions))

A repository can be designated as ***deployment repository***. *Published*
packages in a deployment repository is considered deployment-ready.
([more details](#deployment))

<!-- Reference links -->
[krm]: https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/resource-management.md
[functions]: https://kpt.dev/book/02-concepts/03-functions
[krm functions]: https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md
[pipeline]: https://kpt.dev/book/04-using-functions/01-declarative-function-execution
[Config Sync]: https://cloud.google.com/anthos-config-management/docs/config-sync-overview
[kpt]: https://kpt.dev/
[git]: https://git-scm.org/
[optimistic-concurrency]: https://en.wikipedia.org/wiki/Optimistic_concurrency_control
[apiserver]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/
[representation]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#differing-representations
[crds]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/

## CaD Core Components

The Core implementation of Configuration as Data, *CaD Core*, is a set of
components and APIs which collectively enable:

* Registration of repositories (Git, OCI) containing kpt packages or functions,
  and discovery of packages and functions
* Porcelain package lifecycle, including authoring, versioning, deletion,
  creation and mutations of a package draft, process of proposing the package
  draft, and publishing of the approved package.
* Package lifecycle operations such as:
  * assisted or automated rollout of package upgrade when a new version
    of the upstream package version becomes available
  * rollback of a package to previous version
* Deployment of packages from deployment repositories and observability of their
  deployment status.
* Permission model that allows role-based access control

### High-Level Architecture

At the high level, the Core CaD functionality comprises:

* a generic (i.e. not task-specific) package orchestration service implementing
  * package repository management
  * package discovery, authoring and lifecycle management
* [kpt][] - a Git-native, schema-aware, extensible client-side tool for
  managing KRM packages
* [Config Sync][] - configuration distribution and deployment mechanism with
  observability of the status of deployed resources
* a task-specific UI supporting repository management, package discovery,
  authoring, and lifecycle

![CaD Core Architecture](./CaD%20Core%20Architecture.svg)

## CaD Concepts Elaborated

Concepts briefly introduced above are elaborated in more detail in this section.

### Repositories

[Config Sync][] and [kpt][] currently integrate with [git][] repositories, and
there is an existing design to add [OCI support](./02-oci-support.md) to kpt.
Initially, the CaD Core system will prioritize integration with [git][], and
support for additional repository types may be added in the future as required.

Requirements applicable to all repositories include: ability to store packages,
their versions, and sufficient metadata associated with package to capture:

* package dependency relationships (upstream - downstream)
* package lifecycle state (draft, proposed, published)
* package purpose (base package)
* (optionally) even customer-defined attributes

At repository registration, customers must be able to specify details needed to
store packages in appropriate locations in the repository. For example,
registration of a Git repository must accept a branch and a directory.

Repositories may have associated guardrails - mutation and validation functions
that ensure and enforce requirements of all packages in the repository,
including gating promotion of a package to a *published* lifecycle stage.

_Note_: A user role with sufficient permissions can register a package or
function repository, including repositories containing functions authored by
the customer, or other providers. Since the functions in the registered
repositories become discoverable, customers must be aware of the implications of
registering function repositories and trust the contents thereof.

### Package Versioning

Packages are sequentially versioned. The important requirements are:

* ability to compare any 2 versions of a package to be either "newer than",
  equal, or "older than" relationship
* ability to support automatic assignment of versions
* ability to support [optimistic concurrency][optimistic-concurrency] of package
  changes via version numbers
* simple model which easily supports automation

We plan to use a simple integer sequence to represent package versions.

### Package Relationships

Kpt packages support the concept of ***upstream***. When a package is cloned
from another, the new package (called ***downstream*** package) maintains an
upstream link to the specific version of the package from which it was cloned.
If a new version of the upstream package becomes available, the upstream link
can be used to [update](https://kpt.dev/book/03-packages/05-updating-a-package)
the downstream package.

### Deployment

[Config Sync][] is the deployment mechanism used by CaD Core implementation.
Here we highlight some key attributes of the deployment mechanism and its
integration within the CaD Core:

* _Published_ packages in a deployment repository are considered ready to be
  deployed
* Config Sync supports deploying individual packages and whole repositories.
  For Git specifically that translates to a requirement to be able to specify
  repository, branch/tag/ref, and directory when instructing Config Sync to
  deploy a package.
* _Draft_ packages need to be identified in such a way that Config Sync can
  easily avoid deploying them.
* Config Sync needs to be able to pin to specific versions of deployable
  packages in order to orchestrate rollouts and rollbacks. This means it must
  be possible to GET a specific version of a package.
* Config Sync needs to be able to discover when new versions are available for
  deployment.

### Functions

Functions, specifically [KRM functions][krm functions], are used in the CaD core
to manipulate resources within packages.

* Similar to packages, functions are stored in repositories. Some repositories
  (such as OCI) are more suitable for storing functions than others (such as
  Git).
* Function discovery will be aided by metadata associated with the function
  by which the function can advertise which resources it acts on, whether the
  function is idempotent or not, whether it is a mutator or validator, etc.
* Function repositories can be registered and subsequently, user can discover
  functions from the registered repositories and use them as follows:

Function can be:

* applied imperatively to a package draft to perform specific mutation to the
  package's resources or meta-resources (`Kptfile` etc.)
* registered in the package's `Kptfile` function pipeline as a *mutator* or
  *validator* in order to be automatically run as part of package rendering
* registered at the repository level as *mutator* or *validator*. Such function
  then applies to all packages in the repository and is evaluated whenever a
  change to a package in the repository occurs.

## Package Orchestration - Porch

Having established the context of the CaD Core components and the overall
architecture, the remainder of the document will focus on **Porch** - Package
Orchestration service.

To reiterate the role of Package Orchestration service among the CaD Core
components, it is:

* [Repository Management](#repository-management)
* [Package Discovery](#package-discovery)
* [Package Authoring](#package-authoring) and Lifecycle

In the following section we'll expand more on each of these areas. The term
_client_ used in these sections can be either a person interacting with the UI
such as a web application or a command-line tool, or an automated agent or
process.

### Repository Management

The repository management functionality of Package Orchestration service enables
the client to:

* register, unregister, update registration of repositories, and discover
  registered repositories. Git repository integration will be available first,
  with OCI and possibly more delivered in the subsequent releases.
* manage repository-wide upstream/downstream relationships, i.e. designate
  default upstream repository from which packages will be cloned.
* annotate repository with metadata such as whether repository contains
  deployment ready packages or not; metadata can be application or customer
  specific
* define and enforce package invariants (guardrails) at the repository level, by
  registering mutator and/or validator functions with the repository; those
  registered functions will be applied to packages in the repository to enforce
  invariants

### Package Discovery

The package discovery functionality of Package Orchestration service enables
the client to:

* browse packages in a repository
* discover configuration packages in registered repositories and sort/filter
  based on the repository containing the package, package metadata, version,
  package lifecycle stage (draft, proposed, published)
* retrieve resources and metadata of an individual package, including latest
  version or any specific version or draft of a package, for the purpose of
  introspection of a single package or for comparison of contents of multiple
  versions of a package, or related packages
* enumerate _upstream_ packages available for creating (cloning) a _downstream_
  package
* identify downstream packages that need to be upgraded after a change is made
  to an upstream package
* identify all deployment-ready packages in a deployment repository that are
  ready to be synced to a deployment target by Config Sync
* identify new versions of packages in a deployment repository that can be
  rolled out to a deployment target by Config Sync
* discover functions in registered repositories based on filtering criteria
  including containing repository, applicability of a function to a specific
  package or specific resource type(s), function metadata (mutator/validator),
  idempotency (function is idempotent/not), etc.

### Package Authoring

The package authoring and lifecycle functionality of the package Orchestration
service enables the client to:

* Create a package _draft_ via one of the following means:
  * an empty draft 'from scratch' (equivalent to
    [kpt pkg init](https://kpt.dev/reference/cli/pkg/init/))
  * clone of an upstream package (equivalent to
    [kpt pkg get](https://kpt.dev/reference/cli/pkg/get/)) from either a
    registered upstream repository or from another accessible, unregistered,
    repository
  * edit an existing package (similar to the CLI command(s)
    [kpt fn source](https://kpt.dev/reference/cli/fn/source/) or
    [kpt pkg pull](https://github.com/GoogleContainerTools/kpt/issues/2557))
  * roll back / restore a package to any of its previous versions
    ([kpt pkg pull](https://github.com/GoogleContainerTools/kpt/issues/2557)
    of a previous version)
* Apply changes to a package _draft_. In general, mutations include
  adding/modifying/deleting any part of the package's contents. Some specific
  examples include:
  * add/change/delete package metadata (i.e. some properties in the `Kptfile`)
  * add/change/delete resources in the package
  * add function mutators/validators to the package's [pipeline][]
  * invoke a function imperatively on the package draft to perform a desired
    mutation
  * add/change/delete sub-package
  * retrieve the contents of the package for arbitrary client-side mutations
    (equivalent to [kpt fn source](https://kpt.dev/reference/cli/fn/source/))
  * update/replace the package contents with new contents, for example results
    of a client-side mutations by a UI (equivalent to
    [kpt fn sink](https://kpt.dev/reference/cli/fn/sink/))
* Rebase a package onto another upstream base package
  ([detail](https://github.com/GoogleContainerTools/kpt/issues/2548)) or onto
  a newer version of the same package (to aid with conflict resolution during
  the process of publishing a draft package)
* Get feedback during package authoring, and assistance in recovery from:
  * merge conflicts, invalid package changes, guardrail violations
  * compliance of the drafted package with repository-wide invariants and
    guardrails
* Propose for a _draft_ package be _published_.
* Apply an arbitrary decision criteria, and by a manual or automated action,
  approve (or reject) proposal of a _draft_ package to be _published_.
* Perform bulk operations such as:
  * Assisted/automated update (upgrade, rollback) of groups of packages matching
    specific criteria (i.e. base package has new version or specific base
    package version has a vulnerability and should be rolled back)
  * Proposed change validation (pre-validating change that adds a validator
    function to a base package or a repository)
* Delete an existing package.

#### Authoring & Latency

An important goal of the Package Orchestration service is to support building
of task-specific UIs. In order to deliver low latency user experience acceptable
to UI interactions, the innermost authoring loop (depicted below) will require:

* high performance access to the package store (load/save package) w/ caching
* low latency execution of mutations and transformations on the package contents
* low latency [KRM function][krm functions] evaluation and package rendering
  (evaluation of package's function pipelines)

![Inner Loop](./Porch%20Inner%20Loop.svg)

#### Authoring & Access Control

A client can assign actors (persons, service accounts) to roles that determine
which operations they are allowed to perform in order to satisfy requirements
of the basic roles. For example, only permitted roles can:

* manipulate repository registration, enforcement of repository-wide
  invariants and guardrails
* create a draft of a package and propose the draft be published
* approve (or reject) the proposal to publish a draft package
* clone a package from a specific upstream repository
* perform bulk operations such as rollout upgrade of downstream packages,
  including rollouts across multiple downstream repositories
* etc.

### Porch Architecture

The Package Orchestration service, **Porch** is designed to be hosted in a
[Kubernetes](https://kubernetes.io/) cluster, just like [Config Sync][].

The overall architecture is shown below, and includes also existing components
(k8s apiserver and Config Sync).

![](./Porch%20Architecture.svg)

In addition to satisfying requirements highlighted above, the focus of the
architecture was to:

* establish clear components and interfaces
* support a low-latency package authoring experience required by the UIs

The Porch components are:

#### Porch Server

The Porch server is implemented as [Kubernetes extension API server][apiserver].
The benefits of using Kubernetes extension API server are:

* well-defined and familiar API style
* availability of generated clients
* integration with existing Kubernetes ecosystem and tools such as `kubectl`
  CLI, [RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
* avoids requirement to open another network port to access a separate endpoint
  running inside k8s cluster (this is a distinct advantage over gRPC which we
  considered as an alternative approach)

Resources implemented by Porch include:

* `PackageRevision` - represents the _metadata_ of the configuration package
  revision stored in a _package_ repository.
* `PackageRevisionResources` - represents the _contents_ of the package revision
* `Function` - represents a [KRM function][krm functions] discovered in
  a registered _function_ repository.

Note that each configuration package revision is represented by a _pair_ of
resources which each present a different view (or [representation][] of the same
underlying package revision.

Repository registration is supported by a `Repository` [custom resource][crds].

**Porch server** itself comprises several key components, including:

* The *Porch aggregated apiserver* which implements the integration into the
  main Kubernetes apiserver, and directly serves API requests for the
  `PackageRevision`, `PackageRevisionResources` and `Function` resources.
* Package orchestration *engine* which implements the package lifecycle
  operations, and package mutation workflows
* *CaD Library* which implements specific package manipulation algorithms such
  as package rendering (evaluation of package's function *pipeline*),
  initialization of a new package, etc. The CaD Library is shared with `kpt`
  where it likewise provides the core package manipulation algorithms.
* *Package cache* which enables both local caching, as well as abstract
  manipulation of packages and their contents irrespectively of the underlying
  storage mechanism (Git, or OCI)
* *Repository adapters* for Git and OCI which implement the specific logic of
  interacting with those types of package repositories.
* *Function runtime* which implements support for evaluating
  [kpt functions][functions] and multi-tier cache of functions to support
  low latency function evaluation

#### Function Runner

**Function runner** is a separate service responsible for evaluating
[kpt functions][functions]. Function runner exposes a [gRPC](https://grpc.io/)
endpoint which enables evaluating a kpt function on the provided configuration
package.

The gRPC technology was chosen for the function runner service because the
[requirements](#grpc-api) that informed choice of KRM API for the Package
Orchestration service do not apply. The function runner is an internal
microservice, an implementation detail not exposed to external callers. This
makes gRPC perfectly suitable.

The function runner also maintains cache of functions to support low latency
function evaluation.

#### CaD Library

The [kpt](https://kpt.dev/) CLI already implements foundational package
manipulation algorithms in order to provide the command line user experience,
including:

* [kpt pkg init](https://kpt.dev/reference/cli/pkg/init/) - create an empty,
  valid, KRM package
* [kpt pkg get](https://kpt.dev/reference/cli/pkg/get/) - create a downstream
  package by cloning an upstream package; set up the upstream reference of the
  downstream package
* [kpt pkg update](https://kpt.dev/reference/cli/pkg/update/) - update the
  downstream package with changes from new version of upstream, 3-way merge
* [kpt fn eval](https://kpt.dev/reference/cli/fn/eval/) - evaluate a kpt
  function on a package
* [kpt fn render](https://kpt.dev/reference/cli/fn/render/) - render the package
  by executing the function pipeline of the package and its nested packages
* [kpt fn source](https://kpt.dev/reference/cli/fn/source/) and
  [kpt fn sink](https://kpt.dev/reference/cli/fn/sink/) - read package from
  local disk as a `ResourceList` and write package represented as
  `ResourcesList` into local disk

The same set of primitives form the foundational building blocks of the package
orchestration service. Further, the package orchestration service combines these
primitives into higher-level operations (for example, package orchestrator
renders packages automatically on changes, future versions will support bulk
operations such as upgrade of multiple packages, etc).

The implementation of the package manipulation primitives in kpt was refactored
(with initial refactoring completed, and more to be performed as needed) in
order to:

* create a reusable CaD library, usable by both kpt CLI and Package
  Orchestration service
* create abstractions for dependencies which differ between CLI and Porch,
  most notable are dependency on Docker for function evaluation, and dependency
  on the local file system for package rendering.

Over time, the CaD Library will provide the package manipulation primitives:

* create a valid empty package (init)
* update package upstream pointers (get)
* perform 3-way merge (update)
* render - core package rendering algorithm using a pluggable function evaluator
  to support:
  * function evaluation via Docker (used by kpt CLI)
  * function evaluation via an RPC to a service or appropriate function sandbox
  * high-performance evaluation of trusted, built-in, functions without sandbox
* heal configuration (restore comments after lossy transformation)

and both kpt CLI and Porch will consume the library. This approach will allow
leveraging the investment already made into the high quality package
manipulation primitives, and enable functional parity between KPT CLI and
Package Orchestration service.

## User Guide

Customer will be interacting with Package Orchestration service primarily
_indirectly_, either via the `kpt` CLI or a purpose-built, task-specific UI.
Here we cover the use from the perspective of the `kpt` CLI.

Installation of Porch, including prerequisites, is covered in a
[dedicated document](../../porch/docs/running-on-gke.md) and therefore will not
be elaborated here; focus will be on user experience with the Package
Orchestration service once installed.

### Repository Registration

To start working with Package Orchestration service, first register one or more
configuration package repositories using `kpt alpha repo register` command:

```sh
# Register a Git repository:

GITHUB_USERNAME=<your github username>
GITHUB_TOKEN=<GitHub Personal Access Token>

$ kpt alpha repo register \
  --namespace default \
  --repo-username=${GITHUB_USERNAME} \
  --repo-password=${GITHUB_TOKEN} \
  https://github.com/${GITHUB_USERNAME}/blueprints.git
```

All command line flags supported:

* `--directory` - Directory within the repository where to look for packages.
* `--branch` - Branch in the repository where finalized packages are committed.
  (defaults to `main`)
* `--name` - Name of the package repository Kubernetes resource. If unspecified,
  will default to the name portion (last segment) of the repository URL
  (`blueprint` in the example above)
* `--description` - Brief description of the package repository.
* `--deployment` - Boolean value; If specified, repository is a deployment
  repository; published packages in a deployment repository are considered
  deployment-ready.
* `--repo-username` - Username for repository authentication.
* `--repo-password` - Password for repository authentication.

Additionally, common `kubectl` command line flags for controlling aspects of
interaction with the Kubernetes apiserver, logging, and more (this is true for
all `kpt` CLI commands which interact with Porch).

Use the `kpt alpha repo get` command to query registered repositories:

```sh
# Query registered repositories
$ kpt alpha repo get

NAME         TYPE  CONTENT  DEPLOYMENT  READY  ADDRESS
blueprints   git   Package              True   https://github.com/platkrm/blueprints.git
deployments  git   Package  true        True   https://github.com/platkrm/deployments.git
```

The `kpt alpha <group> get` commands support common `kubectl`
[flags](https://kubernetes.io/docs/reference/kubectl/cheatsheet/#formatting-output)
to format output, for example `kpt alpha repo get --output=yaml`.

The command `kpt alpha repo unregister` can be used to unregister a repository:

```sh
# Unregister a repository
$ kpt alpha repo unregister deployments
```

### Package Discovery And Introspection

The `kpt alpha rpkg` command group contains commands for interacting with
packages managed by the Package Orchestration service. the `r` prefix used
in the command group name stands for 'remote'.

The `kpt alpha rpkg get` command list the packages in registered repositories:

```sh
# List package revisions in registered repositories
$ kpt alpha rpkg get

NAME                                                 PACKAGE  REVISION  LATEST  LIFECYCLE  REPOSITORY
blueprints-0349d71330b89ee48ac85167598ef23021fd0484  basens   main      false   Published  blueprints
blueprints-2e47615fda05664491f72c58b8ab658683afa036  basens   v1        true    Published  blueprints
blueprints-7e2fe44bfdbb744d49bdaaaeac596200102c5f7c  istions  main      false   Published  blueprints
blueprints-ac6e872be4a4a3476922deca58cca3183b16a5f7  istions  v1        false   Published  blueprints
blueprints-421a5b5e43b03bc697d96f471929efc6ba3f54b3  istions  v2        true    Published  blueprints
...
```

The `LATEST` column indicates whether the package revision is the latest among
the revisions of the same package. In the output above, v4 is the latest
revision of `istions` package and `v1` is the latest revision of `basens`
package.

The `LIFECYCLE` column indicates the lifecycle stage of the package revision,
one of: `Published`, `Draft` or `Proposed`.

*Note* on package revision names. Packages exist in a hierarchical directory
structure maintained by the underlying repository such as git, or in a
filesystem bundle of OCI images. The hierarchical, filesystem-compatible names
of packages do not satisfy the Kubernetes naming
[constraints](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names).
Therefore, the names of the Kubernetes resources representing package revisions
are computed as a hash.

Simple filtering of package revisions by name (substring) and revision (exact
match) is supported by the CLI using `--name` and `--revision` flags:

```sh
# List package with `istio` in the package name, and `v2` revision
$ kpt alpha rpkg get --name istio --revision=v2

NAME                                                 PACKAGE  REVISION  LATEST  LIFECYCLE  REPOSITORY
blueprints-421a5b5e43b03bc697d96f471929efc6ba3f54b3  istions  v2        true    Published  blueprints
```

The common `kubectl` flags that control output format are available as well:

```sh
# Get the package revision in YAML format
$ kpt alpha rpkg get blueprints-421a5b5e43b03bc697d96f471929efc6ba3f54b3 -ndefault -oyaml

apiVersion: porch.kpt.dev/v1alpha1
kind: PackageRevision
metadata:
  labels:
    kpt.dev/latest-revision: "true"
  name: blueprints-421a5b5e43b03bc697d96f471929efc6ba3f54b3
  namespace: default
spec:
  lifecycle: Published
  packageName: istions
  repository: blueprints
  revision: v2
...
```

The `kpt alpha rpkg pull` command can be used to read the package resources.

The command can be used to print the package revision resources as
`ResourceList` to `stdout`, which enables
[chaining](https://kpt.dev/book/04-using-functions/02-imperative-function-execution?id=chaining-functions-using-the-unix-pipe)
evaluation of functions on the package revision pulled from the Package
Orchestration server.

```sh
# Pull package revision resources, output as ResourceList to stdout
$ kpt alpha rpkg pull blueprints-421a5b5e43b03bc697d96f471929efc6ba3f54b3 -ndefault

apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: istions
...
```

Or, the package contents can be saved on local disk for direct introspection
or editing:

```sh
# Pull package revision resources, save to local disk into `./istions` directory
$ kpt alpha rpkg pull blueprints-421a5b5e43b03bc697d96f471929efc6ba3f54b3 ./istions -ndefault

# Explore the package contents
$ find istions

istions
istions/istions.yaml
istions/README.md
istions/Kptfile
istions/package-context.yaml
...
```

### Authoring Packages

Several commands in the `kpt alpha rpkg` group support package authoring:

* `init` - Initializes a new package revision in the target repository.
* `clone` - Creates a clone of a source package in the target repository.
* `edit` - Creates a copy of a source package in the target repository to start
  creation of a new revision of the package.
* `push` - Pushes package resources into a remote package.
* `del` - Deletes one or more packages in registered repositories.


The `kpt alpha rpkg init` command can be used to initialize a new package
revision:

```sh
# Initialize a new (empty) package revision:
$ kpt alpha rpkg init new-package --repository=deployments --revision=v1 --ndefault

deployments-c32b851b591b860efda29ba0e006725c8c1f7764 created

# List the available package revisions.
$ kpt alpha rpkg get

NAME                                                  PACKAGE      REVISION  LATEST  LIFECYCLE  REPOSITORY
deployments-c32b851b591b860efda29ba0e006725c8c1f7764  new-package  v1        false   Draft      deployments
...
```

The new package is created in the `Draft` lifecycle stage. This is true also for
all commands that create new package revision (`init`, `clone` and `edit`).

Additional flags supported by the `kpt alpha rpkg init` command are:

* `--repository` - Repository in which the package will be created.
* `--revision` - Revision of the new package.
* `--description` -  Short description of the package.
* `--keywords` - List of keywords for the package.
* `--site` - Link to page with information about the package.


Use `kpt alpha rpkg clone` command to create a _downstream_ package by cloning
an _upstream_ package:

```sh
# Clone an upstream package to create a downstream package
$ kpt alpha rpkg clone blueprints-421a5b5e43b03bc697d96f471929efc6ba3f54b3 istions-clone \
  --repository=deployments -ndefault
deployments-11ca1db650fa4bfa33deeb7f488fbdc50cdb3b82 created

# Confirm the package revision was created
kpt alpha rpkg get deployments-11ca1db650fa4bfa33deeb7f488fbdc50cdb3b82 -ndefault
NAME                                                   PACKAGE         REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-11ca1db650fa4bfa33deeb7f488fbdc50cdb3b82   istions-clone   v1         false    Draft       deployments
```

`kpt alpha rpkg clone` can also be used to clone packages that are in
repositories not registered with Porch, for example:

```sh
# Clone a package from Git repository directly (repository is not registered)
$ kpt alpha rpkg clone \
  https://github.com/GoogleCloudPlatform/blueprints.git cloned-bucket \
  --directory=catalog/bucket \
  --ref=main \
  --repository=deployments \
  --namespace=default
deployments-e06c2f6ec1afdd8c7d977fcf204e4d543778ddac created

# Confirm the package revision was created
kpt alpha rpkg get deployments-e06c2f6ec1afdd8c7d977fcf204e4d543778ddac -ndefault
NAME                                                   PACKAGE         REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-e06c2f6ec1afdd8c7d977fcf204e4d543778ddac   cloned-bucket   v1         false    Draft       deployments
```

The flags supported by the `kpt alpha rpkg clone` command are:

* `--directory` - Directory within the upstream repository where the upstream
  package is located.
* `--ref` - Ref in the upstream repository where the upstream package is
  located. This can be a branch, tag, or SHA.
* `--repository` - Repository to which package will be cloned (downstream
  repository).
* `--revision` - Revision to assign to the downstream package.
* `--strategy` - Update strategy that should be used when updating this package;
  one of: `resource-merge`, `fast-forward`, `force-delete-replace`.


The `kpt alpha rpkg edit` command can be used to create a new revision of an
existing package. It is a means to modifying an already published package
revision.

```sh
# Create a new revision of an existing package
$ kpt alpha rpkg edit \
  blueprints-421a5b5e43b03bc697d96f471929efc6ba3f54b3 istions \
  --repository=blueprints --revision=v3 -ndefault

# Confirm the package revision was created
$ kpt alpha rpkg get blueprints-bf11228f80de09f1a5dd9374dc92ebde3b503689 -ndefault
NAME                                                  PACKAGE   REVISION   LATEST   LIFECYCLE   REPOSITORY
blueprints-bf11228f80de09f1a5dd9374dc92ebde3b503689   istions   v3         false    Draft       blueprints
```

The `kpt alpha rpkg push` command can be used to update the resources (package
contents) of a package _draft_:

```sh
# Pull package draft contents into a local directory
$ kpt alpha rpkg pull \
  deployments-c32b851b591b860efda29ba0e006725c8c1f7764 ./new-package -ndefault

# Make edits using your favorite YAML editor, for example adding a new resource
$ cat <<EOF > ./new-package/config-map.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config-map
data:
  color: orange
EOF

# Push the updated contents to the Package Orchestration server, updating the
# package contents.
$ kpt alpha rpkg push \
  deployments-c32b851b591b860efda29ba0e006725c8c1f7764 ./new-package -ndefault

# Confirm that the remote package now includes the new ConfigMap resource
$ kpt alpha rpkg pull deployments-c32b851b591b860efda29ba0e006725c8c1f7764 -ndefault

apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
...
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: example-config-map
  data:
    color: orange
...
```

Package revision can be deleted using `kpt alpha rpkg del` command:

```sh
# Delete package revision
$ kpt alpha rpkg del blueprints-bf11228f80de09f1a5dd9374dc92ebde3b503689 -ndefault

blueprints-bf11228f80de09f1a5dd9374dc92ebde3b503689 deleted
```

### Package Lifecycle and Approval Flow

Authoring is performed on the package revisions in the _Draft_ lifecycle stage.
Before a package can be deployed or cloned, it must be _Published_. The approval
flow is the process by which the package is advanced from _Draft_ state through
_Proposed_ state and finally to _Published_ lifecycle stage.

The commands used to manage package lifecycle stages include:

* `propose` - Proposes to finalize a package revision draft
* `approve` - Approves a proposal to finalize a package revision.
* `reject`  - Rejects a proposal to finalize a package revision

In the [Authoring Packages](#authoring-packages) section above we created
several _draft_ packages and in this section we will create proposals for
publishing some of them.

```sh
# List package revisions to identify relevant drafts:
$ kpt alpha rpkg get
NAME                                                   PACKAGE         REVISION   LATEST   LIFECYCLE   REPOSITORY
...
deployments-e06c2f6ec1afdd8c7d977fcf204e4d543778ddac   cloned-bucket   v1         false    Draft       deployments
deployments-11ca1db650fa4bfa33deeb7f488fbdc50cdb3b82   istions-clone   v1         false    Draft       deployments
deployments-c32b851b591b860efda29ba0e006725c8c1f7764   new-package     v1         false    Draft       deployments

# Propose two packge revisions to be be published
$ kpt alpha rpkg propose \
  deployments-11ca1db650fa4bfa33deeb7f488fbdc50cdb3b82 \
  deployments-c32b851b591b860efda29ba0e006725c8c1f7764 \
  -ndefault

deployments-11ca1db650fa4bfa33deeb7f488fbdc50cdb3b82 proposed
deployments-c32b851b591b860efda29ba0e006725c8c1f7764 proposed

# Confirm the package revisions are now Proposed
$ kpt alpha rpkg get
NAME                                                   PACKAGE         REVISION   LATEST   LIFECYCLE   REPOSITORY
...
deployments-e06c2f6ec1afdd8c7d977fcf204e4d543778ddac   cloned-bucket   v1         false    Draft       deployments
deployments-11ca1db650fa4bfa33deeb7f488fbdc50cdb3b82   istions-clone   v1         false    Proposed    deployments
deployments-c32b851b591b860efda29ba0e006725c8c1f7764   new-package     v1         false    Proposed    deployments
```

At this point, a person in _platform administrator_ role, or even an automated
process, will review and either approve or reject the proposals. To aid with the
decision, the platform administrator may inspect the package contents using the
commands above, such as `kpt alpha rpkg pull`.

```sh
# Approve a proposal to publish a package revision
$ kpt alpha rpkg approve deployments-11ca1db650fa4bfa33deeb7f488fbdc50cdb3b82 -ndefault
deployments-11ca1db650fa4bfa33deeb7f488fbdc50cdb3b82 approved

# Reject a proposal to publish a package revision
$ kpt alpha rpkg reject deployments-c32b851b591b860efda29ba0e006725c8c1f7764 -ndefault
deployments-c32b851b591b860efda29ba0e006725c8c1f7764 rejected
```

Now the user can confirm lifecycle stages of the package revisions:

```sh
# Confirm package revision lifecycle stages after approvals:
$ kpt alpha rpkg get
NAME                                                   PACKAGE         REVISION   LATEST   LIFECYCLE   REPOSITORY
...
deployments-e06c2f6ec1afdd8c7d977fcf204e4d543778ddac   cloned-bucket   v1         false    Draft       deployments
deployments-11ca1db650fa4bfa33deeb7f488fbdc50cdb3b82   istions-clone   v1         true     Published   deployments
deployments-c32b851b591b860efda29ba0e006725c8c1f7764   new-package     v1         false    Draft       deployments
```

Observe that the rejected proposal returned the package revision back to _Draft_
lifecycle stage. The package whose proposal was approved is now in _Published_
state.

### Deploying a Package

Commands used in the context of deploying a package include are in the
`kpt alpha sync` command group (named `sync` to emphasize that Config Sync
is the deploying mechanism and that configuration is being synchronized with the
actuation target as a means of deployment) and include:

* `create` - Creates a sync of a package in the deployment cluster.
* `del` - Deletes the package RootSync.
* `get` - Gets a RootSync resource with which package was deployed.


```sh
# Create a sync resource to deploy a package using Config Sync
$ kpt alpha sync create -ndefault \
  --package=deployments-11ca1db650fa4bfa33deeb7f488fbdc50cdb3b82 \
  sync-istions-clone

Created RootSync config-management-system/sync-istions-clone

# Get the status of the sync resource
$ kpt alpha sync get -nconfig-management-system sync-istions-clone -oyaml
apiVersion: configsync.gke.io/v1beta1
kind: RootSync
metadata:
  name: sync-istions-clone
  namespace: config-management-system
...

# Delete the sync resource
$ kpt alpha sync delete -nconfig-management-system sync-istions-clone
Deleting synced resources
Waiting for deleted resources to be removed
Sync sync-istions-clone successfully deleted
```

## Open Issues/Questions

### Deployment Rollouts & Orchestration

__Not Yet Resolved__

Cross-cluster rollouts and orchestration of deployment activity. For example,
package deployed by Config Sync in cluster A, and only on success, the same
(or a different) package deployed by Config Sync in cluster B.

## Alternatives Considered

### gRPC API

We considered the use of [gRPC]() for the Porch API. The primary advantages of
implementing Porch as an extension Kubernetes apiserver are:
* customers won't have to open another port to their Kubernetes cluster and can
  reuse their existing infrastructure
* customers can likewise reuse existing, familiar, Kubernetes tooling ecosystem
