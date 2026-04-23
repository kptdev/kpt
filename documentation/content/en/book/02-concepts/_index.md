---
title: "Chapter 2: Concepts"
linkTitle: "Chapter 2: Concepts"
description: This chapter describes what kpt is and what the main concepts are behind kpt.
toc: true
menu:
  main:
    parent: "Book"
    weight: 20
---

## What is kpt?

kpt stands for Kubernetes Package Transformation. It supports the management of configuration as data (CaD).

*Configuration as data* is an approach to the management of configurations, including the configuration of infrastructure, policy, services, applications, and so on, which comprises the following actions:

* Making configuration data the source of truth, stored separately from the live state.
* Using a uniform, serializable data model to represent the configuration.
* Separating code that acts on the configuration from the data and from the packages/bundles of the data.
* Abstracting the configuration file structure and storage from the operations that act upon the configuration data. Clients manipulating the configuration data do not need to directly interact with the storage (Git, container images).

This enables the machine manipulation of the configuration for Kubernetes and any infrastructure represented in the 
[Kubernetes Resource Model (KRM)](https://github.com/kubernetes/design-proposals-archive/blob/main/architecture/resource-management.md).

![img](/images/cad-overview.svg)

### Key principles of configuration as data

There are a number of key principles to be borne in mind, with regard to configuration as data (CaD). These principles are as follows:

* Storage of secrets separately, in a secret-focused storage system.
* Storage of a versioned history of the configuration changes by change sets to bundles of related configuration data.
* Reliance on uniformity and consistency of the configuration format, including type metadata, to enable pattern-based operations on the configuration data, along the lines of duck typing.
* Separation of schemas for the configuration data from the data, and reliance on schema information for strongly typed operations and to disambiguate data structures and other variations within the model.
* Decoupling of abstractions of configurations from collections of configuration data.
* Representation of abstractions of configuration generators as data with schemas, like other configuration data.
* Finding, filtering/querying/selecting, and/or validating configuration data that can be operated on by the given code (functions).
* Finding and/or filtering/querying/selecting code (functions) that can operate on the resource types contained within a body of configuration data.
* Actuation (reconciliation of the configuration data with the live state) that is separate from the transformation of the configuration data, and is driven by the declarative data model.
* Transformations, particularly value propagation, are preferable to wholesale configuration generation, except when the expansion is dramatic (for example, >10x).
* Transformation input generation should usually be decoupled from propagation.
* Deployment context inputs should be taken from well-defined “provider context” objects.
* Identifiers and references should be declarative.
* The live state should be linked back to the sources of truth (configuration).

### Components of the kpt toolchain

The kpt toolchain includes the following components:

- [**kpt CLI**](../../reference/cli/): The kpt CLI supports package and function operations, as well as deployment, either through direct apply or through GitOps. By keeping an inventory of deployed resources, kpt enables resource pruning, aggregated status and observability, and an improved preview experience.

- [**Function SDK**](https://github.com/kptdev/krm-functions-sdk): Any general-purpose or domain-specific language can be used to create functions to transform and/or validate the YAML KRM input/output format. However, we provide software development kits (SDKs) to simplify the function authoring process, in
  [Go](../05-developing-functions/#developing-in-go).

- [**Function catalog**](https://catalog.kpt.dev/function-catalog): This is a catalog of off-the-shelf, tested functions. kpt makes configurations easy to create and transform, via reusable functions. Because the functions are expected to be used for in-place transformation, they need to be idempotent.

## Packages

kpt manages the KRM resources in bundles called **packages**.

Off-the-shelf packages are rarely deployed without any customization. Like [kustomize](https://kustomize.io), kpt applies transformation **functions**, using the same
[KRM function specification](https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md).
However, kpt optimizes for in-place configuration transformation rather than out-of-place transformation. 

Validation goes hand-in-hand with customization. KRM functions can be used to automate both mutation and validation of resources, similarly to 
[Kubernetes admission control](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/). 

A kpt package is a bundle of configuration _data_. It is represented as a directory tree containing the KRM resources using YAML as the file format.

A package is explicitly declared using a file named `Kptfile`. This file contains a KRM resource of type `Kptfile`. The Kptfile contains metadata about the package and is simply a regular resource in the YAML format.

Just as directories can be nested, a package can contain another package. This is called a _subpackage_.

### Kptfile annotations

The Kptfile supports annotations that control package-level behavior:

- **`kpt.dev/bfs-rendering`**: When set to `"true"`, this annotation renders the package hierarchy in breadth-first order, instead of the default depth-first
  post-order.
- **`kpt.dev/save-on-render-failure`**: When set to `"true"`, this annotation saves partially rendered resources to disk, even when rendering fails, instead of reverting all changes. This is particularly useful for debugging render failures and is essential for programmatic package rendering scenarios, where preserving partial progress is valuable.

### Status Conditions

The Kptfile includes a `status.conditions` field that provides a declarative way to track the execution status of kpt operations. This makes package management operations observable and traceable.

When the `kpt fn render` command is executed, a `Rendered` status condition is automatically added to the root Kptfile to indicate whether the rendering operation succeeded or failed. This status is recorded only for in-place renders (the default behavior). It is not written for out-of-place modes, such as stdout (`-o stdout`), unwrap (`-o unwrap`), or directory output (`-o <dir>`).

**On a successful render:**
```yaml
status:
  conditions:
    - type: Rendered
      status: "True"
      reason: RenderSuccess
```

**On a failed render:**
```yaml
status:
  conditions:
    - type: Rendered
      status: "False"
      reason: RenderFailed
      message: |-
        pkg.render: pkg .:
        	pipeline.run: must run with `--allow-exec` option to allow running function binaries
```

The status condition is recorded only in the root Kptfile, not in subpackages. The error message, in failure cases, provides details about what went wrong during the render operation.

Let us have a look at the wordpress package as an example:

```shell
kpt pkg get https://github.com/kptdev/kpt/package-examples/wordpress@v1.0.0-beta.59
```

You can view the package hierarchy, using the `tree` command:

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

1. `wordpress`: This is the top-level package in the hierarchy. It is declared using `wordpress/Kptfile`. This package contains two subdirectories.
`wordpress/deployment` is a regular directory used for organizing resources that belong to the `wordpress` package itself. The `wordpress` package contains
three direct resources in three files:
  - `service.yaml`
  - `deployment/deployment.yaml`
  - `deployment/volume.yaml`
2. `wordpress/mysql`: This is a subpackage of the `wordpress` package, since it contains a `Kptfile`. This package contains three resources in the 
`wordpress/mysql/deployment.yaml` file.

kpt uses Git as the underlying version control system. A typical workflow starts by fetching an _upstream_ package from a Git repository to the local filesystem using `kpt pkg` commands. All the other functionalities (namely,
`kpt fn` and `kpt live`) use the package from the local filesystem, rather than the remote Git repository. It can be thought of as the _vendoring_ used by tooling for some programming languages. The main difference is that kpt is designed to enable you to modify the vendored package on the local filesystem, and then update the package by merging the local and upstream changes.

There is one scenario where a Kptfile is implicit: you can use kpt to fetch any Git directory containing KRM resources, even if the directory does not contain a `Kptfile`. Effectively, you are telling kpt to treat the Git directory as a package. kpt automatically creates the `Kptfile` on the local filesystem to keep track of the upstream repository. This means that kpt is compatible with a
large corpus of existing Kubernetes configurations currently stored on Git.

For example, `spark` is essentially a vanilla directory of KRM:

```shell
kpt pkg get https://github.com/kubernetes/examples/tree/master/_archived/spark
```

Details of how to work with packages are set out in [Chapter 3](../03-packages).

## Workflows

In this section, we will describe the typical workflows in kpt. The word _typical_ is used here because there is no single correct way of using kpt. A
user may choose to use a specific command but not another. This modularity is a key design principle. However, we would still like to provide guidance on how the functionality can be used in real-world scenarios.

A workflow in kpt can be best modeled as performing some verbs on the noun _package_. For example, when consuming an upstream package, the initial workflow may look like this:

![img](/images/lifecycle/flow1.svg)

- **Get**: Use the `kpt pkg get` command.
- **Explore**: Use an editor or run commands, such as `kpt pkg tree`.
- **Edit**: Customize the package manually or automatically using the `kpt fn eval` command. This may involve editing the functions pipeline in the `Kptfile` which is executed in the next stage.
- **Render**: Use the `kpt fn render` command.

1. Get a package from upstream.
2. Explore the content of the package to understand it better.
3. Customize the package to suit your needs.
4. Render the package. This produces the final resources that can be directly applied to the cluster.
Render is a required step, as it ensures that certain preconditions and
postconditions about the state of the package hold true.

This workflow is an iterative process. There is usually a tight Edit/Render loop, in order to produce the desired outcome.

You may later wish to update to a newer version of the upstream package:

![img](/images/lifecycle/flow2.svg)

- **Update**: Use the `kpt pkg update` command.

Updating the package involves merging your local changes with the changes made by the upstream package authors between the two specified versions. This is a resource-based merge strategy, and not a line-based merge strategy used by
`git merge`.

Instead of consuming an existing package, you can also create a package from scratch:

![img](/images/lifecycle/flow5.svg)

- **Create**: Initialize a package using the `kpt pkg init` command. This command creates the directory if it does not exist.

Let us suppose that you have rendered the package, and would like to deploy it to a cluster. The workflow may look like this:

![img](/images/lifecycle/flow3.svg)

- **Initialize**: This is a one-time process using the `kpt live init` command.
- **Preview**: Use the `kpt live apply --dry-run` command.
- **Apply**: Use the `kpt live apply` command.
- **Observe**: Use the `kpt live status` command.

First, use the `kpt live apply --dry-run` command to validate the resources in your package and verify that the expected resources will be applied and pruned. If the preview looks good, then apply the package, using the `kpt live apply` command. Afterwards, you may observe the status of the package on the cluster.

Typically, it is best to store the package in Git:

![img](/images/lifecycle/flow4.svg)

- **Publish**: Use the `git commit` command.

The publishing flow is orthogonal to the deployment flow. This allows you to act as a publisher of an upstream package, even though you may not deploy the package personally.

## Functions

A Kubernetes Resource Model (KRM) function (formerly called a _kpt function_) is a containerized program that can perform create, read, update, and delete (CRUD) operations on KRM resources stored on the local filesystem. KRM functions are the extensible mechanism to automate the mutation and validation of KRM resources. The following are some example use cases:

- Enforce all `Namespace` resources to have a `cost-center` label.
- Add a label to resources based on certain filtering criteria.
- Use a `Team` custom resource to generate a `Namespace` and associated
  organization-mandated defaults (for example, `RBAC`, `ResourceQuota`, and so on) when bootstrapping a new team.
- Bulk transformation of all `PodSecurityPolicy` resources to improve the
  security posture.
- Inject a sidecar container (service mesh, mysql proxy, logging) in a workload
  resource (for example, `Deployment`).

Since the functions are containerized, they can encapsulate different toolchains, languages, and runtimes. For example, the function container image can encapsulate the following:

- A binary built using kpt's official Go software development kit (SDK).
- Wrap an existing KRM tool, such as `kubeconform`.
- Invoke a bash script performing low-level operations.
- The interpreter for "executable configuration", such as `Starlark` or `Rego`.

To astute readers, this model will sound familiar: the functions are the client-side analog to the Kubernetes controllers:

|                  | Client-side              | Server-side       |
| ---------------- | ------------------------ | ----------------- |
| **Orchestrator** | kpt                      | Kubernetes        |
| **Data**         | YAML files on filesystem | resources on etcd |
| **Programs**     | functions                | controllers       |

Just as the Kubernetes system orchestrates the server-side containers, the kpt CLI orchestrates the client-side containers operating on the configuration. By standardizing the input and output of the function containers, and how the containers are executed, kpt can provide the following guarantees:

- The functions are interoperable.
- The functions can be chained together.
- The functions are hermetic. For correctness, security and speed, it is desirable to be able to run functions hermetically without any privileges, thereby preventing out-of-band access to the host filesystem and networking.

We will discuss the KRM Functions Specification Standard in detail in
[Chapter 5](../05-developing-functions). At a high level, a function execution looks like this:

![img](/images/func.svg)

Where:

- `input items`: This is the input list of the KRM resources on which to operate.
- `output items`: This is the output list obtained from adding, removing, or modifying items in the input.
- `functionConfig`: This is an optional meta resource containing the arguments to this invocation of the function.
- `results`: This is an optional meta resource emitted by the function for observability and debugging purposes.

Functions can be chained together in a pipeline, as illustrated here:

![img](/images/pipeline.svg)

There are two different CLI commands that execute the functions corresponding to two fundamentally different approaches:

- `kpt fn render`: This command executes the pipeline of functions declared in the package and its subpackages. This is a declarative way to run the functions.
- `kpt fn eval`: This command executes a given function on the package. The image to run and the `functionConfig` are specified as a CLI argument. This is an imperative way to run functions. Since the function is provided explicitly by the user, an imperative invocation can be more privileged and low-level than a declarative invocation. For example, it can have access to the host system.

We will discuss how to run functions in [Chapter 4](../04-using-functions), and how to develop functions in [Chapter 5](../05-developing-functions).


