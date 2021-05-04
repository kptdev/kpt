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

`render` executes the pipeline of functions, defined in the package Kptfile [kptfile reference] and writes the output back to the package directory.

If a package contains sub packages, the sub packages are rendered before the parent package. For example, if a package A contains sub package B, then the pipeline in B is executed first and then the pipeline in A is executed on the KRM resources in A and resources produced by rendering of B.

Note that the Kptfile and the function config files referred in the pipeline are excluded from the inputs to the functions.

### Synopsis
<!--mdtogo:Long-->
```
kpt fn render [DIR]
```

#### Args
```
DIR:
  Path to a package directory. Defaults to current directory if unspecified.
```
<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# Render the package in current directory
kpt fn render
```

```shell
# Render the package in `my-package-dir` directory
kpt fn render my-package-dir
```

[kptfile reference]: https://kpt.dev/reference/kptfile#pipeline

<!--mdtogo-->