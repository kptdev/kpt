---
title: "`render`"
linkTitle: "render"
type: docs
description: >
  Render a package
---

<!--mdtogo:Short
   Render a package.
-->

`render` executes the pipeline of functions on resources in the package and
writes the output to the local filesystem in-place.

`render` executes the pipelines in the package hierarchy in a depth-first order.
For example, if a package A contains subpackage B, then the pipeline in B is executed
on resources in B and then the pipeline in A is executed on resources in A and
the output of the pipeline from package B. The output of the pipeline from A is
then written to the local filesystem in-place.

`render` formats the resources before writing them to the local filesystem.

Meta resources (i.e. `Kptfile` and `functionConfig`) are excluded from the inputs
to the functions.

If any of the functions in the pipeline fails, then the entire pipeline is aborted and
the local filesystem is left intact.

Refer to the [Declarative Functions Execution] for more details.

### Synopsis

<!--mdtogo:Long-->

```shell
kpt fn render [PKG_PATH] [flags]
```

#### Args

```shell
PKG_PATH:
  Local package path to render. Directory must exist and contain a Kptfile
  to be updated. Defaults to the current working directory.
```

#### Flags

```shell
--image-pull-policy:
  If the image should be pulled before rendering the package(s). It can be set
  to one of always, ifNotPresent, never. If unspecified, always will be the
  default.

--results-dir:
  Path to a directory to write structured results. Directory will be created if
  it doesn't exist. Structured results emitted by the functions are aggregated and saved
  to `results.yaml` file in the specified directory.
  If not specified, no result files are written to the local filesystem.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# Render the package in current directory
$ kpt fn render
```

```shell
# Render the package in current directory and save results in my-results-dir
$ kpt fn render --results-dir my-results-dir
```

```shell
# Render my-package-dir
$ kpt fn render my-package-dir
```

<!--mdtogo-->

[declarative functions execution]: /book/04-using-functions/01-declarative-function-execution
