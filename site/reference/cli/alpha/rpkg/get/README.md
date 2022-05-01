---
title: "`get`"
linkTitle: "get"
type: docs
description: >
  List package revisions in registered repositories.
---

<!--mdtogo:Short
    List package revisions in registered repositories.
-->

`get` lists package revisions in the registered repositories.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg get [PACKAGE_REV_NAME] [flags]
```

#### Args

```
PACKAGE_REV_NAME:
  The name of a package revision. If provided, only that specific
  package revision will be shown. Defaults to showing all package
  revisions from all repositories.
```

#### Flags

```
--name
  Name of the packages to get. Any package whose name contains 
  this value will be included in the results.

--revision
  Revision of the package to get. Any package whose revision
  matches this value will be included in the results.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# get a specific package revision in the default namespace
$ kpt alpha rpkg get blueprint-e982b2196b35a4f5e81e92f49a430fe463aa9f1a --namespace=default
```

```shell
# get all package revisions in the bar namespace
$ kpt alpha rpkg get --namespace=bar
```

```shell
# get all package revisions with revision v0
$ kpt alpha rpkg get --revision=v0
```

<!--mdtogo-->