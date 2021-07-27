## Migrating from kpt `v0.39` to `v1.0`

- [Before you begin](#Before-you-begin)
- [What's new and changed?](#What's-new-and-changed?)
  - [CLI changes](#CLI-changes)
  - [Kptfile schema changes](#Kptfile-schema-changes)
  - [`pkg`](#pkg)
    - [`sync` merged with `update`](#sync-merged-with-update)
  - [`cfg`](#cfg)
    - [Changes to Setters](#Changes-to-Setters)
    - [Setter validations deprecated](#Setter-validations-deprecated)
    - [Auto setters deprecated](#Auto-setters-deprecated)
  - [`fn`](#fn)
    - [`run` is split into `eval` and `render`](#run-is-split-into-eval-and-render)
    - [Function Config](#Function-Config)
    - [Function Results](#Function-Results)
  - [`live`](#live)
- [Migration steps](#Migration-steps)
  - [For Package Publishers](#For-Package-Publishers)
    - [Automated portion of migration](#Automated-portion-of-migration)
      - [Changes made by the function](#Changes-made-by-the-function)
    - [Manual portion of migration](#Manual-portion-of-migration)
  - [For Package Consumers](#For-Package-Consumers)
- [Timeline](#Timeline)

kpt `v1.0` is going to be the latest major release of the kpt CLI. The
implementation of kpt `v1.0` has changed considerably from kpt `v0.39`. A rich
set of new features have been added as a result of the users’ input and
requirements. Some features have been deprecated or refactored in ways that make
them incompatible with kpt `v0.39`. Since these are backwards incompatible
changes, there should be a way for users to migrate/fix their existing kpt
packages which are compatible with `v0.39` version of kpt, to become compatible
with kpt `v1.0`. This document outlines the end to end migration journey of
users using a comprehensive kpt package example.

## Before you begin

Please go through [installation instructions] for installing `v1.0` binary and
at least Chapter 1 and 2 of [The kpt Book] for understanding the basic model of
kpt `v1.0`.

## What's new and changed?

### CLI changes

To start with, all the commands in kpt `v1.0` will follow the consistent pattern

```
$ kpt <group> <command> <positional_args> [PKG_PATH | DIR | - STDIN] [flags]
```

Almost all the existing commands/features in kpt `v0.39` are also offered in kpt
`v1.0` but in a better and enhanced format. In kpt `v0.39`, `cfg` group had been
a dumping ground for many cli commands which don’t have a coherent story. `cfg`
is often confused with `pkg` and vice-versa. As a major step, we removed the
`cfg` subgroup and rearranged the functionality. Here are the one-liners for
each command group in kpt `v1.0`.

1. `pkg:` Commands for composing and describing packages.
2. `fn:` Commands for running functions for validation and customization.
3. `live:` Commands for interacting with a live cluster.

`PKG_PATH vs DIR:` Most of the commands accept only `PKG_PATH` as input which
means the input directory must have a valid Kptfile. Few commands can work on
just the simple `DIR` with resources. Commands operate on the current working
directory by default.

| [v0.39 Commands]                                                                                  | [v1.0 Commands]                                                                                                                                                                                                                                      |
| ------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] LOCAL_DEST_DIRECTORY [flags]`                      | `kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] [flags] [LOCAL_DEST_DIRECTORY]` <br> Fetch a remote package from a git subdirectory and writes it to a new local directory.                                                                           |
| `kpt pkg init DIR [flags]`                                                                        | `kpt pkg init [DIR] [flags]` <br> Initializes an existing empty directory as a kpt package by adding a Kptfile.                                                                                                                                      |
| `kpt pkg update LOCAL_PKG_DIR[@VERSION] [flags]`                                                  | `kpt pkg update [PKG_PATH][@version] [flags]` <br> Pulls in upstream changes and merges them into a local package.                                                                                                                                   |
| `kpt pkg fix DIR [flags]`                                                                         | `kpt fn eval --image gcr.io/kpt-fn/fix:v0.2 --include-meta-resources` <br> Fix a local package which is using deprecated features.                                                                                                                                            |
| `kpt pkg desc DIR [flags]`                                                                        | Deprecated in favor of reading Kptfile directly                                                                                                                                                                                                      |
| `kpt pkg diff DIR[@VERSION] [flags]`                                                              | `kpt pkg diff [PKG_PATH][@version] [flags]` <br> Display differences between upstream and local packages.                                                                                                                                            |
| `kpt cfg fmt DIR/STDIN [flags]`                                                                   | `kpt fn eval --image gcr.io/kpt-fn/format:v0.1`                                                                                                                                                                                                      |
| `kpt cfg tree DIR/STDIN [flags]`                                                                  | `kpt pkg tree [DIR] [flags]` <br> Displays resources, files and packages in a tree structure.                                                                                                                                                        |
| `kpt cfg cat DIR/STDIN [flags]`                                                                   | `kpt fn source [DIR] -o unwrap`                                                                                                                                                                                                                      |
| `kpt fn run DIR/STDIN [flags]`                                                                    | `kpt fn eval [DIR / -] [flags]` <br> Executes a single function on resources in a directory. <br> <br> `kpt fn render [PKG_PATH]` <br> Executes the pipeline of functions on resources in the package and writes the output to the local filesystem. |
| `kpt fn source DIR [flags]`                                                                       | `kpt fn source [DIR] [flags]` <br> Reads resources from a local directory and writes them in Function Specification wire format to stdout.                                                                                                           |
| `kpt fn sink DIR [flags]`                                                                         | `kpt fn sink [DIR] [flags]` <br> Reads resources from stdin and writes them to a local directory.                                                                                                                                                    |
| `kpt fn export DIR [flags]`                                                                       | Deprecated.                                                                                                                                                                                                                                          |
| `kpt live init DIR [flags]`                                                                       | `kpt live init [PKG_PATH] [flags]` <br> Initializes the package with the name, namespace and id of the resource that will keep track of the package inventory.                                                                                       |
| `kpt live apply DIR/STDIN [flags]`                                                                | `kpt live apply [PKG_PATH / -] [flags]` <br> Creates, updates and deletes resources in the cluster to make the remote cluster resources match the local package configuration.                                                                       |
| `kpt live preview DIR/STDIN [flags]`                                                              | Deprecated. The functionality will be provided by `dry-run` flag in `apply` and `destroy` commands.                                                                                                                                                  |
| `kpt live destroy DIR/STDIN [flags]`                                                              | `kpt live destroy [PKG_PATH / -] [flags]` <br> Removes all files belonging to a package from the cluster.                                                                                                                                            |
| `kpt live status DIR/STDIN [flags]`                                                               | `kpt live status [PKG_PATH / -] [flags]` <br> Shows the resource status for resources belonging to the package.                                                                                                                                      |
| `kpt live diff DIR/STDIN [flags]`                                                                 | Deprecated. The functionality will be provided by `dry-run` flag in `apply` and `destroy` commands.                                                                                                                                                  |
| `kpt cfg set DIR setter_name setter_value`                                                        | `kpt fn eval --image gcr.io/kpt-fn/apply-setters:v0.1 -- 'foo=bar' 'env=[dev, stage]'`                                                                                                                                                               |
| `kpt cfg create-setter DIR setter_name setter_value`                                              | `kpt fn eval --image gcr.io/kpt-fn/search-replace:v0.1 -- 'by-value=nginx' 'put-comment=kpt-set: ${image}'`                                                                                                                                          |
| `kpt cfg create-subst DIR subst_name --field-value nginx:1.7.1 --pattern ${image}:${tag} [flags]` | `kpt fn eval --image gcr.io/kpt-fn/search-replace:v0.1 -- 'by-value=nginx:1.7.1' 'put-comment=kpt-set: ${image}:${tag}'`                                                                                                                             |
| `kpt cfg delete-setter DIR setter_name`                                                           | `kpt fn eval --image gcr.io/kpt-fn/search-replace:v0.1 -- 'by-value=nginx' put-comment=''`                                                                                                                                                           |
| `kpt cfg delete-subst DIR subst_name [flags]`                                                     | `kpt fn eval --image gcr.io/kpt-fn/search-replace:0.1 -- 'by-value=nginx:1.7.1' put-comment=''`                                                                                                                                                      |
| `kpt cfg annotate DIR/STDIN [flags]`                                                              | `kpt fn eval --image gcr.io/kpt-fn/set-annotations:v0.1 -- 'name=foo' 'value=bar'`                                                                                                                                                                   |
| `kpt cfg grep DIR/STDIN [flags]`                                                                  | `kpt fn eval --image gcr.io/kpt-fn/search-replace:v0.1 -- 'by-value=foo' 'by-path=bar'`                                                                                                                                                              |
| `kpt cfg count DIR/STDIN [flags]`                                                                 | Deprecated.                                                                                                                                                                                                                                          |

### Kptfile schema changes

The existing [v1alpha1 Kptfile] format/schema is not compatible with kpt `v1.0`.
Here is the schema for [v1 Kptfile] which is compatible with kpt `v1.0`.

1. The `packageMetaData` section in `v1alpha1` Kptfile is transformed to `info`
   section in `v1` Kptfile.
2. `upstream` section, in the `v1alpha1` Kptfile is split into `upstream` and
   `upstreamLock` sections in `v1` Kptfile. `upstream` section can be
   modified by users to declare the desired version of the package,
   `upstreamLock` is the resolved package state and should not be modified by
   users.
3. The `OpenAPI` section in `v1alpha1` Kptfile is deprecated.
   [Details below](#changes-to-setters).
4. `dependencies` section in `v1alpha1` Kptfile is deprecated.
   [Details below](#sync-merged-with-update).
5. `functions` section in `v1alpha1` Kptfile holds definitions for Starlark
   functions only. This section is deprecated and all the functions can be
   declared in the `pipeline` section including [Starlark function].
6. `inventory-object` is moved to `inventory` section in `v1` Kptfile.
   [Details below](#live-group-changes).

### `pkg`

#### `sync` merged with `update`

Nothing to worry about. `sync` is just a declarative version of `update`
functionality. The functionality is offered by `kpt pkg update` in a more user
friendly fashion. Running the new `kpt pkg update` command on a package which
has subpackages, simply means traversing the package hierarchy, and for each
package encountered, update the package if needed. If you want to declaratively
update nested subpackages, you can declare the desired version in the upstream
section of the respective subpackage and invoking `kpt pkg update` on the root
package will take care of updating all the nested subpackages. [Update guide]

### `cfg`

#### Changes to Setters

`Setters` and `substitutions` no longer follow the complex and verbose OpenAPI
format. The setters and substitutions are simplified to use new and simple
setter pattern comments. Creating a setter is as simple as adding a line comment
to the desired field. Users need not add any `OpenAPI` definitions. Please refer
to [apply-setters] for the information about new syntax for setters and how to
apply setter values, this is equivalent to the `kpt cfg set` in `v0.39`. [Setter
Inheritance] works as usual in kpt `v1.0`.

Here is the [simple example] of setter syntax transformation.

#### Setter validations deprecated

We want to keep the authoring experience of setters as simple as possible and
hence [OpenAPI validations] and [required setters] feature offered by `v0.39`
setters is no longer offered in `v1.0` version of kpt. However, we are working
on providing an easy way to achieve similar functionality retaining the
simplicity of setters, which is scoped for post `v1.0` release. Stay tuned.

#### Auto setters deprecated

[Auto-setters] feature is deprecated in `v1.0` version of kpt. Since the setters
are migrated to a new and simple declarative version, package consumers can
easily declare all the setter values and render them all at once.

### `fn`

#### `run` is split into `eval` and `render`

The functionality of `kpt fn run` command in `v0.39` is split into two different
CLI commands that execute functions corresponding to two fundamentally different
approaches:

`kpt fn render:` Executes the pipeline of functions declared in the package and
its subpackages. This is a declarative way to run functions. [render guide]

`kpt fn eval:` Executes a given function on the package. The image to run and
the functionConfig is specified as CLI argument. This is an imperative way to
run functions. [eval guide]

#### Function Config

As a result of these changes, we no longer need function configs to hold
`config.kubernetes.io/function` annotation. Functions can be declared in the
pipeline section of Kptfile and invoked using `kpt fn render`. The declared
function can point to the [function config] using `configPath` argument.

#### Function Results

In both `render` and `eval`, structured results can be enabled using the
`--results-dir` flag. Please refer to the [function results] section for more
information on the new structure of function results.

### `live`

`kpt live` in `v1.0` no longer uses an inventory object to track the grouping of
resources in the cluster. Instead, it uses a more expressive `ResourceGroup`
CRD. Please refer to the user guide on [migrating inventory objects] to the
`ResourceGroup` equivalent.

## Migration Steps

### For Package Publishers

Based on the changes discussed above, this section walks package publishers through an end to
end workflow to migrate their existing packages which are compatible with
`v0.39` version of kpt, and make them compatible with kpt `v1.0`.

#### Automated portion of migration

Since you are the package publisher, you are expected to have the latest version
of published package on your local disk. If you do not already have it, you can [git clone]
the latest version of remote package on to your local disk.

```shell
$ DEMO_HOME=$(mktemp -d); cd $DEMO_HOME
```

```shell
# replace it with your package repo uri
$ git clone https://github.com/GoogleContainerTools/kpt-functions-catalog.git
```

```shell
# cd to the package directory which you want to migrate
$ cd kpt-functions-catalog/testdata/fix/nginx-v1alpha1
```

```shell
# verify the version of kpt
$ kpt version
1.0.0+
```

Invoke `gcr.io/kpt-fn/fix` function on the kpt package.

```shell
# you must be using 1.0+ version of kpt
$ kpt fn eval --image gcr.io/kpt-fn/fix:v0.2 --include-meta-resources --truncate-output=false
```

```shell
# observe the changes done by the fix function
$ git diff
```

##### Changes made by the function

1. Best effort is made by the function to transform the `packageMetaData`
   section to the `info` section.
2. `upstream` section, in the `v1alpha1` Kptfile is converted to `upstream` and
   `upstreamLock` sections in `v1` version of Kptfile.
3. `dependencies` section is removed from Kptfile.
4. Starlark functions section is removed from Kptfile.
5. `Setters` and `substitutions` are converted to new and simple setter
   patterns. The setter comments in the resources are updated to follow new
   comment syntax.
6. The `apply-setters` function and all the setters are added to the mutators
   section in the pipeline.
7. Function annotation from function configs will be removed and corresponding
   function definitions will be declared in `pipeline` section of Kptfile.
   Reference to function config is added via [configPath] option.

Note: This function modifies only the local package files and doesn’t make any
changes to the resources in the live cluster.

#### Manual portion of migration

1. All the functions are treated as `mutators` by the `gcr.io/kpt-fn/fix`
   function and are added to the `mutators` section in the pipeline. Users must
   manually go through the functions and move the validator functions to the
   `validators` section in the pipeline section of `v1` Kptfile.
   1. The order of functions also must be re-arranged manually by users if
      needed.
   2. Also, note that the [function config] is used to configure the function
      and is not included in the input list of resources to function.
2. [OpenAPI validations] and required setters feature offered by `v0.39` setters
   is no longer offered in `v1.0` version of kpt. Users must write their own
   validation functions to achieve the functionality. `Tip:` Adding a [starlark
   function] would be an easier alternative to achieve the equivalent validation
   functionality.
3. If you have used [Starlark runtime] in `v0.39`, please checkout the new and
   improved [Starlark functions] and declare it accordingly in the pipeline.
4. [Auto-setters] feature is deprecated in `v1.0` version of kpt. Since the
   setters are migrated to a new and simple declarative version, package
   consumers can easily declare all the setter values and render them all at
   once.

Test your migrated kpt package end-to-end and make sure that the
functionality is as expected. `gcr.io/kpt-fn/fix` is a helper for migration and
doesn't guarantee functional parity.

Finally, [publish your package] to git by upgrading the version so that your
consumers can fetch the specific version of the package.

### For Package Consumers

This section walks package consumers through an end to end workflow in order to
fetch the latest version of the published package which is compatible with kpt `v1.0`
and migrate the local customizations(if any) already performed to their existing package.

- As a package consumer, you are expected to have some version of the kpt package
  which is compatible with kpt `v0.39` on your local.
  Fetch the latest version of published remote package in to a new directory different
  from your existing package directory.

```shell
$ DEMO_HOME=$(mktemp -d); cd $DEMO_HOME
```

```shell
# verify the version of kpt
$ kpt version
1.0.0+
```

```shell
# fetch the package with upgraded version
$ kpt pkg get https://github.com/GoogleContainerTools/kpt-functions-catalog.git/testdata/fix/nginx-v1@master
```

- You might have performed some customizations to your existing package such as,
  updated setter values, made some in-place edits to your resources etc.
  Please make sure that you capture those customizations and add them to the newly
  fetched package.

- Render the package resources with customizations

```shell
$ kpt fn render nginx-v1/
```

- The step is only applicable if you're using `kpt live` functionality.
  a. If you are using the inventory object in order to manage live resources in the cluster,
     please refer to `live migrate` command docs to perform [live migration].
  b. If you are using ResourceGroup CRD to manage live resources, copy the inventory
     section in the Kptfile of existing package to the Kptfile of new package.

- Once you test your new package and confirm that all the changes are as expected,
  you can simply discard the existing package and move forward with the new version
  of the fetched and customized package.

Here is an [example kpt package], migrated from `v1alpha1` version(compatible with
kpt `v0.39`) to `v1` version(compatible with kpt `v1.0`).

## Timeline

1. `Right now:` You can [install] and try the pre-release version of kpt `v1.0`
   binary.
2. `June 1, 2021:` `gcr.io/kpt-fn/fix` function will be released in
   [kpt-functions-catalog]. You can start migrating your existing kpt packages
   using the function.
3. `July 1, 2021:` Package format `v1` will be released which guarantees
   backwards compatability for new packages going forward. The existing kpt
   packages are not compatible with the kpt `v1.0` binary and users will be
   prompted to migrate/fix their packages.
4. `TBD:` Feature releases and bug fixes for pre-v1 versions of kpt will be
   reserved for serious security bugs only. Users will be asked to migrate to
   kpt `v1.0`.

[v0.39 commands]: https://googlecontainertools.github.io/kpt/reference/
[v1.0 commands]: https://kpt.dev/reference/cli/
[v1 kptfile]: https://github.com/GoogleContainerTools/kpt/blob/main/pkg/api/kptfile/v1/types.go
[starlark function]: https://catalog.kpt.dev/starlark/v0.2/
[apply-setters]: https://catalog.kpt.dev/apply-setters/v0.1/
[setter inheritance]: https://googlecontainertools.github.io/kpt/concepts/setters/#inherit-setter-values-from-parent-package
[openapi validations]: https://googlecontainertools.github.io/kpt/guides/producer/setters/#openapi-validations
[required setters]: https://googlecontainertools.github.io/kpt/guides/producer/setters/#required-setters
[auto-setters]: https://googlecontainertools.github.io/kpt/concepts/setters/#auto-setters
[migrating inventory objects]: https://googlecontainertools.github.io/kpt/reference/live/alpha/
[live migration]: https://googlecontainertools.github.io/kpt/reference/cli/live/alpha/
[configpath]: https://kpt.dev/book/04-using-functions/01-declarative-function-execution?id=configpath
[example kpt package]: https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/master/testdata/fix
[simple example]: https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/master/functions/go/fix#examples
[function config]: https://kpt.dev/book/04-using-functions/01-declarative-function-execution?id=configpath
[starlark runtime]: https://googlecontainertools.github.io/kpt/guides/producer/functions/starlark/
[update guide]: https://kpt.dev/book/03-packages/05-updating-a-package
[render guide]: https://kpt.dev/book/04-using-functions/01-declarative-function-execution
[eval guide]: https://kpt.dev/book/04-using-functions/02-imperative-function-execution
[function results]: https://kpt.dev/book/04-using-functions/03-function-results
[the kpt book]: https://kpt.dev/book/
[installation instructions]: https://kpt.dev/installation/
[install]: https://kpt.dev/installation/
[kpt-functions-catalog]: https://catalog.kpt.dev/
[v1alpha1 kptfile]: https://github.com/GoogleContainerTools/kpt/blob/master/pkg/kptfile/pkgfile.go#L39
[git clone]: https://git-scm.com/docs/git-clone
[publish your package]: https://kpt.dev/book/03-packages/08-publishing-a-package
