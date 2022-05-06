---
title: "`get`"
linkTitle: "get"
type: docs
description: >
  List registered repositories.
---

<!--mdtogo:Short
    List registered repositories.
-->

`get` lists registered repositories.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha repo get [REPOSITORY_NAME] [flags]
```

#### Args

```
REPOSITORY_NAME:
  The name of a repository. If provided, only that specific
  repository will be shown. Defaults to showing all registered
  repositories.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# list all repositories registered in the default namespace
$ kpt alpha repo get --namespace default
```

```shell
# show the repository named foo in the bar namespace
$ kpt alpha repo get foo --namespace bar
```

<!--mdtogo-->