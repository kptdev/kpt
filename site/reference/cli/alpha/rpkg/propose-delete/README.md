---
title: "`propose-delete`"
linkTitle: "propose-delete"
type: docs
description: >
  Propose deletion of a published package revision.
---

<!--mdtogo:Short
    Propose deletion of a published package revision.
-->

`propose-delete` proposes a published package revision for deletion, i.e.
changes its lifecycle from 'Published' to 'DeletionProposed'.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg propose-delete PACKAGE_REV_NAME... [flags]
```

#### Args

```
PACKAGE_REV_NAME...:
The name of one or more package revisions. If more than
one is provided, they must be space-separated.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# Propose published package revision blueprint-e982b2196b35a4f5e81e92f49a430fe463aa9f1a for deletion.
$ kpt alpha rpkg propose-delete blueprint-e982b2196b35a4f5e81e92f49a430fe463aa9f1a --namespace=default
```

<!--mdtogo-->