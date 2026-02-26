---
title: "Chapter 1: Getting started"
linkTitle: "Chapter 1: Getting started"
description: This chapter provides a quick introduction to kpt, using examples to demonstrate the important concepts and features. The following chapters cover these concepts in detail.
toc: true
menu:
  main:
    parent: "Book"
    weight: 10
---

## System requirements

### kpt

Install the [kpt CLI](installation/kpt-cli), using the following command:

```shell
kpt version
```

### Git

`kpt` requires that you have [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git) installed and
configured.

### Container runtime

`kpt` requires that you have at least one of the following runtimes installed and configured.

#### Docker

Follow the [instructions](https://docs.docker.com/get-docker) to install and configure Docker.

#### Podman

Follow the [instructions](https://podman.io/getting-started/installation) to install and configure Podman.

If you want to set up a rootless container runtime, then [this](https://rootlesscontaine.rs/) may be a useful resource for you.

Environment variables can be used to control which container runtime to use. More details can be found in the reference
documents for [`kpt fn render`](../../reference/cli/fn/render/) and [`kpt fn eval`](../../reference/cli/fn/eval/).

### Kubernetes cluster

To deploy the examples, you need a Kubernetes cluster and a configured kubeconfig context.

For testing purposes, the [kind](https://kind.sigs.k8s.io/docs/user/quick-start/) tool is useful for running an ephemeral Kubernetes
cluster on your local host.

## Quickstart

In this example, you are going to configure and deploy Nginx to a Kubernetes cluster.

### Fetching the package

`kpt` is fully integrated with Git and enables the forking, rebasing, and versioning of a configuration package using the
underlying Git version control system.

First, using the following command, fetch the kpt package from Git to your local filesystem:

```shell
kpt pkg get https://github.com/kptdev/kpt/package-examples/nginx@v1.0.0-beta.59
```

Subsequent commands are run from the `nginx` directory:

```shell
cd nginx
```

The `kpt pkg` commands provide the functionality for working with packages on Git and on your local filesystem.

Next, use the following command to view the content of the package:

```shell
kpt pkg tree
Package "nginx"
├── [Kptfile]  Kptfile nginx
├── [deployment.yaml]  Deployment my-nginx
└── [svc.yaml]  Service my-nginx-svc
```

As can be seen, this package contains three resources in three files. There is a special file named `Kptfile`. This file is used by
the kpt tool itself and is not deployed to the cluster. Later chapters will explain the `Kptfile` in detail.

Initialize a local Git repo and commit the forked copy of the package, using the following commands:

```shell
git init; git add .; git commit -m "Pristine nginx package"
```

### Customizing the package

At this point, it is a good idea to customize the package. With kpt, you can use different approaches, depending on your use case.

#### Manual editing

You may want to edit the files manually. For example, modify the value of `spec.replicas` in the `deployment.yaml` using your favorite
editor:

```shell
vim deployment.yaml
```

#### Automating one-time edits with functions

The [`kpt fn`](../../reference/cli/fn/) set of commands enables you to execute programs called _kpt functions_. These programs are
packaged as containers and take in YAML files, mutate or validate them, and then output YAML.

For example, you can use a function (`ghcr.io/kptdev/krm-functions-catalog/search-replace:latest`) to search for and replace all the occurrences of the `app` key, in the `spec` section of the YAML document (`spec.**.app`), and set the value to `my-nginx`. 

You can use the `kpt fn eval` command to run this mutation on your local files a single time:

```shell
kpt fn eval --image ghcr.io/kptdev/krm-functions-catalog/search-replace:latest -- by-path='spec.**.app' put-value=my-nginx
```

To see what changes were made to the local package, use the following command:

```shell
git diff
```

#### Declaratively defining edits

For operations that need to be performed repeatedly, there is a _declarative_ way to define a pipeline of functions as part of the
package (in the `Kptfile`). In this `nginx` package, the author has already declared a function (`kubeconform`) that validates the
resources using their OpenAPI schema.

```yaml
pipeline:
  validators:
    - image: ghcr.io/kptdev/krm-functions-catalog/kubeconform:latest
```

It might be a good idea to label all the resources in the package. To achieve this, you can declare the `set-labels` function, in the
`pipeline` section of the `Kptfile`. Add this by running the following command:

```shell
cat >> Kptfile <<EOF
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:latest
      configMap:
        env: dev
EOF
```

This function ensures that the `env: dev` label is added to all the resources in the package.

The pipeline is executed using the `render` command, as follows:

```shell
kpt fn render
```

Regardless of how you choose to customize the package — whether by manually editing it or running one-time functions using `kpt fn eval`
 — you need to _render_ the package before applying it to the cluster. This ensures that all the functions declared in the package
have been executed, and the package is ready to be applied to the cluster.

### Applying the package

The `kpt live` commands provide the functionality for deploying the packages to a Kubernetes cluster.

First, initialize the kpt package, using the following command:

```shell
kpt live init
```

This adds to the `Kptfile` the metadata required to keep track of the changes made to the state of the cluster. This allows kpt to
group the resources, so that they can be applied, updated, pruned, and deleted together.

Apply the resources to the cluster:

```shell
kpt live apply --reconcile-timeout=15m
```

This waits for the resources to be reconciled on the cluster by monitoring their status.

### Updating the package

At some point, there will be a new version of the upstream `nginx` package, and you will need to merge the upstream changes with the
changes to your local package.

First, commit your local changes, using the following command:

```shell
git add .; git commit -m "My customizations"
```

Update to version `latest`:

```shell
kpt pkg update @latest
```

This merges the upstream changes with your local changes, using a schema-aware merge strategy.

Apply the updated resources to the cluster, as follows:

```shell
kpt live apply --reconcile-timeout=15m
```

### Cleaning up

Delete the package from the cluster, using the following command:

```shell
kpt live destroy
```

You should now have a rough idea of what kpt is and what you can do
with it. Let us now delve into the details.
