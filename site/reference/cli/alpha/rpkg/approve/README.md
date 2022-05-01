---
title: "`approve`"
linkTitle: "approve"
type: docs
description: >
  Approve a proposal to publish a package revision.
---

<!--mdtogo:Short
    Approve a proposal to publish a package revision.
-->

`approve` publishes a package revision

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg approve [PACKAGE_REV_NAME...] [flags]
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
# approve package revision blueprint-91817620282c133138177d16c981cf35f0083cad
$ kpt alpha rpkg approve blueprint-91817620282c133138177d16c981cf35f0083cad --namespace=default
```

<!--mdtogo-->