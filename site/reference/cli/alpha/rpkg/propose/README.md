---
title: "`propose`"
linkTitle: "propse"
type: docs
description: >
  Propose that a package revision should be published.
---

<!--mdtogo:Short
    Propose that a package revision should be published.
-->

`propose` creates a proposal for the package revision to be published.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg propose [PACKAGE_REV_NAME...] [flags]
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
# propose that package revision blueprint-91817620282c133138177d16c981cf35f0083cad should be finalized.
$ kpt alpha rpkg propose blueprint-91817620282c133138177d16c981cf35f0083cad --namespace=default
```

<!--mdtogo-->