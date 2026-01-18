---
title: "Chapter 2: Concepts"
linkTitle: "Chapter 2: Concepts"
description: This describes what is kpt and what are the main concepts behind it
toc: true
menu:
  main:
    parent: "Book"
    weight: 20
---

## What is kpt?

kpt (Kubernetes Package Transformation) supports management of Configuration as Data (CaD).

*Configuration as Data* is an approach to management of configuration (incl. configuration of infrastructure, policy,
services, applications, etc.)  which:

* makes configuration data the source of truth, stored separately from the live
  state
* uses a uniform, serializable data model to represent configuration
* separates code that acts on the configuration from the data and from packages / bundles of the data
* abstracts configuration file structure and storage from operations that act upon the configuration data; clients
  manipulating configuration data don’t need to directly interact with storage (git, container images)

This enables machine manipulation of configuration for Kubernetes and any infrastructure represented in the 
[Kubernetes Resource Model (KRM)](https://github.com/kubernetes/design-proposals-archive/blob/main/architecture/resource-management.md).

![img](/images/cad-overview.svg)

### Configuration as data key principles

* secrets should be stored separately, in a secret-focused storage system
* stores a versioned history of configuration changes by change sets to bundles of related configuration data
* relies on uniformity and consistency of the configuration format, including type metadata, to enable pattern-based
  operations on the configuration data, along the lines of duck typing
* separates schemas for the configuration data from the data, and relies on schema information for strongly typed
  operations and to disambiguate data structures and other variations within the model
* decouples abstractions of configuration from collections of configuration data
* represents abstractions of configuration generators as data with schemas, like other configuration data
* finds, filters / queries / selects, and/or validates configuration data that can be operated on by given code
  (functions)
* finds and/or filters / queries / selects code (functions) that can operate on resource types contained within a body
  of configuration data
* actuation (reconciliation of configuration data with live state) is separate from transformation of configuration
  data, and is driven by the declarative data model
* transformations, particularly value propagation, are preferable to wholesale configuration generation except when the
  expansion is dramatic (say, >10x)
* transformation input generation should usually be decoupled from propagation
* deployment context inputs should be taken from well defined “provider context” objects
* identifiers and references should be declarative
* live state should be linked back to sources of truth (configuration)

### Components of the kpt toolchain

The kpt toolchain includes the following components:

- [**kpt CLI**](../../reference/cli/): The kpt CLI supports package and function operations, and also   deployment, via
  either direct apply or GitOps. By keeping an inventory of deployed resources, kpt enables resource pruning, aggregated
  status and observability, and an improved preview experience.

- [**Function SDK**](https://github.com/kptdev/krm-functions-sdk): Any general-purpose or domain-specific language can be used to create functions to transform
 and/or validate the YAML KRM input/output format, but we provide SDKs to simplify the function authoring process, in 
  [Go](../05-developing-functions/#developing-in-Go). 

- [**Function catalog**](https://catalog.kpt.dev/function-catalog): A catalog of off-the-shelf, tested functions. kpt makes
  configuration easy to create and transform, via reusable functions. Because they are expected to be used for in-place
  transformation, the functions need to be idempotent.

## Packages
  
kpt manages KRM resources in bundles called **packages**.

Off-the-shelf packages are rarely deployed without any customization. Like [kustomize](https://kustomize.io), kpt
applies transformation **functions**, using the same
[KRM function specification](https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md),
but optimizes for in-place configuration transformation rather than out-of-place transformation. 

Validation goes hand-in-hand with customization and KRM functions can be used to automate both mutation and validation
of resources, similar to
[Kubernetes admission control](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/). 



A kpt package is a bundle of configuration _data_. It is represented as a directory tree containing KRM resources using
YAML as the file format.

A package is explicitly declared using a file named `Kptfile` containing a KRM resource of kind `Kptfile`. The Kptfile
contains metadata about the package and is just a regular resource in the YAML format.

Just as directories can be nested, a package can contain another package, called a _subpackage_.

Let's take a look at the wordpress package as an example:

```shell
kpt pkg get https://github.com/kptdev/kpt/package-examples/wordpress@v1.0.0-beta.59
```

View the package hierarchy using the `tree` command:

```shell
kpt pkg tree wordpress/
Package "wordpress"
├── [Kptfile]  Kptfile wordpress
├── [service.yaml]  Service wordpress
├── deployment
│   ├── [deployment.yaml]  Deployment wordpress
│   └── [volume.yaml]  PersistentVolumeClaim wp-pv-claim
└── Package "mysql"
    ├── [Kptfile]  Kptfile mysql
    ├── [deployment.yaml]  PersistentVolumeClaim mysql-pv-claim
    ├── [deployment.yaml]  Deployment wordpress-mysql
    └── [deployment.yaml]  Service wordpress-mysql
```

This _package hierarchy_ contains two packages:

1. `wordpress` is the top-level package in the hierarchy declared using `wordpress/Kptfile`. This package contains 2
   subdirectories. `wordpress/deployment` is a regular directory used for organizing resources that belong to the
   `wordpress` package itself. The `wordpress` package contains 3 direct resources in 3 files: `service.yaml`,
   `deployment/deployment.yaml`, and `deployment/volume.yaml`.
2. `wordpress/mysql` is a subpackage of `wordpress` package since it contains a `Kptfile`. This package contains 3
   resources in `wordpress/mysql/deployment.yaml` file.

kpt uses Git as the underlying version control system. A typical workflow starts by fetching an _upstream_ package from
a Git repository to the local filesystem using `kpt pkg` commands. All other functionality
(i.e. `kpt fn` and `kpt live`) use the package from the local filesystem, not the remote Git repository. You may think
of this as the _vendoring_ used by tooling for some programming languages. The main difference is that kpt is designed
to enable you to modify the vendored package on the local filesystem and then later update the package by merging the
local and upstream changes.

There is one scenario where a Kptfile is implicit: You can use kpt to fetch any Git directory containing KRM resources,
even if it does not contain a `Kptfile`. Effectively, you are telling kpt to treat that Git directory as a package. kpt
automatically creates the `Kptfile` on the local filesystem to keep track of the upstream repo. This means that kpt is
compatible with large corpus of existing Kubernetes configuration stored on Git today!

For example, `spark` is just a vanilla directory of KRM:

```shell
kpt pkg get https://github.com/kubernetes/examples/tree/master/_archived/spark
```

We will go into details of how to work with packages in [Chapter 3](../03-packages).

## Workflows

In this section, we'll describe the typical workflows in kpt. We say "typical", because there is no single right way of
using kpt. A user may choose to use some command but not another. This modularity is a key design principle. However, we
still want to provide guidance on how the functionality could be used in real-world scenarios.

A workflow in kpt can be best modelled as performing some verbs on the noun _package_. For example, when consuming an
upstream package, the initial workflow can look like this:

![img](/images/lifecycle/flow1.svg)

- **Get**: Using `kpt pkg get`
- **Explore**: Using an editor or running commands such as `kpt pkg tree`
- **Edit**: Customize the package either manually or automatically using `kpt fn eval`. This may involve editing the
  functions pipeline in the `Kptfile` which is executed in the next stage.
- **Render**: Using `kpt fn render`

First, you get a package from upstream. Then, you explore the content of the package to understand it better. Then you
typically want to customize the package for you specific needs. Finally, you render the package which produces the final
resources that can be directly applied to the cluster. Render is a required step as it ensures certain preconditions and
postconditions hold true about the state of the package.

This workflow is an iterative process. There is usually a tight Edit/Render loop in order to produce the desired
outcome.

Some time later, you may want to update to a newer version of the upstream package:

![img](/images/lifecycle/flow2.svg)

- **Update**: Using `kpt pkg update`

Updating the package involves merging your local changes with the changes made by the upstream package authors between
the two specified versions. This is a resource-based merge strategy, and not a line-based merge strategy used by
`git merge`.

Instead of consuming an existing package, you can also create a package from scratch:

![img](/images/lifecycle/flow5.svg)

- **Create**: Initialize a directory using `kpt pkg init`.

Now, let's say you have rendered the package, and want to deploy it to a cluster. The workflow
may look like this:

![img](/images/lifecycle/flow3.svg)

- **Initialize**: One-time process using `kpt live init`
- **Preview**: Using `kpt live apply --dry-run`
- **Apply**: Using `kpt live apply`
- **Observe**: Using `kpt live status`

First, you use dry-run to validate the resources in your package and verify that the expected
resources will be applied and pruned. Then if that looks good, you apply the package. Afterwards,
you may observe the status of the package on the cluster.

You typically want to store the package on Git:

![img](/images/lifecycle/flow4.svg)

- **Publish**: Using `git commit`

The publishing flow is orthogonal to deployment flow. This allows you to act as a publisher of an
upstream package even though you may not deploy the package personally.

## Functions

A KRM function (formerly called a _kpt_ function_) is a containerized program that
can perform CRUD operations on KRM resources stored on the local filesystem. kpt
functions are the extensible mechanism to automate mutation and validation of
KRM resources. Some example use cases:

- Enforce all `Namespace` resources to have a `cost-center` label.
- Add a label to resources based on some filtering criteria
- Use a `Team` custom resource to generate a `Namespace` and associated
  organization-mandated defaults (e.g. `RBAC`, `ResourceQuota`, etc.) when
  bootstrapping a new team
- Bulk transformation of all `PodSecurityPolicy` resources to improve the
  security posture.
- Inject a sidecar container (service mesh, mysql proxy, logging) in a workload
  resource (e.g. `Deployment`)

Since functions are containerized, they can encapsulate different toolchains,
languages, and runtimes. For example, the function container image can
encapsulate:

- A binary built using kpt's official Go SDK
- Wrap an existing KRM tool such as `kubeconform`
- Invoke a bash script performing low-level operations
- The interpreter for "executable configuration" such as `Starlark` or `Rego`

To astute readers, this model will sound familiar: functions are the client-side
analog to Kubernetes controllers:

|                  | Client-side              | Server-side       |
| ---------------- | ------------------------ | ----------------- |
| **Orchestrator** | kpt                      | Kubernetes        |
| **Data**         | YAML files on filesystem | resources on etcd |
| **Programs**     | functions                | controllers       |

Just as Kubernetes system orchestrates server-side containers, kpt CLI
orchestrates client-side containers operating on configuration. By standardizing
the input and output of the function containers, and how the containers are
executed, kpt can provide the following guarantees:

- Functions are interoperable
- Functions can be chained together
- Functions are hermetic. For correctness, security and speed, it's desirable to
  be able to run functions hermetically without any privileges; preventing
  out-of-band access to the host filesystem and networking.

We will discuss the KRM Functions Specification Standard in detail in
[Chapter 5](../05-developing-functions).
At a high level, a function execution looks like this:

![img](/images/func.svg)

where:

- `input items`: The input list of KRM resources to operate on.
- `output items`: The output list obtained from adding, removing, or modifying
  items in the input.
- `functionConfig`: An optional meta resource containing the arguments to this
  invocation of the function.
- `results`: An optional meta resource emitted by the function for observability
  and debugging purposes.

Naturally, functions can be chained together in a pipeline:

![img](/images/pipeline.svg)

There are two different CLI commands that execute functions corresponding to two
fundamentally different approaches:

- `kpt fn render`: Executes the pipeline of functions declared in the package
  and its subpackages. This is a declarative way to run functions.
- `kpt fn eval`: Executes a given function on the package. The image to run and
  the `functionConfig` is specified as CLI argument. This is an imperative way
  to run functions. Since the function is provided explicitly by the user, an
  imperative invocation can be more privileged and low-level than an declarative
  invocation. For example, it can have access to the host system.

We will discuss how to run functions in [Chapter 4](../04-using-functions) and how to develop functions
in [Chapter 5](../05-developing-functions).


