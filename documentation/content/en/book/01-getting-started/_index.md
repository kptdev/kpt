---
title: "Chapter 1: Getting started"
linkTitle: "Chapter 1: Getting started"
description: This chapter is a quick introduction to kpt using an example to demonstrate important concepts and
             features. The following chapters will cover these concepts in detail.
toc: true
menu:
  main:
    parent: "Book"
    weight: 10
---

## System Requirements

### kpt

Install the [kpt CLI](installation/kpt-cli):

```shell
kpt version
```

### Git

kpt requires that you have [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git) installed and
configured.

### Container Runtime

`kpt` requires you to have at least one of the following runtimes installed and configured.

#### Docker

Please follow the [instructions](https://docs.docker.com/get-docker) to install and configure Docker.

#### Podman

Please follow the [instructions](https://podman.io/getting-started/installation) to install and configure Podman.

If you want to set up rootless container runtime, [this](https://rootlesscontaine.rs/) may be a useful resource for you.

Environment variables can be used to control which container runtime to use. More details can be found in the reference
documents for [`kpt fn render`](../../reference/cli/fn/render/) and [`kpt fn eval`](../../reference/cli/fn/eval/).

### Kubernetes cluster

In order to deploy the examples, you need a Kubernetes cluster and a configured kubeconfig context.

For testing purposes, [kind](https://kind.sigs.k8s.io/docs/user/quick-start/) tool is useful for running ephemeral
Kubernetes cluster on your local host.

## Quickstart

In this example, you are going to configure and deploy Nginx to a Kubernetes cluster.

### Fetch the package

kpt is fully integrated with Git and enables forking, rebasing and versioning a package of configuration using the
underlying Git version control system.

First, let's fetch the _kpt package_ from Git to your local filesystem:

```shell
kpt pkg get https://github.com/kptdev/kpt/package-examples/nginx@v1.0.0-beta.61
```

Subsequent commands are run from the `nginx` directory:

```shell
cd nginx
```

`kpt pkg` commands provide the functionality for working with packages on Git and on your local filesystem.

Next, let's quickly view the content of the package:

```shell
kpt pkg tree
Package "nginx"
├── [Kptfile]  Kptfile nginx
├── [deployment.yaml]  Deployment my-nginx
└── [svc.yaml]  Service my-nginx-svc
```

As you can see, this package contains 3 resources in 3 files. There is a special file named `Kptfile` which is used by
the kpt tool itself and is not deployed to the cluster. Later chapters will explain the `Kptfile` in detail.

Initialize a local Git repo and commit the forked copy of the package:

```shell
git init; git add .; git commit -m "Pristine nginx package"
```

### Customize the package

At this point, you typically want to customize the package. With kpt, you can use different approaches depending on your
use case.

#### Manual Editing

You may want to manually edit the files. For example, modify the value of `spec.replicas` in `deployment.yaml` using
your favorite editor:

```shell
vim deployment.yaml
```

#### Automating One-time Edits with Functions

The [`kpt fn`](../../reference/cli/fn/) set of commands enable you to execute programs called _kpt functions_. These
programs are packaged as containers and take in YAML files, mutate or validate them, and then output YAML.

For instance, you can use a function (`ghcr.io/kptdev/krm-functions-catalog/search-replace:latest`) to search and replace all the occurrences of
the `app` key in the `spec` section of the YAML document (`spec.**.app`) and set the value to `my-nginx`. 

You can use the `kpt fn eval` command to run this mutation on your local files a single time:

```shell
kpt fn eval --image ghcr.io/kptdev/krm-functions-catalog/search-replace:latest -- by-path='spec.**.app' put-value=my-nginx
```

To see what changes were made to the local package:

```shell
git diff
```

#### Declaratively Defining Edits

For operations that need to be performed repeatedly, there is a _declarative_ way to define a pipeline of functions as
part of the package (in the `Kptfile`). In this `nginx` package, the author has already declared a function (`kubeconform`)
that validates the resources using their OpenAPI schema.

```yaml
pipeline:
  validators:
    - image: ghcr.io/kptdev/krm-functions-catalog/kubeconform:latest
```

You might want to label all resources in the package. To achieve that, you can declare `set-labels` function in the
`pipeline` section of `Kptfile`. Add this by running the following command:

```shell
cat >> Kptfile <<EOF
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:latest
      configMap:
        env: dev
EOF
```

This function will ensure that the label `env: dev` is added to all the resources in the package.

The pipeline is executed using the `render` command:

```shell
kpt fn render
```

Regardless of how you choose to customize the package — whether by manually editing it or running one-time functions
using `kpt fn eval` — you need to _render_ the package before applying it the cluster. This ensures all the functions
declared in the package are executed, and the package is ready to be applied to the cluster.

### Apply the Package

`kpt live` commands provide the functionality for deploying packages to a Kubernetes cluster.

First, initialize the kpt package:

```shell
kpt live init
```

This adds metadata to the `Kptfile` required to keep track of changes made
to the state of the cluster. This 
allows kpt to group resources so that they can be applied, updated, pruned, and
deleted together.

Apply the resources to the cluster:

```shell
kpt live apply --reconcile-timeout=15m
```

This waits for the resources to be reconciled on the cluster by monitoring their
status.

### Update the package

At some point, there will be a new version of the upstream `nginx` package, and
you want to merge the upstream changes with changes to your local package.

First, commit your local changes:

```shell
git add .; git commit -m "My customizations"
```

Then update to version `latest`:

```shell
kpt pkg update @latest
```

This merges the upstream changes with your local changes using a schema-aware
merge strategy.

Apply the updated resources to the cluster:

```shell
kpt live apply --reconcile-timeout=15m
```

### Clean up

Delete the package from the cluster:

```shell
kpt live destroy
```

Congrats! You should now have a rough idea of what kpt is and what you can do
with it. Now, let's delve into the details.
