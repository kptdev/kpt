---
title: "`get`"
linkTitle: "get"

description: >
  Get sync resources from the cluster.
---

<!--mdtogo:Short
    Get sync resources from the cluster.
-->

`get` lists sync resources in the cluster.

### Synopsis

<!--mdtogo:Long-->

```shell
kpt alpha sync get [DEPLOYMENT_NAME] [flags]
```

#### Args

```shell
DEPLOYMENT_NAME:
  The name of a sync resource.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# get the sync resource named my-app from the cluster.
$ kpt alpha sync get my-app
```

<!--mdtogo-->