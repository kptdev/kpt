---
title: "Chapter 8: Package Orchestration"
linkTitle: "Chapter 8: Package Orchestration"
description: |
    In this chapter, we are going to cover _package orchestration_ - management of package lifecycle supported by the
    kpt toolchain and Porch - Package Orchestration service.
toc: true
menu:
  main:
    parent: "Book"
    weight: 80
---

{{< warning >}}
This chapter is no longer valid and will be moved to the [Porch](https://docs.nephio.org/docs/porch/) documentation.
The `kpt alpha repo|rpkg` command set are now part of the [porchctl](https://docs.nephio.org/docs/porch/user-guides/porchctl-cli-guide/) tool.
{{< /warning >}}

## Introduction

Package Orchestration encompasses management of the overall lifecycle of
configuration [**packages**](../02-concepts/#packages), including:

* management of package repositories
* discovery of configuration packages and kpt [**functions**](../02-concepts/#functions)
* creating, modifying, updating, and deleting packages
* versioning packages
* WYSIWYG package authoring
* package customization with _guardrails_
* evaluation of functions on package contents
* approval flow to publish a change to a configuration package
* deployment and rollouts

Package Orchestration enables [workflows](../02-concepts/#workflows)
similar to those supported by the kpt CLI, but makes them available as a
service. This enables creation of WYSIWYG user experiences, similar to the
proof-of-concept [Backstage plugin](../../guides/namespace-provisioning-ui).

Dedicated user guides are available for the use of Porch with the
[Backstage UI](../../guides/namespace-provisioning-ui) and
the [kpt cli](../../guides/porch-user-guide). In the following sections
of the book you will follow the basic journey of discovering and authoring
configuration packages using Porch - the Package Orchestration service.

## System Requirements

In order to follow the package orchestration examples, you will need:

* the Package Orchestration service (Porch)
  [installed](https://docs.nephio.org/docs/porch/user-guides/install-and-using-porch/) in your Kubernetes cluster and
  [kubectl context](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/) configured
  with your cluster.
* kpt CLI [installed](../../installation/) on your system

## Quickstart

​​In this quickstart you will use Porch to discover configuration packages
in a [sample repository](https://github.com/kptdev/kpt-samples).

You will use the kpt CLI - the new `kpt alpha` command sub-groups to interact
with the Package Orchestration service.

### Register the repository

Start by registering the sample repository with Porch. The repository already
contains a [`basens`](https://github.com/kptdev/kpt-samples/tree/main/basens) package.

```sh
# Register a sample Git repository:
$ kpt alpha repo register --namespace default \
  https://github.com/kptdev/kpt-samples.git
```

 > Refer to the [register command reference](../../reference/cli/alpha/repo/reg/) for usage.

The sample repository is public and Porch therefore doesn't require
authentication to read the repository and discover packages within it.

You can confirm the repository is now registered with Porch by using the
`kpt alpha repo get` command. Similar to `kubectl get`, the command will list
all repositories registered with Porch, or get information about specific ones
if list of names is provided

```sh
# Query repositories registered with Porch:
$ kpt alpha repo get
NAME         TYPE  CONTENT  DEPLOYMENT  READY  ADDRESS
kpt-samples  git   Package              True   https://github.com/kptdev/kpt-samples.git
```

 > Refer to the [get command reference](../../reference/cli/alpha/repo/get/) for usage.

From the output you can see that:

* the repository was registered by the name `kpt-samples`. This was chosen
  by kpt automatically from the repository URL, but can be overridden
* it is a `git` repository (OCI repositories are also supported, though
  currently with some limitations)
* the repository is *not* a deployment repository. Repository can be marked
  as deployment repository which indicates that packages in the repository are
  intended to be deployed into live state.
* the repository is ready - Porch successfully registered it and discovered
  packages stored in the repository.

The Package Orchestration service is designed to be part of the Kubernetes
ecosystem. The [resources](https://docs.nephio.org/docs/porch/user-guides/porchctl-cli-guide/) managed by Porch are KRM
resources.

You can use the `-oyaml` to see the YAML representation of the repository
registration resource:

 > kpt uses the same output format flags as `kubectl`. Flags with which you are
already familiar from using `kubectl get` will work with the kpt commands
that get or list Porch resources.

```sh
# View the Repository registration resource as YAML:
$ kpt alpha repo get kpt-samples --namespace default -oyaml
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: kpt-samples
  namespace: default
spec:
  content: Package
  git:
    branch: main
    directory: /
    repo: https://github.com/kptdev/kpt-samples.git
    secretRef:
      name: ""
  type: git
status:
  conditions:
  - reason: Ready
    status: "True"
    type: Ready
```

Few additional details are available in the YAML listing:

The name of the `main` branch and a directory. These specify location within
the repository where Porch will be managing packages. Porch also analyzes tags
in the repository to identify all packages (and their specific versions), all
within the directory specified. By default Porch will analyze the whole
repository.

The `secretRef` can contain a name of a Kubernetes [Secret](https://kubernetes.io/docs/concepts/configuration/secret/)
resource with authentication credentials for Porch to access the repository.

#### kubectl

Thanks to the integration with Kubernetes ecosystem, you can also use `kubectl`
directly to interact with Porch, such as listing repository resources:

```sh
# List registered repositories using kubectl
$ kubectl get repository
NAME          TYPE   CONTENT   DEPLOYMENT   READY   ADDRESS
kpt-samples   git    Package                True    https://github.com/kptdev/kpt-samples.git
```

You can use kubectl for _all_ interactions with Porch server if you prefer.
The kpt CLI integration provides a variety of convenience features.

### Discover packages

You can use the `kpt alpha rpkg get` command to list the packages discovered
by Porch across all registered repositories.

```sh
# List package revisions in registered repositories
$ kpt alpha rpkg get
NAME                                                   PACKAGE   REVISION   LATEST   LIFECYCLE   REPOSITORY
kpt-samples-da07e9611f9b99028f761c07a79e3c746d6fc43b   basens    main       false    Published   kpt-samples
kpt-samples-afcf4d1fac605a60ba1ea4b87b5b5b82e222cb69   basens    v0         true     Published   kpt-samples
```

 > Refer to the [get command reference](../../reference/cli/alpha/repo/get/) for usage.

 > The `r` prefix of the `rpkg` command group stands for `remote`. The commands
in the `kpt alpha rpkg` group interact with packages managed (remotely) by Porch
server. The commands in the `rpkg` group are similar to the `kpt pkg` commands
except that they operate on remote packages managed by Porch server rather than
on a local disk.

The output shows that Porch discovered the `basens` package, and found two
different revisions of it. The `v0` revision (associated with the
[`basens/v0`](https://github.com/kptdev/kpt-samples/tree/basens/v0) tag) and the `main` revision associated with the
[`main` branch](https://github.com/kptdev/kpt-samples/tree/main) in the repository.

The `LIFECYCLE` column indicates the lifecycle stage of the package revision.
The package revisions in the repository are *`Published`* - ready to be used.
Package revision may be also *`Draft`* (the package revision is being authored)
or *`Proposed`* (the author of the package revision proposed that it be
published). We will encounter examples of these 

Porch identifies the latest revision of the package (`LATEST` column).

#### View package resources

The `kpt alpha rpkg get` command displays package metadata. To view the
_contents_ of the package revision, use the `kpt alpha rpkg pull` command.

You can use the command to output the resources as a
[`ResourceList`][resourcelist] on standard output, or save them into a local
directory:

```sh
# View contents of the basens/v0 package revision
$ kpt alpha rpkg pull kpt-samples-afcf4d1fac605a60ba1ea4b87b5b5b82e222cb69 -ndefault

apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: basens
    annotations:
...
```

Add a name of a local directory on the command line to save the package onto
local disk for inspection or editing.

```sh
# Pull package revision resources, save to local disk into `./basens` directory
$ kpt alpha rpkg pull kpt-samples-afcf4d1fac605a60ba1ea4b87b5b5b82e222cb69 ./basens -ndefault

# Explore the package contents
$ find basens

basens
basens/README.md
basens/namespace.yaml
basens/Kptfile
...
```

 > Refer to the [pull command reference](../../reference/cli/alpha/rpkg/pull/) for usage.

### Unregister the repository

When you are done using the repository with Porch, you can unregister it:

```sh
# Unregister the repository
$ kpt alpha repo unregister kpt-samples -ndefault
```

### More resources

To continue learning about Porch, you can review:

* [Porch User Guide](/guides/porch-user-guide)
* [Provisioning Namespaces with the UI](/guides/namespace-provisioning-ui)
* [Porch Design Document](https://github.com/kptdev/kpt/blob/main/docs/design-docs/07-package-orchestration.md)

## Registering a Repository

In the following sections of this chapter you will explore package authoring
using Porch. You will need:

* A GitHub repository for your blueprints. An otherwise empty repository with an
  initial commit works best. The initial commit is required to establish the
  `main` branch.
* A GitHub [Personal Access Token](https://github.com/settings/tokens) with
  the `repo` scope for Porch to authenticate with the repository and allow it
  to create commits in the repository.

A repository is a porch representation of either a git repo or an oci registry.
Package revisions always belong to a single repository. A repository exists in
a Kubernetes namespace and all package revisions in a repo also belong to
the same namespace.

Use the `kpt alpha repo register` command to register your repository with
Porch: The command below uses the repository `deployments.git`.
Your repository name may be different; please update the command with the
correct repository name.

```sh
# Register your Git repository:

GITHUB_USERNAME=<GitHub Username>
GITHUB_TOKEN=<GitHub Personal Access Token>
REPOSITORY_ADDRESS=<Your Repository URL>

$ kpt alpha repo register \
  --namespace default \
  --name deployments \
  --deployment \
  --repo-basic-username=${GITHUB_USERNAME} \
  --repo-basic-password=${GITHUB_TOKEN} \
  ${REPOSITORY_ADDRESS}
```

And register the sample repository we used in the [quickstart](#quickstart):

```sh
# Register the sample repository:

kpt alpha repo register --namespace default \
  https://github.com/kptdev/kpt-samples.git
```

 > Refer to the [register command reference](../../reference/cli/alpha/repo/reg/) for usage.

You now have two repositories registered, and your repository is marked as
deployment repository. This indicates that published packages in the repository
are considered deployment-ready.

```sh
# Query repositories registered with Porch:
$ kpt alpha repo get
NAME         TYPE  CONTENT  DEPLOYMENT  READY  ADDRESS
deployments  git   Package  true        True   [Your repository address]
kpt-samples  git   Package              True   https://github.com/kptdev/kpt-samples.git
```

 > Refer to the [get command reference](../../reference/cli/alpha/repo/get/) for usage.

## Package Authoring

There are several ways to author a package revision, including creating
a completely new one, cloning an existing package, or creating a new revision
of an existing package. In this section we will explore the different ways to
author package revisions, and explore how to modify package contents.

### Create a new package revision

Create a new package revision in a repository managed by Porch:

```sh
# Initialize a new (empty) package revision:
$ kpt alpha rpkg init new-package --repository=deployments --revision=v1 -ndefault

deployments-c32b851b591b860efda29ba0e006725c8c1f7764 created

# List the available package revisions.
$ kpt alpha rpkg get

NAME                                                   PACKAGE       REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-c32b851b591b860efda29ba0e006725c8c1f7764   new-package   v1         false    Draft       deployments
kpt-samples-da07e9611f9b99028f761c07a79e3c746d6fc43b   basens        main       false    Published   kpt-samples
kpt-samples-afcf4d1fac605a60ba1ea4b87b5b5b82e222cb69   basens        v0         true     Published   kpt-samples
...
```

 > Refer to the [init command reference](../../reference/cli/alpha/rpkg/init/) for usage.

You can see the `new-package` is created in the `Draft` lifecycle stage. This
means that the package is being authored.

> You may notice that the name of the package revision
> `deployments-c32b851b591b860efda29ba0e006725c8c1f7764` was assigned
> automatically. Packages in a git repository may be located in subdirectories
> and to make sure Porch works well with the rest of the Kubernetes ecosystem,
> the resource names must meet Kubernetes requirements. The resource names
> assigned by Porch are stable, and computed as hash of the repository name,
> directory path within the repository, and revision.

The contents of the new package revision are the same as if it was created using
the [`kpt pkg init`](/book/03-packages/06-creating-a-package) command, except it
was created by the Package Orchestration service in your repository.

In fact, if you check your Git repository, you will see a new branch called
`drafts/new-package/v1` which Porch created for the draft authoring. You will
also see one or more commits made into the branch by Porch on your behalf.

### Clone an existing package

Another way to create a new package revision is by cloning an already existing
package. The existing package is referred to as *upstream* and the newly created
package is *downstream*.

Use `kpt alpha rpkg clone` command to create a new *downstream* package
`istions` by cloning the sample `basens/v0` package revision:

```sh
# Clone an upstream package to create a downstream package
$ kpt alpha rpkg clone \
  kpt-samples-afcf4d1fac605a60ba1ea4b87b5b5b82e222cb69 \
  istions \
  --repository=deployments -ndefault

deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b created

# Confirm the package revision was created
kpt alpha rpkg get deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b -ndefault
NAME                                                   PACKAGE   REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b   istions   v1         false    Draft       deployments
```

 > Refer to the [clone command reference](../../reference/cli/alpha/rpkg/clone/) for usage.

Cloning a package using the Package Orchestration service is an action similar to
[`kpt pkg get`](/book/03-packages/01-getting-a-package) command. Porch will
create the appropriate upstream package links in the new package's `Kptfile`.
Let's take a look:

```sh
# Examine the new package's upstream link (the output has been abbreviated):
$ kpt alpha rpkg pull deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b -ndefault

kpt alpha rpkg pull deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b -ndefault
apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: istions
  upstream:
    type: git
    git:
      repo: https://github.com/kptdev/kpt-samples.git
      directory: basens
      ref: basens/v0
  upstreamLock:
    type: git
    git:
      repo: https://github.com/kptdev/kpt-samples.git
      directory: basens
      ref: basens/v0
      commit: 026dfe8e3ef8d99993bc8f7c0c6ba639faa9a634
  info:
    description: kpt package for provisioning namespace
...
```

You can find out more about the `upstream` and `upstreamLock` sections of the
`Kptfile` in an [earlier chapter](/book/03-packages/01-getting-a-package)
of the book.

> A cloned package must be created in a repository in the same namespace as
> the source package. Cloning a package with the Package Orchestration Service
> retains a reference to the upstream package revision in the clone, and
> cross-namespace references are not allowed. Package revisions in repositories
> in other namespaces can be cloned using a reference directly to the underlying
> oci or git repository as described below.

You can also clone a package from a repository that is _not_ registered with
Porch, for example:

```sh
# Clone a package from Git repository directly (repository is not registered)
$ kpt alpha rpkg clone \
  https://github.com/GoogleCloudPlatform/blueprints.git/catalog/bucket@main my-bucket \
  --repository=deployments \
  --namespace=default

deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486 created

# Confirm the package revision was created
$ kpt alpha rpkg get deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486 -ndefault
NAME                                                   PACKAGE     REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486   my-bucket   v1         false    Draft       deployments
```

### Create a new revision of an existing package

Finally, with Porch you can create a new revision of an existing,
**`Published`** package. All the package revisions in your repository are
**`Draft`** revisions and need to be published first. We will cover the package
approval flow in more detail in the next section. For now we will quickly
propose and approve one of our draft package revisions and create a new revision
from it.

```sh
# Propose the package draft to be published
$ kpt alpha rpkg propose deployments-c32b851b591b860efda29ba0e006725c8c1f7764 -ndefault
deployments-c32b851b591b860efda29ba0e006725c8c1f7764 proposed

# Approve the proposed package revision for publishing
$ kpt alpha rpkg approve deployments-c32b851b591b860efda29ba0e006725c8c1f7764 -ndefault
deployments-c32b851b591b860efda29ba0e006725c8c1f7764 approved
```

You now have a **`Published`** package revision in the repository managed by Porch
and next you will create a new revision of it. A **`Published`** package is ready
to be used, such as deployed or copied.

```sh
# Confirm the package is published:
$ kpt alpha rpkg get deployments-c32b851b591b860efda29ba0e006725c8c1f7764 -ndefault
NAME                                                   PACKAGE       REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-c32b851b591b860efda29ba0e006725c8c1f7764   new-package   v1         true     Published   deployments
```

Copy the existing, **`Published`** package revision to create a **`Draft`** of
a new package revision that you can further customize:

```sh
# Copy the published package:
$ kpt alpha rpkg copy deployments-c32b851b591b860efda29ba0e006725c8c1f7764 \
  -ndefault --revision v2
deployments-93bb9ac8c2fb7a5759547a38f5f48b369f42d08a created

# List all revisions of the new-package that we just copied:
$ kpt alpha rpkg get --name new-package
NAME                                                   PACKAGE       REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-af86ae3c767b0602a198856af513733e4e37bf10   new-package   main       false    Published   deployments
deployments-c32b851b591b860efda29ba0e006725c8c1f7764   new-package   v1         true     Published   deployments
deployments-93bb9ac8c2fb7a5759547a38f5f48b369f42d08a   new-package   v2         false    Draft       deployments
```

 > Refer to the [copy command reference](../../reference/cli/alpha/rpkg/copy/) for usage.

Unlike `clone` of a package which establishes the upstream-downstream
relationship between the respective packages, and updates the `Kptfile`
to reflect the relationship, the `copy` command does *not* change the
upstream-downstream relationships. The copy of a package shares the same
upstream package as the package from which it was copied. Specifically,
in this case both `new-package/v1` and `new-package/v2` have identical contents,
including upstream information, and differ in revision only.

### Editing package revision resources

One of the driving motivations for the Package Orchestration service is enabling
WYSIWYG authoring of packages, including their contents, in highly usable UIs.
Porch therefore supports reading and updating package *contents*.

In addition to using a [UI](/guides/namespace-provisioning-ui) with Porch, we
can change the package contents by pulling the package from Porch onto the local
disk, make any desired changes, and then pushing the updated contents to Porch.

```sh
# Pull the package contents of istions/v1 onto the local disk:
$ kpt alpha rpkg pull deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b ./istions -ndefault
```

 > Refer to the [pull command reference][rpkg-pull] for usage.

The command downloaded the `istions/v1` package revision contents and saved
them in the `./istions` directory. Now you will make some changes.

First, note that even though Porch updated the namespace name (in
`namespace.yaml`) to `istions` when the package was cloned, the `README.md`
was not updated. Let's fix it first.

Open the `README.md` in your favorite editor and update its contents, for
example:

```
# istions

## Description
kpt package for provisioning Istio namespace
```

In the second change, add a new mutator to the `Kptfile` pipeline. Use the
[set-labels](https://catalog.kpt.dev/set-labels/v0.1/) function which will add
labels to all resources in the package. Add the following mutator to the
`Kptfile` `pipeline` section:

```yaml
  - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:v0.2.1
    configMap:
      color: orange
      fruit: apple
```

The whole `pipeline` section now looks like this:

```yaml
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:v0.4.1
    configPath: package-context.yaml
  - image: ghcr.io/kptdev/krm-functions-catalog/apply-replacements:v0.1.2
    configPath: update-rolebinding.yaml
  - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:v0.2.1
    configMap:
      color: orange
      fruit: apple
```

Save the changes and push the package contents back to the server:

```sh
# Push updated package contents to the server
$ kpt alpha rpkg push deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b ./istions -ndefault
```

 > Refer to the [push command reference](../../reference/cli/alpha/rpkg/push/) for usage.

Now, pull the contents of the package revision again, and inspect one of the
configuration files.

```sh
# Pull the updated package contents to local drive for inspection:
$ kpt alpha rpkg pull deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b ./updated-istions -ndefault

# Inspect updated-istions/namespace.yaml
$ cat updated-istions/namespace.yaml 

apiVersion: v1
kind: Namespace
metadata:
  name: istions
  labels:
    color: orange
    fruit: apple
spec: {}
```

The updated namespace now has new labels! What happened?

Whenever package is updated during the authoring process, Porch automatically
re-renders the package to make sure that all mutators and validators are
executed. So when we added the new `set-labels` mutator, as soon as we pushed
the updated package contents to Porch, Porch re-rendered the package and
the `set-labels` function applied the labels we requested (`color: orange` and
`fruit: apple`).

### Summary of package authoring

In this section we reviewed how to use Porch to author packages, including

* creating a new package ([`kpt alpha rpkg init`](../../reference/cli/alpha/rpkg/init/))
* cloning an existing package ([`kpt alpha rpkg clone`](../../reference/cli/alpha/rpkg/clone/))
* creating a new revision of an existing package
  ([`kpt alpha rpkg copy`](../../reference/cli/alpha/rpkg/copy/))
* pulling package contents for local editing
  ([`kpt alpha rpkg pull`](../../reference/cli/alpha/rpkg/pull/))
* and pushing updated package contents to Porch
  ([`kpt alpha rpkg push`](../../reference/cli/alpha/rpkg/push/))

## Package Lifecycle

When a new package revision is created, it is in a **`Draft`** lifecycle stage,
where the package can be authored, including updating its contents.

Before a package can be deployed or cloned, it must be **`Published`**.
The approval flow is the process by which the package is advanced from
**`Draft`** to **`Proposed`** and finally **`Published`** lifecycle stage.

In the [previous section](#package-authoring) we created several packages,
let's explore how to publish some of them.

```sh
# List package revisions (the output was abbreviated to only include Draft)
# packages
$ kpt alpha rpkg get
NAME                                                   PACKAGE       REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b   istions       v1         false    Draft       deployments
deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486   my-bucket     v1         false    Draft       deployments
deployments-93bb9ac8c2fb7a5759547a38f5f48b369f42d08a   new-package   v2         false    Draft       deployments
...
```

Now, in the role of the package author, we will propose two of those packages
to be published: `istions/v1` and `my-bucket/v2`.

```sh
# Propose two package revisions to be be published
$ kpt alpha rpkg propose \
  deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b \
  deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486 \
  -ndefault

deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b proposed
deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486 proposed
```

 > Refer to the [propose command reference](../../reference/cli/alpha/rpkg/propose/) for usage.

The two package revisions are now **`Proposed`**:

```sh
# Confirm the package revisions are now Proposed (the output was abbreviated
# to only show relevant packages)
$ kpt alpha rpkg get      
NAME                                                   PACKAGE       REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b   istions       v1         false    Proposed    deployments
deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486   my-bucket     v1         false    Proposed    deployments
deployments-93bb9ac8c2fb7a5759547a38f5f48b369f42d08a   new-package   v2         false    Draft       deployments
...
```

At this point, a person in the _platform administrator_ role, or even an
automated process, will review and either approve or reject the proposals.
To aid with the decision, the platform administrator may inspect the package
contents using the commands above, such as `kpt alpha rpkg pull`.

```sh
# Approve a proposal to publish istions/v1
$ kpt alpha rpkg approve deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b -ndefault
deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b approved

# Reject a proposal to publish a my-bucket/v1
$ kpt alpha rpkg reject deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486 -ndefault
deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486 rejected
```

 > Refer to the [approve](../../reference/cli/alpha/rpkg/approve/) and [reject](../../reference/cli/alpha/rpkg/reject/)
   command reference for usage.

> Approving a package revisions requires that the current user has been granted
> update access to the `approve` subresource of `packagerevisions`. This allows
> for giving only a limited set of users permission to approve package revisions.

Now, confirm lifecycle stages of the package revisions:

```sh
# Confirm package revision lifecycle stages after approvals (output was
# abbreviated to display only relevant package revisions):
$ kpt alpha rpkg get
NAME                                                   PACKAGE       REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-98bc9a49246a5bd0f4c7a82f3d07d0d2d1293cd0   istions       main       false    Published   deployments
deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b   istions       v1         true     Published   deployments
deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486   my-bucket     v1         false    Draft       deployments
deployments-93bb9ac8c2fb7a5759547a38f5f48b369f42d08a   new-package   v2         false    Draft       deployments
...
```

The rejected proposal returned the package to **`Draft`**, and the approved
proposal resulted in **`Published`** package revision.

You may have noticed that a `main` revision of the istions package appeared.
When a package is approved, Porch will commit it into the branch which was
provided at the repository registration (`main` in this case) and apply a tag.
As a result, the package revision exists in two locations - tag, and the `main`
branch.


