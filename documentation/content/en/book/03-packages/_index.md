---
title: "Chapter 3: Packages"
linkTitle: "Chapter 3: Packages"
description: |
    [Chapter 2](../02-concepts#packages) provided a high-level conceptual explanation of a package and the package
    lifecycle. This chapter covers working with packages in detail: how to get, explore, edit, update, and publish
    packages.
toc: true
menu:
  main:
    parent: "Book"
    weight: 30
---

## Getting a package

Packaging in kpt is based on Git forking. The producer publishes a package
by committing it to a Git repository. The consumer forks the package in order to use it.

Let us revisit the WordPress example:

```shell
kpt pkg get https://github.com/kptdev/kpt.git/package-examples/wordpress@v1.0.0-beta.61
```

A package in a Git repository can be fetched by specifying a branch, tag, or commit SHA. In the above example, the tag `v1.0.0-beta.61` is specified.

See the [get command reference](../../reference/cli/pkg/get/) for usage.

The `Kptfile` contains the metadata about the origin of the forked package. Have a look at the content of the `Kptfile` on
your local filesystem:

```yaml
# wordpress/Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: wordpress
upstream:
  type: git
  git:
    repo: https://github.com/kptdev/kpt
    directory: /package-examples/wordpress
    ref: v1.0.0-beta.61
  updateStrategy: resource-merge
upstreamLock:
  type: git
  git:
    repo: https://github.com/kptdev/kpt
    directory: /package-examples/wordpress
    ref: package-examples/wordpress/v1.0.0-beta.61
    commit: b9ea0bca019dafa9f9f91fd428385597c708518c
info:
  emails:
    - road-runner@theacmecorporation.com
    - wily.e.coyote@theacmecorporation.com
  description: This is an example wordpress package with mysql subpackage.
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:latest
      configMap:
        app: wordpress
  validators:
    - image: ghcr.io/kptdev/krm-functions-catalog/kubeconform:latest
status:
  conditions:
    - type: Rendered
      status: "True"
      reason: RenderSuccess
```

The `Kptfile` contains several sections to keep track of the package and its state: the `upstream` section, the `upstreamLock` section, and the `status` section. These sections are defined as follows:

1. The `upstream` section contains the user-specified Git reference to the upstream package. This contains three pieces
   of information:
   - `repo`: This is the Git repository where the package can be found.
   - `directory`: This is the directory within the Git repository where this package can be found.
   - `ref`: This is the Git reference for the package. This can be a branch, tag, or a commit SHA.
2. The `upstreamLock` section records the upstream Git reference (the exact Git SHA) that was
   fetched by kpt. This section is managed by kpt and should not be changed manually.
3. The `status` section records the operational state of the package. This is managed by kpt and tracks the execution
   status of operations such as `render`. The `status.conditions` field contains a list of condition objects, similarly to the way in which Kubernetes tracks the conditions on the resources. For example, after running `kpt fn render`, a `Rendered` condition is automatically recorded to indicate whether the last render succeeded or failed.

Let us now look at the `Kptfile` for the `mysql` subpackage:

```yaml
# wordpress/mysql/Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: mysql
info:
  emails:
    - wily.e.coyote@theacmecorporation.com
  description: This is an example mysql package.
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:latest
      configMap:
        tier: mysql
```

This `Kptfile` does not have the `upstream` and `upstreamLock` sections. This is because there are two different package types in kpt: the **independent package** and the **dependent package**:

- **Independent package:** This is a package in which the `Kptfile` has `upstream` defined.
- **Dependent package:** This is a package in which the `Kptfile` does not have `upstream` defined.

In this case, the `mysql` subpackage is a _dependent package_. The upstream package for `mysql` is automatically
inferred from the parent package. Think of the `Kptfile` in the `mysql` package as implicitly inheriting the `upstream` section
of its parent, with the only difference being that the `upstream.directory` in the subpackage instead points to `/package-examples/wordpress/mysql`.

### Package name and identifier

It is possible to specify a different local directory name to the `get` command. For example, the following command fetches the packages to a directory named `mywordpress`:

```shell
kpt pkg get https://github.com/kptdev/kpt.git/package-examples/wordpress@v1.0.0-beta.61 mywordpress
```

The _name of a package_ is given by its directory name. Since the `Kptfile` is a KRM resource and follows the familiar
structure of the KRM resources, the name of the package is also available from the `metadata.name` field. This must always
be the name of the directory. kpt updates it automatically when forking a package. In this case, `metadata.name` is set to
`mywordpress`.

In general, the package name is not unique. The _unique identifier_ for a package is defined as the relative path from
the top package to the subpackage. For example, we could have two subpackages with the name `mysql` having the following
identifiers:

- `wordpress/backend/mysql`
- `wordpress/frontend/mysql`

## Exploring a package

After having fetched a package to your local filesystem, it is generally a good idea to analyze the package, in order to understand how it is structured and how it can be customized to suit your needs. Given that a kpt package is simply an ordinary directory of human-readable YAML files, you can use your favorite file explorer, shell commands, or editor to explore the
package.

kpt also provides the `tree` command which is handy for quickly viewing the package hierarchy and the constituent packages, files, and resources:

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

See the [tree command reference](../../reference/cli/pkg/tree/) for usage.

In addition, you can use a kpt function, such as `search-replace`, to run a query on the package. For example, to search for resources that have a field with the
`spec.selector.tier` path, use the following kpt function:

```shell
kpt fn eval wordpress -i search-replace:latest -- 'by-path=spec.selector.tier'
```

## Editing a package

kpt does not maintain any state on your local machine outside the directory from where you fetched the package. Making changes to the package is achieved by manipulating the local filesystem. At the lowest level, _editing_ a package is simply a process that does one of the following:

- It changes the resources within the package. Examples are as follows:
  - Authoring a new Deployment resource.
  - Customizing an existing Deployment resource.
  - Modifying the `Kptfile`.
- It changes the package hierarchy, also called _package composition_. Examples are as follows:
  - Adding a subpackage.
  - Creating a new dependent subpackage.

Editing a package ultimately results in a Git commit that fully specifies the package. Depending on your use case, this process can be manual or automated.

We will cover package composition later in this chapter. For now, let us focus on editing the resources _within_ a package.

### Initializing the local repository

Before you make any changes to the package, first initialize and commit the pristine package, using the following command:

```shell
git init; git add .; git commit -m "Pristine wordpress package"
```

### Manual edits

As mentioned earlier, you can manually edit or author the KRM resources, using your favorite editor. Since every KRM resource has a known schema, you can take advantage of the tooling that assists in authoring and validating the resource configuration. For example, [Cloud Code](https://cloud.google.com/code) extensions for VS Code and IntelliJ provide IDE features such as auto-completion, inline documentation, linting, and snippets.

If, for example, you have VS Code installed, try modifying the resources in the `wordpress` package, as follows:

```shell
code wordpress
```

### Automation

It is often necessary to automate repetitive or complex operations. Having standardized on KRM for all resources in a package allows you to easily develop automation in different toolchains and languages, as well as at levels of abstraction.

For example, setting a label on all the resources in the `wordpress` package can be done using the following function:

```shell
kpt fn eval wordpress -i set-labels:latest -- env=dev
```

[Chapter 4](../04-using-functions/) discusses in detail the different ways of running the functions.

## Rendering a package

Regardless of how you have edited the package, you need to _render_ it. Use the following command to render the package:

```shell
kpt fn render wordpress
```

See the [render command reference](../../reference/cli/fn/render/) for command usage.

`render` is a critical step in the package lifecycle. At a high level, it performs the following steps:

1. It enforces the package preconditions. For example, it validates the `Kptfile`.
2. It executes the functions declared in the package hierarchy in a depth-first order. By default, the packages are modified in-place.
3. It guarantees the package postconditions. For example, it enforces consistent formatting of resources, even though a function (developed by different people using different toolchains) may have modified the formatting in some way.
4. It records the render execution status in the root `Kptfile` as a `Rendered` condition, under `status.conditions`. If the render execution was a success, then the condition has `status: "True"` and `reason: RenderSuccess`. If the render execution was unsuccessful, then the condition has `status: "False"` and `reason: RenderFailed`, and includes the error details in the `message` field.

Note that status conditions are only written for in-place renders (this is the default behavior). When using out-of-place output modes, such as `kpt fn render -o stdout` or `kpt fn render -o <dir>`,
no status condition is indicated because the package is not being updated on disk.

[Chapter 4](../04-using-functions/) discusses in detail the different ways of running the functions.

## Updating a package

An independent package records the exact commit where the local fork and the upstream package diverged. This enables kpt to fetch any update to the upstream package and merge it with the local changes.

### Committing your local changes

Before you update the package, you must commit your local changes.

First, use the following command to see the changes you have made to the fork of the upstream package:

```shell
git diff
```

If you are happy with the changes, then commit them, using the following command:

```shell
git add .; git commit -m "My changes"
```

### Updating the package

You can, for example, update to the `main` version of the `wordpress` package, using the following command:

```shell
kpt pkg update wordpress@main
```

This is a porcelain for manually updating the `upstream` section in the `Kptfile`, as follows:

```yaml
upstream:
  type: git
  git:
    repo: https://github.com/kptdev/kpt
    directory: /package-examples/wordpress
    # Change this from v1.0.0-beta.61 to main
    ref: main
  updateStrategy: resource-merge
```

and then running the following command:

```shell
kpt pkg update wordpress
```

The `update` command updates the local `wordpress` package and the dependent `mysql` package to the upstream version `main` by doing a three-way merge between the following:

1. the original upstream commit
2. the new upstream commit
3. the local (edited) package

Several different strategies are available for handling the merge. By default, the `resource-merge` strategy is used. This performs a structural comparison of the resource using the OpenAPI schema.

See the [update command reference](../../reference/cli/pkg/update/) for usage.

### Committing the updated resources

Once you have successfully updated the package, commit the changes, using the following command:

```shell
git add .; git commit -m "Updated wordpress to main"
```
## Creating a package

Creating a package is a simple task. Use the `pkg init` command to initialize a directory as a kpt package with a minimal `Kptfile` and `README` files:

```shell
kpt pkg init awesomeapp
```

This command automatically creates the `awesomeapp` directory, if it does not already exist, eliminating the need to manually create the directory beforehand.

See the [init command reference](../../reference/cli/pkg/init/) for usage.

The `info` section of the `Kptfile` contains some optional package metadata that you may want to set. These fields are not consumed by any functionality in kpt:

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: awesomeapp
info:
  description: Awesomeapp solves all the world's problems in half the time.
  site: awesomeapp.example.com
  emails:
    - jack@example.com
    - jill@example.com
  license: Apache-2.0
  keywords:
    - awesome-tech
    - world-saver
```

## Composing a package

A package can be _composed_ of subpackages (_HAS A_ relationship). _Package composition_ is when you change the package hierarchy by adding or removing subpackages.

There are two different ways to add a subpackage to a package on the local filesystem:

1. [Create a new package](#creating-a-package) in a subdirectory.
2. [Get an existing package](#getting-a-package) in a subdirectory.

Let us revisit the `wordpress` package and see how it was composed in the first place. Currently, it has the following package hierarchy:

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

First, delete the `mysql` subpackage. Deleting a subpackage is done by deleting the subdirectory, as follows:

```shell
rm -r wordpress/mysql
```

You now need to add back the `mysql` subpackage, using either of the following two approaches:

- creating a package
- getting an existing package

These two approaches are described in the following sections.

### Creating a `mysql` subpackage

Initialize the package, using the following command:

```shell
kpt pkg init wordpress/mysql
# author resources in mysql
```

This creates the `wordpress/mysql` directory, if it does not already exist, and initializes it as a [dependent package](#getting-a-package).

### Getting an existing package

Remove the existing directory, using the following command:

```shell
rm -rf wordpress/mysql
```

Fetch the package, using the following command:

```shell
kpt pkg get https://github.com/kubernetes/website.git/content/en/examples/application/mysql@snapshot-initial-v1.20 wordpress/mysql
```

This creates an [independent package](#getting-a-package). If you wish to make this a dependent package, then delete the `upstream` and `upstreamLock` sections of the `Kptfile` in the `mysql` directory.

## Publishing a package

A kpt package is published as a Git subdirectory containing the KRM resources. Publishing a package simply requires a normal Git push. This also means that any existing Git directory of the KRM resources is a valid kpt package.

As an example, republish the local `wordpress` package to your own repository.

Start by initializing the `wordpress` directory as a Git repository, if you have not already done so:

```shell
cd wordpress; git init; git add .; git commit -m "My wordpress package"
```

Tag the commit:

```shell
git tag v0.1
```

Push the commit, which requires you to have access to the repository:

```shell
git push origin v0.1
```

You can then fetch the published package:

```shell
kpt pkg get <MY_REPO_URL>/@v0.1
```

### Monorepo versioning

You may have a Git repository containing multiple packages. kpt provides a tagging convention to enable packages to be independently versioned.

For example, let us assume the `wordpress` directory is not at the root of the repository, but is instead in the `packages/wordpress` directory.

Tag the commit, using the following command:

```shell
git tag packages/wordpress/v0.1
```

Push the commit, using the following command:

```shell
git push origin packages/wordpress/v0.1
```

You can then fetch the published package:

```shell
kpt pkg get <MY_REPO_URL>/packages/wordpress@v0.1
```

[tagging]: https://git-scm.com/book/en/v2/Git-Basics-Tagging
