---
title: "`del`"
linkTitle: "del"
type: docs
description: >
  Delete a package revision.
---

<!--mdtogo:Short
    Delete a package revision.
-->

`del` removes a package revision from the repository.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg del PACKAGE_REV_NAME... [flags]
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
# remove package revision blueprint-e982b2196b35a4f5e81e92f49a430fe463aa9f1a from the default namespace
$ kpt alpha rpkg del blueprint-e982b2196b35a4f5e81e92f49a430fe463aa9f1a --namespace=default
```

<!--mdtogo-->