---
title: "Packages"
weight: 1
description: >
  kpt packages
---

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

### Kptfile Annotations

The Kptfile supports annotations that control package-level behaviour:

- **`kpt.dev/bfs-rendering`**: When set to `"true"`, renders the package hierarchy in breadth-first order instead of
the default depth-first post-order.
- **`kpt.dev/save-on-render-failure`**: When set to `"true"`, saves partially rendered resources to disk even when
rendering fails, instead of reverting all changes. This is particularly useful for debugging render failures and is
essential for programmatic package rendering scenarios where preserving partial progress is valuable.

### Status Conditions
The Kptfile includes a `status.conditions` field that provides a declarative way to track the execution status of kpt
operations. This makes package management operations observable and traceable.

When `kpt fn render` executes, a `Rendered` status condition is automatically added to the root Kptfile to indicate 
whether the rendering operation succeeded or failed.
This status is recorded only for in-place renders (the default behavior).
It is not written for out-of-place modes such as stdout (`-o stdout`), unwrap (`-o unwrap`), or 
directory output (`-o <dir>`).

**On successful render:**
```yaml
status:
  conditions:
    - type: Rendered
      status: "True"
      reason: RenderSuccess
```

**On failed render:**
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

The status condition is recorded only in the root Kptfile, not in subpackages. The error message in failure cases 
provides details about what went wrong during the render operation.

Just as directories can be nested, a package can contain another package, called a _subpackage_.

Let's take a look at the wordpress package as an example:

```shell
kpt pkg get https://github.com/kptdev/kpt/package-examples/wordpress@v1.0.0-beta.59
```

View the package hierarchy using the `tree` command:

```shell
kpt pkg tree wordpress/
Package "wordpress" (independent)
├── [Kptfile]  Kptfile wordpress
├── [service.yaml]  Service wordpress
├── deployment
│   ├── [deployment.yaml]  Deployment wordpress
│   └── [volume.yaml]  PersistentVolumeClaim wp-pv-claim
└── Package "mysql" (dependent)
    ├── [Kptfile]  Kptfile mysql
    ├── [deployment.yaml]  PersistentVolumeClaim mysql-pv-claim
    ├── [deployment.yaml]  Deployment wordpress-mysql
    └── [service.yaml]  Service wordpress-mysql
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

We will go into details of how to work with packages in [Chapter 3]({{% relref "/book/03-packages" %}}).

### Local Configuration

A package can contain resources that are used only during rendering and should not
be applied to the cluster. These are called local configuration resources.
Common examples include function configs (referenced via `configPath`) and helper
resources used as input to pipeline functions.

You mark a resource as local by adding the `config.kubernetes.io/local-config`
annotation:

```yaml
metadata:
  annotations:
    config.kubernetes.io/local-config: "true"
```

When you deploy the package with `kpt live apply`, local configuration resources
are automatically filtered out. They exist in the package solely to support the
pipeline. See [Chapter 4]({{% relref "/book/04-using-functions" %}}) for how function configs use
this annotation, and the
[`local-config` annotation reference]({{% relref "/documentation/6-references/annotations/local-config" %}}) for
full details.
