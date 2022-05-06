---
title: "`delete`"
linkTitle: "delete"
type: docs
description: >
  Remove a sync resource from the cluster.
---

<!--mdtogo:Short
    Remove a sync resource from the cluster.
-->

`delete` removes a sync resource from the cluster. This will also remove
the package resources.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha sync delete DEPLOYMENT_NAME [flags]
```

#### Args

```
DEPLOYMENT_NAME:
  The name of the sync resource deleted from the cluster.
```

#### Flags

```
--keep-auth-secret
  Do not delete the repository authentication secret, if it exists.

--timeout
  How long we should wait for all resources to be deleted from the cluster.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# remove the my-app sync resource from the cluster. Wait up to 5 minutes for
# resources to be deleted.
$ kpt alpha sync delete my-app --timeout=5m
```

<!--mdtogo-->