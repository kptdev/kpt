---
title: "`pull`"
linkTitle: "pull"
type: docs
description: >
  Pull the content of the package revision.
---

<!--mdtogo:Short
    Pull the content of the package revision.
-->

`pull` fetches the content of the package revision from the
repository.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg pull PACKAGE_REV_NAME [DIR] [flags]
```

#### Args

```
PACKAGE_REV_NAME:
  The name of a an existing package revision in a repository.

DIR:
  A local directory where the package manifests will be written.
  If not provided, the manifests are written to stdout.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# pull the content of package revision blueprint-d5b944d27035efba53836562726fb96e51758d97
$ kpt alpha rpkg pull blueprint-d5b944d27035efba53836562726fb96e51758d97 --namespace=default
```

<!--mdtogo-->