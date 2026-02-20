---
title: "Chapter 3: Packages"
linkTitle: "Chapter 3: Packages"
description: |
    [Chapter 2](../02-concepts#packages) provided a high-level conceptual explanation of a package and the package
    lifecycle. This chapter will cover working with packages in detail: how to get, explore, edit, update, and publish
    them.
toc: true
menu:
  main:
    parent: "Book"
    weight: 30
---

## Getting a package

Packaging in kpt is based on Git forking. Producers publish packages by committing them to a Git repository. Consumers
fork the package to use it.

Let's revisit the Wordpress example:

```shell
kpt pkg get https://github.com/kptdev/kpt.git/package-examples/wordpress@v1.0.0-beta.61
```

A package in a Git repo can be fetched by specifying a branch, tag, or commit SHA. In this case, we are specifying tag
`v1.0.0-beta.61`.

Refer to the [get command reference](../../reference/cli/pkg/get/) for usage.

The `Kptfile` contains metadata about the origin of the forked package. Take a look at the content of the `Kptfile` on
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
```

The `Kptfile` contains two sections to keep track of the upstream package:

1. The `upstream` section contains the user-specified Git reference to the upstream package. This contains three pieces
   of information:
   - `repo`: The Git repository where the package can be found
   - `directory`: The directory within the Git repository where this package can
     be found
   - `ref`: The Git reference for the package. This can be either a branch, tag,
     or commit SHA.
2. The `upstreamLock` section records the upstream Git reference (exact Git SHA) that was fetched by kpt. This section
   is managed by kpt and should not be changed manually.

Now, let's look at the `Kptfile` for the `mysql` subpackage:

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

As you can see, this `Kptfile` doesn't have the `upstream` and `upstreamLock` sections. This is because there are two
different package types in kpt:

- **Independent package:** A package where the `Kptfile` has `upstream` defined.
- **Dependent package:** A package where the `Kptfile` doesn’t have `upstream` defined.

In this case, the `mysql` subpackage is a _dependent package_. The upstream package for `mysql` is automatically
inferred from the parent package. You can think of the `Kptfile` in the `mysql` package as implicitly inheriting the
`upstream` section of its parent, with the only difference being that `upstream.directory` in the subpackage would
instead point to `/package-examples/wordpress/mysql`.

### Package Name and Identifier

It is possible to specify a different local directory name to the `get` command.
For example, the following fetches the packages to a directory named `mywordpress`:

```shell
kpt pkg get https://github.com/kptdev/kpt.git/package-examples/wordpress@v1.0.0-beta.61 mywordpress
```

The _name of a package_ is given by its directory name. Since the Kptfile is a KRM resource and follows the familiar
structure of KRM resources, the name of the package is also available from the `metadata.name` field. This must always
be the name of the directory, and kpt will update it automatically when forking a package. In this case, `metadata.name`
is set to `mywordpress`.

In general, the package name is not unique. The _unique identifier_ for a package is defined as the relative path from
the top package to the subpackage. For example, we could have two subpackages with the name `mysql` having the
following identifiers:

- `wordpress/backend/mysql`
- `wordpress/frontend/mysql`

## Exploring a package

After you fetch a package to your local filesystem, you typically want to explore the package to understand how it is
composed and how it can be customized for your needs. Given a kpt package is just an ordinary directory of
human-readable YAML files, you can naturally use your favorite file explorer, shell commands, or editor to explore the
package.

kpt also provides the `tree` command which is handy for quickly viewing package
hierarchy and the constituent packages, files, and resources:

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

Refer to the [tree command reference](../../reference/cli/pkg/tree/) for usage.

In addition, you can use a kpt function such as `search-replace` to run a query
on the package. For example, to search for resources that have a field with path
`spec.selector.tier`:

```shell
kpt fn eval wordpress -i search-replace:latest -- 'by-path=spec.selector.tier'
```

## Editing a package

kpt does not maintain any state on your local machine outside of the directory where you fetched the
package. Making changes to the package is accomplished by manipulating the local filesystem. At the
lowest-level, _editing_ a package is simply a process that either:

- Changes the resources within that package. Examples:
  - Authoring new a Deployment resource
  - Customizing an existing Deployment resource
  - Modifying the Kptfile
- Changes the package hierarchy, also called _package composition_. Examples:
  - Adding a subpackage.
  - Create a new dependent subpackage.

At the end of the day, editing a package will result in a Git commit that fully specifies
the package. This process can be manual or automated depending on your use case.

We will cover package composition later in this chapter. For now, let's focus on editing resources
_within_ a package.

### Initialize the local repo

Before you make any changes to package, you should first initialize and commit the pristine package:

```shell
git init; git add .; git commit -m "Pristine wordpress package"
```

### Manual edits

As mentioned earlier, you can manually edit or author KRM resources using your favorite editor.
Since every KRM resource has a known schema, you can take advantage of tooling that assists in
authoring and validating resource configuration. For example, [Cloud Code](https://cloud.google.com/code) extensions for
VS Code and IntelliJ provide IDE features such as auto-completion, inline documentation, linting, and snippets.

For example, if you have VS Code installed, try modifying the resources in the `wordpress` package:

```shell
code wordpress
```

### Automation

Oftentimes, you want to automate repetitive or complex operations. Having standardized on KRM for
all resources in a package allows us to easily develop automation in different
toolchains and languages, as well as at levels of abstraction.

For example, setting a label on all the resources in the `wordpress` package can be done
using the following function:

```shell
kpt fn eval wordpress -i set-labels:latest -- env=dev
```

[Chapter 4](../04-using-functions/) discusses different ways of running functions in detail.

## Rendering a package

Regardless of how you have edited the package, you want to _render_ the package:

```shell
kpt fn render wordpress
```

Refer to the [render command reference](../../reference/cli/fn/render/) for usage.

`render` is a critical step in the package lifecycle. At a high level, it
perform the following steps:

1. Enforces package preconditions. For example, it validates the `Kptfile`.
2. Executes functions declared in the package hierarchy in a depth-first order.
   By default, the packages are modified in-place.
3. Guarantees package postconditions. For example, it enforces a consistent
   formatting of resources, even though a function (developed by different
   people using different toolchains) may have modified the formatting in some
   way.

[Chapter 4](../04-using-functions/) discusses different ways of running functions in detail.

## Updating a package

An independent package records the exact commit where the local fork and the
upstream package diverged. This enables kpt to fetch any update to the upstream
package and merge it with local changes.

### Commit your local changes

Before you update the package, you want to commit your local changes.

First, to see the changes you've made to the fork of the upstream package:

```shell
git diff
```

If you're happy with the changes, commit them:

```shell
git add .; git commit -m "My changes"
```

## Update the package

For example, you can update to version `main` of the `wordpress` package:

```shell
kpt pkg update wordpress@main
```

This is a porcelain for manually updating the `upstream` section in the
`Kptfile` :

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

and then running:

```shell
kpt pkg update wordpress
```

The `update` command updates the local `wordpress` package and the dependent
`mysql` package to the upstream version `main` by doing a 3-way merge between:

1. Original upstream commit
2. New upstream commit
3. Local (edited) package

Several different strategies are available to handle the merge. By default, the
`resource-merge` strategy is used which performs a structural comparison of the
resource using OpenAPI schema.

Refer to the [update command reference](../../reference/cli/pkg/update/) for usage.

### Commit the updated resources

Once you have successfully updated the package, commit the changes:

```shell
git add .; git commit -m "Updated wordpress to main"
```
## Creating a package

Creating a new package is simple. Use the `pkg init` command to create a package directory with a minimal `Kptfile` and `README` files:

```shell
kpt pkg init awesomeapp
```

This will create the `awesomeapp` directory if it doesn't exist, and initialize it with the necessary files.

Refer to the [init command reference](../../reference/cli/pkg/init/) for usage.

The `info` section of the `Kptfile` contains some optional package metadata you
may want to set. These fields are not consumed by any functionality in kpt:

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

A package can be _composed_ of subpackages (_HAS A_ relationship). _Package
composition_ is when you change the package hierarchy by adding or removing
subpackages.

There are two different ways to add a subpackage to a package on the local
filesystem:

1. [Create a new package](#creating-a-package) in a subdirectory
2. [Get an existing package](#getting-a-package) in a subdirectory

Let's revisit the `wordpress` package and see how it was composed in the first
place. Currently, it has the following package hierarchy:

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

First, let's delete the `mysql` subpackage. Deleting a subpackage is done by
simply deleting the subdirectory:

```shell
rm -r wordpress/mysql
```

We're going to add back the `mysql` subpackage using the two different
approaches:

### Create a new package

Initialize the package:

```shell
kpt pkg init wordpress/mysql
# author resources in mysql
```

This will create the `wordpress/mysql` directory if it doesn't exist, and initialize it as a [dependent package](#getting-a-package).

### Get an existing package

Remove the existing directory if it exists:

```shell
rm -rf wordpress/mysql
```

Fetch the package:

```shell
kpt pkg get https://github.com/kubernetes/website.git/content/en/examples/application/mysql@snapshot-initial-v1.20 wordpress/mysql
```

This creates an [independent package](#getting-a-package). If you wish to make this a dependent
package, you can delete the `upstream` and `upstreamLock` sections of the
`Kptfile` in `mysql` directory.

## Publishing a package

A kpt package is published as a Git subdirectory containing KRM resources.
Publishing a package is just a normal Git push. This also means that any
existing Git directory of KRM resources is a valid kpt package.

As an example, let's re-publish the local `wordpress` package to your own repo.

Start by initializing the the `wordpress` directory as a Git repo if you haven't
already done so:

```shell
cd wordpress; git init; git add .; git commit -m "My wordpress package"
```

Tag the commit:

```shell
git tag v0.1
```

Push the commit which requires you to have access to the repo:

```shell
git push origin v0.1
```

You can then fetch the published package:

```shell
kpt pkg get <MY_REPO_URL>/@v0.1
```

### Monorepo Versioning

You may have a Git repo containing multiple packages. kpt provides a tagging
convention to enable packages to be independently versioned.

For example, let's assume the `wordpress` directory is not at the root of the
repo but instead is in the directory `packages/wordpress`.

Tag the commit:

```shell
git tag packages/wordpress/v0.1
```

Push the commit:

```shell
git push origin packages/wordpress/v0.1
```

You can then fetch the published package:

```shell
kpt pkg get <MY_REPO_URL>/packages/wordpress@v0.1
```

[tagging]: https://git-scm.com/book/en/v2/Git-Basics-Tagging
